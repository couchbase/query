//  Copyright 2014-Present Couchbase, Inc.
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

/*
Set implements a hash set of Values.
*/
type Set struct {
	nills     bool
	missings  Value
	nulls     Value
	booleans  map[bool]Value
	floats    map[float64]Value
	ints      map[int64]Value
	strings   map[string]Value
	arrays    map[string]Value
	objects   map[string]Value
	binaries  map[string]Value
	collect   bool
	numeric   bool
	objectCap int
}

var _MAP_CAP = 64

func NewSet(objectCap int, collect, numeric bool) *Set {
	mapCap := util.MaxInt(objectCap, _MAP_CAP)

	rv := &Set{
		floats:    make(map[float64]Value, mapCap),
		ints:      make(map[int64]Value, mapCap),
		numeric:   numeric,
		collect:   collect,
		objectCap: objectCap,
	}
	if !numeric {
		rv.booleans = make(map[bool]Value, 2)
		rv.strings = make(map[string]Value, mapCap)
		rv.arrays = make(map[string]Value, _MAP_CAP)
		rv.objects = make(map[string]Value, objectCap)
		rv.binaries = make(map[string]Value, _MAP_CAP)
	}
	return rv
}

func (this *Set) Add(item Value) {
	this.Put(item, item)
}

func (this *Set) AddAll(items []interface{}) {
	for _, item := range items {
		this.Add(NewValue(item))
	}
}

func (this *Set) Put(key, item Value) {
	if key == nil {
		this.nills = (this.numeric == false)
		return
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
		this.missings = item
	case NULL:
		this.nulls = item
	case BOOLEAN:
		this.booleans[key.Actual().(bool)] = mapItem
	case NUMBER:
		num := key.unwrap()
		switch num := num.(type) {
		case floatValue:
			f := float64(num)
			if IsInt(f) {
				this.ints[int64(f)] = mapItem
			} else {
				this.floats[f] = mapItem
			}
		case intValue:
			this.ints[int64(num)] = mapItem
		default:
			panic(fmt.Sprintf("Unsupported value type %T.", key))
		}
	case STRING:
		this.strings[key.Actual().(string)] = mapItem
	case ARRAY:
		this.arrays[key.String()] = mapItem
	case OBJECT:
		this.objects[key.String()] = mapItem
	case BINARY:
		str := base64.StdEncoding.EncodeToString(key.Actual().([]byte))
		this.binaries[str] = mapItem
	default:
		panic(fmt.Sprintf("Unsupported value type %T.", key))
	}
}

func (this *Set) Remove(key Value) {
	if key == nil {
		this.nills = false
		return
	}

	if this.numeric && key.Type() != NUMBER {
		panic(fmt.Sprintf("Numeric set will not support value type %T.", key))
		return
	}

	switch key.Type() {
	case MISSING:
		this.missings = nil
	case NULL:
		this.nulls = nil
	case BOOLEAN:
		delete(this.booleans, key.Actual().(bool))
	case NUMBER:
		num := key.unwrap()
		switch num := num.(type) {
		case floatValue:
			f := float64(num)
			if IsInt(f) {
				delete(this.ints, int64(f))
			} else {
				delete(this.floats, f)
			}
		case intValue:
			delete(this.ints, int64(num))
		default:
			panic(fmt.Sprintf("Unsupported value type %T.", key))
		}
	case STRING:
		delete(this.strings, key.Actual().(string))
	case ARRAY:
		delete(this.arrays, key.String())
	case OBJECT:
		delete(this.objects, key.String())
	case BINARY:
		str := base64.StdEncoding.EncodeToString(key.Actual().([]byte))
		delete(this.binaries, str)
	default:
		panic(fmt.Sprintf("Unsupported value type %T.", key))
	}
}

