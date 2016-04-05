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
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

func arraysFor(f *algebra.UpdateFor, val value.Value, context *Context) (
	arrays [][]interface{}, buffers [][]interface{}, pairs [][]util.IPair, n int, mismatch bool, err error) {
	var bv value.Value

	for i, b := range f.Bindings() {
		bv, err = b.Expression().Evaluate(val, context)
		if err != nil {
			return
		}

		switch bv.Type() {
		case value.ARRAY, value.OBJECT:
			// Do nothing
		default:
			mismatch = true
			return
		}

		if b.NameVariable() == "" {
			if b.Descend() {
				if buffers == nil {
					buffers = _INTERFACES_POOL.Get()
				}

				buffer := _INTERFACE_POOL.Get()
				buffers = append(buffers, buffer)
				bv = value.NewValue(bv.Descendants(buffer))
			}

			switch bv.Type() {
			case value.ARRAY:
				if arrays == nil {
					arrays = _INTERFACES_POOL.GetSized(len(f.Bindings()))
				}
				arrays[i] = bv.Actual().([]interface{})
			default:
				mismatch = true
				return
			}
		} else {
			if pairs == nil {
				pairs = _IPAIRS_POOL.GetSized(len(f.Bindings()))
			}

			bp := _IPAIR_POOL.Get()

			if b.Descend() {
				bp = bv.DescendantPairs(bp)
			} else {
				switch bv.Type() {
				case value.OBJECT:
					names := _NAME_POOL.GetSized(len(bv.Fields()))
					defer _NAME_POOL.Put(names)
					for _, n := range bv.FieldNames(names) {
						v, _ := bv.Field(n)
						bp = append(bp, util.IPair{n, v})
					}
				case value.ARRAY:
					for n, v := range bv.Actual().([]interface{}) {
						bp = append(bp, util.IPair{n, v})
					}
				}
			}

			pairs[i] = bp
		}
	}

	// Return length of shortest array
	n = -1
	for _, a := range arrays {
		if a != nil && (n < 0 || len(a) < n) {
			n = len(a)
		}
	}

	for _, p := range pairs {
		if p != nil && (n < 0 || len(p) < n) {
			n = len(p)
		}
	}

	return
}

func buildFor(f *algebra.UpdateFor, val value.Value, arrays [][]interface{},
	pairs [][]util.IPair, n int, context *Context) ([]value.Value, error) {
	rv := _VALUE_POOL.GetSized(n)

	for i := 0; i < n; i++ {
		sv := value.NewScopeValue(make(map[string]interface{}, len(f.Bindings())), val)
		rv[i] = sv

		for j, b := range f.Bindings() {
			if b.NameVariable() == "" {
				sv.SetField(b.Variable(), arrays[j][i])
			} else {
				pair := pairs[j][i]
				sv.SetField(b.NameVariable(), pair.Name)
				sv.SetField(b.Variable(), pair.Value)
			}
		}
	}

	return rv, nil
}

func releaseBuffersFor(arrays, buffers [][]interface{}, pairs [][]util.IPair) {
	for _, b := range buffers {
		if b != nil {
			_INTERFACE_POOL.Put(b)
		}
	}

	for _, p := range pairs {
		if p != nil {
			_IPAIR_POOL.Put(p)
		}
	}

	if arrays != nil {
		_INTERFACES_POOL.Put(arrays)
	}

	if buffers != nil {
		_INTERFACES_POOL.Put(buffers)
	}

	if pairs != nil {
		_IPAIRS_POOL.Put(pairs)
	}
}

func releaseValsFor(vals []value.Value) {
	_VALUE_POOL.Put(vals)
}

var _IPAIR_POOL = util.NewIPairPool(1024)
var _INTERFACE_POOL = util.NewInterfacePool(1024)
var _VALUE_POOL = value.NewValuePool(1024)

var _INTERFACES_POOL = util.NewInterfacesPool(8)
var _IPAIRS_POOL = util.NewIPairsPool(8)

var _NAME_POOL = util.NewStringPool(64)
