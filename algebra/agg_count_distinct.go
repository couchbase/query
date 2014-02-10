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

type CountDistinct struct {
	aggregateBase
}

func NewCountDistinct(parameter Expression) Aggregate {
	return &CountDistinct{aggregateBase{parameter}}
}

func (this *CountDistinct) Default() value.Value {
	set := value.NewSet(32)
	av := value.NewAnnotatedValue(nil)
	av.SetAttachment("set", set)
	return av
}

func (this *CountDistinct) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	item, e := this.parameter.Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() <= value.NULL {
		return cumulative, nil
	}

	switch cumulative := cumulative.(type) {
	case value.AnnotatedValue:
		set := cumulative.GetAttachment("set")
		switch set := set.(type) {
		case *value.Set:
			set.Add(item)
			return cumulative, nil
		default:
			return nil, fmt.Errorf("Invalid COUNT DISTINCT set %v of type %T.", set, set)
		}
	default:
		return nil, fmt.Errorf("Invalid COUNT DISTINCT %v of type %T.", cumulative, cumulative)
	}
}

func (this *CountDistinct) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return this.cumulatePart(part, cumulative, context)
}

func (this *CountDistinct) CumulateFinal(part, cumulative value.Value, context Context) (result value.Value, e error) {
	result, e = this.cumulatePart(part, cumulative, context)
	if e != nil {
		return result, e
	}

	av := cumulative.(value.AnnotatedValue)
	set := av.GetAttachment("set").(*value.Set)
	return value.NewAnnotatedValue(set.Len()), nil
}

func (this *CountDistinct) cumulatePart(part, cumulative value.Value, context Context) (value.Value, error) {
	pset, e := getSet(part)
	if e != nil {
		return nil, e
	}

	cset, e := getSet(cumulative)
	if e != nil {
		return nil, e
	}

	// For efficiency, add smaller set to bigger
	var smaller, bigger *value.Set
	if pset.Len() <= cset.Len() {
		smaller = pset
		bigger = cset
	} else {
		smaller = cset
		bigger = pset
	}

	for _, v := range smaller.Values() {
		bigger.Add(v)
	}

	cumulative.(value.AnnotatedValue).SetAttachment("set", bigger)
	return cumulative, nil
}

func getSet(item value.Value) (*value.Set, error) {
	switch item := item.(type) {
	case value.AnnotatedValue:
		ps := item.GetAttachment("set")
		switch ps := ps.(type) {
		case *value.Set:
			return ps, nil
		default:
			return nil, fmt.Errorf("Invalid DISTINCT set %v of type %T.", ps, ps)
		}
	default:
		return nil, fmt.Errorf("Invalid DISTINCT %v of type %T.", item, item)
	}
}
