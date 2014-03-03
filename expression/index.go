//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"time"

	"github.com/couchbaselabs/query/value"
)

type Index interface {
	BucketPath() string
	Equal() CompositeExpression
	Range() CompositeExpression
	Condition() Expression
}

type Spans []*Span

type Span struct {
	Equal value.CompositeValue
	Range *Range
}

type Range struct {
	Low       value.CompositeValue
	High      value.CompositeValue
	Inclusion Inclusion
}

// Inclusion controls how the boundary values of a range are treated.
type Inclusion int

const (
	NEITHER Inclusion = iota
	LOW
	HIGH
	BOTH
)

type IndexContext struct {
	now time.Time
}

func NewIndexContext() Context {
	return &IndexContext{
		now: time.Now(),
	}
}

func (this *IndexContext) Now() time.Time {
	return this.now
}
