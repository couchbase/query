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
	"github.com/couchbaselabs/query/catalog"
)

type Join struct {
	outer      bool
	bucket     catalog.Bucket
	projection algebra.Path
	as         string
	keys       algebra.Expression
}

type Nest struct {
	outer   bool
	bucket  catalog.Bucket
	project algebra.Path
	as      string
	keys    algebra.Expression
}

type Unnest struct {
	outer   bool
	project algebra.Path
	as      string
}

func NewJoin(outer bool, bucket catalog.Bucket, project algebra.Path,
	as string, keys algebra.Expression) *Join {
	return &Join{outer, bucket, project, as, keys}
}

func (this *Join) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitJoin(this)
}

func NewNest(outer bool, bucket catalog.Bucket, project algebra.Path,
	as string, keys algebra.Expression) *Nest {
	return &Nest{outer, bucket, project, as, keys}
}

func (this *Nest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitNest(this)
}

func NewUnnest(outer bool, project algebra.Path, as string) *Unnest {
	return &Unnest{outer, project, as}
}

func (this *Unnest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnnest(this)
}
