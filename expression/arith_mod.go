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

	"github.com/couchbase/query/value"
)

/*
Represents Mod for arithmetic expressions. Type Mod is a struct
that implements BinaryFunctionBase.
*/
type Mod struct {
	BinaryFunctionBase
}

func NewMod(first, second Expression) Function {
	rv := &Mod{
		*NewBinaryFunctionBase("mod", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Mod) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMod(this)
}

func (this *Mod) Type() value.Type { return value.NUMBER }

func (this *Mod) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	if second.Type() == value.NUMBER {
		s := second.Actual().(float64)
		if s == 0.0 {
			return value.NULL_VALUE, nil
		}

		if first.Type() == value.NUMBER {
			m := math.Mod(first.Actual().(float64), s)
			return value.NewValue(m), nil
		}
	}

	return value.NULL_VALUE, nil
}

/*
Factory method pattern.
*/
func (this *Mod) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewMod(operands[0], operands[1])
	}
}
