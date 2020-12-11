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
	"idiv": &IDiv{},
	"imod": &IMod{},
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
	"is_known":       &IsValued{},
	"is_missing":     &IsMissing{},
	"is_not_known":   &IsNotValued{},
	"is_not_missing": &IsNotMissing{},
	"is_not_null":    &IsNotNull{},
	"is_not_valued":  &IsNotValued{},
	"is_not_unknown": &IsValued{},
	"is_null":        &IsNull{},
	"is_valued":      &IsValued{},
	"isknown":        &IsValued{},
	"ismissing":      &IsMissing{},
	"isnotknown":     &IsNotValued{},
	"isnotmissing":   &IsNotMissing{},
	"isnotnull":      &IsNotNull{},
	"isnotunknown":   &IsValued{},
	"isnotvalued":    &IsNotValued{},
	"isnull":         &IsNull{},
	"isunknown":      &IsNotValued{},
	"isvalued":       &IsValued{},
	"le":             &LE{},
	"like":           &Like{},
	"lt":             &LT{},
	"like_prefix":    &LikePrefix{},
	"like_stop":      &LikeStop{},
	"like_suffix":    &LikeSuffix{},
	"regexp_prefix":  &RegexpPrefix{},
	"regexp_stop":    &RegexpStop{},
	"regexp_suffix":  &RegexpSuffix{},

	// Concat
	"concat":  &Concat{},
	"concat2": &Concat2{},

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

	// Curl
	"curl": &Curl{},

	// Date
	"clock_local":          &ClockStr{},
	"clock_millis":         &ClockMillis{},
	"clock_str":            &ClockStr{},
	"clock_tz":             &ClockTZ{},
	"clock_utc":            &ClockUTC{},
	"date_add_millis":      &DateAddMillis{},
	"date_add_str":         &DateAddStr{},
	"date_diff_millis":     &DateDiffMillis{},
	"date_diff_str":        &DateDiffStr{},
	"date_diff_abs_str":    &DateDiffAbsStr{},
	"date_diff_abs_millis": &DateDiffAbsMillis{},
	"date_format_str":      &DateFormatStr{},
	"date_part_millis":     &DatePartMillis{},
	"date_part_str":        &DatePartStr{},
	"date_range_millis":    &DateRangeMillis{},
	"date_range_str":       &DateRangeStr{},
	"date_trunc_millis":    &DateTruncMillis{},
	"date_trunc_str":       &DateTruncStr{},
	"duration_to_str":      &DurationToStr{},
	"millis":               &StrToMillis{},
	"millis_to_local":      &MillisToStr{},
	"millis_to_str":        &MillisToStr{},
	"millis_to_tz":         &MillisToZoneName{},
	"millis_to_utc":        &MillisToUTC{},
	"millis_to_zone_name":  &MillisToZoneName{},
	"now_local":            &NowStr{},
	"now_millis":           &NowMillis{},
	"now_str":              &NowStr{},
	"now_tz":               &NowTZ{},
	"now_utc":              &NowUTC{},
	"str_to_duration":      &StrToDuration{},
	"str_to_millis":        &StrToMillis{},
	"str_to_tz":            &StrToZoneName{},
	"str_to_utc":           &StrToUTC{},
	"str_to_zone_name":     &StrToZoneName{},
	"weekday_millis":       &WeekdayMillis{},
	"weekday_str":          &WeekdayStr{},

	// String
	"contains":  &Contains{},
	"initcap":   &Title{},
	"length":    &Length{},
	"lower":     &Lower{},
	"ltrim":     &LTrim{},
	"position":  &Position0{},
	"pos":       &Position0{},
	"position0": &Position0{},
	"pos0":      &Position0{},
	"position1": &Position1{},
	"pos1":      &Position1{},
	"repeat":    &Repeat{},
	"replace":   &Replace{},
	"reverse":   &Reverse{},
	"rtrim":     &RTrim{},
	"split":     &Split{},
	"substr":    &Substr0{},
	"substr0":   &Substr0{},
	"substr1":   &Substr1{},
	"suffixes":  &Suffixes{},
	"title":     &Title{},
	"trim":      &Trim{},
	"upper":     &Upper{},

	// Regular expressions
	"contains_regex":   &RegexpContains{},
	"contains_regexp":  &RegexpContains{},
	"regex_contains":   &RegexpContains{},
	"regex_like":       &RegexpLike{},
	"regex_position0":  &RegexpPosition0{},
	"regex_pos0":       &RegexpPosition0{},
	"regexp_position0": &RegexpPosition0{},
	"regexp_pos0":      &RegexpPosition0{},
	"regex_position1":  &RegexpPosition1{},
	"regex_pos1":       &RegexpPosition1{},
	"regexp_position1": &RegexpPosition1{},
	"regexp_pos1":      &RegexpPosition1{},
	"regex_position":   &RegexpPosition0{},
	"regex_pos":        &RegexpPosition0{},
	"regex_replace":    &RegexpReplace{},
	"regexp_contains":  &RegexpContains{},
	"regexp_like":      &RegexpLike{},
	"regexp_position":  &RegexpPosition0{},
	"regexp_pos":       &RegexpPosition0{},
	"regexp_replace":   &RegexpReplace{},
	"regexp_matches":   &RegexpMatches{},
	"regexp_split":     &RegexpSplit{},

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
	"neg_inf": &NegInf{},
	"pi":      &PI{},
	"posinf":  &PosInf{},
	"pos_inf": &PosInf{},
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

	// Bitwise
	"bitand":   &BitAnd{},
	"bitor":    &BitOr{},
	"bitxor":   &BitXor{},
	"bitnot":   &BitNot{},
	"bitshift": &BitShift{},
	"bitset":   &BitSet{},
	"bitclear": &BitClear{},
	"bittest":  &BitTest{},
	"isbitset": &BitTest{},

	// Array
	"array_add":           &ArrayPut{},
	"array_append":        &ArrayAppend{},
	"array_avg":           &ArrayAvg{},
	"array_concat":        &ArrayConcat{},
	"array_contains":      &ArrayContains{},
	"array_count":         &ArrayCount{},
	"array_distinct":      &ArrayDistinct{},
	"array_flatten":       &ArrayFlatten{},
	"array_ifnull":        &ArrayIfNull{},
	"array_insert":        &ArrayInsert{},
	"array_intersect":     &ArrayIntersect{},
	"array_length":        &ArrayLength{},
	"array_max":           &ArrayMax{},
	"array_min":           &ArrayMin{},
	"array_position":      &ArrayPosition{},
	"array_pos":           &ArrayPosition{},
	"array_prepend":       &ArrayPrepend{},
	"array_put":           &ArrayPut{},
	"array_range":         &ArrayRange{},
	"array_remove":        &ArrayRemove{},
	"array_repeat":        &ArrayRepeat{},
	"array_replace":       &ArrayReplace{},
	"array_reverse":       &ArrayReverse{},
	"array_sort":          &ArraySort{},
	"array_star":          &ArrayStar{},
	"array_sum":           &ArraySum{},
	"array_symdiff":       &ArraySymdiff1{},
	"array_symdiff1":      &ArraySymdiff1{},
	"array_symdiffn":      &ArraySymdiffn{},
	"array_union":         &ArrayUnion{},
	"array_swap":          &ArraySwap{},
	"array_move":          &ArrayMove{},
	"array_except":        &ArrayExcept{},
	"array_binary_search": &ArrayBinarySearch{},

	// Object
	"object_add":          &ObjectAdd{},
	"object_concat":       &ObjectConcat{},
	"object_inner_pairs":  &ObjectInnerPairs{},
	"object_innerpairs":   &ObjectInnerPairs{},
	"object_inner_values": &ObjectInnerValues{},
	"object_innervalues":  &ObjectInnerValues{},
	"object_length":       &ObjectLength{},
	"object_names":        &ObjectNames{},
	"object_outer_pairs":  &ObjectPairs{},
	"object_outerpairs":   &ObjectPairs{},
	"object_outer_values": &ObjectValues{},
	"object_outervalues":  &ObjectValues{},
	"object_pairs":        &ObjectPairs{},
	"object_put":          &ObjectPut{},
	"object_remove":       &ObjectRemove{},
	"object_rename":       &ObjectRename{},
	"object_replace":      &ObjectReplace{},
	"object_unwrap":       &ObjectUnwrap{},
	"object_values":       &ObjectValues{},

	// JSON
	"decode_json":  &JSONDecode{},
	"encode_json":  &JSONEncode{},
	"encoded_size": &EncodedSize{},
	"json_decode":  &JSONDecode{},
	"json_encode":  &JSONEncode{},
	"pairs":        &Pairs{},
	"poly_length":  &PolyLength{},

	// Base64
	"base64":        &Base64Encode{},
	"base64_decode": &Base64Decode{},
	"base64_encode": &Base64Encode{},
	"decode_base64": &Base64Decode{},
	"encode_base64": &Base64Encode{},

	// Comparison
	"greatest":  &Greatest{},
	"least":     &Least{},
	"successor": &Successor{},

	// Token
	"contains_token":        &ContainsToken{},
	"contains_token_like":   &ContainsTokenLike{},
	"contains_token_regex":  &ContainsTokenRegexp{},
	"contains_token_regexp": &ContainsTokenRegexp{},
	"has_token":             &ContainsToken{},
	"tokens":                &Tokens{},

	// Conditional for unknowns
	"if_missing":         &IfMissing{},
	"if_missing_or_null": &IfMissingOrNull{},
	"if_null":            &IfNull{},
	"missing_if":         &MissingIf{},
	"null_if":            &NullIf{},
	"ifmissing":          &IfMissing{},
	"ifmissingornull":    &IfMissingOrNull{},
	"ifnull":             &IfNull{},
	"missingif":          &MissingIf{},
	"nullif":             &NullIf{},
	"coalesce":           &IfMissingOrNull{},
	"nvl":                &NVL{},
	"nvl2":               &NVL2{},

	// Conditional for numbers
	"if_inf":        &IfInf{},
	"if_nan":        &IfNaN{},
	"if_nan_or_inf": &IfNaNOrInf{},
	"nan_if":        &NaNIf{},
	"neginf_if":     &NegInfIf{},
	"posinf_if":     &PosInfIf{},
	"ifinf":         &IfInf{},
	"ifnan":         &IfNaN{},
	"ifnanorinf":    &IfNaNOrInf{},
	"nanif":         &NaNIf{},
	"neginfif":      &NegInfIf{},
	"posinfif":      &PosInfIf{},

	// Flow Control
	"abort": &Abort{},

	// Meta
	"meta":          &Meta{},
	"min_version":   &MinVersion{},
	"self":          &Self{},
	"uuid":          &Uuid{},
	"version":       &Version{},
	"current_users": &CurrentUsers{},
	"ds_version":    &DsVersion{},

	// Distributed
	"node_name": &NodeName{},

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
	"decode":     &Decode{},

	// Unnest
	"unnest_position": &UnnestPosition{},
	"unnest_pos":      &UnnestPosition{},

	// Index Advisor
	"advisor": &Advisor{},
}
