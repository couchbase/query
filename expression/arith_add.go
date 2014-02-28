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

type Add struct {
	nAryBase
}

func NewAdd(operands ...Expression) Expression {
	return &Add{
		nAryBase{
			operands: operands,
		},
	}
}

func (this *Add) evaluate(operands value.Values) (value.Value, error) {
	null := false
	sum := 0.0
	for _, v := range operands {
		if !null && v.Type() == value.NUMBER {
			sum += v.Actual().(float64)
		} else if v.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else {
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(sum), nil
}

func (this *Add) construct(constant value.Value, others Expressions) Expression {
	if constant.Type() == value.MISSING {
		return NewConstant(constant)
	} else if constant.Type() == value.NUMBER && constant.Actual().(float64) == 0.0 {
		return NewAdd(others...)
	}

	return NewAdd(append(others, NewConstant(constant))...)
}
