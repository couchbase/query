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

type Not struct {
	unaryBase
}

func NewNot(operand Expression) Expression {
	return &Not{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *Not) Fold() Expression {
	this.operand = this.operand.Fold()
	switch o := this.operand.(type) {
	case *Constant:
		v, e := this.evaluate(o.Value())
		if e == nil {
			return NewConstant(v)
		}
	case *Not:
		return o.operand
	}

	return this
}

func (this *Not) evaluate(operand value.Value) (value.Value, error) {
	if operand.Type() > value.NULL {
		return value.NewValue(!operand.Truth()), nil
	} else {
		return operand, nil
	}
}
