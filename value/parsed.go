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
	"regexp"
	"strconv"
	"strings"
	"sync"

	atomic "github.com/couchbase/go-couchbase/platform"
	json "github.com/couchbase/go_json"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
)

// we try to find a balance between the need to have a find state
// and the cost of using it: for documents shorter than this, not
// worth it! (unless we'll be extracting only a few fields)
const PARSED_THRESHOLD = 2560
const _NUM_PARSED_FIELDS = 32

// A Value with delayed parsing.
type parsedValue struct {
	raw          []byte
	len          uint64
	parsedType   Type
	parsed       Value
	sync.RWMutex // to access fields
	fields       map[string]Value
	elements     map[int]Value
	useState     bool
	keyState     json.KeyState
	indexState   json.IndexState
	cleanupState bool  // state was in use when value was unwrapped, so indicate to clean-up when done
	refCnt       int32 // to check for recycling
	used         int32 // to access state
}

var parsedPool util.FastPool

func init() {
	util.NewFastPool(&parsedPool, func() interface{} {
		return &parsedValue{}
	})
}

func newParsedValue() *parsedValue {
	rv := parsedPool.Get().(*parsedValue)
	*rv = parsedValue{}
	rv.refCnt = 1
	return rv
}

func NewParsedValue(bytes []byte, isValidated bool) Value {
	return NewParsedValueWithOptions(bytes, isValidated, len(bytes) > PARSED_THRESHOLD)
}

func NewParsedValueWithOptions(bytes []byte, isValidated, useState bool) Value {
	parsedType := identifyType(bytes)

	// Atomic types
	switch parsedType {
	case NUMBER, STRING, BOOLEAN, NULL:

		// for scalar values we can skip validation, as the simple unmarshaler will validate while scanning
		p, err := json.SimpleUnmarshal(bytes)
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

	rv := newParsedValue()
	rv.raw = bytes
	rv.len = uint64(len(bytes))
	rv.parsedType = parsedType
	rv.useState = useState
	return rv
}

func ToJSON(v Value) []byte {
	val, ok := v.(*parsedValue)
	if !ok || val.raw == nil {
		return nil
	}
	return val.raw
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
		case 't', 'f', 'T', 'F':
			return BOOLEAN
		case 'n', 'N':
			return NULL
		case ' ', '\t', '\n', '\r':
			continue
		default:
			return BINARY
		}
	}
	return BINARY
}

func (this *parsedValue) String() string {
	return this.unwrap().String()
}

func (this *parsedValue) ToString() string {
	return this.unwrap().String()
}

func (this *parsedValue) MarshalJSON() ([]byte, error) {
	if this.raw != nil {
		return this.raw, nil
	}
	return this.unwrap().MarshalJSON()
}

func (this *parsedValue) WriteXML(order []string, w io.Writer, prefix, indent string, fast bool) error {
	return this.unwrap().WriteXML(order, w, prefix, indent, fast)
}

func (this *parsedValue) WriteJSON(order []string, w io.Writer, prefix, indent string, fast bool) error {
	raw := this.raw
	if raw == nil || order != nil {
		return this.unwrap().WriteJSON(order, w, prefix, indent, fast)
	} else if prefix != "" || indent != "" {
		return json.IndentWriter(w, raw, prefix, indent)
	}
	_, err := w.Write(raw)
	return err
}

func (this *parsedValue) WriteSpill(w io.Writer, buf []byte) error {
	b := []byte{_SPILL_TYPE_VALUE_PARSED}
	_, err := w.Write([]byte(b))
	if err == nil {
		err = writeSpillValue(w, this.raw, buf)
	}
	if err == nil {
		err = writeSpillValue(w, this.len, buf)
	}
	if err == nil {
		err = writeSpillValue(w, int(this.parsedType), buf)
	}
	if err == nil {
		err = writeSpillValue(w, this.parsed, buf)
	}
	if err == nil {
		err = writeSpillValue(w, this.fields, buf)
	}
	if err == nil {
		err = writeSpillValue(w, this.elements, buf)
	}
	if err == nil {
		err = writeSpillValue(w, this.useState, buf)
	}
	if err == nil {
		err = writeSpillValue(w, this.refCnt, buf)
	}
	return err
}

