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
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type CountDistinct struct {
	aggregateBase
}

func NewCountDistinct(argument expression.Expression) Aggregate {
	return &CountDistinct{aggregateBase{argument: argument}}
}

func (this *CountDistinct) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

func (this *CountDistinct) Fold() (expression.Expression, error) {
	return this.fold(this)
}

func (this *CountDistinct) Formalize(allowed value.Value, keyspace string) (expression.Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *CountDistinct) VisitChildren(visitor expression.Visitor) (expression.Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *CountDistinct) Constructor() expression.FunctionConstructor {
	return func(arguments expression.Expressions) expression.Function {
		return NewCountDistinct(arguments[0])
	}
}

func (this *CountDistinct) Default() value.Value {
	return _ZERO
}

func (this *CountDistinct) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	item, e := this.argument.Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() <= value.NULL {
		return cumulative, nil
	}

	return setAdd(item, cumulative)
}

func (this *CountDistinct) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	if part == _ZERO {
		return cumulative, nil
	} else if cumulative == _ZERO {
		return part, nil
	}

	return cumulateSets(part, cumulative)
}

func (this *CountDistinct) ComputeFinal(cumulative value.Value, context Context) (c value.Value, e error) {
	if cumulative == _ZERO {
		return cumulative, nil
	}

	av := cumulative.(value.AnnotatedValue)
	set := av.GetAttachment("set").(*value.Set)
	return value.NewValue(set.Len()), nil
}
