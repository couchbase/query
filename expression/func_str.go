//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

// CONTAINS(expr, substr).  Returns true if the string contains the substring.

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

func (this *Contains) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Contains) Type() value.Type { return value.BOOLEAN }

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

// If this expression is in the WHERE clause of a partial index, lists the Expressions that are implicitly covered.
// For boolean functions, simply list this expression.
func (this *Contains) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *Contains) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	covers[this] = value.TRUE_VALUE
	return covers
}

func (this *Contains) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewContains(operands[0], operands[1])
	}
}

// LENGTH(expr). Returns the length of the string value.

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

func (this *Length) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Length) Type() value.Type { return value.NUMBER }

func (this *Length) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}
	return value.NewValue(len(arg.ToString())), nil
}

func (this *Length) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLength(operands[0])
	}
}

// Multi-byte aware variant

type MBLength struct {
	UnaryFunctionBase
}

func NewMBLength(operand Expression) Function {
	rv := &MBLength{
		*NewUnaryFunctionBase("mb_length", operand),
	}
	rv.expr = rv
	return rv
}

func (this *MBLength) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *MBLength) Type() value.Type { return value.NUMBER }

func (this *MBLength) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := utf8.RuneCountInString(arg.ToString())
	return value.NewValue(rv), nil
}

func (this *MBLength) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewMBLength(operands[0])
	}
}

// LOWER(expr). Returns the input string with all characters converted to lowercase.

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

func (this *Lower) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Lower) Type() value.Type { return value.STRING }

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

func (this *Lower) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLower(operands[0])
	}
}

// LTRIM(expr [, chars ]).  Returns a string with all leading <chars> (whitespace by default) removed.

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

func (this *LTrim) MinArgs() int { return 1 }

func (this *LTrim) MaxArgs() int { return 2 }

func (this *LTrim) Constructor() FunctionConstructor {
	return NewLTrim
}

var _WHITESPACE = value.NewValue(" \t\n\f\r")

// POSITION0(expr, substr).  Returns the first position of the substring within the string, or -1. The position is 0-based.

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
	return strPositionApply(false, first, second, 0)
}

func (this *Position0) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewPosition0(operands[0], operands[1])
	}
}

// Multi-byte aware variant

type MBPosition0 struct {
	BinaryFunctionBase
}

func NewMBPosition0(first, second Expression) Function {
	rv := &MBPosition0{
		*NewBinaryFunctionBase("mb_position0", first, second),
	}

	rv.expr = rv
	return rv
}

func (this *MBPosition0) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *MBPosition0) Type() value.Type { return value.NUMBER }

func (this *MBPosition0) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	return strPositionApply(true, first, second, 0)
}

func (this *MBPosition0) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewMBPosition0(operands[0], operands[1])
	}
}

// Same as Position0 but the returned index is 1-based.

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
	return strPositionApply(false, first, second, 1)
}

func (this *Position1) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewPosition1(operands[0], operands[1])
	}
}

// Multi-byte aware variant

type MBPosition1 struct {
	BinaryFunctionBase
}

func NewMBPosition1(first, second Expression) Function {
	rv := &MBPosition1{
		*NewBinaryFunctionBase("mb_position1", first, second),
	}

	rv.expr = rv
	return rv
}

func (this *MBPosition1) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *MBPosition1) Type() value.Type { return value.NUMBER }

func (this *MBPosition1) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	return strPositionApply(true, first, second, 1)
}

func (this *MBPosition1) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewMBPosition1(operands[0], operands[1])
	}
}

func strPositionApply(inRunes bool, first, second value.Value, startPos int) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	var rv int
	s := first.ToString()
	if inRunes {
		rv = util.RuneIndex(s, second.ToString())
	} else {
		rv = strings.Index(s, second.ToString())
	}
	return value.NewValue(rv + startPos), nil
}

// REPEAT(expr, n).  Returns string formed by repeating <expr> n times.

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

	sz := uint64(len(first.ToString())) * uint64(ni)
	err = checkSizeWithinLimit(fmt.Sprintf("%s()", this.name), context, sz/uint64(ni), ni, sz, 20*util.MiB)
	if err != nil {
		return nil, err
	}

	rv := strings.Repeat(first.ToString(), ni)
	return value.NewValue(rv), nil
}

func (this *Repeat) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewRepeat(operands[0], operands[1])
	}
}

