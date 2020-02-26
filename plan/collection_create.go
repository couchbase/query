//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plan

import (
	"encoding/json"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
)

// Create collection
type CreateCollection struct {
	readwrite
	scope datastore.Scope
	node  *algebra.CreateCollection
}

func NewCreateCollection(scope datastore.Scope, node *algebra.CreateCollection) *CreateCollection {
	return &CreateCollection{
		scope: scope,
		node:  node,
	}
}

func (this *CreateCollection) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateCollection(this)
}

func (this *CreateCollection) New() Operator {
	return &CreateCollection{}
}

func (this *CreateCollection) Scope() datastore.Scope {
	return this.scope
}

func (this *CreateCollection) Node() *algebra.CreateCollection {
	return this.node
}

func (this *CreateCollection) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *CreateCollection) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "CreateCollection"}
	this.node.Keyspace().MarshalKeyspace(r)

	if f != nil {
		f(r)
	}
	return r
}

func (this *CreateCollection) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string `json:"#operator"`
		Namespace string `json:"namespace"`
		Bucket    string `json:"bucket"`
		Scope     string `json:"scope"`
		Keyspace  string `json:"keyspace"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	ksref := algebra.NewKeyspaceRefFromPath(algebra.NewPathLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), "")
	this.scope, err = datastore.GetScope(ksref.Path().Parts()[0:3]...)
	if err != nil {
		return err
	}

	this.node = algebra.NewCreateCollection(ksref)
	return nil
}

func (this *CreateCollection) verify(prepared *Prepared) bool {
	var res bool

	this.scope, res = verifyScope(this.scope, prepared)
	return res
}
