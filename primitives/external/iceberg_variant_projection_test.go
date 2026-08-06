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
	go_context "context"
	"reflect"
	"sort"
	"testing"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/extensions"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
)

// singleLevelPathRoots builds a one-level *pathNode tree for field, e.g.
// singleLevelPathRoots("variant", "color") is the tree the query "SELECT
// d.variant.color" would produce -- a convenience for tests that don't care
// about deeper nesting.
func singleLevelPathRoots(field string, subs ...string) map[string]*pathNode {
	children := make(map[string]*pathNode, len(subs))
	for _, s := range subs {
		children[s] = &pathNode{whole: true}
	}
	return map[string]*pathNode{field: {children: children}}
}

// flattenTestPathRoots flattens a *pathNode tree into sorted dotted-path
// strings, mirroring planner/build_select_from.go's flattenPathNode, for
// readable test assertions against fieldSubNames()'s output.
func flattenTestPathRoots(roots map[string]*pathNode) []string {
	var flatten func(prefix string, node *pathNode) []string
	flatten = func(prefix string, node *pathNode) []string {
		if node.whole || len(node.children) == 0 {
			return []string{prefix}
		}
		var out []string
		for name, child := range node.children {
			out = append(out, flatten(prefix+"."+name, child)...)
		}
		return out
	}
	var got []string
	for name, node := range roots {
		got = append(got, flatten(name, node)...)
	}
	sort.Strings(got)
	return got
}

// buildShreddedVariantParquet writes a 2-row Parquet file with a single shredded
// VARIANT column named "variant" whose typed_value struct has "color" and "size"
// sub-fields, and returns an arrow/parquet reader over it.
func buildShreddedVariantParquet(t *testing.T) *pqarrow.FileReader {
	t.Helper()

	vt := extensions.NewShreddedVariantType(arrow.StructOf(
		arrow.Field{Name: "color", Type: arrow.BinaryTypes.String},
		arrow.Field{Name: "size", Type: arrow.PrimitiveTypes.Int64},
	))

	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	t.Cleanup(func() { mem.AssertSize(t, 0) })

	bldr := vt.NewBuilder(mem)
	defer bldr.Release()

	jsonData := `[
		{"color": "red", "size": 10},
		{"color": "blue", "size": 20}
	]`
	if err := bldr.UnmarshalJSON([]byte(jsonData)); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	arr := bldr.NewArray()
	defer arr.Release()

	rec := array.NewRecordBatch(arrow.NewSchema([]arrow.Field{
		{Name: "variant", Type: arr.DataType(), Nullable: true},
	}, nil), []arrow.Array{arr}, -1)
	defer rec.Release()

	var buf bytes.Buffer
	wr, err := pqarrow.NewFileWriter(rec.Schema(), &buf,
		parquet.NewWriterProperties(parquet.WithDictionaryDefault(false)),
		pqarrow.DefaultWriterProps())
	if err != nil {
		t.Fatalf("NewFileWriter: %v", err)
	}
	if err := wr.Write(rec); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := wr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	pqReader, err := file.NewParquetReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("NewParquetReader: %v", err)
	}
	t.Cleanup(func() { pqReader.Close() })

	arrowReader, err := pqarrow.NewFileReader(pqReader, pqarrow.ArrowReadProperties{}, memory.DefaultAllocator)
	if err != nil {
		t.Fatalf("NewFileReader: %v", err)
	}
	return arrowReader
}

