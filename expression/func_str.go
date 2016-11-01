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
	"math"
	"strings"

	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// Contains
//
///////////////////////////////////////////////////

/*
This represents the String function CONTAINS(expr, substr).
It returns true if the string contains the substring.
*/
type Contains struct {
	BinaryFunctionBase
}

func NewContains(first, second Expression) Function {
	rv := &Contains{
		*NewBinaryFunctionBase("contains", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Contains) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Contains) Type() value.Type { return value.BOOLEAN }

func (this *Contains) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For boolean functions, simply list this expression.
*/
func (this *Contains) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

/*
This method takes in two values and returns new value that returns a boolean
value that depicts if the second value is contained within the first. If
either of the input values are missing, return a missing value, and if they
arent strings then return a null value.
*/
func (this *Contains) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := strings.Contains(first.Actual().(string), second.Actual().(string))
	return value.NewValue(rv), nil
}

/*
Factory method pattern.
*/
func (this *Contains) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewContains(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// Length
//
///////////////////////////////////////////////////

/*
This represents the String function LENGTH(expr). It
returns the length of the string value.
*/
type Length struct {
	UnaryFunctionBase
}

func NewLength(operand Expression) Function {
	rv := &Length{
		*NewUnaryFunctionBase("length", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Length) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Length) Type() value.Type { return value.NUMBER }

func (this *Length) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an argument value and returns its length
as value. If the input type is missing return missing, and if
it is not string then return null value.
*/
func (this *Length) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := len(arg.Actual().(string))
	return value.NewValue(rv), nil
}

/*
Factory method pattern.
*/
func (this *Length) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLength(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Lower
//
///////////////////////////////////////////////////

/*
This represents the String function LOWER(expr). It returns
the lowercase of the string value.
*/
type Lower struct {
	UnaryFunctionBase
}

func NewLower(operand Expression) Function {
	rv := &Lower{
		*NewUnaryFunctionBase("lower", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Lower) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Lower) Type() value.Type { return value.STRING }

func (this *Lower) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an argument value and returns a
lowercase string as value. If the input type is
missing return missing, and if it is not string then
return null value.
*/
func (this *Lower) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := strings.ToLower(arg.Actual().(string))
	return value.NewValue(rv), nil
}

/*
Factory method pattern.
*/
func (this *Lower) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLower(operands[0])
	}
}

///////////////////////////////////////////////////
//
// LTrim
//
///////////////////////////////////////////////////

/*
This represents the String function LTRIM(expr [, chars ]).
It returns a string with all leading chars removed
(whitespace by default).
*/
type LTrim struct {
	FunctionBase
}

