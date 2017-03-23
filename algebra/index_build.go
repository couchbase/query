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
	"encoding/json"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

type BuildIndexes struct {
	statementBase

	keyspace *KeyspaceRef        `json:"keyspace"`
	using    datastore.IndexType `json:"using"`
	names    []string            `json:"names"`
}

func NewBuildIndexes(keyspace *KeyspaceRef, using datastore.IndexType, names ...string) *BuildIndexes {
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
	return nil
}

func (this *BuildIndexes) MapExpressions(mapper expression.Mapper) error {
	return nil
}

func (this *BuildIndexes) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *BuildIndexes) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	fullName := this.keyspace.FullName()
	privs.Add(fullName, auth.PRIV_QUERY_BUILD_INDEX)
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

func (this *BuildIndexes) Names() []string {
	return this.names
}

func (this *BuildIndexes) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "BuildIndexes"}
	r["keyspaceRef"] = this.keyspace
	r["using"] = this.using
	r["names"] = this.names
	return json.Marshal(r)
}