// decodeVariantColumn reads colIndices from arrowReader and decodes every row of
// the (single) "variant" column via this package's own decodeVariantScalar, exactly
// as the production Parquet scan path does.
func decodeVariantColumn(t *testing.T, arrowReader *pqarrow.FileReader, colIndices []int) []map[string]interface{} {
	t.Helper()

	rr, err := arrowReader.GetRecordReader(go_context.Background(), colIndices, nil)
	if err != nil {
		t.Fatalf("GetRecordReader: %v", err)
	}
	defer rr.Release()

	var rows []map[string]interface{}
	for rr.Next() {
		rec := rr.Record()
		va, ok := rec.Column(0).(*extensions.VariantArray)
		if !ok {
			rec.Release()
			t.Fatalf("expected *extensions.VariantArray, got %T", rec.Column(0))
		}
		for i := 0; i < va.Len(); i++ {
			v, err := va.Value(i)
			if err != nil {
				rec.Release()
				t.Fatalf("Value(%d): %v", i, err)
			}
			obj, ok := decodeVariantScalar(v, false).(map[string]interface{})
			if !ok {
				rec.Release()
				t.Fatalf("row %d: expected map[string]interface{}", i)
			}
			rows = append(rows, obj)
		}
		rec.Release()
	}
	if err := rr.Err(); err != nil {
		t.Fatalf("record reader error: %v", err)
	}
	return rows
}

// TestResolveColumnIndicesVariantSubFieldPruning exercises the production
// resolveColumnIndices/collectVariantLeaves path end to end: narrowing a shredded
// variant's typed_value columns to a requested sub-name must (a) actually drop the
// unrequested sub-column's index and (b) still decode correctly via
// decodeVariantScalar, with the unrequested sub-name simply absent.
func TestResolveColumnIndicesVariantSubFieldPruning(t *testing.T) {
	arrowReader := buildShreddedVariantParquet(t)

	t.Run("pruned to color only", func(t *testing.T) {
		fieldSet := map[string]bool{"variant": true}
		pathRoots := singleLevelPathRoots("variant", "color")

		colIndices := resolveColumnIndices(arrowReader.Manifest, fieldSet, pathRoots)
		if colIndices == nil {
			t.Fatalf("expected a non-nil pruned column list")
		}

		totalLeaves := countLeafColumns(arrowReader.Manifest)
		if len(colIndices) >= totalLeaves {
			t.Errorf("expected fewer than %d leaves, got %d: %v", totalLeaves, len(colIndices), colIndices)
		}

		rows := decodeVariantColumn(t, arrowReader, colIndices)
		if len(rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(rows))
		}
		for i, obj := range rows {
			if _, present := obj["size"]; present {
				t.Errorf("row %d: expected 'size' absent from pruned decode, got %v", i, obj)
			}
			if _, present := obj["color"]; !present {
				t.Errorf("row %d: expected 'color' present, decoded object: %v", i, obj)
			}
		}
	})

	t.Run("bare field request reads everything (no regression)", func(t *testing.T) {
		fieldSet := map[string]bool{"variant": true}

		colIndices := resolveColumnIndices(arrowReader.Manifest, fieldSet, nil)
		rows := decodeVariantColumn(t, arrowReader, colIndices)
		if len(rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(rows))
		}
		for i, obj := range rows {
			if _, present := obj["color"]; !present {
				t.Errorf("row %d: expected 'color' present, decoded object: %v", i, obj)
			}
			if _, present := obj["size"]; !present {
				t.Errorf("row %d: expected 'size' present (whole-field read), decoded object: %v", i, obj)
			}
		}
	})
}

