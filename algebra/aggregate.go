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

	"github.com/couchbaselabs/query/value"
)

type Aggregates []Aggregate

type Aggregate interface {
	Expression

	Default() value.Value
	Parameter() Expression

	CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error)
	CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error)
	CumulateFinal(part, cumulative value.Value, context Context) (value.Value, error)
}

type aggregateBase struct {
	parameter Expression
}

func (this *aggregateBase) Evaluate(item value.Value, context Context) (result value.Value, e error) {
	defer func() {
		e = fmt.Errorf("Error evaluating aggregate: %v.", recover())
	}()

	av := item.(value.AnnotatedValue)
	aggregates := av.GetAttachment("aggregates").(map[Aggregate]value.Value)
	result = aggregates[interface{}(this).(Aggregate)]
	return result, e
}

func (this *aggregateBase) EquivalentTo(other Expression) bool {
	return reflect.TypeOf(this) == reflect.TypeOf(other) &&
		(this.parameter == nil && other.(Aggregate).Parameter() == nil) ||
		this.parameter.EquivalentTo(other.(Aggregate).Parameter())
}

func (this *aggregateBase) Dependencies() Expressions {
	return nil
}

func (this *aggregateBase) Alias() string {
	return ""
}

func (this *aggregateBase) Parameter() Expression {
	return this.parameter
}
