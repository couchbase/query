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
	"github.com/couchbaselabs/query/value"
)

type CountDistinct struct {
	aggregateBase
}

func NewCountDistinct(parameter Expression) Aggregate {
	return &CountDistinct{aggregateBase{parameter}}
}

func (this *CountDistinct) Default() value.Value {
	return _ZERO
}

func (this *CountDistinct) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	item, e := this.parameter.Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() <= value.NULL {
		return cumulative, nil
	}

	return setAdd(cumulative, item)
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
