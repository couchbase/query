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
	"github.com/couchbaselabs/query/value"
)

///////////////////////////////////////////////////
//
// IfMissing
//
///////////////////////////////////////////////////

type IfMissing struct {
	FunctionBase
}

func NewIfMissing(operands ...Expression) Function {
	return &IfMissing{
		*NewFunctionBase("ifmissing", operands...),
	}
}

func (this *IfMissing) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IfMissing) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *IfMissing) Apply(context Context, args ...value.Value) (value.Value, error) {
	for _, a := range args {
		if a.Type() != value.MISSING {
			return a, nil
		}
	}

	return value.NULL_VALUE, nil
}

func (this *IfMissing) Constructor() FunctionConstructor { return NewIfMissing }

///////////////////////////////////////////////////
//
// IfMissingOrNull
//
///////////////////////////////////////////////////

type IfMissingOrNull struct {
	FunctionBase
}

func NewIfMissingOrNull(operands ...Expression) Function {
	return &IfMissingOrNull{
		*NewFunctionBase("ifmissingornull", operands...),
	}
}

func (this *IfMissingOrNull) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IfMissingOrNull) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *IfMissingOrNull) Apply(context Context, args ...value.Value) (value.Value, error) {
	for _, a := range args {
		if a.Type() > value.NULL {
			return a, nil
		}
	}

	return value.NULL_VALUE, nil
}

func (this *IfMissingOrNull) Constructor() FunctionConstructor { return NewIfMissingOrNull }

///////////////////////////////////////////////////
//
// IfNull
//
///////////////////////////////////////////////////

type IfNull struct {
	FunctionBase
}

func NewIfNull(operands ...Expression) Function {
	return &IfNull{
		*NewFunctionBase("ifnull", operands...),
	}
}

func (this *IfNull) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IfNull) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *IfNull) Apply(context Context, args ...value.Value) (value.Value, error) {
	for _, a := range args {
		if a.Type() != value.NULL {
			return a, nil
		}
	}

	return value.NULL_VALUE, nil
}

func (this *IfNull) Constructor() FunctionConstructor { return NewIfNull }

///////////////////////////////////////////////////
//
// FirstVal
//
///////////////////////////////////////////////////

type FirstVal struct {
	FunctionBase
}

func NewFirstVal(operands ...Expression) Function {
	return &FirstVal{
		*NewFunctionBase("firstval", operands...),
	}
}

func (this *FirstVal) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *FirstVal) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *FirstVal) Apply(context Context, args ...value.Value) (value.Value, error) {
	for _, a := range args {
		if a.Type() > value.NULL {
			return a, nil
		}
	}

	return value.NULL_VALUE, nil
}

func (this *FirstVal) Constructor() FunctionConstructor { return NewFirstVal }

///////////////////////////////////////////////////
//
// MissingIf
//
///////////////////////////////////////////////////

type MissingIf struct {
	BinaryFunctionBase
}

func NewMissingIf(first, second Expression) Function {
	return &MissingIf{
		*NewBinaryFunctionBase("missingif", first, second),
	}
}

func (this *MissingIf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *MissingIf) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *MissingIf) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Equals(second) {
		return value.MISSING_VALUE, nil
	} else {
		return first, nil
	}
}

func (this *MissingIf) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewMissingIf(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// NullIf
//
///////////////////////////////////////////////////

type NullIf struct {
	BinaryFunctionBase
}

func NewNullIf(first, second Expression) Function {
	return &NullIf{
		*NewBinaryFunctionBase("nullif", first, second),
	}
}

func (this *NullIf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NullIf) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *NullIf) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Equals(second) {
		return value.NULL_VALUE, nil
	} else {
		return first, nil
	}
}

func (this *NullIf) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewNullIf(operands[0], operands[1])
	}
}
