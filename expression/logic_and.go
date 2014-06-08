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

type And struct {
	caNAryBase
}

func NewAnd(operands ...Expression) Expression {
	return &And{
		caNAryBase{
			nAryBase{
				operands: operands,
			},
		},
	}
}

func (this *And) evaluate(operands value.Values) (value.Value, error) {
	missing := false
	null := false
	for _, v := range operands {
		if v.Type() > value.NULL {
			if !v.Truth() {
				return value.NewValue(false), nil
			}
		} else if v.Type() == value.NULL {
			null = true
		} else if v.Type() == value.MISSING {
			missing = true
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	} else {
		return value.NewValue(true), nil
	}
}

func (this *And) construct(constant value.Value, others Expressions) Expression {
	if constant.Type() == value.MISSING {
		return NewConstant(constant)
	} else if constant.Type() > value.NULL && !constant.Truth() {
		return NewConstant(value.NewValue(false))
	}

	return NewAnd(append(others, NewConstant(constant))...)
}
