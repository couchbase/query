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

	"github.com/couchbase/query/value"
)

/*
Nested expressions are used to access slices inside of arrays.
Type Slice is a struct that implements FunctionBase.
*/
type Slice struct {
	FunctionBase
}

/*
The function NewSlice calls NewFunctionBase to define the
slice with input operands of type expression as input.
*/
func NewSlice(operands ...Expression) Function {
	rv := &Slice{
		*NewFunctionBase("slice", operands...),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitSlice method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Slice) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSlice(this)
}

/*
It returns a value type ARRAY.
*/
func (this *Slice) Type() value.Type { return value.ARRAY }

/*
Calls the Eval method and passes in the
receiver, current item and current context.
*/
func (this *Slice) Evaluate(item value.Value, context Context) (rv value.Value, re error) {
	return this.Eval(this, item, context)
}

/*
This method Evaluates the slive using the input args depending on the
number of args. The form source-expr [ start : end ] is called array
slicing. It returns a new array containing a subset of the source,
containing the elements from position start to end-1. The element at
start is included, while the element at end is not. If end is omitted,
all elements from start to the end of the source array are included.
The source is the first argument. If it is missing return a missing
value. The first argument represents start. If missing return missing.
If there are more than 2 arguments, then an end is specified. Check
its type, and if missing return missing value. Since start and end
represent indices, make sure they are integer values and if not return
null value. Call Slice or Slice tail on the source (depending on whether
end is specified) to return the specified slice.
*/
func (this *Slice) Apply(context Context, args ...value.Value) (rv value.Value, re error) {
	source := args[0]
	if source.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	start := args[1]
	if start.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	ev := -1
	var end value.Value
	if len(args) >= 3 {
		end = args[2]
		if end.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		}

		ea, ok := end.Actual().(float64)
		if !ok || ea != math.Trunc(ea) {
			return value.NULL_VALUE, nil
		}

		ev = int(ea)
	}

	sa, ok := start.Actual().(float64)
	if !ok || sa != math.Trunc(sa) {
		return value.NULL_VALUE, nil
	}

	if source.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	if end != nil {
		rv, _ = source.Slice(int(sa), ev)
	} else {
		rv, _ = source.SliceTail(int(sa))
	}

	return
}

/*
Minimum input arguments required for Slices is 2.
*/
func (this *Slice) MinArgs() int { return 2 }

/*
Minimum input arguments allowed for Slices is 3.
*/
func (this *Slice) MaxArgs() int { return 3 }

/*
Return NewSlice as FunctionConstructor.
*/
func (this *Slice) Constructor() FunctionConstructor {
	return NewSlice
}
