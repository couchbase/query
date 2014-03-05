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

type IfInf struct {
	nAryBase
}

func NewIfInf(args Expressions) Function {
	return &IfInf{
		nAryBase{
			operands: args,
		},
	}
}

func (this *IfInf) evaluate(args value.Values) (value.Value, error) {
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

func (this *IfInf) MinArgs() int { return 2 }

func (this *IfInf) Constructor() FunctionConstructor { return NewIfInf }

type IfNaN struct {
	nAryBase
}

func NewIfNaN(args Expressions) Function {
	return &IfNaN{
		nAryBase{
			operands: args,
		},
	}
}

func (this *IfNaN) evaluate(args value.Values) (value.Value, error) {
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

func (this *IfNaN) MinArgs() int { return 2 }

func (this *IfNaN) Constructor() FunctionConstructor { return NewIfNaN }

type IfNaNOrInf struct {
	nAryBase
}

func NewIfNaNOrInf(args Expressions) Function {
	return &IfNaNOrInf{
		nAryBase{
			operands: args,
		},
	}
}

func (this *IfNaNOrInf) evaluate(args value.Values) (value.Value, error) {
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

func (this *IfNaNOrInf) MinArgs() int { return 2 }

func (this *IfNaNOrInf) Constructor() FunctionConstructor { return NewIfNaNOrInf }

type IfNegInf struct {
	nAryBase
}

func NewIfNegInf(args Expressions) Function {
	return &IfNegInf{
		nAryBase{
			operands: args,
		},
	}
}

func (this *IfNegInf) evaluate(args value.Values) (value.Value, error) {
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

func (this *IfNegInf) MinArgs() int { return 2 }

func (this *IfNegInf) Constructor() FunctionConstructor { return NewIfNegInf }

type IfPosInf struct {
	nAryBase
}

func NewIfPosInf(args Expressions) Function {
	return &IfPosInf{
		nAryBase{
			operands: args,
		},
	}
}

func (this *IfPosInf) evaluate(args value.Values) (value.Value, error) {
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

func (this *IfPosInf) MinArgs() int { return 2 }

func (this *IfPosInf) Constructor() FunctionConstructor { return NewIfPosInf }

type FirstNum struct {
	nAryBase
}

func NewFirstNum(args Expressions) Function {
	return &FirstNum{
		nAryBase{
			operands: args,
		},
	}
}

func (this *FirstNum) evaluate(args value.Values) (value.Value, error) {
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

func (this *FirstNum) MinArgs() int { return 2 }

func (this *FirstNum) Constructor() FunctionConstructor { return NewFirstNum }

type NaNIf struct {
	binaryBase
}

func NewNaNIf(first, second Expression) Function {
	return &NaNIf{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *NaNIf) evaluate(first, second value.Value) (value.Value, error) {
	if first.Equals(second) {
		return value.NewValue(math.NaN()), nil
	} else {
		return first, nil
	}
}

func (this *NaNIf) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewNaNIf(args[0], args[1])
	}
}

type NegInfIf struct {
	binaryBase
}

func NewNegInfIf(first, second Expression) Function {
	return &NegInfIf{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *NegInfIf) evaluate(first, second value.Value) (value.Value, error) {
	if first.Equals(second) {
		return value.NewValue(math.Inf(-1)), nil
	} else {
		return first, nil
	}
}

func (this *NegInfIf) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewNegInfIf(args[0], args[1])
	}
}

type PosInfIf struct {
	binaryBase
}

func NewPosInfIf(first, second Expression) Function {
	return &PosInfIf{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *PosInfIf) evaluate(first, second value.Value) (value.Value, error) {
	if first.Equals(second) {
		return value.NewValue(math.Inf(1)), nil
	} else {
		return first, nil
	}
}

func (this *PosInfIf) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewPosInfIf(args[0], args[1])
	}
}
