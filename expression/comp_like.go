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
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/couchbase/query/value"
)

/*
Comparison terms allow for comparing two expressions.
LIKE and NOT LIKE are used to to search for a specified
pattern in an expression. The LIKE operator allows for
wildcard matching of string values. The right-hand side
of the operator is a pattern, optionally containg '%'
and '_' wildcard characters.
*/
type Like struct {
	FunctionBase
	re            *regexp.Regexp
	part          *regexp.Regexp
	canCacheRegex bool
}

// This is cached in the context for each operator instance
type LikeRegex struct {
	Orig string
	Re   *regexp.Regexp
}

var DEFAULT_ESCAPE_EXPR = NewConstant(value.NewValue("\\"))

func NewLike(operands ...Expression) Function {
	if len(operands) < 3 {
		operands = append(operands, DEFAULT_ESCAPE_EXPR)
	}
	rv := &Like{
		*NewFunctionBase("like", operands...),
		nil,
		nil,
		(operands[1].StaticNoVariable() != nil && operands[2].StaticNoVariable() != nil),
	}
	p := operands[1].Value()
	ev := operands[2].Value()
	// only precompile if both pattern and escape are values at this point
	if p != nil && ev != nil {
		r, _ := getEscapeRuneFromValue(ev)
		// escape has to be valid to precompile
		if r != utf8.RuneError {
			rv.re, rv.part, _ = precompileLike(p, r)
		}
	}
	rv.expr = rv
	return rv
}

func (this *Like) First() Expression {
	return this.operands[0]
}

func (this *Like) Second() Expression {
	return this.operands[1]
}

func (this *Like) Escape() Expression {
	return this.operands[2]
}

func (this *Like) IsDefaultEscape() bool {
	return this.Escape() == DEFAULT_ESCAPE_EXPR
}

/*
Visitor pattern.
*/
func (this *Like) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitLike(this)
}

func (this *Like) Type() value.Type { return value.BOOLEAN }

func (this *Like) Evaluate(item value.Value, context Context) (value.Value, error) {

	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	third, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	if first.Type() == value.MISSING || second.Type() == value.MISSING || third.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING || third.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	f := first.ToString()
	s := second.ToString()
	escape, err := getEscapeRuneFromValue(third)
	if err != nil {
		return nil, err
	}

	re := this.re
	if re == nil {
		var likeContext LikeContext
		ok := false
		var key string
		if this.canCacheRegex {
			key = fmt.Sprintf("%c%s", escape, s)
			likeContext, ok = context.(LikeContext)
			if ok {
				re = likeContext.GetLikeRegex(this, key)
			}
		}
		if re == nil {
			re, _, err = LikeCompile(s, escape)
			if err != nil {
				return nil, err
			}
			if ok {
				likeContext.CacheLikeRegex(this, key, re)
			}
		}
	}

	return value.NewValue(re.MatchString(f)), nil
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For LIKE, simply list this expression.
*/
func (this *Like) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *Like) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	covers[this] = value.TRUE_VALUE
	return covers
}

/*
Factory method pattern.
*/
func (this *Like) Constructor() FunctionConstructor {
	return NewLike
}

func (this *Like) MinArgs() int { return 2 }

func (this *Like) MaxArgs() int { return 3 }

/*
Return the regular expression without delimiters.
*/
func (this *Like) Regexp() *regexp.Regexp {
	return this.part
}

/*
Initializes regexp fields.
*/
func precompileLike(sv value.Value, escape rune) (re, part *regexp.Regexp, err error) {
	if sv == nil || sv.Type() != value.STRING {
		return
	}

	s := sv.ToString()
	return LikeCompile(s, escape)
}

/*
This method compiles the input string s into a regular expression and
returns it. Convert LIKE special characters to regexp special
characters. Escape regexp special characters. Add start and end
boundaries.

If the escape character specified is a LIKE special character, that meaning
is lost (this is permitted as it might be desired).
*/

const (
	anyStringWildcard = '%'
	anyCharWildcard   = '_'
	anyString         = "(.*)"
	anyChar           = "(.)"
)

func LikeCompile(s string, escape rune) (re, part *regexp.Regexp, err error) {
	pat := make([]rune, 0, len(s)*2)
	literal := make([]rune, 0, len(s)*2)
	escaped := false
	for _, r := range s {
		switch {
		case escaped == true:
			fallthrough
		default:
			literal = append(literal, r)
		case r == escape:
			escaped = true
			continue
		case r == anyStringWildcard:
			if len(literal) > 0 {
				pat = append(pat, []rune(regexp.QuoteMeta(string(literal)))...)
				literal = literal[:0]
			}
			// collapse ajacent anyString patterns
			if !strings.HasSuffix(string(pat), anyString) {
				pat = append(pat, []rune(anyString)...)
			}
		case r == anyCharWildcard:
			if len(literal) > 0 {
				pat = append(pat, []rune(regexp.QuoteMeta(string(literal)))...)
				literal = literal[:0]
			}
			pat = append(pat, []rune(anyChar)...)
		}
		escaped = false
	}
	if escaped {
		return nil, nil, fmt.Errorf("Trailing escape character (%c) in pattern", escape)
	}
	if len(literal) > 0 {
		pat = append(pat, []rune(regexp.QuoteMeta(string(literal)))...)
	}

	part, err = regexp.Compile(string(pat))
	if err != nil {
		return
	}

	// turn on wildcard matching of \n as it may be embedded; DO NOT turn on multi-line mode (MB-39569)
	pat = append([]rune("(?s)^"), append(pat, rune('$'))...)

	re, err = regexp.Compile(string(pat))
	return
}

func likeLiteralPrefix(s string, escape rune) (string, bool) {
	escaped := false
	res := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case escaped == true:
		case r == escape:
			escaped = true
			continue
		case r == anyStringWildcard:
			fallthrough
		case r == anyCharWildcard:
			return string(res), false
		}
		escaped = false
		res = append(res, r)
	}
	return string(res), true
}

/*
This function implements the NOT LIKE operation.
*/
func NewNotLike(first, second, third Expression) Expression {
	return NewNot(NewLike(first, second, third))
}

func getEscapeRuneFromValue(v value.Value) (rune, error) {
	s := v.ToString()
	escape, _ := utf8.DecodeRuneInString(s)
	if escape == utf8.RuneError || utf8.RuneCountInString(s) != 1 {
		return utf8.RuneError, fmt.Errorf("ESCAPE clause must resolve to a single character")
	}
	return escape, nil
}
