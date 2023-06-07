//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

// mark AST nodes as correlated when appropriate
func markCorrelated(node Node) error {
	markCorr := newMarkCorrelation()
	_, err := node.Accept(markCorr)
	return err
}

type markCorrelation struct {
}

func newMarkCorrelation() *markCorrelation {
	return &markCorrelation{}
}

func (this *markCorrelation) visitJoin(left FromTerm, right SimpleFromTerm) error {
	_, err := left.Accept(this)
	if err == nil {
		_, err = right.Accept(this)
	}

	return err
}

func (this *markCorrelation) visitSetop(first Subresult, second Subresult) error {
	_, err := first.Accept(this)
	if err == nil {
		_, err = second.Accept(this)
	}

	return err
}

func (this *markCorrelation) VisitSelectTerm(node *SelectTerm) (interface{}, error) {
	return node.query.subresult.Accept(this)
}

func (this *markCorrelation) VisitSubselect(node *Subselect) (interface{}, error) {
	if node.From() != nil {
		return node.From().Accept(this)
	}
	return nil, nil
}

func (this *markCorrelation) VisitKeyspaceTerm(node *KeyspaceTerm) (interface{}, error) {
	if node.keys != nil || node.joinKeys != nil {
		node.correlated = true
	}
	return nil, nil
}

func (this *markCorrelation) VisitExpressionTerm(node *ExpressionTerm) (interface{}, error) {
	if node.IsKeyspace() {
		return node.KeyspaceTerm().Accept(this)
	}
	node.correlated = true
	return nil, nil
}

func (this *markCorrelation) VisitSubqueryTerm(node *SubqueryTerm) (interface{}, error) {
	// rely on the CORRELATED marking on the subquery itself
	return nil, nil
}

func (this *markCorrelation) VisitJoin(node *Join) (interface{}, error) {
	return nil, this.visitJoin(node.Left(), node.Right())
}

func (this *markCorrelation) VisitIndexJoin(node *IndexJoin) (interface{}, error) {
	return nil, this.visitJoin(node.Left(), node.Right())
}

func (this *markCorrelation) VisitAnsiJoin(node *AnsiJoin) (interface{}, error) {
	return nil, this.visitJoin(node.Left(), node.Right())
}

func (this *markCorrelation) VisitNest(node *Nest) (interface{}, error) {
	return nil, this.visitJoin(node.Left(), node.Right())
}

func (this *markCorrelation) VisitIndexNest(node *IndexNest) (interface{}, error) {
	return nil, this.visitJoin(node.Left(), node.Right())
}

func (this *markCorrelation) VisitAnsiNest(node *AnsiNest) (interface{}, error) {
	return nil, this.visitJoin(node.Left(), node.Right())
}

func (this *markCorrelation) VisitUnnest(node *Unnest) (interface{}, error) {
	return node.Left().Accept(this)
}

func (this *markCorrelation) VisitUnion(node *Union) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}

func (this *markCorrelation) VisitUnionAll(node *UnionAll) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}

func (this *markCorrelation) VisitIntersect(node *Intersect) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}

func (this *markCorrelation) VisitIntersectAll(node *IntersectAll) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}

func (this *markCorrelation) VisitExcept(node *Except) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}

func (this *markCorrelation) VisitExceptAll(node *ExceptAll) (interface{}, error) {
	return nil, this.visitSetop(node.First(), node.Second())
}
