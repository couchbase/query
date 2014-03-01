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
	expression.Function

	Default() value.Value
	Argument() expression.Expression

	CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error)
	CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error)
	ComputeFinal(cumulative value.Value, context Context) (value.Value, error)
}

type aggregateBase struct {
	argument expression.Expression
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
		(this.argument == nil && other.(Aggregate).Argument() == nil) ||
		this.argument.EquivalentTo(other.(Aggregate).Argument())
}

func (this *aggregateBase) Dependencies() expression.Expressions {
	if this.argument != nil {
		return expression.Expressions{this.argument}
	} else {
		return nil
	}
}

func (this *aggregateBase) Alias() string {
	return ""
}

func (this *aggregateBase) Fold() expression.Expression {
	if this.argument != nil {
		this.argument = this.argument.Fold()
	}

	return this
}

func (this *aggregateBase) Formalize() {
	if this.argument != nil {
		this.argument.Formalize()
	}
}

func (this *aggregateBase) SubsetOf(other expression.Expression) bool {
	return false
}

func (this *aggregateBase) Spans(index expression.Index) expression.Spans {
	return nil
}

func (this *aggregateBase) MinArgs() int {
	return 1
}

func (this *aggregateBase) MaxArgs() int {
	return 1
}

func (this *aggregateBase) Argument() expression.Expression {
	return this.argument
}
