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
	"bytes"
	"encoding/json"
	"sort"
)

/*
objectValue is a type of map from string to interface.
*/
type objectValue map[string]interface{}

func (this objectValue) MarshalJSON() ([]byte, error) {
	if this == nil {
		return _NULL_BYTES, nil
	}

	buf := bytes.NewBuffer(make([]byte, 0, 1<<8))
	buf.WriteString("{")

	names := sortedNames(this)
	for i, n := range names {
		v := NewValue(this[n])
		if v.Type() == MISSING {
			continue
		}

		if i > 0 {
			buf.WriteString(",")
		}

		b, err := json.Marshal(n)
		if err != nil {
			return nil, err
		}

		buf.Write(b)
		buf.WriteString(":")

		b, err = json.Marshal(v)
		if err != nil {
			return nil, err
		}

		buf.Write(b)
	}

	buf.WriteString("}")
	return buf.Bytes(), nil
}

/*
Type OBJECT.
*/
func (this objectValue) Type() Type {
	return OBJECT
}

func (this objectValue) Actual() interface{} {
	return (map[string]interface{})(this)
}

func (this objectValue) Equals(other Value) Value {
	other = other.unwrap()
	switch other := other.(type) {
	case missingValue:
		return other
	case *nullValue:
		return other
	case objectValue:
		return objectEquals(this, other)
	}

	return FALSE_VALUE
}

func (this objectValue) Collate(other Value) int {
	other = other.unwrap()
	switch other := other.(type) {
	case objectValue:
		return objectCollate(this, other)
	default:
		return int(OBJECT - other.Type())
	}
}

func (this objectValue) Compare(other Value) Value {
	other = other.unwrap()
	switch other := other.(type) {
	case missingValue:
		return other
	case *nullValue:
		return other
	case objectValue:
		return objectCompare(this, other)
	default:
		return NewValue(int(OBJECT - other.Type()))
	}
}

/*
If length of the object is greater than 0 return true.
*/
func (this objectValue) Truth() bool {
	return len(this) > 0
}

func (this objectValue) Copy() Value {
	return objectValue(copyMap(this, self))
}

func (this objectValue) CopyForUpdate() Value {
	return objectValue(copyMap(this, copyForUpdate))
}

func (this objectValue) Field(field string) (Value, bool) {
	result, ok := this[field]
	if ok {
		return NewValue(result), true
	}

	return missingField(field), false
}

func (this objectValue) SetField(field string, val interface{}) error {
	switch val := val.(type) {
	case missingValue:
		delete(this, field)
	default:
		this[field] = val
	}

	return nil
}

func (this objectValue) UnsetField(field string) error {
	delete(this, field)
	return nil
}

/*
Calls missingIndex.
*/
func (this objectValue) Index(index int) (Value, bool) {
	return missingIndex(index), false
}

/*
Not valid for objects.
*/
func (this objectValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}

/*
Returns NULL_VALUE.
*/
func (this objectValue) Slice(start, end int) (Value, bool) {
	return NULL_VALUE, false
}

/*
Returns NULL_VALUE.
*/
func (this objectValue) SliceTail(start int) (Value, bool) {
	return NULL_VALUE, false
}

func (this objectValue) Descendants(buffer []interface{}) []interface{} {
	names := sortedNames(this)

	if cap(buffer) < len(buffer)+len(this) {
		buf2 := make([]interface{}, len(buffer), (len(buffer)+len(this)+1)<<1)
		copy(buf2, buffer)
		buffer = buf2
	}

	for _, name := range names {
		val := this[name]
		buffer = append(buffer, val)
		buffer = NewValue(val).Descendants(buffer)
	}

	return buffer
}

/*
Return the receiver this.
*/
func (this objectValue) Fields() map[string]interface{} {
	return this
}

/*
Return a successor object.
*/
func (this objectValue) Successor() Value {
	if len(this) == 0 {
		return _SMALL_OBJECT_VALUE
	}

	s := copyMap(this, self)
	names := sortedNames(this)
	n := names[len(names)-1]
	s[n] = NewValue(this[n]).Successor()
	return objectValue(s)
}

func (this objectValue) unwrap() Value {
	return this
}

var _SMALL_OBJECT_VALUE = objectValue(map[string]interface{}{"": nil})

func objectEquals(obj1, obj2 map[string]interface{}) Value {
	if len(obj1) != len(obj2) {
		return FALSE_VALUE
	}

	var null Value
	for name1, val1 := range obj1 {
		val2, ok := obj2[name1]
		if !ok {
			return FALSE_VALUE
		}

		v1 := NewValue(val1)
		v2 := NewValue(val2)
		eq := v1.Equals(v2)
		switch eq.Type() {
		case NULL:
			null = eq
		default:
			if !eq.Truth() {
				return eq
			}
		}
	}

	if null != nil {
		return null
	} else {
		return TRUE_VALUE
	}
}

/*
This code originally taken from https://github.com/couchbaselabs/walrus.
*/
func objectCollate(obj1, obj2 map[string]interface{}) int {
	// first see if one object is longer than the other
	delta := len(obj1) - len(obj2)
	if delta != 0 {
		return delta
	}

	// if not, proceed to do name by name comparision
	combined := combineNames(obj1, obj2)
	allnames := sortedNames(combined)

	// now compare the values associated with each name
	for _, name := range allnames {
		val1, ok := obj1[name]
		if !ok {
			// obj1 did not have this name, so it is larger
			return 1
		}

		val2, ok := obj2[name]
		if !ok {
			// ojb2 did not have this name, so it is larger
			return -1
		}

		// name was in both objects, so compare the corresponding values
		cmp := NewValue(val1).Collate(NewValue(val2))
		if cmp != 0 {
			return cmp
		}
	}

	// all names and values are equal
	return 0
}

func objectCompare(obj1, obj2 map[string]interface{}) Value {
	// first see if one object is longer than the other
	delta := len(obj1) - len(obj2)
	if delta != 0 {
		return NewValue(delta)
	}

	// if not, proceed to do name by name comparision
	combined := combineNames(obj1, obj2)
	allnames := sortedNames(combined)

	// now compare the values associated with each name
	for _, name := range allnames {
		val1, ok := obj1[name]
		if !ok {
			// obj1 did not have this name, so it is larger
			return ONE_VALUE
		}

		val2, ok := obj2[name]
		if !ok {
			// ojb2 did not have this name, so it is larger
			return NEG_ONE_VALUE
		}

		// name was in both objects, so compare the corresponding values
		cmp := NewValue(val1).Compare(NewValue(val2))
		if !cmp.Equals(ZERO_VALUE).Truth() {
			return cmp
		}
	}

	// all names and values are equal
	return ZERO_VALUE
}

func copyMap(source map[string]interface{}, copier copyFunc) map[string]interface{} {
	if source == nil {
		return nil
	}

	result := make(map[string]interface{}, len(source))
	for n, v := range source {
		result[n] = copier(v)
	}

	return result
}

func sortedNames(obj map[string]interface{}) []string {
	names := make(sort.StringSlice, 0, len(obj))
	for name, _ := range obj {
		names = append(names, name)
	}

	names.Sort()
	return names
}

func combineNames(objs ...map[string]interface{}) map[string]interface{} {
	n := 0
	for _, obj := range objs {
		n += len(obj)
	}

	all := make(map[string]interface{}, n)

	for _, obj := range objs {
		for k, _ := range obj {
			all[k] = nil
		}
	}

	return all
}
