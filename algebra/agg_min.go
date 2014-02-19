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

type Min struct {
	aggregateBase
}

func NewMin(parameter Expression) Aggregate {
	return &Min{aggregateBase{parameter: parameter}}
}

func (this *Min) Default() value.Value {
	return _NULL
}

func (this *Min) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	item, e := this.parameter.Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() <= value.NULL {
		return cumulative, nil
	}

	return this.cumulatePart(item, cumulative, context)
}

func (this *Min) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return this.cumulatePart(part, cumulative, context)
}

func (this *Min) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

func (this *Min) cumulatePart(part, cumulative value.Value, context Context) (value.Value, error) {
	if part == _NULL {
		return cumulative, nil
	} else if cumulative == _NULL {
		return part, nil
	} else if part.Collate(cumulative) < 0 {
		return part, nil
	} else {
		return cumulative, nil
	}
}
