//  Copyright (c) 2016 Couchbase, Inc.
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
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/value"
)

type IndexCountScan struct {
	legacy
	index        datastore.CountIndex
	indexer      datastore.Indexer
	term         *algebra.KeyspaceTerm
	spans        Spans
	covers       expression.Covers
	filterCovers map[*expression.Cover]value.Value
}

func NewIndexCountScan(index datastore.CountIndex, term *algebra.KeyspaceTerm, spans Spans,
	covers expression.Covers, filterCovers map[*expression.Cover]value.Value) *IndexCountScan {
	return &IndexCountScan{
		index:        index,
		indexer:      index.Indexer(),
		term:         term,
		spans:        spans,
		covers:       covers,
		filterCovers: filterCovers,
	}
}

func (this *IndexCountScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexCountScan(this)
}

func (this *IndexCountScan) New() Operator {
	return &IndexCountScan{}
}

func (this *IndexCountScan) Index() datastore.CountIndex {
	return this.index
}

func (this *IndexCountScan) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *IndexCountScan) Spans() Spans {
	return this.spans
}

func (this *IndexCountScan) Covers() expression.Covers {
	return this.covers
}

func (this *IndexCountScan) FilterCovers() map[*expression.Cover]value.Value {
	return this.filterCovers
}

func (this *IndexCountScan) Covering() bool {
	return len(this.covers) > 0
}

func (this *IndexCountScan) Limit() expression.Expression {
	return nil
}

func (this *IndexCountScan) SetLimit(limit expression.Expression) {
}

func (this *IndexCountScan) Offset() expression.Expression {
	return nil
}

func (this *IndexCountScan) SetOffset(offset expression.Expression) {
}

func (this *IndexCountScan) IsUnderNL() bool {
	return this.term.IsUnderNL()
}

func (this *IndexCountScan) CoverJoinSpanExpressions(coverer *expression.Coverer) error {
	return nil
}

func (this *IndexCountScan) GroupAggs() *IndexGroupAggregates {
	return nil
}

func (this *IndexCountScan) OrderTerms() IndexKeyOrders {
	return nil
}

func (this *IndexCountScan) SetCovers(covers expression.Covers) {
	this.covers = covers
}

func (this *IndexCountScan) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

func (this *IndexCountScan) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IndexCountScan) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "IndexCountScan"}
	r["index"] = this.index.Name()
	r["index_id"] = this.index.Id()
	this.term.MarshalKeyspace(r)
	r["using"] = this.index.Type()
	r["spans"] = this.spans

	if this.term.As() != "" {
		r["as"] = this.term.As()
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

func (this *IndexCountScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_            string                 `json:"#operator"`
		Index        string                 `json:"index"`
		IndexId      string                 `json:"index_id"`
		Namespace    string                 `json:"namespace"`
		Bucket       string                 `json:"bucket"`
		Scope        string                 `json:"scope"`
		Keyspace     string                 `json:"keyspace"`
		As           string                 `json:"as"`
		Using        datastore.IndexType    `json:"using"`
		Spans        Spans                  `json:"spans"`
		Covers       []string               `json:"covers"`
		FilterCovers map[string]interface{} `json:"filter_covers"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.term = algebra.NewKeyspaceTermFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), _unmarshalled.As, nil, nil)
	k, err := datastore.GetKeyspace(this.term.Path().Parts()...)
	if err != nil {
		return err
	}

	this.spans = _unmarshalled.Spans

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
	countIndex, ok := index.(datastore.CountIndex)
	if !ok {
		return errors.NewError(nil, "Unable to find Count() for index")
	}

	this.index = countIndex
	return nil
}

func (this *IndexCountScan) verify(prepared *Prepared) bool {
	return verifyIndex(this.index, this.indexer, nil, prepared)
}
