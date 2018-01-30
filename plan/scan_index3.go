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

type IndexScan3 struct {
	readonly
	index        datastore.Index3
	indexer      datastore.Indexer
	term         *algebra.KeyspaceTerm
	spans        Spans2
	reverse      bool
	distinct     bool
	groupAggs    *IndexGroupAggregates
	projection   *IndexProjection
	orderTerms   IndexKeyOrders
	offset       expression.Expression
	limit        expression.Expression
	covers       expression.Covers
	filterCovers map[*expression.Cover]value.Value
}

func NewIndexScan3(index datastore.Index3, term *algebra.KeyspaceTerm, spans Spans2,
	reverse, distinct bool, offset, limit expression.Expression,
	projection *IndexProjection, orderTerms IndexKeyOrders,
	groupAggs *IndexGroupAggregates, covers expression.Covers,
	filterCovers map[*expression.Cover]value.Value) *IndexScan3 {
	return &IndexScan3{
		index:        index,
		indexer:      getIndexer(term.Namespace(), term.Keyspace(), index.Type()),
		term:         term,
		spans:        spans,
		reverse:      reverse,
		distinct:     distinct,
		groupAggs:    groupAggs,
		projection:   projection,
		orderTerms:   orderTerms,
		offset:       offset,
		limit:        limit,
		covers:       covers,
		filterCovers: filterCovers,
	}
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

func (this *IndexScan3) Spans() Spans2 {
	return this.spans
}

func (this *IndexScan3) SetSpans(spans Spans2) {
	this.spans = spans
}

func (this *IndexScan3) Distinct() bool {
	return this.distinct
}

func (this *IndexScan3) Reverse() bool {
	return this.reverse
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
	r["namespace"] = this.term.Namespace()
	r["keyspace"] = this.term.Keyspace()
	r["using"] = this.index.Type()
	r["spans"] = this.spans

	if this.term.As() != "" {
		r["as"] = this.term.As()
	}

	if this.reverse {
		r["reverse"] = this.reverse
	}

	if this.distinct {
		r["distinct"] = this.distinct
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

	if f != nil {
		f(r)
	}
	return r
}

func (this *IndexScan3) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_            string                 `json:"#operator"`
		Index        string                 `json:"index"`
		IndexId      string                 `json:"index_id"`
		Namespace    string                 `json:"namespace"`
		Keyspace     string                 `json:"keyspace"`
		As           string                 `json:"as"`
		Using        datastore.IndexType    `json:"using"`
		Spans        Spans2                 `json:"spans"`
		Reverse      bool                   `json:"reverse"`
		Distinct     bool                   `json:"distinct"`
		UnderNL      bool                   `json:"nested_loop"`
		GroupAggs    *IndexGroupAggregates  `json:"index_group_aggs"`
		Projection   *IndexProjection       `json:"index_projection"`
		OrderTerms   IndexKeyOrders         `json:"index_order"`
		Offset       string                 `json:"offset"`
		Limit        string                 `json:"limit"`
		Covers       []string               `json:"covers"`
		FilterCovers map[string]interface{} `json:"filter_covers"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	k, err := datastore.GetKeyspace(_unmarshalled.Namespace, _unmarshalled.Keyspace)
	if err != nil {
		return err
	}

	this.term = algebra.NewKeyspaceTerm(_unmarshalled.Namespace, _unmarshalled.Keyspace, _unmarshalled.As, nil, nil)

	this.spans = _unmarshalled.Spans
	this.reverse = _unmarshalled.Reverse
	this.distinct = _unmarshalled.Distinct
	this.groupAggs = _unmarshalled.GroupAggs
	this.projection = _unmarshalled.Projection
	this.orderTerms = _unmarshalled.OrderTerms

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

	this.indexer, err = k.Indexer(_unmarshalled.Using)
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
	return verifyIndex(this.index, this.indexer, prepared)
}