func (this *Set) Has(key Value) bool {
	if key == nil {
		return this.nills
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

func (this *Set) Len() int {
	rv := len(this.booleans) + len(this.floats) + len(this.ints) + len(this.strings) +
		len(this.arrays) + len(this.objects) + len(this.binaries)

	if this.nills {
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

func (this *Set) Values() []Value {
	if !this.collect {
		return nil
	}

	rv := make([]Value, 0, this.Len())

	if !this.numeric {
		if this.nills {
			rv = append(rv, nil)
		}

		if this.missings != nil {
			rv = append(rv, this.missings)
		}

		if this.nulls != nil {
			rv = append(rv, this.nulls)
		}

		for _, av := range this.booleans {
			rv = append(rv, av)
		}
	}

	for _, av := range this.floats {
		rv = append(rv, av)
	}

	for _, av := range this.ints {
		rv = append(rv, av)
	}

	if !this.numeric {
		for _, av := range this.strings {
			rv = append(rv, av)
		}

		for _, av := range this.arrays {
			rv = append(rv, av)
		}

		for _, av := range this.objects {
			rv = append(rv, av)
		}

		for _, av := range this.binaries {
			rv = append(rv, av)
		}
	}

	return rv
}

func (this *Set) Actuals() []interface{} {
	if !this.collect {
		return nil
	}

	rv := make([]interface{}, 0, this.Len())

	if !this.numeric {
		if this.nills || this.missings != nil || this.nulls != nil {
			rv = append(rv, nil)
		}

		for _, av := range this.booleans {
			rv = append(rv, av.Actual())
		}
	}

	for _, av := range this.floats {
		rv = append(rv, av.Actual())
	}

	for _, av := range this.ints {
		rv = append(rv, av.Actual())
	}

	if !this.numeric {
		for _, av := range this.strings {
			rv = append(rv, av.Actual())
		}

		for _, av := range this.arrays {
			rv = append(rv, av.Actual())
		}

		for _, av := range this.objects {
			rv = append(rv, av.Actual())
		}

		for _, av := range this.binaries {
			rv = append(rv, av.Actual())
		}
	}

	return rv
}

func (this *Set) Items() []interface{} {
	if !this.collect {
		return nil
	}

	rv := make([]interface{}, 0, this.Len())

	if !this.numeric {
		if this.nills {
			rv = append(rv, nil)
		}

		if this.missings != nil {
			rv = append(rv, this.missings)
		}

		if this.nulls != nil {
			rv = append(rv, this.nulls)
		}

		for _, av := range this.booleans {
			rv = append(rv, av)
		}
	}

	for _, av := range this.floats {
		rv = append(rv, av)
	}

	for _, av := range this.ints {
		rv = append(rv, av)
	}

	if !this.numeric {
		for _, av := range this.strings {
			rv = append(rv, av)
		}

		for _, av := range this.arrays {
			rv = append(rv, av)
		}

		for _, av := range this.objects {
			rv = append(rv, av)
		}

		for _, av := range this.binaries {
			rv = append(rv, av)
		}
	}

	return rv
}

func (this *Set) Clear() {
	this.nills = false
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

func (this *Set) Copy() *Set {
	rv := &Set{}

	rv.collect = this.collect
	rv.nills = this.nills
	rv.missings = this.missings
	rv.nulls = this.nulls
	rv.numeric = this.numeric

	rv.floats = make(map[float64]Value, 2*(1+len(this.floats)))
	rv.ints = make(map[int64]Value, 2*(1+len(this.ints)))

	if !rv.numeric {
		rv.booleans = make(map[bool]Value, len(this.booleans))
		rv.strings = make(map[string]Value, 2*(1+len(this.strings)))
		rv.arrays = make(map[string]Value, 2*(1+len(this.arrays)))
		rv.objects = make(map[string]Value, 2*(1+len(this.objects)))
		rv.binaries = make(map[string]Value, 2*(1+len((this.binaries))))

		for k, v := range this.booleans {
			rv.booleans[k] = v
		}

		for k, v := range this.strings {
			rv.strings[k] = v
		}

		for k, v := range this.arrays {
			rv.arrays[k] = v
		}

		for k, v := range this.objects {
			rv.objects[k] = v
		}

		for k, v := range this.binaries {
			rv.binaries[k] = v
		}
	}

	for k, v := range this.floats {
		rv.floats[k] = v
	}

	for k, v := range this.ints {
		rv.ints[k] = v
	}

	return rv
}

func (this *Set) Union(other *Set) {
	this.AddAll(other.Items())
}

func (this *Set) Intersect(other *Set) {
	for _, v := range this.Values() {
		if !other.Has(v) {
			this.Remove(v)
		}
	}
}

func (this *Set) ObjectCap() int {
	return this.objectCap
}
