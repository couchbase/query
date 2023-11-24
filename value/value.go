//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

/*
Package value represents the N1QL data model. It is the in memory
representation of the data in flight. It provides a native abstraction
for JSON data values, with delayed parsing.
*/
package value

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	json "github.com/couchbase/go_json"
	"github.com/couchbase/query/util"
)

type Tristate int

const (
	NONE Tristate = iota
	FALSE
	TRUE
)

var TRISTATE_NAMES = []string{
	NONE:  "unset",
	FALSE: "off",
	TRUE:  "on",
}

var TRISTATE_NAME_MAP = map[string]Tristate{
	"unset": NONE,
	"off":   FALSE,
	"on":    TRUE,
}

func ParseTristateString(ts string) (Tristate, bool) {
	tv := strings.ToLower(strings.TrimSpace(ts))

	tsv, ok := TRISTATE_NAME_MAP[tv]

	if !ok {
		return NONE, ok
	}

	return tsv, true
}

func TristateToString(tristate Tristate) string {
	return TRISTATE_NAMES[tristate]
}

const (
	_INTERFACE_SIZE = 16
	_POINTER_SIZE   = 8
)

/*
Function ToTristate converts a boolean into a Tristate type. If the
function argument represents a True then it returns a True Tristate
value, else it returns False. It is used to represent the metrics
(which is defined as a type of value. Tristate in the struct of type
BaseRequest) in server/http/http_request.go, which handles the http
request step in the N1ql Architecture diagram and provides the
metadata before the results.
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

/*
The data types supported by Value, present and supported in N1QL.
*/
type Type int

/*
List of valid N1QL types. Missing is specific to N1QL and Binary
refers to unparsed JSON bytes, represented by a bytes array. It is a
non-JSON value. The value type JSON is all-encompassing and covers all
N1ql values.
*/
const (
	MISSING = Type(iota) // Missing field
	NULL                 // Explicit null
	BOOLEAN              // JSON boolean
	NUMBER               // JSON number
	STRING               // JSON string
	ARRAY                // JSON array
	OBJECT               // JSON object
	JSON                 // Non-specific JSON; used in result sets
	BINARY               // non-JSON
)

/*
Stringer interface, which is used in expression/stringer.go to
visit nodes and convert from type defined to a string and return
it.
*/
func (this Type) String() string {
	return _TYPE_NAMES[this]
}

/*
The _TYPE_NAMES variable is a slice of strings that contains the Type
and its corresponding string representation.
*/
var _TYPE_NAMES = []string{
	MISSING: "missing",
	NULL:    "null",
	BOOLEAN: "boolean",
	NUMBER:  "number",
	STRING:  "string",
	ARRAY:   "array",
	OBJECT:  "object",
	JSON:    "json",
	BINARY:  "binary",
}

func (this Type) Successor() Type {
	return _TYPE_SUCCESSORS[this]
}

var _TYPE_SUCCESSORS = []Type{
	MISSING: NULL,
	NULL:    BOOLEAN,
	BOOLEAN: JSON,
	NUMBER:  NUMBER,
	STRING:  STRING,
	ARRAY:   ARRAY,
	OBJECT:  OBJECT,
	JSON:    JSON,
	BINARY:  BINARY,
}

/*
When you try to set a nested property or index that does not exist,
the return error will be Unsettable.
*/
type Unsettable string

/*
Description of which property or index was unsettable (if known).
*/
func (this Unsettable) Error() string {
	if string(this) != "" {
		return fmt.Sprintf("Field or index %s is not settable.", string(this))
	}

	return "Not settable."
}

/*
The _MARSHAL_ERROR constant represents an error string that is output
when there is an unexpected marshal error on valid data. Marshal
returns the JSON encoding of any input interface. It is used while
implementing the method MarshalJSON
*/
const _MARSHAL_ERROR = "Unexpected marshal error on valid data."

/*
A channel of Value objects
*/
type ValueChannel chan Value

/*
Value collections
*/
type Values []Value
type CompositeValues []Values

