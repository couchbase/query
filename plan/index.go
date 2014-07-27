//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plan

import (
	"github.com/couchbaselabs/query/algebra"
)

// Create index
type CreateIndex struct {
	readwrite
	node *algebra.CreateIndex
}

func NewCreateIndex(node *algebra.CreateIndex) *CreateIndex {
	return &CreateIndex{
		node: node,
	}
}

func (this *CreateIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateIndex(this)
}

func (this *CreateIndex) Node() *algebra.CreateIndex {
	return this.node
}

// Drop index
type DropIndex struct {
	readwrite
	node *algebra.DropIndex
}

func NewDropIndex(node *algebra.DropIndex) *DropIndex {
	return &DropIndex{
		node: node,
	}
}

func (this *DropIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropIndex(this)
}

func (this *DropIndex) Node() *algebra.DropIndex {
	return this.node
}

// Alter index
type AlterIndex struct {
	readwrite
	node *algebra.AlterIndex
}

func NewAlterIndex(node *algebra.AlterIndex) *AlterIndex {
	return &AlterIndex{
		node: node,
	}
}

func (this *AlterIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAlterIndex(this)
}

func (this *AlterIndex) Node() *algebra.AlterIndex {
	return this.node
}
