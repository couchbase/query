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

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

/*
Represents the keyspace_ref used in DML statements
*/
type ScopeRef struct {
	path *Path  `json:"path"`
	as   string `json:"as"`
}

func NewScopeRefFromPath(path *Path, as string) *ScopeRef {
	return &ScopeRef{path, as}
}

/*
Qualify identifiers for the keyspace. It also makes sure that the
keyspace term contains a name or alias.
*/
func (this *ScopeRef) Formalize() (f *expression.Formalizer, err error) {
	keyspace := this.Alias()
	if keyspace == "" {
		err = errors.NewNoTermNameError("Keyspace", "semantics.keyspace.reference_requires_name_or_alias")
		return
	}

	f = expression.NewFormalizer(keyspace, nil)
	return
}

func (this *ScopeRef) Path() *Path {
	return this.path
}

/*
Returns the namespace string.
*/
func (this *ScopeRef) Namespace() string {
	return this.path.Namespace()

}

/*
Set the default namespace.
FIXME ideally this should go
*/
func (this *ScopeRef) SetDefaultNamespace(namespace string) {
	this.path.SetDefaultNamespace(namespace)
}

/*
Returns the scope string.
*/
func (this *ScopeRef) Scope() string {
	return this.path.Scope()
}

/*
Returns the AS alias string.
*/
func (this *ScopeRef) As() string {
	return this.as
}

/*
Returns the alias as the keyspace or the as string
based on if as is empty.
*/
func (this *ScopeRef) Alias() string {
	if this.as != "" {
		return this.as
	} else {
		return this.path.Alias()
	}
}

/*
Marshals input into byte array.
*/
func (this *ScopeRef) MarshalJSON() ([]byte, error) {
	r := make(map[string]interface{}, 3)
	r["path"] = this.path
	if this.as != "" {
		r["as"] = this.as
	}

	return json.Marshal(r)
}

func (this *ScopeRef) MarshalKeyspace(m map[string]interface{}) {
	this.path.marshalKeyspace(m)
}

/*
Returns the full keyspace name, including the namespace.
*/
func (this *ScopeRef) FullName() string {
	return this.path.SimpleString()
}
