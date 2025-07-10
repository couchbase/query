//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"encoding/json"
	"strings"

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
	indexAll bool                   `json:"index_all"`
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

func NewUpdateStatisticsIndexAll(keyspace *KeyspaceRef,
	using datastore.IndexType, with value.Value) *UpdateStatistics {
	rv := &UpdateStatistics{
		keyspace: keyspace,
		with:     with,
		using:    using,
		indexAll: true,
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

func (this *UpdateStatistics) IndexAll() bool {
	return this.indexAll
}

func (this *UpdateStatistics) String() string {
	path := this.keyspace.path
	if path == nil {
		// not expected
		return ""
	}
	using := false
	var sb strings.Builder
	sb.WriteString("UPDATE STATISTICS FOR ")
	sb.WriteString(path.ProtectedString())
	if this.delete {
		if len(this.terms) == 0 {
			sb.WriteString(" DELETE ALL")
		} else {
			sb.WriteString(" DELETE (")
			for i, term := range this.terms {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(strings.Trim(term.String(), "\""))
			}
			sb.WriteString(")")
		}
	} else if this.indexAll {
		sb.WriteString(" INDEX ALL")
		using = true
	} else if len(this.indexes) > 0 {
		sb.WriteString(" INDEX (")
		for i, idx := range this.indexes {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(strings.Trim(idx.String(), "\""))
		}
		sb.WriteString(")")
		using = true
	} else if len(this.terms) > 0 {
		sb.WriteString(" (")
		for i, term := range this.terms {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(strings.Trim(term.String(), "\""))
		}
		sb.WriteString(")")
	}
	if using && this.using != datastore.DEFAULT {
		// currently only GSI indexes supported
		switch this.using {
		case datastore.GSI:
			sb.WriteString(" USING GSI")
		}
	}
	if this.with != nil {
		sb.WriteString(" WITH ")
		sb.WriteString(this.with.ToString())
	}
	return sb.String()
}

func (this *UpdateStatistics) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "updateStatistics"}
	r["keyspaceRef"] = this.keyspace
	r["terms"] = this.terms
	r["with"] = this.with
	r["indexes"] = this.indexes
	r["using"] = this.using
	r["index_all"] = this.indexAll
	r["delete"] = this.delete

	return json.Marshal(r)
}

func (this *UpdateStatistics) Type() string {
	return "UPDATE_STATISTICS"
}
