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

type Max struct {
	aggregateBase
}

func NewMax(parameter Expression) Aggregate {
	return &Max{aggregateBase{parameter}}
}

func (this *Max) Default() value.Value {
	return nil
}

func (this *Max) Initial() InitialAggregate {
	return this
}

func (this *Max) Intermediate() IntermediateAggregate {
	return this
}

func (this *Max) Final() FinalAggregate {
	return this
}

func (this *Max) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	return this.cumulate(item, cumulative, context)
}

func (this *Max) CumulateIntermediate(item, cumulative value.Value, context Context) (value.Value, error) {
	return this.cumulate(item, cumulative, context)
}

func (this *Max) CumulateFinal(item, cumulative value.Value, context Context) (value.Value, error) {
	return this.cumulate(item, cumulative, context)
}

func (this *Max) cumulate(item, cumulative value.Value, context Context) (value.Value, error) {
	item, e := this.parameter.Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() <= value.NULL {
		return cumulative, nil
	} else if cumulative == nil {
		return item, nil
	} else if item.Collate(cumulative) > 0 {
		return item, nil
	} else {
		return cumulative, nil
	}
}
