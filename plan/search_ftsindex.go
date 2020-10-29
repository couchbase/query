//  Copyright (c) 2019 Couchbase, Inc.
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

type IndexFtsSearch struct {
	readonly
	index            datastore.Index
	indexer          datastore.Indexer
	term             *algebra.KeyspaceTerm
	keyspace         datastore.Keyspace
	searchInfo       *FTSSearchInfo
	covers           expression.Covers
	filterCovers     map[*expression.Cover]value.Value
	hasDeltaKeyspace bool
}

func NewIndexFtsSearch(index datastore.Index, term *algebra.KeyspaceTerm,
	searchInfo *FTSSearchInfo, covers expression.Covers,
	filterCovers map[*expression.Cover]value.Value, hasDeltaKeyspace bool) *IndexFtsSearch {

	rv := &IndexFtsSearch{
		index:            index,
		indexer:          index.Indexer(),
		term:             term,
		searchInfo:       searchInfo,
		covers:           covers,
		filterCovers:     filterCovers,
		hasDeltaKeyspace: hasDeltaKeyspace,
	}

	rv.keyspace, _ = datastore.GetKeyspace(term.Path().Parts()...)
	return rv
}

func (this *IndexFtsSearch) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexFtsSearch(this)
}

func (this *IndexFtsSearch) New() Operator {
	return &IndexFtsSearch{}
}

func (this *IndexFtsSearch) Index() datastore.Index {
	return this.index
}

func (this *IndexFtsSearch) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *IndexFtsSearch) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *IndexFtsSearch) SearchInfo() *FTSSearchInfo {
	return this.searchInfo
}

func (this *IndexFtsSearch) OrderTerms() IndexKeyOrders {
	return nil
}

func (this *IndexFtsSearch) Offset() expression.Expression {
	if this.searchInfo != nil {
		return this.searchInfo.offset
	}
	return nil
}

func (this *IndexFtsSearch) Limit() expression.Expression {
	if this.searchInfo != nil {
		return this.searchInfo.limit
	}
	return nil
}

func (this *IndexFtsSearch) GroupAggs() *IndexGroupAggregates {
	return nil
}

func (this *IndexFtsSearch) SetLimit(limit expression.Expression) {
	if this.searchInfo != nil {
		this.searchInfo.limit = limit
	}
}

func (this *IndexFtsSearch) SetOffset(offset expression.Expression) {
	if this.searchInfo != nil {
		this.searchInfo.offset = offset
	}
}

func (this *IndexFtsSearch) IsUnderNL() bool {
	return this.term.IsUnderNL()
}

func (this *IndexFtsSearch) CoverJoinSpanExpressions(coverer *expression.Coverer) error {
	return nil
}

func (this *IndexFtsSearch) Covers() expression.Covers {
	return this.covers
}

func (this *IndexFtsSearch) SetCovers(covers expression.Covers) {
	this.covers = covers
}

func (this *IndexFtsSearch) FilterCovers() map[*expression.Cover]value.Value {
	return this.filterCovers
}

func (this *IndexFtsSearch) Covering() bool {
	return len(this.covers) != 0
}

func (this *IndexFtsSearch) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

func (this *IndexFtsSearch) Cost() float64 {
	return PLAN_COST_NOT_AVAIL
}

func (this *IndexFtsSearch) Cardinality() float64 {
	return PLAN_CARD_NOT_AVAIL
}

func (this *IndexFtsSearch) HasDeltaKeyspace() bool {
	return this.hasDeltaKeyspace
}

