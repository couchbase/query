//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package value

import (
	"fmt"
	"strconv"

	jsonpointer "github.com/dustin/go-jsonpointer"
	json "github.com/dustin/gojson"
)

// When you try to access a nested property or index that does not exist,
// the return value will be nil, and the return error will be Undefined.
type Undefined string

// Description of which property or index was undefined (if known).
func (this Undefined) Error() string {
	if string(this) != "" {
		return fmt.Sprintf("Field or index %s is not defined.", string(this))
	}
	return "Not defined."
}

// When you try to set a nested property or index that does not exist,
// the return error will be Unsettable.
type Unsettable string

// Description of which property or index was unsettable (if known).
func (this Unsettable) Error() string {
	if string(this) != "" {
		return fmt.Sprintf("Field or index %s is not settable.", string(this))
	}
	return "Not settable."
}

const _MARSHAL_ERROR = "Unexpected marshal error on valid data."

// A channel of Value objects
type ValueChannel chan Value

// A composite Value
type CompositeValue []Value

// A collection of Value objects
type ValueCollection []Value

// An interface for storing and manipulating a (possibly JSON) value.
type Value interface {
	Type() int                                    // Data type constant
	Actual() interface{}                          // Native Go representation
	Duplicate() Value                             // Shallow copy
	DuplicateForUpdate() Value                    // Deep copy for UPDATE statements; returns Values whose SetIndex() will extend arrays as needed
	Bytes() []byte                                // JSON byte encoding
	Field(field string) (Value, error)            // Object field dereference
	SetField(field string, val interface{}) error // Object field setting
	Index(index int) (Value, error)               // Array index dereference
	SetIndex(index int, val interface{}) error    // Array index setting
}

type AnnotatedValue interface {
	Value
	GetAttachment(key string) interface{}
	SetAttachment(key string, val interface{})
	RemoveAttachment(key string) interface{}
}

// Bring a data object into the Value type system
func NewValue(val interface{}) Value {
	switch val := val.(type) {
	case Value:
		return val
	case float64:
		return floatValue(val)
	case string:
		return stringValue(val)
	case bool:
		return boolValue(val)
	case nil:
		return &_NULL_VALUE
	case []byte:
		return NewValueFromBytes(val)
	case []interface{}:
		return sliceValue(val)
	case map[string]interface{}:
		return objectValue(val)
	default:
		panic(fmt.Sprintf("Cannot create value for type %T.", val))
	}
}

// Create a new Value from a slice of bytes. (this need not be valid JSON)
func NewValueFromBytes(bytes []byte) Value {
	var parsedType int
	err := json.Validate(bytes)

	if err == nil {
		parsedType = identifyType(bytes)

		switch parsedType {
		case NUMBER, STRING, BOOLEAN:
			var p interface{}
			err := json.Unmarshal(bytes, &p)
			if err != nil {
				panic("Unexpected parse error on valid JSON.")
			}

			return NewValue(p)
		case NULL:
			return &_NULL_VALUE
		}
	}

	rv := parsedValue{
		raw: bytes,
	}

	if err != nil {
		rv.parsedType = NOT_JSON
	} else {
		rv.parsedType = parsedType
	}

	return &rv
}

// Create an AnnotatedValue to hold attachments
func NewAnnotatedValue(val interface{}) AnnotatedValue {
	switch val := val.(type) {
	case AnnotatedValue:
		return val
	case Value:
		av := annotatedValue{
			Value:    val,
			attacher: attacher{nil},
		}
		return &av
	default:
		return NewAnnotatedValue(NewValue(val))
	}
}

// The data types supported by Value
const (
	NOT_JSON = iota
	NULL
	BOOLEAN
	NUMBER
	STRING
	ARRAY
	OBJECT
)

type floatValue float64

func (this floatValue) Type() int {
	return NUMBER
}

func (this floatValue) Actual() interface{} {
	return float64(this)
}

func (this floatValue) Duplicate() Value {
	return this
}

func (this floatValue) DuplicateForUpdate() Value {
	return this
}

func (this floatValue) Bytes() []byte {
	bytes, err := json.Marshal(this.Actual())
	if err != nil {
		panic(_MARSHAL_ERROR)
	}
	return bytes
}

func (this floatValue) Field(field string) (Value, error) {
	return nil, Undefined(field)
}

