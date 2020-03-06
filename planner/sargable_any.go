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
	base "github.com/couchbase/query/plannerbase"
)

func (this *sargable) VisitAny(pred *expression.Any) (interface{}, error) {
	if this.defaultSargable(pred) {
		return true, nil
	}

	all, ok := this.key.(*expression.All)
	if !ok {
		return false, nil
	}

	array, ok := all.Array().(*expression.Array)
	if !ok {
		bindings := pred.Bindings()
		return len(bindings) == 1 &&
				!bindings[0].Descend() &&
				bindings[0].Expression().EquivalentTo(all.Array()),
			nil
	}

	if !pred.Bindings().SubsetOf(array.Bindings()) {
		return false, nil
	}

	renamer := expression.NewRenamer(pred.Bindings(), array.Bindings())
	satisfies, err := renamer.Map(pred.Satisfies().Copy())
	if err != nil {
		return nil, err
	}

	if array.When() != nil && !base.SubsetOf(satisfies, array.When()) {
		return false, nil
	}

	mappings := expression.Expressions{array.ValueMapping()}
	min, _, _ := SargableFor(satisfies, mappings, this.missing, this.gsi)
	return min > 0, nil
}
