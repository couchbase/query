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
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
)

type dynamicKey struct {
	variable *expression.Identifier
	pairs    *expression.Pairs
}

func (this *builder) buildDynamicScan(node *algebra.KeyspaceTerm,
	id, pred expression.Expression, arrays map[datastore.Index]*indexEntry,
	primaryKey expression.Expressions, formalizer *expression.Formalizer) (
	op plan.SecondaryScan, sargLength int, err error) {

	// Prevent infinite recursion
	if this.skipDynamic {
		return nil, 0, nil
	}

	skipDynamic := this.skipDynamic
	defer func() { this.skipDynamic = skipDynamic }()
	this.skipDynamic = true

	var index datastore.Index
	var dk *dynamicKey
	alias := expression.NewIdentifier(node.Alias())

outer:
	for i, e := range arrays {
		if e.cond != nil && !SubsetOf(pred, e.cond) {
			continue
		}

		for _, k := range e.keys {
			if dk = toDynamicKey(alias, pred, k); dk != nil {
				index = i
				break outer
			}
		}
	}

	if index == nil {
		return nil, 0, nil
	}

	newPred, err := DynamicFor(pred, dk.variable, dk.pairs)
	if err != nil || newPred.EquivalentTo(pred) {
		return nil, 0, err
	}

	return this.buildTermScan(node, id, newPred, nil, []datastore.Index{index}, primaryKey, formalizer)
}

func toDynamicKey(alias *expression.Identifier, pred, key expression.Expression) *dynamicKey {
	if all, ok := key.(*expression.All); ok {
		if array, ok := all.Array().(*expression.Array); ok && len(array.Bindings()) == 1 {
			binding := array.Bindings()[0]

			if variable, ok := array.ValueMapping().(*expression.Identifier); ok &&
				variable.Identifier() == binding.Variable() {
				if pairs, ok := binding.Expression().(*expression.Pairs); ok {
					scope := pairs.Operand()
					if scope.EquivalentTo(alias) ||
						pred.CoveredBy(alias.Identifier(), expression.Expressions{scope}) {
						return &dynamicKey{
							variable: variable,
							pairs:    pairs,
						}
					}
				}
			}
		}
	}

	return nil
}
