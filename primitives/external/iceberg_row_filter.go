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
	"math"
	"reflect"
	"strings"

	"github.com/apache/arrow-go/v18/parquet/metadata"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
	"github.com/apache/iceberg-go"
	"github.com/couchbase/query/logging"
)

// rowMatcher reports whether a converted row passes the pushed-down filter.
// A nil rowMatcher means no filter is applied; every row matches.
type rowMatcher func(row map[string]interface{}) bool

// buildRowMatcher returns a row-level evaluator for an iceberg filter expression.
// The parallel-files scan path reads raw Parquet/Avro/Arrow/ORC bytes directly and
// bypasses iceberg-go's ToArrowRecords() row filtering, so without this evaluator
// pushed-down filters affect only file/row-group pruning and every surviving row
// is emitted unfiltered. Returns (nil, nil) when no filtering is needed.
func buildRowMatcher(schema *iceberg.Schema, expr iceberg.BooleanExpression) (rowMatcher, error) {
	if expr == nil {
		return nil, nil
	}
	if _, isTrue := expr.(iceberg.AlwaysTrue); isTrue {
		return nil, nil
	}
	if _, isFalse := expr.(iceberg.AlwaysFalse); isFalse {
		return func(map[string]interface{}) bool { return false }, nil
	}

	bound, err := iceberg.BindExpr(schema, expr, true)
	if err != nil {
		return nil, err
	}
	if _, isTrue := bound.(iceberg.AlwaysTrue); isTrue {
		return nil, nil
	}
	if _, isFalse := bound.(iceberg.AlwaysFalse); isFalse {
		return func(map[string]interface{}) bool { return false }, nil
	}

	return func(row map[string]interface{}) bool {
		ev := &rowEvaluator{row: row, schema: schema}
		result, err := iceberg.VisitExpr(bound, ev)
		if err != nil {
			logging.Debugf("Iceberg row filter: evaluation error, keeping row: %v", err)
			return true
		}
		return result
	}, nil
}

// rowEvaluator implements iceberg.BoundBooleanExprVisitor[bool] over a single row
// represented as map[string]interface{}. Values in the row are the Go-native types
// produced by Reader.getColumnValue (string, bool, int32/int64, float32/float64,
// []byte, etc.). Comparisons coerce numeric types so int/float field values can
// still match literals of a different numeric subtype.
//
// The schema lets us recover the dotted column name (e.g. "address.city") from a
// bound reference whose Field() only carries the leaf name. We walk that path
// through nested struct maps in the row.
type rowEvaluator struct {
	row    map[string]interface{}
	schema *iceberg.Schema
}

func (e *rowEvaluator) VisitTrue() bool                { return true }
func (e *rowEvaluator) VisitFalse() bool               { return false }
func (e *rowEvaluator) VisitNot(child bool) bool       { return !child }
func (e *rowEvaluator) VisitAnd(left, right bool) bool { return left && right }
func (e *rowEvaluator) VisitOr(left, right bool) bool  { return left || right }

func (e *rowEvaluator) VisitUnbound(iceberg.UnboundPredicate) bool {
	// BindExpr converts all references; any unbound leaf left here means we
	// couldn't bind it, in which case keep the row rather than silently drop.
	return true
}

func (e *rowEvaluator) VisitBound(pred iceberg.BoundPredicate) bool {
	return iceberg.VisitBoundPredicate[bool](pred, e)
}

func (e *rowEvaluator) fieldValue(term iceberg.BoundTerm) (interface{}, bool) {
	fieldID := term.Ref().Field().ID
	path, ok := e.schema.FindColumnName(fieldID)
	if !ok {
		return nil, false
	}
	return navigateRow(e.row, path)
}

