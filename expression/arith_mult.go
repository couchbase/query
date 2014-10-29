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

type Mult struct {
	CommutativeFunctionBase
}

func NewMult(operands ...Expression) Function {
	return &Mult{
		*NewCommutativeFunctionBase("mult", operands...),
	}
}

func (this *Mult) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMult(this)
}

func (this *Mult) Type() value.Type { return value.NUMBER }

func (this *Mult) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *Mult) Apply(context Context, args ...value.Value) (value.Value, error) {
	null := false
	prod := 1.0

	for _, arg := range args {
		if !null && arg.Type() == value.NUMBER {
			prod *= arg.Actual().(float64)
		} else if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else {
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(prod), nil
}

func (this *Mult) Constructor() FunctionConstructor { return NewMult }
