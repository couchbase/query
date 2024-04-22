//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

// assign arrayId
func AssignArrayId(expr Expression, arrayId int) (int, error) {
	if expr == nil {
		return arrayId, nil
	}

	assign := newAssignArrayId(arrayId)
	_, err := expr.Accept(assign)
	if err != nil {
		return arrayId, err
	}
	return assign.arrayId, nil
}

type assignArrayId struct {
	TraverserBase

	arrayId int
}

func newAssignArrayId(arrayId int) *assignArrayId {
	rv := &assignArrayId{
		arrayId: arrayId,
	}

	rv.traverser = rv
	return rv
}

func (this *assignArrayId) VisitAny(pred *Any) (interface{}, error) {
	return nil, this.visitCollPredBase(&pred.collPredBase)
}

func (this *assignArrayId) VisitAnyEvery(pred *AnyEvery) (interface{}, error) {
	return nil, this.visitCollPredBase(&pred.collPredBase)
}

func (this *assignArrayId) VisitEvery(pred *Every) (interface{}, error) {
	return nil, this.visitCollPredBase(&pred.collPredBase)
}

func (this *assignArrayId) visitCollPredBase(predBase *collPredBase) (err error) {
	this.arrayId++
	predBase.arrayId = this.arrayId
	this.arrayId, err = AssignArrayId(predBase.satisfies, this.arrayId)
	return
}
