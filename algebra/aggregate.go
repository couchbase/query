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

	"encoding/json"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type Aggregates []Aggregate

type Aggregate interface {
	expression.Function

	Default() value.Value
	Operand() expression.Expression

	CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error)
	CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error)
	ComputeFinal(cumulative value.Value, context Context) (value.Value, error)
}

type AggregateBase struct {
	expression.UnaryFunctionBase
}

func NewAggregateBase(name string, operand expression.Expression) *AggregateBase {
	return &AggregateBase{
		*expression.NewUnaryFunctionBase(name, operand),
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
	aggregates := av.GetAttachment("aggregates").(map[Aggregate]value.Value)
	result = aggregates[agg]
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

func (this *AggregateBase) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "aggregateFunction"}
	r["function"] = this.UnaryFunctionBase.Name()
	return json.Marshal(r)
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
