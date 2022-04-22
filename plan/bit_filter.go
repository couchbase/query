//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

/*
Package plan provides query plans.
*/
package plan

import (
	"encoding/json"
	"fmt"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type BitFilterIndex struct {
	indexName string
	indexId   string
	exprs     expression.Expressions
}

func newBitFilterIndex(index datastore.Index, exprs expression.Expressions) *BitFilterIndex {
	return &BitFilterIndex{
		indexName: index.Name(),
		indexId:   index.Id(),
		exprs:     exprs,
	}
}

func (this *BitFilterIndex) IndexName() string {
	return this.indexName
}

func (this *BitFilterIndex) IndexId() string {
	return this.indexId
}

func (this *BitFilterIndex) Expressions() expression.Expressions {
	return this.exprs
}

func (this *BitFilterIndex) MarshalJSON() ([]byte, error) {
	stringer := expression.NewStringer()
	r := make(map[string]interface{}, 3)
	r["index_name"] = this.indexName
	r["index_id"] = this.indexId
	bfexprs := make([]string, 0, len(this.exprs))
	for _, exp := range this.exprs {
		bfexprs = append(bfexprs, stringer.Visit(exp))
	}
	r["bit_filter_expressions"] = bfexprs
	return json.Marshal(r)
}

func (this *BitFilterIndex) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		IndexName      string   `json:"index_name"`
		IndexId        string   `json:"index_id"`
		BitFilterExprs []string `json:"bit_filter_expressions"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.indexName = _unmarshalled.IndexName
	this.indexId = _unmarshalled.IndexId

	if len(_unmarshalled.BitFilterExprs) > 0 {
		this.exprs = make(expression.Expressions, len(_unmarshalled.BitFilterExprs))
		for i, bfltr := range _unmarshalled.BitFilterExprs {
			expr, err := parser.Parse(bfltr)
			if err != nil {
				return err
			}
			this.exprs[i] = expr
		}
	}

	return nil
}

type BitFilterTerm struct {
	alias    string
	indexBFs []*BitFilterIndex
}

func newBitFilterTerm(alias string) *BitFilterTerm {
	return &BitFilterTerm{
		alias: alias,
	}
}

func (this *BitFilterTerm) Alias() string {
	return this.alias
}

func (this *BitFilterTerm) IndexBitFilters() []*BitFilterIndex {
	return this.indexBFs
}

func (this *BitFilterTerm) addBitFilterIndex(bfIndexExprs map[datastore.Index]expression.Expressions) error {
	if this.indexBFs == nil {
		this.indexBFs = make([]*BitFilterIndex, 0, len(bfIndexExprs))
	}
	for index, exprs := range bfIndexExprs {
		for _, bfIndex := range this.indexBFs {
			if index.Id() == bfIndex.indexId {
				return errors.NewPlanInternalError(fmt.Sprintf("BitFilterTerm.addBitFilterIndex: bit filters for term %s (index %s) already exists", this.alias, bfIndex.indexName))
			}
		}
		bfIndex := newBitFilterIndex(index, exprs)
		this.indexBFs = append(this.indexBFs, bfIndex)
	}
	return nil
}

func (this *BitFilterTerm) MarshalJSON() ([]byte, error) {
	r := make(map[string]interface{}, 2)
	r["alias"] = this.alias
	bfixs := make([]interface{}, 0, len(this.indexBFs))
	for _, ix := range this.indexBFs {
		bfixs = append(bfixs, ix)
	}
	r["index_bit_filters"] = bfixs
	return json.Marshal(r)
}

func (this *BitFilterTerm) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Alias           string            `json:"alias"`
		IndexBitFilters []json.RawMessage `json:"index_bit_filters"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.alias = _unmarshalled.Alias

	if len(_unmarshalled.IndexBitFilters) > 0 {
		this.indexBFs = make([]*BitFilterIndex, len(_unmarshalled.IndexBitFilters))
		for i, ibf := range _unmarshalled.IndexBitFilters {
			indexBF := &BitFilterIndex{}
			err = indexBF.UnmarshalJSON(ibf)
			if err != nil {
				return err
			}
			this.indexBFs[i] = indexBF
		}
	}

	return nil
}

