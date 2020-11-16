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
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/util"
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
	alias.SetKeyspaceAlias(true)

outer:
	for i, e := range arrays {
		if e.cond != nil && !base.SubsetOf(pred, e.cond) {
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

	baseKeyspaces := base.CopyBaseKeyspaces(this.baseKeyspaces)
	_, err = ClassifyExpr(newPred, baseKeyspaces, this.keyspaceNames, false, this.useCBO,
		this.advisorValidate(), this.context)
	if err != nil {
		return nil, 0, err
	}
	baseKeyspace, ok := baseKeyspaces[node.Alias()]
	if !ok {
		return nil, 0, errors.NewPlanInternalError(fmt.Sprintf("buildDynamicScan: missing baseKeyspace %s", node.Alias()))
	}
	err = CombineFilters(baseKeyspace, true, false)
	if err != nil {
		return nil, 0, err
	}
	return this.buildTermScan(node, baseKeyspace, id, []datastore.Index{index}, primaryKey, formalizer)
}

func toDynamicKey(alias *expression.Identifier, pred, key expression.Expression) *dynamicKey {
	if all, ok := key.(*expression.All); ok {
		variable := _DEFAULT_PAIRS_VARIABLE
		pairs, ok := all.Array().(*expression.Pairs)

		if !ok {
			if array, ok := all.Array().(*expression.Array); ok && len(array.Bindings()) == 1 {
				binding := array.Bindings()[0]

				if variable, ok = array.ValueMapping().(*expression.Identifier); ok &&
					variable.Identifier() == binding.Variable() {

					pairs, _ = binding.Expression().(*expression.Pairs)
				}
			}
		}

		if pairs != nil {
			scope := pairs.Operand()
			if scope.EquivalentTo(alias) ||
				expression.IsCovered(pred, alias.Identifier(), aliasNamed(scope)) {

				return &dynamicKey{
					variable: variable,
					pairs:    pairs,
				}
			}
		}
	}

	return nil
}

func aliasNamed(expr expression.Expression) expression.Expressions {
	oc, ok := expr.(*expression.ObjectConstruct)
	if !ok {
		return _EMPTY_EXPRESSIONS
	}

	names := _NAMES_POOL.Get()
	defer _NAMES_POOL.Put(names)

	// Skip duplicate names
	mapping := oc.Mapping()
	for name, _ := range mapping {
		names[name.String()]++
	}

	rv := make(expression.Expressions, 0, len(mapping))
	for name, val := range mapping {
		str := name.String()
		if names[str] == 1 && str == expression.NewConstant(val.Alias()).String() {
			rv = append(rv, val)
		}
	}

	return rv
}

var _EMPTY_EXPRESSIONS = expression.Expressions{}
var _NAMES_POOL = util.NewStringIntPool(64)
var _DEFAULT_PAIRS_VARIABLE = expression.NewIdentifier("p")
