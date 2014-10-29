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
)

type sliceValue []interface{}

var EMPTY_ARRAY_VALUE = NewValue([]interface{}{})

func (this sliceValue) MarshalJSON() ([]byte, error) {
	return marshalArray(this)
}

func (this sliceValue) Type() Type { return ARRAY }

func (this sliceValue) Actual() interface{} {
	return ([]interface{})(this)
}

func (this sliceValue) Equals(other Value) bool {
	switch other := other.(type) {
	case sliceValue:
		return arrayEquals(this, other)
	case *listValue:
		return arrayEquals(this, other.slice)
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

func (this sliceValue) Collate(other Value) int {
	switch other := other.(type) {
	case sliceValue:
		return arrayCollate(this, other)
	case *listValue:
		return arrayCollate(this, other.slice)
	case *ScopeValue:
		return this.Collate(other.Value)
	case *annotatedValue:
		return this.Collate(other.Value)
	case *parsedValue:
		return this.Collate(other.parse())
	default:
		return int(ARRAY - other.Type())
	}
}

func (this sliceValue) Truth() bool {
	return len(this) > 0
}

func (this sliceValue) Copy() Value {
	return sliceValue(copySlice(this, self))
}

func (this sliceValue) CopyForUpdate() Value {
	return &listValue{copySlice(this, copyForUpdate)}
}

func (this sliceValue) Field(field string) (Value, bool) {
	return NULL_VALUE, false
}

func (this sliceValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

func (this sliceValue) UnsetField(field string) error {
	return Unsettable(field)
}

func (this sliceValue) Index(index int) (Value, bool) {
	if index < 0 {
		index = len(this) + index
	}

	if index >= 0 && index < len(this) {
		return NewValue(this[index]), true
	}

	return missingIndex(index), false
}

// NOTE: Slices do NOT extend beyond length.
func (this sliceValue) SetIndex(index int, val interface{}) error {
	if index < 0 {
		index = len(this) + index
	}

	if index < 0 || index >= len(this) {
		return Unsettable(index)
	}

	switch val := val.(type) {
	case missingValue:
		this[index] = nil
	default:
		this[index] = val
	}

	return nil
}

func (this sliceValue) Slice(start, end int) (Value, bool) {
	if start < 0 {
		start = len(this) + start
	}

	if end < 0 {
		end = len(this) + end
	}

	if start <= end && start >= 0 && end <= len(this) {
		return NewValue(this[start:end]), true
	}

	return MISSING_VALUE, false
}

func (this sliceValue) SliceTail(start int) (Value, bool) {
	if start < 0 {
		start = len(this) + start
	}

	if start >= 0 {
		return NewValue(this[start:]), true
	}

	return MISSING_VALUE, false
}

func (this sliceValue) Descendants(buffer []interface{}) []interface{} {
	if cap(buffer) < len(buffer)+len(this) {
		buf2 := make([]interface{}, len(buffer), (len(buffer)+len(this)+1)<<1)
		copy(buf2, buffer)
		buffer = buf2
	}

	for _, child := range this {
		buffer = append(buffer, child)
		buffer = NewValue(child).Descendants(buffer)
	}

	return buffer
}

func (this sliceValue) Fields() map[string]interface{} {
	return nil
}

type listValue struct {
	slice sliceValue
}

func (this *listValue) MarshalJSON() ([]byte, error) {
	return this.slice.MarshalJSON()
}

func (this *listValue) Type() Type { return ARRAY }

func (this *listValue) Actual() interface{} {
	return this.slice.Actual()
}

func (this *listValue) Equals(other Value) bool {
	return this.slice.Equals(other)
}

func (this *listValue) Collate(other Value) int {
	return this.slice.Collate(other)
}

func (this *listValue) Truth() bool {
	return this.slice.Truth()
}

func (this *listValue) Copy() Value {
	return &listValue{this.slice.Copy().(sliceValue)}
}

func (this *listValue) CopyForUpdate() Value {
	return this.slice.CopyForUpdate()
}

func (this *listValue) Field(field string) (Value, bool) {
	return this.slice.Field(field)
}

func (this *listValue) SetField(field string, val interface{}) error {
	return this.slice.SetField(field, val)
}

func (this *listValue) UnsetField(field string) error {
	return this.slice.UnsetField(field)
}

func (this *listValue) Index(index int) (Value, bool) {
	return this.slice.Index(index)
}

func (this *listValue) SetIndex(index int, val interface{}) error {
	if index >= len(this.slice) {
		if index < cap(this.slice) {
			this.slice = this.slice[0 : index+1]
		} else {
			slice := make(sliceValue, index+1, (index+1)<<1)
			copy(slice, this.slice)
			this.slice = slice
		}
	}

	return this.slice.SetIndex(index, val)
}

func (this *listValue) Slice(start, end int) (Value, bool) {
	return this.slice.Slice(start, end)
}

func (this *listValue) SliceTail(start int) (Value, bool) {
	return this.slice.SliceTail(start)
}

func (this *listValue) Descendants(buffer []interface{}) []interface{} {
	return this.slice.Descendants(buffer)
}

func (this *listValue) Fields() map[string]interface{} {
	return this.slice.Fields()
}

func arrayEquals(array1, array2 []interface{}) bool {
	if len(array1) != len(array2) {
		return false
	}

	for i, item1 := range array1 {
		if !NewValue(item1).Equals(NewValue(array2[i])) {
			return false
		}
	}

	return true
}

// this code originally taken from walrus
// https://github.com/couchbaselabs/walrus
func arrayCollate(array1, array2 []interface{}) int {
	for i, item1 := range array1 {
		if i >= len(array2) {
			return 1
		}

		cmp := NewValue(item1).Collate(NewValue(array2[i]))
		if cmp != 0 {
			return cmp
		}
	}

	return len(array1) - len(array2)
}

func copySlice(source []interface{}, copier copyFunc) []interface{} {
	if source == nil {
		return nil
	}

	result := make([]interface{}, len(source))
	for i, v := range source {
		result[i] = copier(v)
	}

	return result
}

func marshalArray(slice []interface{}) (b []byte, err error) {
	if slice == nil {
		return _NULL_BYTES, nil
	}

	buf := bytes.NewBuffer(make([]byte, 0, 1<<8))
	buf.WriteString("[")

	for i, e := range slice {
		if i > 0 {
			buf.WriteString(",")
		}

		v := NewValue(e)
		b, err = json.Marshal(v)
		if err != nil {
			return
		}

		buf.Write(b)
	}

	buf.WriteString("]")
	return buf.Bytes(), nil
}
