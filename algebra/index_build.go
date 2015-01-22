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

	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type BuildIndexes struct {
	statementBase

	keyspace *KeyspaceRef `json:"keyspace"`
	names    []string     `json:"names"`
}

func NewBuildIndexes(keyspace *KeyspaceRef, names ...string) *BuildIndexes {
	rv := &BuildIndexes{
		keyspace: keyspace,
		names:    names,
	}

	rv.stmt = rv
	return rv
}

func (this *BuildIndexes) Accept(visitor Visitor) (interface{}, error) {
	// FIXME
	//return visitor.VisitBuildIndexes(this)
	return nil, nil
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
func (this *BuildIndexes) Privileges() (datastore.Privileges, errors.Error) {
	return datastore.Privileges{
		this.keyspace.Namespace() + ":" + this.keyspace.Keyspace(): datastore.PRIV_DDL,
	}, nil
}

func (this *BuildIndexes) Keyspace() *KeyspaceRef {
	return this.keyspace
}

func (this *BuildIndexes) Names() []string {
	return this.names
}

func (this *BuildIndexes) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "BuildIndexes"}
	r["keyspaceRef"] = this.keyspace
	r["names"] = this.names
	return json.Marshal(r)
}
