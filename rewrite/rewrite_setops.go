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

func (this *Rewrite) VisitUnion(node *algebra.Union) (r interface{}, err error) {
	if _, err = node.First().Accept(this); err == nil {
		_, err = node.Second().Accept(this)
	}
	return node, err
}

func (this *Rewrite) VisitUnionAll(node *algebra.UnionAll) (r interface{}, err error) {
	if _, err = node.First().Accept(this); err == nil {
		_, err = node.Second().Accept(this)
	}
	return node, err
}

func (this *Rewrite) VisitIntersect(node *algebra.Intersect) (r interface{}, err error) {
	if _, err = node.First().Accept(this); err == nil {
		_, err = node.Second().Accept(this)
	}
	return node, err
}

func (this *Rewrite) VisitIntersectAll(node *algebra.IntersectAll) (r interface{}, err error) {
	if _, err = node.First().Accept(this); err == nil {
		_, err = node.Second().Accept(this)
	}
	return node, err
}

func (this *Rewrite) VisitExcept(node *algebra.Except) (r interface{}, err error) {
	if _, err = node.First().Accept(this); err == nil {
		_, err = node.Second().Accept(this)
	}
	return node, err
}

func (this *Rewrite) VisitExceptAll(node *algebra.ExceptAll) (r interface{}, err error) {
	if _, err = node.First().Accept(this); err == nil {
		_, err = node.Second().Accept(this)
	}
	return node, err
}
