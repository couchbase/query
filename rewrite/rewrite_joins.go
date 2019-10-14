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

func (this *Rewrite) visitJoin(left algebra.FromTerm, right algebra.SimpleFromTerm) (err error) {
	if _, err = left.Accept(this); err == nil {
		_, err = right.Accept(this)
	}

	return err
}

func (this *Rewrite) VisitJoin(node *algebra.Join) (r interface{}, err error) {
	return node, this.visitJoin(node.Left(), node.Right())
}

func (this *Rewrite) VisitIndexJoin(node *algebra.IndexJoin) (interface{}, error) {
	return node, this.visitJoin(node.Left(), node.Right())
}

func (this *Rewrite) VisitAnsiJoin(node *algebra.AnsiJoin) (r interface{}, err error) {
	return node, this.visitJoin(node.Left(), node.Right())
}

func (this *Rewrite) VisitNest(node *algebra.Nest) (interface{}, error) {
	return node, this.visitJoin(node.Left(), node.Right())
}

func (this *Rewrite) VisitIndexNest(node *algebra.IndexNest) (interface{}, error) {
	return node, this.visitJoin(node.Left(), node.Right())
}

func (this *Rewrite) VisitAnsiNest(node *algebra.AnsiNest) (r interface{}, err error) {
	return node, this.visitJoin(node.Left(), node.Right())
}

func (this *Rewrite) VisitUnnest(node *algebra.Unnest) (r interface{}, err error) {
	if _, err = node.Left().Accept(this); err == nil {
		_, err = this.Map(node.Expression())
	}
	return node, err
}
