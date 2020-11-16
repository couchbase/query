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
		return anySargFor(pred.Satisfies(), variable, nil, this.isJoin, this.doSelec,
			this.baseKeyspace, this.keyspaceNames, variable.Alias(), selec, true,
			this.advisorValidate, this.context)
	}

	if !pred.Bindings().SubsetOf(array.Bindings()) {
		return sp, nil
	}

	renamer := expression.NewRenamer(pred.Bindings(), array.Bindings())
	satisfies, err := renamer.Map(pred.Satisfies().Copy())
	if err != nil {
		return nil, err
	}

	if array.When() != nil && !base.SubsetOf(satisfies, array.When()) {
		return sp, nil
	}

	// Array Index key can have only single binding
	return anySargFor(satisfies, array.ValueMapping(), array.When(), this.isJoin, this.doSelec,
		this.baseKeyspace, this.keyspaceNames, array.Bindings()[0].Variable(), selec, true,
		this.advisorValidate, this.context)
}

func anySargFor(pred, key, cond expression.Expression, isJoin, doSelec bool,
	baseKeyspace *base.BaseKeyspace, keyspaceNames map[string]string, alias string,
	selec float64, any, advisorValidate bool, context *PrepareContext) (SargSpans, error) {

	sp, err := sargFor(pred, key, isJoin, doSelec, baseKeyspace, keyspaceNames, advisorValidate, context)
	if err != nil || sp == nil {
		return sp, err
	}

	if tsp, ok := sp.(*TermSpans); ok && tsp.Size() == 1 {
		spans := tsp.Spans()
		if len(spans[0].Ranges) == 1 {
			spans[0].Ranges[0].Selec1 = selec
			if any {
				spans[0].Ranges[0].SetFlag(plan.RANGE_ARRAY_ANY)
			} else {
				spans[0].Ranges[0].SetFlag(plan.RANGE_ARRAY_ANY_EVERY)
			}
		}
	}

	if !sp.Exact() {
		return sp, nil
	}

	exprs := expression.Expressions{key}
	if cond != nil {
		fc := make(map[string]value.Value, 4)
		fc = cond.FilterCovers(fc)
		filterCovers, err := mapFilterCovers(fc, alias)
		if err == nil {
			for c, _ := range filterCovers {
				exprs = append(exprs, c.Covered())
			}
		}
	}

	if err != nil || !expression.IsArrayCovered(pred, alias, exprs) {
		sp.SetExact(false)
	}

	return sp, nil
}