/*
An interface for storing and manipulating a JSON value.Each 'value'
implements the methods that correspond to it.
*/
type Value interface {
	/*
	   String marshaling.
	*/
	fmt.Stringer

	/*
	   JSON marshaling.
	*/
	json.Marshaler

	/*
	   Marshal JSON and write it out.
	*/
	WriteJSON(order []string, w io.Writer, prefix, indent string, fast bool) error

	/*
	 * Write out in XML format.
	 */
	WriteXML(order []string, w io.Writer, prefix, indent string, fast bool) error

	/*
	   Returns the type of the input based on the previously
	   defined Types(Data type constant).
	*/
	Type() Type

	/*
	   N1QL to native Go representation.
	*/
	Actual() interface{}

	/*
	   N1QL to native Go representation. Exact for indexing.
	*/
	ActualForIndex() interface{}

	/*
	   Equality comparison. It is faster than Collate().
	*/
	Equals(other Value) Value

	/*
	   Equivalence.
	*/
	EquivalentTo(other Value) bool

	/*
	   Returns –int, 0 or +int depending on if the receiver this
	   sorts less than, equal to, or greater than the input
	   argument Value to the method. It uses the type order
	   defined previously.  (This order has also been defined in
	   the N1QL spec under ORDER BY.)
	*/
	Collate(other Value) int

	/*
	   Returns –int, 0 or +int depending on if the receiver this
	   is less than, equal to, or greater than the input argument
	   Value to the method. MAY return MISSING or NULL per the
	   N1QL comparison rules. It uses the type order defined
	   previously.  (This order has also been defined in the N1QL
	   spec under ORDER BY.)
	*/
	Compare(other Value) Value

	/*
	   Returns the Boolean interpretation of the input this for
	   different values(Truth value).
	*/
	Truth() bool

	/*
	   Quick conversion to non marshalled string. Used to avoid heap escapes on value returns
	*/
	ToString() string

	/*
	   Returns a Value, which is a shallow copy of the input.
	*/
	Copy() Value

	/*
	   Returns a Value that is a deep copy of the receiver. It is
	   used for Updates.
	*/
	CopyForUpdate() Value

	/*
	   Access a field or nested data in an object.(Object field
	   dereference) Returns a value and a Boolean; the value being
	   either a missing or the N1QL Value of the input for
	   objects, and a true if found.  This function returns a
	   missingField and false; for all the value types except
	   Object.
	*/
	Field(field string) (Value, bool)

	/*
	   Set a field in an object. For types other than object,
	   Unsettable is called since this method is not valid for
	   those types.
	*/
	SetField(field string, val interface{}) error

	/*
	   It deletes the input field for an object. For types other
	   than object, Unsettable is called.
	*/
	UnsetField(field string) error

	/*
	   Access an entry at a particular index in the array.(Array
	   index dereference) The return value is the Value at that
	   index and a Boolean; the value being a N1QL value of the
	   input for the slice and a true if found. It returns
	   missingIndex and false; for all types except slice/array.
	*/
	Index(index int) (Value, bool)

	/*
	   Populate the value at a particular index in the slice with
	   val of type interface. Slices do not extend beyond their
	   length. For any attempt to set an index that is greater
	   than length, Unsettable is called.
	*/
	SetIndex(index int, val interface{}) error

	/*
	   Array slicing. Takes a start and end index and returns a
	   new slice; also returns a bool that is true if receiver is
	   of type array. For all non slice values it returns
	   NULL_VALUE and false.
	*/
	Slice(start, end int) (Value, bool)

	/*
	   Array slicing to the end of the array. Takes a start index
	   and returns a new slice till the end of the slice; bool
	   returns true if found. For all non array/slice values it
	   returns a NULL_VALUE and false.
	*/
	SliceTail(start int) (Value, bool)

	/*
	   Lists the descendants of an array or object in depth first
	   order (multilevel list flattening) by appending to an input
	   buffer and returning it.
	*/
	Descendants(buffer []interface{}) []interface{}

	/*
	   Lists the name-value fields in this value. Nil for
	   non-object values.
	*/
	Fields() map[string]interface{}

	/*
	   Lists the field names in this value. Nil for non-object
	   values.
	*/
	FieldNames(buffer []string) []string

	/*
	   Lists the descendant pairs of an array or object in depth
	   first order (multilevel list flattening) by appending to an
	   input buffer and returning it.
	*/
	DescendantPairs(buffer []util.IPair) []util.IPair

	/*
	   Returns an approximate successor to this value, or nil. An
	   approximate successor is a value greater than this value,
	   yet as small as is reasonable. The successor is not exact,
	   to account for floating point and other imprecisions.
	*/
	Successor() Value

	/*
	   Recycle this Value's memory. This Value should not be
	   referenced or used after this call.
	*/
	Recycle()

	/*
	   Track keeps a reference count (if required), so that recycling
	   is only done when no references remain
	*/
	Track()

	/*
	   MB-20850. List all the atomic tokens of a value.
	*/
	Tokens(set *Set, options Value) *Set

	/*
	   Token search.
	*/
	ContainsToken(token, options Value) bool

	/*
	   Token search.
	*/
	ContainsMatchingToken(matcher MatchFunc, options Value) bool

	/*
	   Size in bytes, for memory quota
	*/
	Size() uint64

	/*
	   Returns a value that is not wrapped. For internal use.
	*/
	unwrap() Value

	WriteSpill(io.Writer, []byte) error
	ReadSpill(io.Reader, []byte) error
}

var _CONVERSIONS = []reflect.Type{
	reflect.TypeOf(int64(0)),
	reflect.TypeOf(0.0),
	reflect.TypeOf(false),
	reflect.TypeOf(""),
}

