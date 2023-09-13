//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"math"

	"github.com/couchbase/query/value"
)

const (
	_BITONE   = 0x01
	_BITSTART = 1
	_BITEND   = 64
)

///////////////////////////////////////////////////
//
// BITAND
//
///////////////////////////////////////////////////

/*
This represents the bit manipulation function BITAND(num1,num2...).
It returns result of the bitwise AND on all input arguments.
*/

type BitAnd struct {
	FunctionBase
}

func NewBitAnd(operands ...Expression) Function {
	rv := &BitAnd{}
	rv.Init("bitand", operands...)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *BitAnd) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *BitAnd) Type() value.Type { return value.NUMBER }

func (this *BitAnd) Evaluate(item value.Value, context Context) (value.Value, error) {
	var result int64
	missing := false
	null := false
	for i, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			missing = true
		} else if arg.Type() != value.NUMBER {
			null = true
		} else if !missing && !null {
			var val int64
			var ok bool
			if val, ok = value.IsIntValue(arg); !ok {
				null = true
			} else if i == 0 {
				result = val
			} else {
				result = result & val
			}
		}
	}
	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(result), nil
}

/*
Minimum input arguments required is 2.
*/
func (this *BitAnd) MinArgs() int { return 2 }

/*
Maximum input arguments allowed.
*/
func (this *BitAnd) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *BitAnd) Constructor() FunctionConstructor {
	return NewBitAnd
}

///////////////////////////////////////////////////
//
// BITOR
//
///////////////////////////////////////////////////

/*
This represents the bit manipulation function BITOR(num1,num2...).
It returns result of the bitwise OR on all input arguments.
*/

type BitOr struct {
	FunctionBase
}

func NewBitOr(operands ...Expression) Function {
	rv := &BitOr{}
	rv.Init("bitor", operands...)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *BitOr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *BitOr) Type() value.Type { return value.NUMBER }

func (this *BitOr) Evaluate(item value.Value, context Context) (value.Value, error) {
	var result int64
	null := false
	missing := false
	for i, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			missing = true
		} else if arg.Type() != value.NUMBER {
			null = true
		} else if !missing && !null {
			var val int64
			var ok bool
			if val, ok = value.IsIntValue(arg); !ok {
				null = true
			} else if i == 0 {
				result = val
			} else {
				result = result | val
			}
		}
	}
	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(result), nil
}

/*
Minimum input arguments required is 2.
*/
func (this *BitOr) MinArgs() int { return 2 }

/*
Maximum input arguments allowed.
*/
func (this *BitOr) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *BitOr) Constructor() FunctionConstructor {
	return NewBitOr
}

///////////////////////////////////////////////////
//
// BITXOR
//
///////////////////////////////////////////////////

/*
This represents the bit manipulation function BITXOR(num1,num2...).
It returns result of the bitwise XOR on all input arguments.
*/

type BitXor struct {
	FunctionBase
}

func NewBitXor(operands ...Expression) Function {
	rv := &BitXor{}
	rv.Init("bitxor", operands...)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *BitXor) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *BitXor) Type() value.Type { return value.NUMBER }

func (this *BitXor) Evaluate(item value.Value, context Context) (value.Value, error) {
	var result int64
	null := false
	missing := false
	for i, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			missing = true
		} else if arg.Type() != value.NUMBER {
			null = true
		} else if !missing && !null {
			var val int64
			var ok bool
			if val, ok = value.IsIntValue(arg); !ok {
				null = true
			} else if i == 0 {
				result = val
			} else {
				result = result ^ val
			}
		}
	}
	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(result), nil
}

/*
Minimum input arguments required is 2.
*/
func (this *BitXor) MinArgs() int { return 2 }

/*
Maximum input arguments allowed.
*/
func (this *BitXor) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *BitXor) Constructor() FunctionConstructor {
	return NewBitXor
}

///////////////////////////////////////////////////
//
// BITNOT
//
///////////////////////////////////////////////////

/*
This represents the bit manipulation function BITNOT(num1).
It returns result of the bitwise NOT on all input arguments.
*/

type BitNot struct {
	UnaryFunctionBase
}

