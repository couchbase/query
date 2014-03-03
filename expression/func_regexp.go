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

func (this *RegexpContains) evaluate(first, second value.Value) (value.Value, error) {
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

func (this *RegexpLike) evaluate(first, second value.Value) (value.Value, error) {
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

func (this *RegexpPosition) evaluate(first, second value.Value) (value.Value, error) {
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

func NewRegexpReplace(operands Expressions) Function {
	return &RegexpReplace{
		nAryBase: nAryBase{
			operands: operands,
		},
	}
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

	return this.nAryBase.Fold()
}

func (this *RegexpReplace) evaluate(operands value.Values) (value.Value, error) {
	null := false
	for i := 0; i < 3; i++ {
		if operands[i].Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if operands[i].Type() != value.STRING {
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	if len(operands) == 4 && operands[3].Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	f := operands[0].Actual().(string)
	s := operands[1].Actual().(string)
	r := operands[2].Actual().(string)

	re := this.re
	if re == nil {
		var e error
		re, e = regexp.Compile(s)
		if e != nil {
			return nil, e
		}
	}

	if len(operands) == 3 {
		return value.NewValue(re.ReplaceAllLiteralString(f, r)), nil
	}

	nf := operands[3].Actual().(float64)
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
