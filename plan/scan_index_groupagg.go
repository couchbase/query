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

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type IndexGroupKeys []*IndexGroupKey

type IndexGroupKey struct {
	EntryKeyId int
	KeyPos     int
	Depends    []int
	Expr       expression.Expression
}

func NewIndexGroupKey(id, pos int, expr expression.Expression, depends []int) *IndexGroupKey {
	return &IndexGroupKey{
		EntryKeyId: id,
		KeyPos:     pos,
		Expr:       expr,
		Depends:    depends,
	}
}

func (this *IndexGroupKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IndexGroupKey) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := make(map[string]interface{}, 4)

	r["id"] = this.EntryKeyId
	r["keypos"] = this.KeyPos
	r["expr"] = expression.NewStringer().Visit(this.Expr)
	if len(this.Depends) > 0 {
		r["depends"] = this.Depends
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *IndexGroupKey) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		EntryKeyId int    `json:"id"`
		KeyPos     int    `json:"keypos"`
		Expr       string `json:"expr"`
		Depends    []int  `json:"depends"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.EntryKeyId = _unmarshalled.EntryKeyId

	if _unmarshalled.Expr != "" {
		this.Expr, err = parser.Parse(_unmarshalled.Expr)
	}
	this.KeyPos = _unmarshalled.KeyPos
	this.Depends = _unmarshalled.Depends

	return nil
}

func (this *IndexGroupKey) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

type IndexAggregates []*IndexAggregate

type IndexAggregate struct {
	Operation  datastore.AggregateType
	EntryKeyId int
	KeyPos     int
	Depends    []int
	Expr       expression.Expression
	Distinct   bool
}

func NewIndexAggregate(op datastore.AggregateType, id, pos int, expr expression.Expression, distinct bool,
	depends []int) *IndexAggregate {

	return &IndexAggregate{
		Operation:  op,
		EntryKeyId: id,
		KeyPos:     pos,
		Expr:       expr,
		Distinct:   distinct,
		Depends:    depends,
	}
}

func (this *IndexAggregate) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IndexAggregate) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := make(map[string]interface{}, 4)

	r["aggregate"] = this.Operation
	r["id"] = this.EntryKeyId
	if this.Distinct {
		r["distinct"] = this.Distinct
	}

	r["keypos"] = this.KeyPos
	r["expr"] = expression.NewStringer().Visit(this.Expr)
	if len(this.Depends) > 0 {
		r["depends"] = this.Depends
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *IndexAggregate) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Operation  datastore.AggregateType `json:"aggregate"`
		EntryKeyId int                     `json:"id"`
		KeyPos     int                     `json:"keypos"`
		Expr       string                  `json:"expr"`
		Distinct   bool                    `json:"distinct"`
		Depends    []int                   `json:"depends"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.Operation = _unmarshalled.Operation
	this.EntryKeyId = _unmarshalled.EntryKeyId
	this.Distinct = _unmarshalled.Distinct
	this.Depends = _unmarshalled.Depends

	if _unmarshalled.Expr != "" {
		this.Expr, err = parser.Parse(_unmarshalled.Expr)
	}
	this.KeyPos = _unmarshalled.KeyPos

	return nil
}

func (this *IndexAggregate) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

type IndexGroupAggregates struct {
	Name               string
	Group              IndexGroupKeys
	Aggregates         IndexAggregates
	DependsOnIndexKeys []int
	Partial            bool
	DistinctDocid      bool
}

func NewIndexGroupAggregates(name string, group IndexGroupKeys, aggs IndexAggregates,
	dependsOnIndexKeys []int, partial, distinctDocid bool) *IndexGroupAggregates {
	return &IndexGroupAggregates{
		Name:               name,
		Group:              group,
		Aggregates:         aggs,
		DependsOnIndexKeys: dependsOnIndexKeys,
		Partial:            partial,
		DistinctDocid:      distinctDocid,
	}
}

func (this *IndexGroupAggregates) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IndexGroupAggregates) MarshalBase(f func(map[string]interface{})) map[string]interface{} {

	r := make(map[string]interface{}, 4)

	if this.Name != "" {
		r["name"] = this.Name
	}
	if len(this.Group) > 0 {
		r["group"] = this.Group
	}

	if len(this.Aggregates) > 0 {
		r["aggregates"] = this.Aggregates
	}

	if len(this.DependsOnIndexKeys) > 0 {
		r["depends"] = this.DependsOnIndexKeys
	}

	if this.Partial {
		r["partial"] = this.Partial
	}

	if this.DistinctDocid {
		r["distinctdocid"] = this.DistinctDocid
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *IndexGroupAggregates) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Name               string            `json:"name"`
		Group              []json.RawMessage `json:"group"`
		Aggregates         []json.RawMessage `json:"aggregates"`
		DependsOnIndexKeys []int             `json:"depends"`
		Partial            bool              `json:"partial"`
		DistinctDocid      bool              `json:"distinctdocid"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.Name = _unmarshalled.Name
	this.Partial = _unmarshalled.Partial
	this.DistinctDocid = _unmarshalled.DistinctDocid

	if len(_unmarshalled.Group) > 0 {
		this.Group = make(IndexGroupKeys, 0, len(_unmarshalled.Group))
		for _, s := range _unmarshalled.Group {
			r := &IndexGroupKey{}
			err = r.UnmarshalJSON(s)
			if err != nil {
				return err
			}
			this.Group = append(this.Group, r)
		}
	}

	if len(_unmarshalled.Aggregates) > 0 {
		this.Aggregates = make(IndexAggregates, 0, len(_unmarshalled.Aggregates))
		for _, s := range _unmarshalled.Aggregates {
			r := &IndexAggregate{}
			err = r.UnmarshalJSON(s)
			if err != nil {
				return err
			}
			this.Aggregates = append(this.Aggregates, r)
		}
	}

	this.DependsOnIndexKeys = _unmarshalled.DependsOnIndexKeys

	return nil
}

func (this *IndexGroupAggregates) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}
