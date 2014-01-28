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
	_ "fmt"
	_ "github.com/couchbaselabs/query/value"
)

type SelectNode struct {
	from       FromNode             `json:"from"`
	where      Expression           `json:"where"`
	groupBy    ExpressionList       `json:"group_by"`
	having     Expression           `json:"having"`
	projection ResultExpressionList `json:"select"`
	distinct   bool                 `json:"distinct"`
	orderBy    SortExpressionList   `json:"orderby"`
	limit      Expression           `json:"limit"`
	offset     Expression           `json:"offset"`
}

type FromNode interface {
	GetAs() string
	GetProjection() Path
	GetKeys() Expression
	PrimaryBucket() *FromBucketNode
}

type FromBucketNode struct {
	pool       string
	bucket     string
	projection Path
	as         string
	keys       Expression
}

type Joiner int

const (
	JOIN   Joiner = 1
	NEST          = 2
	UNNEST        = 3
)

type JoinNode struct {
	left   FromNode
	outer  bool
	joiner Joiner
	right  *FromBucketNode
}

type ResultExpression struct {
	star bool       `json:"star"`
	expr Expression `json:"expr"`
	as   string     `json:"as"`
}

type ResultExpressionList []*ResultExpression

type SortExpression struct {
	expr      Expression `json:"expr"`
	ascending bool       `json:"asc"`
}

type SortExpressionList []*SortExpression