// REPLACE(expr, substr, repl [, n ]).  Replace <n> (default all) occurrences of <substr> in <expr> with <repl>.

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

func (this *Replace) MinArgs() int { return 3 }

func (this *Replace) MaxArgs() int { return 4 }

func (this *Replace) Constructor() FunctionConstructor {
	return NewReplace
}

// REVERSE(expr). Returns the string in reverse _character_ order.

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

func (this *Reverse) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewReverse(operands[0])
	}
}

// RTRIM(expr, [, chars ]).  Returns a string with all trailing <chars> (whitespace by default) removed.

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

func (this *RTrim) MinArgs() int { return 1 }

func (this *RTrim) MaxArgs() int { return 2 }

func (this *RTrim) Constructor() FunctionConstructor {
	return NewRTrim
}

// SPLIT(expr [, sep ]).  Split a string into an array of substrings separated by <sep> (default adjacent whitespace).

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

func (this *Split) MinArgs() int { return 1 }

func (this *Split) MaxArgs() int { return 2 }

func (this *Split) Constructor() FunctionConstructor {
	return NewSplit
}

// SUBSTR(expr, position [, length ]).  Return a <length>-character (default: all remaining) substring starting at <position>.
// <position> is 0-based and if negative, it is taken as a backwards count from the end of the string.

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
	return strSubstrApply(false, args, 0)
}

func (this *Substr0) MinArgs() int { return 2 }

func (this *Substr0) MaxArgs() int { return 3 }

func (this *Substr0) Constructor() FunctionConstructor {
	return NewSubstr0
}

// Multi-byte aware variant

type MBSubstr0 struct {
	FunctionBase
}

func NewMBSubstr0(operands ...Expression) Function {
	rv := &MBSubstr0{
		*NewFunctionBase("mb_substr0", operands...),
	}

	rv.expr = rv
	return rv
}

func (this *MBSubstr0) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *MBSubstr0) Type() value.Type { return value.STRING }

func (this *MBSubstr0) Evaluate(item value.Value, context Context) (value.Value, error) {
	args := _ARGS_POOL.GetSized(len(this.operands))
	defer _ARGS_POOL.Put(args)
	for i, op := range this.operands {
		var err error
		args[i], err = op.Evaluate(item, context)
		if err != nil {
			return nil, err
		}
	}
	return strSubstrApply(true, args, 0)
}

func (this *MBSubstr0) MinArgs() int { return 2 }

func (this *MBSubstr0) MaxArgs() int { return 3 }

func (this *MBSubstr0) Constructor() FunctionConstructor {
	return NewMBSubstr0
}

// Like Substr0 but <position> is 1-based.

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
	return strSubstrApply(false, args, 1)
}

func (this *Substr1) MinArgs() int { return 2 }

func (this *Substr1) MaxArgs() int { return 3 }

func (this *Substr1) Constructor() FunctionConstructor {
	return NewSubstr1
}

// Multi-byte aware variant

type MBSubstr1 struct {
	FunctionBase
}

func NewMBSubstr1(operands ...Expression) Function {
	rv := &MBSubstr1{
		*NewFunctionBase("mb_substr1", operands...),
	}

	rv.expr = rv
	return rv
}

func (this *MBSubstr1) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *MBSubstr1) Type() value.Type { return value.STRING }

func (this *MBSubstr1) Evaluate(item value.Value, context Context) (value.Value, error) {
	args := _ARGS_POOL.GetSized(len(this.operands))
	defer _ARGS_POOL.Put(args)
	for i, op := range this.operands {
		var err error
		args[i], err = op.Evaluate(item, context)
		if err != nil {
			return nil, err
		}
	}
	return strSubstrApply(true, args, 1)
}

func (this *MBSubstr1) MinArgs() int { return 2 }

func (this *MBSubstr1) MaxArgs() int { return 3 }

func (this *MBSubstr1) Constructor() FunctionConstructor {
	return NewMBSubstr1
}

func strSubstrApply(inRunes bool, args []value.Value, startPos int) (value.Value, error) {
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
	var l int
	if inRunes {
		l = utf8.RuneCountInString(str)
	} else {
		l = len(str)
	}

	if pos < 0 {
		pos = l + pos
	} else if pos > 0 && startPos > 0 {
		pos = pos - startPos
	}

	if pos < 0 || pos >= l {
		return value.NULL_VALUE, nil
	}

	if inRunes {
		if len(args) == 2 {
			return value.NewValue(util.SubStringRune(str, pos, -1)), nil
		}
		length := int(args[2].Actual().(float64))
		if length < 0 {
			return value.NULL_VALUE, nil
		}

		if pos+length > l {
			length = l - pos
		}

		return value.NewValue(util.SubStringRune(str, pos, length)), nil

	} else {
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

}

// SUFFIXES(expr). Return an array containing all the suffixes of the string value.

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

func (this *Suffixes) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewSuffixes(operands[0])
	}
}