func (this *IndexFtsSearch) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IndexFtsSearch) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "IndexFtsSearch"}
	r["index"] = this.index.Name()
	r["index_id"] = this.index.Id()
	this.term.MarshalKeyspace(r)
	r["using"] = this.index.Type()

	if this.term.As() != "" {
		r["as"] = this.term.As()
	}

	if this.term.IsUnderNL() {
		r["nested_loop"] = this.term.IsUnderNL()
	}

	if this.searchInfo != nil {
		r["search_info"] = this.searchInfo
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

func (this *IndexFtsSearch) UnmarshalJSON(body []byte) error {
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
		UnderNL          bool                   `json:"nested_loop"`
		SearchInfo       *FTSSearchInfo         `json:"search_info"`
		Covers           []string               `json:"covers"`
		FilterCovers     map[string]interface{} `json:"filter_covers"`
		HasDeltaKeyspace bool                   `json:"has_delta_keyspace"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.searchInfo = _unmarshalled.SearchInfo
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

	this.indexer, err = this.keyspace.Indexer(_unmarshalled.Using)
	if err != nil {
		return err
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

	this.index, err = this.indexer.IndexById(_unmarshalled.IndexId)
	if _, ok := this.index.(datastore.FTSIndex); !ok {
		return fmt.Errorf("Unable to find Index for %v", this.index.Name())
	}

	return nil
}

func (this *IndexFtsSearch) verify(prepared *Prepared) bool {
	return verifyIndex(this.index, this.indexer, verifyCovers(this.covers, this.keyspace), prepared)
}

type FTSSearchInfo struct {
	field   expression.Expression
	query   expression.Expression
	options expression.Expression
	offset  expression.Expression
	limit   expression.Expression
	order   []string
	outName string
}

func NewFTSSearchInfo(field, query, options, offset, limit expression.Expression,
	order []string, outName string) *FTSSearchInfo {

	return &FTSSearchInfo{
		field:   field,
		query:   query,
		options: options,
		offset:  offset,
		limit:   limit,
		order:   order,
		outName: outName,
	}
}

func (this *FTSSearchInfo) Copy() *FTSSearchInfo {
	return &FTSSearchInfo{
		field:   expression.Copy(this.field),
		query:   expression.Copy(this.query),
		options: expression.Copy(this.options),
		offset:  expression.Copy(this.offset),
		limit:   expression.Copy(this.limit),
		order:   this.order,
		outName: this.outName,
	}
}

func (this *FTSSearchInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *FTSSearchInfo) FieldName() expression.Expression {
	return this.field
}

func (this *FTSSearchInfo) Query() expression.Expression {
	return this.query
}

func (this *FTSSearchInfo) Options() expression.Expression {
	return this.options
}

func (this *FTSSearchInfo) Offset() expression.Expression {
	return this.offset
}

func (this *FTSSearchInfo) Limit() expression.Expression {
	return this.limit
}

func (this *FTSSearchInfo) Order() []string {
	return this.order
}

func (this *FTSSearchInfo) OutName() string {
	return this.outName
}

func (this *FTSSearchInfo) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := make(map[string]interface{}, 7)

	if this.field != nil {
		r["field"] = expression.NewStringer().Visit(this.field)
	}

	if this.query != nil {
		r["query"] = expression.NewStringer().Visit(this.query)
	}

	if this.options != nil {
		r["options"] = expression.NewStringer().Visit(this.options)
	}

	if this.offset != nil {
		r["offset"] = expression.NewStringer().Visit(this.offset)
	}

	if this.limit != nil {
		r["limit"] = expression.NewStringer().Visit(this.limit)
	}
	if len(this.order) > 0 {
		r["order"] = this.order
	}

	r["outname"] = this.outName

	if f != nil {
		f(r)
	}
	return r
}

func (this *FTSSearchInfo) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Field   string   `json:"field"`
		Query   string   `json:"query"`
		Options string   `json:"options"`
		Offset  string   `json:"offset"`
		Limit   string   `json:"limit"`
		Order   []string `json:"order"`
		OutName string   `json:"outname"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.Field != "" {
		this.field, err = parser.Parse(_unmarshalled.Field)
		if err != nil {
			return err
		}
	}

	if _unmarshalled.Query != "" {
		this.query, err = parser.Parse(_unmarshalled.Query)
		if err != nil {
			return err
		}
	}

	if _unmarshalled.Options != "" {
		this.options, err = parser.Parse(_unmarshalled.Options)
		if err != nil {
			return err
		}
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

	this.order = _unmarshalled.Order
	this.outName = _unmarshalled.OutName

	return nil
}

func (this *FTSSearchInfo) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}
