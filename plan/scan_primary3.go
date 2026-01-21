//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"encoding/json"
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type PrimaryScan3 struct {
	readonly
	optEstimate
	index            datastore.PrimaryIndex3
	indexer          datastore.Indexer
	keyspace         datastore.Keyspace
	term             *algebra.KeyspaceTerm
	groupAggs        *IndexGroupAggregates
	projection       *IndexProjection
	orderTerms       IndexKeyOrders
	offset           expression.Expression
	limit            expression.Expression
	hasDeltaKeyspace bool
	skipNewKeys      bool
}

func NewPrimaryScan3(index datastore.PrimaryIndex3, keyspace datastore.Keyspace,
	term *algebra.KeyspaceTerm, offset, limit expression.Expression,
	projection *IndexProjection, orderTerms IndexKeyOrders,
	groupAggs *IndexGroupAggregates, cost, cardinality float64,
	size int64, frCost float64, hasDeltaKeyspace bool, skipNewKeys bool) *PrimaryScan3 {
	rv := &PrimaryScan3{
		index:            index,
		indexer:          index.Indexer(),
		keyspace:         keyspace,
		term:             term,
		groupAggs:        groupAggs,
		projection:       projection,
		orderTerms:       orderTerms,
		offset:           offset,
		limit:            limit,
		hasDeltaKeyspace: hasDeltaKeyspace,
		skipNewKeys:      skipNewKeys,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
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

func (this *PrimaryScan3) HasDeltaKeyspace() bool {
	return this.hasDeltaKeyspace
}

func (this *PrimaryScan3) GetIndex() datastore.Index {
	return this.index
}

func (this *PrimaryScan3) SkipNewKeys() bool {
	return this.skipNewKeys
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
		r["offset"] = this.offset.String()
	}

	if this.limit != nil {
		r["limit"] = this.limit.String()
	}

	if this.groupAggs != nil {
		r["index_group_aggs"] = this.groupAggs
	}

	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}

	if this.hasDeltaKeyspace {
		r["has_delta_keyspace"] = this.hasDeltaKeyspace
	}

	if this.skipNewKeys {
		r["skip_new_keys"] = this.skipNewKeys
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *PrimaryScan3) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_                string                 `json:"#operator"`
		Index            string                 `json:"index"`
		Namespace        string                 `json:"namespace"`
		Bucket           string                 `json:"bucket"`
		Scope            string                 `json:"scope"`
		Keyspace         string                 `json:"keyspace"`
		As               string                 `json:"as"`
		Using            datastore.IndexType    `json:"using"`
		GroupAggs        *IndexGroupAggregates  `json:"index_group_aggs"`
		Projection       *IndexProjection       `json:"index_projection"`
		OrderTerms       IndexKeyOrders         `json:"index_order"`
		Offset           string                 `json:"offset"`
		Limit            string                 `json:"limit"`
		OptEstimate      map[string]interface{} `json:"optimizer_estimates"`
		HasDeltaKeyspace bool                   `json:"has_delta_keyspace"`
		SkipNewKeys      bool                   `json:"skip_new_keys"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.projection = _unmarshalled.Projection
	this.orderTerms = _unmarshalled.OrderTerms
	this.groupAggs = _unmarshalled.GroupAggs
	this.hasDeltaKeyspace = _unmarshalled.HasDeltaKeyspace
	this.skipNewKeys = _unmarshalled.SkipNewKeys

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

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

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

	primary, ok := index.(datastore.PrimaryIndex3)
	if !ok {
		return fmt.Errorf("Unable to find Primary Index3 for %v", index.Name())
	}
	this.index = primary

	planContext := this.PlanContext()
	if planContext != nil {
		if this.limit != nil {
			_, err = planContext.Map(this.limit)
			if err != nil {
				return err
			}
		}
		if this.offset != nil {
			_, err = planContext.Map(this.offset)
			if err != nil {
				return err
			}
		}
		planContext.addKeyspaceAlias(this.term.Alias())
	}

	return nil
}

func (this *PrimaryScan3) verify(prepared *Prepared) errors.Error {
	return verifyIndex(this.index, this.indexer, verifyCoversAndSeqScan(nil, this.keyspace, this.indexer), prepared)
}

func (this *PrimaryScan3) keyspaceReferences(prepared *Prepared) {
	prepared.addKeyspaceReference(this.keyspace.QualifiedName())
}
