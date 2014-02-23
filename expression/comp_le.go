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

type LE struct {
	binaryBase
}

func NewLE(first, second Expression) Expression {
	return &LE{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *LE) evaluate(first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return _MISSING_VALUE, nil
	} else if first.Type() == value.NULL || second.Type() == value.NULL ||
		first.Type() != second.Type() {
		return _NULL_VALUE, nil
	}

	return value.NewValue(first.Collate(second) <= 0), nil
}
