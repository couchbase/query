//  Copyright (c) 2017 Couchbase, Inc.
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
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type PrimaryScan3 struct {
	readonly
	index            datastore.PrimaryIndex3
	indexer          datastore.Indexer
	keyspace         datastore.Keyspace
	term             *algebra.KeyspaceTerm
	groupAggs        *IndexGroupAggregates
	projection       *IndexProjection
	orderTerms       IndexKeyOrders
	offset           expression.Expression
	limit            expression.Expression
	cost             float64
	cardinality      float64
	hasDeltaKeyspace bool
}

func NewPrimaryScan3(index datastore.PrimaryIndex3, keyspace datastore.Keyspace,
	term *algebra.KeyspaceTerm, offset, limit expression.Expression,
	projection *IndexProjection, orderTerms IndexKeyOrders,
	groupAggs *IndexGroupAggregates, cost, cardinality float64,
	hasDeltaKeyspace bool) *PrimaryScan3 {
	return &PrimaryScan3{
		index:            index,
		indexer:          index.Indexer(),
		keyspace:         keyspace,
		term:             term,
		groupAggs:        groupAggs,
		projection:       projection,
		orderTerms:       orderTerms,
		offset:           offset,
		limit:            limit,
		cost:             cost,
		cardinality:      cardinality,
		hasDeltaKeyspace: hasDeltaKeyspace,
	}
}

func (this *PrimaryScan3) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitPrimaryScan3(this)
}

func (this *PrimaryScan3) New() Operator {
	return &PrimaryScan3{}
}

func (this *PrimaryScan3) Index() datastore.PrimaryIndex3 {
	return this.index
}

func (this *PrimaryScan3) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *PrimaryScan3) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *PrimaryScan3) Projection() *IndexProjection {
	return this.projection
}

func (this *PrimaryScan3) OrderTerms() IndexKeyOrders {
	return this.orderTerms
}

func (this *PrimaryScan3) Offset() expression.Expression {
	return this.offset
}

func (this *PrimaryScan3) Limit() expression.Expression {
	return this.limit
}

func (this *PrimaryScan3) GroupAggs() *IndexGroupAggregates {
	return this.groupAggs
}

func (this *PrimaryScan3) SetGroupAggs(groupAggs *IndexGroupAggregates) {
	this.groupAggs = groupAggs
}

func (this *PrimaryScan3) SetLimit(limit expression.Expression) {
	this.limit = limit
}

func (this *PrimaryScan3) SetOffset(offset expression.Expression) {
	this.offset = offset
}

func (this *PrimaryScan3) Cost() float64 {
	return this.cost
}

func (this *PrimaryScan3) Cardinality() float64 {
	return this.cardinality
}

func (this *PrimaryScan3) HasDeltaKeyspace() bool {
	return this.hasDeltaKeyspace
}

func (this *PrimaryScan3) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *PrimaryScan3) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "PrimaryScan3"}
	r["index"] = this.index.Name()
	this.term.MarshalKeyspace(r)
	r["using"] = this.index.Type()

	if this.term.As() != "" {
		r["as"] = this.term.As()
	}

	if this.projection != nil {
		r["index_projection"] = this.projection
	}

	if len(this.orderTerms) > 0 {
		r["index_order"] = this.orderTerms
	}

	if this.offset != nil {
		r["offset"] = expression.NewStringer().Visit(this.offset)
	}

	if this.limit != nil {
		r["limit"] = expression.NewStringer().Visit(this.limit)
	}

	if this.groupAggs != nil {
		r["index_group_aggs"] = this.groupAggs
	}

	if this.cost > 0.0 {
		r["cost"] = this.cost
	}

	if this.cardinality > 0.0 {
		r["cardinality"] = this.cardinality
	}

	if this.hasDeltaKeyspace {
		r["has_delta_keyspace"] = this.hasDeltaKeyspace
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *PrimaryScan3) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_                string                `json:"#operator"`
		Index            string                `json:"index"`
		Namespace        string                `json:"namespace"`
		Bucket           string                `json:"bucket"`
		Scope            string                `json:"scope"`
		Keyspace         string                `json:"keyspace"`
		As               string                `json:"as"`
		Using            datastore.IndexType   `json:"using"`
		GroupAggs        *IndexGroupAggregates `json:"index_group_aggs"`
		Projection       *IndexProjection      `json:"index_projection"`
		OrderTerms       IndexKeyOrders        `json:"index_order"`
		Offset           string                `json:"offset"`
		Limit            string                `json:"limit"`
		Cost             float64               `json:"cost"`
		Cardinality      float64               `json:"cardinality"`
		HasDeltaKeyspace bool                  `json:"has_delta_keyspace"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.projection = _unmarshalled.Projection
	this.orderTerms = _unmarshalled.OrderTerms
	this.groupAggs = _unmarshalled.GroupAggs
	this.hasDeltaKeyspace = _unmarshalled.HasDeltaKeyspace

	if _unmarshalled.Offset != "" {
		this.offset, err = parser.Parse(_unmarshalled.Offset)
		if err != nil {
			return err
		}
	}

	if _unmarshalled.Limit != "" {
		this.limit, err = parser.Parse(_unmarshalled.Limit)
		if err != nil {
			return err
		}
	}

	this.cost = getCost(_unmarshalled.Cost)
	this.cardinality = getCardinality(_unmarshalled.Cardinality)

	this.term = algebra.NewKeyspaceTermFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), _unmarshalled.As, nil, nil)
	this.keyspace, err = datastore.GetKeyspace(this.term.Path().Parts()...)
	if err != nil {
		return err
	}

	this.indexer, err = this.keyspace.Indexer(_unmarshalled.Using)
	if err != nil {
		return err
	}

	index, err := this.indexer.IndexByName(_unmarshalled.Index)
	if err != nil {
		return err
	}

	if primary, ok := index.(datastore.PrimaryIndex3); ok {
		this.index = primary
		return nil
	}

	return fmt.Errorf("Unable to find Primary Index3 for %v", index.Name())
}

func (this *PrimaryScan3) verify(prepared *Prepared) bool {
	return verifyIndex(this.index, this.indexer, nil, prepared)
}
