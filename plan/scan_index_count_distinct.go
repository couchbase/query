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

type IndexCountDistinctScan2 struct {
	legacy
	index        datastore.CountIndex2
	indexer      datastore.Indexer
	term         *algebra.KeyspaceTerm
	spans        Spans2
	covers       expression.Covers
	filterCovers map[*expression.Cover]value.Value
}

func NewIndexCountDistinctScan2(index datastore.CountIndex2, term *algebra.KeyspaceTerm,
	spans Spans2, covers expression.Covers, filterCovers map[*expression.Cover]value.Value) *IndexCountDistinctScan2 {
	return &IndexCountDistinctScan2{
		index:        index,
		indexer:      index.Indexer(),
		term:         term,
		spans:        spans,
		covers:       covers,
		filterCovers: filterCovers,
	}
}

func (this *IndexCountDistinctScan2) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexCountDistinctScan2(this)
}

func (this *IndexCountDistinctScan2) New() Operator {
	return &IndexCountDistinctScan2{}
}

func (this *IndexCountDistinctScan2) Index() datastore.CountIndex2 {
	return this.index
}

func (this *IndexCountDistinctScan2) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *IndexCountDistinctScan2) Spans() Spans2 {
	return this.spans
}

func (this *IndexCountDistinctScan2) Covers() expression.Covers {
	return this.covers
}

func (this *IndexCountDistinctScan2) FilterCovers() map[*expression.Cover]value.Value {
	return this.filterCovers
}

func (this *IndexCountDistinctScan2) Covering() bool {
	return len(this.covers) > 0
}

func (this *IndexCountDistinctScan2) Limit() expression.Expression {
	return nil
}

func (this *IndexCountDistinctScan2) SetLimit(limit expression.Expression) {
}

func (this *IndexCountDistinctScan2) Offset() expression.Expression {
	return nil
}

func (this *IndexCountDistinctScan2) SetOffset(offset expression.Expression) {
}

func (this *IndexCountDistinctScan2) IsUnderNL() bool {
	return false
}

func (this *IndexCountDistinctScan2) CoverJoinSpanExpressions(coverer *expression.Coverer,
	implicitArrayKey *expression.All) error {
	return nil
}

func (this *IndexCountDistinctScan2) GroupAggs() *IndexGroupAggregates {
	return nil
}

func (this *IndexCountDistinctScan2) OrderTerms() IndexKeyOrders {
	return nil
}

func (this *IndexCountDistinctScan2) SetCovers(covers expression.Covers) {
	this.covers = covers
}

func (this *IndexCountDistinctScan2) GetIndex() datastore.Index {
	return nil
}

func (this *IndexCountDistinctScan2) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

func (this *IndexCountDistinctScan2) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IndexCountDistinctScan2) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "IndexCountDistinctScan2"}
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

func (this *IndexCountDistinctScan2) UnmarshalJSON(body []byte) error {
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
	if !ok || !countIndex2.CanCountDistinct() {
		return errors.NewPlanInternalError("Unable to find CountDistinct() for index")
	}
	this.index = countIndex2

	planContext := this.PlanContext()
	if planContext != nil {
		planContext.addKeyspaceAlias(this.term.Alias())
	}

	return nil
}

func (this *IndexCountDistinctScan2) verify(prepared *Prepared) bool {
	return verifyIndex(this.index, this.indexer, nil, prepared)
}

func (this *IndexCountDistinctScan2) Equals(i interface{}) bool {
	if ds, ok := i.(*IndexCountDistinctScan2); ok {
		return this.String() == ds.String()
	}
	return false
}
