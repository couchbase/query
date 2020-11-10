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
	"github.com/couchbase/query/datastore"
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
	indexes  expression.Expressions `json:"indexes"`
	using    datastore.IndexType    `json:"using"`
	delete   bool                   `json:"delete"`
}

func NewUpdateStatistics(keyspace *KeyspaceRef, terms expression.Expressions,
	with value.Value) *UpdateStatistics {
	rv := &UpdateStatistics{
		keyspace: keyspace,
		terms:    terms,
		with:     with,
	}

	rv.stmt = rv
	return rv
}

func NewUpdateStatisticsIndex(keyspace *KeyspaceRef, indexes expression.Expressions,
	using datastore.IndexType, with value.Value) *UpdateStatistics {
	rv := &UpdateStatistics{
		keyspace: keyspace,
		with:     with,
		indexes:  indexes,
		using:    using,
	}

	rv.stmt = rv
	return rv
}

func NewUpdateStatisticsDelete(keyspace *KeyspaceRef, terms expression.Expressions) *UpdateStatistics {
	rv := &UpdateStatistics{
		keyspace: keyspace,
		terms:    terms,
		delete:   true,
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
	// terms and indexes are mutually exclusive
	if len(this.terms) > 0 {
		f := expression.NewKeyspaceFormalizer(this.keyspace.Keyspace(), nil)
		err := this.terms.MapExpressions(f)
		if err != nil {
			return err
		}
	} else if len(this.indexes) > 0 {
		f := expression.NewFormalizer("", nil)
		for i, e := range this.indexes {
			if ei, ok := e.(*expression.Identifier); ok {
				this.indexes[i] = expression.NewConstant(ei.Identifier())
			} else {
				expr, err := f.Map(e)
				if err != nil {
					return err
				}
				this.indexes[i] = expr
			}
		}
	}
	return nil
}

func (this *UpdateStatistics) MapExpressions(mapper expression.Mapper) error {
	// terms and indexes are mutually exclusive
	if len(this.terms) > 0 {
		return this.terms.MapExpressions(mapper)
	} else if len(this.indexes) > 0 {
		return this.indexes.MapExpressions(mapper)
	}
	return nil
}

func (this *UpdateStatistics) Expressions() expression.Expressions {
	return append(this.terms, this.indexes...)
}

func (this *UpdateStatistics) Privileges() (*auth.Privileges, errors.Error) {
	privs, err := PrivilegesFromPath(auth.PRIV_QUERY_SELECT, this.keyspace.path)
	if err != nil {
		return privs, err
	}

	for _, term := range this.terms {
		privs.AddAll(term.Privileges())
	}

	for _, index := range this.indexes {
		privs.AddAll(index.Privileges())
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

func (this *UpdateStatistics) Indexes() expression.Expressions {
	return this.indexes
}

func (this *UpdateStatistics) Using() datastore.IndexType {
	return this.using
}

func (this *UpdateStatistics) Delete() bool {
	return this.delete
}

func (this *UpdateStatistics) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "updateStatistics"}
	r["keyspaceRef"] = this.keyspace
	r["terms"] = this.terms
	r["with"] = this.with
	r["indexes"] = this.indexes
	r["using"] = this.using
	r["delete"] = this.delete

	return json.Marshal(r)
}

func (this *UpdateStatistics) Type() string {
	return "UPDATE_STATISTICS"
}
