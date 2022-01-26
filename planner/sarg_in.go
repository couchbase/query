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
	"github.com/couchbase/query/value"
)

func (this *sarg) VisitIn(pred *expression.In) (interface{}, error) {
	if base.SubsetOf(pred, this.key) {
		if expression.Equivalent(pred, this.key) {
			return _EXACT_SELF_SPANS, nil
		}
		return _SELF_SPANS, nil
	}

	if !pred.First().EquivalentTo(this.key) {
		if pred.DependsOn(this.key) {
			return _VALUED_SPANS, nil
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

	aval := pred.Second().Value()
	if aval != nil {
		vals, ok := aval.Actual().([]interface{})
		if !ok || len(vals) == 0 {
			return _EMPTY_SPANS, nil
		}

		// De-dup and Sort before generating spans for EXPLAIN stability
		vals = expression.SortInList(vals)

		array = make(expression.Expressions, len(vals))
		for i, val := range vals {
			array[i] = expression.NewConstant(val)
		}
	}

	if array == nil {
		second := pred.Second()
		if acons, ok := second.(*expression.ArrayConstruct); ok {
			array = acons.Operands()
		} else {
			static := this.getSarg(second)
			if static == nil {
				return _VALUED_SPANS, nil
			}

			selec := OPT_SELEC_NOT_AVAIL
			if this.doSelec {
				selec = optDefInSelec(this.baseKeyspace.Keyspace(), this.key.String(), this.advisorValidate)
			}
			static.SetExprFlag(expression.EXPR_DYNAMIC_IN)
			range2 := plan.NewRange2(expression.NewArrayMin(static), expression.NewArrayMax(static), datastore.BOTH, selec, OPT_SELEC_NOT_AVAIL, 0)
			span := plan.NewSpan2(nil, plan.Ranges2{range2}, false)
			return NewTermSpans(span), nil
		}
	}

	if len(array) == 0 {
		return _EMPTY_SPANS, nil
	}

	spans := make(plan.Spans2, 0, len(array))
	var keyspaces map[string]string
	var err error
	if this.doSelec && !this.isJoin {
		keyspaces = make(map[string]string, 1)
		keyspaces[this.baseKeyspace.Name()] = this.baseKeyspace.Keyspace()
	}
	for _, elem := range array {
		static := this.getSarg(elem)
		if static == nil {
			return _VALUED_SPANS, nil
		}

		val := static.Value()
		if val != nil && val.Type() <= value.NULL {
			continue
		}

		selec := OPT_SELEC_NOT_AVAIL
		if this.doSelec {
			newExpr := expression.NewEq(pred.First(), static)
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