/*
Bring a data object into the Value type system from a Go Type.
*/
func NewValue(val interface{}) Value {
	if val == nil {
		return NULL_VALUE
	}

	switch val := val.(type) {
	case Value:
		return val
	case int64:
		return intValue(val)
	case float64:
		if IsInt(val) {
			return intValue(int64(val))
		} else {
			return floatValue(val)
		}
	case string:
		return stringValue(val)
	case bool:
		return boolValue(val)
	case nil:
		return NULL_VALUE
	case []byte:
		return NewParsedValue(val, false)
	case []interface{}:
		return sliceValue(val)
	case map[string]interface{}:
		return objectValue(val)
	case *parsedValue:
		return val
	case int:
		return intValue(val)
	case Values:
		rv := make([]interface{}, len(val))
		for i, v := range val {
			rv[i] = v
		}
		return sliceValue(rv)
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
	case map[string]Value:
		rv := make(map[string]interface{}, len(val))
		for i, v := range val {
			rv[i] = v
		}
		return objectValue(rv)
	default:
		for _, c := range _CONVERSIONS {
			if reflect.TypeOf(val).ConvertibleTo(c) {
				return NewValue(reflect.ValueOf(val).Convert(c).Interface())
			}
		}

		panic(fmt.Sprintf("Cannot create value for type %T.", val))
	}
}

/*
Figures here derived from testing various maps.
*/
func mapBaseSize(l int) uint64 {
	switch {
	case l == 0:
		return 24
	case l <= 8:
		return 336
	case l <= 13:
		return 912
	case l <= 26:
		return 1536
	default:
		return uint64(1024 + (l/26)*2048)
	}
}

/*
Used by some Value.Size() functions to obtain the size of elements.
*/
func AnySize(v interface{}) uint64 {
	switch v := v.(type) {
	case Value:
		return v.Size()
	case string:
		return uint64(len(v) + _INTERFACE_SIZE)
	case map[string]interface{}:
		s := mapBaseSize(len(v))
		for k, vv := range v {
			s += AnySize(k)
			s += AnySize(vv)
		}
		return s
	case map[string]Value:
		s := mapBaseSize(len(v))
		for k, vv := range v {
			s += AnySize(k)
			s += vv.Size()
		}
		return s
	case []interface{}:
		s := uint64(_INTERFACE_SIZE * cap(v))
		for i := range v {
			s += AnySize(v[i])
		}
		return s
	case []Value:
		s := uint64(_POINTER_SIZE * cap(v))
		for i := range v {
			s += v[i].Size()
		}
		return s
	case []AnnotatedValue:
		s := uint64(_POINTER_SIZE * cap(v))
		for i := range v {
			s += v[i].Size()
		}
		return s
	case Values:
		s := uint64(_POINTER_SIZE * cap(v))
		for i := range v {
			s += v[i].Size()
		}
		return s
	case int32:
		return 4
	case uint32:
		return 4
	case float32:
		return 4
	case uint8:
		return 1
	case bool:
		return 1
	default:
		return 8
	}
}

/*
For token search.
*/
type MatchFunc func(token interface{}) bool

/*
Function takes an input interface and returns an interface.
*/
type copyFunc func(interface{}) interface{}

/*
It is passed an interface and returns itself. It allows us to
access the input object.
*/
func self(val interface{}) interface{} {
	return val
}

/*
Calls the CopyForUpdate method for that particular value type
and returns the interface returned by that call. This is used
by array.go, when there is a necessity to copy all the elements
of the array, which can have any JSON type.
*/
func copyForUpdate(val interface{}) interface{} {
	return NewValue(val).CopyForUpdate()
}

func marshalString(v Value) string {
	bytes, err := v.MarshalJSON()
	if err != nil {
		// We should not get here.
		panic(fmt.Sprintf("Error marshaling Value %v: %v", v, err))
	}
	return string(bytes)
}

// Used in WriteJSON functions to avoid heap contention
var _JSON_WRITE_BYTE_POOL = util.NewBytePool(128)

// Portions of this constant are clipped for the indent which saves heap allocation
// If indentation is longer than this constant (1000 characters) then heap allocations occur
const _INDENT_CHARS = "\n" +
	"                                                                                                    " +
	"                                                                                                    " +
	"                                                                                                    " +
	"                                                                                                    " +
	"                                                                                                    " +
	"                                                                                                    " +
	"                                                                                                    " +
	"                                                                                                    " +
	"                                                                                                    " +
	"                                                                                                    "

func getFullPrefix(prefix, indent string) string {
	l := len(prefix) + len(indent) + 1
	if l <= len(_INDENT_CHARS) {
		return _INDENT_CHARS[0:l]
	}
	return "\n" + prefix + indent
}

func ApplyDurationStyleToValue(s util.DurationStyle, filter func(string) bool, v Value) Value {
	if s == util.LEGACY {
		return v
	}
	for k, f := range v.Fields() {
		fv := NewValue(f)
		if fv.Type() == STRING && (filter == nil || filter(k)) {
			d, err := time.ParseDuration(fv.ToString())
			if err == nil {
				v.SetField(k, util.FormatDuration(d, s))
			}
		} else if fv.Type() == OBJECT {
			ApplyDurationStyleToValue(s, filter, fv)
		} else if fv.Type() == ARRAY {
			av, ok := fv.Index(0)
			for i := 1; ok; i++ {
				ApplyDurationStyleToValue(s, filter, av)
				av, ok = fv.Index(i)
			}
		}
	}
	return v
}
