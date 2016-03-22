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
pattern. Type RegexpContains is a struct that implements
BinaryFunctionBase. It has a field that
represents a regular expression. Regexp is the
representation of a compiled regular expression.
*/
type RegexpContains struct {
	BinaryFunctionBase
	re *regexp.Regexp
}

/*
The function NewRegexpContains calls NewBinaryFunctionBase to
create a function named REGEXP_CONTAINS with the two
expressions as input. It calls precompile to populate the
regexp field in the struct.
*/
func NewRegexpContains(first, second Expression) Function {
	rv := &RegexpContains{
		*NewBinaryFunctionBase("regexp_contains", first, second),
		nil,
	}

	rv.re, _ = precompileRegexp(second, false)
	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *RegexpContains) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type BOOLEAN.
*/
func (this *RegexpContains) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
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

/*
This method takes in two values and returns a value that
corresponds to whether the regular expression (already set
or populated using the second value) contains the first
string. If the input type is missing return missing, and
if it isnt string then return null value. Use the
FindStringIndex method in the regexp package to return
a two-element slice of integers defining the location of
the leftmost match in the string of the regular expression as per the
Go Docs. A return value of nil indicates no match. Return this value.
*/
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
The constructor returns a NewRegexpContains with the two operands
cast to a Function as the FunctionConstructor.
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
expression pattern. Type RegexpLike is a struct that implements
BinaryFunctionBase. It has a field that represents a regular
expression. Regexp is the representation of a compiled regular
expression.
*/
type RegexpLike struct {
	BinaryFunctionBase
	re   *regexp.Regexp
	part *regexp.Regexp
}

/*
The function NewRegexpLike calls NewBinaryFunctionBase to
create a function named REGEXP_LIKE with the two
expressions as input. It calls precompile to populate the
regexp field in the struct.
*/
func NewRegexpLike(first, second Expression) Function {
	rv := &RegexpLike{
		*NewBinaryFunctionBase("regexp_like", first, second),
		nil,
		nil,
	}

	rv.re, _ = precompileRegexp(second, true)
	rv.part, _ = precompileRegexp(second, false)
	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *RegexpLike) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type BOOLEAN.
*/
func (this *RegexpLike) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
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

/*
This method takes in two values and returns a value that
corresponds to whether the first string value matches the
regular expression (already set or populated using the
second value) If the input type is missing return missing, and
if it isnt string then return null value. Use the MatchString
method to compare the regexp and string and return that value.
*/
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
		re, err = regexp.Compile("^" + s + "$")
		if err != nil {
			return nil, err
		}
	}

	return value.NewValue(re.MatchString(f)), nil
}

/*
The constructor returns a NewRegexpLike with the two operands
cast to a Function as the FunctionConstructor.
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
within the string, or -1. Type RegexpPosition is a struct that
implements BinaryFunctionBase. It has a field that represents a
regular expression. Regexp is the representation of a compiled
regular expression.
*/
type RegexpPosition struct {
	BinaryFunctionBase
	re *regexp.Regexp
}

/*
The function NewRegexpPosition calls NewBinaryFunctionBase to
create a function named REGEXP_POSITION with the two
expressions as input. It calls precompile to populate the
regexp field in the struct.
*/
func NewRegexpPosition(first, second Expression) Function {
	rv := &RegexpPosition{
		*NewBinaryFunctionBase("regexp_position", first, second),
		nil,
	}

	rv.re, _ = precompileRegexp(second, false)
	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *RegexpPosition) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type NUMBER.
*/
func (this *RegexpPosition) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *RegexpPosition) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method takes in two values and returns a value that
corresponds to the first position of the regular expression
pattern (already set or populated using the second value)
in the first string value, or -1 if it isnt found. If the
input type is missing return missing, and if it isnt
string then return null value. Use the FindStringIndex
method in the regexp package to return a two-element slice
of integers defining the location of the leftmost match in
the string of the regular expression as per the Go Docs. Return
the first element of this slice as a value. If a FindStringIndex
returns nil, then the regexp pattern isnt found. Hence return -1.
*/
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

/*
The constructor returns a NewRegexpPosition with the two operands
cast to a Function as the FunctionConstructor.
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
If n is given, at most n replacements are performed. Type
RegexpReplace is a struct that implements FunctionBase.
It has a field that represents a regular expression. Regexp
is the representation of a compiled regular expression.
*/
type RegexpReplace struct {
	FunctionBase
	re *regexp.Regexp
}

/*
The function NewRegexpReplace calls NewFunctionBase to
create a function named REGEXP_REPLACE with the two
expressions as input. It calls precompile to populate the
regexp field in the struct.
*/
func NewRegexpReplace(operands ...Expression) Function {
	rv := &RegexpReplace{
		*NewFunctionBase("regexp_replace", operands...),
		nil,
	}

	rv.re, _ = precompileRegexp(operands[1], false)
	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *RegexpReplace) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type STRING.
*/
func (this *RegexpReplace) Type() value.Type { return value.STRING }

/*
Calls the Eval method for functions and passes in the
receiver, current item and current context.
*/
func (this *RegexpReplace) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
For Regexp Replace there can be either 3 or 4 input arguments.
It searches for occurences of the regular expression pattern
(representing the substring already set or populated using the
second value) in the first arg (the expr) and replaces it with
the third arg (the repacer). If there are only 3 args then use
the ReplaceAllLiteralString method in the Regexp package to
return a string after replacing matches of the Regexp with the
replacement string. If the fourth arg exists it contains
the replacements to a maximum number. Make sure its an integer
value, and call ReplaceAllString. Keep track of the count.
If either of the first three input arg values are missing
return a missing, or if not a string return a null value. If
the fourth input arg is not a number return a null value.
*/
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
The constructor returns a NewRegexpReplace as the FunctionConstructor.
*/
func (this *RegexpReplace) Constructor() FunctionConstructor { return NewRegexpReplace }

/*
This method compiles and sets the regular expression re
later used to set the field in the RegexpReplace structure.
If the input expression value is nil or type string, return.
If not then call Compile with the value, to parse a regular
expression and return, if successful, a Regexp object that
can be used to match against text.
*/
func precompileRegexp(rexpr Expression, full bool) (re *regexp.Regexp, err error) {
	rv := rexpr.Value()
	if rv == nil || rv.Type() != value.STRING {
		return
	}

	s := rv.Actual().(string)
	if full {
		s = "^" + s + "$"
	}

	re, err = regexp.Compile(s)
	return
}
