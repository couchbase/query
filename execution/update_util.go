//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execution

import (
	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/value"
)

func arraysFor(f *algebra.UpdateFor, val value.Value, context *Context) ([]value.Value, error) {
	var e error
	arrays := make([]value.Value, len(f.Bindings()))
	for i, b := range f.Bindings() {
		arrays[i], e = b.Expression().Evaluate(val, context)
		if e != nil {
			return nil, e
		}
	}

	return arrays, nil
}

func buildFor(f *algebra.UpdateFor, val value.Value, arrays []value.Value, context *Context) ([]value.Value, error) {
	n := -1
	for _, a := range arrays {
		act := a.Actual()
		switch act := act.(type) {
		case []interface{}:
			if n < 0 || len(act) < n {
				n = len(act)
			}
		}
	}

	rv := make([]value.Value, n)
	for i, _ := range rv {
		rv[i] = value.NewScopeValue(make(map[string]interface{}, len(f.Bindings())), val)
		for j, b := range f.Bindings() {
			v, _ := arrays[j].Index(i)
			if v.Type() != value.MISSING {
				rv[i].SetField(b.Variable(), v)
			}
		}
	}

	return rv, nil
}
