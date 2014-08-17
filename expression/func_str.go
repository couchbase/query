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

	"github.com/couchbaselabs/query/value"
)

type Contains struct {
	binaryBase
}

func NewContains(first, second Expression) Function {
	return &Contains{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *Contains) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Contains) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Contains) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Contains) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Contains) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Contains) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Contains) eval(first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := strings.Contains(first.Actual().(string), second.Actual().(string))
	return value.NewValue(rv), nil
}

func (this *Contains) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewContains(args[0], args[1])
	}
}

type Length struct {
	unaryBase
}

func NewLength(arg Expression) Function {
	return &Length{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *Length) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Length) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Length) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Length) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Length) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Length) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Length) eval(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := len(arg.Actual().(string))
	return value.NewValue(float64(rv)), nil
}

func (this *Length) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewLength(args[0])
	}
}

type Lower struct {
	unaryBase
}

func NewLower(arg Expression) Function {
	return &Lower{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *Lower) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Lower) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Lower) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Lower) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Lower) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Lower) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Lower) eval(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := strings.ToLower(arg.Actual().(string))
	return value.NewValue(rv), nil
}

func (this *Lower) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewLower(args[0])
	}
}

type LTrim struct {
	nAryBase
}

func NewLTrim(args Expressions) Function {
	return &LTrim{
		nAryBase{
			operands: args,
		},
	}
}

func (this *LTrim) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *LTrim) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *LTrim) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *LTrim) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *LTrim) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *LTrim) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *LTrim) MinArgs() int { return 1 }

func (this *LTrim) MaxArgs() int { return 2 }

func (this *LTrim) eval(args value.Values) (value.Value, error) {
	null := false
	for _, o := range args {
		if o.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if o.Type() != value.STRING {
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

func (this *LTrim) Constructor() FunctionConstructor { return NewLTrim }

var _WHITESPACE = value.NewValue(" \t\n\f\r")

type Position struct {
	binaryBase
}

func NewPosition(first, second Expression) Function {
	return &Position{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *Position) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Position) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Position) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Position) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Position) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Position) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Position) eval(first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := strings.Index(first.Actual().(string), second.Actual().(string))
	return value.NewValue(float64(rv)), nil
}

func (this *Position) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewPosition(args[0], args[1])
	}
}

type Repeat struct {
	binaryBase
}

func NewRepeat(first, second Expression) Function {
	return &Repeat{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *Repeat) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Repeat) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Repeat) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Repeat) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Repeat) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Repeat) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Repeat) eval(first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	nf := second.Actual().(float64)
	if nf != math.Trunc(nf) {
		return value.NULL_VALUE, nil
	}

	rv := strings.Repeat(first.Actual().(string), int(nf))
	return value.NewValue(rv), nil
}

func (this *Repeat) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewRepeat(args[0], args[1])
	}
}

type Replace struct {
	nAryBase
}

func NewReplace(args Expressions) Function {
	return &Replace{
		nAryBase{
			operands: args,
		},
	}
}

func (this *Replace) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Replace) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Replace) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Replace) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Replace) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Replace) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Replace) eval(args value.Values) (value.Value, error) {
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

func (this *Replace) MinArgs() int { return 3 }

func (this *Replace) MaxArgs() int { return 4 }

func (this *Replace) Constructor() FunctionConstructor { return NewReplace }

type RTrim struct {
	nAryBase
}

func NewRTrim(args Expressions) Function {
	return &RTrim{
		nAryBase{
			operands: args,
		},
	}
}

func (this *RTrim) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *RTrim) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *RTrim) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *RTrim) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *RTrim) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *RTrim) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *RTrim) MinArgs() int { return 1 }

func (this *RTrim) MaxArgs() int { return 2 }

func (this *RTrim) eval(args value.Values) (value.Value, error) {
	null := false
	for _, o := range args {
		if o.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if o.Type() != value.STRING {
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

func (this *RTrim) Constructor() FunctionConstructor { return NewRTrim }

type Split struct {
	nAryBase
}

func NewSplit(args Expressions) Function {
	return &Split{
		nAryBase{
			operands: args,
		},
	}
}

func (this *Split) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Split) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Split) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Split) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Split) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Split) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Split) MinArgs() int { return 1 }

func (this *Split) MaxArgs() int { return 2 }

func (this *Split) eval(args value.Values) (value.Value, error) {
	null := false
	for _, o := range args {
		if o.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if o.Type() != value.STRING {
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

func (this *Split) Constructor() FunctionConstructor { return NewSplit }

type Substr struct {
	nAryBase
}

func NewSubstr(args Expressions) Function {
	return &Substr{
		nAryBase{
			operands: args,
		},
	}
}

func (this *Substr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Substr) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Substr) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Substr) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Substr) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Substr) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Substr) MinArgs() int { return 2 }

func (this *Substr) MaxArgs() int { return 3 }

func (this *Substr) eval(args value.Values) (value.Value, error) {
	null := false
	if args[0].Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if args[0].Type() != value.STRING {
		null = true
	}

	for i := 1; i < len(args); i++ {
		if args[i].Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if args[i].Type() != value.NUMBER {
			null = true
		} else {
			vf := args[i].Actual().(float64)
			if vf != math.Trunc(vf) {
				null = true
			}
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

func (this *Substr) Constructor() FunctionConstructor { return NewSubstr }

type Title struct {
	unaryBase
}

func NewTitle(arg Expression) Function {
	return &Title{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *Title) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Title) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Title) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Title) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Title) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Title) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Title) eval(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := strings.ToTitle(arg.Actual().(string))
	return value.NewValue(rv), nil
}

func (this *Title) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewTitle(args[0])
	}
}

type Trim struct {
	nAryBase
}

func NewTrim(args Expressions) Function {
	return &Trim{
		nAryBase{
			operands: args,
		},
	}
}

func (this *Trim) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Trim) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Trim) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Trim) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Trim) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Trim) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Trim) MinArgs() int { return 1 }

func (this *Trim) MaxArgs() int { return 2 }

func (this *Trim) eval(args value.Values) (value.Value, error) {
	null := false
	for _, o := range args {
		if o.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if o.Type() != value.STRING {
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

func (this *Trim) Constructor() FunctionConstructor { return NewTrim }

type Upper struct {
	unaryBase
}

func NewUpper(arg Expression) Function {
	return &Upper{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *Upper) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Upper) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Upper) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Upper) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Upper) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Upper) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Upper) eval(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := strings.ToUpper(arg.Actual().(string))
	return value.NewValue(rv), nil
}

func (this *Upper) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewUpper(args[0])
	}
}
