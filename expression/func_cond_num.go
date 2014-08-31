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

	"github.com/couchbaselabs/query/value"
)

///////////////////////////////////////////////////
//
// IfInf
//
///////////////////////////////////////////////////

type IfInf struct {
	FunctionBase
}

func NewIfInf(operands ...Expression) Function {
	return &IfInf{
		*NewFunctionBase("ifinf", operands...),
	}
}

func (this *IfInf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IfInf) Type() value.Type { return value.NUMBER }

func (this *IfInf) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *IfInf) Apply(context Context, args ...value.Value) (value.Value, error) {
	for _, a := range args {
		if a.Type() == value.MISSING {
			continue
		} else if a.Type() != value.NUMBER {
			return value.NULL_VALUE, nil
		}

		f := a.Actual().(float64)
		if !math.IsInf(f, 0) {
			return value.NewValue(f), nil
		}
	}

	return value.NULL_VALUE, nil
}

func (this *IfInf) Constructor() FunctionConstructor { return NewIfInf }

///////////////////////////////////////////////////
//
// IfNaN
//
///////////////////////////////////////////////////

type IfNaN struct {
	FunctionBase
}

func NewIfNaN(operands ...Expression) Function {
	return &IfNaN{
		*NewFunctionBase("ifnan", operands...),
	}
}

func (this *IfNaN) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IfNaN) Type() value.Type { return value.NUMBER }

func (this *IfNaN) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *IfNaN) Apply(context Context, args ...value.Value) (value.Value, error) {
	for _, a := range args {
		if a.Type() == value.MISSING {
			continue
		} else if a.Type() != value.NUMBER {
			return value.NULL_VALUE, nil
		}

		f := a.Actual().(float64)
		if !math.IsNaN(f) {
			return value.NewValue(f), nil
		}
	}

	return value.NULL_VALUE, nil
}

func (this *IfNaN) Constructor() FunctionConstructor { return NewIfNaN }

///////////////////////////////////////////////////
//
// IfNaNOrInf
//
///////////////////////////////////////////////////

type IfNaNOrInf struct {
	FunctionBase
}

func NewIfNaNOrInf(operands ...Expression) Function {
	return &IfNaNOrInf{
		*NewFunctionBase("ifnanorinf", operands...),
	}
}

func (this *IfNaNOrInf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IfNaNOrInf) Type() value.Type { return value.NUMBER }

func (this *IfNaNOrInf) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *IfNaNOrInf) Apply(context Context, args ...value.Value) (value.Value, error) {
	for _, a := range args {
		if a.Type() == value.MISSING {
			continue
		} else if a.Type() != value.NUMBER {
			return value.NULL_VALUE, nil
		}

		f := a.Actual().(float64)
		if !math.IsInf(f, 0) && !math.IsNaN(f) {
			return value.NewValue(f), nil
		}
	}

	return value.NULL_VALUE, nil
}

func (this *IfNaNOrInf) Constructor() FunctionConstructor { return NewIfNaNOrInf }

///////////////////////////////////////////////////
//
// IfNegInf
//
///////////////////////////////////////////////////

type IfNegInf struct {
	FunctionBase
}

func NewIfNegInf(operands ...Expression) Function {
	return &IfNegInf{
		*NewFunctionBase("ifneginf", operands...),
	}
}

func (this *IfNegInf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IfNegInf) Type() value.Type { return value.NUMBER }

func (this *IfNegInf) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *IfNegInf) Apply(context Context, args ...value.Value) (value.Value, error) {
	for _, a := range args {
		if a.Type() == value.MISSING {
			continue
		} else if a.Type() != value.NUMBER {
			return value.NULL_VALUE, nil
		}

		f := a.Actual().(float64)
		if !math.IsInf(f, -1) {
			return value.NewValue(f), nil
		}
	}

	return value.NULL_VALUE, nil
}

func (this *IfNegInf) Constructor() FunctionConstructor { return NewIfNegInf }

///////////////////////////////////////////////////
//
// IfPosInf
//
///////////////////////////////////////////////////

type IfPosInf struct {
	FunctionBase
}

func NewIfPosInf(operands ...Expression) Function {
	return &IfPosInf{
		*NewFunctionBase("ifposinf", operands...),
	}
}

func (this *IfPosInf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IfPosInf) Type() value.Type { return value.NUMBER }

func (this *IfPosInf) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *IfPosInf) Apply(context Context, args ...value.Value) (value.Value, error) {
	for _, a := range args {
		if a.Type() == value.MISSING {
			continue
		} else if a.Type() != value.NUMBER {
			return value.NULL_VALUE, nil
		}

		f := a.Actual().(float64)
		if !math.IsInf(f, 1) {
			return value.NewValue(f), nil
		}
	}

	return value.NULL_VALUE, nil
}

func (this *IfPosInf) Constructor() FunctionConstructor { return NewIfPosInf }

///////////////////////////////////////////////////
//
// FirstNum
//
///////////////////////////////////////////////////

type FirstNum struct {
	FunctionBase
}

func NewFirstNum(operands ...Expression) Function {
	return &FirstNum{
		*NewFunctionBase("firstnum", operands...),
	}
}

func (this *FirstNum) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *FirstNum) Type() value.Type { return value.NUMBER }

func (this *FirstNum) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *FirstNum) Apply(context Context, args ...value.Value) (value.Value, error) {
	for _, a := range args {
		if a.Type() <= value.NULL {
			continue
		} else if a.Type() != value.NUMBER {
			return value.NULL_VALUE, nil
		} else {
			f := a.Actual().(float64)
			if !math.IsNaN(f) && !math.IsInf(f, 0) {
				return value.NewValue(f), nil
			}
		}
	}

	return value.NULL_VALUE, nil
}

func (this *FirstNum) Constructor() FunctionConstructor { return NewFirstNum }

///////////////////////////////////////////////////
//
// NaNIf
//
///////////////////////////////////////////////////

type NaNIf struct {
	BinaryFunctionBase
}

func NewNaNIf(first, second Expression) Function {
	return &NaNIf{
		*NewBinaryFunctionBase("nanif", first, second),
	}
}

func (this *NaNIf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NaNIf) Type() value.Type { return value.JSON }

func (this *NaNIf) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *NaNIf) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Equals(second) {
		return value.NewValue(math.NaN()), nil
	} else {
		return first, nil
	}
}

func (this *NaNIf) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewNaNIf(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// NegInfIf
//
///////////////////////////////////////////////////

type NegInfIf struct {
	BinaryFunctionBase
}

func NewNegInfIf(first, second Expression) Function {
	return &NegInfIf{
		*NewBinaryFunctionBase("neginfif", first, second),
	}
}

func (this *NegInfIf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NegInfIf) Type() value.Type { return value.JSON }

func (this *NegInfIf) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *NegInfIf) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Equals(second) {
		return value.NewValue(math.Inf(-1)), nil
	} else {
		return first, nil
	}
}

func (this *NegInfIf) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewNegInfIf(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// PosInfIf
//
///////////////////////////////////////////////////

type PosInfIf struct {
	BinaryFunctionBase
}

func NewPosInfIf(first, second Expression) Function {
	return &PosInfIf{
		*NewBinaryFunctionBase("posinfif", first, second),
	}
}

func (this *PosInfIf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *PosInfIf) Type() value.Type { return value.JSON }

func (this *PosInfIf) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *PosInfIf) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Equals(second) {
		return value.NewValue(math.Inf(1)), nil
	} else {
		return first, nil
	}
}

func (this *PosInfIf) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewPosInfIf(operands[0], operands[1])
	}
}
