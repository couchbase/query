//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package value

import (
	"io"
	"strconv"

	"github.com/couchbase/query/util"
)

/*
Type Empty struct
*/
type nullValue struct {
}

/*
Initialized as a pointer to an empty nullValue.
*/
var NULL_VALUE Value = &nullValue{}

/*
Returns a NULL_VALUE.
*/
func NewNullValue() Value {
	return NULL_VALUE
}

var _NULL_BYTES = []byte("null")

func (this *nullValue) String() string {
	return "null"
}

func (this *nullValue) ToString() string {
	return this.String()
}

func (this *nullValue) MarshalJSON() ([]byte, error) {
	return _NULL_BYTES, nil
}

func (this *nullValue) WriteJSON(order []string, w io.Writer, prefix, indent string, fast bool) error {
	_, err := w.Write(_NULL_BYTES)
	return err
}

func (this nullValue) WriteSpill(w io.Writer, buf []byte) error {
	b := []byte{_SPILL_TYPE_VALUE_NULL}
	_, err := w.Write(b)
	return err
}

func (this nullValue) ReadSpill(w io.Reader, buf []byte) error {
	return nil
}

/*
Type NULL
*/
func (this *nullValue) Type() Type {
	return NULL
}

/*
Returns nil.
*/
func (this *nullValue) Actual() interface{} {
	return nil
}

func (this *nullValue) ActualForIndex() interface{} {
	return nil
}

/*
Returns MISSING or NULL.
*/
func (this *nullValue) Equals(other Value) Value {
	other = other.unwrap()
	switch other.Type() {
	case MISSING:
		return other
	default:
		return this
	}
}

func (this *nullValue) EquivalentTo(other Value) bool {
	return other.Type() == NULL
}

/*
Returns the relative position of null wrt other.
*/
func (this *nullValue) Collate(other Value) int {
	return int(NULL - other.Type())
}

func (this *nullValue) Compare(other Value) Value {
	other = other.unwrap()
	switch other := other.(type) {
	case missingValue:
		return other
	default:
		return this
	}
}

/*
Returns false.
*/
func (this *nullValue) Truth() bool {
	return false
}

/*
Return receiver.
*/
func (this *nullValue) Copy() Value {
	return this
}

/*
Return receiver.
*/
func (this *nullValue) CopyForUpdate() Value {
	return this
}

/*
Calls missingField.
*/
func (this *nullValue) Field(field string) (Value, bool) {
	return missingField(field), false
}

/*
Not valid for NULL.
*/
func (this *nullValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

/*
Not valid for NULL.
*/
func (this *nullValue) UnsetField(field string) error {
	return Unsettable(field)
}

/*
Calls missingIndex.
*/
func (this *nullValue) Index(index int) (Value, bool) {
	return missingIndex(index), false
}

/*
Not valid for NULL.
*/
func (this *nullValue) SetIndex(index int, val interface{}) error {
	return Unsettable(strconv.Itoa(index))
}

/*
Returns NULL_VALUE
*/
func (this *nullValue) Slice(start, end int) (Value, bool) {
	return NULL_VALUE, false
}

/*
Returns NULL_VALUE
*/
func (this *nullValue) SliceTail(start int) (Value, bool) {
	return NULL_VALUE, false
}

/*
Returns the input buffer as is.
*/
func (this *nullValue) Descendants(buffer []interface{}) []interface{} {
	return buffer
}

/*
Null has no fields to list. Hence return nil.
*/
func (this *nullValue) Fields() map[string]interface{} {
	return nil
}

func (this *nullValue) FieldNames(buffer []string) []string {
	return nil
}

/*
Returns the input buffer as is.
*/
func (this *nullValue) DescendantPairs(buffer []util.IPair) []util.IPair {
	return buffer
}

/*
NULL is succeeded by FALSE.
*/
func (this *nullValue) Successor() Value {
	return FALSE_VALUE
}

func (this *nullValue) Track() {
}

func (this *nullValue) Recycle() {
}

func (this *nullValue) Tokens(set *Set, options Value) *Set {
	set.Add(this)
	return set
}

func (this *nullValue) ContainsToken(token, options Value) bool {
	return false
}

func (this *nullValue) ContainsMatchingToken(matcher MatchFunc, options Value) bool {
	return false
}

func (this *nullValue) Size() uint64 {
	return uint64(4) // len("null")
}

func (this *nullValue) unwrap() Value {
	return this
}
