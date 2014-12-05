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

/*
Represents range transform First, that allow you to map and
filter the elements or attributes of a collection or object(s).
FIRST evaluates to a single element based on the operand expression.
Type First is a struct that implements collMap.
*/
type First struct {
	collMap
}

/*
This method returns a pointer to the First struct that has the
bindings,mapping and when fields populated by the input args
bindings and expression when/mapping.
*/
func NewFirst(mapping Expression, bindings Bindings, when Expression) Expression {
	rv := &First{
		collMap: collMap{
			mapping:  mapping,
			bindings: bindings,
			when:     when,
		},
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFirst method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *First) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFirst(this)
}

/*
It returns a value that is the receivers mapping type. This is
because First evaluates to a single element based on the operand
expression.
*/
func (this *First) Type() value.Type { return this.mapping.Type() }

/*
This method evaluates the First range transform and returns a single
element based on the operand expression after mapping/filtering the
elements/attributes of a collection or objects. The first step is to accumulate the
elements or attributes of a collection/object. This is done by
ranging over the bindings, evaluating the expressions and populating
a slice of descendants if present. If any of these binding values are
missing or null then, return a missing/null. The next step is to get
the number of elements/attributes by ranging over the bindings slice.
In order to evaluate the any clause, evaluate the when condition
with respect to the collection until it is true and return that
element (after evaluation.) If no such element is encountered, return a
missing value.
*/
func (this *First) Evaluate(item value.Value, context Context) (value.Value, error) {
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

		return mv, nil
	}

	return value.MISSING_VALUE, nil
}
