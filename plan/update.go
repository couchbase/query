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

// Copy-before-write, so that all reads use old values
type Copy struct {
}

// Write to copy
type Set struct {
	node *algebra.Set
}

// Write to copy
type Unset struct {
	node *algebra.Unset
}

// Send to bucket
type Update struct {
	bucket catalog.Bucket
}

func NewCopy() *Copy {
	return &Copy{}
}

func (this *Copy) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCopy(this)
}

func NewSet(node *algebra.Set) *Set {
	return &Set{node}
}

func (this *Set) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSet(this)
}

func NewUnset(node *algebra.Unset) *Unset {
	return &Unset{node}
}

func (this *Unset) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnset(this)
}

func NewUpdate(bucket catalog.Bucket) *Update {
	return &Update{bucket}
}

func (this *Update) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUpdate(this)
}
