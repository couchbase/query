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

/*
Type ArrayConstruct is a struct that implements FunctionBase.
Arrays can be constructed with arbitrary structure, nesting, 
and embedded expressions, as represented by the construction
expressions as per the N1QL specs.
*/
type ArrayConstruct struct {
	FunctionBase
}

/*
This method creates a new ArrayConstruct structure (a pointer
to which is returned), by creating a new function using 
NewFunctionBase using the input operands with function name 
array.
*/
func NewArrayConstruct(operands ...Expression) Function {
	return &ArrayConstruct{
		*NewFunctionBase("array", operands...),
	}
}

/*
It calls the VisitArrayConstruct method by passing in the receiver, 
and returns the interface. It is a visitor pattern.
*/
func (this *ArrayConstruct) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitArrayConstruct(this)
}

/*
Type ARRAY value. Returns value.ARRAY.
*/
func (this *ArrayConstruct) Type() value.Type { return value.ARRAY }

/*
Calls the Eval function and passes in the receiver, current item and 
current context.
*/
func (this *ArrayConstruct) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
This method takes in a set of values args and context and returns a value. 
Create a map to interface with length as the number of arguments. Range over
the input value args and add the values to the map. Create a valid value
by calling NewValue and return it.
*/
func (this *ArrayConstruct) Apply(context Context, args ...value.Value) (value.Value, error) {
	aa := make([]interface{}, len(args))

	for i, arg := range args {
		aa[i] = arg
	}

	return value.NewValue(aa), nil
}

/*
Minimum input arguments required for the defined ArrayConstruct 
function. It is 0.
*/
func (this *ArrayConstruct) MinArgs() int { return 0 }

/*
Maximum number of input arguments defined for the ArrayConstruct 
function is MaxInt16  = 1<<15 - 1. This is defined using the 
math package.
*/
func (this *ArrayConstruct) MaxArgs() int { return math.MaxInt16 }

/*
The constructor returns a NewArrayConstruct FunctionConstructor.
*/
func (this *ArrayConstruct) Constructor() FunctionConstructor { return NewArrayConstruct }
