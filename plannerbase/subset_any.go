//  Copyright (c) 2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plannerbase

import (
	"github.com/couchbase/query/expression"
)

func (this *subset) VisitAny(expr *expression.Any) (interface{}, error) {
	switch expr2 := this.expr2.(type) {
	case *expression.Any:
		return this.visitCollectionPredicate(expr, expr2)
	case *expression.AnyEvery:
		return this.visitCollectionPredicate(expr, expr2)
	default:
		return this.visitDefault(expr)
	}
}

func (this *subset) visitCollectionPredicate(expr, expr2 expression.CollectionPredicate) (
	interface{}, error) {

	if !expr.Bindings().SubsetOf(expr2.Bindings()) {
		return false, nil
	}

	renamer := expression.NewRenamer(expr.Bindings(), expr2.Bindings())
	satisfies, err := renamer.Map(expr.Satisfies().Copy())
	if err != nil {
		return nil, err
	}

	return SubsetOf(satisfies, expr2.Satisfies()), nil
}
