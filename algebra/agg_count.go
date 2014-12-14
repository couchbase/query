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

type Count struct {
	AggregateBase
}

func NewCount(operand expression.Expression) Aggregate {
	rv := &Count{
		*NewAggregateBase("count", operand),
	}

	rv.SetExpr(rv)
	return rv
}

func (this *Count) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Count) Type() value.Type { return value.NUMBER }

func (this *Count) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	if this.Operand() != nil {
		return this.evaluate(this, item, context)
	}

	// Full keyspace count is short-circuited
	switch item := item.(type) {
	case value.AnnotatedValue:
		count := item.GetAttachment("count")
		if count != nil {
			return value.NewValue(count), nil
		}
	}

	return this.evaluate(this, item, context)
}

func (this *Count) MinArgs() int { return 0 }

func (this *Count) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		if len(operands) > 0 {
			return NewCount(operands[0])
		} else {
			return NewCount(nil)
		}
	}
}

func (this *Count) Default() value.Value { return value.ZERO_VALUE }

func (this *Count) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	if this.Operand() != nil {
		item, e := this.Operand().Evaluate(item, context)
		if e != nil {
			return nil, e
		}

		if item.Type() <= value.NULL {
			return cumulative, nil
		}
	}

	return this.cumulatePart(value.ONE_VALUE, cumulative, context)

}

func (this *Count) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return this.cumulatePart(part, cumulative, context)
}

func (this *Count) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

func (this *Count) cumulatePart(part, cumulative value.Value, context Context) (value.Value, error) {
	actual := part.Actual()
	switch actual := actual.(type) {
	case float64:
		count := cumulative.Actual()
		switch count := count.(type) {
		case float64:
			return value.NewValue(count + actual), nil
		default:
			return nil, fmt.Errorf("Invalid COUNT %v of type %T.", count, count)
		}
	default:
		return nil, fmt.Errorf("Invalid partial COUNT %v of type %T.", actual, actual)
	}
}
