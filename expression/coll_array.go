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
Represents range transform ARRAY, that allow you to map and filter the
elements of a collection or objects.
*/
type Array struct {
	collMapBase
	identity bool // True if this is the identity mapping, i.e. it does not transform its input.
}

func NewArray(mapping Expression, bindings Bindings, when Expression) Expression {
	rv := &Array{
		collMapBase: collMapBase{
			valueMapping: mapping,
			bindings:     bindings,
			when:         when,
		},
	}

	if v, ok := mapping.(*Identifier); ok && when == nil && len(bindings) == 1 &&
		!bindings[0].Descend() && v.Identifier() == bindings[0].Variable() &&
		bindings[0].NameVariable() == "" {
		rv.identity = true
	}

	rv.expr = rv
	return rv
}

func (this *Array) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitArray(this)
}

func (this *Array) Type() value.Type { return value.ARRAY }

func (this *Array) IsIdentityMapping() bool {
	return this.identity
}

func (this *Array) Evaluate(item value.Value, context Context) (value.Value, error) {
	if this.identity {
		return this.identityEval(item, context)
	}

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

	rv := make([]interface{}, 0, n)
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

		if this.when != nil {
			wv, e := this.when.Evaluate(av, context)
			if e != nil {
				return nil, e
			}

			if !wv.Truth() {
				continue
			}
		}

		mv, e := this.valueMapping.Evaluate(av, context)
		if e != nil {
			return nil, e
		}

		if mv.Type() != value.MISSING {
			rv = append(rv, mv)
		}
	}

	return value.NewValue(rv), nil
}

func (this *Array) EvaluateForIndex(item value.Value, context Context) (value.Value, value.Values, error) {
	if this.identity {
		return this.identityEvalForIndex(item, context)
	}

	bvals, buffers, bpairs, n, missing, null, err := collEval(this.bindings, item, context)
	defer collReleaseBuffers(bvals, buffers, bpairs)
	if err != nil {
		return nil, nil, err
	}

	if missing {
		return value.MISSING_VALUE, nil, nil
	}

	if null {
		return value.NULL_VALUE, nil, nil
	}

	rv := make([]interface{}, 0, n)
	var rvs value.Values
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

		if this.when != nil {
			wv, e := this.when.Evaluate(av, context)
			if e != nil {
				return nil, nil, e
			}

			if !wv.Truth() {
				continue
			}
		}

		mv, mvs, e := this.valueMapping.EvaluateForIndex(av, context)
		if e != nil {
			return nil, nil, e
		}

		if mv.Type() != value.MISSING {
			rv = append(rv, mv)
		}

		if mvs != nil {
			if rvs == nil {
				rvs = make(value.Values, 0, n*(1+len(mvs)))
			}
			rvs = append(rvs, mvs...)
		}
	}

	return value.NewValue(rv), rvs, nil
}

func (this *Array) DependsOn(other Expression) bool {
	// If identical except for mapping, check for mapping dependency
	if o, ok := other.(*Array); ok &&
		this.bindings.EquivalentTo(o.bindings) &&
		Equivalent(this.when, o.when) &&
		Equivalent(this.nameMapping, o.nameMapping) &&
		this.valueMapping.DependsOn(o.valueMapping) {
		return true
	}

	return this.collMapBase.DependsOn(other)
}

func (this *Array) Copy() Expression {
	rv := NewArray(this.valueMapping.Copy(), this.bindings.Copy(), Copy(this.when))
	rv.BaseCopy(this)
	return rv
}

func (this *Array) identityEval(item value.Value, context Context) (value.Value, error) {
	val, err := this.bindings[0].Expression().Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	switch val.Type() {
	case value.ARRAY, value.MISSING:
		return val, nil
	default:
		return value.NULL_VALUE, nil
	}
}

func (this *Array) identityEvalForIndex(item value.Value, context Context) (value.Value, value.Values, error) {
	val, vals, err := this.bindings[0].Expression().EvaluateForIndex(item, context)
	if err != nil {
		return nil, nil, err
	}

	switch val.Type() {
	case value.ARRAY, value.MISSING:
		return val, vals, nil
	default:
		return value.NULL_VALUE, vals, nil
	}
}