// navigateRow walks a dotted column path through nested struct maps.
// For "address.city" it descends row["address"]["city"]. Returns false if any
// segment is missing or the intermediate value isn't a string-keyed map.
//
// Some format readers return nested structs as named map types
// (e.g. scritchley/orc.Struct = map[string]interface{}), so we fall back to
// reflection when the direct type assertion fails.
func navigateRow(row map[string]interface{}, path string) (interface{}, bool) {
	if !strings.Contains(path, ".") {
		v, ok := row[path]
		return v, ok
	}
	var cur interface{} = row
	start := 0
	for i := 0; i <= len(path); i++ {
		if i == len(path) || path[i] == '.' {
			seg := path[start:i]
			v, ok := lookupMapKey(cur, seg)
			if !ok {
				return nil, false
			}
			cur = v
			start = i + 1
		}
	}
	return cur, true
}

// lookupMapKey reads a key from a string-keyed map value, accepting both
// plain map[string]interface{} and named types whose underlying type is one.
func lookupMapKey(m interface{}, key string) (interface{}, bool) {
	if mm, ok := m.(map[string]interface{}); ok {
		v, ok := mm[key]
		return v, ok
	}
	rv := reflect.ValueOf(m)
	if rv.Kind() != reflect.Map || rv.Type().Key().Kind() != reflect.String {
		return nil, false
	}
	val := rv.MapIndex(reflect.ValueOf(key))
	if !val.IsValid() {
		return nil, false
	}
	return val.Interface(), true
}

func (e *rowEvaluator) VisitIsNull(term iceberg.BoundTerm) bool {
	v, ok := e.fieldValue(term)
	return !ok || v == nil
}

func (e *rowEvaluator) VisitNotNull(term iceberg.BoundTerm) bool {
	v, ok := e.fieldValue(term)
	return ok && v != nil
}

func (e *rowEvaluator) VisitIsNan(term iceberg.BoundTerm) bool {
	v, ok := e.fieldValue(term)
	if !ok || v == nil {
		return false
	}
	switch f := v.(type) {
	case float64:
		return math.IsNaN(f)
	case float32:
		return math.IsNaN(float64(f))
	}
	return false
}

func (e *rowEvaluator) VisitNotNan(term iceberg.BoundTerm) bool {
	v, ok := e.fieldValue(term)
	if !ok || v == nil {
		return false
	}
	switch f := v.(type) {
	case float64:
		return !math.IsNaN(f)
	case float32:
		return !math.IsNaN(float64(f))
	}
	return true
}

func (e *rowEvaluator) VisitEqual(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	v, ok := e.fieldValue(term)
	if !ok || v == nil {
		return false
	}
	c, ok := compareToLiteral(v, lit)
	return ok && c == 0
}

func (e *rowEvaluator) VisitNotEqual(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	v, ok := e.fieldValue(term)
	if !ok || v == nil {
		return false
	}
	c, ok := compareToLiteral(v, lit)
	return ok && c != 0
}

func (e *rowEvaluator) VisitGreaterEqual(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	v, ok := e.fieldValue(term)
	if !ok || v == nil {
		return false
	}
	c, ok := compareToLiteral(v, lit)
	return ok && c >= 0
}

func (e *rowEvaluator) VisitGreater(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	v, ok := e.fieldValue(term)
	if !ok || v == nil {
		return false
	}
	c, ok := compareToLiteral(v, lit)
	return ok && c > 0
}

func (e *rowEvaluator) VisitLessEqual(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	v, ok := e.fieldValue(term)
	if !ok || v == nil {
		return false
	}
	c, ok := compareToLiteral(v, lit)
	return ok && c <= 0
}

func (e *rowEvaluator) VisitLess(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	v, ok := e.fieldValue(term)
	if !ok || v == nil {
		return false
	}
	c, ok := compareToLiteral(v, lit)
	return ok && c < 0
}

func (e *rowEvaluator) VisitStartsWith(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	v, ok := e.fieldValue(term)
	if !ok || v == nil {
		return false
	}
	s, sOK := v.(string)
	p, pOK := lit.Any().(string)
	if !sOK || !pOK {
		return false
	}
	return strings.HasPrefix(s, p)
}

