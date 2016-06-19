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
	return this.UnaryEval(this, item, context)
}

func (this *LikePrefix) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	s := arg.Actual().(string)
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
	return this.UnaryEval(this, item, context)
}

func (this *LikeStop) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	s := arg.Actual().(string)
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
	return this.UnaryEval(this, item, context)
}

func (this *RegexpPrefix) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
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
	return this.UnaryEval(this, item, context)
}

func (this *RegexpStop) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
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
