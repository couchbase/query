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
Standard OR operators are supported. Type Or is a struct that
implements CommutativeFunctionBase.
*/
type Or struct {
	CommutativeFunctionBase
}

/*
The function NewOr calls NewCommutativeFunctionBase to define OR
with input operand expressions as input.
*/
func NewOr(operands ...Expression) *Or {
	rv := &Or{
		*NewCommutativeFunctionBase("or", operands...),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitOr method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Or) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitOr(this)
}

/*
It returns a value type Boolean.
*/
func (this *Or) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method and passes in the receiver, current item
and current context.
*/
func (this *Or) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For OR, intersect the implicit covers of each child operand.
*/
func (this *Or) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	c := _COVERS_POOL.Get()
	defer _COVERS_POOL.Put(c)

	c = this.operands[0].FilterCovers(c)
	if len(c) == 0 {
		return covers
	}

	for i := 1; i < len(this.operands); i++ {
		ci := _COVERS_POOL.Get()
		defer _COVERS_POOL.Put(ci)

		ci = this.operands[i].FilterCovers(ci)
		if len(ci) == 0 {
			return covers
		}

		for s, v := range c {
			vi, ok := ci[s]
			if !ok || !v.Equals(vi).Truth() {
				delete(c, s)
			}
		}
	}

	for s, v := range c {
		covers[s] = v
	}

	return covers
}

var _COVERS_POOL = value.NewStringValuePool(16)

/*
Range over input arguments, for all types other than missing and null,
if the truth value of the argument is true, then return true. If
the type is missing, return missing, and if null return null. If all
inputs are false then return false. For null and missing, it returns
null.
*/
func (this *Or) Apply(context Context, args ...value.Value) (value.Value, error) {
	missing := false
	null := false

	for _, arg := range args {
		switch arg.Type() {
		case value.NULL:
			null = true
		case value.MISSING:
			missing = true
		default:
			if arg.Truth() {
				return value.TRUE_VALUE, nil
			}
		}
	}

	if null {
		return value.NULL_VALUE, nil
	} else if missing {
		return value.MISSING_VALUE, nil
	} else {
		return value.FALSE_VALUE, nil
	}
}

/*
Returns NewOr as FunctionConstructor.
*/
func (this *Or) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewOr(operands...)
	}
}
