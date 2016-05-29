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
Comparison terms allow for comparing two expressions.
For BETWEEN and NOT BETWEEN, we have three expressions,
the input item and the low and high expressions.
*/
type Between struct {
	TernaryFunctionBase
}

func NewBetween(item, low, high Expression) Function {
	rv := &Between{
		*NewTernaryFunctionBase("between", item, low, high),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Between) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitBetween(this)
}

func (this *Between) Type() value.Type { return value.BOOLEAN }

func (this *Between) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.TernaryEval(this, item, context)
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For Between, simply list this expression.
*/
func (this *Between) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *Between) Apply(context Context, item, low, high value.Value) (value.Value, error) {
	lowCmp := item.Compare(low)
	if lowCmp.Type() == value.MISSING {
		return lowCmp, nil
	}

	highCmp := item.Compare(high)
	if highCmp.Type() == value.MISSING {
		return highCmp, nil
	}

	switch lowActual := lowCmp.Actual().(type) {
	case float64:
		switch highActual := highCmp.Actual().(type) {
		case float64:
			return value.NewValue(lowActual >= 0 && highActual <= 0), nil
		}
	}

	return value.NULL_VALUE, nil
}

/*
Factory method pattern.
*/
func (this *Between) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewBetween(operands[0], operands[1], operands[2])
	}
}

/*
This function implements the NOT BETWEEN operation.
*/
func NewNotBetween(item, low, high Expression) Expression {
	return NewNot(NewBetween(item, low, high))
}
