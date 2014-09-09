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
	"github.com/couchbaselabs/query/datastore"
)

// Create primary index
type CreatePrimaryIndex struct {
	readwrite
	keyspace datastore.Keyspace
}

func NewCreatePrimaryIndex(keyspace datastore.Keyspace) *CreatePrimaryIndex {
	return &CreatePrimaryIndex{
		keyspace: keyspace,
	}
}

func (this *CreatePrimaryIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreatePrimaryIndex(this)
}

func (this *CreatePrimaryIndex) Keyspace() datastore.Keyspace {
	return this.keyspace
}

// Create index
type CreateIndex struct {
	readwrite
	keyspace datastore.Keyspace
	node     *algebra.CreateIndex
}

func NewCreateIndex(keyspace datastore.Keyspace, node *algebra.CreateIndex) *CreateIndex {
	return &CreateIndex{
		keyspace: keyspace,
		node:     node,
	}
}

func (this *CreateIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateIndex(this)
}

func (this *CreateIndex) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *CreateIndex) Node() *algebra.CreateIndex {
	return this.node
}

// Drop index
type DropIndex struct {
	readwrite
	index datastore.Index
	node  *algebra.DropIndex
}

func NewDropIndex(index datastore.Index, node *algebra.DropIndex) *DropIndex {
	return &DropIndex{
		index: index,
		node:  node,
	}
}

func (this *DropIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropIndex(this)
}

func (this *DropIndex) Index() datastore.Index {
	return this.index
}

func (this *DropIndex) Node() *algebra.DropIndex {
	return this.node
}

// Alter index
type AlterIndex struct {
	readwrite
	index datastore.Index
	node  *algebra.AlterIndex
}

func NewAlterIndex(index datastore.Index, node *algebra.AlterIndex) *AlterIndex {
	return &AlterIndex{
		index: index,
		node:  node,
	}
}

func (this *AlterIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAlterIndex(this)
}

func (this *AlterIndex) Index() datastore.Index {
	return this.index
}

func (this *AlterIndex) Node() *algebra.AlterIndex {
	return this.node
}
