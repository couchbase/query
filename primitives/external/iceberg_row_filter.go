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
	switch lv := lit.Any().(type) {
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