func (e *rowEvaluator) VisitNotStartsWith(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	v, ok := e.fieldValue(term)
	if !ok || v == nil {
		return false
	}
	s, sOK := v.(string)
	p, pOK := lit.Any().(string)
	if !sOK || !pOK {
		return true
	}
	return !strings.HasPrefix(s, p)
}

func (e *rowEvaluator) VisitIn(term iceberg.BoundTerm, lits iceberg.Set[iceberg.Literal]) bool {
	v, ok := e.fieldValue(term)
	if !ok || v == nil {
		return false
	}
	for _, l := range lits.Members() {
		if c, ok := compareToLiteral(v, l); ok && c == 0 {
			return true
		}
	}
	return false
}

func (e *rowEvaluator) VisitNotIn(term iceberg.BoundTerm, lits iceberg.Set[iceberg.Literal]) bool {
	v, ok := e.fieldValue(term)
	if !ok || v == nil {
		return false
	}
	for _, l := range lits.Members() {
		if c, ok := compareToLiteral(v, l); ok && c == 0 {
			return false
		}
	}
	return true
}

// compareToLiteral compares a row value to an iceberg literal, returning
// -1/0/1 like bytes.Compare. The second result is false if the two values
// aren't comparable (e.g. comparing a string row value to a numeric literal).
func compareToLiteral(rowVal interface{}, lit iceberg.Literal) (int, bool) {
	return compareValues(rowVal, lit.Any())
}

// compareValues is compareToLiteral's core comparison, taking a plain Go value
// instead of an iceberg.Literal -- reusable from contexts with no Iceberg literal
// to unwrap in the first place (see buildVariantRowGroupMatcher, which works from
// N1QL-extracted Go values directly since VARIANT sub-paths can't bind to an
// iceberg.Literal at all).
func compareValues(rowVal interface{}, litVal interface{}) (int, bool) {
	switch lv := litVal.(type) {
	case string:
		if rv, ok := rowVal.(string); ok {
			return strings.Compare(rv, lv), true
		}
		if rv, ok := rowVal.([]byte); ok {
			return strings.Compare(string(rv), lv), true
		}
	case bool:
		if rv, ok := rowVal.(bool); ok {
			switch {
			case rv == lv:
				return 0, true
			case !rv && lv:
				return -1, true
			default:
				return 1, true
			}
		}
	case []byte:
		if rv, ok := rowVal.([]byte); ok {
			return bytes.Compare(rv, lv), true
		}
		if rv, ok := rowVal.(string); ok {
			return bytes.Compare([]byte(rv), lv), true
		}
	case int32, int64, int16, int8, uint8, uint16, uint32, uint64:
		li, _ := toInt64(lv)
		return compareRowValToInt64(rowVal, li)
	case iceberg.Date:
		return compareRowValToInt64(rowVal, int64(lv))
	case iceberg.Time:
		return compareRowValToInt64(rowVal, int64(lv))
	case iceberg.Timestamp:
		return compareRowValToInt64(rowVal, int64(lv))
	case float32, float64:
		lf, _ := toFloat64(lv)
		return compareRowValToFloat64(rowVal, lf)
	}
	return 0, false
}

// compareRowValToInt64 compares a row value against an int64-valued literal,
// falling back to a float comparison if the row value isn't itself integral.
func compareRowValToInt64(rowVal interface{}, li int64) (int, bool) {
	if ri, ok := toInt64(rowVal); ok {
		switch {
		case ri < li:
			return -1, true
		case ri > li:
			return 1, true
		default:
			return 0, true
		}
	}
	return compareRowValToFloat64(rowVal, float64(li))
}

// compareRowValToFloat64 compares a row value against a float64-valued literal.
func compareRowValToFloat64(rowVal interface{}, lf float64) (int, bool) {
	if rf, ok := toFloat64(rowVal); ok {
		switch {
		case rf < lf:
			return -1, true
		case rf > lf:
			return 1, true
		default:
			return 0, true
		}
	}
	return 0, false
}

