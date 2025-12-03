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

/*
Nested expressions are used to access slices inside of arrays.
*/
type Slice struct {
	FunctionBase
	start bool
	end   bool
}

func NewSlice(operands ...Expression) Function {
	rv := &Slice{
		*NewFunctionBase("slice", operands...),
		false,
		false,
	}

	if len(operands) == 3 {
		rv.start = true
		rv.end = true
	} else if len(operands) == 2 {
		rv.start = true
	}

	rv.expr = rv
	return rv
}

func NewSliceEnd(operands ...Expression) Function {
	rv := NewSlice(operands...)
	s, _ := rv.(*Slice)
	s.start = false
	s.end = true
	return rv
}

func (this *Slice) Start() Expression {
	if !this.start {
		return nil
	}
	return this.operands[1]
}

func (this *Slice) End() Expression {
	if !this.end {
		return nil
	} else if this.start {
		return this.operands[2]
	}
	return this.operands[1]
}

/*
Visitor pattern.
*/
func (this *Slice) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSlice(this)
}

func (this *Slice) Type() value.Type { return value.ARRAY }

func (this *Slice) IsFromEmptyBrackets() bool {
	return !this.start && !this.end
}

func (this *Slice) Copy() Expression {
	rv := &Slice{
		*NewFunctionBase(this.FunctionBase.Name(), CopyExpressions(this.operands)...),
		this.start,
		this.end,
	}

	rv.BaseCopy(this)
	rv.expr = rv
	return rv
}

/*
This method Evaluates the slice using the input args depending on the
number of args. The form source-expr [ start : end ] is called array
slicing. It returns a new array containing a subset of the source,
containing the elements from position start to end-1. The element at
start is included, while the element at end is not. If end is omitted,
all elements from start to the end of the source array are included.
*/
func (this *Slice) Evaluate(item value.Value, context Context) (value.Value, error) {
	missing := false
	source, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if source.Type() == value.MISSING {
		missing = true
	}

	var start value.Value
	n := 1
	if this.start {
		start, err = this.operands[n].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if start.Type() == value.MISSING {
			missing = true
		}
		n++
	} else {
		start = value.ZERO_VALUE
	}

	ev := -1
	if this.end {
		end, err := this.operands[n].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if end.Type() == value.MISSING {
			missing = true
		} else if !missing {
			ea, ok := end.Actual().(float64)
			if !ok || ea != math.Trunc(ea) {
				return value.NULL_VALUE, nil
			}
			ev = int(ea)
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	}

	if source.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	sa, ok := start.Actual().(float64)
	if !ok || sa != math.Trunc(sa) {
		return value.NULL_VALUE, nil
	}

	if this.IsFromEmptyBrackets() {
		if se, ok := this.operands[0].(*Slice); ok && se.IsFromEmptyBrackets() {
			// [] of []; parent slice must contain only anonymous array elements
			for i := 0; ; i++ {
				v, ok := source.Index(i)
				if !ok {
					break
				}
				if v.Type() != value.ARRAY {
					return value.MISSING_VALUE, nil
				}
			}
		}
	}

	var rv value.Value
	if this.end {
		rv, _ = source.Slice(int(sa), ev)
	} else {
		rv, _ = source.SliceTail(int(sa))
	}

	return rv, nil
}

/*
Minimum input arguments required for Slices is 1.
*/
func (this *Slice) MinArgs() int { return 1 }

/*
Minimum input arguments allowed for Slices is 3.
*/
func (this *Slice) MaxArgs() int { return 3 }

/*
Factory method pattern.
*/
func (this *Slice) Constructor() FunctionConstructor {
	return NewSlice
}
