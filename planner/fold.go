//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"github.com/couchbase/query/expression"
)

// Fold redundant AND and OR terms in the predicate
func Fold(pred expression.Expression) (expression.Expression, error) {
	f := newFold()
	rv, err := pred.Accept(f)
	if err != nil {
		return nil, err
	}

	return rv.(expression.Expression), nil
}

type fold struct {
	expression.MapperBase
}

func newFold() *fold {
	rv := &fold{}

	rv.SetMapper(rv)
	rv.SetMapFunc(
		func(expr expression.Expression) (expression.Expression, error) {
			return expr, nil
		})

	return rv
}

// Logic

func (this *fold) VisitAnd(expr *expression.And) (interface{}, error) {
	operands := expr.Operands()
	found := false

outer:
	for i, lhs := range operands[0 : len(operands)-1] {
		for j := i + 1; j < len(operands); j++ {
			rhs := operands[j]
			if rhs == nil {
				continue
			}

			if SubsetOf(rhs, lhs) {
				operands[i] = nil
				found = true
				continue outer
			} else if SubsetOf(lhs, rhs) {
				operands[j] = nil
				found = true
			}
		}
	}

	if !found {
		return expr, nil
	}

	terms := make(expression.Expressions, 0, len(operands))
	for _, op := range operands {
		if op != nil {
			terms = append(terms, op)
		}
	}

	if len(terms) == 1 {
		return terms[0], nil
	} else {
		return expression.NewAnd(terms...), nil
	}
}

func (this *fold) VisitOr(expr *expression.Or) (interface{}, error) {
	operands := expr.Operands()
	found := false

outer:
	for i, lhs := range operands[0 : len(operands)-1] {
		if lhs == nil {
			continue
		}

		for j := i + 1; j < len(operands); j++ {
			rhs := operands[j]
			if rhs == nil {
				continue
			}

			if SubsetOf(lhs, rhs) {
				operands[i] = nil
				found = true
				continue outer
			} else if SubsetOf(rhs, lhs) {
				operands[j] = nil
				found = true
			}
		}
	}

	if !found {
		return expr, nil
	}

	terms := make(expression.Expressions, 0, len(operands))
	for _, op := range operands {
		if op != nil {
			terms = append(terms, op)
		}
	}

	if len(terms) == 1 {
		return terms[0], nil
	} else {
		return expression.NewOr(terms...), nil
	}
}
