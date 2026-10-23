//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package rewrite

import (
	"github.com/couchbase/query/algebra"
)

func (this *Rewrite) visitJoin(left algebra.FromTerm, right algebra.SimpleFromTerm) (err error) {
	if _, err = left.Accept(this); err == nil {
		_, err = right.Accept(this)
	}

	return err
}

func (this *Rewrite) VisitJoin(node *algebra.Join) (r any, err error) {
	return node, this.visitJoin(node.Left(), node.Right())
}

func (this *Rewrite) VisitIndexJoin(node *algebra.IndexJoin) (any, error) {
	return node, this.visitJoin(node.Left(), node.Right())
}

func (this *Rewrite) VisitAnsiJoin(node *algebra.AnsiJoin) (r any, err error) {
	return node, this.visitJoin(node.Left(), node.Right())
}

func (this *Rewrite) VisitNest(node *algebra.Nest) (any, error) {
	return node, this.visitJoin(node.Left(), node.Right())
}

func (this *Rewrite) VisitIndexNest(node *algebra.IndexNest) (any, error) {
	return node, this.visitJoin(node.Left(), node.Right())
}

func (this *Rewrite) VisitAnsiNest(node *algebra.AnsiNest) (r any, err error) {
	return node, this.visitJoin(node.Left(), node.Right())
}

func (this *Rewrite) VisitUnnest(node *algebra.Unnest) (r any, err error) {
	if _, err = node.Left().Accept(this); err == nil {
		_, err = this.Map(node.Expression())
	}
	return node, err
}
