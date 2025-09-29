//  Copyright 2016-Present Couchbase, Inc.
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

type InferKeyspace struct {
	statementBase

	keyspace *KeyspaceRef            `json:"keyspace"`
	using    datastore.InferenceType `json:"using"`
	with     value.Value             `json:"with"`
}

func NewInferKeyspace(keyspace *KeyspaceRef, using datastore.InferenceType,
	with value.Value) *InferKeyspace {
	rv := &InferKeyspace{
		keyspace: keyspace,
		using:    using,
		with:     with,
	}

	rv.stmt = rv
	return rv
}

func (this *InferKeyspace) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInferKeyspace(this)
}

func (this *InferKeyspace) Signature() value.Value {
	return nil
}

func (this *InferKeyspace) Formalize() error {
	return nil
}

func (this *InferKeyspace) MapExpressions(mapper expression.Mapper) error {
	return nil
}

func (this *InferKeyspace) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *InferKeyspace) Privileges() (*auth.Privileges, errors.Error) {
	privs, err := PrivilegesFromPath(auth.PRIV_QUERY_SELECT, this.keyspace.path)
	if err != nil {
		return privs, err
	}

	return privs, nil
}

func (this *InferKeyspace) Keyspace() *KeyspaceRef {
	return this.keyspace
}

/*
Returns the index type string for the using clause.
*/
func (this *InferKeyspace) Using() datastore.InferenceType {
	return this.using
}

func (this *InferKeyspace) With() value.Value {
	return this.with
}

func (this *InferKeyspace) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "InferKeyspace"}
	r["keyspaceRef"] = this.keyspace
	r["using"] = this.using
	r["with"] = this.with
	return json.Marshal(r)
}

func (this *InferKeyspace) Type() string {
	return "INFER"
}

func (this *InferKeyspace) String() string {
	var s strings.Builder
	s.WriteString("INFER KEYSPACE ")
	s.WriteString(this.Keyspace().Path().ProtectedString())

	if this.with != nil {
		s.WriteString(" WITH ")
		s.WriteString(this.with.String())
	}

	return s.String()
}
