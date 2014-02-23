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
	"github.com/couchbaselabs/query/catalog"
	"github.com/couchbaselabs/query/expression"
)

type SendInsert struct {
	bucket catalog.Bucket
	key    expression.Expression
}

func NewSendInsert(bucket catalog.Bucket, key expression.Expression) *SendInsert {
	return &SendInsert{bucket, key}
}

func (this *SendInsert) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendInsert(this)
}

func (this *SendInsert) Bucket() catalog.Bucket {
	return this.bucket
}

func (this *SendInsert) Key() expression.Expression {
	return this.key
}