func NewBitNot(operand Expression) Function {
	rv := &BitNot{}
	rv.Init("bitnot", operand)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *BitNot) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *BitNot) Type() value.Type { return value.NUMBER }

/*
This method reverses the input array value and returns it.
If the input value is of type missing return a missing
value, and for all non array values return null.
*/
func (this *BitNot) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	// If not a numeric value return NULL.
	if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	var result int64
	var ok bool

	if result, ok = value.IsIntValue(arg); !ok {
		return value.NULL_VALUE, nil
	}

	result = ^result
	return value.NewValue(result), nil
}

/*
Factory method pattern.
*/
func (this *BitNot) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewBitNot(operands[0])
	}
}

///////////////////////////////////////////////////
//
// BITSHIFT
//
///////////////////////////////////////////////////

/*
This represents the bit manipulation function BITSHIFT(num1,shift amt,is_rotate).
It returns result of the bitwise left or right shift on input argument. If is_rotate
is true then it performs a circular shift. Otherwise it performs a logical shift.
*/

type BitShift struct {
	FunctionBase
}

func NewBitShift(operands ...Expression) Function {
	rv := &BitShift{}
	rv.Init("bitshift", operands...)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *BitShift) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *BitShift) Type() value.Type { return value.NUMBER }

func (this *BitShift) Evaluate(item value.Value, context Context) (value.Value, error) {
	if len(this.operands) < 2 {
		return value.MISSING_VALUE, nil
	}
	var num1, shift int64
	var ok bool
	isRotate := false
	null := false
	missing := false
	for k, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			missing = true
		} else if arg.Type() == value.NULL {
			null = true
		} else if !missing && !null && k < 3 {
			if k == 0 {
				if arg.Type() != value.NUMBER {
					null = true
				} else if num1, ok = value.IsIntValue(arg); !ok {
					null = true
				}
			} else if k == 1 {
				if arg.Type() != value.NUMBER {
					null = true
				} else if shift, ok = value.IsIntValue(arg); !ok {
					null = true
				}
			} else if k == 2 {
				if arg.Type() != value.BOOLEAN {
					null = true
				} else {
					isRotate = arg.Actual().(bool)
				}
			}
		} else {
			// shouldn't ever exist, but if it does it must be a number
			if arg.Type() != value.NUMBER {
				null = true
			}
		}
	}
	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	var result uint64
	// Check if it is rotate and shift
	if isRotate == true {
		result = rotateLeft(uint64(num1), int(shift))
	} else {
		result = shiftLeft(uint64(num1), int(shift))
	}

	return value.NewValue(int64(result)), nil

}

/*
Minimum input arguments required is 2.
*/
func (this *BitShift) MinArgs() int { return 2 }

/*
Maximum input arguments allowed.
*/
func (this *BitShift) MaxArgs() int { return 3 }

/*
Factory method pattern.
*/
func (this *BitShift) Constructor() FunctionConstructor {
	return NewBitShift
}

///////////////////////////////////////////////////
//
// BITSET
//
///////////////////////////////////////////////////

/*
This represents the bit manipulation function BITSET(num1,[list of positions]]).
It returns the value after setting the bits at the input positions.
*/

type BitSet struct {
	BinaryFunctionBase
}

func NewBitSet(first, second Expression) Function {
	rv := &BitSet{}
	rv.Init("bitset", first, second)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *BitSet) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *BitSet) Type() value.Type { return value.NUMBER }

func (this *BitSet) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	return bitSetNClear(true, context, first, second)
}

func (this *BitSet) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewBitSet(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// BITCLEAR
//
///////////////////////////////////////////////////

/*
This represents the bit manipulation function BITCLEAR(num1,[list of positions]]).
It returns the value after clearing the bits at the input positions.
*/

type BitClear struct {
	BinaryFunctionBase
}

func NewBitClear(first, second Expression) Function {
	rv := &BitClear{}
	rv.Init("bitclear", first, second)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *BitClear) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *BitClear) Type() value.Type { return value.NUMBER }

func (this *BitClear) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	return bitSetNClear(false, context, first, second)
}

