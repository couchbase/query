//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	base "github.com/couchbase/query/plannerbase"
)

func (this *sarg) VisitAnyEvery(pred *expression.AnyEvery) (interface{}, error) {
	if this.isVector {
		return nil, nil
	}

	var spans SargSpans
	if pred.PropagatesNull() {
		spans = _VALUED_SPANS
	} else if pred.PropagatesMissing() {
		spans = _FULL_SPANS
	}

	if base.SubsetOf(pred, this.key) {
		return _SELF_SPANS, nil
	}

	sp := spans
	if !pred.DependsOn(this.key) {
		sp = nil
	}

	all, ok := this.key.(*expression.All)
	if !ok {
		return sp, nil
	}

	arrayId := pred.ArrayId()
	if arrayId <= 0 {
		return nil, errors.NewPlanInternalError(fmt.Sprintf("sarg.VisitAnyEvery: unexpected array id (%d) for ANY AND EVERY expression %v",
			arrayId, pred))
	}

	selec := this.getSelec(pred)

	array, ok := all.Array().(*expression.Array)
	if !ok {
		bindings := pred.Bindings()
		if len(bindings) != 1 ||
			bindings[0].Descend() ||
			!bindings[0].Expression().EquivalentTo(all.Array()) {
			return sp, nil
		}

		variable := expression.NewIdentifier(bindings[0].Variable())
		variable.SetBindingVariable(true)
		return anySargFor(pred.Satisfies(), variable, nil, this.index, this.isJoin, this.doSelec,
			this.baseKeyspace, this.keyspaceNames, variable.Alias(), selec, false,
			this.advisorValidate, false, this.isMissing, this.isVector, this.isInclude,
			this.keyPos, this.aliases, arrayId, this.context)
	}

	if !pred.Bindings().SubsetOf(array.Bindings()) {
		return sp, nil
	}

	satisfies, err := getSatisfies(pred, this.key, array, this.aliases)
	if err != nil {
		return nil, err
	}

	if array.When() != nil && !checkSubset(satisfies, array.When(), this.context) {
		return sp, nil
	}

	// Array Index key can have only single binding
	return anySargFor(satisfies, array.ValueMapping(), array.When(), this.index, this.isJoin,
		this.doSelec, this.baseKeyspace, this.keyspaceNames, array.Bindings()[0].Variable(),
		selec, false, this.advisorValidate, all.IsDerivedFromFlatten(), this.isMissing,
		this.isVector, this.isInclude, this.keyPos, this.aliases, arrayId, this.context)
}
