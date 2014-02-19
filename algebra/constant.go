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

type Constant struct {
	expressionBase
	value value.Value
}

func NewConstant(value value.Value) Expression {
	return &Constant{value: value}
}

func (this *Constant) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.value, nil
}

func (this *Constant) EquivalentTo(other Expression) bool {
	switch other := other.(type) {
	case *Constant:
		return this.value.Equals(other.value)
	default:
		return false
	}
}

func (this *Constant) Dependencies() Expressions {
	return nil
}

func (this *Constant) Alias() string {
	return ""
}

func (this *Constant) Fold() Expression {
	return this
}

func (this *Constant) CNF() Expression {
	return this
}

func (this *Constant) Value() value.Value {
	return this.value
}
