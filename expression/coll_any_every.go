//  Copyright (c) 2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"github.com/couchbase/query/value"
)

/*
Represents range predicate ANY AND EVERY, that allow testing of a bool
condition over the elements of a collection or object.
*/
type AnyEvery struct {
	collPredBase
}

func NewAnyEvery(bindings Bindings, satisfies Expression) Expression {
	rv := &AnyEvery{
		collPredBase: collPredBase{
			bindings:  bindings,
			satisfies: satisfies,
		},
	}

	rv.expr = rv
	return rv
}

func (this *AnyEvery) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAnyEvery(this)
}

func (this *AnyEvery) Type() value.Type { return value.BOOLEAN }

func (this *AnyEvery) Evaluate(item value.Value, context Context) (value.Value, error) {
	bvals, buffers, bpairs, n, missing, null, err := collEval(this.bindings, item, context)
	defer collReleaseBuffers(bvals, buffers, bpairs)
	if err != nil {
		return nil, err
	}

	if missing {
		return value.MISSING_VALUE, nil
	}

	if null {
		return value.NULL_VALUE, nil
	}

	for i := 0; i < n; i++ {
		cv := value.NewScopeValue(make(map[string]interface{}, len(this.bindings)), item)
		for j, b := range this.bindings {
			if b.NameVariable() == "" {
				cv.SetField(b.Variable(), bvals[j][i])
			} else {
				pair := bpairs[j][i]
				cv.SetField(b.NameVariable(), pair.Name)
				cv.SetField(b.Variable(), pair.Value)
			}
		}

		av := value.NewAnnotatedValue(cv)
		if ai, ok := item.(value.AnnotatedValue); ok {
			av.CopyAnnotations(ai)
		}

		sv, e := this.satisfies.Evaluate(av, context)
		if e != nil {
			return nil, e
		}

		if !sv.Truth() {
			return value.FALSE_VALUE, nil
		}
	}

	return value.NewValue(n > 0), nil
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For ANY AND EVERY, simply list this expression.
*/
func (this *AnyEvery) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *AnyEvery) Copy() Expression {
	rv := NewAnyEvery(this.bindings.Copy(), Copy(this.satisfies))
	rv.BaseCopy(this)
	return rv
}
