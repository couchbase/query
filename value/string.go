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
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	json "github.com/couchbase/go_json"

	"github.com/couchbase/query/util"
)

/*
stringValue is defined as type string.
*/
type stringValue string

/*
Define a value representing an empty string and
assign it to EMPTY_STRING_VALUE.
*/
var EMPTY_STRING_VALUE Value = stringValue("")

/*
Use built-in JSON string marshalling, which handles special
characters.
*/
func (this stringValue) String() string {
	rawString := string(this)
	bytes, err := json.MarshalNoEscape(rawString)
	if err != nil {
		// We should not get here.
		panic(fmt.Sprintf("Error marshaling string Value %s: %v", rawString, err))
	}
	return string(bytes)
}

func (this stringValue) ToString() string {
	return string(this)
}

/*
Use built-in JSON string marshalling, which handles special
characters.
*/
func (this stringValue) MarshalJSON() ([]byte, error) {
	return json.MarshalNoEscape(string(this))
}

func (this stringValue) WriteJSON(order []string, w io.Writer, prefix, indent string, fast bool) error {
	return json.MarshalStringNoEscapeToBuffer(string(this), w.(*bytes.Buffer))
}

func (this stringValue) WriteXML(order []string, w io.Writer, prefix string, indent string, fast bool) error {
	var err error
	if prefix != "" {
		_, err = w.Write([]byte(getFullPrefix(prefix, "")))
		if err != nil {
			return err
		}
	}
	_, err = w.Write([]byte("<string>"))
	if err != nil {
		return err
	}
	err = xml.EscapeText(w, []byte(this))
	if err == nil {
		_, err = w.Write([]byte("</string>"))
	}
	return err
}

func (this stringValue) WriteSpill(w io.Writer, buf []byte) error {
	b := []byte{_SPILL_TYPE_VALUE}
	_, err := w.Write(b)
	if err == nil {
		err = writeSpillValue(w, string(this), buf)
	}
	return err
}

func (this stringValue) ReadSpill(trackMem func(int64) error, r io.Reader, buf []byte) error {
	v, err := readSpillValue(trackMem, r, buf)
	if err == nil && v != nil {
		this = stringValue(v.(string))
	} else {
		this = ""
	}
	return err
}

/*
Type STRING.
*/
func (this stringValue) Type() Type {
	return STRING
}

func (this stringValue) Actual() interface{} {
	return string(this)
}

func (this stringValue) ActualForIndex() interface{} {
	return string(this)
}

/*
If other is type stringValue and is the same as the receiver
return true.
*/
func (this stringValue) Equals(other Value) Value {
	switch other.Type() {
	case MISSING, NULL:
		return other.unwrap()
	case STRING:
		if string(this) == other.ToString() {
			return TRUE_VALUE
		}
	}

	return FALSE_VALUE
}

func (this stringValue) EquivalentTo(other Value) bool {
	switch other.Type() {
	case STRING:
		return string(this) == other.ToString()
	default:
		return false
	}
}

/*
If other is type stringValue, compare with receiver,
if its less than (string comparison) return -1, greater
than return 1, otherwise return 0. For value of type
parsedValue and annotated value call collate again with the
value. The default behavior is to return the position wrt
others type.
*/
func (this stringValue) Collate(other Value) int {
	switch other.Type() {
	case STRING:
		ta := string(this)
		oa := other.ToString()
		if ta < oa {
			return -1
		} else if ta > oa {
			return 1
		} else {
			return 0
		}
	default:
		return int(STRING - other.Type())
	}
}

func (this stringValue) Compare(other Value) Value {
	switch other.Type() {
	case MISSING, NULL:
		return other.unwrap()
	default:
		return intValue(this.Collate(other))
	}
}

/*
If length of string greater than 0, its a valid string.
Return true.
*/
func (this stringValue) Truth() bool {
	return len(this) > 0
}

/*
Return receiver.
*/
func (this stringValue) Copy() Value {
	return this
}

/*
Return receiver.
*/
func (this stringValue) CopyForUpdate() Value {
	return this
}

/*
Calls missingField.
*/
func (this stringValue) Field(field string) (Value, bool) {
	return missingField(field), false
}

