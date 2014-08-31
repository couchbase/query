//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"fmt"

	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type Sum struct {
	AggregateBase
}

func NewSum(operand expression.Expression) Aggregate {
	return &Sum{
		*NewAggregateBase("sum", operand),
	}
}

func (this *Sum) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Sum) Type() value.Type { return value.NUMBER }

func (this *Sum) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

func (this *Sum) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewSum(operands[0])
	}
}

func (this *Sum) Default() value.Value { return value.NULL_VALUE }

func (this *Sum) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	item, e := this.Operand().Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() != value.NUMBER {
		return cumulative, nil
	}

	return this.cumulatePart(item, cumulative, context)
}

func (this *Sum) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return this.cumulatePart(part, cumulative, context)
}

func (this *Sum) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

func (this *Sum) cumulatePart(part, cumulative value.Value, context Context) (value.Value, error) {
	if part == value.NULL_VALUE {
		return cumulative, nil
	} else if cumulative == value.NULL_VALUE {
		return part, nil
	}

	actual := part.Actual()
	switch actual := actual.(type) {
	case float64:
		sum := cumulative.Actual()
		switch sum := sum.(type) {
		case float64:
			return value.NewValue(sum + actual), nil
		default:
			return nil, fmt.Errorf("Invalid SUM %v of type %T.", sum, sum)
		}
	default:
		return nil, fmt.Errorf("Invalid partial SUM %v of type %T.", actual, actual)
	}
}