// TITLE(expr). Converts the string so that the first letter of each word is uppercase and every other letter is lowercase.

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

func (this *Title) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewTitle(operands[0])
	}
}

// TRIM(expr [, chars ]).  Return a string with all leading and trailing <chars> (whitespace by default) removed.

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

func (this *Trim) MinArgs() int { return 1 }

func (this *Trim) MaxArgs() int { return 2 }

func (this *Trim) Constructor() FunctionConstructor {
	return NewTrim
}

// UPPER(expr). Returns the input string with all characters converted to uppercase.

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

func (this *Upper) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewUpper(operands[0])
	}
}

// MASK(expr,options)  Apply a mask to a string.

type Mask struct {
	FunctionBase
}

func NewMask(operands ...Expression) Function {
	rv := &Mask{
		*NewFunctionBase("mask", operands...),
	}

	rv.expr = rv
	return rv
}

func (this *Mask) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Mask) Type() value.Type { return value.STRING }

type _AnchorType int

const (
	_START _AnchorType = iota
	_END
	_TEXT
	_POSITION
)

func (this *Mask) Evaluate(item value.Value, context Context) (value.Value, error) {
	var s string
	null := false
	missing := false

	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		missing = true
	} else if arg.Type() != value.STRING {
		null = true
	} else {
		s = arg.ToString()
	}

	var mask string
	hole := " "
	inject := ""
	preserve := false

	anchorType := _START
	var anchorPos int
	var anchorRe *regexp.Regexp

	if len(this.operands) > 1 {
		options, err := this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if options.Type() == value.MISSING {
			missing = true
		} else if options.Type() != value.OBJECT {
			null = true
		} else if !null && !missing {

			if m, ok := options.Field("mask"); ok && m.Type() == value.STRING {
				mask = m.Actual().(string)
			}

			if c, ok := options.Field("hole"); ok && c.Type() == value.STRING {
				hole = c.Actual().(string)
			}

			if c, ok := options.Field("inject"); ok && c.Type() == value.STRING {
				inject = c.Actual().(string)
			}

			if r, ok := options.Field("anchor"); ok {
				switch r.Type() {
				case value.NUMBER:
					anchorType = _POSITION
					anchorPos = int(r.(value.NumberValue).Int64())
					anchorRe = nil
				case value.STRING:
					p := r.Actual().(string)
					if strings.ToLower(p) == "start" {
						anchorType = _START
						anchorRe = nil
					} else if strings.ToLower(p) == "end" {
						anchorType = _END
						anchorRe = nil
					} else {
						anchorType = _TEXT
						anchorRe, err = regexp.Compile(r.Actual().(string))
						if err != nil {
							return nil, err
						}
					}
					anchorPos = 0
				default:
					anchorType = _START
					anchorRe = nil
					anchorPos = 0
				}
			}

			if r, ok := options.Field("length"); ok && r.Type() == value.STRING {
				if strings.ToLower(r.Actual().(string)) == "source" {
					preserve = true
				} else {
					preserve = false
				}
			}
		}
	}
	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	if len(mask) == 0 {
		mask = "********"
	}

	if anchorType == _TEXT {
		m := anchorRe.FindStringIndex(s)
		if m == nil {
			return value.NewValue(s), nil
		}
		anchorPos = util.ByteIndexToRuneIndex(s, m[0])
	}

	var l int
	if preserve {
		l = utf8.RuneCountInString(s)
	} else {
		l = utf8.RuneCountInString(mask)
	}

	if preserve {
		for _, mc := range mask {
			if strings.ContainsRune(inject, mc) {
				l++
			}
		}
	}

	right := anchorType == _END || anchorPos < 0
	if anchorPos < 0 {
		anchorPos *= -1
	}

	if anchorPos > utf8.RuneCountInString(s) {
		return value.NewValue(s), nil
	}

	if !preserve {
		l += anchorPos
	}
	rv := make([]rune, l)
	mr := getReader(mask, right)
	sr := getReader(s, right)

	i := 0

	body := func() {
		mc, _, e := mr.ReadRune()
		if e == nil {
			if strings.ContainsRune(hole, mc) {
				sc, _, e := sr.ReadRune()
				if e == nil {
					rv[i] = sc
				} else {
					rv[i] = mc
				}
			} else if strings.ContainsRune(inject, mc) {
				rv[i] = mc
			} else {
				rv[i] = mc
				_, _, _ = sr.ReadRune()
			}
		} else {
			sc, _, e := sr.ReadRune()
			if e == nil {
				rv[i] = sc
			}
		}
	}

	if !right {
		i = 0
		for ; anchorPos > 0 && i < l; anchorPos-- {
			sc, _, e := sr.ReadRune()
			if e != nil {
				break
			}
			rv[i] = sc
			i++
		}
		for ; i < l; i++ {
			body()
		}
	} else {
		i = l - 1
		for ; anchorPos > 0 && i >= 0; anchorPos-- {
			sc, _, e := sr.ReadRune()
			if e != nil {
				break
			}
			rv[i] = sc
			i--
		}
		for ; i >= 0; i-- {
			body()
		}
	}

	return value.NewValue(rv), nil
}

