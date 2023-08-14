//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package value

import (
	"bytes"
	"io"
	"strconv"

	"github.com/couchbase/query/util"
)

/*
sliceValue is defined as a slice of interfaces.
*/
type sliceValue []interface{}

/*
EMPTY_ARRAY_VALUE is initialized as a slice of interface.
*/
var EMPTY_ARRAY_VALUE Value = sliceValue([]interface{}{})
var TRUE_ARRAY_VALUE Value = sliceValue([]interface{}{true})

func (this sliceValue) String() string {
	return marshalString(this)
}

func (this sliceValue) ToString() string {
	return marshalString(this)
}

func (this sliceValue) MarshalJSON() ([]byte, error) {
	return marshalArray(this)
}

func (this sliceValue) WriteSpill(w io.Writer, buf []byte) error {
	b := []byte{_SPILL_TYPE_VALUE}
	_, err := w.Write(b)
	if err == nil {
		err = writeSpillValue(w, ([]interface{})(this), buf)
	}
	return err
}

func (this sliceValue) ReadSpill(r io.Reader, buf []byte) error {
	v, err := readSpillValue(r, buf)
	if err == nil && v != nil {
		this = sliceValue(v.([]interface{}))
	} else {
		this = nil
	}
	return err
}

func (this sliceValue) WriteXML(order []string, w io.Writer, prefix string, indent string, fast bool) error {
	var err error

	if this == nil {
		_, err = w.Write(_NULL_XML)
		return err
	}

	// TODO workaround for GSI using an old golang that doesn't know about StringWriter
	stringWriter := w.(*bytes.Buffer)

	var fullPrefix string
	writePrefix := (prefix != "" && indent != "")
	if writePrefix {
		fullPrefix = getFullPrefix(prefix, indent)
	}

	if writePrefix {
		if _, err = stringWriter.WriteString(fullPrefix[:len(prefix)+1]); err != nil {
			return err
		}
	}
	if len(this) == 0 {
		_, err = stringWriter.WriteString("<array/>")
		return err
	}

	if _, err = stringWriter.WriteString("<array>"); err != nil {
		return err
	}

	for _, e := range this {
		v := NewValue(e)
		if writePrefix {
			if err = v.WriteXML(order, w, fullPrefix[1:], indent, fast); err != nil {
				return err
			}
		} else {
			if err = v.WriteXML(order, w, "", "", fast); err != nil {
				return err
			}
		}
	}

	if writePrefix {
		if _, err = stringWriter.WriteString(fullPrefix[:len(prefix)+1]); err != nil {
			return err
		}
	}
	_, err = stringWriter.WriteString("</array>")
	return err
}

func (this sliceValue) WriteJSON(order []string, w io.Writer, prefix, indent string, fast bool) (err error) {
	if this == nil {
		_, err = w.Write(_NULL_BYTES)
		return
	}

	// TODO workaround for GSI using an old golang that doesn't know about StringWriter
	stringWriter := w.(*bytes.Buffer)

	if _, err = stringWriter.WriteString("["); err != nil {
		return
	}

	var fullPrefix string
	writePrefix := (prefix != "" && indent != "")
	if writePrefix {
		fullPrefix = getFullPrefix(prefix, indent)
	}

	for i, e := range this {
		if i > 0 {
			if _, err = stringWriter.WriteString(","); err != nil {
				return
			}
		}
		v := NewValue(e)
		if writePrefix {
			if _, err = stringWriter.WriteString(fullPrefix); err != nil {
				return
			}
			if err = v.WriteJSON(order, w, fullPrefix[1:], indent, fast); err != nil {
				return
			}
		} else {
			if err = v.WriteJSON(order, w, "", "", fast); err != nil {
				return
			}
		}
	}

	if len(this) > 0 && prefix != "" {
		if _, err = stringWriter.WriteString(fullPrefix[:len(prefix)+1]); err != nil {
			return
		}
	}
	_, err = stringWriter.WriteString("]")
	return err
}

/*
Type ARRAY
*/
func (this sliceValue) Type() Type {
	return ARRAY
}

func (this sliceValue) Actual() interface{} {
	return ([]interface{})(this)
}

func (this sliceValue) ActualForIndex() interface{} {
	return ([]interface{})(this)
}

func (this sliceValue) Equals(other Value) Value {
	other = other.unwrap()
	switch other := other.(type) {
	case missingValue:
		return other
	case *nullValue:
		return other
	case sliceValue:
		return arrayEquals(this, other)
	case copiedSliceValue:
		return arrayEquals(this, other.sliceValue)
	case *listValue:
		return arrayEquals(this, other.slice)
	default:
		return FALSE_VALUE
	}
}

func (this sliceValue) EquivalentTo(other Value) bool {
	other = other.unwrap()
	switch other := other.(type) {
	case sliceValue:
		return arrayEquivalent(this, other)
	case copiedSliceValue:
		return arrayEquivalent(this, other.sliceValue)
	case *listValue:
		return arrayEquivalent(this, other.slice)
	default:
		return false
	}
}

