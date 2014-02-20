//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"github.com/couchbaselabs/query/value"
)

type Or struct {
	nAryBase
}

func NewOr(operands ...Expression) Expression {
	return &Or{
		nAryBase{
			operands: operands,
		},
	}
}

func (this *Or) evaluate(operands value.Values) (value.Value, error) {
	missing := false
	null := false
	for _, v := range operands {
		if v.Type() > value.NULL {
			if v.Truth() {
				return value.NewValue(true), nil
			}
		} else if v.Type() == value.NULL {
			null = true
		} else if v.Type() == value.MISSING {
			missing = true
		}
	}

	if null {
		return _NULL_VALUE, nil
	} else if missing {
		return _MISSING_VALUE, nil
	} else {
		return value.NewValue(false), nil
	}
}

func (this *Or) construct(constant value.Value, others Expressions) Expression {
	if constant.Truth() {
		return NewConstant(value.NewValue(true))
	}

	return NewOr(append(others, NewConstant(constant))...)
}