func getReader(s string, reverse bool) *strings.Reader {
	if reverse {
		return strings.NewReader(util.ReversePreservingCombiningCharacters(s))
	} else {
		return strings.NewReader(s)
	}
}

func (this *Mask) MinArgs() int { return 1 }

func (this *Mask) MaxArgs() int { return 2 }

func (this *Mask) Constructor() FunctionConstructor {
	return NewMask
}

// LPAD(expr, size [,str])  Pad a string to length on the left (start) using <str> or spaces if <str> is not supplied.

type LPad struct {
	FunctionBase
}

func NewLPad(operands ...Expression) Function {
	rv := &LPad{
		*NewFunctionBase("lpad", operands...),
	}

	rv.expr = rv
	return rv
}

func (this *LPad) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *LPad) Type() value.Type { return value.STRING }

func (this *LPad) Evaluate(item value.Value, context Context) (value.Value, error) {
	return padString(item, context, this.operands, false, false)
}

func (this *LPad) MinArgs() int { return 2 }

func (this *LPad) MaxArgs() int { return 3 }

func (this *LPad) Constructor() FunctionConstructor {
	return NewLPad
}

// Multi-byte aware variant

type MBLPad struct {
	FunctionBase
}

func NewMBLPad(operands ...Expression) Function {
	rv := &MBLPad{
		*NewFunctionBase("mb_lpad", operands...),
	}

	rv.expr = rv
	return rv
}

func (this *MBLPad) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *MBLPad) Type() value.Type { return value.STRING }

func (this *MBLPad) Evaluate(item value.Value, context Context) (value.Value, error) {
	return padString(item, context, this.operands, false, true)
}

func (this *MBLPad) MinArgs() int { return 2 }

func (this *MBLPad) MaxArgs() int { return 3 }

func (this *MBLPad) Constructor() FunctionConstructor {
	return NewMBLPad
}

// RPAD(expr, size [,str])  Pad a string to length on the right (end) using <str> or spaces if <str> is not supplied.

type RPad struct {
	FunctionBase
}

func NewRPad(operands ...Expression) Function {
	rv := &RPad{
		*NewFunctionBase("rpad", operands...),
	}

	rv.expr = rv
	return rv
}

func (this *RPad) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *RPad) Type() value.Type { return value.STRING }

func (this *RPad) Evaluate(item value.Value, context Context) (value.Value, error) {
	return padString(item, context, this.operands, true, false)
}

func (this *RPad) MinArgs() int { return 2 }

func (this *RPad) MaxArgs() int { return 3 }

func (this *RPad) Constructor() FunctionConstructor {
	return NewRPad
}

// Multi-byte aware variant

type MBRPad struct {
	FunctionBase
}

func NewMBRPad(operands ...Expression) Function {
	rv := &MBRPad{
		*NewFunctionBase("mb_rpad", operands...),
	}

	rv.expr = rv
	return rv
}

func (this *MBRPad) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *MBRPad) Type() value.Type { return value.STRING }

func (this *MBRPad) Evaluate(item value.Value, context Context) (value.Value, error) {
	return padString(item, context, this.operands, true, true)
}

func (this *MBRPad) MinArgs() int { return 2 }

