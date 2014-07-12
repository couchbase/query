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
	aggregateBase
}

func NewCount(argument expression.Expression) Aggregate {
	return &Count{aggregateBase{argument: argument}}
}

func (this *Count) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	if this.argument != nil {
		return this.aggregateBase.Evaluate(item, context)
	}

	switch item := item.(type) {
	case value.AnnotatedValue:
	default:
		return this.aggregateBase.Evaluate(item, context)
	}

	// Full keyspace count is short-circuited
	count := item.(value.AnnotatedValue).GetAttachment("count")

	switch count := count.(type) {
	case value.Value:
		return count, nil
	case nil:
		return this.aggregateBase.Evaluate(item, context)
	default:
		return nil, fmt.Errorf("Invalid count %v of type %T.", count, count)
	}
}

func (this *Count) MinArgs() int {
	return 0
}

func (this *Count) Constructor() expression.FunctionConstructor {
	return func(arguments expression.Expressions) expression.Function {
		if len(arguments) > 0 {
			return NewCount(arguments[0])
		} else {
			return NewCount(nil)
		}
	}
}

func (this *Count) Default() value.Value {
	return _ZERO
}

func (this *Count) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	if this.argument != nil {
		item, e := this.argument.Evaluate(item, context)
		if e != nil {
			return nil, e
		}

		if item.Type() <= value.NULL {
			return cumulative, nil
		}
	}

	return this.cumulatePart(_ONE, cumulative, context)

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

var _ZERO = value.NewValue(0)
var _ONE = value.NewValue(1)
