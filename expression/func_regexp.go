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

	"github.com/couchbaselabs/query/value"
)

///////////////////////////////////////////////////
//
// RegexpContains
//
///////////////////////////////////////////////////

type RegexpContains struct {
	BinaryFunctionBase
	re *regexp.Regexp
}

func NewRegexpContains(first, second Expression) Function {
	rv := &RegexpContains{
		*NewBinaryFunctionBase("regexp_contains", first, second),
		nil,
	}

	rv.re, _ = precompileRegexp(second)
	rv.expr = rv
	return rv
}

func (this *RegexpContains) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *RegexpContains) Type() value.Type { return value.BOOLEAN }

func (this *RegexpContains) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *RegexpContains) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	f := first.Actual().(string)
	s := second.Actual().(string)

	re := this.re
	if re == nil {
		var err error
		re, err = regexp.Compile(s)
		if err != nil {
			return nil, err
		}
	}

	return value.NewValue(re.FindStringIndex(f) != nil), nil
}

func (this *RegexpContains) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewRegexpContains(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// RegexpLike
//
///////////////////////////////////////////////////

type RegexpLike struct {
	BinaryFunctionBase
	re *regexp.Regexp
}

func NewRegexpLike(first, second Expression) Function {
	rv := &RegexpLike{
		*NewBinaryFunctionBase("regexp_like", first, second),
		nil,
	}

	rv.re, _ = precompileRegexp(second)
	rv.expr = rv
	return rv
}

func (this *RegexpLike) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *RegexpLike) Type() value.Type { return value.BOOLEAN }

func (this *RegexpLike) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *RegexpLike) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	f := first.Actual().(string)
	s := second.Actual().(string)

	re := this.re
	if re == nil {
		var err error
		re, err = regexp.Compile(s)
		if err != nil {
			return nil, err
		}
	}

	return value.NewValue(re.MatchString(f)), nil
}

func (this *RegexpLike) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewRegexpLike(operands[0], operands[1])
	}
}

func (this *RegexpLike) Regexp() *regexp.Regexp { return this.re }

///////////////////////////////////////////////////
//
// RegexpPosition
//
///////////////////////////////////////////////////

type RegexpPosition struct {
	BinaryFunctionBase
	re *regexp.Regexp
}

func NewRegexpPosition(first, second Expression) Function {
	rv := &RegexpPosition{
		*NewBinaryFunctionBase("regexp_position", first, second),
		nil,
	}

	rv.re, _ = precompileRegexp(second)
	rv.expr = rv
	return rv
}

func (this *RegexpPosition) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *RegexpPosition) Type() value.Type { return value.NUMBER }

func (this *RegexpPosition) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *RegexpPosition) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	f := first.Actual().(string)
	s := second.Actual().(string)

	re := this.re
	if re == nil {
		var err error
		re, err = regexp.Compile(s)
		if err != nil {
			return nil, err
		}
	}

	loc := re.FindStringIndex(f)
	if loc == nil {
		return value.NewValue(-1.0), nil
	}

	return value.NewValue(float64(loc[0])), nil
}

func (this *RegexpPosition) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewRegexpPosition(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// RegexpReplace
//
///////////////////////////////////////////////////

type RegexpReplace struct {
	FunctionBase
	re *regexp.Regexp
}

func NewRegexpReplace(operands ...Expression) Function {
	rv := &RegexpReplace{
		*NewFunctionBase("regexp_replace", operands...),
		nil,
	}

	rv.re, _ = precompileRegexp(operands[1])
	rv.expr = rv
	return rv
}

func (this *RegexpReplace) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *RegexpReplace) Type() value.Type { return value.STRING }

func (this *RegexpReplace) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *RegexpReplace) Apply(context Context, args ...value.Value) (value.Value, error) {
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

	re := this.re
	if re == nil {
		var err error
		re, err = regexp.Compile(s)
		if err != nil {
			return nil, err
		}
	}

	if len(args) == 3 {
		return value.NewValue(re.ReplaceAllLiteralString(f, r)), nil
	}

	nf := args[3].Actual().(float64)
	if nf != math.Trunc(nf) {
		return value.NULL_VALUE, nil
	}

	n := int(nf)
	rv := re.ReplaceAllStringFunc(f,
		func(m string) string {
			if n > 0 {
				n--
				return r
			} else {
				return m
			}
		})

	return value.NewValue(rv), nil
}

func (this *RegexpReplace) MinArgs() int { return 3 }

func (this *RegexpReplace) MaxArgs() int { return 4 }

func (this *RegexpReplace) Constructor() FunctionConstructor { return NewRegexpReplace }

func precompileRegexp(rexpr Expression) (re *regexp.Regexp, err error) {
	rv := rexpr.Value()
	if rv == nil || rv.Type() != value.STRING {
		return
	}

	re, err = regexp.Compile(rv.Actual().(string))
	return
}