func (this sliceValue) Collate(other Value) int {
	other = other.unwrap()
	switch other := other.(type) {
	case sliceValue:
		return arrayCollate(this, other)
	case copiedSliceValue:
		return arrayCollate(this, other.sliceValue)
	case *listValue:
		return arrayCollate(this, other.slice)
	default:
		return int(ARRAY - other.Type())
	}
}

func (this sliceValue) Compare(other Value) Value {
	other = other.unwrap()
	switch other := other.(type) {
	case missingValue:
		return other
	case *nullValue:
		return other
	case sliceValue:
		return arrayCompare(this, other)
	case copiedSliceValue:
		return arrayCompare(this, other.sliceValue)
	case *listValue:
		return arrayCompare(this, other.slice)
	default:
		return intValue(int(ARRAY - other.Type()))
	}
}

/*
If length of the slice  greater than 0, its a valid slice.
Return true.
*/
func (this sliceValue) Truth() bool {
	return len(this) > 0
}

/*
Call copySlice on the receiver and self and cast it to a
sliceValue.
*/
func (this sliceValue) Copy() Value {
	return copiedSliceValue{sliceValue: sliceValue(copySlice(this, self))}
}

/*
Call copySlice on the receiver and copyForUpdate, return a
pointer to a list value encapsulating it.This allows for a
copy for every element of the array by calling its
CopyForUpdate function.
*/
func (this sliceValue) CopyForUpdate() Value {
	return &listValue{copySlice(this, copyForUpdate)}
}

/*
Calls missingField.
*/
func (this sliceValue) Field(field string) (Value, bool) {
	return missingField(field), false
}

