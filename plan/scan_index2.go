//  Copyright 2014-Present Couchbase, Inc.
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
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/value"
)

type IndexScan2 struct {
	legacy
	index            datastore.Index2
	indexer          datastore.Indexer
	term             *algebra.KeyspaceTerm
	keyspace         datastore.Keyspace
	spans            Spans2
	reverse          bool
	distinct         bool
	ordered          bool
	projection       *IndexProjection
	offset           expression.Expression
	limit            expression.Expression
	covers           expression.Covers
	filterCovers     map[*expression.Cover]value.Value
	hasDeltaKeyspace bool
	nested_loop      bool
}

func NewIndexScan2(index datastore.Index2, term *algebra.KeyspaceTerm, spans Spans2,
	reverse, distinct, ordered bool, offset, limit expression.Expression,
	projection *IndexProjection, covers expression.Covers,
	filterCovers map[*expression.Cover]value.Value, hasDeltaKeyspace, nested_loop bool) *IndexScan2 {
	rv := &IndexScan2{
		index:            index,
		indexer:          index.Indexer(),
		term:             term,
		spans:            spans,
		reverse:          reverse,
		distinct:         distinct,
		ordered:          ordered,
		projection:       projection,
		offset:           offset,
		limit:            limit,
		covers:           covers,
		filterCovers:     filterCovers,
		hasDeltaKeyspace: hasDeltaKeyspace,
		nested_loop:      nested_loop,
	}

	rv.keyspace, _ = datastore.GetKeyspace(term.Path().Parts()...)
	return rv
}

func (this *IndexScan2) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexScan2(this)
}

func (this *IndexScan2) New() Operator {
	return &IndexScan2{}
}

func (this *IndexScan2) Index() datastore.Index2 {
	return this.index
}

func (this *IndexScan2) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *IndexScan2) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *IndexScan2) Spans() Spans2 {
	return this.spans
}

func (this *IndexScan2) SetSpans(spans Spans2) {
	this.spans = spans
}

func (this *IndexScan2) Distinct() bool {
	return this.distinct
}

func (this *IndexScan2) Reverse() bool {
	return this.reverse
}

func (this *IndexScan2) Ordered() bool {
	return this.ordered
}

func (this *IndexScan2) Projection() *IndexProjection {
	return this.projection
}

func (this *IndexScan2) Offset() expression.Expression {
	return this.offset
}

func (this *IndexScan2) Limit() expression.Expression {
	return this.limit
}

func (this *IndexScan2) SetLimit(limit expression.Expression) {
	this.limit = limit
}

func (this *IndexScan2) SetOffset(offset expression.Expression) {
	this.offset = offset
}

func (this *IndexScan2) IsUnderNL() bool {
	return this.nested_loop
}

func (this *IndexScan2) CoverJoinSpanExpressions(coverer *expression.Coverer,
	implicitArrayKey *expression.All) (err error) {
	err = anyRenameExpressions(implicitArrayKey, this.spans)
	if err == nil {
		err = coverJoinSpanExpressions(coverer, this.spans)
	}
	return err
}

func (this *IndexScan2) Covers() expression.Covers {
	return this.covers
}

func (this *IndexScan2) FilterCovers() map[*expression.Cover]value.Value {
	return this.filterCovers
}

func (this *IndexScan2) Covering() bool {
	return len(this.covers) > 0
}

func (this *IndexScan2) GroupAggs() *IndexGroupAggregates {
	return nil
}

func (this *IndexScan2) OrderTerms() IndexKeyOrders {
	return nil
}

func (this *IndexScan2) SetCovers(covers expression.Covers) {
	this.covers = covers
}

func (this *IndexScan2) HasDeltaKeyspace() bool {
	return this.hasDeltaKeyspace
}

func (this *IndexScan2) GetIndex() datastore.Index {
	return this.index
}

func (this *IndexScan2) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

func (this *IndexScan2) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IndexScan2) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "IndexScan2"}
	r["index"] = this.index.Name()
	r["index_id"] = this.index.Id()
	this.term.MarshalKeyspace(r)
	r["using"] = this.index.Type()

	setRangeIndexKey(this.spans, this.index)
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

	if this.ordered {
		r["ordered"] = this.ordered
	}

	if this.nested_loop {
		r["nested_loop"] = this.nested_loop
	}

	if this.projection != nil {
		r["index_projection"] = this.projection
	}

	if this.offset != nil {
		r["offset"] = expression.NewStringer().Visit(this.offset)
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

func (this *IndexScan2) UnmarshalJSON(body []byte) error {
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
		Ordered          bool                   `json:"ordered"`
		UnderNL          bool                   `json:"nested_loop"`
		Projection       *IndexProjection       `json:"index_projection"`
		Offset           string                 `json:"offset"`
		Limit            string                 `json:"limit"`
		Covers           []string               `json:"covers"`
		FilterCovers     map[string]interface{} `json:"filter_covers"`
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

	this.nested_loop = _unmarshalled.UnderNL
	this.spans = _unmarshalled.Spans
	this.reverse = _unmarshalled.Reverse
	this.distinct = _unmarshalled.Distinct
	this.ordered = _unmarshalled.Ordered
	this.projection = _unmarshalled.Projection
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

	index, err := this.indexer.IndexById(_unmarshalled.IndexId)
	if err != nil {
		return err
	}

	index2, ok := index.(datastore.Index2)
	if !ok {
		return fmt.Errorf("Unable to find Index2 for %v", index.Name())
	}
	this.index = index2

	return nil
}

func (this *IndexScan2) verify(prepared *Prepared) bool {
	return verifyIndex(this.index, this.indexer, verifyCoversAndSeqScan(this.covers, this.keyspace, this.indexer), prepared)
}

func (this *IndexScan2) Equals(i interface{}) bool {
	if is, ok := i.(*IndexScan2); ok {
		return this.String() == is.String()
	}
	return false
}
