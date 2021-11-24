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
	"fmt"
	"io"
	"strconv"

	"github.com/couchbase/query/util"
)

type binaryValue []byte

func NewBinaryValue(bytes []byte) Value {
	return binaryValue(bytes)
}

func (this binaryValue) String() string {
	return fmt.Sprintf("\"<binary (%d b)>\"", len(this))
}

func (this binaryValue) ToString() string {
	return fmt.Sprintf("\"<binary (%d b)>\"", len(this))
}

func (this binaryValue) MarshalJSON() ([]byte, error) {
	return []byte(this.String()), nil
}

func (this binaryValue) WriteJSON(w io.Writer, prefix, indent string, fast bool) error {
	_, err := w.(*bytes.Buffer).WriteString(this.String())
	return err
}

func (this binaryValue) Type() Type {
	return BINARY
}

func (this binaryValue) Actual() interface{} {
	return []byte(this)
}

func (this binaryValue) ActualForIndex() interface{} {
	return []byte(this)
}

func (this binaryValue) Equals(other Value) Value {
	other = other.unwrap()
	switch other := other.(type) {
	case missingValue:
		return other
	case *nullValue:
		return other
	case binaryValue:
		if bytes.Equal(this, other) {
			return TRUE_VALUE
		}
	}

	return FALSE_VALUE
}

func (this binaryValue) EquivalentTo(other Value) bool {
	other = other.unwrap()
	switch other := other.(type) {
	case binaryValue:
		return bytes.Equal(this, other)
	default:
		return false
	}
}

func (this binaryValue) Collate(other Value) int {
	other = other.unwrap()
	switch other := other.(type) {
	case binaryValue:
		return bytes.Compare(this, other)
	default:
		return int(BINARY - other.Type())
	}
}

func (this binaryValue) Compare(other Value) Value {
	other = other.unwrap()
	switch other := other.(type) {
	case missingValue:
		return other
	case *nullValue:
		return other
	default:
		return intValue(this.Collate(other))
	}
}

func (this binaryValue) Truth() bool {
	return len(this) > 0
}

func (this binaryValue) Copy() Value {
	return this
}

func (this binaryValue) CopyForUpdate() Value {
	return this
}

func (this binaryValue) Field(field string) (Value, bool) {
	return missingField(field), false
}

func (this binaryValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

func (this binaryValue) UnsetField(field string) error {
	return Unsettable(field)
}

func (this binaryValue) Index(index int) (Value, bool) {
	return missingIndex(index), false
}

func (this binaryValue) SetIndex(index int, val interface{}) error {
	return Unsettable(strconv.Itoa(index))
}

func (this binaryValue) Slice(start, end int) (Value, bool) {
	return NULL_VALUE, false
}

func (this binaryValue) SliceTail(start int) (Value, bool) {
	return NULL_VALUE, false
}

func (this binaryValue) Descendants(buffer []interface{}) []interface{} {
	return buffer
}

func (this binaryValue) Fields() map[string]interface{} {
	return nil
}

func (this binaryValue) FieldNames(buffer []string) []string {
	return nil
}

func (this binaryValue) DescendantPairs(buffer []util.IPair) []util.IPair {
	return buffer
}

func (this binaryValue) Successor() Value {
	return binaryValue(append(this, byte(0)))
}

func (this binaryValue) Track() {
}

func (this binaryValue) Recycle() {
}

func (this binaryValue) Tokens(set *Set, options Value) *Set {
	set.Add(this)
	return set
}

func (this binaryValue) ContainsToken(token, options Value) bool {
	return this.EquivalentTo(token)
}

func (this binaryValue) ContainsMatchingToken(matcher MatchFunc, options Value) bool {
	return matcher([]byte(this))
}

func (this binaryValue) Size() uint64 {
	return uint64(len(this))
}

func (this binaryValue) unwrap() Value {
	return this
}
