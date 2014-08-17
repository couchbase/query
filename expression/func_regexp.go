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
	"fmt"
	"math"
	"regexp"

	"github.com/couchbaselabs/query/value"
)

type RegexpContains struct {
	reBinaryBase
}

func NewRegexpContains(first, second Expression) Function {
	return &RegexpContains{
		reBinaryBase{
			binaryBase: binaryBase{
				first:  first,
				second: second,
			},
		},
	}
}

func (this *RegexpContains) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *RegexpContains) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *RegexpContains) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *RegexpContains) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *RegexpContains) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *RegexpContains) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *RegexpContains) eval(first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	f := first.Actual().(string)
	s := second.Actual().(string)

	re := this.re
	if re == nil {
		var e error
		re, e = regexp.Compile(s)
		if e != nil {
			return nil, e
		}
	}

	return value.NewValue(re.FindStringIndex(f) != nil), nil
}

func (this *RegexpContains) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewRegexpContains(args[0], args[1])
	}
}

type RegexpLike struct {
	reBinaryBase
}

func NewRegexpLike(first, second Expression) Function {
	return &RegexpLike{
		reBinaryBase{
			binaryBase: binaryBase{
				first:  first,
				second: second,
			},
		},
	}
}

func (this *RegexpLike) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *RegexpLike) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *RegexpLike) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *RegexpLike) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *RegexpLike) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *RegexpLike) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *RegexpLike) eval(first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	f := first.Actual().(string)
	s := second.Actual().(string)

	re := this.re
	if re == nil {
		var e error
		re, e = regexp.Compile(s)
		if e != nil {
			return nil, e
		}
	}

	return value.NewValue(re.MatchString(f)), nil
}

func (this *RegexpLike) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewRegexpLike(args[0], args[1])
	}
}

type RegexpPosition struct {
	reBinaryBase
}

func NewRegexpPosition(first, second Expression) Function {
	return &RegexpPosition{
		reBinaryBase{
			binaryBase: binaryBase{
				first:  first,
				second: second,
			},
		},
	}
}

func (this *RegexpPosition) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *RegexpPosition) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *RegexpPosition) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *RegexpPosition) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *RegexpPosition) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *RegexpPosition) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *RegexpPosition) eval(first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	f := first.Actual().(string)
	s := second.Actual().(string)

	re := this.re
	if re == nil {
		var e error
		re, e = regexp.Compile(s)
		if e != nil {
			return nil, e
		}
	}

	loc := re.FindStringIndex(f)
	if loc == nil {
		return value.NewValue(-1.0), nil
	}

	return value.NewValue(float64(loc[0])), nil
}

func (this *RegexpPosition) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewRegexpPosition(args[0], args[1])
	}
}

type RegexpReplace struct {
	nAryBase
	re *regexp.Regexp
}

func NewRegexpReplace(args Expressions) Function {
	return &RegexpReplace{
		nAryBase: nAryBase{
			operands: args,
		},
	}
}

func (this *RegexpReplace) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *RegexpReplace) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *RegexpReplace) Fold() (Expression, error) {
	var e error
	this.operands[1], e = this.operands[1].Fold()
	if e != nil {
		return nil, e
	}

	switch s := this.operands[1].(type) {
	case *Constant:
		sv := s.Value()
		if sv.Type() == value.MISSING {
			return NewConstant(value.MISSING_VALUE), nil
		} else if sv.Type() != value.STRING {
			sa := sv.Actual()
			return nil, fmt.Errorf("Invalid REGEXP pattern %v of type %T.", sa, sa)
		}

		re, e := regexp.Compile(sv.Actual().(string))
		if e != nil {
			return nil, e
		}

		this.re = re
	}

	return this.fold(this)
}

func (this *RegexpReplace) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *RegexpReplace) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *RegexpReplace) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *RegexpReplace) eval(args value.Values) (value.Value, error) {
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
		var e error
		re, e = regexp.Compile(s)
		if e != nil {
			return nil, e
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

func (this *RegexpReplace) MinArgs() int { return 3 }

func (this *RegexpReplace) MaxArgs() int { return 4 }

func (this *RegexpReplace) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewRegexpReplace(args)
	}
}