func (this *BitClear) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewBitClear(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// BITTEST OR ISBITSET
//
///////////////////////////////////////////////////

/*
This represents the bit manipulation function BitTest(num1, <list of bit positions>,<all set>).
It returns true if any or all the bits in positions are set.
*/

type BitTest struct {
	FunctionBase
}

func NewBitTest(operands ...Expression) Function {
	rv := &BitTest{}
	rv.Init("bittest", operands...)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *BitTest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *BitTest) Type() value.Type { return value.BOOLEAN }

func (this *BitTest) Evaluate(item value.Value, context Context) (value.Value, error) {
	var num1 int64
	var bitP uint64
	var ok bool

	if len(this.operands) < 2 {
		return value.MISSING_VALUE, nil
	}
	isAll := false
	null := false
	missing := false
	for k, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			missing = true
		} else if arg.Type() == value.NULL {
			null = true
		} else if !missing && !null && k < 3 {
			if k == 0 {
				if arg.Type() != value.NUMBER {
					null = true
				} else if num1, ok = value.IsIntValue(arg); !ok {
					null = true
				}
			} else if k == 1 {
				// For 2nd arg - num or array ok
				if arg.Type() != value.NUMBER && arg.Type() != value.ARRAY {
					null = true
				} else {
					bitP, ok = bitPositions(arg)
					if !ok {
						null = true
					}
				}
			} else if k == 2 {
				if arg.Type() != value.BOOLEAN {
					null = true
				} else {
					isAll = arg.Actual().(bool)
				}
			}
		} else {
			// shouldn't ever exist, but if it does it must be a number
			if arg.Type() != value.NUMBER {
				null = true
			}
		}
	}
	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	if isAll {
		return value.NewValue((uint64(num1) & bitP) == bitP), nil
	}
	return value.NewValue((uint64(num1) & bitP) != 0), nil
}

/*
Minimum input arguments required is 2.
*/
func (this *BitTest) MinArgs() int { return 2 }

/*
Maximum input arguments allowed.
*/
func (this *BitTest) MaxArgs() int { return 3 }

/*
Factory method pattern.
*/
func (this *BitTest) Constructor() FunctionConstructor {
	return NewBitTest
}

// Function to set a bit or clear a bit in the input number
func bitSetNClear(bitset bool, context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	} else if second.Type() != value.NUMBER && second.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	var num1 int64
	var bitP, result uint64
	var ok bool

	if num1, ok = value.IsIntValue(first); !ok {
		return value.NULL_VALUE, nil
	}

	bitP, ok = bitPositions(second)

	if !ok {
		return value.NULL_VALUE, nil
	} else {
		if bitset {
			result = uint64(num1) | bitP
		} else {
			result = uint64(num1) & ^bitP
		}
	}
	return value.NewValue(result), nil
}

func bitPositions(arg value.Value) (uint64, bool) {

	var pp, ppos int64
	var ok bool

	var pos []interface{}
	num1 := uint64(0)

	if arg.Type() == value.NUMBER {
		if pp, ok = value.IsIntValue(arg); !ok {
			return num1, false
		}
		pos = []interface{}{pp}
	} else {
		pos = arg.Actual().([]interface{})
	}

	// now that the array or positions has been populated.
	// range through

	for _, p := range pos {
		if ppos, ok = value.IsIntValue(value.NewValue(p)); !ok || ppos < _BITSTART || ppos > _BITEND {
			return num1, false
		}
		num1 = num1 | _BITONE<<uint64(ppos-_BITSTART)
	}
	return num1, true
}

// RotateLeft returns the value of x rotated left by (k mod 64) bits.
// To rotate x right by k bits, call RotateLeft64(x, -k).

// shift count type int64, must be unsigned integer

func rotateLeft(x uint64, k int) uint64 {
	const n = 64
	s := uint(k) & (n - 1)
	return x<<s | x>>(n-s)
}

// ShiftLeft returns the value of x shift left by input bits.
// To shift x right by k bits, call ShiftLeft(x, -k).
func shiftLeft(x uint64, k int) uint64 {
	if k < 0 {
		return x >> uint64(-k)
	}
	return x << uint64(k)
}