func toInt64(v interface{}) (int64, bool) {
	switch x := v.(type) {
	case int:
		return int64(x), true
	case int8:
		return int64(x), true
	case int16:
		return int64(x), true
	case int32:
		return int64(x), true
	case int64:
		return x, true
	case uint8:
		return int64(x), true
	case uint16:
		return int64(x), true
	case uint32:
		return int64(x), true
	case uint64:
		return int64(x), true
	}
	return 0, false
}

// rowGroupMatcher reports whether a Parquet row group could contain rows that
// satisfy the pushed-down filter, using only the row group's column-level
// statistics (min/max/null-count) from the Parquet footer -- no row data is
// read to make this decision. A nil rowGroupMatcher means no filter is
// applied; every row group is kept. It is deliberately conservative: it may
// return true (keep) in cases a tighter reading of the statistics would allow
// skipping, but it must never return false for a row group that could
// actually contain a match.
type rowGroupMatcher func(rgMeta *metadata.RowGroupMetaData) bool

// buildRowGroupMatcher returns a row-group-level evaluator for an iceberg filter
// expression, for pruning Parquet row groups via their footer statistics before
// any column data is downloaded or decoded. colIndexByPath maps each leaf
// column's dotted schema path (as returned by iceberg.Schema.FindColumnName) to
// its physical column index in this specific file (see buildColumnPathIndex).
// Returns (nil, nil) when no filtering is needed.
func buildRowGroupMatcher(schema *iceberg.Schema, expr iceberg.BooleanExpression,
	colIndexByPath map[string]int) (rowGroupMatcher, error) {

	if expr == nil {
		return nil, nil
	}
	if _, isTrue := expr.(iceberg.AlwaysTrue); isTrue {
		return nil, nil
	}
	if _, isFalse := expr.(iceberg.AlwaysFalse); isFalse {
		return func(*metadata.RowGroupMetaData) bool { return false }, nil
	}

	bound, err := iceberg.BindExpr(schema, expr, true)
	if err != nil {
		return nil, err
	}
	if _, isTrue := bound.(iceberg.AlwaysTrue); isTrue {
		return nil, nil
	}
	if _, isFalse := bound.(iceberg.AlwaysFalse); isFalse {
		return func(*metadata.RowGroupMetaData) bool { return false }, nil
	}

	return func(rgMeta *metadata.RowGroupMetaData) bool {
		ev := &rowGroupEvaluator{schema: schema, colIndexByPath: colIndexByPath, rgMeta: rgMeta}
		result, err := iceberg.VisitExpr(bound, ev)
		if err != nil {
			logging.Debugf("Iceberg row-group filter: evaluation error, keeping row group: %v", err)
			return true
		}
		return result
	}, nil
}

// buildColumnPathIndex maps each leaf column's dotted schema path (matching
// iceberg.Schema.FindColumnName's output, e.g. "address.city") to its physical
// leaf column index in this specific parquet file, for row-group statistics
// lookups. Unlike resolveColumnIndices (which only needs top-level names for
// column-projection pruning), row-group pruning must resolve filter references
// at any nesting depth.
func buildColumnPathIndex(manifest *pqarrow.SchemaManifest) map[string]int {
	idx := make(map[string]int)
	var walk func(f *pqarrow.SchemaField, prefix string)
	walk = func(f *pqarrow.SchemaField, prefix string) {
		path := f.Field.Name
		if prefix != "" {
			path = prefix + "." + path
		}
		if len(f.Children) == 0 {
			idx[path] = f.ColIndex
			return
		}
		for i := range f.Children {
			walk(&f.Children[i], path)
		}
	}
	for i := range manifest.Fields {
		walk(&manifest.Fields[i], "")
	}
	return idx
}

// findVariantLeaf locates the physical Parquet leaf column for a shredded VARIANT
// path "top.path[0].path[1]..." (e.g. top="attrs", path=["geo","lat"]), to
// arbitrary depth, directly in the file's pqarrow schema manifest -- no Iceberg
// schema/field-ID involved, since a VARIANT sub-path has none there.
//
// ok is false when top isn't a top-level field in this file or matchShreddedPath
// can't resolve path within it.
func findVariantLeaf(manifest *pqarrow.SchemaManifest, top string, path []string) (colIdx int, ok bool) {
	for i := range manifest.Fields {
		field := &manifest.Fields[i]
		if field.Field == nil || field.Field.Name != top {
			continue
		}
		return matchShreddedPath(field, path)
	}
	return 0, false // top not a top-level field in this file
}

