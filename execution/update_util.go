//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

func buildFor(f *algebra.UpdateFor, val value.Value, context *opContext) (
	vals []value.Value, mismatch bool, err error) {
	return buildFors(f.Bindings(), value.Values{val}, context)
}

func buildFors(dimensions []expression.Bindings, vals value.Values, context *opContext) (
	rvals []value.Value, mismatch bool, err error) {
	if len(dimensions) == 0 {
		return
	}

	bindings := dimensions[0]
	nvals := _VALUE_POOL.Get()

	for _, val := range vals {
		arrays, buffers, pairs, n, mismatch, err := arraysFor(bindings, val, context)
		defer releaseBuffersFor(arrays, buffers, pairs)

		if mismatch || err != nil {
			_VALUE_POOL.Put(nvals)
			return nil, mismatch, err
		}

		for i := 0; i < n; i++ {
			sv := value.NewScopeValue(make(map[string]interface{}, len(bindings)), val)

			for j, b := range bindings {
				if b.NameVariable() == "" {
					sv.SetField(b.Variable(), arrays[j][i])
				} else {
					pair := pairs[j][i]
					sv.SetField(b.NameVariable(), pair.Name)
					sv.SetField(b.Variable(), pair.Value)
				}
			}

			nvals = append(nvals, sv)
		}
	}

	if len(dimensions) == 1 {
		rvals = nvals
		return
	}

	defer _VALUE_POOL.Put(nvals)
	return buildFors(dimensions[1:], nvals, context)
}

func arraysFor(bindings expression.Bindings, val value.Value, context *opContext) (
	arrays, buffers [][]interface{}, pairs [][]util.IPair, n int, mismatch bool, err error) {
	var bv value.Value

	for i, b := range bindings {
		bv, err = b.Expression().Evaluate(val, context)
		if err != nil {
			return
		}

		switch bv.Type() {
		case value.ARRAY, value.OBJECT:
			// Do nothing
		default:
			bv = value.EMPTY_ARRAY_VALUE
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
					arrays = _INTERFACES_POOL.GetSized(len(bindings))
				}
				arrays[i] = bv.Actual().([]interface{})
			default:
				arrays[i] = _EMPTY_ARRAY
			}
		} else {
			if pairs == nil {
				pairs = _IPAIRS_POOL.GetSized(len(bindings))
			}

			bp := _IPAIR_POOL.Get()

			if b.Descend() {
				bp = bv.DescendantPairs(bp)
			} else {
				switch bv.Type() {
				case value.OBJECT:
					fields := bv.Fields()

					var nameBuf [_NAME_CAP]string
					var names []string
					if len(fields) <= len(nameBuf) {
						names = nameBuf[0:0]
					} else {
						names := _NAME_POOL.GetCapped(len(fields))
						defer _NAME_POOL.Put(names)
					}

					for _, n := range bv.FieldNames(names) {
						v, _ := bv.Field(n)
						bp = append(bp, util.IPair{n, v})
					}
				case value.ARRAY:
					for n, v := range bv.Actual().([]interface{}) {
						bp = append(bp, util.IPair{n, v})
					}
				default:
					// Do nothing, bp is empty slice
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

var _EMPTY_ARRAY = []interface{}{}

func releaseValsFor(vals []value.Value) {
	_VALUE_POOL.Put(vals)
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

var _IPAIR_POOL = util.NewIPairPool(1024)
var _INTERFACE_POOL = util.NewInterfacePool(1024)
var _VALUE_POOL = value.NewValuePool(1024)

var _INTERFACES_POOL = util.NewInterfacesPool(64)
var _IPAIRS_POOL = util.NewIPairsPool(64)

const _NAME_CAP = 16

var _NAME_POOL = util.NewStringPool(256)
