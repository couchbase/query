//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"regexp"

	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// ContainsToken
//
///////////////////////////////////////////////////

type ContainsToken struct {
	FunctionBase
}

func NewContainsToken(operands ...Expression) Function {
	rv := &ContainsToken{}
	rv.Init("contains_token", operands...)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ContainsToken) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ContainsToken) Type() value.Type { return value.BOOLEAN }

func (this *ContainsToken) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	missing := false
	source, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if source.Type() == value.MISSING {
		missing = true
	} else if source.Type() == value.NULL {
		null = true
	}
	token, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if token.Type() == value.MISSING {
		missing = true
	} else if token.Type() == value.NULL {
		null = true
	}

	options := _EMPTY_OPTIONS
	if len(this.operands) > 2 {
		options, err = this.operands[2].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if options.Type() == value.MISSING {
			missing = true
		} else if options.Type() != value.OBJECT {
			null = true
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	contains := source.ContainsToken(token, options)
	return value.NewValue(contains), nil
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For boolean functions, simply list this expression.
*/
func (this *ContainsToken) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *ContainsToken) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	covers[this] = value.TRUE_VALUE
	return covers
}

func (this *ContainsToken) MinArgs() int { return 2 }

func (this *ContainsToken) MaxArgs() int { return 3 }

/*
Factory method pattern.
*/
func (this *ContainsToken) Constructor() FunctionConstructor {
	return NewContainsToken
}

///////////////////////////////////////////////////
//
// ContainsTokenLike
//
///////////////////////////////////////////////////

type ContainsTokenLike struct {
	FunctionBase
	re   *regexp.Regexp
	part *regexp.Regexp
}

func NewContainsTokenLike(operands ...Expression) Function {
	rv := &ContainsTokenLike{}
	rv.Init("contains_token_like", operands...)

	rv.re, rv.part, _ = precompileLike(operands[1].Value(), '\\')
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ContainsTokenLike) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ContainsTokenLike) Type() value.Type { return value.BOOLEAN }

func (this *ContainsTokenLike) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	missing := false
	source, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if source.Type() == value.MISSING {
		missing = true
	} else if source.Type() == value.NULL {
		null = true
	}
	pattern, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if pattern.Type() == value.MISSING {
		missing = true
	} else if pattern.Type() != value.STRING {
		null = true
	}

	options := _EMPTY_OPTIONS
	if len(this.operands) > 2 {
		options, err = this.operands[2].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if options.Type() == value.MISSING {
			missing = true
		} else if options.Type() != value.OBJECT {
			null = true
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	re := this.re
	if re == nil {
		var err error
		// because the tokenising disallows special characters we don't have to care about the escape character
		// (tokens can never contain anything other than letters and numbers)
		// if we change the tokenising at some point to allow character groups then we'll have to add "escape" to the options
		re, _, err = LikeCompile(pattern.ToString(), '\\')
		if err != nil {
			return nil, err
		}
	}

	matcher := func(token interface{}) bool {
		str, ok := token.(string)
		if !ok {
			return false
		}

		return re.MatchString(str)
	}

	contains := source.ContainsMatchingToken(matcher, options)
	return value.NewValue(contains), nil
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For boolean functions, simply list this expression.
*/
func (this *ContainsTokenLike) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *ContainsTokenLike) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	covers[this] = value.TRUE_VALUE
	return covers
}

func (this *ContainsTokenLike) MinArgs() int { return 2 }

func (this *ContainsTokenLike) MaxArgs() int { return 3 }

/*
Factory method pattern.
*/
func (this *ContainsTokenLike) Constructor() FunctionConstructor {
	return NewContainsTokenLike
}

///////////////////////////////////////////////////
//
// ContainsTokenRegexp
//
///////////////////////////////////////////////////

type ContainsTokenRegexp struct {
	FunctionBase
	re   *regexp.Regexp
	part *regexp.Regexp
	err  error
}

func NewContainsTokenRegexp(operands ...Expression) Function {
	rv := &ContainsTokenRegexp{}
	rv.Init("contains_token_regexp", operands...)

	rv.re, _ = precompileRegexp(operands[1].Value(), true)
	rv.part, rv.err = precompileRegexp(operands[1].Value(), false)
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ContainsTokenRegexp) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ContainsTokenRegexp) Type() value.Type { return value.BOOLEAN }

func (this *ContainsTokenRegexp) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	missing := false
	source, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if source.Type() == value.MISSING {
		missing = true
	} else if source.Type() == value.NULL {
		null = true
	}
	pattern, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if pattern.Type() == value.MISSING {
		missing = true
	} else if pattern.Type() != value.STRING {
		null = true
	}

	options := _EMPTY_OPTIONS
	if len(this.operands) > 2 {
		options, err = this.operands[2].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if options.Type() == value.MISSING {
			missing = true
		} else if options.Type() != value.OBJECT {
			null = true
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	/* MB-20677 make sure full regexp doesn't skew RegexpLike
	   into accepting wrong partial regexps
	*/
	if this.err != nil {
		return nil, this.err
	}

	fullRe := this.re
	partRe := this.part

	if partRe == nil {
		var err error
		s := pattern.ToString()

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

	matcher := func(token interface{}) bool {
		str, ok := token.(string)
		if !ok {
			return false
		}

		return fullRe.MatchString(str)
	}

	contains := source.ContainsMatchingToken(matcher, options)
	return value.NewValue(contains), nil
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For boolean functions, simply list this expression.
*/
func (this *ContainsTokenRegexp) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *ContainsTokenRegexp) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	covers[this] = value.TRUE_VALUE
	return covers
}

func (this *ContainsTokenRegexp) MinArgs() int { return 2 }

func (this *ContainsTokenRegexp) MaxArgs() int { return 3 }

/*
Factory method pattern.
*/
func (this *ContainsTokenRegexp) Constructor() FunctionConstructor {
	return NewContainsTokenRegexp
}

///////////////////////////////////////////////////
//
// Tokens
//
///////////////////////////////////////////////////

/*
MB-20850. Enumerate list of all tokens within the operand. For
strings, this is the list of discrete words within the string. For all
other atomic JSON values, it is the operand itself. For arrays, all
the individual array elements are tokenized. And for objects, the
names are included verbatim, while the values are tokenized.
*/
type Tokens struct {
	FunctionBase
}

func NewTokens(operands ...Expression) Function {
	rv := &Tokens{}
	rv.Init("tokens", operands...)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Tokens) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Tokens) Type() value.Type { return value.ARRAY }

func (this *Tokens) Evaluate(item value.Value, context Context) (value.Value, error) {
	null := false
	missing := false
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		missing = true
	}

	options := _EMPTY_OPTIONS
	if len(this.operands) > 1 {
		options, err = this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if options.Type() == value.MISSING {
			missing = true
		} else if options.Type() != value.OBJECT {
			null = true
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	set := _SET_POOL.Get()
	defer _SET_POOL.Put(set)
	set = arg.Tokens(set, options)
	items := set.Items()
	return value.NewValue(items), nil
}

func (this *Tokens) MinArgs() int { return 1 }

func (this *Tokens) MaxArgs() int { return 2 }

/*
Factory method pattern.
*/
func (this *Tokens) Constructor() FunctionConstructor {
	return NewTokens
}

var _EMPTY_OPTIONS = value.NewValue(map[string]interface{}{})
