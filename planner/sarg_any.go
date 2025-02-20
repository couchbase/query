//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"fmt"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/value"
)

func (this *sarg) VisitAny(pred *expression.Any) (interface{}, error) {
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
		return nil, errors.NewPlanInternalError(fmt.Sprintf("sarg.VisitAny: unexpected array id (%d) for ANY expression %v", arrayId, pred))
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
		return anySargFor(pred.Satisfies(), variable, nil, this.index, this.isJoin,
			this.doSelec, this.baseKeyspace, this.keyspaceNames, variable.Alias(), selec,
			true, this.advisorValidate, false, this.isMissing, this.isVector, this.isInclude,
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
		selec, true, this.advisorValidate, all.IsDerivedFromFlatten(), this.isMissing,
		this.isVector, this.isInclude, this.keyPos, this.aliases, arrayId, this.context)
}

func anySargFor(pred, key, cond expression.Expression, index datastore.Index, isJoin, doSelec bool,
	baseKeyspace *base.BaseKeyspace, keyspaceNames map[string]string, alias string,
	selec float64, any, advisorValidate, flatten, isMissing, isVector, isInclude bool, keyPos int,
	aliases map[string]bool, arrayId int, context *PrepareContext) (SargSpans, error) {

	sp, _, err := sargFor(pred, index, key, isJoin, doSelec, baseKeyspace, keyspaceNames,
		advisorValidate, isMissing, true, isVector, isInclude, keyPos, aliases, context)
	if err != nil || sp == nil {
		return sp, err
	}

	if sp.HasStatic() {
		sp = sp.Copy()
	}

	// set the arrayId
	// if the arrayId is already set (e.g. nested array key with nested ANY clause)
	// then the call here is a no-op (will not overwrite existing arrayId)
	sp.SetArrayId(arrayId)

	if tsp, ok := sp.(*TermSpans); ok {
		spans := tsp.Spans()
		if selec > 0.0 && len(spans) > 1 {
			// distribute selectivity among multiple spans
			selec /= float64(len(spans))
		}
		for _, span := range spans {
			if len(span.Ranges) == 1 {
				span.Ranges[0].Selec1 = selec
				span.Ranges[0].Selec2 = OPT_SELEC_NOT_AVAIL
				if any {
					span.Ranges[0].SetFlag(plan.RANGE_ARRAY_ANY)
				} else {
					span.Ranges[0].SetFlag(plan.RANGE_ARRAY_ANY_EVERY)
				}
			}
		}
	}

	if !sp.Exact() {
		return sp, nil
	}

	exprs := expression.Expressions{key}
	if cond != nil {
		fc := make(map[expression.Expression]value.Value, 4)
		fc = cond.FilterExpressionCovers(fc)
		filterCovers := mapFilterCovers(fc, true)
		for c, _ := range filterCovers {
			exprs = append(exprs, c.Covered())
		}
	}

	if err != nil || (!flatten && !expression.IsArrayCovered(pred, alias, exprs)) {
		sp.SetExact(false)
	}

	return sp, nil
}
