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

func NewElement(first, second Expression) Expression {
	return &Element{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *Element) evaluate(first, second value.Value) (value.Value, error) {
	switch second.Type() {
	case value.NUMBER:
		s := second.Actual().(float64)
		if s == math.Trunc(s) {
			v, _ := first.Index(int(s))
			return v, nil
		}
	case value.STRING:
		s := second.Actual().(string)
		v, _ := first.Field(s)
		return v, nil
	case value.MISSING:
		return value.MISSING_VALUE, nil
	}

	return value.NULL_VALUE, nil
}
