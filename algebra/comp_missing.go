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

type IsMissing struct {
	unaryBase
}

func NewIsMissing(operand Expression) Expression {
	return &IsMissing{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *IsMissing) Fold() Expression {
	this.operand = this.operand.Fold()
	switch o := this.operand.(type) {
	case *Constant:
		v, e := this.evaluate(o.Value())
		if e == nil {
			return NewConstant(v)
		}
	}

	return this
}

func (this *IsMissing) evaluate(operand value.Value) (value.Value, error) {
	switch operand.Type() {
	case value.MISSING:
		return value.NewValue(true), nil
	default:
		return value.NewValue(false), nil
	}
}

func NewIsNotMissing(operand Expression) Expression {
	return NewNot(NewIsMissing(operand))
}
