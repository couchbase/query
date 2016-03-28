//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

func collEval(bindings Bindings, item value.Value, context Context) (
	bvals [][]interface{}, bpairs [][]util.Pair, n int, missing, null bool, err error) {
	var bv value.Value

	for i, b := range bindings {
		bv, err = b.Expression().Evaluate(item, context)
		if err != nil {
			return
		}

		if b.NameVariable() == "" {
			if b.Descend() && (bv.Type() == value.ARRAY || bv.Type() == value.OBJECT) {
				buffer := _INTERFACE_POOL.Get()
				bv = value.NewValue(bv.Descendants(buffer))
			}

			switch bv.Type() {
			case value.ARRAY:
				if bvals == nil {
					bvals = make([][]interface{}, len(bindings))
				}
				bvals[i] = bv.Actual().([]interface{})
			case value.MISSING:
				missing = true
				return
			default:
				null = true
			}
		} else {
			var bp []util.Pair
			if b.Descend() && (bv.Type() == value.OBJECT || bv.Type() == value.ARRAY) {
				bp = _PAIR_POOL.Get()
				bp = bv.DescendantFields(bp)
			} else if bv.Type() == value.OBJECT {
				bp = _PAIR_POOL.Get()
				for k, v := range bv.Fields() {
					bp = append(bp, util.Pair{k, v})
				}
			}

			if bp != nil {
				if bpairs == nil {
					bpairs = make([][]util.Pair, len(bindings))
				}
				bpairs[i] = bp
				continue
			}

			switch bv.Type() {
			case value.MISSING:
				missing = true
				return
			default:
				null = true
			}
		}
	}

	if null {
		return
	}

	// Return length of shortest array
	n = -1
	for _, b := range bvals {
		if b != nil && (n < 0 || len(b) < n) {
			n = len(b)
		}
	}

	for _, b := range bpairs {
		if b != nil && (n < 0 || len(b) < n) {
			n = len(b)
		}
	}

	return
}

// Release buffers to pools
func collReleaseBuffers(bvals [][]interface{}, bpairs [][]util.Pair) {
	for _, b := range bvals {
		if b != nil {
			_INTERFACE_POOL.Put(b)
		}
	}

	for _, b := range bpairs {
		if b != nil {
			_PAIR_POOL.Put(b)
		}
	}
}

var _INTERFACE_POOL = util.NewInterfacePool(1024)
var _PAIR_POOL = util.NewPairPool(1024)
