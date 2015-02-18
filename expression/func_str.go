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
	"strings"

	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// Contains
//
///////////////////////////////////////////////////

/*
This represents the String function CONTAINS(expr, substr).
It returns true if the string contains the substring. Type
Contains is a struct that implements BinaryFunctionBase.
*/
type Contains struct {
	BinaryFunctionBase
}

/*
The function NewContains calls NewBinaryFunctionBase to
create a function named CONTAINS with the two
expressions as input.
*/
func NewContains(first, second Expression) Function {
	rv := &Contains{
		*NewBinaryFunctionBase("contains", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Contains) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type BOOLEAN.
*/
func (this *Contains) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *Contains) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method takes in two values and returns new value that returns a boolean
value that depicts if the second value is contained within the first. If
either of the input values are missing, return a missing value, and if they
arent strings then return a null value. Use the Contains method from the
string package to return a boolean value that is true if substring (second)
is within the string(first).
*/
func (this *Contains) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := strings.Contains(first.Actual().(string), second.Actual().(string))
	return value.NewValue(rv), nil
}

/*
The constructor returns a NewContains with the two operands
cast to a Function as the FunctionConstructor.
*/
func (this *Contains) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewContains(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// Length
//
///////////////////////////////////////////////////

/*
This represents the String function LENGTH(expr). It
returns the length of the string value. Type Length
is a struct that implements UnaryFunctionBase.
*/
type Length struct {
	UnaryFunctionBase
}

/*
The function NewLength calls NewUnaryFunctionBase to
create a function named LENGTH with an expression as
input.
*/
func NewLength(operand Expression) Function {
	rv := &Length{
		*NewUnaryFunctionBase("length", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Length) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type NUMBER.
*/
func (this *Length) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Length) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an argument value and returns its length
as value. If the input type is missing return missing, and if
it isnt string then return null value. Use the len method to
return the length of the input string. Convert it into valid
N1QL value and return.
*/
func (this *Length) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := len(arg.Actual().(string))
	return value.NewValue(float64(rv)), nil
}

/*
The constructor returns a NewLength with the an operand
cast to a Function as the FunctionConstructor.
*/
func (this *Length) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLength(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Lower
//
///////////////////////////////////////////////////

/*
This represents the String function LOWER(expr). It returns
the lowercase of the string value. Type Lower is a struct
that implements UnaryFunctionBase.
*/
type Lower struct {
	UnaryFunctionBase
}

/*
The function NewLower calls NewUnaryFunctionBase to
create a function named LOWER with an expression as
input.
*/
func NewLower(operand Expression) Function {
	rv := &Lower{
		*NewUnaryFunctionBase("lower", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Lower) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type STRING.
*/
func (this *Lower) Type() value.Type { return value.STRING }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Lower) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an argument value and returns a
lowercase string as value. If the input type is
missing return missing, and if it isnt string then
return null value. Use the ToLower method to
convert the string to lower case on a valid Go type
from the Actual method on the argument value. Return
this lower case string as Value.
*/
func (this *Lower) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := strings.ToLower(arg.Actual().(string))
	return value.NewValue(rv), nil
}

/*
The constructor returns a NewLower with the an operand
cast to a Function as the FunctionConstructor.
*/
func (this *Lower) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLower(operands[0])
	}
}

///////////////////////////////////////////////////
//
// LTrim
//
///////////////////////////////////////////////////

/*
This represents the String function LTRIM(expr [, chars ]).
It returns a string with all leading chars removed
(whitespace by default). Type NewLTrim is a struct that
implements FunctionBase.
*/
type LTrim struct {
	FunctionBase
}

