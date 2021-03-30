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
	"regexp"

	"github.com/couchbase/query/value"
)

type LikeFunction interface {
	BinaryFunction
	Regexp() *regexp.Regexp
}

///////////////////////////////////////////////////
//
// LikePrefix
//
///////////////////////////////////////////////////

/*
This represents the pattern matching function LIKE_PREFIX(expr).
*/
type LikePrefix struct {
	UnaryFunctionBase
}

func NewLikePrefix(operand Expression) Function {
	rv := &LikePrefix{
		*NewUnaryFunctionBase("like_prefix", operand),
	}

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
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	s := arg.ToString()
	_, part, err := likeCompile(s)
	if err != nil {
		return value.NULL_VALUE, err
	}

	prefix, _ := part.LiteralPrefix()
	return value.NewValue(prefix), nil
}

/*
Factory method pattern.
*/
func (this *LikePrefix) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLikePrefix(operands[0])
	}
}

///////////////////////////////////////////////////
//
// LikeStop
//
///////////////////////////////////////////////////

/*
This represents the pattern-matching function LIKE_STOP(expr).
*/
type LikeStop struct {
	UnaryFunctionBase
}

func NewLikeStop(operand Expression) Function {
	rv := &LikeStop{
		*NewUnaryFunctionBase("like_stop", operand),
	}

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
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	s := arg.ToString()
	_, part, err := likeCompile(s)
	if err != nil {
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
func (this *LikeStop) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLikeStop(operands[0])
	}
}

///////////////////////////////////////////////////
//
// LikeSuffix
//
///////////////////////////////////////////////////

/*
This represents the pattern matching function LIKE_SUFFIX(expr).
*/
type LikeSuffix struct {
	UnaryFunctionBase
}

func NewLikeSuffix(operand Expression) Function {
	rv := &LikeSuffix{
		*NewUnaryFunctionBase("like_suffix", operand),
	}

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
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	for s := arg.ToString(); s != ""; s = s[1:] {
		_, part, err := likeCompile(s)
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
func (this *LikeSuffix) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLikeSuffix(operands[0])
	}
}

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
	rv := &RegexpPrefix{
		*NewUnaryFunctionBase("regexp_prefix", operand),
	}

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
	rv := &RegexpStop{
		*NewUnaryFunctionBase("regexp_stop", operand),
	}

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
	rv := &RegexpSuffix{
		*NewUnaryFunctionBase("regexp_suffix", operand),
	}

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
