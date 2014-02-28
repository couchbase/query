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

type Field struct {
	unaryBase
	field string
}

func NewField(operand Expression, field string) Expression {
	return &Field{
		unaryBase: unaryBase{
			operand: operand,
		},
		field: field,
	}
}

func (this *Field) evaluate(operand value.Value) (value.Value, error) {
	v, _ := operand.Field(this.field)
	return v, nil
}
