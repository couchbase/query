//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

func collEval(bindings Bindings, item value.Value, context Context) (
	bvals, buffers [][]interface{}, bpairs [][]util.IPair, n int, missing, null bool, err error) {
	var bv value.Value

	for i, b := range bindings {
		bv, err = b.Expression().Evaluate(item, context)
		if err != nil {
			return
		}

		switch bv.Type() {
		case value.ARRAY, value.OBJECT:
			// Do nothing
		case value.MISSING:
			missing = true
			return
		default:
			null = true
		}

		if null {
			continue
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
				if bvals == nil {
					bvals = _INTERFACES_POOL.GetSized(len(bindings))
				}
				bvals[i] = bv.Actual().([]interface{})
			default:
				null = true
			}
		} else {
			bp := _IPAIR_POOL.Get()

			if b.Descend() {
				bp = bv.DescendantPairs(bp)
			} else {
				switch bv.Type() {
				case value.OBJECT:
					fields := bv.Fields()

					var nameBuf [_NAME_CAP]string
					var names []string
					put := false
					if len(fields) <= len(nameBuf) {
						names = nameBuf[0:0]
					} else {
						names = _NAME_POOL.GetCapped(len(fields))
						put = true
					}

					for _, n := range bv.FieldNames(names) {
						v, _ := bv.Field(n)
						bp = append(bp, util.IPair{n, v})
					}

					if put {
						_NAME_POOL.Put(names)
					}
				case value.ARRAY:
					for n, v := range bv.Actual().([]interface{}) {
						bp = append(bp, util.IPair{n, v})
					}
				}
			}

			if bpairs == nil {
				bpairs = _IPAIRS_POOL.GetSized(len(bindings))
			}
			bpairs[i] = bp
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

	if n < 0 {
		null = true
	}

	return
}

// Release buffers to pools
func collReleaseBuffers(bvals, buffers [][]interface{}, bpairs [][]util.IPair) {
	for _, b := range buffers {
		if b != nil {
			_INTERFACE_POOL.Put(b)
		}
	}

	for _, b := range bpairs {
		if b != nil {
			_IPAIR_POOL.Put(b)
		}
	}

	if bvals != nil {
		_INTERFACES_POOL.Put(bvals)
	}

	if buffers != nil {
		_INTERFACES_POOL.Put(buffers)
	}

	if bpairs != nil {
		_IPAIRS_POOL.Put(bpairs)
	}
}

var _INTERFACE_POOL = util.NewInterfacePool(1024)
var _IPAIR_POOL = util.NewIPairPool(1024)

var _INTERFACES_POOL = util.NewInterfacesPool(8)
var _IPAIRS_POOL = util.NewIPairsPool(8)
