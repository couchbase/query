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
	"bytes"
	"math"

	"github.com/couchbase/query/value"
)

/*
This represents the concatenation operation for strings.
*/
type Concat struct {
	FunctionBase
}

func NewConcat(operands ...Expression) Function {
	rv := &Concat{
		*NewFunctionBase("concat", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Concat) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitConcat(this)
}

func (this *Concat) Type() value.Type { return value.STRING }

func (this *Concat) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *Concat) Apply(context Context, args ...value.Value) (value.Value, error) {
	var buf bytes.Buffer
	null := false

	for _, arg := range args {
		switch arg.Type() {
		case value.STRING:
			if !null {
				buf.WriteString(arg.Actual().(string))
			}
		case value.MISSING:
			return value.MISSING_VALUE, nil
		default:
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(buf.String()), nil
}

/*
Minimum input arguments required for the concatenation
is 2.
*/
func (this *Concat) MinArgs() int { return 2 }

/*
Maximum number of input arguments defined for CONCAT is
MaxInt16 = 1<<15 - 1.
*/
func (this *Concat) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *Concat) Constructor() FunctionConstructor {
	return NewConcat
}

/*
This represents the concatenation operation for strings or array of strings with separator.
*/
type Concat2 struct {
	FunctionBase
}

func NewConcat2(operands ...Expression) Function {
	rv := &Concat2{
		*NewFunctionBase("concat2", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Concat2) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Concat2) Type() value.Type { return value.STRING }

func (this *Concat2) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *Concat2) Apply(context Context, args ...value.Value) (value.Value, error) {
	var buf bytes.Buffer
	var sp string
	null := false
	addSp := false

	for i, arg := range args {
		if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if (arg.Type() != value.ARRAY && arg.Type() != value.STRING) ||
			(i == 0 && arg.Type() == value.ARRAY) {
			null = true
		} else if arg.Type() == value.STRING {
			if i == 0 {
				sp = arg.Actual().(string)
			} else if !null {
				if addSp && sp != "" {
					buf.WriteString(sp)
				}
				buf.WriteString(arg.Actual().(string))
				addSp = true
			}
		} else if arg.Type() == value.ARRAY {
			for _, ae := range arg.Actual().([]interface{}) {
				ael := value.NewValue(ae)
				if ael.Type() == value.MISSING {
					return value.MISSING_VALUE, nil
				} else if ael.Type() != value.STRING {
					null = true
				} else if !null {
					if addSp && sp != "" {
						buf.WriteString(sp)
					}
					buf.WriteString(ael.Actual().(string))
					addSp = true
				}
			}
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(buf.String()), nil
}

/*
Minimum input arguments required for the concatenation
is 2.
*/
func (this *Concat2) MinArgs() int { return 2 }

/*
Maximum number of input arguments defined for CONCAT2 is
MaxInt16 = 1<<15 - 1.
*/
func (this *Concat2) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *Concat2) Constructor() FunctionConstructor {
	return NewConcat2
}
