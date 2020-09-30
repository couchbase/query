//  Copyright (c) 2014 Couchbase, Inc.
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
Represents range predicate ANY, that allow testing of a bool condition
over the elements of a collection or object.
*/
type Any struct {
	collPredBase
}

func NewAny(bindings Bindings, satisfies Expression) Expression {
	rv := &Any{
		collPredBase: collPredBase{
			bindings:  bindings,
			satisfies: satisfies,
		},
	}

	rv.expr = rv
	return rv
}

func (this *Any) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAny(this)
}

func (this *Any) Type() value.Type { return value.BOOLEAN }

func (this *Any) Evaluate(item value.Value, context Context) (value.Value, error) {
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

		sv, err := this.satisfies.Evaluate(av, context)
		if err != nil {
			return nil, err
		}

		if sv.Truth() {
			return value.TRUE_VALUE, nil
		}
	}

	return value.FALSE_VALUE, nil
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For ANY, simply list this expression.
*/
func (this *Any) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *Any) Copy() Expression {
	rv := NewAny(this.bindings.Copy(), Copy(this.satisfies))
	rv.BaseCopy(this)
	return rv
}
