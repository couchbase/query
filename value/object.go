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
	"encoding/xml"
	"io"
	"slices"
	"strconv"

	json "github.com/couchbase/go_json"
	"github.com/couchbase/query/util"
)

/*
objectValue is a type of map from string to interface.
*/
type objectValue map[string]any

var EMPTY_OBJECT_VALUE = objectValue(map[string]any{})

func (this objectValue) String() string {
	return marshalString(this)
}

func (this objectValue) ToString() string {
	return marshalString(this)
}

func (this objectValue) MarshalJSON() ([]byte, error) {
	if this == nil {
		return _NULL_BYTES, nil
	}

	if len(this) == 0 {
		return _EMPTY_OBJECT_BYTES, nil
	}

	buf := bytes.NewBuffer(make([]byte, 0, len(this)*_MARSHAL_FIELD_SIZE+2))
	if err := this.marshalToBuffer(buf); err != nil {
		return nil, err
	}
	return trimMarshalled(buf.Bytes()), nil
}

/*
Copy b to an exact-size slice when it carries a lot of spare capacity, so
that callers that keep the bytes don't keep the slack too.
*/
func trimMarshalled(b []byte) []byte {
	if cap(b) > 2*len(b) && cap(b)-len(b) > _MARSHAL_TRIM_SLACK {
		return append([]byte(nil), b...)
	}
	return b
}

// spare capacity worth an extra copy to release
const _MARSHAL_TRIM_SLACK = 1024

// shared, like _NULL_BYTES (see sharedBytes)
var _EMPTY_OBJECT_BYTES = sharedBytes("{}")

// estimated encoded size of a field, used to size the MarshalJSON buffer
const _MARSHAL_FIELD_SIZE = 32

/*
Append the MarshalJSON() encoding of the object to buf. Nested values are
written into the same buffer rather than marshalled into slices of their own.
*/
func (this objectValue) marshalToBuffer(buf *bytes.Buffer) error {
	if this == nil {
		buf.Write(_NULL_BYTES)
		return nil
	}

	var nameBuf [_NAME_CAP]string
	names, pooled := sortedNamesIn(this, &nameBuf)
	if pooled != nil {
		defer _NAME_POOL.Put(pooled)
	}

	buf.WriteByte('{')
	subsequent := false
	for _, n := range names {
		mark := buf.Len()
		if subsequent {
			buf.WriteByte(',')
		}

		err := json.MarshalStringToBuffer(n, buf)
		if err != nil {
			return err
		}
		buf.WriteByte(':')

		written, err := marshalToBuffer(this[n], buf, true)
		if err != nil {
			return err
		}
		if !written {
			// MISSING field: drop the name
			buf.Truncate(mark)
			continue
		}
		subsequent = true
	}
	buf.WriteByte('}')
	return nil
}

func (this objectValue) WriteXML(order []string, w io.Writer, prefix string, indent string, fast bool) error {
	var err error

	if this == nil {
		_, err = w.Write(_NULL_XML)
		return err
	}

	stringWriter := w.(*bytes.Buffer)

	var fullPrefix string
	var fullValuePrefix string
	writePrefix := (prefix != "" || indent != "")
	if writePrefix {
		fullPrefix = getFullPrefix(prefix, indent)
		fullValuePrefix = getFullPrefix(fullPrefix[1:], indent)
	}

	if writePrefix {
		if _, err = stringWriter.WriteString(fullPrefix[:len(prefix)+1]); err != nil {
			return err
		}
	}

	l := len(this)
	if l == 0 {
		_, err = stringWriter.WriteString("<object/>")
		return err
	}

	if order == nil {
		var nameBuf [_NAME_CAP]string
		var pooled []string
		order, pooled = sortedNamesIn(this, &nameBuf)
		if pooled != nil {
			defer _NAME_POOL.Put(pooled)
		}
	}

	if _, err = stringWriter.WriteString("<object>"); err != nil {
		return err
	}

	for _, n := range order {
		thisv, found := this[n]
		if !found {
			continue
		}
		v := NewValue(thisv)
		if skipJSONField(v) {
			continue
		}

		if writePrefix {
			if _, err = stringWriter.WriteString(fullPrefix); err != nil {
				return err
			}
		}
		if _, err = stringWriter.WriteString("<field name=\""); err != nil {
			return err
		}
		if err = xml.EscapeText(stringWriter, []byte(n)); err != nil {
			return err
		}
		if _, err = stringWriter.WriteString("\">"); err != nil {
			return err
		}

		if writePrefix {
			if err = v.WriteXML(nil, w, fullValuePrefix[1:], indent, false); err != nil {
				return err
			}
		} else {
			if err = v.WriteXML(nil, w, "", "", fast); err != nil {
				return err
			}
		}

		if writePrefix {
			if _, err = stringWriter.WriteString(fullPrefix); err != nil {
				return err
			}
		}
		if _, err = stringWriter.WriteString("</field>"); err != nil {
			return err
		}
	}
	if writePrefix {
		if _, err = stringWriter.WriteString(fullPrefix[:len(prefix)+1]); err != nil {
			return err
		}
	}
	_, err = stringWriter.WriteString("</object>")
	return err
}

