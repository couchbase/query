//  Copyright (c) 2018 Couchbase, Inc.
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

// Update Statistics
type UpdateStatistics struct {
	execution
	keyspace datastore.Keyspace
	node     *algebra.UpdateStatistics
}

func NewUpdateStatistics(keyspace datastore.Keyspace,
	node *algebra.UpdateStatistics) *UpdateStatistics {
	return &UpdateStatistics{
		keyspace: keyspace,
		node:     node,
	}
}

func (this *UpdateStatistics) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUpdateStatistics(this)
}

func (this *UpdateStatistics) New() Operator {
	return &UpdateStatistics{}
}

func (this *UpdateStatistics) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *UpdateStatistics) Node() *algebra.UpdateStatistics {
	return this.node
}

func (this *UpdateStatistics) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *UpdateStatistics) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "UpdateStatistics"}
	this.node.Keyspace().MarshalKeyspace(r)

	if len(this.node.Terms()) > 0 {
		terms := make([]interface{}, 0, len(this.node.Terms()))
		for _, term := range this.node.Terms() {
			terms = append(terms, expression.NewStringer().Visit(term))
		}
		r["terms"] = terms
	}
	if len(this.node.Indexes()) > 0 {
		indexes := make([]interface{}, 0, len(this.node.Indexes()))
		for _, index := range this.node.Indexes() {
			indexes = append(indexes, expression.NewStringer().Visit(index))
		}
		r["indexes"] = indexes
		r["using"] = this.node.Using()
	}
	if this.node.IndexAll() {
		r["index_all"] = this.node.IndexAll()
		r["using"] = this.node.Using()
	}
	if this.node.Delete() {
		r["delete"] = this.node.Delete()
	}
	if this.node.With() != nil {
		r["with"] = this.node.With()
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *UpdateStatistics) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string              `json:"#operator"`
		Namespace string              `json:"namespace"`
		Bucket    string              `json:"bucket"`
		Scope     string              `json:"scope"`
		Keyspace  string              `json:"keyspace"`
		Terms     []string            `json:"terms"`
		Indexes   []string            `json:"indexes"`
		Using     datastore.IndexType `json:"using"`
		IndexAll  bool                `json:"index_all"`
		Delete    bool                `json:"delete"`
		With      json.RawMessage     `json:"with"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	ksref := algebra.NewKeyspaceRefFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), "")

	this.keyspace, err = datastore.GetKeyspace(ksref.Path().Parts()...)
	if err != nil {
		return err
	}

	var expr expression.Expression
	var terms, indexes expression.Expressions

	if len(_unmarshalled.Terms) > 0 {
		terms = make(expression.Expressions, len(_unmarshalled.Terms))

		for i, term := range _unmarshalled.Terms {
			expr, err = parser.Parse(term)
			if err != nil {
				return err
			}
			terms[i] = expr
		}
	}

	if len(_unmarshalled.Indexes) > 0 {
		indexes = make(expression.Expressions, len(_unmarshalled.Indexes))

		for i, index := range _unmarshalled.Indexes {
			expr, err = parser.Parse(index)
			if err != nil {
				return err
			}
			indexes[i] = expr
		}
	}

	var with value.Value
	if len(_unmarshalled.With) > 0 {
		with = value.NewValue([]byte(_unmarshalled.With))
	}

	if _unmarshalled.Delete {
		this.node = algebra.NewUpdateStatisticsDelete(ksref, terms)
	} else if _unmarshalled.IndexAll {
		this.node = algebra.NewUpdateStatisticsIndexAll(ksref, _unmarshalled.Using, with)
	} else if len(indexes) > 0 {
		this.node = algebra.NewUpdateStatisticsIndex(ksref, indexes, _unmarshalled.Using, with)
	} else {
		this.node = algebra.NewUpdateStatistics(ksref, terms, with)
	}
	return nil
}

func (this *UpdateStatistics) verify(prepared *Prepared) bool {
	var res bool

	this.keyspace, res = verifyKeyspace(this.keyspace, prepared)
	return res
}
