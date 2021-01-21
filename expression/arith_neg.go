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
	"github.com/couchbase/query/value"
)

/*
Represents negation for arithmetic expressions. Type Neg is a struct
that implements UnaryFunctionBase.
*/
type Neg struct {
	UnaryFunctionBase
}

func NewNeg(operand Expression) Function {
	rv := &Neg{
		*NewUnaryFunctionBase("neg", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Neg) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitNeg(this)
}

func (this *Neg) Type() value.Type { return value.NUMBER }

/*
Return the neagation of the input value, if the type of input is a number.
For missing return a missing value, and for all other input types return a
null.
*/
func (this *Neg) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.NUMBER {
		return value.AsNumberValue(arg).Neg(), nil
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else {
		return value.NULL_VALUE, nil
	}
}

/*
Factory method pattern.
*/
func (this *Neg) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewNeg(operands[0])
	}
}
