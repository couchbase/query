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
	"github.com/couchbaselabs/query/value"
)

type Array struct {
	collMap
}

func NewArray(mapping Expression, bindings Bindings, when Expression) Expression {
	return &Array{
		collMap: collMap{
			mapping:  mapping,
			bindings: bindings,
			when:     when,
		},
	}
}

func (this *Array) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitArray(this)
}

func (this *Array) Evaluate(item value.Value, context Context) (value.Value, error) {
	missing := false
	null := false

	barr := make([][]interface{}, len(this.bindings))
	for i, b := range this.bindings {
		bv, err := b.Expression().Evaluate(item, context)
		if err != nil {
			return nil, err
		}

		if b.Descend() {
			buffer := make([]interface{}, 0, 256)
			bv = value.NewValue(bv.Descendants(buffer))
		}

		switch bv.Type() {
		case value.ARRAY:
			barr[i] = bv.Actual().([]interface{})
		case value.MISSING:
			missing = true
		default:
			null = true
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	}

	if null {
		return value.NULL_VALUE, nil
	}

	n := -1
	for _, b := range barr {
		if n < 0 || len(b) < n {
			n = len(b)
		}
	}

	rv := make([]interface{}, 0, n)
	for i := 0; i < n; i++ {
		cv := value.NewScopeValue(make(map[string]interface{}, len(this.bindings)), item)
		for j, b := range this.bindings {
			cv.SetField(b.Variable(), barr[j][i])
		}

		if this.when != nil {
			wv, e := this.when.Evaluate(cv, context)
			if e != nil {
				return nil, e
			}

			if !wv.Truth() {
				continue
			}
		}

		mv, e := this.mapping.Evaluate(cv, context)
		if e != nil {
			return nil, e
		}

		if mv.Type() != value.MISSING {
			rv = append(rv, mv)
		}
	}

	return value.NewValue(rv), nil
}

func (this *Array) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Array) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}
