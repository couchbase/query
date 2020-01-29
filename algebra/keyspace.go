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

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

/*
Represents the keyspace_ref used in DML statements
*/
type KeyspaceRef struct {
	path *Path  `json:"path"`
	as   string `json:"as"`
}

func NewKeyspaceRefFromPath(path *Path, as string) *KeyspaceRef {
	return &KeyspaceRef{path, as}
}

func NewKeyspaceRefWithContext(keyspace, as, namespace, queryContext string) *KeyspaceRef {
	return &KeyspaceRef{NewPathWithContext(keyspace, namespace, queryContext), as}
}

/*
Qualify identifiers for the keyspace. It also makes sure that the
keyspace term contains a name or alias.
*/
func (this *KeyspaceRef) Formalize() (f *expression.Formalizer, err error) {
	keyspace := this.Alias()
	if keyspace == "" {
		err = errors.NewNoTermNameError("Keyspace", "semantics.keyspace.reference_requires_name_or_alias")
		return
	}

	f = expression.NewFormalizer(keyspace, nil)
	return
}

func (this *KeyspaceRef) Path() *Path {
	return this.path
}

/*
Returns the namespace string.
*/
func (this *KeyspaceRef) Namespace() string {
	return this.path.Namespace()

}

/*
Set the default namespace.
FIXME ideally this should go
*/
func (this *KeyspaceRef) SetDefaultNamespace(namespace string) {
	this.path.SetDefaultNamespace(namespace)
}

/*
Returns the keyspace string.
*/
func (this *KeyspaceRef) Keyspace() string {
	return this.path.Keyspace()
}

/*
Returns the AS alias string.
*/
func (this *KeyspaceRef) As() string {
	return this.as
}

/*
Returns the alias as the keyspace or the as string
based on if as is empty.
*/
func (this *KeyspaceRef) Alias() string {
	if this.as != "" {
		return this.as
	} else {
		return this.path.Alias()
	}
}

/*
Marshals input into byte array.
*/
func (this *KeyspaceRef) MarshalJSON() ([]byte, error) {
	r := make(map[string]interface{}, 3)
	r["path"] = this.path
	if this.as != "" {
		r["as"] = this.as
	}

	return json.Marshal(r)
}

func (this *KeyspaceRef) MarshalKeyspace(m map[string]interface{}) {
	this.path.marshalKeyspace(m)
}

/*
Returns the full keyspace name, including the namespace.
*/
func (this *KeyspaceRef) FullName() string {
	return this.path.SimpleString()
}
