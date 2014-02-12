//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execute

import (
	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/value"
)

func buildFor(pf *algebra.PathFor, val value.Value, context *Context) ([]value.Value, error) {
	var e error
	arrays := make([]value.Value, len(pf.Bindings()))
	for i, b := range pf.Bindings() {
		arrays[i], e = b.Expression().Evaluate(val, context)
		if e != nil {
			return nil, e
		}
	}

	n := 0
	for _, a := range arrays {
		act := a.Actual()
		switch act := act.(type) {
		case []interface{}:
			if len(act) > n {
				n = len(act)
			}
		}
	}

	rv := make([]value.Value, n)
	for i, _ := range rv {
		rv[i] = value.NewCorrelatedValue(val)
		for j, b := range pf.Bindings() {
			v := arrays[j].Index(i)
			rv[i].SetField(b.Variable(), v)
		}
	}

	return rv, nil
}
