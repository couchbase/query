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
	rv := &RegexpContains{}
	rv.Init("regexp_contains", first, second)

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

	f := first.ToString()
	s := second.ToString()

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
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For boolean functions, simply list this expression.
*/
func (this *RegexpContains) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *RegexpContains) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	covers[this] = value.TRUE_VALUE
	return covers
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
	rv := &RegexpLike{}
	rv.Init("regexp_like", first, second)

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

	/* MB-20677 make sure full regexp doesn't skew RegexpLike
	   into accepting wrong partial regexps
	*/
	if this.err != nil {
		return nil, this.err
	}

	f := first.ToString()
	s := second.ToString()

	fullRe := this.re
	partRe := this.part

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
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For boolean functions, simply list this expression.
*/
func (this *RegexpLike) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *RegexpLike) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	covers[this] = value.TRUE_VALUE
	return covers
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
// RegexpPosition0 / RegexpPosition
//
///////////////////////////////////////////////////

/*
This represents the String function REGEXP_POSITION(expr, pattern)
It returns the 0 based - first position of the regular expression pattern
within the string, or -1.
*/
type RegexpPosition0 struct {
	BinaryFunctionBase
	re *regexp.Regexp
}

func NewRegexpPosition0(first, second Expression) Function {
	rv := &RegexpPosition0{}
	rv.Init("regexp_position0", first, second)

	rv.re, _ = precompileRegexp(second.Value(), false)
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *RegexpPosition0) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *RegexpPosition0) Type() value.Type { return value.NUMBER }

func (this *RegexpPosition0) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	re := this.re
	return regexpPositionApply(first, second, re, 0)
}

/*
Factory method pattern.
*/
func (this *RegexpPosition0) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewRegexpPosition0(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// RegexpPosition1
//
///////////////////////////////////////////////////

/*
This represents the String function REGEXP_POSITION1(expr, pattern)
It returns the 1 based - first position of the regular expression pattern
within the string, or -1.
*/
type RegexpPosition1 struct {
	BinaryFunctionBase
	re *regexp.Regexp
}

func NewRegexpPosition1(first, second Expression) Function {
	rv := &RegexpPosition1{}
	rv.Init("regexp_position1", first, second)

	rv.re, _ = precompileRegexp(second.Value(), false)
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *RegexpPosition1) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *RegexpPosition1) Type() value.Type { return value.NUMBER }

func (this *RegexpPosition1) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	re := this.re
	return regexpPositionApply(first, second, re, 1)
}

/*
Factory method pattern.
*/
func (this *RegexpPosition1) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewRegexpPosition1(operands[0], operands[1])
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
	rv := &RegexpReplace{}
	rv.Init("regexp_replace", operands...)

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
	var arg3 value.Value
	null := false
	missing := false

	arg0, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg0.Type() == value.MISSING {
		missing = true
	} else if arg0.Type() != value.STRING {
		null = true
	}
	arg1, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg1.Type() == value.MISSING {
		missing = true
	} else if arg1.Type() != value.STRING {
		null = true
	}
	arg2, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg2.Type() == value.MISSING {
		missing = true
	} else if arg2.Type() != value.STRING {
		null = true
	}
	if len(this.operands) == 4 {
		arg3, err = this.operands[3].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg3.Type() == value.MISSING {
			missing = true
		} else if arg3.Type() != value.NUMBER {
			null = true
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	f := arg0.ToString()
	s := arg1.ToString()
	r := arg2.ToString()

	re := this.re
	if re == nil {
		var err error
		re, err = regexp.Compile(s)
		if err != nil {
			return nil, err
		}
	}

	if len(this.operands) == 3 {
		return value.NewValue(re.ReplaceAllLiteralString(f, r)), nil
	}

	nf := arg3.Actual().(float64)
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

	s := rexpr.ToString()
	if full {
		s = "^" + s + "$"
	}

	re, err = regexp.Compile(s)
	return
}

func regexpPositionApply(first, second value.Value, re *regexp.Regexp, startPos int) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	f := first.ToString()
	s := second.ToString()

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

	return value.NewValue(loc[0] + startPos), nil
}

///////////////////////////////////////////////////
//
// RegexpMatches
//
///////////////////////////////////////////////////

/*
This represents the String function REGEXP_MATCHES(expr, pattern).
It returns an array containing all the substrings in expr that
matche the pattern.
*/
type RegexpMatches struct {
	BinaryFunctionBase
	re *regexp.Regexp
}

func NewRegexpMatches(first, second Expression) Function {
	rv := &RegexpMatches{}
	rv.Init("regexp_matches", first, second)

	rv.re, _ = precompileRegexp(second.Value(), false)
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *RegexpMatches) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *RegexpMatches) Type() value.Type { return value.ARRAY }

func (this *RegexpMatches) Evaluate(item value.Value, context Context) (value.Value, error) {
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

	re := this.re
	if re == nil {
		var err error
		re, err = regexp.Compile(second.ToString())
		if err != nil {
			return nil, err
		}
	}

	var res []interface{}
	matches := re.FindAll([]byte(first.ToString()), -1)
	for _, v := range matches {
		res = append(res, string(v))
	}
	return value.NewValue(res), nil
}

/*
Factory method pattern.
*/
func (this *RegexpMatches) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewRegexpMatches(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// RegexpSplit
//
///////////////////////////////////////////////////

/*
This represents the String function REGEXP_SPLIT(expr, pattern).
It returns an array of substrings found in expr that are separated
by pattern.
*/
type RegexpSplit struct {
	BinaryFunctionBase
	re *regexp.Regexp
}

func NewRegexpSplit(first, second Expression) Function {
	rv := &RegexpSplit{}
	rv.Init("regexp_split", first, second)

	rv.re, _ = precompileRegexp(second.Value(), false)
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *RegexpSplit) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *RegexpSplit) Type() value.Type { return value.ARRAY }

func (this *RegexpSplit) Evaluate(item value.Value, context Context) (value.Value, error) {
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

	re := this.re
	if re == nil {
		var err error
		re, err = regexp.Compile(second.ToString())
		if err != nil {
			return nil, err
		}
	}

	var res []interface{}
	f := []byte(first.ToString())
	matches := re.FindAllIndex(f, -1)
	start := 0
	for _, v := range matches {
		// align with string split and don't return empty match before first rune
		if v[0] == v[1] && v[0] == 0 {
			continue
		}
		res = append(res, string(f[start:v[0]]))
		start = v[1]
	}
	if len(res) > 0 && start < len(f) {
		res = append(res, string(f[start:]))
	}
	return value.NewValue(res), nil
}

/*
Factory method pattern.
*/
func (this *RegexpSplit) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewRegexpSplit(operands[0], operands[1])
	}
}
