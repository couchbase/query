//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/value"
)

func (this *sarg) VisitAny(pred *expression.Any) (interface{}, error) {
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
		sp, aid, err := anySargFor(pred.Satisfies(), variable, nil, this.isJoin, this.doSelec,
			this.baseKeyspace, this.keyspaceNames, variable.Alias(), selec, true,
			this.advisorValidate, false, this.isMissing, this.aliases, this.arrayId,
			this.context)
		this.arrayId = aid
		return sp, err
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
	sp, aid, err := anySargFor(satisfies, array.ValueMapping(), array.When(), this.isJoin, this.doSelec,
		this.baseKeyspace, this.keyspaceNames, array.Bindings()[0].Variable(), selec, true,
		this.advisorValidate, all.IsDerivedFromFlatten(), this.isMissing, this.aliases,
		this.arrayId, this.context)
	this.arrayId = aid
	return sp, err
}

func anySargFor(pred, key, cond expression.Expression, isJoin, doSelec bool,
	baseKeyspace *base.BaseKeyspace, keyspaceNames map[string]string, alias string,
	selec float64, any, advisorValidate, flatten, isMissing bool, aliases map[string]bool,
	arrayId int, context *PrepareContext) (SargSpans, int, error) {

	// this is an array predicate
	// note for FLATTEN_KEYS since both the index key format and the ANY clause format
	// should be the same for the expanded index keys, we should end up with the same
	// arrayId for the same ANY/ANY AND EVERY clause when sarging different flattened
	// array index keys
	arrayId++

	sp, _, aid, err := sargFor(pred, key, isJoin, doSelec, baseKeyspace, keyspaceNames,
		advisorValidate, isMissing, true, aliases, arrayId, context)
	if err != nil || sp == nil {
		return sp, aid, err
	}

	if sp.HasStatic() {
		sp = sp.Copy()
	}

	// set the arrayId if not already set (e.g. nested array key with nested ANY clause)
	if arrayId == aid {
		sp.SetArrayId(arrayId)
	}

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
		return sp, aid, nil
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

	return sp, aid, nil
}
