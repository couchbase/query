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
	"math/bits"
	"sort"
	"strconv"

	json "github.com/couchbase/go_json"
	"github.com/couchbase/query/util"
)

/*
objectValue is a type of map from string to interface.
*/
type objectValue map[string]interface{}

var EMPTY_OBJECT_VALUE = objectValue(map[string]interface{}{})

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

	var nameBuf [_NAME_CAP]string
	var names []string
	if len(this) <= len(nameBuf) {
		names = nameBuf[0:len(this)]
	} else {
		names = _NAME_POOL.GetCapped(len(this))
		defer _NAME_POOL.Put(names)
		names = names[0:len(this)]
	}

	names = sortedNames(this, names)

	if len(names) == 0 {
		return []byte("{}"), nil
	}

	var err error
	sz := 0
	marshalledValues := make([][]byte, len(names))
	for i, n := range names {
		v := NewValue(this[n])
		if v.Type() == MISSING {
			continue
		}
		marshalledValues[i], err = v.MarshalJSON()
		if err != nil {
			return nil, err
		}
		// allow for per-line overhead:
		// 2 - name quotes
		// 1 - comma separator
		// 5 - possible HTML escaping in name (to minimise chances of buffer reallocation)
		sz += len(marshalledValues[i]) + len(n) + 8
	}

	buf := bytes.NewBuffer(make([]byte, 0, sz+2)) // overhead: {}
	buf.WriteRune('{')
	subsequent := false
	for i, n := range names {
		if marshalledValues[i] == nil {
			continue
		}
		if subsequent {
			buf.WriteRune(',')
		} else {
			subsequent = true
		}

		err = json.MarshalStringToBuffer(n, buf)
		if err != nil {
			return nil, err
		}

		buf.WriteRune(':')
		buf.Write(marshalledValues[i])
		marshalledValues[i] = nil
	}
	marshalledValues = nil

	buf.WriteRune('}')
	return buf.Bytes(), nil
}

func (this objectValue) WriteJSON(order []string, w io.Writer, prefix, indent string, fast bool) (err error) {
	if this == nil {
		_, err = w.Write(_NULL_BYTES)
		return
	}

	// TODO workaround for GSI using an old golang that doesn't know about StringWriter
	stringWriter := w.(*bytes.Buffer)

	if _, err = stringWriter.WriteString("{"); err != nil {
		return
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
			if v.Type() == MISSING {
				continue
			}
			written++
			if writePrefix {
				if _, err = stringWriter.WriteString(fullPrefix); err != nil {
					return
				}
			}
			if err = json.MarshalStringNoEscapeToBuffer(string(n), w.(*bytes.Buffer)); err != nil {
				return
			}
			if _, err = stringWriter.WriteString(":"); err != nil {
				return
			}
			if writePrefix {
				if _, err = stringWriter.WriteString(" "); err != nil {
					return err
				}
				if err = v.WriteJSON(nil, w, fullPrefix[1:], indent, fast); err != nil {
					return
				}
			} else {
				if err = v.WriteJSON(nil, w, "", "", fast); err != nil {
					return
				}
			}
		}
	} else if l > 0 {

		if order == nil {
			var remaining []string
			if l <= _NAME_CAP {
				var nameBuf [_NAME_CAP]string
				remaining = nameBuf[:0]
			} else {
				remaining = _NAME_POOL.GetCapped(l)
				defer _NAME_POOL.Put(remaining)
				remaining = remaining[:0]
			}
			remaining = remaining[:l]
			order = sortedNames(this, remaining)
		}

		for _, n := range order {
			thisv, found := this[n]
			if !found {
				continue
			}
			v := NewValue(thisv)
			if v.Type() == MISSING {
				continue
			}

			if written > 0 {
				if _, err = stringWriter.WriteString(","); err != nil {
					return
				}
			}
			written++

			if writePrefix {
				if _, err = stringWriter.WriteString(fullPrefix); err != nil {
					return
				}
			}

			b, err := json.Marshal(n)
			if err != nil {
				return err
			}

			if _, err = w.Write(b); err != nil {
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
		err = writeSpillValue(w, (map[string]interface{})(this), buf)
	}
	return err
}

func (this objectValue) ReadSpill(r io.Reader, buf []byte) error {
	v, err := readSpillValue(r, buf)
	if err == nil && v != nil {
		this = objectValue(v.(map[string]interface{}))
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

func (this objectValue) Actual() interface{} {
	return (map[string]interface{})(this)
}

func (this objectValue) ActualForIndex() interface{} {
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
	case copiedObjectValue:
		return objectEquals(this, other.objectValue)
	default:
		return FALSE_VALUE
	}
}

func (this objectValue) EquivalentTo(other Value) bool {
	other = other.unwrap()
	switch other := other.(type) {
	case objectValue:
		return objectEquivalent(this, other)
	case copiedObjectValue:
		return objectEquivalent(this, other.objectValue)
	default:
		return false
	}
}

func (this objectValue) Collate(other Value) int {
	other = other.unwrap()
	switch other := other.(type) {
	case objectValue:
		return objectCollate(this, other)
	case copiedObjectValue:
		return objectCollate(this, other.objectValue)
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
	case copiedObjectValue:
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

func (this objectValue) Descendants(buffer []interface{}) []interface{} {
	var nameBuf [_NAME_CAP]string
	var names []string
	if len(this) <= len(nameBuf) {
		names = nameBuf[0:len(this)]
	} else {
		names = _NAME_POOL.GetCapped(len(this))
		defer _NAME_POOL.Put(names)
		names = names[0:len(this)]
	}

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
	if cap(buffer) < len(this) {
		buffer = make([]string, len(this))
	} else {
		buffer = buffer[0:len(this)]
	}

	return sortedNames(this, buffer)
}

func (this objectValue) DescendantPairs(buffer []util.IPair) []util.IPair {
	var nameBuf [_NAME_CAP]string
	var names []string
	if len(this) <= len(nameBuf) {
		names = nameBuf[0:len(this)]
	} else {
		names = _NAME_POOL.GetCapped(len(this))
		defer _NAME_POOL.Put(names)
		names = names[0:len(this)]
	}

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

	var nameBuf [_NAME_CAP]string
	var names []string
	if len(this) <= len(nameBuf) {
		names = nameBuf[0:len(this)]
	} else {
		names = _NAME_POOL.GetCapped(len(this))
		defer _NAME_POOL.Put(names)
		names = names[0:len(this)]
	}

	names = sortedNames(this, names)

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
	n := 1 << bits.Len64(uint64(len(this)))
	size := uint64(_INTERFACE_SIZE*n) + _MAP_SIZE
	for e, v := range this {
		size += anySize(v) + uint64(len(e))
	}
	return size
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

	var nameBuf [_NAME_CAP]string
	var allNames []string
	if len(combined) <= len(nameBuf) {
		allNames = nameBuf[0:len(combined)]
	} else {
		allNames = _NAME_POOL.GetCapped(len(combined))
		defer _NAME_POOL.Put(allNames)
		allNames = allNames[0:len(combined)]
	}

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

	var nameBuf [_NAME_CAP]string
	var allNames []string
	if len(combined) <= len(nameBuf) {
		allNames = nameBuf[0:len(combined)]
	} else {
		allNames = _NAME_POOL.GetCapped(len(combined))
		defer _NAME_POOL.Put(allNames)
		allNames = allNames[0:len(combined)]
	}

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

const _NAME_CAP = 16

var _NAME_POOL = util.NewStringPool(256)
