//  Copyright (c) 2014 Couchbase, Inc.
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

// Build indexes
type BuildIndexes struct {
	readwrite
	keyspace datastore.Keyspace
	node     *algebra.BuildIndexes
}

func NewBuildIndexes(keyspace datastore.Keyspace, node *algebra.BuildIndexes) *BuildIndexes {
	return &BuildIndexes{
		keyspace: keyspace,
		node:     node,
	}
}

func (this *BuildIndexes) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitBuildIndexes(this)
}

func (this *BuildIndexes) New() Operator {
	return &BuildIndexes{}
}

func (this *BuildIndexes) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *BuildIndexes) Node() *algebra.BuildIndexes {
	return this.node
}

func (this *BuildIndexes) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *BuildIndexes) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "BuildIndexes"}
	r["keyspace"] = this.keyspace.Name()
	r["namespace"] = this.keyspace.NamespaceId()
	r["using"] = this.node.Using()

	if len(this.node.Names()) > 0 {
		r["indexes"] = this.node.Names()
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *BuildIndexes) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_       string              `json:"#operator"`
		Keys    string              `json:"keyspace"`
		Names   string              `json:"namespace"`
		Using   datastore.IndexType `json:"using"`
		Indexes []string            `json:"indexes"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	ksref := algebra.NewKeyspaceRef(_unmarshalled.Names, _unmarshalled.Keys, "")
	this.node = algebra.NewBuildIndexes(ksref, _unmarshalled.Using, _unmarshalled.Indexes...)

	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)
	return err
}
