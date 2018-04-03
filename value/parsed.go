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
	"io"
	"strconv"

	atomic "github.com/couchbase/go-couchbase/platform"
	json "github.com/couchbase/go_json"
	"github.com/couchbase/query/util"
)

// we try to find a balance between the need to have a find state
// and the cost of using it: for documents shorter than this, not
// worth it!
const _THRESHOLD = 2560

/*
A Value with delayed parsing.
*/
type parsedValue struct {
	raw        []byte
	parsedType Type
	parsed     Value
	useState   bool
	state      *json.FindState
	used       int32
}

func NewParsedValue(bytes []byte, isValidated bool) Value {
	return NewParsedValueWithOptions(bytes, isValidated, len(bytes) > _THRESHOLD)
}

func NewParsedValueWithOptions(bytes []byte, isValidated, useState bool) Value {
	parsedType := identifyType(bytes)

	// Atomic types
	switch parsedType {
	case NUMBER, STRING, BOOLEAN, NULL:
		var p interface{}
		err := json.Unmarshal(bytes, &p)
		if err != nil {
			return binaryValue(bytes)
		}

		return NewValue(p)
	case BINARY:
		return binaryValue(bytes)
	}

	// Container types

	// skip validation if already done elsewhere
	if !isValidated && json.Validate(bytes) != nil {
		return binaryValue(bytes)
	}

	return &parsedValue{
		raw:        bytes,
		parsedType: parsedType,
		useState:   useState,
	}
}

/*
Used to return the type of input bytes. It ranges over bytes,
and classifies it into an object (if '{' is seen), array ('['),
string ('"'), number (for any digit and '-'), boolean ('t/f'),
and null ('n'). If a whitespace is encountered, look at the
next byte. If none of these types fit then it has to be binary.
*/
func identifyType(bytes []byte) Type {
	for _, b := range bytes {
		switch b {
		case '{':
			return OBJECT
		case '[':
			return ARRAY
		case '"':
			return STRING
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
			return NUMBER
		case 't', 'f':
			return BOOLEAN
		case 'n':
			return NULL
		case ' ', '\t', '\n':
			continue
		}
		break
	}

	return BINARY
}
func (this *parsedValue) String() string {
	return this.unwrap().String()
}

func (this *parsedValue) MarshalJSON() ([]byte, error) {
	return this.unwrap().MarshalJSON()
}

func (this *parsedValue) WriteJSON(w io.Writer, prefix, indent string) error {
	raw := this.raw
	if prefix != "" || indent != "" || raw == nil {
		return this.unwrap().WriteJSON(w, prefix, indent)
	}
	_, err := w.Write(raw)
	return err
}

func (this *parsedValue) Type() Type {
	return this.parsedType
}

func (this *parsedValue) Actual() interface{} {
	return this.unwrap().Actual()
}

func (this *parsedValue) ActualForIndex() interface{} {
	return this.unwrap().ActualForIndex()
}

func (this *parsedValue) Equals(other Value) Value {
	return this.unwrap().Equals(other)
}

func (this *parsedValue) EquivalentTo(other Value) bool {
	return this.unwrap().EquivalentTo(other)
}

func (this *parsedValue) Collate(other Value) int {
	return this.unwrap().Collate(other)
}

func (this *parsedValue) Compare(other Value) Value {
	return this.unwrap().Compare(other)
}

func (this *parsedValue) Truth() bool {
	return this.unwrap().Truth()
}

func (this *parsedValue) Copy() Value {
	return this.unwrap().Copy()
}

func (this *parsedValue) CopyForUpdate() Value {
	return this.unwrap().CopyForUpdate()
}

/*
Delayed parsing
*/
func (this *parsedValue) Field(field string) (Value, bool) {
	if this.parsed != nil {
		return this.parsed.Field(field)
	}

	if this.parsedType != OBJECT {
		return missingField(field), false
	}

	raw := this.raw
	if raw != nil {
		var res []byte
		var err error

		goahead := int32(0)
		if this.useState {
			goahead = atomic.AddInt32(&this.used, 1)
			defer atomic.AddInt32(&this.used, -1)
		}

		// Two operators can use the same value at the same time
		// this is particularly the case for unnest, which scans
		// an object looking for array elements.
		// Since the state is, well, statefull, we'll only let the
		// first served modify it, while the other will have to go
		// the slow route
		// For small documents manipulating the state is constly,
		// so we do a scan anyway
		if this.useState && goahead == 1 {
			if this.state == nil {
				this.state = json.NewFindState(this.raw)
			}
			res, err = json.FirstFindWithState(this.state, field)
		} else {
			res, err = json.FirstFind(raw, field)
		}
		if err != nil {
			return missingField(field), false
		}
		if res != nil {

			// since this field was part of a validated value,
			// we don't need to validate it again
			return NewParsedValue(res, true), true
		}
	}

	return missingField(field), false
}

