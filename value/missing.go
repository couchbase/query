//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package value

import (
	"fmt"
	"io"
	"strconv"

	"github.com/couchbase/query/util"
)

/*
Missing value is of type string
*/
type missingValue string

/*
Initialized to an empty string cast to missingValue.
*/
var MISSING_VALUE Value = missingValue("")

/*
Returns variable MISING_VALUE.
*/
func NewMissingValue() Value {
	return MISSING_VALUE
}

/*
NOTE: This differs from the JSON marshalling of MISSING.
*/
func (this missingValue) String() string {
	return "missing"
}

func (this missingValue) ToString() string {
	return this.String()
}

/*
MISSING is marshalled as NULL in JSON arrays.
*/
func (this missingValue) MarshalJSON() ([]byte, error) {
	return _NULL_BYTES, nil
}

func (this missingValue) WriteXML(order []string, w io.Writer, prefix string, indent string, fast bool) error {
	var err error
	if prefix != "" {
		_, err = w.Write([]byte(getFullPrefix(prefix, "")))
		if err != nil {
			return err
		}
	}
	_, err = w.Write(_NULL_XML)
	return err
}

func (this missingValue) WriteJSON(order []string, w io.Writer, prefix, indent string, fast bool) error {
	_, err := w.Write(_NULL_BYTES)
	return err
}

func (this missingValue) WriteSpill(w io.Writer, buf []byte) error {
	b := []byte{_SPILL_TYPE_VALUE_MISSING}
	_, err := w.Write(b)
	return err
}

func (this missingValue) ReadSpill(r io.Reader, buf []byte) error {
	return nil
}

/*
Description of which property or index was undefined (if known).
If not known, return a message stating Missing field or index.
*/
func (this missingValue) Error() string {
	if string(this) != "" {
		return fmt.Sprintf("Missing field or index %s.", string(this))
	} else {
		return "Missing field or index."
	}
}

/*
Type MISSING
*/
func (this missingValue) Type() Type {
	return MISSING
}

/*
Returns nil.
*/
func (this missingValue) Actual() interface{} {
	return nil
}

func (this missingValue) ActualForIndex() interface{} {
	return nil
}

/*
Returns MISSING.
*/
func (this missingValue) Equals(other Value) Value {
	return this
}

func (this missingValue) EquivalentTo(other Value) bool {
	return other.Type() == MISSING
}

/*
Returns an integer representing the position of Missing with
respect to the other values type by subtracting them and
casting the result to an integer.
*/
func (this missingValue) Collate(other Value) int {
	return int(MISSING - other.Type())
}

func (this missingValue) Compare(other Value) Value {
	return this
}

/*
As per the N1ql specs the truth-value of a missing evaluates
to a false, and hence the Truth method returns a false.
*/
func (this missingValue) Truth() bool {
	return false
}

/*
Return receiver this.
*/
func (this missingValue) Copy() Value {
	return this
}

/*
Return receiver.
*/
func (this missingValue) CopyForUpdate() Value {
	return this
}

/*
Calls missingField.
*/
func (this missingValue) Field(field string) (Value, bool) {
	return missingField(field), false
}

/*
Invalid for missing.
*/
func (this missingValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

/*
Invalid for missing.
*/
func (this missingValue) UnsetField(field string) error {
	return Unsettable(field)
}

/*
Calls missingIndex.
*/
func (this missingValue) Index(index int) (Value, bool) {
	return missingIndex(index), false
}

/*
Invalid for missing.
*/
func (this missingValue) SetIndex(index int, val interface{}) error {
	return Unsettable(strconv.Itoa(index))
}

/*
Returns MISSING_VALUE.
*/
func (this missingValue) Slice(start, end int) (Value, bool) {
	return MISSING_VALUE, false
}

/*
Returns MISSING_VALUE.
*/
func (this missingValue) SliceTail(start int) (Value, bool) {
	return MISSING_VALUE, false
}

/*
Returns MISSING_VALUE.
*/
func (this missingValue) Append(elems []interface{}) (Value, bool) {
	return MISSING_VALUE, false
}

/*
Returns the input buffer as is.
*/
func (this missingValue) Descendants(buffer []interface{}) []interface{} {
	return buffer
}

/*
MISSING has no fields to list. Hence return nil.
*/
func (this missingValue) Fields() map[string]interface{} {
	return nil
}

func (this missingValue) FieldNames(buffer []string) []string {
	return nil
}

/*
Returns the input buffer as is.
*/
func (this missingValue) DescendantPairs(buffer []util.IPair) []util.IPair {
	return buffer
}

/*
MISSING is succeeded by NULL.
*/
func (this missingValue) Successor() Value {
	return NULL_VALUE
}

func (this missingValue) Track() {
}

func (this missingValue) Recycle() {
}

func (this missingValue) Tokens(set *Set, options Value) *Set {
	return set
}

func (this missingValue) ContainsToken(token, options Value) bool {
	return false
}

func (this missingValue) ContainsMatchingToken(matcher MatchFunc, options Value) bool {
	return false
}

func (this missingValue) unwrap() Value {
	return this
}

func (this missingValue) Size() uint64 {
	return uint64(0)
}

/*
Cast input field to missingValue.
*/
func missingField(field string) missingValue {
	return missingValue(field)
}

/*
Cast input index to missingValue after casting to string.
*/
func missingIndex(index int) missingValue {
	return missingValue(strconv.Itoa(index))
}
