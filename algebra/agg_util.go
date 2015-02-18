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

	"github.com/couchbase/query/value"
)

/*
Capacity for object sets.
*/
var _OBJECT_CAP = 64

/*
Add input item to the cumulative set. Get the set. If
no errors enountered add the item to the set and return
it. If set has not been initialized yet, create a new set
with capacity _OBJECT_CAP and add the item. Return the
set value.
*/
func setAdd(item, cumulative value.Value) (value.Value, error) {
	set, e := getSet(cumulative)
	if e == nil {
		set.Add(item)
		return cumulative, nil
	}

	set = value.NewSet(_OBJECT_CAP)
	set.Add(item)
	av := value.NewAnnotatedValue(nil)
	av.SetAttachment("set", set)
	return av, nil
}

/*
Aggregate distinct intermediate results and return them.
If no partial result exists(its value is a null) return the
cumulative value. If the cumulative input value is null,
return the partial value. Get the input partial and cumulative
sets and add the smaller set to the bigger. Return this set.
*/
func cumulateSets(part, cumulative value.Value) (value.Value, error) {
	if part.Type() == value.NULL {
		return cumulative, nil
	} else if cumulative.Type() == value.NULL {
		return part, nil
	}

	pset, e := getSet(part)
	if e != nil {
		return nil, e
	}

	cset, e := getSet(cumulative)
	if e != nil {
		return nil, e
	}

	// Add smaller set to bigger
	var smaller, bigger *value.Set
	if pset.Len() <= cset.Len() {
		smaller, bigger = pset, cset
	} else {
		smaller, bigger = cset, pset
	}

	for _, v := range smaller.Values() {
		bigger.Add(v)
	}

	cumulative.(value.AnnotatedValue).SetAttachment("set", bigger)
	return cumulative, nil
}

/*
Retrieve the set for annotated values. If the attachment type
is not a set, then throw an invalid distinct set error and
return.
*/
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
