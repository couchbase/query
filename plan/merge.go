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
	_ "fmt"

	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/catalog"
)

type ComputeMerge struct {
	update *MergeUpdate
	delete *MergeDelete
	insert *MergeInsert
}

type MergeUpdate struct {
	node *algebra.MergeUpdate
}

type MergeDelete struct {
	node *algebra.MergeDelete
}

type MergeInsert struct {
	node *algebra.MergeInsert
}

type SendMerge struct {
	bucket catalog.Bucket
}

func NewComputeMerge(update *MergeUpdate, delete *MergeDelete,
	insert *MergeInsert) *ComputeMerge {
	return &ComputeMerge{update, delete, insert}
}

func (this *ComputeMerge) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitComputeMerge(this)
}

func NewMergeUpdate(node *algebra.MergeUpdate) *MergeUpdate {
	return &MergeUpdate{node}
}

func (this *MergeUpdate) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMergeUpdate(this)
}

func NewMergeDelete(node *algebra.MergeDelete) *MergeDelete {
	return &MergeDelete{node}
}

func (this *MergeDelete) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMergeDelete(this)
}

func NewMergeInsert(node *algebra.MergeInsert) *MergeInsert {
	return &MergeInsert{node}
}

func (this *MergeInsert) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMergeInsert(this)
}

func NewSendMerge(bucket catalog.Bucket) *SendMerge {
	return &SendMerge{bucket}
}

func (this *SendMerge) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendMerge(this)
}
