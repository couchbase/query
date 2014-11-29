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

type Like struct {
	BinaryFunctionBase
	re *regexp.Regexp
}

func NewLike(first, second Expression) Function {
	rv := &Like{
		*NewBinaryFunctionBase("like", first, second),
		nil,
	}

	rv.Precompile()
	rv.expr = rv
	return rv
}

func (this *Like) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitLike(this)
}

func (this *Like) Type() value.Type { return value.BOOLEAN }

func (this *Like) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
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
		re, err = this.Compile(s)
		if err != nil {
			return nil, err
		}
	}

	return value.NewValue(re.MatchString(f)), nil
}

func (this *Like) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLike(operands[0], operands[1])
	}
}

func (this *Like) Regexp() *regexp.Regexp { return this.re }

func (this *Like) Precompile() {
	switch s := this.Second().(type) {
	case *Constant:
		sv := s.Value()
		if sv.Type() != value.STRING {
			return
		}

		re, err := this.Compile(sv.Actual().(string))
		if err != nil {
			return
		}

		this.re = re
	}
}

func (this *Like) Compile(s string) (*regexp.Regexp, error) {
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
