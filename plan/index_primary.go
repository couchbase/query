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
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/value"
)

// Create primary index
type CreatePrimaryIndex struct {
	readwrite
	keyspace datastore.Keyspace
	node     *algebra.CreatePrimaryIndex
}

func NewCreatePrimaryIndex(keyspace datastore.Keyspace, node *algebra.CreatePrimaryIndex) *CreatePrimaryIndex {
	return &CreatePrimaryIndex{
		keyspace: keyspace,
		node:     node,
	}
}

func (this *CreatePrimaryIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreatePrimaryIndex(this)
}

func (this *CreatePrimaryIndex) New() Operator {
	return &CreatePrimaryIndex{}
}

func (this *CreatePrimaryIndex) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *CreatePrimaryIndex) Node() *algebra.CreatePrimaryIndex {
	return this.node
}

func (this *CreatePrimaryIndex) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *CreatePrimaryIndex) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "CreatePrimaryIndex"}
	this.node.Keyspace().MarshalKeyspace(r)
	r["index"] = this.node.Name()
	r["using"] = this.node.Using()
	if this.node.With() != nil {
		r["with"] = this.node.With()
	}
	if this.node.Partition() != nil && this.node.Partition().Strategy() != datastore.NO_PARTITION {
		q := make(map[string]interface{}, 2)
		q["exprs"] = this.node.Partition().Expressions()
		q["strategy"] = this.node.Partition().Strategy()
		r["partition"] = q
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *CreatePrimaryIndex) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string                      `json:"#operator"`
		Namespace string                      `json:"namespace"`
		Bucket    string                      `json:"bucket"`
		Scope     string                      `json:"scope"`
		Keyspace  string                      `json:"keyspace"`
		Node      *algebra.CreatePrimaryIndex `json:"node"`
		Index     string                      `json:"index"`
		Using     datastore.IndexType         `json:"using"`
		With      json.RawMessage             `json:"with"`
		Partition *struct {
			Exprs    []string                `json:"exprs"`
			Strategy datastore.PartitionType `json:"strategy"`
		} `json:"partition"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	var partition *algebra.IndexPartitionTerm
	if _unmarshalled.Partition != nil {
		exprs := make(expression.Expressions, len(_unmarshalled.Partition.Exprs))
		for i, p := range _unmarshalled.Partition.Exprs {
			exprs[i], err = parser.Parse(p)
			if err != nil {
				return err
			}
		}
		partition = algebra.NewIndexPartitionTerm(_unmarshalled.Partition.Strategy, exprs)
	}

	var with value.Value
	if len(_unmarshalled.With) > 0 {
		with = value.NewValue([]byte(_unmarshalled.With))
	}

	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Namespace, _unmarshalled.Bucket, _unmarshalled.Scope, _unmarshalled.Keyspace)
	if err != nil {
		return err
	}

	if _unmarshalled.Index != "" {
		ksref := algebra.NewKeyspaceRefFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
			_unmarshalled.Scope, _unmarshalled.Keyspace), "")
		this.node = algebra.NewCreatePrimaryIndex(_unmarshalled.Index, ksref,
			partition, _unmarshalled.Using, with)
	}

	return err
}

func (this *CreatePrimaryIndex) verify(prepared *Prepared) bool {
	var res bool

	this.keyspace, res = verifyKeyspace(this.keyspace, prepared)
	return res
}
