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

	json "github.com/dustin/gojson"
)

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

// Value collections
type Values []Value
type CompositeValues []Values

// An interface for storing and manipulating a (possibly JSON) value.
type Value interface {
	Type() int                                    // Data type constant
	Actual() interface{}                          // Native Go representation
	Equals(other Value) bool                      // Faster than Collate()
	Collate(other Value) int                      // -int if this precedes other
	Truth() bool                                  // Truth value
	Copy() Value                                  // Shallow copy
	CopyForUpdate() Value                         // Deep copy for UPDATEs; returns Values whose SetIndex() can extend arrays
	Bytes() []byte                                // JSON byte encoding
	Field(field string) (Value, bool)             // Object field dereference, or MISSING; true if found
	SetField(field string, val interface{}) error // Object field setting
	UnsetField(field string) error                // Object field unsetting
	Index(index int) (Value, bool)                // Array index dereference, or MISSING; true if found
	SetIndex(index int, val interface{}) error    // Array index setting
	Slice(start, end int) (Value, bool)           // Array slicing; true if found
}

var _CONVERSIONS = []reflect.Type{
	reflect.TypeOf(0.0),
	reflect.TypeOf(false),
	reflect.TypeOf(""),
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
		return NULL_VALUE
	case []byte:
		return NewValueFromBytes(val)
	case []interface{}:
		return sliceValue(val)
	case map[string]interface{}:
		return objectValue(val)
	default:
		for _, c := range _CONVERSIONS {
			if reflect.TypeOf(val).ConvertibleTo(c) {
				return NewValue(reflect.ValueOf(val).Convert(c).Interface())
			}
		}

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
			return NULL_VALUE
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

// The data types supported by Value
const (
	MISSING = iota
	NULL
	NOT_JSON
	BOOLEAN
	NUMBER
	STRING
	ARRAY
	OBJECT
)

func TypeName(t int) (string, bool) {
	if t >= 0 && t < len(_TYPE_NAMES) {
		return _TYPE_NAMES[t], true
	} else {
		return "", false
	}
}

var _TYPE_NAMES = []string{
	MISSING:  "missing",
	NULL:     "null",
	NOT_JSON: "not_json",
	BOOLEAN:  "boolean",
	NUMBER:   "number",
	STRING:   "string",
	ARRAY:    "array",
	OBJECT:   "object",
}

type copyFunc func(interface{}) interface{}

func self(val interface{}) interface{} {
	return val
}

func copyForUpdate(val interface{}) interface{} {
	return NewValue(val).CopyForUpdate()
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
