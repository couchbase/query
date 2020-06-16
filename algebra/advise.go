//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

type Advise struct {
	statementBase

	stmt    Statement `json:"stmt"`
	query   string    `json:"query"`
	context interface{}
}

func NewAdvise(stmt Statement, text string) *Advise {
	rv := &Advise{
		stmt:  stmt,
		query: text,
	}
	rv.statementBase.stmt = rv
	return rv
}

func (this *Advise) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAdvise(this)
}

func (this *Advise) Signature() value.Value {
	return value.NewValue(value.JSON.String())
}

func (this *Advise) Formalize() error {
	return this.stmt.Formalize()
}

func (this *Advise) MapExpressions(mapper expression.Mapper) error {
	return this.stmt.MapExpressions(mapper)
}

func (this *Advise) Expressions() expression.Expressions {
	return this.stmt.Expressions()
}

func (this *Advise) Privileges() (*auth.Privileges, errors.Error) {
	return this.stmt.Privileges()
}

func (this *Advise) Statement() Statement {
	return this.stmt
}

func (this *Advise) Query() string {
	return this.query
}

func (this *Advise) Type() string {
	return "Advise"
}

func (this *Advise) SetContext(context interface{}) {
	this.context = context
}

func (this *Advise) Context() interface{} {
	return this.context
}