func NewLTrim(operands ...Expression) Function {
	rv := &LTrim{
		*NewFunctionBase("ltrim", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *LTrim) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *LTrim) Type() value.Type { return value.STRING }

func (this *LTrim) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *LTrim) Apply(context Context, args ...value.Value) (value.Value, error) {
	null := false

	for _, a := range args {
		if a.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if a.Type() != value.STRING {
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	chars := _WHITESPACE
	if len(args) > 1 {
		chars = args[1]
	}

	rv := strings.TrimLeft(args[0].Actual().(string), chars.Actual().(string))
	return value.NewValue(rv), nil
}

/*
Minimum input arguments required for the LTRIM function
is 1.
*/
func (this *LTrim) MinArgs() int { return 1 }

/*
Maximum input arguments required for the LTRIM function
is 2.
*/
func (this *LTrim) MaxArgs() int { return 2 }

/*
Factory method pattern.
*/
func (this *LTrim) Constructor() FunctionConstructor {
	return NewLTrim
}

/*
Define variable whitespace that constructs a value from
' ','\t','\n','\f' and '\r'.
*/
var _WHITESPACE = value.NewValue(" \t\n\f\r")

///////////////////////////////////////////////////
//
// Position
//
///////////////////////////////////////////////////

/*
This represents the String function POSITION(expr, substr).
It returns the first position of the substring within the
string, or -1. The position is 0-based.
*/
type Position struct {
	BinaryFunctionBase
}

func NewPosition(first, second Expression) Function {
	rv := &Position{
		*NewBinaryFunctionBase("position", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Position) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Position) Type() value.Type { return value.NUMBER }

func (this *Position) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *Position) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := strings.Index(first.Actual().(string), second.Actual().(string))
	return value.NewValue(rv), nil
}

/*
Factory method pattern.
*/
func (this *Position) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewPosition(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// Repeat
//
///////////////////////////////////////////////////

/*
This represents the String function REPEAT(expr, n).
It returns string formed by repeating expr n times.
*/
type Repeat struct {
	BinaryFunctionBase
}

func NewRepeat(first, second Expression) Function {
	rv := &Repeat{
		*NewBinaryFunctionBase("repeat", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Repeat) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Repeat) Type() value.Type { return value.STRING }

func (this *Repeat) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *Repeat) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	nf := second.Actual().(float64)
	if nf < 0.0 || nf != math.Trunc(nf) {
		return value.NULL_VALUE, nil
	}

	rv := strings.Repeat(first.Actual().(string), int(nf))
	return value.NewValue(rv), nil
}

/*
Factory method pattern.
*/
func (this *Repeat) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewRepeat(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// Replace
//
///////////////////////////////////////////////////

/*
This represents the String function REPLACE(expr, substr, repl [, n ]).
It returns a string with all occurences of substr replaced with repl.
If n is given, at most n replacements are performed.
*/
type Replace struct {
	FunctionBase
}

func NewReplace(operands ...Expression) Function {
	rv := &Replace{
		*NewFunctionBase("replace", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Replace) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Replace) Type() value.Type { return value.STRING }

func (this *Replace) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *Replace) Apply(context Context, args ...value.Value) (value.Value, error) {
	null := false

	for i := 0; i < 3; i++ {
		if args[i].Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if args[i].Type() != value.STRING {
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	if len(args) == 4 && args[3].Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	f := args[0].Actual().(string)
	s := args[1].Actual().(string)
	r := args[2].Actual().(string)
	n := -1

	if len(args) == 4 {
		nf := args[3].Actual().(float64)
		if nf != math.Trunc(nf) {
			return value.NULL_VALUE, nil
		}

		n = int(nf)
	}

	rv := strings.Replace(f, s, r, n)
	return value.NewValue(rv), nil
}

/*
Minimum input arguments required for the REPLACE function
is 3.
*/
func (this *Replace) MinArgs() int { return 3 }

/*
Maximum input arguments required for the REPLACE function
is 4.
*/
func (this *Replace) MaxArgs() int { return 4 }

/*
Factory method pattern.
*/
func (this *Replace) Constructor() FunctionConstructor {
	return NewReplace
}

///////////////////////////////////////////////////
//
// Reverse
//
///////////////////////////////////////////////////

/*
This represents the string function REVERSE(expr). It returns the
reverse order of the unicode characters of the string value.
*/
type Reverse struct {
	UnaryFunctionBase
}

func NewReverse(operand Expression) Function {
	rv := &Reverse{
		*NewUnaryFunctionBase("reverse", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Reverse) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Reverse) Type() value.Type { return value.STRING }

func (this *Reverse) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Reverse) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	s := arg.Actual().(string)
	r := util.ReversePreservingCombiningCharacters(s)
	return value.NewValue(r), nil
}

/*
Factory method pattern.
*/
func (this *Reverse) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewReverse(operands[0])
	}
}

///////////////////////////////////////////////////
//
// RTrim
//
///////////////////////////////////////////////////

/*
This represents the String function RTRIM(expr, [, chars ]).
It returns a string with all trailing chars removed (whitespace
by default).
*/
type RTrim struct {
	FunctionBase
}

func NewRTrim(operands ...Expression) Function {
	rv := &RTrim{
		*NewFunctionBase("rtrim", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *RTrim) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *RTrim) Type() value.Type { return value.STRING }

func (this *RTrim) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *RTrim) Apply(context Context, args ...value.Value) (value.Value, error) {
	null := false

	for _, a := range args {
		if a.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if a.Type() != value.STRING {
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	chars := _WHITESPACE
	if len(args) > 1 {
		chars = args[1]
	}

	rv := strings.TrimRight(args[0].Actual().(string), chars.Actual().(string))
	return value.NewValue(rv), nil
}

/*
Minimum input arguments required for the RTRIM function
is 1.
*/
func (this *RTrim) MinArgs() int { return 1 }

/*
Maximum input arguments required for the RTRIM function
is 2.
*/
func (this *RTrim) MaxArgs() int { return 2 }

/*
Factory method pattern.
*/
func (this *RTrim) Constructor() FunctionConstructor {
	return NewRTrim
}

///////////////////////////////////////////////////
//
// Split
//
///////////////////////////////////////////////////

/*
This represents the String function SPLIT(expr [, sep ]).
It splits the string into an array of substrings separated
by sep. If sep is not given, any combination of whitespace
characters is used.
*/
type Split struct {
	FunctionBase
}

func NewSplit(operands ...Expression) Function {
	rv := &Split{
		*NewFunctionBase("split", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Split) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Split) Type() value.Type { return value.ARRAY }

func (this *Split) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *Split) Apply(context Context, args ...value.Value) (value.Value, error) {
	null := false

	for _, a := range args {
		if a.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if a.Type() != value.STRING {
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	var sa []string
	if len(args) > 1 {
		sep := args[1]
		sa = strings.Split(args[0].Actual().(string),
			sep.Actual().(string))
	} else {
		sa = strings.Fields(args[0].Actual().(string))
	}

	rv := make([]interface{}, len(sa))
	for i, s := range sa {
		rv[i] = s
	}

	return value.NewValue(rv), nil
}

/*
Minimum input arguments required for the SPLIT function
is 1.
*/
func (this *Split) MinArgs() int { return 1 }

/*
Maximum input arguments required for the SPLIT function
is 2.
*/
func (this *Split) MaxArgs() int { return 2 }

/*
Factory method pattern.
*/
func (this *Split) Constructor() FunctionConstructor {
	return NewSplit
}

///////////////////////////////////////////////////
//
// Substr
//
///////////////////////////////////////////////////

/*
This represents the String function SUBSTR(expr, position [, length ]).
It returns a substring from the integer position of the given length,
or to the end of the string. The position is 0-based, i.e. the first
position is 0. If position is negative, it is counted from the end
of the string; -1 is the last position in the string.
*/
type Substr struct {
	FunctionBase
}

func NewSubstr(operands ...Expression) Function {
	rv := &Substr{
		*NewFunctionBase("substr", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Substr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Substr) Type() value.Type { return value.STRING }

func (this *Substr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *Substr) Apply(context Context, args ...value.Value) (value.Value, error) {
	null := false

	if args[0].Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if args[0].Type() != value.STRING {
		null = true
	}

	for i := 1; i < len(args); i++ {
		switch args[i].Type() {
		case value.MISSING:
			return value.MISSING_VALUE, nil
		case value.NUMBER:
			vf := args[i].Actual().(float64)
			if vf != math.Trunc(vf) {
				null = true
			}
		default:
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	str := args[0].Actual().(string)
	pos := int(args[1].Actual().(float64))

	if pos < 0 {
		pos = len(str) + pos
	}

	if pos < 0 || pos >= len(str) {
		return value.NULL_VALUE, nil
	}

	if len(args) == 2 {
		return value.NewValue(str[pos:]), nil
	}

	length := int(args[2].Actual().(float64))
	if length < 0 {
		return value.NULL_VALUE, nil
	}

	if pos+length > len(str) {
		length = len(str) - pos
	}

	return value.NewValue(str[pos : pos+length]), nil
}

/*
Minimum input arguments required for the SUBSTR function
is 2.
*/
func (this *Substr) MinArgs() int { return 2 }

/*
Maximum input arguments required for the SUBSTR function
is 3.
*/
func (this *Substr) MaxArgs() int { return 3 }

/*
Factory method pattern.
*/
func (this *Substr) Constructor() FunctionConstructor {
	return NewSubstr
}

///////////////////////////////////////////////////
//
// Suffixes
//
///////////////////////////////////////////////////

/*
This represents the String function SUFFIXES(expr). It returns an
array of all the suffixes of the string value.
*/
type Suffixes struct {
	UnaryFunctionBase
}

func NewSuffixes(operand Expression) Function {
	rv := &Suffixes{
		*NewUnaryFunctionBase("suffixes", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Suffixes) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Suffixes) Type() value.Type { return value.ARRAY }

func (this *Suffixes) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Suffixes) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	s := arg.Actual().(string)
	rv := make([]interface{}, 0, len(s))
	// Range over Unicode code points, not bytes
	for i, _ := range s {
		rv = append(rv, s[i:])
	}

	return value.NewValue(rv), nil
}

/*
Factory method pattern.
*/
func (this *Suffixes) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewSuffixes(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Title
//
///////////////////////////////////////////////////

/*
This represents the String function TITLE(expr). It converts
the string so that the first letter of each word is uppercase
and every other letter is lowercase.
*/
type Title struct {
	UnaryFunctionBase
}

func NewTitle(operand Expression) Function {
	rv := &Title{
		*NewUnaryFunctionBase("title", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Title) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Title) Type() value.Type { return value.STRING }

func (this *Title) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Title) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	av := arg.Actual().(string)
	rv := strings.Title(strings.ToLower(av))
	return value.NewValue(rv), nil
}

/*
Factory method pattern.
*/
func (this *Title) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewTitle(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Trim
//
///////////////////////////////////////////////////

/*
This represents the String function TRIM(expr [, chars ]).
It returns a string with all leading and trailing chars
removed (whitespace by default).
*/
type Trim struct {
	FunctionBase
}

func NewTrim(operands ...Expression) Function {
	rv := &Trim{
		*NewFunctionBase("trim", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Trim) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Trim) Type() value.Type { return value.STRING }

func (this *Trim) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *Trim) Apply(context Context, args ...value.Value) (value.Value, error) {
	null := false

	for _, a := range args {
		if a.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if a.Type() != value.STRING {
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	chars := _WHITESPACE
	if len(args) > 1 {
		chars = args[1]
	}

	rv := strings.Trim(args[0].Actual().(string), chars.Actual().(string))
	return value.NewValue(rv), nil
}

/*
Minimum input arguments required for the TRIM function
is 1.
*/
func (this *Trim) MinArgs() int { return 1 }

/*
Maximum input arguments required for the TRIM function
is 2.
*/
func (this *Trim) MaxArgs() int { return 2 }

/*
Factory method pattern.
*/
func (this *Trim) Constructor() FunctionConstructor {
	return NewTrim
}

///////////////////////////////////////////////////
//
// Upper
//
///////////////////////////////////////////////////

/*
This represents the String function UPPER(expr). It returns
the uppercase of the string value.
*/
type Upper struct {
	UnaryFunctionBase
}

func NewUpper(operand Expression) Function {
	rv := &Upper{
		*NewUnaryFunctionBase("upper", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Upper) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Upper) Type() value.Type { return value.STRING }

func (this *Upper) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Upper) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := strings.ToUpper(arg.Actual().(string))
	return value.NewValue(rv), nil
}

/*
Factory method pattern.
*/
func (this *Upper) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewUpper(operands[0])
	}
}
