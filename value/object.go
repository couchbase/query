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
	"sort"

	json "github.com/dustin/gojson"
)

type objectValue map[string]interface{}

func (this objectValue) Type() int {
	return OBJECT
}

func (this objectValue) Actual() interface{} {
	return (map[string]interface{})(this)
}

func (this objectValue) Equals(other Value) bool {
	switch other := other.(type) {
	case objectValue:
		return objectEquals(this, other)
	case *correlatedValue:
		return objectEquals(this, other.entries)
	case *parsedValue:
		return this.Equals(other.parse())
	case *annotatedValue:
		return this.Equals(other.Value)
	default:
		return false
	}
}

func (this objectValue) Collate(other Value) int {
	switch other := other.(type) {
	case objectValue:
		return objectCollate(this, other)
	case *correlatedValue:
		return objectCollate(this, other.entries)
	case *parsedValue:
		return this.Collate(other.parse())
	case *annotatedValue:
		return this.Collate(other.Value)
	default:
		return 1
	}
}

func (this objectValue) Truth() bool {
	return len(this) > 0
}

func (this objectValue) Copy() Value {
	return objectValue(copyMap(this, self))
}

func (this objectValue) CopyForUpdate() Value {
	return objectValue(copyMap(this, copyForUpdate))
}

func (this objectValue) Bytes() []byte {
	bytes, err := json.Marshal(this.Actual())
	if err != nil {
		panic(_MARSHAL_ERROR)
	}
	return bytes
}

func (this objectValue) Field(field string) (Value, error) {
	result, ok := this[field]
	if ok {
		return NewValue(result), nil
	}

	// consistent with parsedValue
	return nil, Undefined(field)
}

func (this objectValue) SetField(field string, val interface{}) error {
	this[field] = val
	return nil
}

func (this objectValue) Index(index int) (Value, error) {
	return nil, Undefined(index)
}

func (this objectValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}

func objectEquals(obj1, obj2 map[string]interface{}) bool {
	if len(obj1) != len(obj2) {
		return false
	}

	for key1, val1 := range obj1 {
		val2, ok := obj2[key1]
		if !ok || !NewValue(val1).Equals(NewValue(val2)) {
			return false
		}
	}

	return true
}

// this code originally taken from walrus
// https://github.com/couchbaselabs/walrus
func objectCollate(obj1, obj2 map[string]interface{}) int {
	// first see if one object is larger than the other
	if len(obj1) < len(obj2) {
		return -1
	} else if len(obj1) > len(obj2) {
		return 1
	}

	// if not, proceed to do key by key comparision

	// collect all the keys
	allmap := make(map[string]bool, len(obj1)+len(obj2))
	for k, _ := range obj1 {
		allmap[k] = false
	}
	for k, _ := range obj2 {
		allmap[k] = false
	}

	allkeys := make(sort.StringSlice, len(allmap))
	i := 0
	for k, _ := range allmap {
		allkeys[i] = k
		i++
	}

	// sort the keys
	allkeys.Sort()

	// now compare the values associated with each key
	for _, key := range allkeys {
		val1, ok := obj1[key]
		if !ok {
			// obj1 didn't have this key, so it is smaller
			return -1
		}
		val2, ok := obj2[key]
		if !ok {
			// ojb2 didnt have this key, so its smaller
			return 1
		}
		// key was in both objects, need to compare them
		if cmp := NewValue(val1).Collate(NewValue(val2)); cmp != 0 {
			return cmp
		}
	}

	return 0
}
