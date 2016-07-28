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
Represents the collection expression IN.
*/
type In struct {
	BinaryFunctionBase
}

func NewIn(first, second Expression) Function {
	rv := &In{
		*NewBinaryFunctionBase("in", first, second),
	}

	rv.expr = rv
	return rv
}

func (this *In) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIn(this)
}

func (this *In) Type() value.Type { return value.BOOLEAN }

func (this *In) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For IN, simply list this expression.
*/
func (this *In) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *In) MayOverlapSpans() bool {
	return this.Second().Value() == nil
}

/*
IN evaluates to TRUE if the right-hand-side first value is an array
and directly contains the left-hand-side second value.
*/
func (this *In) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if second.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	sa := second.Actual().([]interface{})
	for _, s := range sa {
		if first.Equals(value.NewValue(s)).Truth() {
			return value.TRUE_VALUE, nil
		}
	}

	return value.FALSE_VALUE, nil
}

/*
Factory method pattern.
*/
func (this *In) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIn(operands[0], operands[1])
	}
}

/*
This function implements the NOT IN collection operation.
*/
func NewNotIn(first, second Expression) Expression {
	return NewNot(NewIn(first, second))
}
