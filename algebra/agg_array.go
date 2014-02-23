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
	"sort"

	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type ArrayAgg struct {
	aggregateBase
}

func NewArrayAgg(parameter expression.Expression) Aggregate {
	return &ArrayAgg{aggregateBase{parameter: parameter}}
}

func (this *ArrayAgg) Default() value.Value {
	return _NULL
}

func (this *ArrayAgg) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	item, e := this.parameter.Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() <= value.MISSING {
		return cumulative, nil
	}

	return this.cumulatePart(value.NewValue([]interface{}{item}), cumulative, context)
}

func (this *ArrayAgg) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return this.cumulatePart(part, cumulative, context)
}

func (this *ArrayAgg) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	if cumulative == _NULL {
		return cumulative, nil
	}

	sort.Sort(value.NewSorter(cumulative))
	return cumulative, nil
}

func (this *ArrayAgg) cumulatePart(part, cumulative value.Value, context Context) (value.Value, error) {
	if part == _NULL {
		return cumulative, nil
	} else if cumulative == _NULL {
		return part, nil
	}

	actual := part.Actual()
	switch actual := actual.(type) {
	case []interface{}:
		array := cumulative.Actual()
		switch array := array.(type) {
		case []interface{}:
			return value.NewValue(append(array, actual...)), nil
		default:
			return nil, fmt.Errorf("Invalid ARRAY_AGG %v of type %T.", array, array)
		}
	default:
		return nil, fmt.Errorf("Invalid partial ARRAY_AGG %v of type %T.", actual, actual)
	}
}
