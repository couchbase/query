//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package value

import (
	"encoding/base64"
	"fmt"

	"github.com/couchbase/query/util"
)

type valueCnt struct {
	value Value
	cnt   int64
}

func NewValueCnt(v Value, cnt int64) *valueCnt {
	return &valueCnt{
		value: v,
		cnt:   cnt,
	}
}

func (this *valueCnt) add(delta int64) {
	this.cnt += delta
}

func (this *valueCnt) getCnt() int64 {
	return this.cnt
}

func (this *valueCnt) getValue() Value {
	return this.value
}

func (this *valueCnt) copy() *valueCnt {
	return NewValueCnt(this.value, this.cnt)
}

func addValueCnt(valueCnt *valueCnt, item Value, delta int64) *valueCnt {
	if delta < 0 && (valueCnt == nil || valueCnt.getCnt()+delta <= 0) {
		return nil
	}
	if valueCnt == nil {
		return NewValueCnt(item, delta)
	}
	valueCnt.add(delta)
	return valueCnt
}

func getCnt(valueCnt *valueCnt) int64 {
	if valueCnt != nil {
		return valueCnt.getCnt()
	}
	return 0
}

/*
MultiSet implements a hash map of Values with number of occurrences.
*/
type MultiSet struct {
	nills     *valueCnt
	missings  *valueCnt
	nulls     *valueCnt
	booleans  map[bool]*valueCnt
	floats    map[float64]*valueCnt
	ints      map[int64]*valueCnt
	strings   map[string]*valueCnt
	arrays    map[string]*valueCnt
	objects   map[string]*valueCnt
	binaries  map[string]*valueCnt
	collect   bool
	numeric   bool
	objectCap int
}

func NewMultiSet(objectCap int, collect, numeric bool) *MultiSet {
	mapCap := util.MaxInt(objectCap, _MAP_CAP)

	rv := &MultiSet{
		floats:    make(map[float64]*valueCnt, mapCap),
		ints:      make(map[int64]*valueCnt, mapCap),
		numeric:   numeric,
		collect:   collect,
		objectCap: objectCap,
	}
	if !numeric {
		rv.booleans = make(map[bool]*valueCnt, 2)
		rv.strings = make(map[string]*valueCnt, mapCap)
		rv.arrays = make(map[string]*valueCnt, _MAP_CAP)
		rv.objects = make(map[string]*valueCnt, objectCap)
		rv.binaries = make(map[string]*valueCnt, _MAP_CAP)
	}
	return rv
}

func (this *MultiSet) Add(item Value) {
	this.Put(item, item, 1)
}

func (this *MultiSet) AddAll(items []interface{}) {
	for _, item := range items {
		this.Add(NewValue(item))
	}
}

func (this *MultiSet) Put(key, item Value, cnt int64) {
	if key == nil && this.numeric == false {
		this.nills = addValueCnt(this.nills, nil, cnt)
	}

	if this.numeric && key.Type() != NUMBER {
		panic(fmt.Sprintf("Numeric set will not support value type %T.", key))
		return
	}

	mapItem := item
	if !this.collect {
		mapItem = nil
	}

	switch key.Type() {
	case MISSING:
		this.missings = addValueCnt(this.missings, item, cnt)
	case NULL:
		this.nulls = addValueCnt(this.nulls, item, cnt)
	case BOOLEAN:
		k := key.Actual().(bool)
		vc := addValueCnt(this.booleans[k], mapItem, cnt)
		if vc == nil {
			delete(this.booleans, k)
		} else {
			this.booleans[k] = vc
		}
	case NUMBER:
		num := key.unwrap()
		switch num := num.(type) {
		case floatValue:
			f := float64(num)
			if IsInt(f) {
				vc := addValueCnt(this.ints[int64(f)], mapItem, cnt)
				if vc == nil {
					delete(this.ints, int64(f))
				} else {
					this.ints[int64(f)] = vc
				}
			} else {
				vc := addValueCnt(this.floats[f], mapItem, cnt)
				if vc == nil {
					delete(this.floats, f)
				} else {
					this.floats[f] = vc
				}
			}
		case intValue:
			vc := addValueCnt(this.ints[int64(num)], mapItem, cnt)
			if vc == nil {
				delete(this.ints, int64(num))
			} else {
				this.ints[int64(num)] = vc
			}
		default:
			panic(fmt.Sprintf("Unsupported value type %T.", key))
		}
	case STRING:
		k := key.Actual().(string)
		vc := addValueCnt(this.strings[k], mapItem, cnt)
		if vc == nil {
			delete(this.strings, k)
		} else {
			this.strings[k] = vc
		}
	case ARRAY:
		vc := addValueCnt(this.arrays[key.String()], mapItem, cnt)
		if vc == nil {
			delete(this.arrays, key.String())
		} else {
			this.arrays[key.String()] = vc
		}
	case OBJECT:
		vc := addValueCnt(this.objects[key.String()], mapItem, cnt)
		if vc == nil {
			delete(this.objects, key.String())
		} else {
			this.objects[key.String()] = vc
		}
	case BINARY:
		str := base64.StdEncoding.EncodeToString(key.Actual().([]byte))
		vc := addValueCnt(this.binaries[str], mapItem, cnt)
		if vc == nil {
			delete(this.binaries, str)
		} else {
			this.binaries[str] = vc
		}
	default:
		panic(fmt.Sprintf("Unsupported value type %T.", key))
	}
}

