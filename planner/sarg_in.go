//  Copyright (c) 2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/sort"
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

		// De-dup before generating spans
		set := value.NewSet(len(vals), true, false)
		set.AddAll(vals)
		vals = set.Actuals()

		// Sort for EXPLAIN stability
		sort.Sort(value.NewSorter(value.NewValue(vals)))

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
	if !this.isJoin {
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
		span := plan.NewSpan2(nil, plan.Ranges2{range2}, (val != nil))
		spans = append(spans, span)
	}

	if len(spans) == 0 {
		return _EMPTY_SPANS, nil
	}

	return NewTermSpans(spans...), nil
}