func (this objectValue) WriteJSON(order []string, w io.Writer, prefix, indent string, fast bool) (err error) {
	if this == nil {
		_, err = w.Write(_NULL_BYTES)
		return
	}

	// TODO workaround for GSI using an old golang that doesn't know about StringWriter
	stringWriter := w.(*bytes.Buffer)

	if _, err = stringWriter.WriteString("{"); err != nil {
		return err
	}

	var fullPrefix string
	writePrefix := (prefix != "" && indent != "")
	if writePrefix {
		fullPrefix = getFullPrefix(prefix, indent)
	}

	l := len(this)
	written := 0

	// handle scoped KV documents without sorts and marshals
	if l == 1 && fast {

		// unluckily there's no direct way to get the only entry out of a map
		// so we still need a range
		for n, _ := range this {
			v := NewValue(this[n])
			if skipJSONField(v) {
				continue
			}
			written++
			if writePrefix {
				if _, err = stringWriter.WriteString(fullPrefix); err != nil {
					return err
				}
			}
			if err = json.MarshalStringNoEscapeToBuffer(string(n), w.(*bytes.Buffer)); err != nil {
				return err
			}
			if _, err = stringWriter.WriteString(":"); err != nil {
				return err
			}
			if writePrefix {
				if _, err = stringWriter.WriteString(" "); err != nil {
					return err
				}
				if err = v.WriteJSON(nil, w, fullPrefix[1:], indent, fast); err != nil {
					return err
				}
			} else {
				if err = v.WriteJSON(nil, w, "", "", fast); err != nil {
					return err
				}
			}
		}
	} else if l > 0 {

		if order == nil {
			var nameBuf [_NAME_CAP]string
			var pooled []string
			order, pooled = sortedNamesIn(this, &nameBuf)
			if pooled != nil {
				defer _NAME_POOL.Put(pooled)
			}
		}

		for _, n := range order {
			thisv, found := this[n]
			if !found {
				continue
			}
			v := NewValue(thisv)
			if skipJSONField(v) {
				continue
			}

			if written > 0 {
				if _, err = stringWriter.WriteString(","); err != nil {
					return err
				}
			}
			written++

			if writePrefix {
				if _, err = stringWriter.WriteString(fullPrefix); err != nil {
					return err
				}
			}

			if err = json.MarshalStringToBuffer(n, stringWriter); err != nil {
				return err
			}
			if _, err = stringWriter.WriteString(":"); err != nil {
				return err
			}
			if prefix != "" || indent != "" {
				if _, err = stringWriter.WriteString(" "); err != nil {
					return err
				}
			}

			if writePrefix {
				if err = v.WriteJSON(nil, w, fullPrefix[1:], indent, false); err != nil {
					return err
				}
			} else {
				if err = v.WriteJSON(nil, w, "", "", fast); err != nil {
					return err
				}
			}
		}
	}
	if written > 0 && prefix != "" {
		if _, err = stringWriter.WriteString(fullPrefix[:len(prefix)+1]); err != nil {
			return err
		}
	}
	_, err = stringWriter.WriteString("}")
	return err
}

