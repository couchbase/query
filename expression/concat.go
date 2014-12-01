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
	"bytes"
	"math"

	"github.com/couchbaselabs/query/value"
)

/*
This represents the concatenation operation between two strings.
Type Concat is a struct that implements FunctionBase.
*/
type Concat struct {
	FunctionBase
}

/*
The method NewConcat uses input expressions as the input
to NewFunctionBase with function named concat. It returns
a pointer to the Concat struct.
*/
func NewConcat(operands ...Expression) Function {
	rv := &Concat{
		*NewFunctionBase("concat", operands...),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitConcat method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Concat) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitConcat(this)
}

/*
It returns STRING value.
*/
func (this *Concat) Type() value.Type { return value.STRING }

/*
Calls the Eval function and passes in the receiver, current item and
current context.
*/
func (this *Concat) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
This method takes in a set of values args and context, calculates the 
concatenation and then returns that string value. Range over the input
arguments and check their type. If it is a string and there have been 
no nulls as of yet, write the string to a temporary buffer. In the case 
of a missing return a missing. For any other values set null as true
(it needs to return a null). All the strings will be appended to the 
buffer. Check if at any time we encountered a null, and if yes return 
a null value. Create a N1QL compatible value out of the string buffer
and return it. 
*/
func (this *Concat) Apply(context Context, args ...value.Value) (value.Value, error) {
	var buf bytes.Buffer
	null := false

	for _, arg := range args {
		switch arg.Type() {
		case value.STRING:
			if !null {
				buf.WriteString(arg.Actual().(string))
			}
		case value.MISSING:
			return value.MISSING_VALUE, nil
		default:
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(buf.String()), nil
}

/*
Minimum input arguments required for the concatenation
is 2.
*/
func (this *Concat) MinArgs() int { return 2 }

/*
Maximum number of input arguments defined for the concat is 
MaxInt16  = 1<<15 - 1. This is defined using the math package.
*/
func (this *Concat) MaxArgs() int { return math.MaxInt16 }

/*
Return NewContact as FunctionConstructor.
*/
func (this *Concat) Constructor() FunctionConstructor { return NewConcat }
