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
	"regexp"

	"github.com/couchbaselabs/query/value"
)

// Binary operators.
type binary interface {
	Expression
	evaluate(first, second value.Value) (value.Value, error)
}

type binaryBase struct {
	ExpressionBase
	first  Expression
	second Expression
}

func (this *binaryBase) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, e := this.first.Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	second, e := this.second.Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	return binary(this).evaluate(first, second)
}

func (this *binaryBase) Fold() (Expression, error) {
	t, e := Expression(this).VisitChildren(&Folder{})
	if e != nil {
		return t, e
	}

	switch f := this.first.(type) {
	case *Constant:
		switch s := this.second.(type) {
		case *Constant:
			v, e := binary(this).evaluate(f.Value(), s.Value())
			if e == nil {
				return NewConstant(v), nil
			}
		}
	}

	return this, nil
}

func (this *binaryBase) Children() Expressions {
	return Expressions{this.first, this.second}
}

func (this *binaryBase) VisitChildren(visitor Visitor) (Expression, error) {
	var e error

	this.first, e = visitor.Visit(this.first)
	if e != nil {
		return nil, e
	}

	this.second, e = visitor.Visit(this.second)
	if e != nil {
		return nil, e
	}

	return this, nil
}

func (this *binaryBase) MinArgs() int { return 2 }

func (this *binaryBase) MaxArgs() int { return 2 }

func (this *binaryBase) evaluate(first, second value.Value) (value.Value, error) {
	panic("Must override.")
}

type reBinaryBase struct {
	binaryBase
	re *regexp.Regexp
}

func (this *reBinaryBase) Fold() (Expression, error) {
	var e error
	this.second, e = this.second.Fold()
	if e != nil {
		return nil, e
	}

	switch s := this.second.(type) {
	case *Constant:
		sv := s.Value()
		if sv.Type() == value.MISSING {
			return NewConstant(value.MISSING_VALUE), nil
		} else if sv.Type() != value.STRING {
			sa := sv.Actual()
			return nil, fmt.Errorf("Invalid LIKE pattern %v of type %T.", sa, sa)
		}

		re, e := regexp.Compile(sv.Actual().(string))
		if e != nil {
			return nil, e
		}

		this.re = re
	}

	return this.binaryBase.Fold()
}
