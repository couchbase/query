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
)

/*
Set implements a hash set of values. It is defined as
a struct containing different values. _nills_ which 
is a bool type, missings and nulls which are Value's, 
booleans, numbers, strings, arrays, objects and blobs
which are maps from bool,float64,and string to Value 
respectively.
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

/*
The maximum defined capacity for a map is 64 bytes.
*/
var _MAP_CAP = 64

/*
Initialize the different elements in the struct. The input
integer decides the capacity of the objects map. 
*/
func NewSet(objectCap int) *Set {
	return &Set{
		booleans: make(map[bool]Value, 2),
		numbers:  make(map[float64]Value, _MAP_CAP),
		strings:  make(map[string]Value, _MAP_CAP),
		arrays:   make(map[string]Value, _MAP_CAP),
		objects:  make(map[string]Value, objectCap),
		blobs:    make(map[string]Value, _MAP_CAP),
	}
}

/*
Adds a Value item to the receiver by calling Put.
*/
func (this *Set) Add(item Value) {
	this.Put(item, item)
}

/*
It adds an item of key and value to the Set. If the key is 
nil then the nills attribute is set to true for the struct 
and return from the function. We check for the keys type. 
If it is an object then call MarshalJSON and set the object 
value in the struct to this item. Similarly for array. If the 
type is null then set the nulls attribute for the receiver 
to item. For missing, the missings attribute is set.If the type 
is Boolean, number or string call actual for the key to 
convert it into native golag and then cast it to bool, float64 
or string before setting the value for that key to item. For 
the type binary, append the item to the blobs after 
converting from binary to string using base64. The default 
case throws an error since the value type is unsupported.
*/
func (this *Set) Put(key, item Value) {
	if key == nil {
		this.nills = true
		return
	}

	switch key.Type() {
	case OBJECT:
		b, _ := key.MarshalJSON()
		this.objects[string(b)] = item
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
		b, _ := key.MarshalJSON()
		this.arrays[string(b)] = item
	case BINARY:
		str := base64.StdEncoding.EncodeToString(key.Actual().([]byte))
		this.blobs[str] = item
	default:
		panic(fmt.Sprintf("Unsupported value type %T.", key))
	}
}

/*
Deletes input key entry from the Set. If key is nil then set 
this.nills to false and return. Else depending on the key 
type, delete.
*/
func (this *Set) Remove(key Value) {
	if key == nil {
		this.nills = false
		return
	}

	switch key.Type() {
	case OBJECT:
		b, _ := key.MarshalJSON()
		delete(this.objects, string(b))
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
		b, _ := key.MarshalJSON()
		delete(this.arrays, string(b))
	case BINARY:
		str := base64.StdEncoding.EncodeToString(key.Actual().([]byte))
		delete(this.blobs, str)
	default:
		panic(fmt.Sprintf("Unsupported value type %T.", key))
	}
}

/*
Checks if the Set has a key, and returns that corresponding value.
Has sets a variable ok to depict true if that particular field is 
set. Finally return that boolean value. 
*/
func (this *Set) Has(key Value) bool {
	if key == nil {
		return this.nills
	}

	ok := false
	switch key.Type() {
	case OBJECT:
		b, _ := key.MarshalJSON()
		_, ok = this.objects[string(b)]
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
		b, _ := key.MarshalJSON()
		_, ok = this.arrays[string(b)]
	case BINARY:
		str := base64.StdEncoding.EncodeToString(key.Actual().([]byte))
		_, ok = this.blobs[str]
	default:
		panic(fmt.Sprintf("Unsupported value type %T.", key))
	}

	return ok
}

/*
Returns the length of the set by adding the length of each 
element. For missings and nulls it increments by one only 
if they arent nil, and if nills are true then it increments 
the length by one. The length is then returned. 
*/
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

/*
Returns a slice of Value that contains all the values in 
the Set. It creates a variable that is a slice of Values 
and appends all the existing elements in the set to it.
*/
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

/*
Convert the set elements into golang Type by calling
Actual for that value, append it to a slice
of interfaces, and return the slice.
*/
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
