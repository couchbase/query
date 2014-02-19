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

	"github.com/couchbaselabs/query/value"
)

type Sum struct {
	aggregateBase
}

func NewSum(parameter Expression) Aggregate {
	return &Sum{aggregateBase{parameter: parameter}}
}

func (this *Sum) Default() value.Value {
	return _NULL
}

func (this *Sum) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	item, e := this.parameter.Evaluate(item, context)
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
	if part == _NULL {
		return cumulative, nil
	} else if cumulative == _NULL {
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