func (this *parsedValue) ReadSpill(r io.Reader, buf []byte) error {
	*this = parsedValue{}
	v, err := readSpillValue(r, buf)
	if err != nil {
		return err
	} else if v != nil {
		this.raw = v.([]byte)
	}

	v, err = readSpillValue(r, buf)
	if err != nil {
		return err
	} else if v != nil {
		this.len = v.(uint64)
	}

	v, err = readSpillValue(r, buf)
	if err != nil {
		return err
	} else if v != nil {
		this.parsedType = Type(v.(int))
	}

	v, err = readSpillValue(r, buf)
	if err != nil {
		return err
	} else if v != nil {
		this.parsed = v.(Value)
	}

	v, err = readSpillValue(r, buf)
	if err != nil {
		return err
	} else if v != nil {
		this.fields = v.(map[string]Value)
	}

	v, err = readSpillValue(r, buf)
	if err != nil {
		return err
	} else if v != nil {
		this.elements = v.(map[int]Value)
	}

	v, err = readSpillValue(r, buf)
	if err != nil {
		return err
	} else if v != nil {
		this.useState = v.(bool)
	}

	v, err = readSpillValue(r, buf)
	if err != nil {
		return err
	} else if v != nil {
		this.refCnt = v.(int32)
	}

	return nil
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

// Delayed parsing
func (this *parsedValue) Field(field string) (Value, bool) {
	if this.parsed != nil {
		return this.parsed.Field(field)
	}

	if this.parsedType != OBJECT {
		return missingField(field), false
	}

	if this.fields != nil {
		this.RLock()
		result, ok := this.fields[field]
		this.RUnlock()
		if ok {
			result.Track()
			return NewValue(result), true
		}
	}

	raw := this.raw
	if raw != nil {
		var res []byte
		var err error

		goahead := int32(0)
		if this.useState {
			goahead = atomic.AddInt32(&this.used, 1)
			defer func() {
				if atomic.AddInt32(&this.used, -1) == 0 && this.cleanupState {
					this.keyState.Release()
					this.indexState.Release()
				}
			}()
		}

		// Two operators can use the same value at the same time this is particularly the case for unnest, which scans
		// an object looking for array elements.  Since the state is, well, statefull, we'll only let the first served modify it,
		// while the other will have to go the slow route.
		// For small documents manipulating the state is costly, so we do a scan anyway
		useState := this.useState && goahead == 1
		if useState {
			json.SetKeyState(&this.keyState, this.raw)
			res, err = this.keyState.FindKey(field)
		} else {
			res, err = json.FindKey(raw, field)
		}
		if err != nil {
			return missingField(field), false
		}
		if res != nil {

			// since this field was part of a validated value,
			// we don't need to validate it again
			val := NewParsedValueWithOptions(res, true, this.useState)

			if useState {
				this.Lock()
				if this.fields == nil {
					this.fields = make(map[string]Value)
				}
				this.fields[field] = val
				this.Unlock()
				val.Track()
			}
			return val, true
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

// Delayed parsing
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

	if this.elements != nil {
		this.RLock()
		result, ok := this.elements[index]
		this.RUnlock()
		if ok {
			result.Track()
			return NewValue(result), true
		}
	}

	raw := this.raw
	if raw != nil {
		var res []byte
		var err error

		goahead := int32(0)
		if this.useState {
			goahead = atomic.AddInt32(&this.used, 1)
			defer func() {
				if atomic.AddInt32(&this.used, -1) == 0 && this.cleanupState {
					this.keyState.Release()
					this.indexState.Release()
				}
			}()
		}

		// Two operators can use the same value at the same time
		// this is particularly the case for unnest, which scans
		// an object looking for array elements.
		// Since the state is, well, statefull, we'll only let the
		// first served modify it, while the other will have to go
		// the slow route
		// For small documents manipulating the state is constly,
		// so we do a scan anyway
		useState := this.useState && goahead == 1
		if useState {
			json.SetIndexState(&this.indexState, this.raw)
			res, err = this.indexState.FindIndex(index)
		} else {
			res, err = json.FindIndex(raw, index)
		}
		if err != nil {
			return missingIndex(index), false
		}
		if res != nil {

			// since this array element was part of a validated value,
			// we don't need to validate it again
			val := NewParsedValueWithOptions(res, true, this.useState)

			if useState {
				this.Lock()
				if this.elements == nil {
					this.elements = make(map[int]Value)
				}
				this.elements[index] = val
				this.Unlock()
				val.Track()

			}
			return val, true
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
		return Unsettable(strconv.Itoa(index))
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

func (this *parsedValue) pfSize() int {
	if this.fields != nil && len(this.fields) > _NUM_PARSED_FIELDS {
		return len(this.fields)
	}
	return _NUM_PARSED_FIELDS
}

func (this *parsedValue) ParsedFields(min, max string, re interface{}) []interface{} {
	raw := this.raw
	var rex *regexp.Regexp

	if re != nil {
		rex, _ = re.(*regexp.Regexp)
	}

	rv := make([]interface{}, 0, this.pfSize())
	if raw != nil {
		var ss json.ScanState
		json.SetScanState(&ss, raw)
		defer ss.Release()
		if re != nil {
			for {
				key, err := ss.ScanKeys()
				if err != nil {
					return nil
				}
				if key == nil {
					break
				}
				if rex.FindStringSubmatchIndex(string(key)) != nil {
					val, err := ss.NextValue()
					if err != nil {
						return nil
					}
					rv = append(rv, map[string]interface{}{"name": string(key), "val": NewParsedValue(val, true)})
				}
			}
		} else if len(min) != 0 || len(max) != 0 {
			for {
				key, err := ss.ScanKeys()
				if err != nil {
					return nil
				}
				if key == nil {
					break
				}
				if (len(min) == 0 || strings.Compare(min, string(key)) <= 0) &&
					(len(max) == 0 || strings.Compare(max, string(key)) == 1) {

					val, err := ss.NextValue()
					if err != nil {
						return nil
					}
					rv = append(rv, map[string]interface{}{"name": string(key), "val": NewParsedValue(val, true)})
				}
			}
		} else {
			for {
				key, err := ss.ScanKeys()
				if err != nil {
					return nil
				}
				if key == nil {
					break
				}
				val, err := ss.NextValue()
				if err != nil {
					return nil
				}
				rv = append(rv, map[string]interface{}{"name": string(key), "val": NewParsedValue(val, true)})
			}
		}
	} else if this.parsed != nil && this.parsed.Type() == OBJECT {
		if re != nil {
			for key, val := range this.parsed.Fields() {
				if rex.FindStringSubmatchIndex(string(key)) != nil {
					rv = append(rv, map[string]interface{}{"name": string(key), "val": val})
				}
			}
		} else if len(min) != 0 || len(max) != 0 {
			for key, val := range this.parsed.Fields() {
				if (len(min) == 0 || strings.Compare(min, string(key)) <= 0) &&
					(len(max) == 0 || strings.Compare(max, string(key)) == 1) {
					rv = append(rv, map[string]interface{}{"name": string(key), "val": val})
				}
			}
		} else {
			for key, val := range this.parsed.Fields() {
				rv = append(rv, map[string]interface{}{"name": string(key), "val": val})
			}
		}
	}
	return rv
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

func (this *parsedValue) Track() {
	atomic.AddInt32(&this.refCnt, 1)
}

func (this *parsedValue) Recycle() {

	// do no recycle if other scope values are using this value
	refcnt := atomic.AddInt32(&this.refCnt, -1)
	if refcnt > 0 {
		return
	}
	if refcnt < 0 {

		// TODO enable
		// panic("parsed value already recycled")
		logging.Infof("parsed value already recycled")
		return
	}
	if this.parsed != nil {
		this.parsed.Recycle()
		this.parsed = nil
	}
	if this.fields != nil {
		for i, field := range this.fields {
			this.fields[i] = nil
			field.Recycle()
		}
		this.fields = nil
	}
	if this.elements != nil {
		for i, element := range this.elements {
			this.elements[i] = nil
			element.Recycle()
		}
		this.elements = nil
	}
	this.raw = nil
	parsedPool.Put(this)
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

func (this *parsedValue) Size() uint64 {
	if this.parsed != nil {
		return this.parsed.Size()
	}
	return this.len
}

// Delayed parse.
func (this *parsedValue) unwrap() Value {
	if this.raw != nil {
		if this.parsedType == BINARY {
			this.parsed = binaryValue(this.raw)
		} else {
			p, err := json.SimpleUnmarshal(this.raw)
			if err != nil {
				this.parsedType = BINARY
				this.parsed = binaryValue(this.raw)
			} else {
				this.parsed = NewValue(p)
			}
		}

		// Release raw memory when no longer needed
		this.raw = nil
		if atomic.AddInt32(&this.used, 1) == 1 {
			this.keyState.Release()
			this.indexState.Release()
		} else {
			this.cleanupState = true
		}
		atomic.AddInt32(&this.used, -1)
		if this.fields != nil || this.elements != nil {
			this.Lock()
			for i, field := range this.fields {
				this.fields[i] = nil
				field.Recycle()
			}
			this.fields = nil
			for i, element := range this.elements {
				this.elements[i] = nil
				element.Recycle()
			}
			this.elements = nil
			this.Unlock()
		}
	}

	return this.parsed
}