func (this *MBRPad) MaxArgs() int { return 3 }

func (this *MBRPad) Constructor() FunctionConstructor {
	return NewMBRPad
}

// FORMALIZE(expr [,query_context]) Return the formalized (optionally using <query_context>) text of the input statment string.

type Formalize struct {
	FunctionBase
}

func NewFormalize(operands ...Expression) Function {
	rv := &Formalize{
		*NewFunctionBase("formalize", operands...),
	}

	rv.expr = rv
	return rv
}

func (this *Formalize) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Formalize) Type() value.Type { return value.NUMBER }

func (this *Formalize) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}
	qc := context.QueryContext()
	if len(this.operands) > 1 {
		qcarg, err := this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if arg.Type() != value.STRING {
			return value.NULL_VALUE, nil
		}
		qc = qcarg.ToString()
	}
	evalContext := context.NewQueryContext(qc, context.Readonly())

	s, err := evalContext.(Context).Parse(arg.ToString())
	if err != nil || s == nil {
		return value.NULL_VALUE, errors.NewParseSyntaxError(err, "Error formalizing statement")
	}
	if st, ok := s.(interface{ String() string }); ok {
		return value.NewValue(st.String()), nil
	}
	return value.NewValue(arg.ToString()), nil
}

func (this *Formalize) Constructor() FunctionConstructor {
	return NewFormalize
}

func (this *Formalize) MinArgs() int { return 1 }

func (this *Formalize) MaxArgs() int { return 2 }

// URLENCODE(expr) Return the string URL-encoded. (e.g. " " to "+")

type URLEncode struct {
	UnaryFunctionBase
}

func NewURLEncode(operand Expression) Function {
	rv := &URLEncode{
		*NewUnaryFunctionBase("urlencode", operand),
	}

	rv.expr = rv
	return rv
}

func (this *URLEncode) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *URLEncode) Type() value.Type { return value.STRING }

func (this *URLEncode) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := url.QueryEscape(arg.ToString())
	return value.NewValue(rv), nil
}

func (this *URLEncode) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewURLEncode(operands[0])
	}
}

// URLDECODE(expr)  Return the string URL-decoded. (e.g. "+" to " ")

type URLDecode struct {
	UnaryFunctionBase
}

func NewURLDecode(operand Expression) Function {
	rv := &URLDecode{
		*NewUnaryFunctionBase("urlencode", operand),
	}

	rv.expr = rv
	return rv
}

func (this *URLDecode) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *URLDecode) Type() value.Type { return value.STRING }

func (this *URLDecode) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv, err := url.QueryUnescape(arg.ToString())
	if err != nil {
		return value.NULL_VALUE, nil
	}
	return value.NewValue(rv), nil
}

func (this *URLDecode) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewURLDecode(operands[0])
	}
}

func padString(item value.Value, context Context, operands Expressions, right bool, inRunes bool) (value.Value, error) {
	var s string
	var l int
	pad := " "
	null := false
	missing := false

	for i, op := range operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			missing = true
		} else if (i == 0 || i == 2) && arg.Type() != value.STRING {
			null = true
		} else if i == 1 && arg.Type() != value.NUMBER {
			null = true
		} else if !null && !missing {
			switch i {
			case 0:
				s = arg.ToString()
			case 1:
				num := arg.Actual().(float64)
				if num < 0.0 || num != math.Trunc(num) {
					null = true
				} else {
					l = int(num)
				}
			case 2:
				pad = arg.ToString()
				if len(pad) < 1 {
					null = true
				}
			}
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	var d int
	if inRunes {
		d = l - utf8.RuneCountInString(s)
	} else {
		d = l - len(s)
	}
	if d <= 0 {
		if inRunes {
			return value.NewValue(util.SubStringRune(s, 0, l)), nil
		}
		return value.NewValue(s[:l]), nil
	}
	var padded strings.Builder
	if right {
		padded.WriteString(s)
	}
	var lp int
	if inRunes {
		lp = utf8.RuneCountInString(pad)
	} else {
		lp = len(pad)
	}
	for d > 0 {
		if lp < d {
			padded.WriteString(pad)
		} else if inRunes {
			padded.WriteString(util.SubStringRune(pad, 0, d))
		} else {
			padded.WriteString(pad[:d])
		}
		d -= lp
	}
	if !right {
		padded.WriteString(s)
	}
	return value.NewValue(padded.String()), nil
}
