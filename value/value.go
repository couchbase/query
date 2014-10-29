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

type Tristate int

const (
	NONE Tristate = iota
	FALSE
	TRUE
)

func ToTristate(b bool) Tristate {
	if b {
		return TRUE
	} else {
		return FALSE
	}
}

func ToBool(t Tristate) bool {
	return t == TRUE
}

// The data types supported by Value
type Type int

const (
	MISSING = Type(iota) // Missing field
	NULL                 // Explicit null
	BINARY               // non-JSON
	BOOLEAN              // JSON boolean
	NUMBER               // JSON number
	STRING               // JSON string
	ARRAY                // JSON array
	OBJECT               // JSON object
	JSON                 // Non-specific JSON; used in result sets
)

// Stringer interface
func (this Type) String() string {
	return _TYPE_NAMES[this]
}

var _TYPE_NAMES = []string{
	MISSING: "missing",
	NULL:    "null",
	BINARY:  "binary",
	BOOLEAN: "boolean",
	NUMBER:  "number",
	STRING:  "string",
	ARRAY:   "array",
	OBJECT:  "object",
	JSON:    "json",
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

// Value collections
type Values []Value
type CompositeValues []Values

// An interface for storing and manipulating a (possibly JSON) value.
type Value interface {
	MarshalJSON() ([]byte, error)                   // JSON byte encoding; error is always nil
	Type() Type                                     // Data type constant
	Actual() interface{}                            // Native golang representation (non-N1QL) of _this_
	Equals(other Value) bool                        // Faster than Collate()
	Collate(other Value) int                        // -int, 0, or +int if _this_ sorts as less than, equals, or greater than _other_
	Truth() bool                                    // Truth value
	Copy() Value                                    // Shallow copy
	CopyForUpdate() Value                           // Deep copy for UPDATEs
	Field(field string) (Value, bool)               // Object field dereference, or MISSING; true if found
	SetField(field string, val interface{}) error   // Object field setting
	UnsetField(field string) error                  // Object field unsetting
	Index(index int) (Value, bool)                  // Array index dereference, or MISSING; true if found
	SetIndex(index int, val interface{}) error      // Array index setting
	Slice(start, end int) (Value, bool)             // Array slicing; true if found
	SliceTail(start int) (Value, bool)              // Array slicing to the end of the array; true if found
	Descendants(buffer []interface{}) []interface{} // Depth-first listing of this value's descendants
	Fields() map[string]interface{}                 // Field names, if any
}

var _CONVERSIONS = []reflect.Type{
	reflect.TypeOf(0.0),
	reflect.TypeOf(false),
	reflect.TypeOf(""),
}

// Bring a data object into the Value type system
func NewValue(val interface{}) Value {
	if val == nil {
		return NULL_VALUE
	}

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
	case []Value:
		rv := make([]interface{}, len(val))
		for i, v := range val {
			rv[i] = v
		}
		return sliceValue(rv)
	case []AnnotatedValue:
		rv := make([]interface{}, len(val))
		for i, v := range val {
			rv[i] = v
		}
		return sliceValue(rv)
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
	var parsedType Type
	err := json.Validate(bytes)

	if err == nil {
		parsedType = identifyType(bytes)

		switch parsedType {
		case NUMBER, STRING, BOOLEAN, NULL:
			var p interface{}
			err := json.Unmarshal(bytes, &p)
			if err != nil {
				panic("Parse error on JSON data.")
			}

			return NewValue(p)
		}
	}

	rv := &parsedValue{
		raw: bytes,
	}

	if err != nil {
		rv.parsedType = BINARY
	} else {
		rv.parsedType = parsedType
	}

	return rv
}

type copyFunc func(interface{}) interface{}

func self(val interface{}) interface{} {
	return val
}

func copyForUpdate(val interface{}) interface{} {
	return NewValue(val).CopyForUpdate()
}

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
	}

	panic("Unable to identify type of JSON data.")
}
