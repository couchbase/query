//  Copyright 2014-Present Couchbase, Inc.
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
)

func (this *sargable) VisitAny(pred *expression.Any) (interface{}, error) {
	if this.vector {
		return false, nil
	} else if this.defaultSargable(pred) {
		return true, nil
	}

	all, ok := this.key.(*expression.All)
	if !ok {
		return false, nil
	}

	var satisfies, mapping expression.Expression
	bindings := pred.Bindings()
	array, ok := all.Array().(*expression.Array)
	if !ok {
		if len(bindings) != 1 || bindings[0].Descend() ||
			!bindings[0].Expression().EquivalentTo(all.Array()) {
			return false, nil
		}
		bindVar := expression.NewIdentifier(bindings[0].Variable())
		bindVar.SetBindingVariable(true)
		mapping = bindVar
		satisfies = pred.Satisfies()
	} else {
		if !bindings.SubsetOf(array.Bindings()) {
			return false, nil
		}
		mapping = array.ValueMapping()

		var err error
		satisfies, err = getSatisfies(pred, this.key, array, this.aliases)
		if err != nil {
			return false, err
		}

		if array.When() != nil && !checkSubset(satisfies, array.When(), this.context) {
			return false, nil
		}
	}

	mappings := datastore.IndexKeys{&datastore.IndexKey{mapping, datastore.IK_NONE}}
	min, _, _, _, _ := SargableFor(satisfies, nil, this.index, mappings, nil, this.missing, this.gsi,
		[]bool{true}, this.context, this.aliases)
	return min > 0, nil
}

func getSatisfies(pred, key expression.Expression, array *expression.Array, aliases map[string]bool) (
	satisfies expression.Expression, err error) {
	var pBindings expression.Bindings
	switch p := pred.(type) {
	case *expression.Any:
		satisfies = p.Satisfies()
		pBindings = p.Bindings()
	case *expression.AnyEvery:
		satisfies = p.Satisfies()
		pBindings = p.Bindings()
	}
	if expression.HasRenameableBindings(pred, key, aliases) == expression.BINDING_VARS_DIFFER {
		renamer := expression.NewRenamer(pBindings, array.Bindings())
		return renamer.Map(satisfies.Copy())
	}
	return satisfies, nil
}
