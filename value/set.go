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
	"fmt"
)

// Set implements a hash set of values.
type Set struct {
	nills    bool
	missings Value
	nulls    Value
	booleans map[bool]Value
	numbers  map[float64]Value
	strings  map[string]Value
	arrays   map[string]Value
	objects  map[string]Value
	blobs    []Value
}

func NewSet(objectCap int) *Set {
	return &Set{
		booleans: make(map[bool]Value, 2),
		numbers:  make(map[float64]Value),
		strings:  make(map[string]Value),
		arrays:   make(map[string]Value),
		objects:  make(map[string]Value, objectCap),
		blobs:    make([]Value, 0, 16),
	}
}

func (this *Set) Add(item Value) {
	this.Put(item, item)
}

func (this *Set) Put(key, item Value) {
	if key == nil {
		this.nills = true
		return
	}

	switch key.Type() {
	case OBJECT:
		this.objects[string(key.Bytes())] = item
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
		this.arrays[string(key.Bytes())] = item
	case NOT_JSON:
		this.blobs = append(this.blobs, item) // FIXME: should compare bytes
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
	case OBJECT:
		delete(this.objects, string(key.Bytes()))
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
		delete(this.arrays, string(key.Bytes()))
	case NOT_JSON:
		// FIXME: should compare bytes
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
	case OBJECT:
		_, ok = this.objects[string(key.Bytes())]
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
		_, ok = this.arrays[string(key.Bytes())]
	case NOT_JSON:
		// FIXME: should compare bytes
		ok = false
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