func (this objectValue) WriteSpill(w io.Writer, buf []byte) error {
	b := []byte{_SPILL_TYPE_VALUE}
	_, err := w.Write([]byte(b))
	if err == nil {
		err = writeSpillValue(w, (map[string]any)(this), buf)
	}
	return err
}

func (this objectValue) ReadSpill(trackMem func(int64) error, r io.Reader, buf []byte) error {
	v, err := readSpillValue(trackMem, r, buf)
	if err == nil && v != nil {
		this = objectValue(v.(map[string]any))
	} else {
		this = nil
	}
	return err
}

/*
Type OBJECT.
*/
func (this objectValue) Type() Type {
	return OBJECT
}

func (this objectValue) Actual() any {
	return (map[string]any)(this)
}

func (this objectValue) ActualForIndex() any {
	return (map[string]any)(this)
}

func (this objectValue) Equals(other Value) Value {
	other = unwrapValue(other)
	switch other := other.(type) {
	case missingValue:
		return other
	case *nullValue:
		return other
	case objectValue:
		return objectEquals(this, other)
	case copiedObjectValue:
		return objectEquals(this, other.objectValue)
	case *trackedObjectValue:
		return objectEquals(this, other.objectValue)
	default:
		return FALSE_VALUE
	}
}

func (this objectValue) EquivalentTo(other Value) bool {
	other = unwrapValue(other)
	switch other := other.(type) {
	case objectValue:
		return objectEquivalent(this, other)
	case copiedObjectValue:
		return objectEquivalent(this, other.objectValue)
	case *trackedObjectValue:
		return objectEquivalent(this, other.objectValue)
	default:
		return false
	}
}

func (this objectValue) Collate(other Value) int {
	other = unwrapValue(other)
	switch other := other.(type) {
	case objectValue:
		return objectCollate(this, other)
	case copiedObjectValue:
		return objectCollate(this, other.objectValue)
	case *trackedObjectValue:
		return objectCollate(this, other.objectValue)
	default:
		return int(OBJECT - other.Type())
	}
}

