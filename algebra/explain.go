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
	"github.com/couchbaselabs/query/value"
)

type Explain struct {
	stmt Statement `json:"stmt"`
}

func NewExplain(stmt Statement) *Explain {
	return &Explain{stmt}
}

func (this *Explain) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExplain(this)
}

func (this *Explain) Signature() value.Value {
	return value.NewValue(value.JSON.String())
}

func (this *Explain) Formalize() error {
	return this.stmt.Formalize()
}

func (this *Explain) Statement() Statement {
	return this.stmt
}
