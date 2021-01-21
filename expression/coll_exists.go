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
Represents the Collection expression EXISTS.
*/
type Exists struct {
	UnaryFunctionBase
}

func NewExists(operand Expression) *Exists {
	rv := &Exists{
		*NewUnaryFunctionBase("exists", operand),
	}

	rv.expr = rv
	return rv
}

func (this *Exists) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExists(this)
}

func (this *Exists) Type() value.Type { return value.BOOLEAN }

/*
Returns true if the value is an array and contains at least one
element.
*/
func (this *Exists) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.ARRAY {
		a := arg.Actual().([]interface{})
		return value.NewValue(len(a) > 0), nil
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else {
		return value.NULL_VALUE, nil
	}
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For EXISTS, simply list this expression.
*/
func (this *Exists) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

/*
Factory method pattern.
*/
func (this *Exists) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewExists(operands[0])
	}
}
