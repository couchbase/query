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
	"regexp"

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
	BinaryFunctionBase
	re   *regexp.Regexp
	part *regexp.Regexp
}

func NewLike(first, second Expression) Function {
	rv := &Like{
		*NewBinaryFunctionBase("like", first, second),
		nil,
		nil,
	}

	rv.precompile()
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Like) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitLike(this)
}

func (this *Like) Type() value.Type { return value.BOOLEAN }

func (this *Like) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
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

func (this *Like) Apply(context Context, first, second value.Value) (value.Value, error) {
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
		re, _, err = likeCompile(s)
		if err != nil {
			return nil, err
		}
	}

	return value.NewValue(re.MatchString(f)), nil
}

/*
Factory method pattern.
*/
func (this *Like) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLike(operands[0], operands[1])
	}
}

/*
Return the regular expression without delimiters.
*/
func (this *Like) Regexp() *regexp.Regexp {
	return this.part
}

/*
This method sets the regexp field in the Like struct.
*/
func (this *Like) precompile() {
	sv := this.Second().Value()
	if sv == nil || sv.Type() != value.STRING {
		return
	}

	s := sv.Actual().(string)
	re, part, err := likeCompile(s)
	if err != nil {
		return
	}

	this.re, this.part = re, part
}

/*
This method compiles the input string s into a regular expression and
returns it. Convert LIKE special characters to regexp special
characters. Escape regexp special characters. Add start and end
boundaries.
*/
func likeCompile(s string) (re, part *regexp.Regexp, err error) {
	s = regexp.QuoteMeta(s)
	repl := regexp.MustCompile(`\\_|\\%|_|%`)
	s = repl.ReplaceAllStringFunc(s, replacer)

	part, err = regexp.Compile(s)
	if err != nil {
		return
	}

	/* MB-19230 only add ^ and $ if we are not
	   starting or ending with %.
	   turn off $ and . matching \n
	*/
	if s != "" && s[0] != '%' && s[0] != '_' {
		s = "^" + s
	}
	l := len(s)
	if l > 0 && s[l-1] != '%' && s[l-1] != '_' {
		s = s + "$"
	}
	s = "(?ms)" + s

	re, err = regexp.Compile(s)
	return
}

/*
The function replaces the input strings with
strings and returns the new string. It is a
regular expression replacer.
Percent (%) matches any string of zero or more
characters; underscore (_) matches any single
character. The wildcards can be escaped by preceding
them with a backslash (\). Backslash itself can also
be escaped by preceding it with another backslash.
All these characters need to be replaced correctly.
*/
func replacer(s string) string {
	switch s {
	case `\_`:
		return "_"
	case `\%`:
		return "%"
	case `_`:
		return "(.)"
	case `%`:
		return "(.*)"
	default:
		return s
	}
}

/*
This function implements the NOT LIKE operation.
*/
func NewNotLike(first, second Expression) Expression {
	return NewNot(NewLike(first, second))
}
