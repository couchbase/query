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
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type KeyspaceRef struct {
	namespace string `json:"namespace"`
	keyspace  string `json:"keyspace"`
	as        string `json:"as"`
}

func NewKeyspaceRef(namespace, keyspace, as string) *KeyspaceRef {
	return &KeyspaceRef{namespace, keyspace, as}
}

func (this *KeyspaceRef) Formalize() (f *expression.Formalizer, err error) {
	keyspace := this.Alias()
	if keyspace == "" {
		err = errors.NewError(nil, "Keyspace term must have a name or alias.")
		return
	}

	allowed := value.NewValue(make(map[string]interface{}))
	allowed.SetField(keyspace, keyspace)

	f = expression.NewFormalizer()
	f.Keyspace = keyspace
	f.Allowed = allowed
	return
}

func (this *KeyspaceRef) Namespace() string {
	return this.namespace
}

func (this *KeyspaceRef) Keyspace() string {
	return this.keyspace
}

func (this *KeyspaceRef) As() string {
	return this.as
}

func (this *KeyspaceRef) Alias() string {
	if this.as != "" {
		return this.as
	} else {
		return this.keyspace
	}
}

func (this *KeyspaceRef) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "keyspaceRef"}
	r["as"] = this.as
	r["keyspace"] = this.keyspace
	r["namespace"] = this.namespace
	return json.Marshal(r)
}
