//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package expression

import (
	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// FlattenKeys
//
///////////////////////////////////////////////////

const (
	IK_ASC = 1 << iota
	IK_DESC
	IK_MISSING
	IK_NONE = 0
)

type FlattenKeys struct {
	FunctionBase
	attributes []uint32
}

func NewFlattenKeys(operands ...Expression) Function {
	rv := &FlattenKeys{
		*NewFunctionBase("flatten_keys", operands...),
		make([]uint32, len(operands)),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *FlattenKeys) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *FlattenKeys) Type() value.Type { return value.ARRAY }

func (this *FlattenKeys) Evaluate(item value.Value, context Context) (value.Value, error) {
	var f []interface{}
	for _, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		}
		f = append(f, arg)
	}
	return value.NewValue(f), nil
}

func (this *FlattenKeys) PropagatesNull() bool {
	return false
}

/*
Minimum input arguments required is 1.
*/
func (this *FlattenKeys) MinArgs() int { return 1 }

/*
Maximum input arguments allowed.
*/
func (this *FlattenKeys) MaxArgs() int { return 32 }

/*
Factory method pattern.
*/
func (this *FlattenKeys) Constructor() FunctionConstructor {
	return NewFlattenKeys
}

func (this *FlattenKeys) Copy() Expression {
	rv := &FlattenKeys{
		*NewFunctionBase("flatten_keys", CopyExpressions(this.Operands())...),
		make([]uint32, len(this.attributes)),
	}
	copy(rv.attributes, this.attributes)
	rv.BaseCopy(this)
	rv.SetExpr(rv)
	return rv
}

func (this *FlattenKeys) SetAttributes(attrs []uint32) {
	this.attributes = attrs
}

func (this *FlattenKeys) Attributes() []uint32 {
	return this.attributes
}

func (this *FlattenKeys) GetAttribute(pos int) uint32 {
	if pos >= 0 && pos < len(this.attributes) {
		return this.attributes[pos]
	}
	return IK_NONE
}

func (this *FlattenKeys) HasAttribute(pos int, attr uint32) bool {
	return (this.GetAttribute(pos) & attr) != 0
}

func (this *FlattenKeys) HasDesc(pos int) bool {
	return (this.GetAttribute(pos) & IK_DESC) != 0
}
func (this *FlattenKeys) HasMissing(pos int) bool {
	return (this.GetAttribute(pos) & IK_MISSING) != 0
}

func (this *FlattenKeys) AttributeString(pos int) string {
	s := ""
	if this.HasMissing(pos) {
		s += " MISSING"
	}
	if this.HasDesc(pos) {
		s += " DESC"
	}
	return s
}