// TestResolveColumnIndicesVariantNestedSubFieldPruning exercises arbitrary-depth
// projection pruning end to end: narrowing to "variant.geo.floor" (a shredded
// sub-field that is itself a nested shredded struct) must drop "zone"'s columns
// while still decoding "floor" correctly via collectSubtree's one-level-deeper
// recursion through collectVariantLeaves.
func TestResolveColumnIndicesVariantNestedSubFieldPruning(t *testing.T) {
	_, arrowReader := buildNestedShreddedVariantParquetRowGroups(t, []int64{1, 2}, []int64{3, 4})

	fieldSet := map[string]bool{"variant": true}
	pathRoots := map[string]*pathNode{
		"variant": {children: map[string]*pathNode{
			"geo": {children: map[string]*pathNode{
				"floor": {whole: true},
			}},
		}},
	}

	colIndices := resolveColumnIndices(arrowReader.Manifest, fieldSet, pathRoots)
	if colIndices == nil {
		t.Fatalf("expected a non-nil pruned column list")
	}
	totalLeaves := countLeafColumns(arrowReader.Manifest)
	if len(colIndices) >= totalLeaves {
		t.Errorf("expected fewer than %d leaves, got %d: %v", totalLeaves, len(colIndices), colIndices)
	}

	rows := decodeVariantColumn(t, arrowReader, colIndices)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	for i, obj := range rows {
		geo, ok := obj["geo"].(map[string]interface{})
		if !ok {
			t.Fatalf("row %d: expected 'geo' present as an object, decoded object: %v", i, obj)
		}
		if _, present := geo["zone"]; present {
			t.Errorf("row %d: expected 'zone' absent from pruned decode, got %v", i, geo)
		}
		if _, present := geo["floor"]; !present {
			t.Errorf("row %d: expected 'floor' present, decoded object: %v", i, geo)
		}
	}
}

func TestScannerFieldSubNames(t *testing.T) {
	tests := []struct {
		name           string
		selectedFields []string
		want           []string
	}{
		{
			name:           "single sub-path",
			selectedFields: []string{"attrs.color"},
			want:           []string{"attrs.color"},
		},
		{
			name:           "multiple sub-paths, same field",
			selectedFields: []string{"attrs.color", "attrs.size"},
			want:           []string{"attrs.color", "attrs.size"},
		},
		{
			name:           "bare field reference reads the whole field",
			selectedFields: []string{"attrs"},
			want:           []string{"attrs"},
		},
		{
			name:           "mixed bare and narrow reads the whole field",
			selectedFields: []string{"attrs.color", "attrs"},
			want:           []string{"attrs"},
		},
		{
			name:           "deeper than one level narrows to the deepest common point",
			selectedFields: []string{"attrs.geo.lat"},
			want:           []string{"attrs.geo.lat"},
		},
		{
			name:           "unrelated bare field alongside a narrowed one",
			selectedFields: []string{"attrs.color", "id"},
			want:           []string{"attrs.color", "id"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &Scanner{selectedFields: tc.selectedFields}
			got := flattenTestPathRoots(s.fieldSubNames())
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("fieldSubNames() flattened = %v, want %v", got, tc.want)
			}
		})
	}
}

// buildPlainStructParquet writes a 2-row Parquet file with a single plain nested
// struct column named "address" with "city" and "state" sub-fields (no VARIANT
// wrapping at all), and returns an arrow/parquet reader over it.
func buildPlainStructParquet(t *testing.T) *pqarrow.FileReader {
	t.Helper()

	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	t.Cleanup(func() { mem.AssertSize(t, 0) })

	structType := arrow.StructOf(
		arrow.Field{Name: "city", Type: arrow.BinaryTypes.String},
		arrow.Field{Name: "state", Type: arrow.BinaryTypes.String},
	)
	bldr := array.NewStructBuilder(mem, structType)
	defer bldr.Release()

	rows := []struct{ city, state string }{
		{"SF", "CA"},
		{"NYC", "NY"},
	}
	cityBldr := bldr.FieldBuilder(0).(*array.StringBuilder)
	stateBldr := bldr.FieldBuilder(1).(*array.StringBuilder)
	for _, r := range rows {
		bldr.Append(true)
		cityBldr.Append(r.city)
		stateBldr.Append(r.state)
	}

	arr := bldr.NewArray()
	defer arr.Release()

	rec := array.NewRecordBatch(arrow.NewSchema([]arrow.Field{
		{Name: "address", Type: arr.DataType(), Nullable: true},
	}, nil), []arrow.Array{arr}, -1)
	defer rec.Release()

	var buf bytes.Buffer
	wr, err := pqarrow.NewFileWriter(rec.Schema(), &buf,
		parquet.NewWriterProperties(parquet.WithDictionaryDefault(false)),
		pqarrow.DefaultWriterProps())
	if err != nil {
		t.Fatalf("NewFileWriter: %v", err)
	}
	if err := wr.Write(rec); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := wr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	pqReader, err := file.NewParquetReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("NewParquetReader: %v", err)
	}
	t.Cleanup(func() { pqReader.Close() })

	arrowReader, err := pqarrow.NewFileReader(pqReader, pqarrow.ArrowReadProperties{}, memory.DefaultAllocator)
	if err != nil {
		t.Fatalf("NewFileReader: %v", err)
	}
	return arrowReader
}

