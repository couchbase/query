//  Copieright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

Package value provides a native abstraction for JSON data values, with
delayed parsing.

*/
package value

import (
	"fmt"
	"reflect"
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
	Equals(other Value) bool                      // False negatives allowed
	Copy() Value                                  // Shallow copy
	CopyForUpdate() Value                         // Deep copy for UPDATE statements; returns Values whose SetIndex() will extend arrays as needed
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

// Missing value
func NewMissingValue() Value {
	return &_MISSING_VALUE
}

// CorrelatedValue enables subqueries.
func NewCorrelatedValue(parent Value) Value {
	return &correlatedValue{
		entries: make(map[string]interface{}),
		parent:  parent,
	}
}

// The data types supported by Value
const (
	NOT_JSON = iota
	MISSING
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

func (this floatValue) Equals(other Value) bool {
	switch other := other.(type) {
	case floatValue:
		return this == other
	default:
		return false
	}
}

func (this floatValue) Copy() Value {
	return this
}

func (this floatValue) CopyForUpdate() Value {
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

func (this stringValue) Equals(other Value) bool {
	switch other := other.(type) {
	case stringValue:
		return this == other
	default:
		return false
	}
}

func (this stringValue) Copy() Value {
	return this
}

func (this stringValue) CopyForUpdate() Value {
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

func (this boolValue) Equals(other Value) bool {
	switch other := other.(type) {
	case boolValue:
		return this == other
	default:
		return false
	}
}

func (this boolValue) Copy() Value {
	return this
}

func (this boolValue) CopyForUpdate() Value {
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

func (this *nullValue) Equals(other Value) bool {
	switch other.(type) {
	case *nullValue:
		return true
	default:
		return false
	}
}

func (this *nullValue) Copy() Value {
	return this
}

func (this *nullValue) CopyForUpdate() Value {
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

type missingValue struct {
}

var _MISSING_VALUE = missingValue{}

func (this *missingValue) Type() int {
	return MISSING
}

func (this *missingValue) Actual() interface{} {
	return nil
}

func (this *missingValue) Equals(other Value) bool {
	switch other.(type) {
	case *missingValue:
		return true
	default:
		return false
	}
}

func (this *missingValue) Copy() Value {
	return this
}

func (this *missingValue) CopyForUpdate() Value {
	return this
}

var _MISSING_BYTES = []byte("missing")

func (this *missingValue) Bytes() []byte {
	return _MISSING_BYTES
}

func (this *missingValue) Field(field string) (Value, error) {
	return nil, Undefined(field)
}

func (this *missingValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

func (this *missingValue) Index(index int) (Value, error) {
	return nil, Undefined(index)
}

func (this *missingValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}

type sliceValue []interface{}

func (this sliceValue) Type() int {
	return ARRAY
}

func (this sliceValue) Actual() interface{} {
	return ([]interface{})(this)
}

func (this sliceValue) Equals(other Value) bool {
	switch other := other.(type) {
	case sliceValue:
		return reflect.DeepEqual(this, other)
	default:
		return false
	}
}

func (this sliceValue) Copy() Value {
	return sliceValue(copySlice(this, self))
}

func (this sliceValue) CopyForUpdate() Value {
	return &listValue{copySlice(this, copyForUpdate)}
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

func (this *listValue) Equals(other Value) bool {
	switch other := other.(type) {
	case *listValue:
		return reflect.DeepEqual(this.actual, other.actual)
	default:
		return false
	}
}

func (this *listValue) Copy() Value {
	return &listValue{copySlice(this.actual, self)}
}

func (this *listValue) CopyForUpdate() Value {
	return &listValue{copySlice(this.actual, copyForUpdate)}
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

func (this objectValue) Equals(other Value) bool {
	switch other := other.(type) {
	case objectValue:
		return reflect.DeepEqual(this, other)
	default:
		return false
	}
}

func (this objectValue) Copy() Value {
	return objectValue(copyMap(this, self))
}

func (this objectValue) CopyForUpdate() Value {
	return objectValue(copyMap(this, copyForUpdate))
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

// CorrelatedValue enables subqueries.
type correlatedValue struct {
	entries map[string]interface{}
	parent  Value
}

func (this *correlatedValue) Type() int {
	return OBJECT
}

func (this *correlatedValue) Actual() interface{} {
	return this.entries
}

func (this *correlatedValue) Equals(other Value) bool {
	switch other := other.(type) {
	case *correlatedValue:
		return reflect.DeepEqual(this.entries, other.entries)
	default:
		return false
	}
}

func (this *correlatedValue) Copy() Value {
	return &correlatedValue{
		entries: copyMap(this.entries, self),
		parent:  this.parent,
	}
}

func (this *correlatedValue) CopyForUpdate() Value {
	return &correlatedValue{
		entries: copyMap(this.entries, copyForUpdate),
		parent:  this.parent,
	}
}

func (this *correlatedValue) Bytes() []byte {
	bytes, err := json.Marshal(this.Actual())
	if err != nil {
		panic(_MARSHAL_ERROR)
	}
	return bytes
}

// Search self and ancestors. Enables subqueries.
func (this *correlatedValue) Field(field string) (Value, error) {
	result, ok := this.entries[field]
	if ok {
		return NewValue(result), nil
	}

	if this.parent != nil {
		return this.parent.Field(field)
	}

	// consistent with parsedValue
	return nil, Undefined(field)
}

func (this *correlatedValue) SetField(field string, val interface{}) error {
	this.entries[field] = val
	return nil
}

func (this *correlatedValue) Index(index int) (Value, error) {
	return nil, Undefined(index)
}

func (this *correlatedValue) SetIndex(index int, val interface{}) error {
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

func (this *parsedValue) Equals(other Value) bool {
	if this.parsed == nil {
		return false
	}

	switch other := other.(type) {
	case *parsedValue:
		return other.parsed != nil && this.parsed.Equals(other.parsed)
	default:
		return this.parsed.Equals(other)
	}
}

func (this *parsedValue) Copy() Value {
	if this.parsed != nil {
		return this.parsed.Copy()
	}

	rv := parsedValue{
		raw:        this.raw,
		parsedType: this.parsedType,
	}

	return &rv
}

func (this *parsedValue) CopyForUpdate() Value {
	if this.parsedType == NOT_JSON {
		return this.Copy()
	}

	return this.parse().CopyForUpdate()
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

type copyFunc func(interface{}) interface{}

func self(val interface{}) interface{} {
	return val
}

func copyForUpdate(val interface{}) interface{} {
	return NewValue(val).CopyForUpdate()
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

func copyMap(source map[string]interface{}, copier copyFunc) map[string]interface{} {
	if source == nil {
		return nil
	}

	result := make(map[string]interface{}, len(source))
	for k, v := range source {
		result[k] = copier(v)
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

func (this *annotatedValue) Copy() Value {
	return &annotatedValue{
		Value:    this.Value.Copy(),
		attacher: attacher{this.attacher.attachments},
	}
}

func (this *annotatedValue) CopyForUpdate() Value {
	return &annotatedValue{
		Value:    this.Value.CopyForUpdate(),
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