/*
The function NewLTrim calls NewFunctionBase to create a
function named LTRIM with input arguments as the
operands from the input expression.
*/
func NewLTrim(operands ...Expression) Function {
	rv := &LTrim{
		*NewFunctionBase("ltrim", operands...),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *LTrim) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type STRING.
*/
func (this *LTrim) Type() value.Type { return value.STRING }

/*
Calls the Eval method for the receiver and passes in the
receiver, current item and current context.
*/
func (this *LTrim) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
This method takes in input arguments and returns a value that
is a string with the leading chars removed. Range over the
args, if its type is missing, return missing. If the argument
type is not a string, set boolean null as true. If null is
true it indicates that one of the args is not a string and
hence return a null value. If not, all input arguments are
strings. If there is more than 1 input arg, use that value
to call TrimLeft method from the strings method and trim that
value from the input string. Return this trimmed value.
*/
func (this *LTrim) Apply(context Context, args ...value.Value) (value.Value, error) {
	null := false

	for _, a := range args {
		if a.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if a.Type() != value.STRING {
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	chars := _WHITESPACE
	if len(args) > 1 {
		chars = args[1]
	}

	rv := strings.TrimLeft(args[0].Actual().(string), chars.Actual().(string))
	return value.NewValue(rv), nil
}

/*
Minimum input arguments required for the LTRIM function
is 1.
*/
func (this *LTrim) MinArgs() int { return 1 }

/*
Maximum input arguments required for the LTRIM function
is 2.
*/
func (this *LTrim) MaxArgs() int { return 2 }

/*
Return NewLTrim as FunctionConstructor.
*/
func (this *LTrim) Constructor() FunctionConstructor { return NewLTrim }

/*
Define variable whitespace that constructs a value from
' ','\t','\n','\f' and '\r'.
*/
var _WHITESPACE = value.NewValue(" \t\n\f\r")

///////////////////////////////////////////////////
//
// Position
//
///////////////////////////////////////////////////

/*
This represents the String function POSITION(expr, substr).
It returns the first position of the substring within the
string, or -1. The position is 0-based. Type Position is a
struct that implements BinaryFunctionBase.
*/
type Position struct {
	BinaryFunctionBase
}

/*
The function NewPosition calls NewBinaryFunctionBase to
create a function named POSITION with two expressions as
input.
*/
func NewPosition(first, second Expression) Function {
	rv := &Position{
		*NewBinaryFunctionBase("position", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Position) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type NUMBER.
*/
func (this *Position) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *Position) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method takes in two values and returns a value that
corresponds to the second expressions position in the
first.  If the input type is missing return missing, and
if it isnt string then return null value. Use the Index
method defined by the strings package to calculate the
offset position of the second string. Return that value.
*/
func (this *Position) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := strings.Index(first.Actual().(string), second.Actual().(string))
	return value.NewValue(float64(rv)), nil
}

/*
The constructor returns a NewPosition with two operands
cast to a Function as the FunctionConstructor.
*/
func (this *Position) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewPosition(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// Repeat
//
///////////////////////////////////////////////////

/*
This represents the String function REPEAT(expr, n).
It returns string formed by repeating expr n times.
Type Repeat is a struct that implements BinaryFunctionBase.
*/
type Repeat struct {
	BinaryFunctionBase
}

/*
The function NewRepeat calls NewBinaryFunctionBase to
create a function named REPEAT with the two
expressions as input.
*/
func NewRepeat(first, second Expression) Function {
	rv := &Repeat{
		*NewBinaryFunctionBase("repeat", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Repeat) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type STRING.
*/
func (this *Repeat) Type() value.Type { return value.STRING }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *Repeat) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method returns a string value that repeats the first value
second number of times. If either of the input values are
missing, return a missing value, and if the first isnt a string
and the second isnt a number then return a null value. Check if the
number n is less than 0 and if it isnt an integer, then return null
value. Call the Repeat method from the strings package with the
string and number and return that stringvalue.
*/
func (this *Repeat) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	nf := second.Actual().(float64)
	if nf < 0.0 || nf != math.Trunc(nf) {
		return value.NULL_VALUE, nil
	}

	rv := strings.Repeat(first.Actual().(string), int(nf))
	return value.NewValue(rv), nil
}

/*
The constructor returns a NewRepeat with the two operands
cast to a Function as the FunctionConstructor.
*/
func (this *Repeat) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewRepeat(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// Replace
//
///////////////////////////////////////////////////

/*
This represents the String function REPLACE(expr, substr, repl [, n ]).
It returns a string with all occurences of substr replaced with repl.
If n is given, at most n replacements are performed. Replace is a type
struct that implements FunctionBase.
*/
type Replace struct {
	FunctionBase
}

/*
The function NewReplace calls NewFunctionBase to create a
function named REPLACE with input arguments as the
operands from the input expression.
*/
func NewReplace(operands ...Expression) Function {
	rv := &Replace{
		*NewFunctionBase("replace", operands...),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Replace) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type STRING.
*/
func (this *Replace) Type() value.Type { return value.STRING }

/*
Calls the Eval method for the receiver and passes in the
receiver, current item and current context.
*/
func (this *Replace) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
This method has input args that depict the string, what to replace it with
and the number of allowable replacements n. Loop over the arguments. If its
type is missing, return missing. If the argument type is not a string,
set boolean null as true. If any of the first 3 arguments are not a
string then return null. If there are 4 input values, and the 4th is not
a number return a null value. Make sure it is an absolute number, and if not
return a null value. If n is not present initialize it to -1 and use the
Replace method defined by the strings package. Return the final string value
after creating a valid N!QL value out of the string.
*/
func (this *Replace) Apply(context Context, args ...value.Value) (value.Value, error) {
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
	n := -1

	if len(args) == 4 {
		nf := args[3].Actual().(float64)
		if nf != math.Trunc(nf) {
			return value.NULL_VALUE, nil
		}

		n = int(nf)
	}

	rv := strings.Replace(f, s, r, n)
	return value.NewValue(rv), nil
}

/*
Minimum input arguments required for the REPLACE function
is 3.
*/
func (this *Replace) MinArgs() int { return 3 }

/*
Maximum input arguments required for the REPLACE function
is 4.
*/
func (this *Replace) MaxArgs() int { return 4 }

/*
Return NewReplace as FunctionConstructor.
*/
func (this *Replace) Constructor() FunctionConstructor { return NewReplace }

///////////////////////////////////////////////////
//
// RTrim
//
///////////////////////////////////////////////////

/*
This represents the String function RTRIM(expr, [, chars ]).
It returns a string with all trailing chars removed (whitespace
by default). RTrim is a type struct that implements FunctionBase.
*/
type RTrim struct {
	FunctionBase
}

/*
The function NewRTrim calls NewFunctionBase to create a
function named RTRIM with input arguments as the
operands from the input expression.
*/
func NewRTrim(operands ...Expression) Function {
	rv := &RTrim{
		*NewFunctionBase("rtrim", operands...),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *RTrim) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type STRING.
*/
func (this *RTrim) Type() value.Type { return value.STRING }

/*
Calls the Eval method for the receiver and passes in the
receiver, current item and current context.
*/
func (this *RTrim) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
This method takes in input arguments and returns a value that
is a string with the leading chars removed. Range over the
args, if its type is missing, return missing. If the argument
type is not a string, set boolean null as true. If null is
true it indicates that one of the args is not a string and
hence return a null value. If not, all input arguments are
strings. If there is more than 1 input arg, use that value
to call TrimRight method from the strings method and trim that
value from the input string. Return this trimmed value.
*/
func (this *RTrim) Apply(context Context, args ...value.Value) (value.Value, error) {
	null := false

	for _, a := range args {
		if a.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if a.Type() != value.STRING {
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	chars := _WHITESPACE
	if len(args) > 1 {
		chars = args[1]
	}

	rv := strings.TrimRight(args[0].Actual().(string), chars.Actual().(string))
	return value.NewValue(rv), nil
}

/*
Minimum input arguments required for the RTRIM function
is 1.
*/
func (this *RTrim) MinArgs() int { return 1 }

/*
Maximum input arguments required for the RTRIM function
is 2.
*/
func (this *RTrim) MaxArgs() int { return 2 }

/*
Return NewRTrim as FunctionConstructor.
*/
func (this *RTrim) Constructor() FunctionConstructor { return NewRTrim }

///////////////////////////////////////////////////
//
// Split
//
///////////////////////////////////////////////////

/*
This represents the String function SPLIT(expr [, sep ]).
It splits the string into an array of substrings separated
by sep. If sep is not given, any combination of whitespace
characters is used. Type Split is a struct that implements
FunctionBase.
*/
type Split struct {
	FunctionBase
}

/*
The function NewSplit calls NewFunctionBase to create a
function named SPLIT with input arguments as the
operands from the input expression.
*/
func NewSplit(operands ...Expression) Function {
	rv := &Split{
		*NewFunctionBase("split", operands...),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Split) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type STRING.
*/
func (this *Split) Type() value.Type { return value.STRING }

/*
Calls the Eval method for the receiver and passes in the
receiver, current item and current context.
*/
func (this *Split) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
In order to split the strings, range over the input arguments,
if its type is missing, return missing. If the argument
type is not a string, set boolean null as true. If null is
true it indicates that one of the args is not a string and
hence return a null value. If not, all input arguments are
strings. If there is more than 1 input arg, use the separator
and the string to call the split function from the strings
package. In the event there is no input then call the
Fields method which splits the string using whitespace
characters as defined by unicode. This returns a slice
of strings. We map it to an interface, convert it to a
valid N1QL value and return.
*/
func (this *Split) Apply(context Context, args ...value.Value) (value.Value, error) {
	null := false

	for _, a := range args {
		if a.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if a.Type() != value.STRING {
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	var sa []string
	if len(args) > 1 {
		sep := args[1]
		sa = strings.Split(args[0].Actual().(string),
			sep.Actual().(string))
	} else {
		sa = strings.Fields(args[0].Actual().(string))
	}

	rv := make([]interface{}, len(sa))
	for i, s := range sa {
		rv[i] = s
	}

	return value.NewValue(rv), nil
}

/*
Minimum input arguments required for the SPLIT function
is 1.
*/
func (this *Split) MinArgs() int { return 1 }

/*
Maximum input arguments required for the SPLIT function
is 2.
*/
func (this *Split) MaxArgs() int { return 2 }

/*
Return NewSplit as FunctionConstructor.
*/
func (this *Split) Constructor() FunctionConstructor { return NewSplit }

///////////////////////////////////////////////////
//
// Substr
//
///////////////////////////////////////////////////

/*
This represents the String function SUBSTR(expr, position [, length ]).
It returns a substring from the integer position of the given length,
or to the end of the string. The position is 0-based, i.e. the first
position is 0. If position is negative, it is counted from the end
of the string; -1 is the last position in the string. Type Substr is a
struct that implements FunctionBase.
*/
type Substr struct {
	FunctionBase
}

/*
The function Substr calls NewFunctionBase to create a
function named SUBSTR with input arguments as the
operands from the input expression.
*/
func NewSubstr(operands ...Expression) Function {
	rv := &Substr{
		*NewFunctionBase("substr", operands...),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Substr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type STRING.
*/
func (this *Substr) Type() value.Type { return value.STRING }

/*
Calls the Eval method for the receiver and passes in the
receiver, current item and current context.
*/
func (this *Substr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
This method returns a string from a start position to the end. It is a substring.
If the input argument value type is missing, then return a missing value, and if null
return a null value. Loop through all the input values, and check the types. If it is
a number type, then check if it is an absolute non floating point number. If not
return null value. If any value other than a number or missing, return a null.
If the position is negative calculate the actual offset by adding it to the length
of the string. If the length of input arguments is 2 or more, it means that the
start and end positions are given, hence return a value which is the
slice starting from that position until the end if specified.
*/
func (this *Substr) Apply(context Context, args ...value.Value) (value.Value, error) {
	null := false

	if args[0].Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if args[0].Type() != value.STRING {
		null = true
	}

	for i := 1; i < len(args); i++ {
		switch args[i].Type() {
		case value.MISSING:
			return value.MISSING_VALUE, nil
		case value.NUMBER:
			vf := args[i].Actual().(float64)
			if vf != math.Trunc(vf) {
				null = true
			}
		default:
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	str := args[0].Actual().(string)
	pos := int(args[1].Actual().(float64))

	if pos < 0 {
		pos = len(str) + pos
	}

	if pos < 0 || pos >= len(str) {
		return value.NULL_VALUE, nil
	}

	if len(args) == 2 {
		return value.NewValue(str[pos:]), nil
	}

	length := int(args[2].Actual().(float64))
	if length < 0 || pos+length > len(str) {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(str[pos : pos+length]), nil
}

/*
Minimum input arguments required for the SUBSTR function
is 2.
*/
func (this *Substr) MinArgs() int { return 2 }

/*
Maximum input arguments required for the SUBSTR function
is 3.
*/
func (this *Substr) MaxArgs() int { return 3 }

/*
Return NewSubstr as FunctionConstructor.
*/
func (this *Substr) Constructor() FunctionConstructor { return NewSubstr }

///////////////////////////////////////////////////
//
// Title
//
///////////////////////////////////////////////////

/*
This represents the String function TITLE(expr). It converts
the string so that the first letter of each word is uppercase
and every other letter is lowercase. Type Title is a struct
that implements UnaryFunctionBase.
*/
type Title struct {
	UnaryFunctionBase
}

/*
The function NewTitle calls NewUnaryFunctionBase to
create a function named TITLE with an expression as
input.
*/
func NewTitle(operand Expression) Function {
	rv := &Title{
		*NewUnaryFunctionBase("title", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Title) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type STRING.
*/
func (this *Title) Type() value.Type { return value.STRING }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Title) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an argument value and returns a string that has
the first letter of input words mapped to upper case. If the input
type is missing return missing, and if it isnt string then return
null value. Use the Title method from the strings package and
return it after conversion to a valid N1QL type.
*/
func (this *Title) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := strings.Title(arg.Actual().(string))
	return value.NewValue(rv), nil
}

/*
The constructor returns a NewTitle with the an operand
cast to a Function as the FunctionConstructor.
*/
func (this *Title) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewTitle(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Trim
//
///////////////////////////////////////////////////

/*
This represents the String function TRIM(expr [, chars ]).
It returns a string with all leading and trailing chars
removed (whitespace by default). Type NewTrim is a struct
that implements FunctionBase.
*/
type Trim struct {
	FunctionBase
}

/*
The function NewTrim calls NewFunctionBase to create a
function named TRIM with input arguments as the
operands from the input expression.
*/
func NewTrim(operands ...Expression) Function {
	rv := &Trim{
		*NewFunctionBase("trim", operands...),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Trim) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type STRING.
*/
func (this *Trim) Type() value.Type { return value.STRING }

/*
Calls the Eval method for the receiver and passes in the
receiver, current item and current context.
*/
func (this *Trim) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
This method takes in input arguments and returns a value that
is a string with the leading and trailing chars removed.
Range over the args, if its type is missing, return missing.
If the argument type is not a string, set boolean null as
true. If null is true it indicates that one of the args is
not a string and hence return a null value. If not, all
input arguments are strings. If there is more than 1
input arg, use that value to call Trim method from the
strings method and trim that value from the input string.
Return this trimmed value.
*/
func (this *Trim) Apply(context Context, args ...value.Value) (value.Value, error) {
	null := false

	for _, a := range args {
		if a.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if a.Type() != value.STRING {
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	chars := _WHITESPACE
	if len(args) > 1 {
		chars = args[1]
	}

	rv := strings.Trim(args[0].Actual().(string), chars.Actual().(string))
	return value.NewValue(rv), nil
}

/*
Minimum input arguments required for the TRIM function
is 1.
*/
func (this *Trim) MinArgs() int { return 1 }

/*
Maximum input arguments required for the TRIM function
is 2.
*/
func (this *Trim) MaxArgs() int { return 2 }

/*
Return NewTrim as FunctionConstructor.
*/
func (this *Trim) Constructor() FunctionConstructor { return NewTrim }

///////////////////////////////////////////////////
//
// Upper
//
///////////////////////////////////////////////////

/*
This represents the String function UPPER(expr). It returns
the uppercase of the string value. Type Upper is a struct
that implements UnaryFunctionBase.
*/
type Upper struct {
	UnaryFunctionBase
}

/*
The function NewUpper calls NewUnaryFunctionBase to
create a function named UPPER with an expression as
input.
*/
func NewUpper(operand Expression) Function {
	rv := &Upper{
		*NewUnaryFunctionBase("upper", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Upper) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type STRING.
*/
func (this *Upper) Type() value.Type { return value.STRING }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Upper) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an argument value and returns a
uppercase string as value. If the input type is
missing return missing, and if it isnt string then
return null value. Use the ToUpper method to
convert the string to upper case on a valid Go type
from the Actual method on the argument value. Return
this Upper case string as Value.
*/
func (this *Upper) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := strings.ToUpper(arg.Actual().(string))
	return value.NewValue(rv), nil
}

/*
The constructor returns a NewUpper with the an operand
cast to a Function as the FunctionConstructor.
*/
func (this *Upper) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewUpper(operands[0])
	}
}
