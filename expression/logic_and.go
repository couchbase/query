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
Logical terms allow for combining other expressions using boolean logic.
Standard AND operators are supported. Type And is a struct that
implements CommutativeFunctionBase.
*/
type And struct {
	CommutativeFunctionBase
}

/*
The function NewAnd calls NewCommutativeFunctionBase to define AND
with input operand expressions as input.
*/
func NewAnd(operands ...Expression) *And {
	rv := &And{
		*NewCommutativeFunctionBase("and", operands...),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitAnd method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *And) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAnd(this)
}

/*
It returns a value type Boolean.
*/
func (this *And) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method and passes in the receiver, current item
and current context.
*/
func (this *And) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For AND, simply cumulate the implicit covers of each child operand.
*/
func (this *And) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	for _, op := range this.operands {
		covers = op.FilterCovers(covers)
	}

	return covers
}

/*
Range over input arguments, for all types other than missing and null,
if the truth value of the argument is false, then return false. If
the type is missing, return missing, and if null return null. If all
inputs are true, return true. For null and missing, it returns missing.
*/
func (this *And) Apply(context Context, args ...value.Value) (value.Value, error) {
	missing := false
	null := false

	for _, arg := range args {
		switch arg.Type() {
		case value.NULL:
			null = true
		case value.MISSING:
			missing = true
		default:
			if !arg.Truth() {
				return value.FALSE_VALUE, nil
			}
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	} else {
		return value.TRUE_VALUE, nil
	}
}

/*
Returns NewAnd as FunctionConstructor.
*/
func (this *And) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewAnd(operands...)
	}
}
