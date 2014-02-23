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

type Modulo struct {
	binaryBase
}

func NewModulo(first, second Expression) Expression {
	return &Modulo{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *Modulo) evaluate(first, second value.Value) (value.Value, error) {
	if second.Type() == value.NUMBER {
		s := second.Actual().(float64)
		if s == 0.0 {
			return _NULL_VALUE, nil
		}

		if first.Type() == value.NUMBER {
			m := math.Mod(first.Actual().(float64), s)
			return value.NewValue(m), nil
		}
	}

	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return _MISSING_VALUE, nil
	}

	return _NULL_VALUE, nil
}
