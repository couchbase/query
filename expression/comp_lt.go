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
This represents the LESS THAN comparison
operation.
*/
type LT struct {
	BinaryFunctionBase
}

func NewLT(first, second Expression) Function {
	rv := &LT{
		*NewBinaryFunctionBase("lt", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *LT) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitLT(this)
}

func (this *LT) Type() value.Type { return value.BOOLEAN }

func (this *LT) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	cmp := first.Compare(second)
	switch actual := cmp.Actual().(type) {
	case float64:
		return value.NewValue(actual < 0), nil
	}

	return cmp, nil
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For LT, simply list this expression.
*/
func (this *LT) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

/*
Factory method pattern.
*/
func (this *LT) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLT(operands[0], operands[1])
	}
}
