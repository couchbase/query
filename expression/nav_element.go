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
	"math"

	"github.com/couchbaselabs/query/value"
)

type Element struct {
	binaryBase
}

func NewElement(first, second Expression) Path {
	return &Element{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *Element) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Element) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Element) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Element) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *Element) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Element) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Element) eval(first, second value.Value) (value.Value, error) {
	switch second.Type() {
	case value.NUMBER:
		s := second.Actual().(float64)
		if s == math.Trunc(s) {
			v, _ := first.Index(int(s))
			return v, nil
		}
	case value.MISSING:
		return value.MISSING_VALUE, nil
	}

	if first.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else {
		return value.NULL_VALUE, nil
	}
}

func (this *Element) Set(item, val value.Value, context Context) bool {
	second, er := this.second.Evaluate(item, context)
	if er != nil {
		return false
	}

	switch second.Type() {
	case value.NUMBER:
		s := second.Actual().(float64)
		if s == math.Trunc(s) {
			er := item.SetIndex(int(s), val)
			return er == nil
		}
	}

	return false
}

func (this *Element) Unset(item value.Value, context Context) bool {
	return false
}
