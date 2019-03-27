//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"encoding/json"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
Represents UPDATE STATISTICS statement
*/
type UpdateStatistics struct {
	statementBase

	keyspace *KeyspaceRef           `json:"keyspace"`
	terms    expression.Expressions `json:"terms"`
	with     value.Value            `json:"with"`
}

func NewUpdateStatistics(keyspace *KeyspaceRef, terms expression.Expressions, with value.Value) *UpdateStatistics {
	rv := &UpdateStatistics{
		keyspace: keyspace,
		terms:    terms,
		with:     with,
	}

	rv.stmt = rv
	return rv
}

func (this *UpdateStatistics) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUpdateStatistics(this)
}

func (this *UpdateStatistics) Signature() value.Value {
	return nil
}

func (this *UpdateStatistics) Formalize() error {
	f := expression.NewKeyspaceFormalizer(this.keyspace.Keyspace(), nil)
	return this.MapExpressions(f)
}

func (this *UpdateStatistics) MapExpressions(mapper expression.Mapper) (err error) {
	for i, term := range this.Expressions() {
		this.terms[i], err = mapper.Map(term)
		if err != nil {
			return
		}
	}

	return
}

func (this *UpdateStatistics) Expressions() expression.Expressions {
	return this.terms
}

func (this *UpdateStatistics) Privileges() (*auth.Privileges, errors.Error) {
	privs, err := privilegesFromPath(this.keyspace.path)
	if err != nil {
		return privs, err
	}

	for _, term := range this.terms {
		privs.AddAll(term.Privileges())
	}

	return privs, nil
}

func (this *UpdateStatistics) Keyspace() *KeyspaceRef {
	return this.keyspace
}

func (this *UpdateStatistics) Terms() expression.Expressions {
	return this.terms
}

func (this *UpdateStatistics) With() value.Value {
	return this.with
}

func (this *UpdateStatistics) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "updateStatistics"}
	r["keyspaceRef"] = this.keyspace
	r["terms"] = this.terms
	r["with"] = this.with

	return json.Marshal(r)
}

func (this *UpdateStatistics) Type() string {
	return "UPDATE_STATISTICS"
}
