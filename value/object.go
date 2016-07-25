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
	"io"
	"sort"

	"github.com/couchbase/query/util"
)

/*
objectValue is a type of map from string to interface.
*/
type objectValue map[string]interface{}

func (this objectValue) String() string {
	return marshalString(this)
}

func (this objectValue) MarshalJSON() ([]byte, error) {
	if this == nil {
		return _NULL_BYTES, nil
	}

	buf := bytes.NewBuffer(make([]byte, 0, 256))
	buf.WriteString("{")

	names := _NAME_POOL.GetSized(len(this))
	defer _NAME_POOL.Put(names)
	names = sortedNames(this, names)

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

		b, err = v.MarshalJSON()
		if err != nil {
			return nil, err
		}

		buf.Write(b)
	}

	buf.WriteString("}")
	return buf.Bytes(), nil
}

func (this objectValue) WriteJSON(w io.Writer, prefix, indent string) (err error) {
	if this == nil {
		_, err = w.Write(_NULL_BYTES)
		return
	}

	if _, err = w.Write([]byte{'{'}); err != nil {
		return
	}

	names := _NAME_POOL.GetSized(len(this))
	defer _NAME_POOL.Put(names)
	names = sortedNames(this, names)

	newPrefix := prefix + indent

	for i, n := range names {
		v := NewValue(this[n])
		if v.Type() == MISSING {
			continue
		}

		if i > 0 {
			if _, err = w.Write([]byte{','}); err != nil {
				return
			}
		}

		if err = writeJsonNewline(w, newPrefix); err != nil {
			return
		}

		b, err := json.Marshal(n)
		if err != nil {
			return err
		}

		if _, err = w.Write(b); err != nil {
			return err
		}
		if _, err = w.Write([]byte{':'}); err != nil {
			return err
		}
		if prefix != "" || indent != "" {
			if _, err = w.Write([]byte{' '}); err != nil {
				return err
			}
		}

		if err = v.WriteJSON(w, newPrefix, indent); err != nil {
			return err
		}
	}

	if len(names) > 0 {
		if err = writeJsonNewline(w, prefix); err != nil {
			return
		}
	}
	_, err = w.Write([]byte{'}'})
	return err
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
	default:
		return FALSE_VALUE
	}
}

func (this objectValue) EquivalentTo(other Value) bool {
	other = other.unwrap()
	switch other := other.(type) {
	case objectValue:
		return objectEquivalent(this, other)
	default:
		return false
	}
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
		return intValue(int(OBJECT - other.Type()))
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
	names := _NAME_POOL.GetSized(len(this))
	defer _NAME_POOL.Put(names)
	names = sortedNames(this, names)

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

func (this objectValue) Fields() map[string]interface{} {
	return this
}

func (this objectValue) FieldNames(buffer []string) []string {
	return sortedNames(this, buffer)
}

func (this objectValue) DescendantPairs(buffer []util.IPair) []util.IPair {
	names := _NAME_POOL.GetSized(len(this))
	defer _NAME_POOL.Put(names)
	names = sortedNames(this, names)

	if cap(buffer) < len(buffer)+len(this) {
		buf2 := make([]util.IPair, len(buffer), (len(buffer)+len(this)+1)<<1)
		copy(buf2, buffer)
		buffer = buf2
	}

	for _, name := range names {
		val := this[name]
		buffer = append(buffer, util.IPair{name, val})
		buffer = NewValue(val).DescendantPairs(buffer)
	}

	return buffer
}

/*
Return a successor object.
*/
func (this objectValue) Successor() Value {
	if len(this) == 0 {
		return _SMALL_OBJECT_VALUE
	}

	s := copyMap(this, self)

	names := _NAME_POOL.GetSized(len(this))
	defer _NAME_POOL.Put(names)
	names = sortedNames(this, names)

	n := names[len(names)-1]
	s[n] = NewValue(this[n]).Successor()
	return objectValue(s)
}

func (this objectValue) Recycle() {
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

func objectEquivalent(obj1, obj2 map[string]interface{}) bool {
	if len(obj1) != len(obj2) {
		return false
	}

	for name1, val1 := range obj1 {
		val2, ok := obj2[name1]
		if !ok {
			return false
		}

		if !NewValue(val1).EquivalentTo(NewValue(val2)) {
			return false
		}
	}

	return true
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

	allNames := _NAME_POOL.GetSized(len(combined))
	defer _NAME_POOL.Put(allNames)
	allNames = sortedNames(combined, allNames)

	// now compare the values associated with each name
	for _, name := range allNames {
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
		return intValue(delta)
	}

	// if not, proceed to do name by name comparision
	combined := combineNames(obj1, obj2)

	allNames := _NAME_POOL.GetSized(len(combined))
	defer _NAME_POOL.Put(allNames)
	allNames = sortedNames(combined, allNames)

	// now compare the values associated with each name
	for _, name := range allNames {
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

func sortedNames(obj map[string]interface{}, buffer []string) []string {
	i := 0
	for name, _ := range obj {
		buffer[i] = name
		i++
	}

	sort.Strings(buffer)
	return buffer
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

func writeJsonNewline(w io.Writer, prefix string) (err error) {
	if prefix != "" {
		if _, err = w.Write([]byte{'\n'}); err != nil {
			return
		}

		_, err = io.WriteString(w, prefix)
	}

	return
}

var _NAME_POOL = util.NewStringPool(64)
