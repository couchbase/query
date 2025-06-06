//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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

type IndexCountScan2 struct {
	legacy
	index        datastore.CountIndex2
	indexer      datastore.Indexer
	term         *algebra.KeyspaceTerm
	spans        Spans2
	covers       expression.Covers
	filterCovers map[*expression.Cover]value.Value
}

func NewIndexCountScan2(index datastore.CountIndex2, term *algebra.KeyspaceTerm,
	spans Spans2, covers expression.Covers, filterCovers map[*expression.Cover]value.Value) *IndexCountScan2 {
	return &IndexCountScan2{
		index:        index,
		indexer:      index.Indexer(),
		term:         term,
		spans:        spans,
		covers:       covers,
		filterCovers: filterCovers,
	}
}

func (this *IndexCountScan2) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexCountScan2(this)
}

func (this *IndexCountScan2) New() Operator {
	return &IndexCountScan2{}
}

func (this *IndexCountScan2) Index() datastore.CountIndex2 {
	return this.index
}

func (this *IndexCountScan2) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *IndexCountScan2) Spans() Spans2 {
	return this.spans
}

func (this *IndexCountScan2) Covers() expression.Covers {
	return this.covers
}

func (this *IndexCountScan2) FilterCovers() map[*expression.Cover]value.Value {
	return this.filterCovers
}

func (this *IndexCountScan2) Covering() bool {
	return len(this.covers) > 0
}

func (this *IndexCountScan2) Limit() expression.Expression {
	return nil
}

func (this *IndexCountScan2) SetLimit(limit expression.Expression) {
}

func (this *IndexCountScan2) Offset() expression.Expression {
	return nil
}

func (this *IndexCountScan2) SetOffset(offset expression.Expression) {
}

func (this *IndexCountScan2) IsUnderNL() bool {
	return false
}

func (this *IndexCountScan2) CoverJoinSpanExpressions(coverer *expression.Coverer,
	implicitArrayKey *expression.All) error {
	return nil
}

func (this *IndexCountScan2) GroupAggs() *IndexGroupAggregates {
	return nil
}

func (this *IndexCountScan2) OrderTerms() IndexKeyOrders {
	return nil
}

func (this *IndexCountScan2) SetCovers(covers expression.Covers) {
	this.covers = covers
}

func (this *IndexCountScan2) GetIndex() datastore.Index {
	return nil
}

func (this *IndexCountScan2) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

func (this *IndexCountScan2) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IndexCountScan2) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "IndexCountScan2"}
	r["index"] = this.index.Name()
	r["index_id"] = this.index.Id()
	this.term.MarshalKeyspace(r)
	r["using"] = this.index.Type()
	setRangeIndexKey(this.spans, this.index)
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

func (this *IndexCountScan2) UnmarshalJSON(body []byte) error {
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
		Spans        Spans2                 `json:"spans"`
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

	countIndex2, ok := index.(datastore.CountIndex2)
	if !ok {
		return errors.NewPlanInternalError("Unable to find Count2() for index")
	}
	this.index = countIndex2

	planContext := this.PlanContext()
	if planContext != nil {
		planContext.addKeyspaceAlias(this.term.Alias())
	}

	return nil
}

func (this *IndexCountScan2) verify(prepared *Prepared) errors.Error {
	return verifyIndex(this.index, this.indexer, nil, prepared)
}

func (this *IndexCountScan2) Equals(i interface{}) bool {
	if cs, ok := i.(*IndexCountScan2); ok {
		return this.String() == cs.String()
	}
	return false
}
