//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execute

import (
	_ "fmt"

	"github.com/couchbaselabs/query/algebra"
)

// Distincting of input data.
type InitialDistinct struct {
	operatorBase
}

// Distincting of distincts. Recursable.
type SubsequentDistinct struct {
	operatorBase
}

func NewInitialDistinct() *InitialDistinct {
	return &InitialDistinct{}
}

func (this *InitialDistinct) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInitialDistinct(this)
}

func (this *InitialDistinct) Copy() Operator {
	return &InitialDistinct{this.operatorBase.copy()}
}

func (this *InitialDistinct) Run(context algebra.Context) {
}

func NewSubsequentDistinct() *SubsequentDistinct {
	return &SubsequentDistinct{}
}

func (this *SubsequentDistinct) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSubsequentDistinct(this)
}

func (this *SubsequentDistinct) Copy() Operator {
	return &SubsequentDistinct{this.operatorBase.copy()}
}

func (this *SubsequentDistinct) Run(context algebra.Context) {
}