// matchShreddedPath resolves path within f's shredded "typed_value" struct, one
// segment at a time. Mirrors collectVariantLeaves' "find the typed_value child"
// pattern in iceberg_scanner.go, generalized to arbitrary depth.
//
// Per the Parquet Variant shredding spec, even a scalar shredded sub-field is
// itself wrapped in its own {value, typed_value} pair (mirroring the top-level
// variant's own shape), to support per-row shredded/not-shredded fallback -- so
// matching path[0] among typed_value's children isn't the leaf itself unless
// that child has no children of its own (some shredding layouts omit the
// wrapper for a scalar); otherwise, on the *last* segment, the leaf is that
// child's own "typed_value" grandchild (the stats there are what's actually
// decodable and comparable -- the child's "value" sibling is raw
// variant-encoded bytes). A nested shredded *struct* sub-field (e.g. "geo") has
// exactly that same {value, typed_value} shape one level down, so any segment
// before the last simply recurses matchShreddedPath on the matched child.
//
// ok is false when f isn't a shredded VARIANT/sub-field (no "typed_value"
// child), path[0] isn't among typed_value's children, or the match's shape
// doesn't fit what the remaining path expects (a scalar wrapper with segments
// left over, or a struct wrapper on the last segment with no "typed_value"
// leaf of its own).
func matchShreddedPath(f *pqarrow.SchemaField, path []string) (colIdx int, ok bool) {
	if len(path) == 0 {
		return 0, false
	}

	var typedValue *pqarrow.SchemaField
	for i := range f.Children {
		if f.Children[i].Field != nil && f.Children[i].Field.Name == "typed_value" {
			typedValue = &f.Children[i]
			break
		}
	}
	if typedValue == nil {
		return 0, false
	}

	for i := range typedValue.Children {
		child := &typedValue.Children[i]
		if child.Field == nil || child.Field.Name != path[0] {
			continue
		}
		if len(child.Children) == 0 {
			// Already a direct leaf -- some shredding layouts may omit the
			// per-field value/typed_value wrapper for a scalar.
			if len(path) == 1 {
				return child.ColIndex, true
			}
			return 0, false // more segments requested past a scalar leaf
		}
		if len(path) == 1 {
			for j := range child.Children {
				gc := &child.Children[j]
				if gc.Field != nil && gc.Field.Name == "typed_value" && len(gc.Children) == 0 {
					return gc.ColIndex, true
				}
			}
			return 0, false // shreds to something more complex than a scalar wrapper
		}
		return matchShreddedPath(child, path[1:])
	}
	return 0, false // path[0] not among typed_value's children
}

// buildVariantRowGroupMatcher returns a row-group-level evaluator for candidate
// VARIANT sub-path predicates (see extractVariantPredicates), pruning row groups
// via each sub-path's physical Parquet column statistics -- entirely independent
// of buildRowGroupMatcher/Iceberg schema binding, which can't reach a VARIANT
// sub-path at all. Purely additive: the caller ANDs this together with the
// existing schema-bound matcher, so a nil/no-op result here (no predicates,
// unresolvable leaf, missing/unusable stats) only ever means "no extra pruning",
// never a wrong result. Returns nil when preds is empty.
func buildVariantRowGroupMatcher(manifest *pqarrow.SchemaManifest, preds []variantPredicate) rowGroupMatcher {
	if len(preds) == 0 {
		return nil
	}

	type resolved struct {
		colIdx int
		op     iceberg.Operation
		lit    interface{}
	}
	var leaves []resolved
	for _, p := range preds {
		segs := strings.Split(p.path, ".")
		if len(segs) < 2 {
			continue // shouldn't happen (extractVariantPredicates only emits 2+ segment paths)
		}
		colIdx, ok := findVariantLeaf(manifest, segs[0], segs[1:])
		if !ok {
			continue
		}
		leaves = append(leaves, resolved{colIdx: colIdx, op: p.op, lit: p.lit})
	}
	if len(leaves) == 0 {
		return nil
	}

	return func(rgMeta *metadata.RowGroupMetaData) bool {
		for _, r := range leaves {
			cs := statsForColIndex(rgMeta, r.colIdx)
			if !cs.ok || !cs.hasMinMax {
				continue // can't reason about this predicate; doesn't disqualify the row group
			}
			if !variantStatsCouldMatch(r.op, cs.min, cs.max, r.lit) {
				return false
			}
		}
		return true
	}
}

