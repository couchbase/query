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
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// ToArray
//
///////////////////////////////////////////////////

/*
This represents the type conversion function TO_ARRAY(expr).
It returns an array where a missing, null and arrays map
to themselves and other non-binary values are wrapped in an array.
*/
type ToArray struct {
	UnaryFunctionBase
}

func NewToArray(operand Expression) Function {
	rv := &ToArray{
		*NewUnaryFunctionBase("to_array", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ToArray) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ToArray) Type() value.Type { return value.ARRAY }

func (this *ToArray) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() <= value.NULL {
		return arg, nil
	} else if arg.Type() == value.ARRAY {
		return arg, nil
	} else if arg.Type() == value.BINARY {
		return value.NULL_VALUE, nil
	}

	return value.NewValue([]interface{}{arg.Actual()}), nil
}

/*
Factory method pattern.
*/
func (this *ToArray) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewToArray(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ToAtom
//
///////////////////////////////////////////////////

/*
This represents the type conversion function TO_ATOM(expr).
It returns atomic values where, missing, null, boolean,
numbers and strings, are themselves, arrays of length 1
are the result of TO_ATOM() on their single element and
objects of length 1 are the result of TO_ATOM() on their
single value. All other values return null.
*/
type ToAtom struct {
	UnaryFunctionBase
}

func NewToAtom(operand Expression) Function {
	rv := &ToAtom{
		*NewUnaryFunctionBase("to_atom", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ToAtom) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ToAtom) Type() value.Type {
	t := this.Operand().Type()
	if t < value.ARRAY || t == value.BINARY {
		return t
	} else {
		return value.JSON
	}
}

func (this *ToAtom) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	return this.DoEvaluate(context, arg)
}

// needed for recursion
func (this *ToAtom) DoEvaluate(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() < value.ARRAY || arg.Type() == value.BINARY {
		return arg, nil
	} else {
		switch a := arg.Actual().(type) {
		case []interface{}:
			if len(a) == 1 {
				return this.DoEvaluate(context, value.NewValue(a[0]))
			}
		case map[string]interface{}:
			if len(a) == 1 {
				for _, v := range a {
					return this.DoEvaluate(context, value.NewValue(v))
				}
			}
		}
	}

	return value.NULL_VALUE, nil
}

/*
Factory method pattern.
*/
func (this *ToAtom) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewToAtom(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ToBoolean
//
///////////////////////////////////////////////////

/*
This represents the type conversion function TO_BOOL(expr).
It returns boolean values where missing, null, false map to
themselves. Numbers +0, -0 and NaN, empty strings, arrays
and objects as expr map to false. All other values are
true.
*/
type ToBoolean struct {
	UnaryFunctionBase
}

func NewToBoolean(operand Expression) Function {
	rv := &ToBoolean{
		*NewUnaryFunctionBase("to_boolean", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ToBoolean) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ToBoolean) Type() value.Type { return value.BOOLEAN }

func (this *ToBoolean) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	switch arg.Type() {
	case value.MISSING, value.NULL, value.BOOLEAN:
		return arg, nil
	default:
		switch a := arg.Actual().(type) {
		case float64:
			return value.NewValue(!math.IsNaN(a) && a != 0), nil
		case string:
			return value.NewValue(len(a) > 0), nil
		case []byte:
			return value.NewValue(len(a) > 0), nil
		case []interface{}:
			return value.NewValue(len(a) > 0), nil
		case map[string]interface{}:
			return value.NewValue(len(a) > 0), nil
		default:
			return value.NULL_VALUE, nil
		}
	}
}

/*
Factory method pattern.
*/
func (this *ToBoolean) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewToBoolean(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ToNumber
//
///////////////////////////////////////////////////

/*
This represents the type conversion function TO_NUM(expr).
It returns number values where missing, null, and numbers
map to themselves. False is 0, true is 1, strings that
parse as numbers are those numbers and all other values
are null (For e.g. "123" is 123 but "a12" will be NULL).
*/
type ToNumber struct {
	FunctionBase
}

func NewToNumber(operands ...Expression) Function {
	rv := &ToNumber{
		*NewFunctionBase("to_number", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ToNumber) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ToNumber) Type() value.Type { return value.NUMBER }

func (this *ToNumber) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	switch arg.Type() {
	case value.MISSING, value.NULL, value.NUMBER:
		return arg, nil
	case value.BOOLEAN:
		a := arg.Actual().(bool)
		if a {
			return value.ONE_VALUE, nil
		}
		return value.ZERO_VALUE, nil
	}

	if arg.Type() == value.STRING {
		s := arg.ToString()
		if len(this.operands) > 1 {
			strip := ""
			sarg, err := this.operands[1].Evaluate(item, context)
			if err != nil {
				return nil, err
			}
			if sarg.Type() == value.STRING {
				strip = sarg.Actual().(string)
			} else {
				return value.NULL_VALUE, nil
			}
			decimalCommaPos := -1
			res := make([]rune, 0, len(s))
			for _, r := range s {
				if r == '.' && decimalCommaPos >= 0 {
					decimalCommaPos = -2
				}
				if !unicode.IsSpace(r) && (strip == "" || strings.IndexRune(strip, r) == -1) {
					if r == ',' {
						if decimalCommaPos == -1 {
							decimalCommaPos = len(res)
						} else {
							decimalCommaPos = -2
						}
					} else if r == '.' {
						decimalCommaPos = -2
					}
					res = append(res, r)
				}
			}
			if len(res) == 0 {
				return value.NULL_VALUE, nil
			}
			if decimalCommaPos >= 0 {
				res[decimalCommaPos] = '.'
			}
			s = string(res)
		}

		i, err := strconv.ParseInt(s, 10, 64)
		if err == nil && ((i > math.MinInt64 && i < math.MaxInt64) || strconv.FormatInt(i, 10) == s) {
			return value.NewValue(i), nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err == nil {
			return value.NewValue(f), nil
		}
	}

	return value.NULL_VALUE, nil
}

func (this *ToNumber) Constructor() FunctionConstructor {
	return NewToNumber
}

func (this *ToNumber) MinArgs() int { return 1 }

func (this *ToNumber) MaxArgs() int { return 2 }

///////////////////////////////////////////////////
//
// ToObject
//
///////////////////////////////////////////////////

/*
This represents the type conversion function TOOBJ(expr).
It returns an object value. The input of types missing,
null and object return themselves. For all other values,
return an _EMPTY_OBJECT value.
*/
type ToObject struct {
	UnaryFunctionBase
}

func NewToObject(operand Expression) Function {
	rv := &ToObject{
		*NewUnaryFunctionBase("to_object", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ToObject) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ToObject) Type() value.Type { return value.OBJECT }

var _EMPTY_OBJECT = value.NewValue(map[string]interface{}{})

func (this *ToObject) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	switch arg.Type() {
	case value.MISSING, value.NULL, value.OBJECT:
		return arg, nil
	}

	return _EMPTY_OBJECT, nil
}

/*
Factory method pattern.
*/
func (this *ToObject) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewToObject(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ToString
//
///////////////////////////////////////////////////

/*
This represents the type conversion function TOSTR(expr).
It returns a string based on the input expr value. Values
missing, null and strings return themselves. False, true
(boolean) and numbers return their string representation.
All other values map to null.
*/
type ToString struct {
	UnaryFunctionBase
}

func NewToString(operand Expression) Function {
	rv := &ToString{
		*NewUnaryFunctionBase("to_string", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ToString) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ToString) Type() value.Type { return value.STRING }

func (this *ToString) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	switch arg.Type() {
	case value.MISSING, value.NULL, value.STRING:
		return arg, nil
	case value.BOOLEAN:
		return value.NewValue(fmt.Sprint(arg.Actual())), nil
	case value.NUMBER:
		var s string
		actual := arg.ActualForIndex()
		switch actual := actual.(type) {
		case float64:
			s = strconv.FormatFloat(actual, 'f', -1, 64)
		case int64:
			s = strconv.FormatInt(actual, 10)
		}
		return value.NewValue(s), nil
	case value.BINARY:
		raw, ok := arg.Actual().([]byte)
		if !ok {
			return value.NULL_VALUE, nil
		}

		s := string(raw)
		return value.NewValue(s), nil
	default:
		return value.NULL_VALUE, nil
	}
}

/*
Factory method pattern.
*/
func (this *ToString) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewToString(operands[0])
	}
}