func (this floatValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

func (this floatValue) Index(index int) (Value, error) {
	return nil, Undefined(index)
}

func (this floatValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}

type stringValue string

func (this stringValue) Type() int {
	return STRING
}

func (this stringValue) Actual() interface{} {
	return string(this)
}

func (this stringValue) Duplicate() Value {
	return this
}

func (this stringValue) DuplicateForUpdate() Value {
	return this
}

func (this stringValue) Bytes() []byte {
	bytes, err := json.Marshal(this.Actual())
	if err != nil {
		panic(_MARSHAL_ERROR)
	}
	return bytes
}

func (this stringValue) Field(field string) (Value, error) {
	return nil, Undefined(field)
}

func (this stringValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

func (this stringValue) Index(index int) (Value, error) {
	return nil, Undefined(index)
}

func (this stringValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}

type boolValue bool

func (this boolValue) Type() int {
	return BOOLEAN
}

func (this boolValue) Actual() interface{} {
	return bool(this)
}

func (this boolValue) Duplicate() Value {
	return this
}

func (this boolValue) DuplicateForUpdate() Value {
	return this
}

var _FALSE_BYTES = []byte("false")
var _TRUE_BYTES = []byte("true")

func (this boolValue) Bytes() []byte {
	if this {
		return _TRUE_BYTES
	} else {
		return _FALSE_BYTES
	}
}

func (this boolValue) Field(field string) (Value, error) {
	return nil, Undefined(field)
}

func (this boolValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

func (this boolValue) Index(index int) (Value, error) {
	return nil, Undefined(index)
}

func (this boolValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}

type nullValue struct {
}

var _NULL_VALUE = nullValue{}

func (this *nullValue) Type() int {
	return NULL
}

func (this *nullValue) Actual() interface{} {
	return nil
}

func (this *nullValue) Duplicate() Value {
	return this
}

func (this *nullValue) DuplicateForUpdate() Value {
	return this
}

var _NULL_BYTES = []byte("null")

func (this *nullValue) Bytes() []byte {
	return _NULL_BYTES
}

func (this *nullValue) Field(field string) (Value, error) {
	return nil, Undefined(field)
}

func (this *nullValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

func (this *nullValue) Index(index int) (Value, error) {
	return nil, Undefined(index)
}

func (this *nullValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}

type sliceValue []interface{}

func (this sliceValue) Type() int {
	return ARRAY
}

func (this sliceValue) Actual() interface{} {
	return ([]interface{})(this)
}

func (this sliceValue) Duplicate() Value {
	return sliceValue(duplicateSlice(this, duplicate))
}

func (this sliceValue) DuplicateForUpdate() Value {
	return &listValue{duplicateSlice(this, duplicateForUpdate)}
}

func (this sliceValue) Bytes() []byte {
	bytes, err := json.Marshal(this.Actual())
	if err != nil {
		panic(_MARSHAL_ERROR)
	}
	return bytes
}

func (this sliceValue) Field(field string) (Value, error) {
	return nil, Undefined(field)
}

func (this sliceValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

func (this sliceValue) Index(index int) (Value, error) {
	if index >= 0 && index < len(this) {
		return NewValue(this[index]), nil
	}

	// consistent with parsedValue
	return nil, Undefined(index)
}

// NOTE: Slices do NOT extend beyond length.
func (this sliceValue) SetIndex(index int, val interface{}) error {
	if index < 0 || index >= len(this) {
		return Unsettable(index)
	}

	this[index] = val
	return nil
}

type listValue struct {
	actual []interface{}
}

func (this *listValue) Type() int {
	return ARRAY
}

func (this *listValue) Actual() interface{} {
	return this.actual
}

func (this *listValue) Duplicate() Value {
	return &listValue{duplicateSlice(this.actual, duplicate)}
}

func (this *listValue) DuplicateForUpdate() Value {
	return &listValue{duplicateSlice(this.actual, duplicateForUpdate)}
}

func (this *listValue) Bytes() []byte {
	bytes, err := json.Marshal(this.Actual())
	if err != nil {
		panic(_MARSHAL_ERROR)
	}
	return bytes
}

func (this *listValue) Field(field string) (Value, error) {
	return nil, Undefined(field)
}

func (this *listValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

func (this *listValue) Index(index int) (Value, error) {
	if index >= 0 && index < len(this.actual) {
		return NewValue(this.actual[index]), nil
	}

	// consistent with parsedValue
	return nil, Undefined(index)
}

func (this *listValue) SetIndex(index int, val interface{}) error {
	if index < 0 {
		return Unsettable(index)
	}

	if index >= len(this.actual) {
		if index < cap(this.actual) {
			this.actual = this.actual[0 : index+1]
		} else {
			act := make([]interface{}, index+1, (index+1)<<1)
			copy(act, this.actual)
			this.actual = act
		}
	}

	this.actual[index] = val
	return nil
}

type objectValue map[string]interface{}

func (this objectValue) Type() int {
	return OBJECT
}

func (this objectValue) Actual() interface{} {
	return (map[string]interface{})(this)
}

func (this objectValue) Duplicate() Value {
	return objectValue(duplicateMap(this, duplicate))
}

func (this objectValue) DuplicateForUpdate() Value {
	return objectValue(duplicateMap(this, duplicateForUpdate))
}

func (this objectValue) Bytes() []byte {
	bytes, err := json.Marshal(this.Actual())
	if err != nil {
		panic(_MARSHAL_ERROR)
	}
	return bytes
}

func (this objectValue) Field(field string) (Value, error) {
	result, ok := this[field]
	if ok {
		return NewValue(result), nil
	}

	// consistent with parsedValue
	return nil, Undefined(field)
}

func (this objectValue) SetField(field string, val interface{}) error {
	this[field] = val
	return nil
}

func (this objectValue) Index(index int) (Value, error) {
	return nil, Undefined(index)
}

func (this objectValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}

// A structure for storing and manipulating a (possibly JSON) value.
type parsedValue struct {
	raw        []byte
	parsedType int
	parsed     Value
}

func (this *parsedValue) Type() int {
	return this.parsedType
}

func (this *parsedValue) Actual() interface{} {
	if this.parsedType == NOT_JSON {
		return nil
	}

	return this.parse().Actual()
}

func (this *parsedValue) Duplicate() Value {
	if this.parsed != nil {
		return this.parsed.Duplicate()
	}

	rv := parsedValue{
		raw:        this.raw,
		parsedType: this.parsedType,
	}

	return &rv
}

func (this *parsedValue) DuplicateForUpdate() Value {
	if this.parsedType == NOT_JSON {
		return this.Duplicate()
	}

	return this.parse().DuplicateForUpdate()
}

func (this *parsedValue) Bytes() []byte {
	switch this.parsedType {
	case ARRAY, OBJECT:
		return this.parse().Bytes()
	default:
		return this.raw
	}
}

func (this *parsedValue) Field(field string) (Value, error) {
	if this.parsed != nil {
		return this.parsed.Field(field)
	}

	if this.parsedType != OBJECT {
		return nil, Undefined(field)
	}

	if this.raw != nil {
		res, err := jsonpointer.Find(this.raw, "/"+field)
		if err != nil {
			return nil, err
		}
		if res != nil {
			return NewValueFromBytes(res), nil
		}
	}

	return nil, Undefined(field)
}

func (this *parsedValue) SetField(field string, val interface{}) error {
	if this.parsedType != OBJECT {
		return Unsettable(field)
	}

	return this.parse().SetField(field, val)
}

func (this *parsedValue) Index(index int) (Value, error) {
	if this.parsed != nil {
		return this.parsed.Index(index)
	}

	if this.parsedType != ARRAY {
		return nil, Undefined(index)
	}

	if this.raw != nil {
		res, err := jsonpointer.Find(this.raw, "/"+strconv.Itoa(index))
		if err != nil {
			return nil, err
		}
		if res != nil {
			return NewValueFromBytes(res), nil
		}
	}

	return nil, Undefined(index)
}

func (this *parsedValue) SetIndex(index int, val interface{}) error {
	if this.parsedType != ARRAY {
		return Unsettable(index)
	}

	return this.parse().SetIndex(index, val)
}

func (this *parsedValue) parse() Value {
	if this.parsed == nil {
		if this.parsedType == NOT_JSON {
			return nil
		}

		var p interface{}
		err := json.Unmarshal(this.raw, &p)
		if err != nil {
			panic("Unexpected parse error on valid JSON.")
		}
		this.parsed = NewValue(p)
	}

	return this.parsed
}

type dupFunc func(interface{}) interface{}

func duplicate(val interface{}) interface{} {
	return val
}

func duplicateForUpdate(val interface{}) interface{} {
	return NewValue(val).DuplicateForUpdate()
}

func duplicateSlice(source []interface{}, dup dupFunc) []interface{} {
	if source == nil {
		return nil
	}

	result := make([]interface{}, len(source))
	for i, v := range source {
		result[i] = dup(v)
	}

	return result
}

func duplicateMap(source map[string]interface{}, dup dupFunc) map[string]interface{} {
	if source == nil {
		return nil
	}

	result := make(map[string]interface{}, len(source))
	for k, v := range source {
		result[k] = dup(v)
	}

	return result
}

func identifyType(bytes []byte) int {
	for _, b := range bytes {
		switch b {
		case '{':
			return OBJECT
		case '[':
			return ARRAY
		case '"':
			return STRING
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			return NUMBER
		case 't', 'f':
			return BOOLEAN
		case 'n':
			return NULL
		}
	}
	panic("Unable to identify type of valid JSON.")
	return -1
}

type annotatedValue struct {
	Value
	attacher
}

func (this *annotatedValue) DuplicateForUpdate() Value {
	return &annotatedValue{
		Value:    this.Value.DuplicateForUpdate(),
		attacher: attacher{this.attacher.attachments},
	}
}

type attacher struct {
	attachments map[string]interface{}
}

// Return the object attached to this Value with this key.
// If no object is attached with this key, nil is returned.
func (this *attacher) GetAttachment(key string) interface{} {
	if this.attachments != nil {
		return this.attachments[key]
	}
	return nil
}

// Attach an arbitrary object to this Value with the specified key.
// Any existing value attached with this same key will be overwritten.
func (this *attacher) SetAttachment(key string, val interface{}) {
	if this.attachments == nil {
		this.attachments = make(map[string]interface{})
	}
	this.attachments[key] = val
}

// Remove an object attached to this Value with this key.
// If there had been an object attached to this Value with this key it is returned, otherwise nil.
func (this *attacher) RemoveAttachment(key string) interface{} {
	var rv interface{}
	if this.attachments != nil {
		rv = this.attachments[key]
		delete(this.attachments, key)
	}
	return rv
}
