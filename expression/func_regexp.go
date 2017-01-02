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

///////////////////////////////////////////////////
//
// RegexpContains
//
///////////////////////////////////////////////////

/*
This represents the String function REGEXP_CONTAINS(expr, pattern).
It returns true if the string value contains the regular expression
pattern.
*/
type RegexpContains struct {
	BinaryFunctionBase
	re *regexp.Regexp
}

func NewRegexpContains(first, second Expression) Function {
	rv := &RegexpContains{
		*NewBinaryFunctionBase("regexp_contains", first, second),
		nil,
	}

	rv.re, _ = precompileRegexp(second.Value(), false)
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *RegexpContains) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *RegexpContains) Type() value.Type { return value.BOOLEAN }

func (this *RegexpContains) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For boolean functions, simply list this expression.
*/
func (this *RegexpContains) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
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

/*
Factory method pattern.
*/
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

/*
This represents the String function REGEXP_LIKE(expr, pattern).
It returns true if the string value matches the regular
expression pattern.
*/
type RegexpLike struct {
	BinaryFunctionBase
	re   *regexp.Regexp
	part *regexp.Regexp
	err  error
}

func NewRegexpLike(first, second Expression) Function {
	rv := &RegexpLike{
		*NewBinaryFunctionBase("regexp_like", first, second),
		nil,
		nil,
		nil,
	}

	rv.re, _ = precompileRegexp(second.Value(), true)
	rv.part, rv.err = precompileRegexp(second.Value(), false)
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *RegexpLike) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *RegexpLike) Type() value.Type { return value.BOOLEAN }

func (this *RegexpLike) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For boolean functions, simply list this expression.
*/
func (this *RegexpLike) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *RegexpLike) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	f := first.Actual().(string)
	s := second.Actual().(string)

	fullRe := this.re
	partRe := this.part

	/* MB-20677 make sure full regexp doesn't skew RegexpLike
	   into accepting wrong partial regexps
	*/
	if this.err != nil {
		return nil, this.err
	}
	if partRe == nil {
		var err error

		/* MB-20677 ditto */
		partRe, err = regexp.Compile(s)
		if err != nil {
			return nil, err
		}
		fullRe, err = regexp.Compile("^" + s + "$")
		if err != nil {
			return nil, err
		}
	}

	return value.NewValue(fullRe.MatchString(f)), nil
}

/*
Factory method pattern.
*/
func (this *RegexpLike) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewRegexpLike(operands[0], operands[1])
	}
}

/*
Disallow pushing limit to index scan
*/

func (this *RegexpLike) IsLimitPushable() bool {
	return false
}

/*
Return the regular expression without delimiters.
*/
func (this *RegexpLike) Regexp() *regexp.Regexp {
	return this.part
}

///////////////////////////////////////////////////
//
// RegexpPosition
//
///////////////////////////////////////////////////

/*
This represents the String function REGEXP_POSITION(expr, pattern)
It returns the first position of the regular expression pattern
within the string, or -1.
*/
type RegexpPosition struct {
	BinaryFunctionBase
	re *regexp.Regexp
}

func NewRegexpPosition(first, second Expression) Function {
	rv := &RegexpPosition{
		*NewBinaryFunctionBase("regexp_position", first, second),
		nil,
	}

	rv.re, _ = precompileRegexp(second.Value(), false)
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
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
		return value.NewValue(-1), nil
	}

	return value.NewValue(loc[0]), nil
}

/*
Factory method pattern.
*/
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

/*
This represents the String function
REGEXP_REPLACE(expr, pattern, repl [, n ]). It returns a
new string with occurences of pattern replaced with repl.
If n is given, at most n replacements are performed.
*/
type RegexpReplace struct {
	FunctionBase
	re *regexp.Regexp
}

func NewRegexpReplace(operands ...Expression) Function {
	rv := &RegexpReplace{
		*NewFunctionBase("regexp_replace", operands...),
		nil,
	}

	rv.re, _ = precompileRegexp(operands[1].Value(), false)
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
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

/*
Minimum input arguments required for the REGEXP_REPLACE
function is 3.
*/
func (this *RegexpReplace) MinArgs() int { return 3 }

/*
MAXIMUM input arguments allowed for the REGEXP_REPLACE
function is 4.
*/
func (this *RegexpReplace) MaxArgs() int { return 4 }

/*
Factory method pattern.
*/
func (this *RegexpReplace) Constructor() FunctionConstructor {
	return NewRegexpReplace
}

/*
This method compiles and sets the regular expression re
later used to set the field in the REGEXP_ function.
If the input expression value is nil or type string, return.
If not then call Compile with the value, to parse a regular
expression and return, if successful, a Regexp object that
can be used to match against text.
*/
func precompileRegexp(rexpr value.Value, full bool) (re *regexp.Regexp, err error) {
	if rexpr == nil || rexpr.Type() != value.STRING {
		return
	}

	s := rexpr.Actual().(string)
	if full {
		s = "^" + s + "$"
	}

	re, err = regexp.Compile(s)
	return
}