func (this objectValue) Compare(other Value) Value {
	other = unwrapValue(other)
	switch other := other.(type) {
	case missingValue:
		return other
	case *nullValue:
		return other
	case objectValue:
		return objectCompare(this, other)
	case copiedObjectValue:
		return objectCompare(this, other.objectValue)
	case *trackedObjectValue:
		return objectCompare(this, other.objectValue)
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
	return copiedObjectValue{objectValue: objectValue(copyMap(this, self))}
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

func (this objectValue) SetField(field string, val any) error {
	return this.setField(field, val, false)
}

func (this objectValue) setField(field string, val any, track bool) error {
	switch val := val.(type) {
	case missingValue:
		curVal, ok := this[field]
		if ok {
			delete(this, field)
			if track {
				v, ok := curVal.(Value)
				if ok {
					v.Recycle()
				}
			}
		}
	default:
		this[field] = val
		if track {
			v, ok := val.(Value)
			if ok {
				v.Track()
			}
		}
	}

	return nil
}

func (this objectValue) UnsetField(field string) error {
	return this.unsetField(field, false)
}

func (this objectValue) unsetField(field string, track bool) error {
	curVal, ok := this[field]
	if ok {
		delete(this, field)
		if track {
			v, ok := curVal.(Value)
			if ok {
				v.Recycle()
			}
		}
	}
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
func (this objectValue) SetIndex(index int, val any) error {
	return Unsettable(strconv.Itoa(index))
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

/*
Returns NULL_VALUE.
*/
func (this objectValue) Append(elems []any) (Value, bool) {
	return NULL_VALUE, false
}

func (this objectValue) Descendants(buffer []any) []any {
	var nameBuf [_NAME_CAP]string
	names, pooled := sortedNamesIn(this, &nameBuf)
	if pooled != nil {
		defer _NAME_POOL.Put(pooled)
	}

	if cap(buffer) < len(buffer)+len(this) {
		buf2 := make([]any, len(buffer), (len(buffer)+len(this)+1)<<1)
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

func (this objectValue) Fields() map[string]any {
	return this
}

func (this objectValue) FieldNames(buffer []string) []string {
	if cap(buffer) < len(this) {
		buffer = make([]string, len(this))
	} else {
		buffer = buffer[0:len(this)]
	}

	return sortedNames(this, buffer)
}

func (this objectValue) DescendantPairs(buffer []util.IPair) []util.IPair {
	var nameBuf [_NAME_CAP]string
	names, pooled := sortedNamesIn(this, &nameBuf)
	if pooled != nil {
		defer _NAME_POOL.Put(pooled)
	}

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

	var nameBuf [_NAME_CAP]string
	names, pooled := sortedNamesIn(this, &nameBuf)
	if pooled != nil {
		defer _NAME_POOL.Put(pooled)
	}

	n := names[len(names)-1]
	s[n] = NewValue(this[n]).Successor()
	return objectValue(s)
}

func (this objectValue) Track() {
}

func (this objectValue) Recycle() {
}

func (this objectValue) Tokens(set *Set, options Value) *Set {
	names := true
	if n, ok := options.Field("names"); ok && n.Type() == BOOLEAN {
		names = n.Truth()
	}

	for n, v := range this {
		if names {
			set = NewValue(n).Tokens(set, options)
		}

		set = NewValue(v).Tokens(set, options)
	}

	return set
}

func (this objectValue) ContainsToken(token, options Value) bool {
	names := token.Type() == STRING
	if names {
		if n, ok := options.Field("names"); ok && n.Type() == BOOLEAN {
			names = n.Truth()
		}
	}

	for n, v := range this {
		if names && NewValue(n).ContainsToken(token, options) {
			return true
		}

		if NewValue(v).ContainsToken(token, options) {
			return true
		}
	}

	return false
}

func (this objectValue) ContainsMatchingToken(matcher MatchFunc, options Value) bool {
	names := true
	if n, ok := options.Field("names"); ok && n.Type() == BOOLEAN {
		names = n.Truth()
	}

	for n, v := range this {
		if names && NewValue(n).ContainsMatchingToken(matcher, options) {
			return true
		}

		if NewValue(v).ContainsMatchingToken(matcher, options) {
			return true
		}
	}

	return false
}

func (this objectValue) Size() uint64 {
	return AnySize(map[string]any(this))
}

func (this objectValue) unwrap() Value {
	return this
}

var _SMALL_OBJECT_VALUE = objectValue(map[string]any{"": nil})

/*
Object fields that are MISSING or a self reference are left out of the JSON,
as in MarshalJSON().
*/
func skipJSONField(v Value) bool {
	if v.Type() == MISSING {
		return true
	}
	_, ok := v.(*annotatedValueSelfReference)
	return ok
}

// trackedObjectValue is just like objectValue, except it does extra Track()/Recycle() for its subelements
type trackedObjectValue struct {
	objectValue
}

func newTrackedObjectValue(obj map[string]any) *trackedObjectValue {
	return &trackedObjectValue{objectValue: objectValue(obj)}
}

func (this *trackedObjectValue) CopyForUpdate() Value {
	return &trackedObjectValue{objectValue: objectValue(copyMap(this.objectValue, copyForUpdate))}
}

func (this *trackedObjectValue) SetField(field string, val any) error {
	return this.objectValue.setField(field, val, true)
}

func (this *trackedObjectValue) UnsetField(field string) error {
	return this.objectValue.unsetField(field, true)
}

func (this *trackedObjectValue) Track() {
	for _, v := range this.objectValue {
		if val, ok := v.(Value); ok {
			val.Track()
		}
	}
}

func (this *trackedObjectValue) Recycle() {
	for _, v := range this.objectValue {
		if val, ok := v.(Value); ok {
			val.Recycle()
		}
	}
}

func (this *trackedObjectValue) WriteSpill(w io.Writer, buf []byte) error {
	b := []byte{_SPILL_TYPE_VALUE_TRACKED_OBJECT}
	_, err := w.Write([]byte(b))
	if err == nil {
		err = writeSpillValue(w, (map[string]any)(this.objectValue), buf)
	}
	return err
}

func (this *trackedObjectValue) ReadSpill(trackMem func(int64) error, r io.Reader, buf []byte) error {
	v, err := readSpillValue(trackMem, r, buf)
	if err == nil && v != nil {
		this.objectValue = objectValue(v.(map[string]any))
	} else {
		this.objectValue = nil
	}
	return err
}

func objectEquals(obj1, obj2 map[string]any) Value {
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

func objectEquivalent(obj1, obj2 map[string]any) bool {
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
func objectCollate(obj1, obj2 map[string]any) int {
	// first see if one object is longer than the other
	delta := len(obj1) - len(obj2)
	if delta != 0 {
		return delta
	}

	// if not, proceed to do name by name comparision
	cmp := 0
	if diff := objectWalk(obj1, obj2, func(val1, val2 any) bool {
		cmp = NewValue(val1).Collate(NewValue(val2))
		return cmp == 0
	}); diff != 0 {
		return diff
	}
	return cmp
}

func objectCompare(obj1, obj2 map[string]any) Value {
	// first see if one object is longer than the other
	delta := len(obj1) - len(obj2)
	if delta != 0 {
		return intValue(delta)
	}

	// if not, proceed to do name by name comparision
	cmp := ZERO_VALUE
	switch objectWalk(obj1, obj2, func(val1, val2 any) bool {
		c := NewValue(val1).Compare(NewValue(val2))
		// built-in types return intValue; anything else uses Equals
		if i, ok := c.(intValue); ok {
			if i == 0 {
				return true
			}
		} else if c.Equals(ZERO_VALUE).Truth() {
			return true
		}
		// only a difference is returned; equal objects return ZERO_VALUE
		cmp = c
		return false
	}) {
	case -1:
		return NEG_ONE_VALUE
	case 1:
		return ONE_VALUE
	}
	return cmp
}

/*
Walk the names of two objects in sorted order, calling equal with the values
of each name, until it returns false. An object with fewer names is smaller.
Otherwise the first name that is only in one object ends the walk: the result
is -1 if obj2 does not have it, 1 if obj1 does not, and 0 otherwise.
*/
func objectWalk(obj1, obj2 map[string]any, equal func(val1, val2 any) bool) int {
	// the current callers check the lengths first, but the walk below
	// depends on them being equal
	if len(obj1) < len(obj2) {
		return -1
	} else if len(obj1) > len(obj2) {
		return 1
	}

	var nameBuf1, nameBuf2 [_NAME_CAP]string
	names1, pooled1 := sortedNamesIn(obj1, &nameBuf1)
	if pooled1 != nil {
		defer _NAME_POOL.Put(pooled1)
	}
	names2, pooled2 := sortedNamesIn(obj2, &nameBuf2)
	if pooled2 != nil {
		defer _NAME_POOL.Put(pooled2)
	}

	// as the objects are the same length, the first name that is only in one
	// object sorts before the other object's name at the same position
	for i, name := range names1 {
		if name < names2[i] {
			// obj2 did not have this name, so it is larger
			return -1
		} else if name > names2[i] {
			// obj1 did not have this name, so it is larger
			return 1
		}

		// name was in both objects, so compare the corresponding values
		if !equal(obj1[name], obj2[name]) {
			break
		}
	}
	return 0
}

func copyMap(source map[string]any, copier copyFunc) map[string]any {
	if source == nil {
		return nil
	}

	result := make(map[string]any, len(source))
	for n, v := range source {
		result[n] = copier(v)
	}

	return result
}

/*
Return the names of obj in sorted order: in nameBuf if they fit, otherwise in
a slice from the pool, which is also returned for the caller to put back.
*/
func sortedNamesIn(obj map[string]any, nameBuf *[_NAME_CAP]string) (names, pooled []string) {
	if len(obj) <= len(nameBuf) {
		names = nameBuf[0:len(obj)]
	} else {
		// full length, so that Put clears the names
		pooled = _NAME_POOL.GetSized(len(obj))
		names = pooled
	}
	return sortedNames(obj, names), pooled
}

func sortedNames(obj map[string]any, buffer []string) []string {
	i := 0
	for name, _ := range obj {
		buffer[i] = name
		i++
	}

	slices.Sort(buffer)
	return buffer
}

const _NAME_CAP = 16

var _NAME_POOL = util.NewStringPool(256)
