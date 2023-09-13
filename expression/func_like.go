//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"math"
	"regexp"

	"github.com/couchbase/query/value"
)

type LikeFunction interface {
	BinaryFunction
	Regexp() *regexp.Regexp
}

var _DEFAULT_ESCAPE = NewConstant("\\")

///////////////////////////////////////////////////
//
// LikePrefix
//
///////////////////////////////////////////////////

/*
This represents the pattern matching function LIKE_PREFIX(expr).
*/
type LikePrefix struct {
	BinaryFunctionBase
}

func NewLikePrefix(operands ...Expression) Function {
	if len(operands) == 1 {
		operands = append(operands, _DEFAULT_ESCAPE)
	}
	rv := &LikePrefix{}
	rv.Init("like_prefix", operands[0], operands[1])

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *LikePrefix) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *LikePrefix) Type() value.Type { return value.STRING }

func (this *LikePrefix) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		null = true
	}

	esc, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if esc.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if esc.Type() != value.STRING || null {
		return value.NULL_VALUE, nil
	}

	s := arg.ToString()
	escape, err := getEscapeRuneFromValue(esc)
	if err != nil {
		return nil, err
	}

	prefix, _ := likeLiteralPrefix(s, escape)
	return value.NewValue(prefix), nil
}

/*
Factory method pattern.
*/
func (this *LikePrefix) Constructor() FunctionConstructor {
	return NewLikePrefix
}

func (this *LikePrefix) MinArgs() int { return 1 }

///////////////////////////////////////////////////
//
// LikeStop
//
///////////////////////////////////////////////////

/*
This represents the pattern-matching function LIKE_STOP(expr).
*/
type LikeStop struct {
	BinaryFunctionBase
}

func NewLikeStop(operands ...Expression) Function {
	if len(operands) == 1 {
		operands = append(operands, _DEFAULT_ESCAPE)
	}
	rv := &LikeStop{}
	rv.Init("like_stop", operands[0], operands[1])

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *LikeStop) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *LikeStop) Type() value.Type { return value.JSON }

func (this *LikeStop) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		null = true
	}

	esc, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if esc.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if esc.Type() != value.STRING || null {
		return value.NULL_VALUE, nil
	}

	s := arg.ToString()
	escape, err := getEscapeRuneFromValue(esc)
	if err != nil {
		return nil, err
	}

	prefix, complete := likeLiteralPrefix(s, escape)
	if complete {
		return value.NewValue(prefix + "\x00"), nil
	}

	last := len(prefix) - 1
	if last >= 0 && prefix[last] < math.MaxUint8 {
		bytes := []byte(prefix)
		bytes[last]++
		return value.NewValue(string(bytes)), nil
	} else {
		return value.EMPTY_ARRAY_VALUE, nil
	}
}

/*
Factory method pattern.
*/
func (this *LikeStop) Constructor() FunctionConstructor {
	return NewLikeStop
}

func (this *LikeStop) MinArgs() int { return 1 }

///////////////////////////////////////////////////
//
// LikeSuffix
//
///////////////////////////////////////////////////

/*
This represents the pattern matching function LIKE_SUFFIX(expr).
*/
type LikeSuffix struct {
	BinaryFunctionBase
}

func NewLikeSuffix(operands ...Expression) Function {
	if len(operands) == 1 {
		operands = append(operands, _DEFAULT_ESCAPE)
	}
	rv := &LikeSuffix{}
	rv.Init("like_suffix", operands[0], operands[1])

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *LikeSuffix) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *LikeSuffix) Type() value.Type { return value.STRING }

func (this *LikeSuffix) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		null = true
	}

	esc, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if esc.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if esc.Type() != value.STRING || null {
		return value.NULL_VALUE, nil
	}

	escape, err := getEscapeRuneFromValue(esc)
	if err != nil {
		return nil, err
	}

	for s := arg.ToString(); s != ""; s = s[1:] {
		prefix, _ := likeLiteralPrefix(s, escape)
		if prefix != "" {
			return value.NewValue(s), nil
		}
	}

	return value.EMPTY_STRING_VALUE, nil
}

/*
Factory method pattern.
*/
func (this *LikeSuffix) Constructor() FunctionConstructor {
	return NewLikeSuffix
}

func (this *LikeSuffix) MinArgs() int { return 1 }

///////////////////////////////////////////////////
//
// RegexpPrefix
//
///////////////////////////////////////////////////

/*
This represents the pattern matching function LIKE_PREFIX(expr).
*/
type RegexpPrefix struct {
	UnaryFunctionBase
}

func NewRegexpPrefix(operand Expression) Function {
	rv := &RegexpPrefix{}
	rv.Init("regexp_prefix", operand)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *RegexpPrefix) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *RegexpPrefix) Type() value.Type { return value.STRING }

func (this *RegexpPrefix) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	part, err := precompileRegexp(arg, false)
	if part == nil || err != nil {
		return value.NULL_VALUE, err
	}

	prefix, _ := part.LiteralPrefix()
	return value.NewValue(prefix), nil
}

/*
Factory method pattern.
*/
func (this *RegexpPrefix) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewRegexpPrefix(operands[0])
	}
}

///////////////////////////////////////////////////
//
// RegexpStop
//
///////////////////////////////////////////////////

/*
This represents the pattern-matching function LIKE_STOP(expr).
*/
type RegexpStop struct {
	UnaryFunctionBase
}

func NewRegexpStop(operand Expression) Function {
	rv := &RegexpStop{}
	rv.Init("regexp_stop", operand)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *RegexpStop) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *RegexpStop) Type() value.Type { return value.JSON }

func (this *RegexpStop) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	part, err := precompileRegexp(arg, false)
	if part == nil || err != nil {
		return value.NULL_VALUE, err
	}

	prefix, complete := part.LiteralPrefix()
	if complete {
		return value.NewValue(prefix + "\x00"), nil
	}

	last := len(prefix) - 1
	if last >= 0 && prefix[last] < math.MaxUint8 {
		bytes := []byte(prefix)
		bytes[last]++
		return value.NewValue(string(bytes)), nil
	} else {
		return value.EMPTY_ARRAY_VALUE, nil
	}
}

/*
Factory method pattern.
*/
func (this *RegexpStop) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewRegexpStop(operands[0])
	}
}

///////////////////////////////////////////////////
//
// RegexpSuffix
//
///////////////////////////////////////////////////

/*
This represents the pattern matching function REGEXP_SUFFIX(expr).
*/
type RegexpSuffix struct {
	UnaryFunctionBase
}

func NewRegexpSuffix(operand Expression) Function {
	rv := &RegexpSuffix{}
	rv.Init("regexp_suffix", operand)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *RegexpSuffix) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *RegexpSuffix) Type() value.Type { return value.STRING }

func (this *RegexpSuffix) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	for s := arg.ToString(); s != ""; s = s[1:] {
		part, err := regexp.Compile(s)
		if err != nil {
			return value.NULL_VALUE, err
		}

		prefix, _ := part.LiteralPrefix()
		if prefix != "" {
			return value.NewValue(s), nil
		}
	}

	return value.EMPTY_STRING_VALUE, nil
}

/*
Factory method pattern.
*/
func (this *RegexpSuffix) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewRegexpSuffix(operands[0])
	}
}
