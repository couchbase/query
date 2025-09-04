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

	"github.com/couchbase/query/sort"
	"github.com/couchbase/query/value"
)

/*
Represents array construction.
*/
type ArrayConstruct struct {
	FunctionBase
}

func NewArrayConstruct(operands ...Expression) Function {
	rv := &ArrayConstruct{}
	rv.Init("array", operands...)

	rv.expr = rv
	rv.Value() // Initialize value
	return rv
}

/*
Visitor pattern.
*/
func (this *ArrayConstruct) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitArrayConstruct(this)
}

func (this *ArrayConstruct) Type() value.Type { return value.ARRAY }

func (this *ArrayConstruct) Evaluate(item value.Value, context Context) (value.Value, error) {
	if this.value != nil && *this.value != nil {
		return *this.value, nil
	}

	if len(this.operands) == 1 && this.HasExprFlag(EXPR_CAN_FLATTEN) {
		return this.operands[0].Evaluate(item, context)
	} else {
		aa := make([]interface{}, len(this.operands))
		for i, _ := range this.operands {
			arg, err := this.operands[i].Evaluate(item, context)
			if err != nil {
				return nil, err
			}
			aa[i] = arg
		}
		return value.NewTrackedValue(aa), nil
	}
}

func (this *ArrayConstruct) EquivalentTo(other Expression) bool {
	if this.HasExprFlag(EXPR_ARRAY_IS_SET) {
		thisVal := this.Value()
		otherVal := other.Value()
		if thisVal != nil && otherVal != nil {
			thisVals, ok1 := thisVal.Actual().([]interface{})
			otherVals, ok2 := otherVal.Actual().([]interface{})
			if ok1 && ok2 {
				thisVals = SortValArr(thisVals)
				otherVals = SortValArr(otherVals)
				return value.NewValue(thisVals).EquivalentTo(value.NewValue(otherVals))
			}
		}
	}
	return this.FunctionBase.EquivalentTo(other)
}

func (this *ArrayConstruct) PropagatesMissing() bool {
	return this.value != nil && *this.value != nil
}

func (this *ArrayConstruct) PropagatesNull() bool {
	return this.value != nil && *this.value != nil
}

func (this *ArrayConstruct) ResetValue() {
	this.ExprBase().ResetValue()
	this.Value() // need to initialize value
}

/*
Minimum input arguments required for the defined ArrayConstruct
function. It is 0.
*/
func (this *ArrayConstruct) MinArgs() int { return 0 }

/*
Maximum number of input arguments defined for the ArrayConstruct
function is MaxInt16  = 1<<15 - 1.
*/
func (this *ArrayConstruct) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *ArrayConstruct) Constructor() FunctionConstructor {
	return NewArrayConstruct
}

func SortValArr(vals []interface{}) []interface{} {
	set := value.NewSet(len(vals), true, false)
	set.AddAll(vals)
	vals = set.Actuals()

	sort.Sort(value.NewSorter(value.NewValue(vals)))
	return vals
}
