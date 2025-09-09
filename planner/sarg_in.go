//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

func (this *sarg) VisitIn(pred *expression.In) (interface{}, error) {
	if this.isVector {
		return nil, nil
	} else if base.SubsetOf(pred, this.key) {
		if expression.Equivalent(pred, this.key) {
			return _EXACT_SELF_SPANS, nil
		}
		return _SELF_SPANS, nil
	}

	if !pred.First().EquivalentTo(this.key) {
		if pred.DependsOn(this.key) {
			return getDependsSpans(pred)
		} else {
			return nil, nil
		}
	}

	var array expression.Expressions

	if len(this.context.NamedArgs()) > 0 || len(this.context.PositionalArgs()) > 0 {
		replaced, err := base.ReplaceParameters(pred, this.context.NamedArgs(), this.context.PositionalArgs())
		if err != nil {
			return nil, err
		}
		if repIn, ok := replaced.(*expression.In); ok {
			pred = repIn
		}
	}

	selec := OPT_SELEC_NOT_AVAIL
	defSelec := OPT_SELEC_NOT_AVAIL
	var err error
	var keyspaces map[string]string
	if this.doSelec {
		selec = this.getSelec(pred)
		defSelec = optDefInSelec(this.baseKeyspace.Keyspace(), this.baseKeyspace.Name(),
			this.key, this.advisorValidate)
		if !this.isJoin {
			keyspaces = make(map[string]string, 1)
			keyspaces[this.baseKeyspace.Name()] = this.baseKeyspace.Keyspace()
		}
	}

	first := pred.First()
	second := pred.Second()
	aval := second.Value()
	if aval != nil {
		vals, ok := aval.Actual().([]interface{})
		if !ok || len(vals) == 0 {
			return _EMPTY_SPANS, nil
		}

		// De-dup and Sort before generating spans for EXPLAIN stability
		vals = expression.SortValArr(vals)

		if len(vals) > util.FullSpanFanout(this.isInclude) {
			// for long IN-list, instead of generating individual spans, just use
			// array_min()/array_max() as span and evaluate the IN-list after
			// the index scan
			minVal := expression.NewConstant(vals[0])
			maxVal := expression.NewConstant(vals[len(vals)-1])
			if this.doSelec {
				expr1 := expression.NewLE(minVal, first)
				expr2 := expression.NewLE(first, maxVal)
				exprAnd := expression.NewAnd(expr1, expr2)
				if this.isJoin {
					// for join filter each element of the IN-list may be different
					keyspaces, err = expression.CountKeySpaces(exprAnd, this.keyspaceNames)
					if err != nil {
						return nil, err
					}
				}
				selec, _ = optExprSelec(keyspaces, exprAnd, this.advisorValidate, this.context)
			}
			range2 := plan.NewRange2(minVal, maxVal, datastore.BOTH, selec, OPT_SELEC_NOT_AVAIL, 0)
			span := plan.NewSpan2(nil, plan.Ranges2{range2}, false)
			return NewTermSpans(span), nil
		} else {
			array = make(expression.Expressions, len(vals))
			for i, val := range vals {
				array[i] = expression.NewConstant(val)
			}
		}
	}

	if array == nil {
		arrayMinMax := false
		dynamicIn := false
		var arrayKey expression.Expression
		if acons, ok := second.(*expression.ArrayConstruct); ok {
			array = acons.Operands()
			if len(array) > util.FullSpanFanout(this.isInclude) {
				// for long IN-list, instead of generating individual spans, just use
				// array_min()/array_max() as span and evaluate the IN-list after
				// the index scan
				arrayMinMax = true
				arrayKey = second
				if this.doSelec {
					selec = defSelec
				}
			}
		} else {
			static := this.getSarg(second)
			if static == nil {
				return _VALUED_SPANS, nil
			}

			arrayMinMax = true
			arrayKey = static
			if !this.isInclude {
				dynamicIn = true
			}
		}
		if arrayMinMax {
			if dynamicIn {
				arrayKey.SetExprFlag(expression.EXPR_DYNAMIC_IN)
			}
			range2 := plan.NewRange2(expression.NewArrayMin(arrayKey), expression.NewArrayMax(arrayKey),
				datastore.BOTH, selec, OPT_SELEC_NOT_AVAIL, 0)
			span := plan.NewSpan2(nil, plan.Ranges2{range2}, false)
			return NewTermSpans(span), nil

		}
	}

	if len(array) == 0 {
		return _EMPTY_SPANS, nil
	}

	spans := make(plan.Spans2, 0, len(array))
	for _, elem := range array {
		static := this.getSarg(elem)
		if static == nil {
			return _VALUED_SPANS, nil
		}

		val := static.Value()
		if val != nil && val.Type() <= value.NULL {
			continue
		}

		selec = OPT_SELEC_NOT_AVAIL
		if this.doSelec {
			newExpr := expression.NewEq(first, static)
			if this.isJoin {
				// for join filter each element of the IN-list may be different
				keyspaces, err = expression.CountKeySpaces(newExpr, this.keyspaceNames)
				if err != nil {
					return nil, err
				}
			}
			selec, _ = optExprSelec(keyspaces, newExpr, this.advisorValidate, this.context)
		}
		range2 := plan.NewRange2(static, static, datastore.BOTH, selec, OPT_SELEC_NOT_AVAIL, 0)
		range2.SetFlag(plan.RANGE_FROM_IN_EXPR)
		// set exact to true to allow query parameters, join fields, etc to be able
		// to use covering index scan (static != nil, which is checked above)
		span := plan.NewSpan2(nil, plan.Ranges2{range2}, true)
		spans = append(spans, span)
	}

	if len(spans) == 0 {
		return _EMPTY_SPANS, nil
	}

	return NewTermSpans(spans...), nil
}
