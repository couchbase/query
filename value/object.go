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

type objectValue map[string]interface{}

var EMPTY_OBJECT_VALUE = NewValue(map[string]interface{}{})

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

		buf.WriteString("\"")
		buf.WriteString(n)
		buf.WriteString("\":")

		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}

		buf.Write(b)
	}

	buf.WriteString("}")
	return buf.Bytes(), nil
}

func (this objectValue) Type() Type { return OBJECT }

func (this objectValue) Actual() interface{} {
	return (map[string]interface{})(this)
}

func (this objectValue) Equals(other Value) bool {
	switch other := other.(type) {
	case objectValue:
		return objectEquals(this, other)
	case *ScopeValue:
		return this.Equals(other.Value)
	case *annotatedValue:
		return this.Equals(other.Value)
	case *parsedValue:
		return this.Equals(other.parse())
	default:
		return false
	}
}

func (this objectValue) Collate(other Value) int {
	switch other := other.(type) {
	case objectValue:
		return objectCollate(this, other)
	case *ScopeValue:
		return this.Collate(other.Value)
	case *annotatedValue:
		return this.Collate(other.Value)
	case *parsedValue:
		return this.Collate(other.parse())
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
	bytes, err := this.MarshalJSON()
	if err != nil {
		panic(_MARSHAL_ERROR)
	}
	return bytes
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

func (this objectValue) Index(index int) (Value, bool) {
	return NULL_VALUE, false
}

func (this objectValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}

func (this objectValue) Slice(start, end int) (Value, bool) {
	return NULL_VALUE, false
}

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

func (this objectValue) Fields() map[string]interface{} {
	return this
}

func objectEquals(obj1, obj2 map[string]interface{}) bool {
	if len(obj1) != len(obj2) {
		return false
	}

	for name1, val1 := range obj1 {
		val2, ok := obj2[name1]
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
	delta := len(obj1) - len(obj2)
	if delta != 0 {
		return delta
	}

	// if not, proceed to do name by name comparision

	// collect all the names
	allmap := make(map[string]bool, len(obj1)<<1)
	for n, _ := range obj1 {
		allmap[n] = false
	}
	for n, _ := range obj2 {
		allmap[n] = false
	}

	allnames := make(sort.StringSlice, len(allmap))
	i := 0
	for n, _ := range allmap {
		allnames[i] = n
		i++
	}

	// sort the names
	allnames.Sort()

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
