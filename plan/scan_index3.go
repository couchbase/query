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
	"github.com/couchbase/query/value"
)

const (
	ISCAN_IS_REVERSE_SCAN = 1 << iota
	ISCAN_IS_DISTINCT_SCAN
	ISCAN_HAS_DYNAMIC_IN_SPAN
)

type IndexScan3 struct {
	readonly
	index            datastore.Index3
	indexer          datastore.Indexer
	term             *algebra.KeyspaceTerm
	keyspace         datastore.Keyspace
	spans            Spans2
	flags            uint32
	groupAggs        *IndexGroupAggregates
	projection       *IndexProjection
	orderTerms       IndexKeyOrders
	offset           expression.Expression
	limit            expression.Expression
	covers           expression.Covers
	filterCovers     map[*expression.Cover]value.Value
	filter           expression.Expression
	cost             float64
	cardinality      float64
	hasDeltaKeyspace bool
}

func NewIndexScan3(index datastore.Index3, term *algebra.KeyspaceTerm, spans Spans2,
	reverse, distinct, dynamicIn bool, offset, limit expression.Expression,
	projection *IndexProjection, orderTerms IndexKeyOrders,
	groupAggs *IndexGroupAggregates, covers expression.Covers,
	filterCovers map[*expression.Cover]value.Value, filter expression.Expression,
	cost, cardinality float64, hasDeltaKeyspace bool) *IndexScan3 {
	flags := uint32(0)
	if reverse {
		flags |= ISCAN_IS_REVERSE_SCAN
	}
	if distinct {
		flags |= ISCAN_IS_DISTINCT_SCAN
	}
	if dynamicIn {
		flags |= ISCAN_HAS_DYNAMIC_IN_SPAN
	}
	rv := &IndexScan3{
		index:            index,
		indexer:          index.Indexer(),
		term:             term,
		spans:            spans,
		flags:            flags,
		groupAggs:        groupAggs,
		projection:       projection,
		orderTerms:       orderTerms,
		offset:           offset,
		limit:            limit,
		covers:           covers,
		filterCovers:     filterCovers,
		filter:           filter,
		cost:             cost,
		cardinality:      cardinality,
		hasDeltaKeyspace: hasDeltaKeyspace,
	}

	rv.keyspace, _ = datastore.GetKeyspace(term.Path().Parts()...)
	return rv
}

func (this *IndexScan3) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexScan3(this)
}

func (this *IndexScan3) New() Operator {
	return &IndexScan3{}
}

func (this *IndexScan3) Index() datastore.Index3 {
	return this.index
}

func (this *IndexScan3) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *IndexScan3) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *IndexScan3) Spans() Spans2 {
	return this.spans
}

func (this *IndexScan3) SetSpans(spans Spans2) {
	this.spans = spans
}

func (this *IndexScan3) Distinct() bool {
	return (this.flags & ISCAN_IS_DISTINCT_SCAN) != 0
}

func (this *IndexScan3) Reverse() bool {
	return (this.flags & ISCAN_IS_REVERSE_SCAN) != 0
}

func (this *IndexScan3) HasDynamicInSpan() bool {
	return (this.flags & ISCAN_HAS_DYNAMIC_IN_SPAN) != 0
}

func (this *IndexScan3) Projection() *IndexProjection {
	return this.projection
}

func (this *IndexScan3) OrderTerms() IndexKeyOrders {
	return this.orderTerms
}

func (this *IndexScan3) Offset() expression.Expression {
	return this.offset
}

func (this *IndexScan3) Limit() expression.Expression {
	return this.limit
}

func (this *IndexScan3) GroupAggs() *IndexGroupAggregates {
	return this.groupAggs
}

func (this *IndexScan3) SetGroupAggs(groupAggs *IndexGroupAggregates) {
	this.groupAggs = groupAggs
}

func (this *IndexScan3) SetLimit(limit expression.Expression) {
	this.limit = limit
}

func (this *IndexScan3) SetOffset(offset expression.Expression) {
	this.offset = offset
}