// variantStatsCouldMatch answers "could some row in this row group satisfy
// `col OP lit`?" from only the column's min/max, mirroring
// rowGroupEvaluator.VisitEqual/VisitGreaterEqual/etc.'s range-check logic exactly
// (same compareValues comparator), just without an iceberg.BoundTerm/Literal to
// visit through.
func variantStatsCouldMatch(op iceberg.Operation, min, max, lit interface{}) bool {
	switch op {
	case iceberg.OpEQ:
		cmin, okMin := compareValues(min, lit)
		cmax, okMax := compareValues(max, lit)
		return !okMin || !okMax || (cmin <= 0 && cmax >= 0)
	case iceberg.OpLT:
		c, ok := compareValues(min, lit)
		return !ok || c < 0
	case iceberg.OpLTEQ:
		c, ok := compareValues(min, lit)
		return !ok || c <= 0
	case iceberg.OpGT:
		c, ok := compareValues(max, lit)
		return !ok || c > 0
	case iceberg.OpGTEQ:
		c, ok := compareValues(max, lit)
		return !ok || c >= 0
	default:
		return true // unsupported op -- keep
	}
}

// buildVariantRowMatcher returns a row-level evaluator for candidate VARIANT
// sub-path predicates (see extractVariantPredicates), applied to already-decoded
// rows -- unlike buildVariantRowGroupMatcher (which reasons about Parquet column
// statistics before any row is read, so it can only ever prove "no match" from a
// range), this checks the actual decoded value via navigateRow (the same helper
// buildRowMatcher/rowEvaluator use), so it's an exact match/no-match rather than a
// range-based "could match". An unresolvable path or incomparable types keep the
// row rather than risk dropping one that does match. Returns nil when preds is
// empty.
func buildVariantRowMatcher(preds []variantPredicate) rowMatcher {
	if len(preds) == 0 {
		return nil
	}
	return func(row map[string]interface{}) bool {
		for _, p := range preds {
			val, ok := navigateRow(row, p.path)
			if !ok {
				continue // can't resolve -- don't disqualify the row
			}
			if !variantValueMatches(p.op, val, p.lit) {
				return false
			}
		}
		return true
	}
}

// variantValueMatches answers `val OP lit` exactly, given the row's actual decoded
// value -- unlike variantStatsCouldMatch's min/max range check, this can return a
// definite no-match.
func variantValueMatches(op iceberg.Operation, val, lit interface{}) bool {
	c, ok := compareValues(val, lit)
	if !ok {
		return true // can't compare these types -- don't drop the row
	}
	switch op {
	case iceberg.OpEQ:
		return c == 0
	case iceberg.OpLT:
		return c < 0
	case iceberg.OpLTEQ:
		return c <= 0
	case iceberg.OpGT:
		return c > 0
	case iceberg.OpGTEQ:
		return c >= 0
	default:
		return true // unsupported op -- keep
	}
}