// Removes a single occurrence of the specified element from this multiset, if present.
func (this *MultiSet) Remove(key Value) {
	this.Put(key, key, -1)
}

func (this *MultiSet) Has(key Value) bool {
	if key == nil {
		return this.nills == nil
	}

	if this.numeric && key.Type() != NUMBER {
		panic(fmt.Sprintf("Numeric set will not support value type %T.", key))
		return false
	}

	ok := false
	switch key.Type() {
	case MISSING:
		return this.missings != nil
	case NULL:
		return this.nulls != nil
	case BOOLEAN:
		_, ok = this.booleans[key.Actual().(bool)]
	case NUMBER:
		num := key.unwrap()
		switch num := num.(type) {
		case floatValue:
			f := float64(num)
			if IsInt(f) {
				_, ok = this.ints[int64(f)]
			} else {
				_, ok = this.floats[f]
			}
		case intValue:
			_, ok = this.ints[int64(num)]
		default:
			panic(fmt.Sprintf("Unsupported value type %T.", key))
		}
	case STRING:
		_, ok = this.strings[key.Actual().(string)]
	case ARRAY:
		_, ok = this.arrays[key.String()]
	case OBJECT:
		_, ok = this.objects[key.String()]
	case BINARY:
		str := base64.StdEncoding.EncodeToString(key.Actual().([]byte))
		_, ok = this.binaries[str]
	default:
		panic(fmt.Sprintf("Unsupported value type %T.", key))
	}

	return ok
}

func (this *MultiSet) Count(key Value) int64 {
	if key == nil {
		return getCnt(this.nills)
	}

	if this.numeric && key.Type() != NUMBER {
		panic(fmt.Sprintf("Numeric set will not support value type %T.", key))
		return -1
	}

	var vc *valueCnt
	var count int64 = 0
	ok := false
	switch key.Type() {
	case MISSING:
		return getCnt(this.missings)
	case NULL:
		return getCnt(this.nulls)
	case BOOLEAN:
		vc, ok = this.booleans[key.Actual().(bool)]
	case NUMBER:
		num := key.unwrap()
		switch num := num.(type) {
		case floatValue:
			f := float64(num)
			if IsInt(f) {
				vc, ok = this.ints[int64(f)]
			} else {
				vc, ok = this.floats[f]
			}
		case intValue:
			vc, ok = this.ints[int64(num)]
		default:
			panic(fmt.Sprintf("Unsupported value type %T.", key))
		}
	case STRING:
		vc, ok = this.strings[key.Actual().(string)]
	case ARRAY:
		vc, ok = this.arrays[key.String()]
	case OBJECT:
		vc, ok = this.objects[key.String()]
	case BINARY:
		str := base64.StdEncoding.EncodeToString(key.Actual().([]byte))
		vc, ok = this.binaries[str]
	default:
		panic(fmt.Sprintf("Unsupported value type %T.", key))
	}

	if ok {
		count = vc.getCnt()
	}
	return count
}

func (this *MultiSet) Len() int {
	rv := len(this.booleans) + len(this.floats) + len(this.ints) + len(this.strings) +
		len(this.arrays) + len(this.objects) + len(this.binaries)

	if this.nills != nil {
		rv++
	}

	if this.missings != nil {
		rv++
	}

	if this.nulls != nil {
		rv++
	}

	return rv
}

func (this *MultiSet) Values() []Value {
	if !this.collect {
		return nil
	}

	rv := make([]Value, 0, this.Len())

	if !this.numeric {
		if this.nills != nil {
			rv = append(rv, nil)
		}

		if this.missings != nil {
			rv = append(rv, this.missings.getValue())
		}

		if this.nulls != nil {
			rv = append(rv, this.nulls.getValue())
		}

		for _, av := range this.booleans {
			rv = append(rv, av.getValue())
		}
	}

	for _, av := range this.floats {
		rv = append(rv, av.getValue())
	}

	for _, av := range this.ints {
		rv = append(rv, av.getValue())
	}

	if !this.numeric {
		for _, av := range this.strings {
			rv = append(rv, av.getValue())
		}

		for _, av := range this.arrays {
			rv = append(rv, av.getValue())
		}

		for _, av := range this.objects {
			rv = append(rv, av.getValue())
		}

		for _, av := range this.binaries {
			rv = append(rv, av.getValue())
		}
	}

	return rv
}

