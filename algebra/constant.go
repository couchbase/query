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
	_ "fmt"

	"github.com/couchbaselabs/query/value"
)

type ConstantExpression struct {
	val value.Value
}

func NewConstantExpression(v value.Value) Expression {
	return &ConstantExpression{v}
}

func (this *ConstantExpression) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.val, nil
}

func (this *ConstantExpression) EquivalentTo(other Expression) bool {
	switch other := other.(type) {
	case *ConstantExpression:
		return this.val.Equals(other.val)
	default:
		return false
	}
}

func (this *ConstantExpression) Dependencies() ExpressionList {
	return nil
}
