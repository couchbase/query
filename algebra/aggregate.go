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
	"reflect"

	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type Aggregates []Aggregate

type Aggregate interface {
	expression.Expression

	Default() value.Value
	Parameter() expression.Expression

	CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error)
	CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error)
	ComputeFinal(cumulative value.Value, context Context) (value.Value, error)
}

type aggregateBase struct {
	parameter expression.Expression
}

func (this *aggregateBase) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	defer func() {
		e = fmt.Errorf("Error evaluating aggregate: %v.", recover())
	}()

	av := item.(value.AnnotatedValue)
	aggregates := av.GetAttachment("aggregates").(map[Aggregate]value.Value)
	result = aggregates[interface{}(this).(Aggregate)]
	return result, e
}

func (this *aggregateBase) EquivalentTo(other expression.Expression) bool {
	return reflect.TypeOf(this) == reflect.TypeOf(other) &&
		(this.parameter == nil && other.(Aggregate).Parameter() == nil) ||
		this.parameter.EquivalentTo(other.(Aggregate).Parameter())
}

func (this *aggregateBase) Dependencies() expression.Expressions {
	if this.parameter != nil {
		return expression.Expressions{this.parameter}
	} else {
		return nil
	}
}

func (this *aggregateBase) Alias() string {
	return ""
}

func (this *aggregateBase) Fold() expression.Expression {
	if this.parameter != nil {
		this.parameter = this.parameter.Fold()
	}

	return this
}

func (this *aggregateBase) Formalize() {
	if this.parameter != nil {
		this.parameter.Formalize()
	}
}

func (this *aggregateBase) SubsetOf(other expression.Expression) bool {
	return false
}

func (this *aggregateBase) Spans(index expression.Index) expression.Spans {
	return nil
}

func (this *aggregateBase) Parameter() expression.Expression {
	return this.parameter
}
