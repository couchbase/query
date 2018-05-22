//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package semantics

import (
	"github.com/couchbase/query/algebra"
)

func (this *SemChecker) visitSetop(first algebra.Subresult, second algebra.Subresult) (interface{}, error) {
	f, err := first.Accept(this)
	if err != nil {
		return f, err
	}
	return second.Accept(this)
}

func (this *SemChecker) VisitUnion(node *algebra.Union) (interface{}, error) {
	return this.visitSetop(node.First(), node.Second())
}

func (this *SemChecker) VisitUnionAll(node *algebra.UnionAll) (interface{}, error) {
	return this.visitSetop(node.First(), node.Second())
}

func (this *SemChecker) VisitIntersect(node *algebra.Intersect) (interface{}, error) {
	return this.visitSetop(node.First(), node.Second())
}

func (this *SemChecker) VisitIntersectAll(node *algebra.IntersectAll) (interface{}, error) {
	return this.visitSetop(node.First(), node.Second())
}

func (this *SemChecker) VisitExcept(node *algebra.Except) (interface{}, error) {
	return this.visitSetop(node.First(), node.Second())
}

func (this *SemChecker) VisitExceptAll(node *algebra.ExceptAll) (interface{}, error) {
	return this.visitSetop(node.First(), node.Second())
}
