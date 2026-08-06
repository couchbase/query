//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of the
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package external

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/extensions"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/parquet"
	pqfile "github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
	iceberg "github.com/apache/iceberg-go"

	"github.com/couchbase/query/expression"
)

func TestExtractVariantPredicates(t *testing.T) {
	alias := "d"

	tests := []struct {
		name string
		expr expression.Expression
		want []variantPredicate
	}{
		{
			name: "simple sub-path equality",
			expr: expression.NewEq(nestedFieldExpr(alias, "attrs", "color"), constStr("red")),
			want: []variantPredicate{{path: "attrs.color", op: iceberg.OpEQ, lit: "red"}},
		},
		{
			name: "flipped equality (literal first) stays EQ",
			expr: expression.NewEq(constStr("red"), nestedFieldExpr(alias, "attrs", "color")),
			want: []variantPredicate{{path: "attrs.color", op: iceberg.OpEQ, lit: "red"}},
		},
		{
			name: "flipped LT becomes GT",
			expr: expression.NewLT(constNum(10), nestedFieldExpr(alias, "attrs", "size")),
			want: []variantPredicate{{path: "attrs.size", op: iceberg.OpGT, lit: float64(10)}},
		},
		{
			name: "LE stays LE when field is first operand",
			expr: expression.NewLE(nestedFieldExpr(alias, "attrs", "size"), constNum(5)),
			want: []variantPredicate{{path: "attrs.size", op: iceberg.OpLTEQ, lit: float64(5)}},
		},
		{
			name: "AND chain: only the 2-segment sub-path conjunct is a candidate",
			expr: expression.NewAnd(
				expression.NewEq(nestedFieldExpr(alias, "attrs", "color"), constStr("red")),
				expression.NewEq(fieldExpr(alias, "other"), constNum(1)),
			),
			want: []variantPredicate{{path: "attrs.color", op: iceberg.OpEQ, lit: "red"}},
		},
		{
			name: "OR at the top yields no candidates",
			expr: expression.NewOr(
				expression.NewEq(nestedFieldExpr(alias, "attrs", "color"), constStr("red")),
				expression.NewEq(fieldExpr(alias, "other"), constNum(1)),
			),
			want: nil,
		},
		{
			name: "bare top-level field (1 segment) yields no candidates",
			expr: expression.NewEq(fieldExpr(alias, "other"), constNum(1)),
			want: nil,
		},
		{
			name: "3-segment path is a candidate (arbitrary depth)",
			expr: expression.NewEq(
				expression.NewField(nestedFieldExpr(alias, "attrs", "geo"), expression.NewFieldName("lat", false)),
				constStr("x"),
			),
			want: []variantPredicate{{path: "attrs.geo.lat", op: iceberg.OpEQ, lit: "x"}},
		},
		{
			name: "correlated reference from a different alias yields no candidates",
			expr: expression.NewEq(nestedFieldExpr("other", "attrs", "color"), constStr("red")),
			want: nil,
		},
		{
			// Unlike projection collection (which recurses into any expression's
			// Children() and so still narrows both operands of a concatenation
			// independently -- see planner/field_sub_paths_test.go), a
			// concatenation used AS one side of a comparison is correctly NOT
			// decomposable here: `a.attrs.color || a.other.x = 'xyz'` constrains
			// the COMBINED string, not either column independently, so there is
			// no valid per-column stats check to extract. n1qlFieldPath (via
			// expression.PathString) fails on the whole Concat node and
			// variantCompareExpr safely gives up -- no pruning, not a wrong
			// answer, and not a wrongly-invented pruning attempt either.
			name: "concatenation on one side of a comparison yields no candidates",
			expr: expression.NewEq(
				expression.NewConcat(nestedFieldExpr(alias, "attrs", "color"), nestedFieldExpr(alias, "other", "x")),
				constStr("xyz"),
			),
			want: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractVariantPredicates(tc.expr, alias, nil)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d predicates %v, want %d %v", len(got), got, len(tc.want), tc.want)
			}
			for i := range got {
				if got[i].path != tc.want[i].path || got[i].op != tc.want[i].op || got[i].lit != tc.want[i].lit {
					t.Errorf("predicate %d: got %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// buildShreddedVariantParquetRowGroups writes one row per row group -- row group i
// gets {"color": colors[i], "size": int64(i)} -- so tests can exercise row-group-level
// stats pruning with disjoint value ranges. Returns the low-level parquet reader
// (for RowGroupMetaData access) and the arrow/parquet reader (for its schema
// manifest), matching what the production streamParquetFile path holds onto.
func buildShreddedVariantParquetRowGroups(t *testing.T, colors []string) (*pqfile.Reader, *pqarrow.FileReader) {
	t.Helper()

	vt := extensions.NewShreddedVariantType(arrow.StructOf(
		arrow.Field{Name: "color", Type: arrow.BinaryTypes.String},
		arrow.Field{Name: "size", Type: arrow.PrimitiveTypes.Int64},
	))

	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	t.Cleanup(func() { mem.AssertSize(t, 0) })

	var buf bytes.Buffer
	// Each Write call below becomes its own row group (one small batch per call),
	// which is what lets this helper produce row groups with disjoint value ranges.
	schema := arrow.NewSchema([]arrow.Field{{Name: "variant", Type: vt, Nullable: true}}, nil)

	wr, err := pqarrow.NewFileWriter(schema, &buf,
		parquet.NewWriterProperties(parquet.WithDictionaryDefault(false)),
		pqarrow.DefaultWriterProps())
	if err != nil {
		t.Fatalf("NewFileWriter: %v", err)
	}

	for i, color := range colors {
		bldr := vt.NewBuilder(mem)
		jsonData := `[{"color": "` + color + `", "size": ` + strconv.Itoa(i) + `}]`
		if err := bldr.UnmarshalJSON([]byte(jsonData)); err != nil {
			bldr.Release()
			t.Fatalf("UnmarshalJSON: %v", err)
		}
		arr := bldr.NewArray()
		bldr.Release()

		rec := array.NewRecordBatch(schema, []arrow.Array{arr}, -1)
		arr.Release()
		if err := wr.Write(rec); err != nil {
			rec.Release()
			t.Fatalf("Write row group %d: %v", i, err)
		}
		rec.Release()
	}
	if err := wr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	pqReader, err := pqfile.NewParquetReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("NewParquetReader: %v", err)
	}
	t.Cleanup(func() { pqReader.Close() })

	arrowReader, err := pqarrow.NewFileReader(pqReader, pqarrow.ArrowReadProperties{}, memory.DefaultAllocator)
	if err != nil {
		t.Fatalf("NewFileReader: %v", err)
	}
	return pqReader, arrowReader
}

func TestFindVariantLeaf(t *testing.T) {
	_, arrowReader := buildShreddedVariantParquetRowGroups(t, []string{"red", "blue"})

	if _, ok := findVariantLeaf(arrowReader.Manifest, "variant", []string{"color"}); !ok {
		t.Errorf("expected to find variant.typed_value.color")
	}
	if _, ok := findVariantLeaf(arrowReader.Manifest, "variant", []string{"nonexistent"}); ok {
		t.Errorf("expected no leaf for a sub-name that doesn't exist")
	}
	if _, ok := findVariantLeaf(arrowReader.Manifest, "nonexistent", []string{"color"}); ok {
		t.Errorf("expected no leaf for a top-level field that doesn't exist")
	}
}

// buildNestedShreddedVariantParquetRowGroups writes one row per row group -- row
// group i gets {"geo": {"floor": floors[i], "zone": zones[i]}} -- exercising a
// shredded VARIANT sub-field that is itself a nested shredded struct, for
// arbitrary-depth path resolution (findVariantLeaf/matchShreddedPath,
// collectVariantLeaves via collectSubtree). Uses int64 fields, not float64:
// confirmed empirically (see TestDumpFlatFloatStats/TestDumpFlatIntStats during
// development) that this Parquet writer doesn't emit min/max statistics for
// FLOAT/DOUBLE columns at all -- a pre-existing, type-specific property of the
// writer (the standard NaN-safety convention for float stats), unrelated to
// shredding depth. That's orthogonal to what this fixture is for (arbitrary-depth
// resolution), so it uses a type that actually carries stats.
func buildNestedShreddedVariantParquetRowGroups(t *testing.T, floors, zones []int64) (*pqfile.Reader, *pqarrow.FileReader) {
	t.Helper()

	vt := extensions.NewShreddedVariantType(arrow.StructOf(
		arrow.Field{Name: "geo", Type: arrow.StructOf(
			arrow.Field{Name: "floor", Type: arrow.PrimitiveTypes.Int64},
			arrow.Field{Name: "zone", Type: arrow.PrimitiveTypes.Int64},
		)},
	))

	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	t.Cleanup(func() { mem.AssertSize(t, 0) })

	var buf bytes.Buffer
	schema := arrow.NewSchema([]arrow.Field{{Name: "variant", Type: vt, Nullable: true}}, nil)

	wr, err := pqarrow.NewFileWriter(schema, &buf,
		parquet.NewWriterProperties(parquet.WithDictionaryDefault(false)),
		pqarrow.DefaultWriterProps())
	if err != nil {
		t.Fatalf("NewFileWriter: %v", err)
	}

	for i := range floors {
		bldr := vt.NewBuilder(mem)
		jsonData := fmt.Sprintf(`[{"geo": {"floor": %d, "zone": %d}}]`, floors[i], zones[i])
		if err := bldr.UnmarshalJSON([]byte(jsonData)); err != nil {
			bldr.Release()
			t.Fatalf("UnmarshalJSON: %v", err)
		}
		arr := bldr.NewArray()
		bldr.Release()

		rec := array.NewRecordBatch(schema, []arrow.Array{arr}, -1)
		arr.Release()
		if err := wr.Write(rec); err != nil {
			rec.Release()
			t.Fatalf("Write row group %d: %v", i, err)
		}
		rec.Release()
	}
	if err := wr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	pqReader, err := pqfile.NewParquetReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("NewParquetReader: %v", err)
	}
	t.Cleanup(func() { pqReader.Close() })

	arrowReader, err := pqarrow.NewFileReader(pqReader, pqarrow.ArrowReadProperties{}, memory.DefaultAllocator)
	if err != nil {
		t.Fatalf("NewFileReader: %v", err)
	}
	return pqReader, arrowReader
}

func TestFindVariantLeafNestedDepth(t *testing.T) {
	_, arrowReader := buildNestedShreddedVariantParquetRowGroups(t, []int64{1, 2}, []int64{3, 4})

	if _, ok := findVariantLeaf(arrowReader.Manifest, "variant", []string{"geo", "floor"}); !ok {
		t.Errorf("expected to find variant.typed_value.geo.typed_value.floor")
	}
	if _, ok := findVariantLeaf(arrowReader.Manifest, "variant", []string{"geo", "nonexistent"}); ok {
		t.Errorf("expected no leaf for a nested sub-name that doesn't exist")
	}
	if _, ok := findVariantLeaf(arrowReader.Manifest, "variant", []string{"geo"}); ok {
		t.Errorf("expected no leaf for 'geo' alone -- it's a struct, not a scalar comparable to a literal")
	}
}

func TestBuildVariantRowGroupMatcherNestedDepth(t *testing.T) {
	pqReader, arrowReader := buildNestedShreddedVariantParquetRowGroups(t, []int64{1, 2}, []int64{3, 4})
	if pqReader.NumRowGroups() != 2 {
		t.Fatalf("test setup: expected 2 row groups, got %d", pqReader.NumRowGroups())
	}

	matcher := buildVariantRowGroupMatcher(arrowReader.Manifest,
		[]variantPredicate{{path: "variant.geo.floor", op: iceberg.OpGT, lit: float64(1)}})
	if matcher == nil {
		t.Fatalf("expected a non-nil matcher")
	}
	if matcher(pqReader.MetaData().RowGroup(0)) {
		t.Errorf("row group 0 (floor=1) should be skipped for floor > 1")
	}
	if !matcher(pqReader.MetaData().RowGroup(1)) {
		t.Errorf("row group 1 (floor=2) should be kept for floor > 1")
	}
}

func TestBuildVariantRowGroupMatcher(t *testing.T) {
	pqReader, arrowReader := buildShreddedVariantParquetRowGroups(t, []string{"red", "blue"})
	if pqReader.NumRowGroups() != 2 {
		t.Fatalf("test setup: expected 2 row groups, got %d", pqReader.NumRowGroups())
	}

	t.Run("no predicates returns nil (no-op)", func(t *testing.T) {
		if m := buildVariantRowGroupMatcher(arrowReader.Manifest, nil); m != nil {
			t.Errorf("expected nil matcher for empty predicates")
		}
	})

	t.Run("matching color keeps only its row group", func(t *testing.T) {
		matcher := buildVariantRowGroupMatcher(arrowReader.Manifest,
			[]variantPredicate{{path: "variant.color", op: iceberg.OpEQ, lit: "red"}})
		if matcher == nil {
			t.Fatalf("expected a non-nil matcher")
		}
		if !matcher(pqReader.MetaData().RowGroup(0)) {
			t.Errorf("row group 0 (color=red) should be kept")
		}
		if matcher(pqReader.MetaData().RowGroup(1)) {
			t.Errorf("row group 1 (color=blue) should be skipped")
		}
	})

	t.Run("non-matching value skips all row groups", func(t *testing.T) {
		matcher := buildVariantRowGroupMatcher(arrowReader.Manifest,
			[]variantPredicate{{path: "variant.color", op: iceberg.OpEQ, lit: "nonexistent"}})
		if matcher == nil {
			t.Fatalf("expected a non-nil matcher")
		}
		if matcher(pqReader.MetaData().RowGroup(0)) || matcher(pqReader.MetaData().RowGroup(1)) {
			t.Errorf("expected both row groups to be skipped for a non-matching value")
		}
	})

	t.Run("range comparison prunes via size", func(t *testing.T) {
		// row group 0 has size=0, row group 1 has size=1.
		matcher := buildVariantRowGroupMatcher(arrowReader.Manifest,
			[]variantPredicate{{path: "variant.size", op: iceberg.OpGT, lit: float64(0)}})
		if matcher == nil {
			t.Fatalf("expected a non-nil matcher")
		}
		if matcher(pqReader.MetaData().RowGroup(0)) {
			t.Errorf("row group 0 (size=0) should be skipped for size > 0")
		}
		if !matcher(pqReader.MetaData().RowGroup(1)) {
			t.Errorf("row group 1 (size=1) should be kept for size > 0")
		}
	})

	t.Run("unresolvable sub-name keeps everything (safe fallback)", func(t *testing.T) {
		matcher := buildVariantRowGroupMatcher(arrowReader.Manifest,
			[]variantPredicate{{path: "variant.nonexistent", op: iceberg.OpEQ, lit: "x"}})
		if matcher != nil {
			t.Errorf("expected nil matcher when no predicate resolves to a physical leaf")
		}
	})
}

// TestBuildVariantRowMatcher covers the row-level filter used by the
// parallel-files scan path (Scanner.ensureRowMatcher/emitRow) -- this is what
// actually drops non-matching rows within a row group that survived
// buildVariantRowGroupMatcher's (necessarily coarser, range-based) pruning. It
// operates on already-decoded rows via navigateRow, so unlike the row-group
// matcher it's an exact match, not a "could match" range check.
func TestBuildVariantRowMatcher(t *testing.T) {
	t.Run("no predicates returns nil (no-op)", func(t *testing.T) {
		if m := buildVariantRowMatcher(nil); m != nil {
			t.Errorf("expected nil matcher for empty predicates")
		}
	})

	t.Run("drops a non-matching row within an otherwise-kept row group", func(t *testing.T) {
		// Simulates a row group with mixed color values that survived
		// row-group-stats pruning (its min/max range spans both) -- row-level
		// filtering must still drop the row that doesn't actually match.
		matcher := buildVariantRowMatcher([]variantPredicate{
			{path: "attrs.color", op: iceberg.OpEQ, lit: "blue"},
		})
		if matcher == nil {
			t.Fatalf("expected a non-nil matcher")
		}
		redRow := map[string]interface{}{"attrs": map[string]interface{}{"color": "red", "size": int64(10)}}
		blueRow := map[string]interface{}{"attrs": map[string]interface{}{"color": "blue", "size": int64(20)}}
		if matcher(redRow) {
			t.Errorf("expected the red row to be dropped")
		}
		if !matcher(blueRow) {
			t.Errorf("expected the blue row to be kept")
		}
	})

	t.Run("nested path matches exactly, not just in range", func(t *testing.T) {
		matcher := buildVariantRowMatcher([]variantPredicate{
			{path: "attrs.geo.floor", op: iceberg.OpGT, lit: float64(1)},
		})
		if matcher == nil {
			t.Fatalf("expected a non-nil matcher")
		}
		floor1 := map[string]interface{}{"attrs": map[string]interface{}{"geo": map[string]interface{}{"floor": int64(1)}}}
		floor2 := map[string]interface{}{"attrs": map[string]interface{}{"geo": map[string]interface{}{"floor": int64(2)}}}
		if matcher(floor1) {
			t.Errorf("expected floor=1 to be dropped for floor > 1")
		}
		if !matcher(floor2) {
			t.Errorf("expected floor=2 to be kept for floor > 1")
		}
	})

	t.Run("unresolvable path keeps the row (safe fallback)", func(t *testing.T) {
		matcher := buildVariantRowMatcher([]variantPredicate{
			{path: "attrs.nonexistent", op: iceberg.OpEQ, lit: "x"},
		})
		if matcher == nil {
			t.Fatalf("expected a non-nil matcher")
		}
		row := map[string]interface{}{"attrs": map[string]interface{}{"color": "red"}}
		if !matcher(row) {
			t.Errorf("expected the row to be kept when the path can't be resolved")
		}
	})

	t.Run("multiple predicates all must match", func(t *testing.T) {
		matcher := buildVariantRowMatcher([]variantPredicate{
			{path: "attrs.color", op: iceberg.OpEQ, lit: "blue"},
			{path: "attrs.size", op: iceberg.OpGT, lit: float64(15)},
		})
		if matcher == nil {
			t.Fatalf("expected a non-nil matcher")
		}
		blueSmall := map[string]interface{}{"attrs": map[string]interface{}{"color": "blue", "size": int64(10)}}
		blueBig := map[string]interface{}{"attrs": map[string]interface{}{"color": "blue", "size": int64(20)}}
		if matcher(blueSmall) {
			t.Errorf("expected blueSmall to be dropped (size predicate fails)")
		}
		if !matcher(blueBig) {
			t.Errorf("expected blueBig to be kept (both predicates match)")
		}
	})
}
