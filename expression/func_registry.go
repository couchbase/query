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
	rv, ok := _FUNCTIONS[strings.ToUpper(name)]
	return rv, ok
}

var _FUNCTIONS = map[string]Function{
	// Date functions
	"CLOCK_NOW_MILLIS": &ClockNowMillis{},
	"CLOCK_NOW_STR":    &ClockNowStr{},

	/*
		"DATE_ADD_MILLIS":   &DateAddMillis{},
		"DATE_ADD_STR":      &DateAddStr{},
		"DATE_DIFF_MILLIS":  &DateDiffMillis{},
		"DATE_DIFF_STR":     &DateDiffStr{},
		"DATE_PART_MILLIS":  &DatePartMillis{},
		"DATE_PART_STR":     &DatePartStr{},
		"DATE_TRUNC_MILLIS": &DateTruncMillis{},
		"DATE_TRUNC_STR":    &DateTruncStr{},
		"MILLIS_TO_STR":     &MillisToStr{},
	*/

	"NOW_MILLIS": &NowMillis{},
	"NOW_STR":    &NowStr{},
	// "STR_TO_MILLIS":     &StrToMillis{},

	// String functions
	"CONTAINS":        &Contains{},
	"INITCAP":         &Title{},
	"LENGTH":          &Length{},
	"LOWER":           &Lower{},
	"LTRIM":           &LTrim{},
	"POSITION":        &Position{},
	"REGEXP_CONTAINS": &RegexpContains{},
	"REGEXP_LIKE":     &RegexpLike{},
	"REGEXP_POSITION": &RegexpPosition{},
	"REGEXP_REPLACE":  &RegexpReplace{},
	"REPEAT":          &Repeat{},
	"REPLACE":         &Replace{},
	"RTRIM":           &RTrim{},
	"SPLIT":           &Split{},
	"SUBSTR":          &Substr{},
	"TITLE":           &Title{},
	"TRIM":            &Trim{},
	"UPPER":           &Upper{},

	// Number functions
	"ABS":     &Abs{},
	"ACOS":    &Acos{},
	"ASIN":    &Asin{},
	"ATAN":    &Atan{},
	"ATAN2":   &Atan2{},
	"CEIL":    &Ceil{},
	"COS":     &Cos{},
	"DEGREES": &Degrees{},
	"EXP":     &Exp{},
	"LN":      &Ln{},
	"LOG":     &Log{},
	"FLOOR":   &Floor{},
	"PI":      &PI{},
	"POWER":   &Power{},
	"RADIANS": &Radians{},
	"RANDOM":  &Random{},
	"ROUND":   &Round{},
	"SIGN":    &Sign{},
	"SIN":     &Sin{},
	"SQRT":    &Sqrt{},
	"TAN":     &Tan{},
	"TRUNC":   &Trunc{},

	// Array functions
	"ARRAY_APPEND":   &ArrayAppend{},
	"ARRAY_CONCAT":   &ArrayConcat{},
	"ARRAY_CONTAINS": &ArrayContains{},
	"ARRAY_DISTINCT": &ArrayDistinct{},
	"ARRAY_IFNULL":   &ArrayIfNull{},
	"ARRAY_LENGTH":   &ArrayLength{},
	"ARRAY_MAX":      &ArrayMax{},
	"ARRAY_MIN":      &ArrayMin{},
	"ARRAY_POSITION": &ArrayPosition{},
	"ARRAY_PREPEND":  &ArrayPrepend{},
	"ARRAY_PUT":      &ArrayPut{},
	"ARRAY_REMOVE":   &ArrayRemove{},
	"ARRAY_REPEAT":   &ArrayRepeat{},
	"ARRAY_REPLACE":  &ArrayReplace{},
	"ARRAY_REVERSE":  &ArrayReverse{},
	"ARRAY_SORT":     &ArraySort{},

	// Object functions
	"OBJECT_KEYS":   &ObjectKeys{},
	"OBJECT_LENGTH": &ObjectLength{},
	"OBJECT_VALUES": &ObjectValues{},

	// JSON functions
	"DECODE_JSON":  &DecodeJSON{},
	"ENCODE_JSON":  &EncodeJSON{},
	"ENCODED_SIZE": &EncodedSize{},
	"POLY_LENGTH":  &PolyLength{},

	// Comparison functions
	"GREATEST": &Greatest{},
	"LEAST":    &Least{},

	// Conditional functions for unknowns
	"IFMISSING":       &IfMissing{},
	"IFMISSINGORNULL": &IfMissingOrNull{},
	"IFNULL":          &IfNull{},
	"MISSINGIF":       &MissingIf{},
	"NULLIF":          &NullIf{},

	// Conditional functions for numbers
	"IFINF":      &IfInf{},
	"IFNAN":      &IfNaN{},
	"IFNANORINF": &IfNaNOrInf{},
	"IFNEGINF":   &IfNegInf{},
	"IFPOSINF":   &IfPosInf{},
	"FIRSTNUM":   &FirstNum{},
	"NANIF":      &NaNIf{},
	"NEGNINFIF":  &NegInfIf{},
	"POSINFIF":   &PosInfIf{},

	/*
		// Meta and value functions
		"BASE64_VALUE": &Base64Value{},
		"META":         &Meta{},
		"VALUE":        &Value{},

		// Type checking functions
		"IS_ARRAY":  &IsArray{},
		"IS_ATOM":   &IsAtom{},
		"IS_BOOL":   &IsBool{},
		"IS_NUM":    &IsNum{},
		"IS_OBJ":    &IsObj{},
		"IS_STR":    &IsStr{},
		"TYPE_NAME": &TypeName{},

		// Type conversion functions
		"TO_ARRAY": &ToArray{},
		"TO_ATOM":  &ToAtom{},
		"TO_BOOL":  &ToBool{},
		"TO_NUM":   &ToNum{},
		"TO_STR":   &ToStr{},
	*/
}
