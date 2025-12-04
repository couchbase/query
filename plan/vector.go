//  Copyright 2024-Present Couchbase, Inc.
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

const RERANK_FACTOR = 5

type IndexVector struct {
	QueryVector expression.Expression
	IndexKeyPos int
	VectorType  string
	Probes      expression.Expression
	ReRank      expression.Expression
	TopNScan    expression.Expression
	SquareRoot  bool
}

func NewIndexVector(queryVector expression.Expression, indexKeyPos int, vectorType string,
	probes, reRank, topNScan expression.Expression, squareRoot bool) *IndexVector {
	return &IndexVector{
		QueryVector: queryVector,
		IndexKeyPos: indexKeyPos,
		VectorType:  vectorType,
		Probes:      probes,
		ReRank:      reRank,
		TopNScan:    topNScan,
		SquareRoot:  squareRoot,
	}
}

func (this *IndexVector) Copy() *IndexVector {
	return &IndexVector{
		QueryVector: expression.Copy(this.QueryVector),
		IndexKeyPos: this.IndexKeyPos,
		VectorType:  this.VectorType,
		Probes:      expression.Copy(this.Probes),
		ReRank:      expression.Copy(this.ReRank),
		TopNScan:    expression.Copy(this.TopNScan),
	}
}

func (this *IndexVector) EquivalentTo(other *IndexVector) bool {
	if !this.QueryVector.EquivalentTo(other.QueryVector) ||
		this.IndexKeyPos != other.IndexKeyPos ||
		this.VectorType != other.VectorType {
		return false
	}
	if (this.Probes == nil && other.Probes != nil) ||
		(this.Probes != nil && other.Probes == nil) {
		return false
	} else if this.Probes != nil && !this.Probes.EquivalentTo(other.Probes) {
		return false
	}
	if (this.TopNScan == nil && other.TopNScan != nil) ||
		(this.TopNScan != nil && other.TopNScan == nil) {
		return false
	} else if this.TopNScan != nil && !this.TopNScan.EquivalentTo(other.TopNScan) {
		return false
	}
	if (this.ReRank == nil && other.ReRank != nil) ||
		(this.ReRank != nil && other.ReRank == nil) {
		return false
	} else if this.ReRank != nil && !this.ReRank.EquivalentTo(other.ReRank) {
		return false
	}
	return true
}

func (this *IndexVector) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IndexVector) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	rv := map[string]interface{}{
		"query_vector":  this.QueryVector,
		"index_key_pos": this.IndexKeyPos,
	}
	if this.VectorType != "" {
		rv["vector_type"] = this.VectorType
	}
	if this.Probes != nil {
		rv["probes"] = this.Probes
	}
	if this.ReRank != nil {
		rv["re_rank"] = this.ReRank
	}
	if this.TopNScan != nil {
		rv["top_nscan"] = this.TopNScan
	}
	if this.SquareRoot {
		rv["square_root"] = this.SquareRoot
	}
	return rv
}

func (this *IndexVector) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		QueryVector string `json:"query_vector"`
		IndexKeyPos int    `json:"index_key_pos"`
		VectorType  string `json:"vector_type"`
		Probes      string `json:"probes"`
		ReRank      string `json:"re_rank"`
		TopNScan    string `json:"top_nscan"`
		SquareRoot  bool   `json:"square_root"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.IndexKeyPos = _unmarshalled.IndexKeyPos
	this.VectorType = _unmarshalled.VectorType
	if this.VectorType == "" {
		this.VectorType = datastore.IK_DENSE_VECTOR_NAME
	}
	this.SquareRoot = _unmarshalled.SquareRoot

	if _unmarshalled.QueryVector != "" {
		this.QueryVector, err = parser.Parse(_unmarshalled.QueryVector)
		if err != nil {
			return err
		}
	}

	if _unmarshalled.Probes != "" {
		this.Probes, err = parser.Parse(_unmarshalled.Probes)
		if err != nil {
			return err
		}
	}

	if _unmarshalled.ReRank != "" {
		this.ReRank, err = parser.Parse(_unmarshalled.ReRank)
		if err != nil {
			return err
		}
	}

	if _unmarshalled.TopNScan != "" {
		this.TopNScan, err = parser.Parse(_unmarshalled.TopNScan)
		if err != nil {
			return err
		}
	}

	return nil
}

type IndexPartitionSets []*IndexPartitionSet

type IndexPartitionSet struct {
	PartitionSet expression.Expressions
}

func NewIndexPartitionSet(partitionSet expression.Expressions) *IndexPartitionSet {
	return &IndexPartitionSet{
		PartitionSet: partitionSet,
	}
}
