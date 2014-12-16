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

type Aggregates []Aggregate

/*
The Aggregate interface represents aggregate functions such as SUM(),
AVG(), COUNT, COUNT(DISTINCT), MIN(), and MAX().

Aggregate functions are computed in parallel. Each aggregate function
must supply the methods CumulateInitial(), CumulateIntermediate(), and
CumulateFinal(). CumulateInitial() aggregates input values and
produces an intermediate aggregate. CumulateIntermediate() aggregates
intermediate aggregates and produces a further intermediate
aggregate. CumulateFinal() takes a final aggregate and performs any
post-processing. For example, Avg.CumulateFinal() divides the final
sum by the final count.

CumulateInitial() and CumulateIntermediate() can be run across
parallel input streams. CumulateFinal() must be run in a single serial
stream. CumulateIntermediate() must be chainable, to provide cascading
aggregation.

If no input data is received, the Default() value is returned.
*/
type Aggregate interface {
        /*
        Represents the aggregate function.
        */
	expression.Function
        
        /*
        Returned if there is no input data to the function.
        */
	Default() value.Value         
 
        /*
        Single operand to this function.
        */
	Operand() expression.Expression

        /*
        Aggregates input data.
        */
	CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error)  

        /*
        Aggregates intermediate results.
        */
	CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error)
 
        /*
        Performs final post-processing, if any.
        */
	ComputeFinal(cumulative value.Value, context Context) (value.Value, error)             
}

type AggregateBase struct {
	expression.UnaryFunctionBase
	text string
}

func NewAggregateBase(name string, operand expression.Expression) *AggregateBase {
	return &AggregateBase{
		*expression.NewUnaryFunctionBase(name, operand),
		"",
	}
}

func (this *AggregateBase) evaluate(agg Aggregate, item value.Value,
	context expression.Context) (result value.Value, err error) {
	defer func() {
		r := recover()
		if r != nil {
			err = fmt.Errorf("Error evaluating aggregate: %v.", r)
		}
	}()

	av := item.(value.AnnotatedValue)
	aggregates := av.GetAttachment("aggregates")
	if aggregates != nil {
		aggs := aggregates.(map[string]value.Value)
		result = aggs[agg.String()]
	}

	if result == nil {
		err = fmt.Errorf("Aggregate %s not found.", agg.String())
	}

	return
}

func (this *AggregateBase) Indexable() bool {
	return false
}

func (this *AggregateBase) EquivalentTo(other expression.Expression) bool {
	return false
}

func (this *AggregateBase) SubsetOf(other expression.Expression) bool {
	return false
}

func (this *AggregateBase) Children() expression.Expressions {
	if this.Operands()[0] == nil {
		return nil
	} else {
		return this.Operands()
	}
}

func (this *AggregateBase) MapChildren(mapper expression.Mapper) error {
	children := this.Children()

	for i, c := range children {
		expr, err := mapper.Map(c)
		if err != nil {
			return err
		}

		children[i] = expr
	}

	return nil
}

type DistinctAggregateBase struct {
	AggregateBase
}

func NewDistinctAggregateBase(name string, operand expression.Expression) *DistinctAggregateBase {
	return &DistinctAggregateBase{
		*NewAggregateBase(name, operand),
	}
}

func (this *DistinctAggregateBase) Distinct() bool { return true }