/*
Not valid for array/slice.
*/
func (this sliceValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

/*
Not valid for array/slice.
*/
func (this sliceValue) UnsetField(field string) error {
	return Unsettable(field)
}

/*
If the input index is negative then count the index from
the last element.
*/
func (this sliceValue) Index(index int) (Value, bool) {
	if index < 0 {
		index = len(this) + index
	}

	if index >= 0 && index < len(this) {
		return NewValue(this[index]), true
	}

	return missingIndex(index), false
}

/*
If index is negative, add to the length and get the actual
index. In the event the new adjusted index is less than 0
or greater than/equal to the length of the slice return
Unsettable since Slices do NOT extend beyond length. If it
is a valid index, check the type of value. If it is a
missing value, set it to nil (do not add this field) and
if anything else, add the value at the particular index.
For all other cases, return a nil.
*/
func (this sliceValue) SetIndex(index int, val interface{}) error {
	if index < 0 {
		index = len(this) + index
	}

	if index < 0 || index >= len(this) {
		return Unsettable(strconv.Itoa(index))
	}

	switch val := val.(type) {
	case missingValue:
		this[index] = nil
	default:
		this[index] = val
	}

	return nil
}

/*
If the start and/or end index is -ve, as per the N1QL specs,
add it to the length to get the actual index (from the end).
If it is a valid slice (start<=end, start >=0 and end less
than the length), return the slice by creating a valid value
and also return true. If the indices are not valid return a
missing value and false.
*/
func (this sliceValue) Slice(start, end int) (Value, bool) {
	if start < 0 {
		start = len(this) + start
	}

	if end < 0 {
		end = len(this) + end
	}

	if start <= end && start >= 0 && end <= len(this) {
		return sliceValue(this[start:end]), true
	}

	return MISSING_VALUE, false
}

/*
If the start index is -ve, as per the N1QL specs, add it to
the length to get the actual index (from the end). If it is
valid(+ve) then return a slice from start till the length
of the slice and a bool value true. If the indices are not
valid return a missing value and false.
*/
func (this sliceValue) SliceTail(start int) (Value, bool) {
	if start < 0 {
		start = len(this) + start
	}

	if start >= 0 && start < len(this) {
		return sliceValue(this[start:]), true
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

func (this sliceValue) FieldNames(buffer []string) []string {
	return nil
}

func (this sliceValue) DescendantPairs(buffer []util.IPair) []util.IPair {
	if cap(buffer) < len(buffer)+len(this) {
		buf2 := make([]util.IPair, len(buffer), (len(buffer)+len(this)+1)<<1)
		copy(buf2, buffer)
		buffer = buf2
	}

	for i, child := range this {
		buffer = append(buffer, util.IPair{i, child})
		buffer = NewValue(child).DescendantPairs(buffer)
	}

	return buffer
}

/*
Append a small value.
*/
func (this sliceValue) Successor() Value {
	if len(this) == 0 {
		return _SMALL_ARRAY_VALUE
	}

	return sliceValue(append(this, nil))
}

func (this sliceValue) Track() {
}

func (this sliceValue) Recycle() {
}

func (this sliceValue) Tokens(set *Set, options Value) *Set {
	for _, v := range this {
		set = NewValue(v).Tokens(set, options)
	}

	return set
}

func (this sliceValue) ContainsToken(token, options Value) bool {
	for _, v := range this {
		if NewValue(v).ContainsToken(token, options) {
			return true
		}
	}

	return false
}

func (this sliceValue) ContainsMatchingToken(matcher MatchFunc, options Value) bool {
	for _, v := range this {
		if NewValue(v).ContainsMatchingToken(matcher, options) {
			return true
		}
	}

	return false
}

func (this sliceValue) Size() uint64 {
	size := uint64(_INTERFACE_SIZE * len(this))
	for e, _ := range this {
		size += anySize(this[e])
	}
	return size
}

func (this sliceValue) unwrap() Value {
	return this
}

var _SMALL_ARRAY_VALUE = sliceValue([]interface{}{nil})

/*
Wrap a sliceValue that can be reallocated for resizing.
*/
type listValue struct {
	slice sliceValue
}

func (this *listValue) String() string {
	return this.slice.String()
}

func (this *listValue) ToString() string {
	return this.slice.String()
}

func (this *listValue) MarshalJSON() ([]byte, error) {
	return this.slice.MarshalJSON()
}

func (this *listValue) WriteXML(order []string, w io.Writer, prefix, indent string, fast bool) (err error) {
	return this.slice.WriteXML(order, w, prefix, indent, fast)
}

func (this *listValue) WriteJSON(order []string, w io.Writer, prefix, indent string, fast bool) (err error) {
	return this.slice.WriteJSON(order, w, prefix, indent, fast)
}

func (this *listValue) WriteSpill(w io.Writer, buf []byte) error {
	b := []byte{_SPILL_TYPE_VALUE_LIST}
	_, err := w.Write(b)
	if err == nil {
		err = writeSpillValue(w, ([]interface{})(this.slice), buf)
	}
	return err
}

func (this *listValue) ReadSpill(r io.Reader, buf []byte) error {
	v, err := readSpillValue(r, buf)
	if err == nil && v != nil {
		this.slice = sliceValue(v.([]interface{}))
	} else {
		this.slice = nil
	}
	return err
}

/*
Type ARRAY.
*/
func (this *listValue) Type() Type { return ARRAY }

func (this *listValue) Actual() interface{} {
	return this.slice.Actual()
}

func (this *listValue) ActualForIndex() interface{} {
	return this.slice.ActualForIndex()
}

func (this *listValue) Equals(other Value) Value {
	return this.slice.Equals(other)
}

func (this *listValue) EquivalentTo(other Value) bool {
	return this.slice.EquivalentTo(other)
}

func (this *listValue) Collate(other Value) int {
	return this.slice.Collate(other)
}

func (this *listValue) Compare(other Value) Value {
	return this.slice.Compare(other)
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

func (this *listValue) FieldNames(buffer []string) []string {
	return this.slice.FieldNames(buffer)
}

func (this *listValue) DescendantPairs(buffer []util.IPair) []util.IPair {
	return this.slice.DescendantPairs(buffer)
}

func (this *listValue) Successor() Value {
	return this.slice.Successor()
}

func (this *listValue) Track() {
}

func (this *listValue) Recycle() {
}

func (this *listValue) Tokens(set *Set, options Value) *Set {
	return this.slice.Tokens(set, options)
}

func (this *listValue) ContainsToken(token, options Value) bool {
	return this.slice.ContainsToken(token, options)
}

func (this *listValue) ContainsMatchingToken(matcher MatchFunc, options Value) bool {
	return this.slice.ContainsMatchingToken(matcher, options)
}

func (this *listValue) Size() uint64 {
	size := uint64(_INTERFACE_SIZE * len(this.slice))
	for e, _ := range this.slice {
		size += anySize(this.slice[e])
	}
	return size
}

func (this *listValue) unwrap() Value {
	return this
}

func arrayEquals(array1, array2 []interface{}) Value {
	if len(array1) != len(array2) {
		return FALSE_VALUE
	}

	var missing, null Value
	for i, item1 := range array1 {
		eq := NewValue(item1).Equals(NewValue(array2[i]))
		switch eq.Type() {
		case MISSING:
			missing = eq
		case NULL:
			null = eq
		default:
			if !eq.Truth() {
				return eq
			}
		}
	}

	if missing != nil {
		return missing
	} else if null != nil {
		return null
	} else {
		return TRUE_VALUE
	}
}

func arrayEquivalent(array1, array2 []interface{}) bool {
	if len(array1) != len(array2) {
		return false
	}

	for i, item1 := range array1 {
		if !NewValue(item1).EquivalentTo(NewValue(array2[i])) {
			return false
		}
	}

	return true
}

/*
This code originally taken from https://github.com/couchbaselabs/walrus
*/
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

func arrayCompare(array1, array2 []interface{}) Value {
	for i, item1 := range array1 {
		if i >= len(array2) {
			return ONE_VALUE
		}

		cmp := NewValue(item1).Compare(NewValue(array2[i]))
		if !cmp.Equals(ZERO_VALUE).Truth() {
			return cmp
		}
	}

	return intValue(len(array1) - len(array2))
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

	buf := bytes.NewBuffer(make([]byte, 0, 256))
	buf.WriteString("[")

	for i, e := range slice {
		if i > 0 {
			buf.WriteString(",")
		}

		v := NewValue(e)
		b, err = v.MarshalJSON()
		if err != nil {
			return
		}

		buf.Write(b)
	}

	buf.WriteString("]")
	return buf.Bytes(), nil
}