func (this *IndexScan3) CoverJoinSpanExpressions(coverer *expression.Coverer) error {
	var err error
	for _, span := range this.spans {
		for i, seek := range span.Seek {
			if seek != nil {
				span.Seek[i], err = coverer.Map(seek)
				if err != nil {
					return err
				}
			}
		}
		for _, srange := range span.Ranges {
			if srange.Low != nil {
				srange.Low, err = coverer.Map(srange.Low)
				if err != nil {
					return err
				}
			}
			if srange.High != nil {
				srange.High, err = coverer.Map(srange.High)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (this *IndexScan3) Covers() expression.Covers {
	return this.covers
}

func (this *IndexScan3) SetCovers(covers expression.Covers) {
	this.covers = covers
}

func (this *IndexScan3) FilterCovers() map[*expression.Cover]value.Value {
	return this.filterCovers
}

func (this *IndexScan3) Covering() bool {
	return len(this.covers) > 0
}

func (this *IndexScan3) Filter() expression.Expression {
	return this.filter
}

func (this *IndexScan3) IsUnderNL() bool {
	return this.term.IsUnderNL()
}

func (this *IndexScan3) Cost() float64 {
	return this.cost
}

func (this *IndexScan3) Cardinality() float64 {
	return this.cardinality
}

func (this *IndexScan3) HasDeltaKeyspace() bool {
	return this.hasDeltaKeyspace
}

func (this *IndexScan3) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

func (this *IndexScan3) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IndexScan3) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "IndexScan3"}
	r["index"] = this.index.Name()
	r["index_id"] = this.index.Id()
	this.term.MarshalKeyspace(r)
	r["using"] = this.index.Type()
	r["spans"] = this.spans

	if this.term.As() != "" {
		r["as"] = this.term.As()
	}

	if this.Reverse() {
		r["reverse"] = true
	}

	if this.Distinct() {
		r["distinct"] = true
	}

	if this.HasDynamicInSpan() {
		r["has_dynamic_in"] = true
	}

	if this.term.IsUnderNL() {
		r["nested_loop"] = this.term.IsUnderNL()
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

	if len(this.covers) > 0 {
		r["covers"] = this.covers
	}

	if len(this.filterCovers) > 0 {
		fc := make(map[string]value.Value, len(this.filterCovers))
		for c, v := range this.filterCovers {
			fc[c.String()] = v
		}

		r["filter_covers"] = fc
	}

	if this.filter != nil {
		r["filter"] = expression.NewStringer().Visit(this.filter)
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

func (this *IndexScan3) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_                string                 `json:"#operator"`
		Index            string                 `json:"index"`
		IndexId          string                 `json:"index_id"`
		Namespace        string                 `json:"namespace"`
		Bucket           string                 `json:"bucket"`
		Scope            string                 `json:"scope"`
		Keyspace         string                 `json:"keyspace"`
		As               string                 `json:"as"`
		Using            datastore.IndexType    `json:"using"`
		Spans            Spans2                 `json:"spans"`
		Reverse          bool                   `json:"reverse"`
		Distinct         bool                   `json:"distinct"`
		DynamicIn        bool                   `json:"has_dynamic_in"`
		UnderNL          bool                   `json:"nested_loop"`
		GroupAggs        *IndexGroupAggregates  `json:"index_group_aggs"`
		Projection       *IndexProjection       `json:"index_projection"`
		OrderTerms       IndexKeyOrders         `json:"index_order"`
		Offset           string                 `json:"offset"`
		Limit            string                 `json:"limit"`
		Covers           []string               `json:"covers"`
		FilterCovers     map[string]interface{} `json:"filter_covers"`
		Filter           string                 `json:"filter"`
		Cost             float64                `json:"cost"`
		Cardinality      float64                `json:"cardinality"`
		HasDeltaKeyspace bool                   `json:"has_delta_keyspace"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.term = algebra.NewKeyspaceTermFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), _unmarshalled.As, nil, nil)
	this.keyspace, err = datastore.GetKeyspace(this.term.Path().Parts()...)
	if err != nil {
		return err
	}

	this.spans = _unmarshalled.Spans
	flags := uint32(0)
	if _unmarshalled.Reverse {
		flags |= ISCAN_IS_REVERSE_SCAN
	}
	if _unmarshalled.Distinct {
		flags |= ISCAN_IS_DISTINCT_SCAN
	}
	if _unmarshalled.DynamicIn {
		flags |= ISCAN_HAS_DYNAMIC_IN_SPAN
	}
	this.flags = flags
	this.groupAggs = _unmarshalled.GroupAggs
	this.projection = _unmarshalled.Projection
	this.orderTerms = _unmarshalled.OrderTerms
	this.hasDeltaKeyspace = _unmarshalled.HasDeltaKeyspace

	if _unmarshalled.UnderNL {
		this.term.SetUnderNL()
	}

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

	if len(_unmarshalled.Covers) > 0 {
		this.covers = make(expression.Covers, len(_unmarshalled.Covers))
		for i, c := range _unmarshalled.Covers {
			expr, err := parser.Parse(c)
			if err != nil {
				return err
			}

			this.covers[i] = expression.NewCover(expr)
		}
	}

	if len(_unmarshalled.FilterCovers) > 0 {
		this.filterCovers = make(map[*expression.Cover]value.Value, len(_unmarshalled.FilterCovers))
		for k, v := range _unmarshalled.FilterCovers {
			expr, err := parser.Parse(k)
			if err != nil {
				return err
			}

			c := expression.NewCover(expr)
			this.filterCovers[c] = value.NewValue(v)
		}
	}

	if _unmarshalled.Filter != "" {
		this.filter, err = parser.Parse(_unmarshalled.Filter)
		if err != nil {
			return err
		}
	}

	this.cost = getCost(_unmarshalled.Cost)
	this.cardinality = getCardinality(_unmarshalled.Cardinality)

	this.indexer, err = this.keyspace.Indexer(_unmarshalled.Using)
	if err != nil {
		return err
	}

	index, err := this.indexer.IndexById(_unmarshalled.IndexId)
	if err != nil {
		return err
	}

	if index3, ok := index.(datastore.Index3); ok {
		this.index = index3
		return nil
	}
	return fmt.Errorf("Unable to find Index for %v", index.Name())
}

func (this *IndexScan3) verify(prepared *Prepared) bool {
	return verifyIndex(this.index, this.indexer, verifyCovers(this.covers, this.keyspace), prepared)
}