/*
Not valid for string.
*/
func (this stringValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

/*
Not valid for string.
*/
func (this stringValue) UnsetField(field string) error {
	return Unsettable(field)
}

/*
Calls missingIndex.
*/
func (this stringValue) Index(index int) (Value, bool) {
	return missingIndex(index), false
}

/*
Not valid for string.
*/
func (this stringValue) SetIndex(index int, val interface{}) error {
	return Unsettable(strconv.Itoa(index))
}

/*
Returns NULL_VALUE
*/
func (this stringValue) Slice(start, end int) (Value, bool) {
	return NULL_VALUE, false
}

/*
Returns NULL_VALUE
*/
func (this stringValue) SliceTail(start int) (Value, bool) {
	return NULL_VALUE, false
}

/*
Returns the input buffer as is.
*/
func (this stringValue) Descendants(buffer []interface{}) []interface{} {
	return buffer
}

/*
No fields to list. Hence return nil.
*/
func (this stringValue) Fields() map[string]interface{} {
	return nil
}

func (this stringValue) FieldNames(buffer []string) []string {
	return nil
}

/*
Returns the input buffer as is.
*/
func (this stringValue) DescendantPairs(buffer []util.IPair) []util.IPair {
	return buffer
}

/*
Append a low-valued byte to string.
*/
func (this stringValue) Successor() Value {
	var buf strings.Builder
	buf.Grow(len(this) + 1) // Pre-allocate buffer size
	buf.WriteString(string(this))
	buf.WriteRune(' ')
	return stringValue(buf.String())
}

func (this stringValue) Track() {
}

func (this stringValue) Recycle() {
}

func (this stringValue) Tokens(set *Set, options Value) *Set {
	tokens := _STRING_TOKENS_POOL.Get()
	defer _STRING_TOKENS_POOL.Put(tokens)

	this.tokens(tokens, options, "", nil)
	for t, _ := range tokens {
		set.Add(stringValue(t))
	}

	return set
}

func (this stringValue) ContainsToken(token, options Value) bool {
	if token.Type() != STRING {
		return false
	}

	tokens := _STRING_TOKENS_POOL.Get()
	defer _STRING_TOKENS_POOL.Put(tokens)

	str := token.ToString()
	return this.tokens(tokens, options, str, nil)
}

func (this stringValue) ContainsMatchingToken(matcher MatchFunc, options Value) bool {
	tokens := _STRING_TOKENS_POOL.Get()
	defer _STRING_TOKENS_POOL.Put(tokens)

	return this.tokens(tokens, options, "", matcher)
}

func (this stringValue) Size() uint64 {
	return uint64(len(this))
}

func (this stringValue) unwrap() Value {
	return this
}

func (this stringValue) tokens(set map[string]bool, options Value,
	token string, matcher MatchFunc) bool {

	// Set case folding function, if specified.
	caseFunc := func(s string) string { return s }
	if caseOption, ok := options.Field("case"); ok && caseOption.Type() == STRING {
		caseStr := caseOption.ToString()
		switch strings.ToLower(caseStr) {
		case "lower":
			caseFunc = strings.ToLower
		case "upper":
			caseFunc = strings.ToUpper
		}
	}

	var fields []string
	split := true

	// To split or not to split.
	if splitOption, ok := options.Field("split"); ok &&
		splitOption.Type() == BOOLEAN && !splitOption.Truth() {

		split = false

		// To trim or not to trim.
		if trimOption, ok := options.Field("trim"); ok &&
			trimOption.Type() == BOOLEAN && !trimOption.Truth() {
			fields = []string{string(this)}
		} else {
			fields = []string{strings.TrimSpace(string(this))}
		}
	}

	// Tokenize alphanumerics.
	if split {
		fields = strings.FieldsFunc(string(this),
			func(c rune) bool {
				return !unicode.IsLetter(c) && !unicode.IsNumber(c)
			})
	}

	for _, field := range fields {
		f := caseFunc(field)
		if f == token || (matcher != nil && matcher(f)) {
			return true
		}

		set[f] = true
	}

	if !split {
		return false
	}

	// Return if not tokenizing specials.
	if specialsOption, ok := options.Field("specials"); !(ok &&
		specialsOption.Type() == BOOLEAN && specialsOption.Truth()) {
		return false
	}

	// Tokenize specials. Specials can be used to preserve email
	// addresses, URLs, hyphenated phone numbers, etc.

	// First tokenize on whitespace and parentheses only.
	fields = strings.FieldsFunc(string(this),
		func(c rune) bool {
			return unicode.IsSpace(c) ||
				c == '(' || c == ')' ||
				c == '[' || c == ']' ||
				c == '{' || c == '}'
		})

	// Right trim special characters.
	for _, field := range fields {
		f := strings.TrimRightFunc(field,
			func(c rune) bool {
				return !unicode.IsLetter(c) && !unicode.IsNumber(c)
			})

		if f != "" {
			f = caseFunc(f)
			if f == token || (matcher != nil && matcher(f)) {
				return true
			}

			set[f] = true
		}
	}

	return false
}

var _STRING_TOKENS_POOL = util.NewStringBoolPool(64)
