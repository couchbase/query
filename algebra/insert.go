//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"github.com/couchbaselabs/query/expression"
)

type Insert struct {
	bucket    *BucketRef            `json:"bucket"`
	key       expression.Expression `json:"key"`
	values    expression.Expression `json:"values"`
	query     *Select               `json:"query"`
	as        string                `json:"as"`
	returning ResultTerms           `json:"returning"`
}

func NewInsert(bucket *BucketRef, key, values expression.Expression, query *Select,
	as string, returning ResultTerms) *Insert {
	return &Insert{bucket, key, values, query, as, returning}
}

func (this *Insert) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInsert(this)
}
