//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package semantics

import (
	"github.com/couchbase/query/algebra"
)

func (this *SemChecker) visitSetop(first algebra.Subresult, second algebra.Subresult) (any, error) {
	f, err := first.Accept(this)
	if err != nil {
		return f, err
	}
	return second.Accept(this)
}

func (this *SemChecker) VisitUnion(node *algebra.Union) (any, error) {
	return this.visitSetop(node.First(), node.Second())
}

func (this *SemChecker) VisitUnionAll(node *algebra.UnionAll) (any, error) {
	return this.visitSetop(node.First(), node.Second())
}

func (this *SemChecker) VisitIntersect(node *algebra.Intersect) (any, error) {
	return this.visitSetop(node.First(), node.Second())
}

func (this *SemChecker) VisitIntersectAll(node *algebra.IntersectAll) (any, error) {
	return this.visitSetop(node.First(), node.Second())
}

func (this *SemChecker) VisitExcept(node *algebra.Except) (any, error) {
	return this.visitSetop(node.First(), node.Second())
}

func (this *SemChecker) VisitExceptAll(node *algebra.ExceptAll) (any, error) {
	return this.visitSetop(node.First(), node.Second())
}
