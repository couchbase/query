//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package expression

import (
	"math"
	"strings"

	"github.com/couchbase/query/errors"
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

/*
This method takes in two values and returns new value that returns a boolean
value that depicts if the second value is contained within the first. If
either of the input values are missing, return a missing value, and if they
arent strings then return a null value.
*/
func (this *Contains) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := strings.Contains(first.ToString(), second.ToString())
	return value.NewValue(rv), nil
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

/*
This method takes in an argument value and returns its length
as value. If the input type is missing return missing, and if
it is not string then return null value.
*/
func (this *Length) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := len(arg.ToString())
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

/*
This method takes in an argument value and returns a
lowercase string as value. If the input type is
missing return missing, and if it is not string then
return null value.
*/
func (this *Lower) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := strings.ToLower(arg.ToString())
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
	var s string
	null := false
	missing := false
	chars := _WHITESPACE

	for i, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			missing = true
		} else if arg.Type() != value.STRING {
			null = true
		} else if !null && !missing {
			if i == 0 {
				s = arg.ToString()
			} else if i == 1 {
				chars = arg
			}
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	rv := strings.TrimLeft(s, chars.ToString())
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
// Position0
//
///////////////////////////////////////////////////

/*
This represents the String function POSITION0(expr, substr).
It returns the first position of the substring within the
string, or -1. The position is 0-based.
*/
type Position0 struct {
	BinaryFunctionBase
}

func NewPosition0(first, second Expression) Function {
	rv := &Position0{
		*NewBinaryFunctionBase("position0", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Position0) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Position0) Type() value.Type { return value.NUMBER }

func (this *Position0) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	return strPositionApply(first, second, 0)
}

/*
Factory method pattern.
*/
func (this *Position0) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewPosition0(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// Position1
//
///////////////////////////////////////////////////

/*
This represents the String function POSITION0(expr, substr).
It returns the first position of the substring within the
string, or -1. The position is 1-based.
*/
type Position1 struct {
	BinaryFunctionBase
}

func NewPosition1(first, second Expression) Function {
	rv := &Position1{
		*NewBinaryFunctionBase("position1", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Position1) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Position1) Type() value.Type { return value.NUMBER }

func (this *Position1) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	return strPositionApply(first, second, 1)
}

/*
Factory method pattern.
*/
func (this *Position1) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewPosition1(operands[0], operands[1])
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
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	nf := second.Actual().(float64)
	if nf < 0.0 || nf != math.Trunc(nf) {
		return value.NULL_VALUE, nil
	}

	ni := int(nf)
	if ni > RANGE_LIMIT {
		return nil, errors.NewRangeError("REPEAT()")
	}

	rv := strings.Repeat(first.ToString(), ni)
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
	var f, s, r string
	n := -1
	null := false
	missing := false

	for i, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			missing = true
		} else if i < 3 && arg.Type() != value.STRING {
			null = true
		} else if i == 3 && arg.Type() != value.NUMBER {
			null = true
		} else if !null && !missing {
			switch i {
			case 0:
				f = arg.ToString()
			case 1:
				s = arg.ToString()
			case 2:
				r = arg.ToString()
			case 3:
				nf := arg.Actual().(float64)
				if nf != math.Trunc(nf) {
					null = true
				} else {
					n = int(nf)
				}
			}

		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
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
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	s := arg.ToString()
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
	var s string
	chars := _WHITESPACE
	null := false
	missing := false

	for i, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			missing = true
		} else if arg.Type() != value.STRING {
			null = true
		} else if !null && !missing {
			if i == 0 {
				s = arg.ToString()
			} else if i == 1 {
				chars = arg
			}
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	rv := strings.TrimRight(s, chars.ToString())
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
	var s, sep value.Value
	null := false
	missing := false

	for i, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			missing = true
		} else if arg.Type() != value.STRING {
			null = true
		} else if !null && !missing {
			if i == 0 {
				s = arg
			} else if i == 1 {
				sep = arg
			}
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null || s == nil {
		return value.NULL_VALUE, nil
	}
	var sa []string
	if sep == nil {
		sa = strings.Fields(s.ToString())
	} else {
		sa = strings.Split(s.ToString(), sep.ToString())
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
// Substr0 / Substr
//
///////////////////////////////////////////////////

/*
This represents the String function SUBSTR(expr, position [, length ]).
It returns a substring from the integer position of the given length,
or to the end of the string. The position is 0-based, i.e. the first
position is 0. If position is negative, it is counted from the end
of the string; -1 is the last position in the string.
*/
type Substr0 struct {
	FunctionBase
}

func NewSubstr0(operands ...Expression) Function {
	rv := &Substr0{
		*NewFunctionBase("substr0", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Substr0) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Substr0) Type() value.Type { return value.STRING }

func (this *Substr0) Evaluate(item value.Value, context Context) (value.Value, error) {
	args := _ARGS_POOL.GetSized(len(this.operands))
	defer _ARGS_POOL.Put(args)
	for i, op := range this.operands {
		var err error
		args[i], err = op.Evaluate(item, context)
		if err != nil {
			return nil, err
		}
	}
	return strSubstrApply(args, 0)
}

/*
Minimum input arguments required for the SUBSTR function
is 2.
*/
func (this *Substr0) MinArgs() int { return 2 }

/*
Maximum input arguments required for the SUBSTR function
is 3.
*/
func (this *Substr0) MaxArgs() int { return 3 }

/*
Factory method pattern.
*/
func (this *Substr0) Constructor() FunctionConstructor {
	return NewSubstr0
}

///////////////////////////////////////////////////
//
// Substr1
//
///////////////////////////////////////////////////

/*
This represents the String function SUBSTR1(expr, position [, length ]).
It returns a substring from the integer position of the given length,
or to the end of the string. The position is 1-based, i.e. the first
position is 0. If position is negative, it is counted from the end
of the string; -1 is the last position in the string.
*/
type Substr1 struct {
	FunctionBase
}

func NewSubstr1(operands ...Expression) Function {
	rv := &Substr1{
		*NewFunctionBase("substr1", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Substr1) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Substr1) Type() value.Type { return value.STRING }

func (this *Substr1) Evaluate(item value.Value, context Context) (value.Value, error) {
	args := _ARGS_POOL.GetSized(len(this.operands))
	defer _ARGS_POOL.Put(args)
	for i, op := range this.operands {
		var err error
		args[i], err = op.Evaluate(item, context)
		if err != nil {
			return nil, err
		}
	}
	return strSubstrApply(args, 1)
}

/*
Minimum input arguments required for the SUBSTR function
is 2.
*/
func (this *Substr1) MinArgs() int { return 2 }

/*
Maximum input arguments required for the SUBSTR function
is 3.
*/
func (this *Substr1) MaxArgs() int { return 3 }

/*
Factory method pattern.
*/
func (this *Substr1) Constructor() FunctionConstructor {
	return NewSubstr1
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
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	s := arg.ToString()
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
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	av := arg.ToString()
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
	var s string
	null := false
	missing := false
	chars := _WHITESPACE

	for i, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			missing = true
		} else if arg.Type() != value.STRING {
			null = true
		} else if !null {
			if i == 0 {
				s = arg.ToString()
			} else if i == 1 {
				chars = arg
			}
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	rv := strings.Trim(s, chars.ToString())
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
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := strings.ToUpper(arg.ToString())
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

func strPositionApply(first, second value.Value, startPos int) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := strings.Index(first.ToString(), second.ToString())
	return value.NewValue(rv + startPos), nil
}

func strSubstrApply(args []value.Value, startPos int) (value.Value, error) {
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

	str := args[0].ToString()
	pos := int(args[1].Actual().(float64))

	if pos < 0 {
		pos = len(str) + pos
	} else if pos > 0 && startPos > 0 {
		pos = pos - startPos
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
