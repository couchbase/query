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

func GetFunction(name string) (Function, bool) {
	rv, ok := _FUNCTIONS[strings.ToLower(name)]
	return rv, ok
}

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
	"between":   &Between{},
	"eq":        &Eq{},
	"le":        &LE{},
	"like":      &Like{},
	"lt":        &LT{},
	"ismissing": &IsMissing{},
	"isnull":    &IsNull{},
	"isvalued":  &IsValued{},

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
	"regexp_contains": &RegexpContains{},
	"regexp_like":     &RegexpLike{},
	"regexp_position": &RegexpPosition{},
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
	"degrees": &Degrees{},
	"exp":     &Exp{},
	"ln":      &Ln{},
	"log":     &Log{},
	"floor":   &Floor{},
	"pi":      &PI{},
	"power":   &Power{},
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
	"array_length":   &ArrayLength{},
	"array_max":      &ArrayMax{},
	"array_min":      &ArrayMin{},
	"array_position": &ArrayPosition{},
	"array_prepend":  &ArrayPrepend{},
	"array_put":      &ArrayPut{},
	"array_range":    &ArrayRange{},
	"array_remove":   &ArrayRemove{},
	"array_repeat":   &ArrayRepeat{},
	"array_replace":  &ArrayReplace{},
	"array_reverse":  &ArrayReverse{},
	"array_sort":     &ArraySort{},
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
	"greatest": &Greatest{},
	"least":    &Least{},

	// Conditional for unknowns
	"ifmissing":       &IfMissing{},
	"ifmissingornull": &IfMissingOrNull{},
	"ifnull":          &IfNull{},
	"firstval":        &FirstVal{},
	"missingif":       &MissingIf{},
	"nullif":          &NullIf{},

	// Conditional for numbers
	"ifinf":      &IfInf{},
	"ifnan":      &IfNaN{},
	"ifnanorinf": &IfNaNOrInf{},
	"ifneginf":   &IfNegInf{},
	"ifposinf":   &IfPosInf{},
	"firstnum":   &FirstNum{},
	"nanif":      &NaNIf{},
	"neginfif":   &NegInfIf{},
	"posinfif":   &PosInfIf{},

	// Meta
	"meta":   &Meta{},
	"base64": &Base64{},

	// Type checking
	"isarray":  &IsArray{},
	"isatom":   &IsAtom{},
	"isbool":   &IsBool{},
	"isnum":    &IsNum{},
	"isobj":    &IsObj{},
	"isstr":    &IsStr{},
	"typename": &TypeName{},

	// Type conversion
	"toarray": &ToArray{},
	"toatom":  &ToAtom{},
	"tobool":  &ToBool{},
	"tonum":   &ToNum{},
	"tostr":   &ToStr{},
}
