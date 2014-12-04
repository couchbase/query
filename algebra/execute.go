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
	"github.com/couchbaselabs/query/value"
)

type Execute struct {
	// prepared is a json object that represents a plan.Prepared
	prepared expression.Expression `json:"prepared"`
}

func NewExecute(prepared expression.Expression) *Execute {
	return &Execute{prepared}
}

func (this *Execute) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExecute(this)
}

func (this *Execute) Signature() value.Value {
	return value.NewValue(value.JSON.String())
}

func (this *Execute) Formalize() error {
	return nil
}

func (this *Execute) MapExpressions(mapper expression.Mapper) error {
	return nil
}

func (this *Execute) Prepared() expression.Expression {
	return this.prepared
}
