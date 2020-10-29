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

type IndexScan struct {
	legacy
	index            datastore.Index
	indexer          datastore.Indexer
	term             *algebra.KeyspaceTerm
	keyspace         datastore.Keyspace
	spans            Spans
	distinct         bool
	limit            expression.Expression
	covers           expression.Covers
	filterCovers     map[*expression.Cover]value.Value
	hasDeltaKeyspace bool
}

func NewIndexScan(index datastore.Index, term *algebra.KeyspaceTerm, spans Spans,
	distinct bool, limit expression.Expression, covers expression.Covers,
	filterCovers map[*expression.Cover]value.Value, hasDeltaKeyspace bool) *IndexScan {
	rv := &IndexScan{
		index:            index,
		indexer:          index.Indexer(),
		term:             term,
		spans:            spans,
		distinct:         distinct,
		limit:            limit,
		covers:           covers,
		filterCovers:     filterCovers,
		hasDeltaKeyspace: hasDeltaKeyspace,
	}

	rv.keyspace, _ = datastore.GetKeyspace(term.Path().Parts()...)
	return rv
}

func (this *IndexScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexScan(this)
}

func (this *IndexScan) New() Operator {
	return &IndexScan{}
}

func (this *IndexScan) Index() datastore.Index {
	return this.index
}

func (this *IndexScan) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *IndexScan) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *IndexScan) Spans() Spans {
	return this.spans
}

func (this *IndexScan) SetSpans(spans Spans) {
	this.spans = spans
}

func (this *IndexScan) Distinct() bool {
	return this.distinct
}

func (this *IndexScan) Limit() expression.Expression {
	return this.limit
}

func (this *IndexScan) SetLimit(limit expression.Expression) {
	this.limit = limit
}

func (this *IndexScan) Offset() expression.Expression {
	return nil
}

func (this *IndexScan) SetOffset(offset expression.Expression) {
}

func (this *IndexScan) IsUnderNL() bool {
	return this.term.IsUnderNL()
}

func (this *IndexScan) CoverJoinSpanExpressions(coverer *expression.Coverer) error {
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
		for i, low := range span.Range.Low {
			if low != nil {
				span.Range.Low[i], err = coverer.Map(low)
				if err != nil {
					return err
				}
			}
		}
		for i, high := range span.Range.High {
			if high != nil {
				span.Range.High[i], err = coverer.Map(high)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (this *IndexScan) Covers() expression.Covers {
	return this.covers
}

func (this *IndexScan) FilterCovers() map[*expression.Cover]value.Value {
	return this.filterCovers
}

func (this *IndexScan) Covering() bool {
	return len(this.covers) > 0
}

func (this *IndexScan) GroupAggs() *IndexGroupAggregates {
	return nil
}

func (this *IndexScan) OrderTerms() IndexKeyOrders {
	return nil
}

func (this *IndexScan) SetCovers(covers expression.Covers) {
	this.covers = covers
}

func (this *IndexScan) HasDeltaKeyspace() bool {
	return this.hasDeltaKeyspace
}

func (this *IndexScan) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

func (this *IndexScan) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IndexScan) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "IndexScan"}
	r["index"] = this.index.Name()
	r["index_id"] = this.index.Id()
	this.term.MarshalKeyspace(r)
	r["using"] = this.index.Type()
	r["spans"] = this.spans

	if this.term.As() != "" {
		r["as"] = this.term.As()
	}

	if this.distinct {
		r["distinct"] = this.distinct
	}

	if this.term.IsUnderNL() {
		r["nested_loop"] = this.term.IsUnderNL()
	}

	if this.limit != nil {
		r["limit"] = expression.NewStringer().Visit(this.limit)
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

	if this.hasDeltaKeyspace {
		r["has_delta_keyspace"] = this.hasDeltaKeyspace
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *IndexScan) UnmarshalJSON(body []byte) error {
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
		Spans            Spans                  `json:"spans"`
		Distinct         bool                   `json:"distinct"`
		UnderNL          bool                   `json:"nested_loop"`
		Limit            string                 `json:"limit"`
		Covers           []string               `json:"covers"`
		FilterCovers     map[string]interface{} `json:"filter_covers"`
		HasDeltaKeyspace bool                   `json:"has_delta_keyspace"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.hasDeltaKeyspace = _unmarshalled.HasDeltaKeyspace

	this.term = algebra.NewKeyspaceTermFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), _unmarshalled.As, nil, nil)
	this.keyspace, err = datastore.GetKeyspace(this.term.Path().Parts()...)
	if err != nil {
		return err
	}

	if _unmarshalled.UnderNL {
		this.term.SetUnderNL()
	}
	this.spans = _unmarshalled.Spans
	this.distinct = _unmarshalled.Distinct

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

	this.indexer, err = this.keyspace.Indexer(_unmarshalled.Using)
	if err != nil {
		return err
	}

	this.index, err = this.indexer.IndexById(_unmarshalled.IndexId)
	return err
}

func (this *IndexScan) verify(prepared *Prepared) bool {
	return verifyIndex(this.index, this.indexer, verifyCovers(this.covers, this.keyspace), prepared)
}
