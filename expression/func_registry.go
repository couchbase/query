//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"strings"
)

/*
This method is used to retrieve a function by the parser.
Based on the input string name it looks through a map and
retrieves the function that corresponds to it. If the
function exists it returns true and the function. While
looking into the map, convert the string name to lower
case.
*/
func GetFunction(name string) (Function, bool) {
	rv, ok := _FUNCTIONS[strings.ToLower(name)]
	return rv, ok
}

/*
The variable _FUNCTIONS represents a map from string to
Function. Each string returns a pointer to that function.
The types of functions can be grouped into Arithmetic,
Collection, Comparison, Concat, Construction, Logic,
Navigation, Date, String, Numeric, Array, Object, JSON,
Comparison, Conditional for numbers and unknowns, meta,
type checking and type conversion.
*/
var _FUNCTIONS = map[string]Function{
	// Arithmetic
	"add":  &Add{},
	"div":  &Div{},
	"mod":  &Mod{},
	"mult": &Mult{},
	"neg":  &Neg{},
	"sub":  &Sub{},

	// Collection
	"exists": &Exists{},
	"in":     &In{},
	"within": &Within{},

	// Comparison
	"between":        &Between{},
	"eq":             &Eq{},
	"le":             &LE{},
	"like":           &Like{},
	"lt":             &LT{},
	"is_missing":     &IsMissing{},
	"is_not_missing": &IsNotMissing{},
	"is_not_null":    &IsNotNull{},
	"is_not_valued":  &IsNotValued{},
	"is_null":        &IsNull{},
	"is_valued":      &IsValued{},
	"ismissing":      &IsMissing{},
	"isnotmissing":   &IsNotMissing{},
	"isnotnull":      &IsNotNull{},
	"isnotvalued":    &IsNotValued{},
	"isnull":         &IsNull{},
	"isvalued":       &IsValued{},

	// Concat
	"concat": &Concat{},

	// Costruction
	"array": &ArrayConstruct{},

	// Logic
	"and": &And{},
	"not": &Not{},
	"or":  &Or{},

	// Navigation
	"element": &Element{},
	"field":   &Field{},
	"slice":   &Slice{},

	// Date
	"clock_millis":        &ClockMillis{},
	"clock_str":           &ClockStr{},
	"date_add_millis":     &DateAddMillis{},
	"date_add_str":        &DateAddStr{},
	"date_diff_millis":    &DateDiffMillis{},
	"date_diff_str":       &DateDiffStr{},
	"date_part_millis":    &DatePartMillis{},
	"date_part_str":       &DatePartStr{},
	"date_trunc_millis":   &DateTruncMillis{},
	"date_trunc_str":      &DateTruncStr{},
	"millis":              &StrToMillis{},
	"millis_to_str":       &MillisToStr{},
	"millis_to_utc":       &MillisToUTC{},
	"millis_to_zone_name": &MillisToZoneName{},
	"now_millis":          &NowMillis{},
	"now_str":             &NowStr{},
	"str_to_millis":       &StrToMillis{},
	"str_to_utc":          &StrToUTC{},
	"str_to_zone_name":    &StrToZoneName{},

	// String
	"contains":        &Contains{},
	"initcap":         &Title{},
	"length":          &Length{},
	"lower":           &Lower{},
	"ltrim":           &LTrim{},
	"position":        &Position{},
	"pos":             &Position{},
	"regex_contains":  &RegexpContains{},
	"regex_like":      &RegexpLike{},
	"regex_position":  &RegexpPosition{},
	"regex_pos":       &RegexpPosition{},
	"regex_replace":   &RegexpReplace{},
	"regexp_contains": &RegexpContains{},
	"regexp_like":     &RegexpLike{},
	"regexp_position": &RegexpPosition{},
	"regexp_pos":      &RegexpPosition{},
	"regexp_replace":  &RegexpReplace{},
	"repeat":          &Repeat{},
	"replace":         &Replace{},
	"rtrim":           &RTrim{},
	"split":           &Split{},
	"substr":          &Substr{},
	"title":           &Title{},
	"trim":            &Trim{},
	"upper":           &Upper{},

	// Numeric
	"abs":     &Abs{},
	"acos":    &Acos{},
	"asin":    &Asin{},
	"atan":    &Atan{},
	"atan2":   &Atan2{},
	"ceil":    &Ceil{},
	"cos":     &Cos{},
	"deg":     &Degrees{},
	"degrees": &Degrees{},
	"e":       &E{},
	"exp":     &Exp{},
	"ln":      &Ln{},
	"log":     &Log{},
	"floor":   &Floor{},
	"inf":     &PosInf{},
	"nan":     &NaN{},
	"neginf":  &NegInf{},
	"pi":      &PI{},
	"posinf":  &PosInf{},
	"power":   &Power{},
	"rad":     &Radians{},
	"radians": &Radians{},
	"random":  &Random{},
	"round":   &Round{},
	"sign":    &Sign{},
	"sin":     &Sin{},
	"sqrt":    &Sqrt{},
	"tan":     &Tan{},
	"trunc":   &Trunc{},

	// Array
	"array_append":   &ArrayAppend{},
	"array_avg":      &ArrayAvg{},
	"array_concat":   &ArrayConcat{},
	"array_contains": &ArrayContains{},
	"array_count":    &ArrayCount{},
	"array_distinct": &ArrayDistinct{},
	"array_ifnull":   &ArrayIfNull{},
	"array_insert":   &ArrayInsert{},
	"array_length":   &ArrayLength{},
	"array_max":      &ArrayMax{},
	"array_min":      &ArrayMin{},
	"array_position": &ArrayPosition{},
	"array_pos":      &ArrayPosition{},
	"array_prepend":  &ArrayPrepend{},
	"array_put":      &ArrayPut{},
	"array_range":    &ArrayRange{},
	"array_remove":   &ArrayRemove{},
	"array_repeat":   &ArrayRepeat{},
	"array_replace":  &ArrayReplace{},
	"array_reverse":  &ArrayReverse{},
	"array_sort":     &ArraySort{},
	"array_star":     &ArrayStar{},
	"array_sum":      &ArraySum{},

	// Object
	"object_length": &ObjectLength{},
	"object_names":  &ObjectNames{},
	"object_pairs":  &ObjectPairs{},
	"object_values": &ObjectValues{},

	// JSON
	"decode_json":  &DecodeJSON{},
	"encode_json":  &EncodeJSON{},
	"encoded_size": &EncodedSize{},
	"poly_length":  &PolyLength{},

	// Comparison
	"greatest":  &Greatest{},
	"least":     &Least{},
	"successor": &Successor{},

	// Conditional for unknowns
	"ifmissing":       &IfMissing{},
	"ifmissingornull": &IfMissingOrNull{},
	"ifnull":          &IfNull{},
	"missingif":       &MissingIf{},
	"nullif":          &NullIf{},

	// Conditional for numbers
	"ifinf":      &IfInf{},
	"ifnan":      &IfNaN{},
	"ifnanorinf": &IfNaNOrInf{},
	"nanif":      &NaNIf{},
	"neginfif":   &NegInfIf{},
	"posinfif":   &PosInfIf{},

	// Meta
	"base64":      &Base64{},
	"meta":        &Meta{},
	"min_version": &MinVersion{},
	"self":        &Self{},
	"uuid":        &Uuid{},
	"version":     &Version{},

	// Type checking
	"is_array":   &IsArray{},
	"is_atom":    &IsAtom{},
	"is_bin":     &IsBinary{},
	"is_binary":  &IsBinary{},
	"is_bool":    &IsBoolean{},
	"is_boolean": &IsBoolean{},
	"is_num":     &IsNumber{},
	"is_number":  &IsNumber{},
	"is_obj":     &IsObject{},
	"is_object":  &IsObject{},
	"is_str":     &IsString{},
	"is_string":  &IsString{},
	"isarray":    &IsArray{},
	"isatom":     &IsAtom{},
	"isbin":      &IsBinary{},
	"isbinary":   &IsBinary{},
	"isbool":     &IsBoolean{},
	"isboolean":  &IsBoolean{},
	"isnum":      &IsNumber{},
	"isnumber":   &IsNumber{},
	"isobj":      &IsObject{},
	"isobject":   &IsObject{},
	"isstr":      &IsString{},
	"isstring":   &IsString{},
	"type":       &Type{},
	"type_name":  &Type{},
	"typename":   &Type{},

	// Type conversion
	"to_array":   &ToArray{},
	"to_atom":    &ToAtom{},
	"to_bool":    &ToBoolean{},
	"to_boolean": &ToBoolean{},
	"to_num":     &ToNumber{},
	"to_number":  &ToNumber{},
	"to_obj":     &ToObject{},
	"to_object":  &ToObject{},
	"to_str":     &ToString{},
	"to_string":  &ToString{},
	"toarray":    &ToArray{},
	"toatom":     &ToAtom{},
	"tobool":     &ToBoolean{},
	"toboolean":  &ToBoolean{},
	"tonum":      &ToNumber{},
	"tonumber":   &ToNumber{},
	"toobj":      &ToObject{},
	"toobject":   &ToObject{},
	"tostr":      &ToString{},
	"tostring":   &ToString{},

	// Unnest
	"unnest_position": &UnnestPosition{},
	"unnest_pos":      &UnnestPosition{},
}
