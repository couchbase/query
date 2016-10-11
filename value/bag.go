//  Copieright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package value

import (
	"encoding/base64"
	"fmt"

	"github.com/couchbase/query/util"
)

/*
Bag implements a multiset of Values.
*/
type Bag struct {
	nills    *BagEntry
	missings *BagEntry
	nulls    *BagEntry
	booleans map[bool]*BagEntry
	floats   map[float64]*BagEntry
	ints     map[int64]*BagEntry
	strings  map[string]*BagEntry
	arrays   map[string]*BagEntry
	objects  map[string]*BagEntry
	binaries map[string]*BagEntry
}

type BagEntry struct {
	Value Value
	Count int
}

func NewBag(objectCap int) *Bag {
	mapCap := util.MaxInt(objectCap, _MAP_CAP)

	return &Bag{
		booleans: make(map[bool]*BagEntry, 2),
		floats:   make(map[float64]*BagEntry, mapCap),
		ints:     make(map[int64]*BagEntry, mapCap),
		strings:  make(map[string]*BagEntry, mapCap),
		arrays:   make(map[string]*BagEntry, _MAP_CAP),
		objects:  make(map[string]*BagEntry, objectCap),
		binaries: make(map[string]*BagEntry, _MAP_CAP),
	}
}

func (this *Bag) Add(item Value) {
	this.Put(item, item)
}

func (this *Bag) AddAll(items []interface{}) {
	for _, item := range items {
		this.Add(NewValue(item))
	}
}

func (this *Bag) Put(key, item Value) {
	if key == nil {
		if this.nills == nil {
			this.nills = &BagEntry{}
		}

		this.nills.Count++
		return
	}

	switch key.Type() {
	case MISSING:
		if this.missings == nil {
			this.missings = &BagEntry{Value: item}
		}

		this.missings.Count++
	case NULL:
		if this.nulls == nil {
			this.nulls = &BagEntry{Value: item}
		}

		this.nulls.Count++
	case BOOLEAN:
		akey := key.Actual().(bool)
		entry := this.booleans[akey]
		if entry == nil {
			entry = &BagEntry{Value: item}
			this.booleans[akey] = entry
		}

		entry.Count++
	case NUMBER:
		num := key.unwrap()
		switch num := num.(type) {
		case floatValue:
			f := float64(num)
			if IsInt(f) {
				akey := int64(f)
				entry := this.ints[akey]
				if entry == nil {
					entry = &BagEntry{Value: item}
					this.ints[akey] = entry
				}

				entry.Count++
			} else {
				akey := f
				entry := this.floats[akey]
				if entry == nil {
					entry = &BagEntry{Value: item}
					this.floats[akey] = entry
				}

				entry.Count++
			}
		case intValue:
			akey := int64(num)
			entry := this.ints[akey]
			if entry == nil {
				entry = &BagEntry{Value: item}
				this.ints[akey] = entry
			}

			entry.Count++
		default:
			panic(fmt.Sprintf("Unsupported value type %T.", key))
		}
	case STRING:
		akey := key.Actual().(string)
		entry := this.strings[akey]
		if entry == nil {
			entry = &BagEntry{Value: item}
			this.strings[akey] = entry
		}

		entry.Count++
	case ARRAY:
		akey := key.String()
		entry := this.arrays[akey]
		if entry == nil {
			entry = &BagEntry{Value: item}
			this.arrays[akey] = entry
		}

		entry.Count++
	case OBJECT:
		akey := key.String()
		entry := this.objects[akey]
		if entry == nil {
			entry = &BagEntry{Value: item}
			this.objects[akey] = entry
		}

		entry.Count++
	case BINARY:
		akey := base64.StdEncoding.EncodeToString(key.Actual().([]byte))
		entry := this.binaries[akey]
		if entry == nil {
			entry = &BagEntry{Value: item}
			this.binaries[akey] = entry
		}

		entry.Count++
	default:
		panic(fmt.Sprintf("Unsupported value type %T.", key))
	}
}

func (this *Bag) Entry(key Value) *BagEntry {
	if key == nil {
		return this.nills
	}

	switch key.Type() {
	case MISSING:
		return this.missings
	case NULL:
		return this.nulls
	case BOOLEAN:
		return this.booleans[key.Actual().(bool)]
	case NUMBER:
		num := key.unwrap()
		switch num := num.(type) {
		case floatValue:
			f := float64(num)
			if IsInt(f) {
				return this.ints[int64(f)]
			} else {
				return this.floats[f]
			}
		case intValue:
			return this.ints[int64(num)]
		default:
			panic(fmt.Sprintf("Unsupported value type %T.", key))
		}
	case STRING:
		return this.strings[key.Actual().(string)]
	case ARRAY:
		return this.arrays[key.String()]
	case OBJECT:
		return this.objects[key.String()]
	case BINARY:
		str := base64.StdEncoding.EncodeToString(key.Actual().([]byte))
		return this.binaries[str]
	default:
		panic(fmt.Sprintf("Unsupported value type %T.", key))
	}
}

func (this *Bag) DistinctLen() int {
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

func (this *Bag) Entries() []*BagEntry {
	rv := make([]*BagEntry, 0, this.DistinctLen())

	if this.nills != nil {
		rv = append(rv, this.nills)
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

	for _, av := range this.floats {
		rv = append(rv, av)
	}

	for _, av := range this.ints {
		rv = append(rv, av)
	}

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

	return rv
}

func (this *Bag) Clear() {
	this.nills = nil
	this.missings = nil
	this.nulls = nil

	for k, _ := range this.booleans {
		this.booleans[k] = nil
		delete(this.booleans, k)
	}

	for k, _ := range this.floats {
		this.floats[k] = nil
		delete(this.floats, k)
	}

	for k, _ := range this.ints {
		this.ints[k] = nil
		delete(this.ints, k)
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
