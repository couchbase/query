//  Copyright 2014-Present Couchbase, Inc.
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

type BuildIndexes struct {
	statementBase

	keyspace *KeyspaceRef           `json:"keyspace"`
	using    datastore.IndexType    `json:"using"`
	names    expression.Expressions `json:"names"`
}

func NewBuildIndexes(keyspace *KeyspaceRef, using datastore.IndexType, names ...expression.Expression) *BuildIndexes {
	rv := &BuildIndexes{
		keyspace: keyspace,
		using:    using,
		names:    names,
	}

	rv.stmt = rv
	return rv
}

func (this *BuildIndexes) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitBuildIndexes(this)
}

func (this *BuildIndexes) Signature() value.Value {
	return nil
}

func (this *BuildIndexes) Formalize() error {
	f := expression.NewFormalizer("", nil)
	for i, e := range this.names {
		if ei, ok := e.(*expression.Identifier); ok {
			this.names[i] = expression.NewConstant(ei.Identifier())
		} else {
			expr, err := f.Map(e)
			if err != nil {
				return err
			}
			this.names[i] = expr
		}
	}
	return nil
}

func (this *BuildIndexes) MapExpressions(mapper expression.Mapper) error {
	return this.names.MapExpressions(mapper)
}

func (this *BuildIndexes) Expressions() expression.Expressions {
	return this.names
}

/*
Returns all required privileges.
*/
func (this *BuildIndexes) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	fullName := this.keyspace.FullName()
	privs.Add(fullName, auth.PRIV_QUERY_BUILD_INDEX, auth.PRIV_PROPS_NONE)
	return privs, nil
}

func (this *BuildIndexes) Keyspace() *KeyspaceRef {
	return this.keyspace
}

/*
Returns the index type string for the using clause.
*/
func (this *BuildIndexes) Using() datastore.IndexType {
	return this.using
}

func (this *BuildIndexes) Names() expression.Expressions {
	return this.names
}

func (this *BuildIndexes) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "BuildIndexes"}
	r["keyspaceRef"] = this.keyspace
	r["using"] = this.using
	r["names"] = this.names
	return json.Marshal(r)
}

func (this *BuildIndexes) Type() string {
	return "BUILD_INDEX"
}

func (this *BuildIndexes) String() string {
	var s strings.Builder
	s.WriteString("BUILD INDEX ON ")
	s.WriteString(this.keyspace.Path().ProtectedString())
	s.WriteString("(")
	for i, n := range this.names {
		if i > 0 {
			s.WriteString(", ")
		}
		s.WriteString(n.String())
	}
	s.WriteString(")")
	if this.using != "" && this.using != datastore.DEFAULT {
		s.WriteString(" USING ")
		s.WriteString(strings.ToUpper(string(this.using)))
	}

	return s.String()
}