type BitFilters []*BitFilterTerm

func addBitFilters(bitFilters BitFilters, alias string, bfIndexExprs map[datastore.Index]expression.Expressions) (BitFilters, error) {

	var bfTerm *BitFilterTerm
	var err error
	for _, bfk := range bitFilters {
		if bfk.alias == alias {
			bfTerm = bfk
			break
		}
	}
	if bfTerm == nil {
		bfTerm = newBitFilterTerm(alias)
		bfTerm.addBitFilterIndex(bfIndexExprs)
		return append(bitFilters, bfTerm), nil
	}
	err = bfTerm.addBitFilterIndex(bfIndexExprs)
	return bitFilters, err
}

type BuildBitFilterBase struct {
	buildBitFilters BitFilters
}

func (this *BuildBitFilterBase) GetBuildFilterBase() *BuildBitFilterBase {
	return this
}

func (this *BuildBitFilterBase) hasBuildBitFilter() bool {
	return len(this.buildBitFilters) > 0
}

func (this *BuildBitFilterBase) SetBuildBitFilters(alias string, buildExprs map[datastore.Index]expression.Expressions) (err error) {
	this.buildBitFilters, err = addBitFilters(this.buildBitFilters, alias, buildExprs)
	return
}

func (this *BuildBitFilterBase) GetBuildBitFilters() BitFilters {
	return this.buildBitFilters
}

func (this *BuildBitFilterBase) marshalBuildBitFilters(r map[string]interface{}) {
	buildBFs := make([]interface{}, 0, len(this.buildBitFilters))
	for _, bbf := range this.buildBitFilters {
		buildBFs = append(buildBFs, bbf)
	}
	r["build_bit_filters"] = buildBFs
}

func (this *BuildBitFilterBase) unmarshalBuildBitFilters(buildBitFilters []json.RawMessage) (err error) {
	this.buildBitFilters = make(BitFilters, 0, len(buildBitFilters))
	for _, bbf := range buildBitFilters {
		buildBF := &BitFilterTerm{}
		err = buildBF.UnmarshalJSON(bbf)
		if err != nil {
			return
		}
		this.buildBitFilters = append(this.buildBitFilters, buildBF)
	}
	return
}

type ProbeBitFilterBase struct {
	probeBitFilters BitFilters
}

func (this *ProbeBitFilterBase) GetProbeFilterBase() *ProbeBitFilterBase {
	return this
}

func (this *ProbeBitFilterBase) hasProbeBitFilter() bool {
	return len(this.probeBitFilters) > 0
}

func (this *ProbeBitFilterBase) SetProbeBitFilters(alias string, probeExprs map[datastore.Index]expression.Expressions) (err error) {
	this.probeBitFilters, err = addBitFilters(this.probeBitFilters, alias, probeExprs)
	return
}

func (this *ProbeBitFilterBase) GetProbeBitFilters() BitFilters {
	return this.probeBitFilters
}

func (this *ProbeBitFilterBase) marshalProbeBitFilters(r map[string]interface{}) {
	probeBFs := make([]interface{}, 0, len(this.probeBitFilters))
	for _, pbf := range this.probeBitFilters {
		probeBFs = append(probeBFs, pbf)
	}
	r["probe_bit_filters"] = probeBFs
}

func (this *ProbeBitFilterBase) unmarshalProbeBitFilters(probeBitFilters []json.RawMessage) (err error) {
	this.probeBitFilters = make(BitFilters, 0, len(probeBitFilters))
	for _, pbf := range probeBitFilters {
		probeBF := &BitFilterTerm{}
		err = probeBF.UnmarshalJSON(pbf)
		if err != nil {
			return
		}
		this.probeBitFilters = append(this.probeBitFilters, probeBF)
	}
	return
}