func (this *MultiSet) Actuals() []interface{} {
	if !this.collect {
		return nil
	}

	rv := make([]interface{}, 0, this.Len())

	if !this.numeric {
		if this.nills != nil || this.missings != nil || this.nulls != nil {
			rv = append(rv, nil)
		}

		for _, av := range this.booleans {
			rv = append(rv, av.getValue().Actual())
		}
	}

	for _, av := range this.floats {
		rv = append(rv, av.getValue().Actual())
	}

	for _, av := range this.ints {
		rv = append(rv, av.getValue().Actual())
	}

	if !this.numeric {
		for _, av := range this.strings {
			rv = append(rv, av.getValue().Actual())
		}

		for _, av := range this.arrays {
			rv = append(rv, av.getValue().Actual())
		}

		for _, av := range this.objects {
			rv = append(rv, av.getValue().Actual())
		}

		for _, av := range this.binaries {
			rv = append(rv, av.getValue().Actual())
		}
	}

	return rv
}

func (this *MultiSet) Items() []interface{} {
	if !this.collect {
		return nil
	}

	rv := make([]interface{}, 0, this.Len())

	if !this.numeric {
		if this.nills != nil {
			rv = append(rv, nil)
		}

		if this.missings != nil {
			rv = append(rv, this.missings.getValue())
		}

		if this.nulls != nil {
			rv = append(rv, this.nulls.getValue())
		}

		for _, av := range this.booleans {
			rv = append(rv, av.getValue())
		}
	}

	for _, av := range this.floats {
		rv = append(rv, av.getValue())
	}

	for _, av := range this.ints {
		rv = append(rv, av.getValue())
	}

	if !this.numeric {
		for _, av := range this.strings {
			rv = append(rv, av.getValue())
		}

		for _, av := range this.arrays {
			rv = append(rv, av.getValue())
		}

		for _, av := range this.objects {
			rv = append(rv, av.getValue())
		}

		for _, av := range this.binaries {
			rv = append(rv, av.getValue())
		}
	}

	return rv
}

func (this *MultiSet) Clear() {
	this.nills = nil
	this.missings = nil
	this.nulls = nil

	for k, _ := range this.floats {
		this.floats[k] = nil
		delete(this.floats, k)
	}

	for k, _ := range this.ints {
		this.ints[k] = nil
		delete(this.ints, k)
	}

	if this.numeric {
		return
	}

	for k, _ := range this.booleans {
		this.booleans[k] = nil
		delete(this.booleans, k)
	}

	for k, _ := range this.strings {
		this.strings[k] = nil
		delete(this.strings, k)
	}

	for k, _ := range this.arrays {
		this.arrays[k] = nil
		delete(this.arrays, k)
	}

	for k, _ := range this.objects {
		this.objects[k] = nil
		delete(this.objects, k)
	}

	for k, _ := range this.binaries {
		this.binaries[k] = nil
		delete(this.binaries, k)
	}
}

func (this *MultiSet) Copy() *MultiSet {
	rv := &MultiSet{}

	rv.collect = this.collect
	rv.nills = this.nills
	if this.missings != nil {
		rv.missings = this.missings.copy()
	}
	if this.nulls != nil {
		rv.nulls = this.nulls.copy()
	}
	rv.numeric = this.numeric

	rv.floats = make(map[float64]*valueCnt, 2*(1+len(this.floats)))
	rv.ints = make(map[int64]*valueCnt, 2*(1+len(this.ints)))

	if !rv.numeric {
		rv.booleans = make(map[bool]*valueCnt, len(this.booleans))
		rv.strings = make(map[string]*valueCnt, 2*(1+len(this.strings)))
		rv.arrays = make(map[string]*valueCnt, 2*(1+len(this.arrays)))
		rv.objects = make(map[string]*valueCnt, 2*(1+len(this.objects)))
		rv.binaries = make(map[string]*valueCnt, 2*(1+len((this.binaries))))

		for k, v := range this.booleans {
			rv.booleans[k] = v.copy()
		}

		for k, v := range this.strings {
			rv.strings[k] = v.copy()
		}

		for k, v := range this.arrays {
			rv.arrays[k] = v.copy()
		}

		for k, v := range this.objects {
			rv.objects[k] = v.copy()
		}

		for k, v := range this.binaries {
			rv.binaries[k] = v.copy()
		}
	}

	for k, v := range this.floats {
		rv.floats[k] = v.copy()
	}

	for k, v := range this.ints {
		rv.ints[k] = v.copy()
	}

	return rv
}

func (this *MultiSet) ObjectCap() int {
	return this.objectCap
}
