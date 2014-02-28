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

var _OBJECT_CAP = 64

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
