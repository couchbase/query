//  Copieright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*
Package value represents the data model. It is the in memory representation of the data in 
flight. It provides a native abstraction for JSON data values, with delayed parsing.
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

/*
 Function ToTristate converts a boolean into a Tristate type. If the function argument represents a True then it returns a True Tristate value, else it returns False. It is used to represent the metrics (which is defined as a type of value. Tristate in the struct of type BaseRequest) in server/http/http_request.go, which handles the http request step in the N1ql Architecture diagram and provides the metadata before the results. 
*/
func ToTristate(b bool) Tristate {
	if b {
		return TRUE
	} else {
		return FALSE
	}
}

/*
Function ToBool converts a Tristate value to a boolean. 
*/
func ToBool(t Tristate) bool {
	return t == TRUE
}

// The data types supported by Value, present and supported in N1QL.
type Type int

/*
List of valid N1QL types. Missing is specific to N1QL and Binary refers to unparsed JSON bytes,
represented by a bytes array. It is a non-JSON value. The value type JSON is all-encompassing and covers all N1ql values.
*/
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

// Stringer interface, which is used in expression/stringer.go to visit nodes and convert from 
// type defined to a string and return it.
func (this Type) String() string {
	return _TYPE_NAMES[this]
}

/*
The _TYPE_NAMES variable is a slice of strings that contains the Type and its corresponding string representation.
*/
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

/*
The _MARSHAL_ERROR constant represents an error string that is output when there is an unexpected marshal error on valid data. Marshal returns the JSON encoding of any input interface. It is used while implementing the method MarshalJSON
*/
const _MARSHAL_ERROR = "Unexpected marshal error on valid data."

// A channel of Value objects
type ValueChannel chan Value

// Value collections
type Values []Value
type CompositeValues []Values

// An interface for storing and manipulating a JSON value.Each 'value' implements the methods that
//correspond to it.

type Value interface {
	/* 
           This method is used by the json package. It is used to convert to JSON byte encoding; 
           and returns a byte array of valid JSON values. error is always nil. 
        */
        MarshalJSON() ([]byte, error)                  

        /*
           Returns the type of the input based on the previously defined Types(Data type constant).
        */
	Type() Type                                     

        /*
           N1QL to native Go representation of method receiver. It returns an interface.
        */
	Actual() interface{}                            

        /*
           Returns a Boolean based on if receiver and input argument Value are equal. It is faster than Collate().
        */
	Equals(other Value) bool                        

        /*
           Returns –int, 0 or +int depending on if the receiver this sorts less than, equal to, or greater 
           than the input argument Value to the method. It uses the type order defined previously.
           (This order has also been defined in the N1QL spec under order by.) 
        */
	Collate(other Value) int                        

        /*
           Returns the Boolean interpretation of the input this for different values(Truth value).
        */
	Truth() bool                                    

        /*
           Returns a Value, which is a shallow copy of the input. 
        */
	Copy() Value                                    

        /*
           Returns a Value that is a deep copy of the receiver. It is used for Updates.
        */
	CopyForUpdate() Value                          

        /*
           Access a field or nested data in an object.(Object field dereference) Returns a value and a Boolean; 
           the value being either a missing or the N1QL Value of the input for objects, and a true if found.
           This function returns a missingField and false; for all the value types except Object.
        */
	Field(field string) (Value, bool)               

        /*
           Set a field in an object. For types other than object, Unsettable is called since this 
           method is not valid for those types.
        */
	SetField(field string, val interface{}) error   

        /*
           It deletes the input field for an object. For types other than object, Unsettable is called.
        */
	UnsetField(field string) error                  

        /*
           Access an entry at a particular index in the array.(Array index dereference) The return value is the 
           Value at that index and a Boolean; the value being a N1QL value of the input for the slice and a 
           true if found. It returns missingIndex and false; for all types except slice/array.
        */
	Index(index int) (Value, bool)                  
	
        /*
           Populate the value at a particular index in the slice with val of type interface. Slices do not 
           extend beyond their length. For any attempt to set an index that is greater than length, 
           Unsettable is called.
        */
        SetIndex(index int, val interface{}) error      

        /*
           Array slicing. Takes a start and end index and returns a new slice; also returns a bool that is true 
           if receiver is of type array. For all non slice values it returns NULL_VALUE and false.
        */
	Slice(start, end int) (Value, bool)             
 
        /*
           Array slicing to the end of the array. Takes a start index and returns a new slice till the end of
           the slice; bool returns true if found. For all non array/slice values it returns a NULL_VALUE and false.
        */
	SliceTail(start int) (Value, bool)              

        /*
           Lists the descendants of an array or object in depth first order (multilevel list flattening) 
           by adding it to an input buffer and returning it.
        */
	Descendants(buffer []interface{}) []interface{} 
	
        /*
           Gives you the object fields by adding them to a string to interface map. This returns null for all 
           types except object. ( it is used in expression/nav_field.go to navigate through fields of value Value).
        */
        Fields() map[string]interface{}                
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
