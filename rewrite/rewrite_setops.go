//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
