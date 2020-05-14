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

// Flush collection
type FlushCollection struct {
	ddl
	keyspace datastore.Keyspace
	node     *algebra.FlushCollection
}

func NewFlushCollection(keyspace datastore.Keyspace, node *algebra.FlushCollection) *FlushCollection {
	return &FlushCollection{
		keyspace: keyspace,
		node:     node,
	}
}

func (this *FlushCollection) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFlushCollection(this)
}

func (this *FlushCollection) New() Operator {
	return &FlushCollection{}
}

func (this *FlushCollection) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *FlushCollection) Node() *algebra.FlushCollection {
	return this.node
}

func (this *FlushCollection) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *FlushCollection) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "FlushCollection"}
	this.node.Keyspace().MarshalKeyspace(r)

	if f != nil {
		f(r)
	}
	return r
}

func (this *FlushCollection) UnmarshalJSON(body []byte) error {
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
	this.keyspace, err = datastore.GetKeyspace(ksref.Path().Parts()...)
	if err != nil {
		return err
	}

	this.node = algebra.NewFlushCollection(ksref)
	return nil
}

func (this *FlushCollection) verify(prepared *Prepared) bool {
	var res bool

	this.keyspace, res = verifyKeyspace(this.keyspace, prepared)
	return res
}