// rowGroupEvaluator implements iceberg.BoundBooleanExprVisitor[bool] over a row
// group's Parquet column statistics instead of a concrete row. Each predicate
// visit answers "could some row in this row group satisfy this condition?"
// using only min/max/null-count -- it never inspects actual values, so on any
// ambiguity (missing stats, an unsupported stat type, or an operator that
// can't be reasoned about from a range alone, e.g. NOT/STARTS_WITH/IS_NAN) it
// returns true (keep) rather than risk skipping a row group with a match.
type rowGroupEvaluator struct {
	schema         *iceberg.Schema
	colIndexByPath map[string]int
	rgMeta         *metadata.RowGroupMetaData
}

func (e *rowGroupEvaluator) VisitTrue() bool         { return true }
func (e *rowGroupEvaluator) VisitFalse() bool        { return false }
func (e *rowGroupEvaluator) VisitNot(bool) bool      { return true } // can't invert a range check safely; keep
func (e *rowGroupEvaluator) VisitAnd(l, r bool) bool { return l && r }
func (e *rowGroupEvaluator) VisitOr(l, r bool) bool  { return l || r }

func (e *rowGroupEvaluator) VisitUnbound(iceberg.UnboundPredicate) bool { return true }

func (e *rowGroupEvaluator) VisitBound(pred iceberg.BoundPredicate) bool {
	return iceberg.VisitBoundPredicate[bool](pred, e)
}

// colStats holds one column's row-group statistics decoded into the same
// Go-native types compareToLiteral expects. ok is false when stats couldn't be
// resolved at all (unmapped column, read error, or an unsupported physical
// stat type); callers must treat that as "keep, can't prune".
type colStats struct {
	min, max     interface{}
	hasMinMax    bool
	nullCount    int64
	hasNullCount bool
	numValues    int64
	ok           bool
}

func (e *rowGroupEvaluator) statsFor(term iceberg.BoundTerm) colStats {
	fieldID := term.Ref().Field().ID
	path, ok := e.schema.FindColumnName(fieldID)
	if !ok {
		return colStats{}
	}
	colIdx, ok := e.colIndexByPath[path]
	if !ok {
		return colStats{}
	}
	return statsForColIndex(e.rgMeta, colIdx)
}

// statsForColIndex decodes a single physical column's row-group statistics into
// colStats, given only its leaf column index -- no Iceberg schema/field-ID lookup
// involved, so it's equally usable from a schema-bound evaluator (via statsFor
// above) and a physical-layer-only one (see buildVariantRowGroupMatcher).
func statsForColIndex(rgMeta *metadata.RowGroupMetaData, colIdx int) colStats {
	cc, err := rgMeta.ColumnChunk(colIdx)
	if err != nil {
		return colStats{}
	}
	stats, err := cc.Statistics()
	if err != nil || stats == nil {
		return colStats{}
	}
	cs := colStats{
		ok:           true,
		numValues:    stats.NumValues(),
		hasNullCount: stats.HasNullCount(),
		nullCount:    stats.NullCount(),
	}
	if !stats.HasMinMax() {
		return cs
	}
	min, max, ok := decodeMinMax(stats)
	if !ok {
		return cs
	}
	cs.min, cs.max, cs.hasMinMax = min, max, true
	return cs
}

// decodeMinMax decodes a Parquet column's min/max statistics into the Go-native
// types compareValues expects. ok is false for a physical stat type not worth
// decoding (Int96/Float16, or any future type) -- callers must treat that as
// "keep, can't prune", same as a missing/absent stats object entirely.
func decodeMinMax(stats metadata.TypedStatistics) (min, max interface{}, ok bool) {
	switch st := stats.(type) {
	case *metadata.Int32Statistics:
		return int64(st.Min()), int64(st.Max()), true
	case *metadata.Int64Statistics:
		return st.Min(), st.Max(), true
	case *metadata.Float32Statistics:
		return float64(st.Min()), float64(st.Max()), true
	case *metadata.Float64Statistics:
		return st.Min(), st.Max(), true
	case *metadata.BooleanStatistics:
		return st.Min(), st.Max(), true
	case *metadata.ByteArrayStatistics:
		return []byte(st.Min()), []byte(st.Max()), true
	case *metadata.FixedLenByteArrayStatistics:
		return []byte(st.Min()), []byte(st.Max()), true
	default:
		// Int96/Float16 or any future stat type: not worth decoding, keep.
		return nil, nil, false
	}
}