/*
Return Unsettable if parsedType is not OBJECT. If it is then parse
the receiver and call the values corresponding SetField.
*/
func (this *parsedValue) SetField(field string, val interface{}) error {
	if this.parsedType != OBJECT {
		return Unsettable(field)
	}

	return this.unwrap().SetField(field, val)
}

/*
Return Unsettable if parsedType is not OBJECT. If it is then parse
the receiver and call the values corresponding UnsetField.
*/
func (this *parsedValue) UnsetField(field string) error {
	if this.parsedType != OBJECT {
		return Unsettable(field)
	}

	return this.unwrap().UnsetField(field)
}

/*
Delayed parsing
*/
func (this *parsedValue) Index(index int) (Value, bool) {
	if this.parsed != nil {
		return this.parsed.Index(index)
	}

	if this.parsedType != ARRAY {
		return missingIndex(index), false
	}

	if index < 0 {
		return this.unwrap().Index(index)
	}

	raw := this.raw
	if raw != nil {
		res, err := json.Find(raw, "/"+strconv.Itoa(index))
		if err != nil {
			return missingIndex(index), false
		}
		if res != nil {

			// since this array element  was part of a validated value,
			// we don't need to validate it again
			return NewParsedValue(res, true), true
		}
	}

	return missingIndex(index), false
}

/*
Return Unsettable if parsedType is not ARRAY. If it is then parse
the receiver and call the values corresponding SetIndex with the
index and value as input arguments.
*/
func (this *parsedValue) SetIndex(index int, val interface{}) error {
	if this.parsedType != ARRAY {
		return Unsettable(index)
	}

	return this.unwrap().SetIndex(index, val)
}

/*
Return NULL_VALUE if parsedType is not ARRAY. If it is then parse
the receiver and call the values corresponding Slice with the indices
as input arguments.
*/
func (this *parsedValue) Slice(start, end int) (Value, bool) {
	if this.parsedType != ARRAY {
		return NULL_VALUE, false
	}

	return this.unwrap().Slice(start, end)
}

/*
Return NULL_VALUE if parsedType is not ARRAY. If it is then parse
the receiver and call the values corresponding SliceTail with the
start index as input arguments.
*/
func (this *parsedValue) SliceTail(start int) (Value, bool) {
	if this.parsedType != ARRAY {
		return NULL_VALUE, false
	}

	return this.unwrap().SliceTail(start)
}

/*
Return the buffer if the parsedType is binary. If not call parse and
then the Descendants method on that value with the input buffer.
*/
func (this *parsedValue) Descendants(buffer []interface{}) []interface{} {
	if this.parsedType == BINARY {
		return buffer
	}

	return this.unwrap().Descendants(buffer)
}

func (this *parsedValue) Fields() map[string]interface{} {
	return this.unwrap().Fields()
}

func (this *parsedValue) FieldNames(buffer []string) []string {
	return this.unwrap().FieldNames(buffer)
}

/*
Return the buffer if the parsedType is binary. If not call parse and
then the DescendantPairs method on that value with the input buffer.
*/
func (this *parsedValue) DescendantPairs(buffer []util.IPair) []util.IPair {
	if this.parsedType == BINARY {
		return buffer
	}

	return this.unwrap().DescendantPairs(buffer)
}

func (this *parsedValue) Successor() Value {
	return this.unwrap().Successor()
}

func (this *parsedValue) Recycle() {
	if this.parsed != nil {
		this.parsed.Recycle()
	}
}

func (this *parsedValue) Tokens(set *Set, options Value) *Set {
	return this.unwrap().Tokens(set, options)
}

func (this *parsedValue) ContainsToken(token, options Value) bool {
	return this.unwrap().ContainsToken(token, options)
}

func (this *parsedValue) ContainsMatchingToken(matcher MatchFunc, options Value) bool {
	return this.unwrap().ContainsMatchingToken(matcher, options)
}

/*
Delayed parse.
*/
func (this *parsedValue) unwrap() Value {
	if this.parsed == nil {
		if this.parsedType == BINARY {
			this.parsed = binaryValue(this.raw)
		} else {
			var p interface{}

			err := json.UnmarshalNoValidate(this.raw, &p)
			if err != nil {
				this.parsedType = BINARY
				this.parsed = binaryValue(this.raw)
			} else {
				this.parsed = NewValue(p)
			}
		}

		// Release raw memory when no longer needed
		this.raw = nil
		this.state = nil
	}

	return this.parsed
}
