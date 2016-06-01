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
Set implements a hash set of Values.
*/
type Set struct {
	nills    bool
	missings Value
	nulls    Value
	booleans map[bool]Value
	numbers  map[float64]Value
	strings  map[string]Value
	arrays   map[string]Value
	objects  map[string]Value
	blobs    map[string]Value
}

var _MAP_CAP = 64

func NewSet(objectCap int) *Set {
	mapCap := util.MaxInt(objectCap, _MAP_CAP)

	return &Set{
		booleans: make(map[bool]Value, 2),
		numbers:  make(map[float64]Value, mapCap),
		strings:  make(map[string]Value, mapCap),
		arrays:   make(map[string]Value, _MAP_CAP),
		objects:  make(map[string]Value, objectCap),
		blobs:    make(map[string]Value, _MAP_CAP),
	}
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
		this.nills = true
		return
	}

	switch key.Type() {
	case MISSING:
		this.missings = item
	case NULL:
		this.nulls = item
	case BOOLEAN:
		this.booleans[key.Actual().(bool)] = item
	case NUMBER:
		this.numbers[key.Actual().(float64)] = item
	case STRING:
		this.strings[key.Actual().(string)] = item
	case ARRAY:
		this.arrays[key.String()] = item
	case OBJECT:
		this.objects[key.String()] = item
	case BINARY:
		str := base64.StdEncoding.EncodeToString(key.Actual().([]byte))
		this.blobs[str] = item
	default:
		panic(fmt.Sprintf("Unsupported value type %T.", key))
	}
}

func (this *Set) Remove(key Value) {
	if key == nil {
		this.nills = false
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
		delete(this.numbers, key.Actual().(float64))
	case STRING:
		delete(this.strings, key.Actual().(string))
	case ARRAY:
		delete(this.arrays, key.String())
	case OBJECT:
		delete(this.objects, key.String())
	case BINARY:
		str := base64.StdEncoding.EncodeToString(key.Actual().([]byte))
		delete(this.blobs, str)
	default:
		panic(fmt.Sprintf("Unsupported value type %T.", key))
	}
}

func (this *Set) Has(key Value) bool {
	if key == nil {
		return this.nills
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
		_, ok = this.numbers[key.Actual().(float64)]
	case STRING:
		_, ok = this.strings[key.Actual().(string)]
	case ARRAY:
		_, ok = this.arrays[key.String()]
	case OBJECT:
		_, ok = this.objects[key.String()]
	case BINARY:
		str := base64.StdEncoding.EncodeToString(key.Actual().([]byte))
		_, ok = this.blobs[str]
	default:
		panic(fmt.Sprintf("Unsupported value type %T.", key))
	}

	return ok
}

func (this *Set) Len() int {
	rv := len(this.booleans) + len(this.numbers) + len(this.strings) +
		len(this.arrays) + len(this.objects) + len(this.blobs)

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
	rv := make([]Value, 0, this.Len())

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

	for _, av := range this.numbers {
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

	for _, av := range this.blobs {
		rv = append(rv, av)
	}

	return rv
}

func (this *Set) Actuals() []interface{} {
	rv := make([]interface{}, 0, this.Len())

	if this.nills || this.missings != nil || this.nulls != nil {
		rv = append(rv, nil)
	}

	for _, av := range this.booleans {
		rv = append(rv, av.Actual())
	}

	for _, av := range this.numbers {
		rv = append(rv, av.Actual())
	}

	for _, av := range this.strings {
		rv = append(rv, av.Actual())
	}

	for _, av := range this.arrays {
		rv = append(rv, av.Actual())
	}

	for _, av := range this.objects {
		rv = append(rv, av.Actual())
	}

	for _, av := range this.blobs {
		rv = append(rv, av.Actual())
	}

	return rv
}
