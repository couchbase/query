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

type Avg struct {
	aggregateBase
}

func NewAvg(argument expression.Expression) Aggregate {
	return &Avg{aggregateBase{argument: argument}}
}

func (this *Avg) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

func (this *Avg) Fold() (expression.Expression, error) {
	return this.fold(this)
}

func (this *Avg) Formalize(forbidden, allowed value.Value, keyspace string) (expression.Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *Avg) VisitChildren(visitor expression.Visitor) (expression.Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Avg) Constructor() expression.FunctionConstructor {
	return func(arguments expression.Expressions) expression.Function {
		return NewAvg(arguments[0])
	}
}

func (this *Avg) Default() value.Value {
	return value.NULL_VALUE
}

func (this *Avg) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	item, e := this.argument.Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() != value.NUMBER {
		return cumulative, nil
	}

	part := value.NewValue(map[string]interface{}{"sum": item.Actual(), "count": 1})
	return this.cumulatePart(part, cumulative, context)
}

func (this *Avg) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return this.cumulatePart(part, cumulative, context)
}

func (this *Avg) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	if cumulative == value.NULL_VALUE {
		return cumulative, nil
	}

	sum, _ := cumulative.Field("sum")
	count, _ := cumulative.Field("count")

	if sum.Type() != value.NUMBER || count.Type() != value.NUMBER {
		return nil, fmt.Errorf("Missing or invalid sum or count in AVG: %v, %v.",
			sum.Actual(), count.Actual())
	}

	if count.Actual().(float64) > 0.0 {
		return value.NewValue(sum.Actual().(float64) / count.Actual().(float64)), nil
	} else {
		return value.NULL_VALUE, nil
	}
}

func (this *Avg) cumulatePart(part, cumulative value.Value, context Context) (value.Value, error) {
	if part == value.NULL_VALUE {
		return cumulative, nil
	} else if cumulative == value.NULL_VALUE {
		return part, nil
	}

	psum, _ := part.Field("sum")
	pcount, _ := part.Field("count")
	csum, _ := cumulative.Field("sum")
	ccount, _ := cumulative.Field("count")

	if psum.Type() != value.NUMBER || pcount.Type() != value.NUMBER ||
		csum.Type() != value.NUMBER || ccount.Type() != value.NUMBER {
		return nil, fmt.Errorf("Missing or invalid partial sum or count in AVG: %v, %v, %v, v.",
			psum.Actual(), pcount.Actual(), csum.Actual(), ccount.Actual())
	}

	cumulative.SetField("sum", psum.Actual().(float64)+csum.Actual().(float64))
	cumulative.SetField("count", pcount.Actual().(float64)+ccount.Actual().(float64))
	return cumulative, nil
}
