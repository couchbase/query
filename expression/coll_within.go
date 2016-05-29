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
Represents the collection expression WITHIN.
*/
type Within struct {
	BinaryFunctionBase
}

func NewWithin(first, second Expression) Function {
	rv := &Within{
		*NewBinaryFunctionBase("within", first, second),
	}

	rv.expr = rv
	return rv
}

func (this *Within) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitWithin(this)
}

func (this *Within) Type() value.Type { return value.BOOLEAN }

func (this *Within) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For WITHIN, simply list this expression.
*/
func (this *Within) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

/*
WITHIN evaluates to TRUE if the right-hand-side first value contains
the left-hand-side second value (or name and value) as a child or
descendant (i.e. directly or indirectly).
*/
func (this *Within) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if second.Type() != value.ARRAY && second.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	desc := second.Descendants(make([]interface{}, 0, 64))
	for _, d := range desc {
		if first.Equals(value.NewValue(d)).Truth() {
			return value.TRUE_VALUE, nil
		}
	}

	return value.FALSE_VALUE, nil
}

/*
Factory method pattern.
*/
func (this *Within) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewWithin(operands[0], operands[1])
	}
}

/*
This function implements the NOT WITHIN collection operation.
*/
func NewNotWithin(first, second Expression) Expression {
	return NewNot(NewWithin(first, second))
}
