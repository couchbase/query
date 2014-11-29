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
// IfMissing
//
///////////////////////////////////////////////////

type IfMissing struct {
	FunctionBase
}

func NewIfMissing(operands ...Expression) Function {
	rv := &IfMissing{
		*NewFunctionBase("ifmissing", operands...),
	}

	rv.expr = rv
	return rv
}

func (this *IfMissing) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IfMissing) Type() value.Type { return value.JSON }

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

func (this *IfMissing) MinArgs() int { return 2 }

func (this *IfMissing) MaxArgs() int { return math.MaxInt16 }

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
	rv := &IfMissingOrNull{
		*NewFunctionBase("ifmissingornull", operands...),
	}

	rv.expr = rv
	return rv
}

func (this *IfMissingOrNull) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IfMissingOrNull) Type() value.Type { return value.JSON }

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

func (this *IfMissingOrNull) MinArgs() int { return 2 }

func (this *IfMissingOrNull) MaxArgs() int { return math.MaxInt16 }

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
	rv := &IfNull{
		*NewFunctionBase("ifnull", operands...),
	}

	rv.expr = rv
	return rv
}

func (this *IfNull) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IfNull) Type() value.Type { return value.JSON }

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

func (this *IfNull) MinArgs() int { return 2 }

func (this *IfNull) MaxArgs() int { return math.MaxInt16 }

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
	rv := &FirstVal{
		*NewFunctionBase("firstval", operands...),
	}

	rv.expr = rv
	return rv
}

func (this *FirstVal) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *FirstVal) Type() value.Type { return value.JSON }

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

func (this *FirstVal) MinArgs() int { return 1 }

func (this *FirstVal) MaxArgs() int { return math.MaxInt16 }

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
	rv := &MissingIf{
		*NewBinaryFunctionBase("missingif", first, second),
	}

	rv.expr = rv
	return rv
}

func (this *MissingIf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *MissingIf) Type() value.Type { return value.JSON }

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
	rv := &NullIf{
		*NewBinaryFunctionBase("nullif", first, second),
	}

	rv.expr = rv
	return rv
}

func (this *NullIf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NullIf) Type() value.Type { return value.JSON }

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
