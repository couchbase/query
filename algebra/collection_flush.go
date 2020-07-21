//  Copyright (c) 2020 Couchbase, Inc.
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
Represents the Flush collection ddl statement. Type FlushCollection is
a struct that contains fields mapping to each clause in the
flush collection statement, namely the keyspace and the collection name.
*/
type FlushCollection struct {
	statementBase

	keyspace *KeyspaceRef `json:"keyspace"`
}

/*
The function NewFlushCollection returns a pointer to the
FlushCollection struct with the input argument values as fields.
*/
func NewFlushCollection(keyspace *KeyspaceRef) *FlushCollection {
	rv := &FlushCollection{
		keyspace: keyspace,
	}

	rv.stmt = rv
	return rv
}

/*
It calls the VisitFlushCollection method by passing in the
receiver and returns the interface. It is a visitor
pattern.
*/
func (this *FlushCollection) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFlushCollection(this)
}

/*
Returns nil.
*/
func (this *FlushCollection) Signature() value.Value {
	return nil
}

/*
Returns nil.
*/
func (this *FlushCollection) Formalize() error {
	return nil
}

/*
Returns nil.
*/
func (this *FlushCollection) MapExpressions(mapper expression.Mapper) error {
	return nil
}

/*
Returns all contained Expressions.
*/
func (this *FlushCollection) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *FlushCollection) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	fullName := this.keyspace.Path().BucketPath().FullName()
	privs.Add(fullName, auth.PRIV_QUERY_BUCKET_ADMIN, auth.PRIV_PROPS_NONE)
	return privs, nil
}

/*
Return the keyspace.
*/
func (this *FlushCollection) Keyspace() *KeyspaceRef {
	return this.keyspace
}

/*
Marshals input receiver into byte array.
*/
func (this *FlushCollection) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "flushCollection"}
	r["keyspaceRef"] = this.keyspace
	return json.Marshal(r)
}

func (this *FlushCollection) Type() string {
	return "FLUSH_COLLECTION"
}