// decodeStructColumn reads colIndices from arrowReader and decodes every row of the
// (single) "address" column via this package's own Reader.getColumnValue, exactly
// as the production Parquet scan path does.
func decodeStructColumn(t *testing.T, arrowReader *pqarrow.FileReader, colIndices []int) []map[string]interface{} {
	t.Helper()

	rr, err := arrowReader.GetRecordReader(go_context.Background(), colIndices, nil)
	if err != nil {
		t.Fatalf("GetRecordReader: %v", err)
	}
	defer rr.Release()

	helper := &Reader{}
	var rows []map[string]interface{}
	for rr.Next() {
		rec := rr.Record()
		col := rec.Column(0)
		for i := 0; i < int(rec.NumRows()); i++ {
			rows = append(rows, helper.getColumnValue(col, i).(map[string]interface{}))
		}
		rec.Release()
	}
	if err := rr.Err(); err != nil {
		t.Fatalf("record reader error: %v", err)
	}
	return rows
}

// TestResolveColumnIndicesPlainStructSubFieldPruning mirrors
// TestResolveColumnIndicesVariantSubFieldPruning for a plain (non-variant) nested
// struct: narrowing to a requested sub-name must drop the unrequested sibling's
// column index and still decode correctly via Reader.getColumnValue, with the
// unrequested sub-name simply absent (Reader.getStructValue builds its result
// dynamically from whatever fields are actually present).
func TestResolveColumnIndicesPlainStructSubFieldPruning(t *testing.T) {
	arrowReader := buildPlainStructParquet(t)

	t.Run("pruned to city only", func(t *testing.T) {
		fieldSet := map[string]bool{"address": true}
		pathRoots := singleLevelPathRoots("address", "city")

		colIndices := resolveColumnIndices(arrowReader.Manifest, fieldSet, pathRoots)
		if colIndices == nil {
			t.Fatalf("expected a non-nil pruned column list")
		}
		totalLeaves := countLeafColumns(arrowReader.Manifest)
		if len(colIndices) >= totalLeaves {
			t.Errorf("expected fewer than %d leaves, got %d: %v", totalLeaves, len(colIndices), colIndices)
		}

		rows := decodeStructColumn(t, arrowReader, colIndices)
		if len(rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(rows))
		}
		for i, obj := range rows {
			if _, present := obj["state"]; present {
				t.Errorf("row %d: expected 'state' absent from pruned decode, got %v", i, obj)
			}
			if _, present := obj["city"]; !present {
				t.Errorf("row %d: expected 'city' present, decoded object: %v", i, obj)
			}
		}
	})

	t.Run("bare field request reads everything (no regression)", func(t *testing.T) {
		fieldSet := map[string]bool{"address": true}

		colIndices := resolveColumnIndices(arrowReader.Manifest, fieldSet, nil)
		rows := decodeStructColumn(t, arrowReader, colIndices)
		if len(rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(rows))
		}
		for i, obj := range rows {
			if _, present := obj["city"]; !present {
				t.Errorf("row %d: expected 'city' present, decoded object: %v", i, obj)
			}
			if _, present := obj["state"]; !present {
				t.Errorf("row %d: expected 'state' present (whole-field read), decoded object: %v", i, obj)
			}
		}
	})
}
