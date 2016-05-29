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
	"github.com/couchbase/query/value"
)

/*
Represents Add for arithmetic expressions. Type Add is a struct
that implements CommutativeFunctionBase.
*/
type Add struct {
	CommutativeFunctionBase
}

func NewAdd(operands ...Expression) Function {
	rv := &Add{
		*NewCommutativeFunctionBase("add", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Add) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAdd(this)
}

func (this *Add) Type() value.Type { return value.NUMBER }

func (this *Add) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
Range over input arguments, if the type is a number add it to the sum.
If the value is missing, return a missing value. For all other types
return a null value. Return the final sum.
*/
func (this *Add) Apply(context Context, args ...value.Value) (value.Value, error) {
	null := false
	sum := 0.0

	for _, arg := range args {
		if !null && arg.Type() == value.NUMBER {
			sum += arg.Actual().(float64)
		} else if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else {
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(sum), nil
}

/*
Factory method pattern.
*/
func (this *Add) Constructor() FunctionConstructor {
	return NewAdd
}
