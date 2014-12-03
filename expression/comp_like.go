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

	"github.com/couchbaselabs/query/value"
)

/*
Comparison terms allow for comparing two expressions.
Like and not like are used to to search for a specified
pattern in an expression. The LIKE operator allows for
wildcard matching of string values. The right-hand side
of the operator is a pattern, optionally containg '%'
and '_' wildcard characters. Type Like is a struct that
implements BinaryFunctionBase. It has a field that
represents a regular expression. Regexp is the
representation of a compiled regular expression.
*/
type Like struct {
	BinaryFunctionBase
	re *regexp.Regexp
}

/*
The function NewLike calls NewBinaryFunctionBase
to define the like comparison expression with input
operand expressions first and second, as input.
*/
func NewLike(first, second Expression) Function {
	rv := &Like{
		*NewBinaryFunctionBase("like", first, second),
		nil,
	}

	rv.Precompile()
	rv.expr = rv
	return rv
}

/*
It calls the VisitLike method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Like) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitLike(this)
}

/*
It returns a value type BOOLEAN.
*/
func (this *Like) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for Binary functions and passes in the
receiver, current item and current context.
*/
func (this *Like) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method evaluates the between comparison operation and returns a
value representing if first value is Like the second.If any of
the input operands are missing, return missing, and if null return null.
Convert the first and second values into valid Go values. Set the
regular expression variable re as the Compiled value of the second
value s. Use the MatchString method from the Regexp package to
compare it to the first expression. MatchString checks whether a
textual regular expression matches a string. Return its return value.
*/
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
		re, err = this.Compile(s)
		if err != nil {
			return nil, err
		}
	}

	return value.NewValue(re.MatchString(f)), nil
}

/*
The constructor returns a NewLike with the operands
cast to a Function as the FunctionConstructor.
*/
func (this *Like) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLike(operands[0], operands[1])
	}
}

/*
Return the regular expression field from the Like structure
from the receiver.
*/
func (this *Like) Regexp() *regexp.Regexp { return this.re }

/*
This method sets the regexp field in the Like struct.
It checks if its value is a String,
and if not returns. Compile the string and set the regular
expression field in the struct for the receiver to this
compiled value.
*/
func (this *Like) Precompile() {
	sv := this.Second().Value()
	if sv == nil || sv.Type() != value.STRING {
		return
	}

	re, err := this.Compile(sv.Actual().(string))
	if err != nil {
		return
	}

	this.re = re
}

/*
This method compiles the input string s into a regular
expression and returns it. Before this, use the
MustCompile method from the Regexp package, which
parses a regular expression and returns
a Regexp object that can be used to match against text.
Using this value, call the ReplaceAllStringFunc, which
as per the Go docs, returns a copy of src in which all
matches of the Regexp have been replaced by the return
value of function replacer applied to the matched substring.
*/
func (this *Like) Compile(s string) (*regexp.Regexp, error) {
	repl := regexp.MustCompile("\\\\|\\_|\\%|_|%")
	s = repl.ReplaceAllStringFunc(s, replacer)

	re, err := regexp.Compile(s)
	if err != nil {
		return nil, err
	}

	return re, nil
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
	case "\\\\":
		return "\\"
	case "\\_":
		return "_"
	case "\\%":
		return "%"
	case "_":
		return "(.)"
	case "%":
		return "(.*)"
	default:
		panic("Unknown regexp replacer " + s)
	}
}

/*
This function implements the not like operation. It calls
the NewLike method to return an expression that
is a complement of the its return type (boolean).
(NewNot represents the Not logical operation)
*/
func NewNotLike(first, second Expression) Expression {
	return NewNot(NewLike(first, second))
}
