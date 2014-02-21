//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"regexp"

	"github.com/couchbaselabs/query/value"
)

type Like struct {
	binaryBase
	re *regexp.Regexp
}

func NewLike(first, second Expression) Expression {
	return &Like{
		binaryBase: binaryBase{
			first:  first,
			second: second,
		},
		re: nil,
	}
}

func (this *Like) Fold() Expression {
	this.first = this.first.Fold()
	this.second = this.second.Fold()

	switch s := this.second.(type) {
	case *Constant:
		sv := s.Value()
		if sv.Type() == value.MISSING {
			return NewConstant(_MISSING_VALUE)
		} else if sv.Type() != value.STRING {
			return NewConstant(_NULL_VALUE)
		}

		re, err := this.compile(sv.Actual().(string))
		if err != nil {
			return this
		}

		this.re = re

		switch f := this.first.(type) {
		case *Constant:
			v, e := this.evaluate(f.Value(), sv)
			if e == nil {
				return NewConstant(v)
			}
		}
	}

	return this
}

func (this *Like) evaluate(first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return _MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return _NULL_VALUE, nil
	}

	f := first.Actual().(string)
	s := second.Actual().(string)

	re := this.re
	if re == nil {
		var e error
		re, e = this.compile(s)
		if e != nil {
			return nil, e
		}
	}

	return value.NewValue(re.MatchString(f)), nil
}

func (this *Like) compile(s string) (*regexp.Regexp, error) {
	repl := regexp.MustCompile("\\\\|\\_|\\%|_|%")
	s = repl.ReplaceAllStringFunc(s, replacer)

	re, err := regexp.Compile(s)
	if err != nil {
		return nil, err
	}

	return re, nil
}

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

func NewNotLike(first, second Expression) Expression {
	return NewNot(NewLike(first, second))
}