func (e *rowGroupEvaluator) VisitIsNull(term iceberg.BoundTerm) bool {
	cs := e.statsFor(term)
	if !cs.ok || !cs.hasNullCount {
		return true
	}
	return cs.nullCount > 0
}

func (e *rowGroupEvaluator) VisitNotNull(term iceberg.BoundTerm) bool {
	cs := e.statsFor(term)
	if !cs.ok || !cs.hasNullCount {
		return true
	}
	return cs.nullCount < cs.numValues
}

func (e *rowGroupEvaluator) VisitIsNan(iceberg.BoundTerm) bool  { return true }
func (e *rowGroupEvaluator) VisitNotNan(iceberg.BoundTerm) bool { return true }

func (e *rowGroupEvaluator) VisitEqual(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	cs := e.statsFor(term)
	if !cs.ok || !cs.hasMinMax {
		return true
	}
	cmin, okMin := compareToLiteral(cs.min, lit)
	cmax, okMax := compareToLiteral(cs.max, lit)
	if !okMin || !okMax {
		return true
	}
	return cmin <= 0 && cmax >= 0
}

func (e *rowGroupEvaluator) VisitNotEqual(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	cs := e.statsFor(term)
	if !cs.ok || !cs.hasMinMax {
		return true
	}
	// Only provably impossible when every value in the row group equals lit exactly.
	cmin, okMin := compareToLiteral(cs.min, lit)
	cmax, okMax := compareToLiteral(cs.max, lit)
	if !okMin || !okMax {
		return true
	}
	return !(cmin == 0 && cmax == 0)
}

func (e *rowGroupEvaluator) VisitGreaterEqual(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	cs := e.statsFor(term)
	if !cs.ok || !cs.hasMinMax {
		return true
	}
	c, ok := compareToLiteral(cs.max, lit)
	return !ok || c >= 0
}

func (e *rowGroupEvaluator) VisitGreater(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	cs := e.statsFor(term)
	if !cs.ok || !cs.hasMinMax {
		return true
	}
	c, ok := compareToLiteral(cs.max, lit)
	return !ok || c > 0
}

func (e *rowGroupEvaluator) VisitLessEqual(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	cs := e.statsFor(term)
	if !cs.ok || !cs.hasMinMax {
		return true
	}
	c, ok := compareToLiteral(cs.min, lit)
	return !ok || c <= 0
}

func (e *rowGroupEvaluator) VisitLess(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	cs := e.statsFor(term)
	if !cs.ok || !cs.hasMinMax {
		return true
	}
	c, ok := compareToLiteral(cs.min, lit)
	return !ok || c < 0
}

func (e *rowGroupEvaluator) VisitStartsWith(iceberg.BoundTerm, iceberg.Literal) bool {
	return true // prefix range-pruning against byte min/max not implemented; keep
}

func (e *rowGroupEvaluator) VisitNotStartsWith(iceberg.BoundTerm, iceberg.Literal) bool {
	return true
}

func (e *rowGroupEvaluator) VisitIn(term iceberg.BoundTerm, lits iceberg.Set[iceberg.Literal]) bool {
	cs := e.statsFor(term)
	if !cs.ok || !cs.hasMinMax {
		return true
	}
	for _, l := range lits.Members() {
		cmin, okMin := compareToLiteral(cs.min, l)
		cmax, okMax := compareToLiteral(cs.max, l)
		if !okMin || !okMax {
			return true // can't reason about this literal's type; be safe
		}
		if cmin <= 0 && cmax >= 0 {
			return true
		}
	}
	return false
}

func (e *rowGroupEvaluator) VisitNotIn(iceberg.BoundTerm, iceberg.Set[iceberg.Literal]) bool {
	return true // could only prune a single-distinct-value row group; not worth the complexity
}

func toFloat64(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float32:
		return float64(x), true
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	}
	return 0, false
}
