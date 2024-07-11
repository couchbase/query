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

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type IndexVector struct {
	QueryVector  expression.Expression
	IndexKeyPos  int
	Probes       expression.Expression
	ActualVector expression.Expression
	SquareRoot   bool
}

func NewIndexVector(queryVector expression.Expression, indexKeyPos int,
	probes, actualVector expression.Expression, squareRoot bool) *IndexVector {
	return &IndexVector{
		QueryVector:  queryVector,
		IndexKeyPos:  indexKeyPos,
		Probes:       probes,
		ActualVector: actualVector,
		SquareRoot:   squareRoot,
	}
}

func (this *IndexVector) Copy() *IndexVector {
	return &IndexVector{
		QueryVector:  expression.Copy(this.QueryVector),
		IndexKeyPos:  this.IndexKeyPos,
		Probes:       expression.Copy(this.Probes),
		ActualVector: expression.Copy(this.ActualVector),
	}
}

func (this *IndexVector) EquivalentTo(other *IndexVector) bool {
	if !this.QueryVector.EquivalentTo(other.QueryVector) ||
		this.IndexKeyPos != other.IndexKeyPos {
		return false
	}
	if (this.Probes == nil && other.Probes != nil) ||
		(this.Probes != nil && other.Probes == nil) {
		return false
	} else if this.Probes != nil && !this.Probes.EquivalentTo(other.Probes) {
		return false
	}
	if (this.ActualVector == nil && other.ActualVector != nil) ||
		(this.ActualVector != nil && other.ActualVector == nil) {
		return false
	} else if this.ActualVector != nil && !this.ActualVector.EquivalentTo(other.ActualVector) {
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
	if this.Probes != nil {
		rv["probes"] = this.Probes
	}
	if this.ActualVector != nil {
		rv["actual_vector"] = this.ActualVector
	}
	if this.SquareRoot {
		rv["square_root"] = this.SquareRoot
	}
	return rv
}

func (this *IndexVector) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		QueryVector  string `json:"query_vector"`
		IndexKeyPos  int    `json:"index_key_pos"`
		Probes       string `json:"probes"`
		ActualVector string `json:"actual_vector"`
		SquareRoot   bool   `json:"square_root"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.IndexKeyPos = _unmarshalled.IndexKeyPos
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

	if _unmarshalled.ActualVector != "" {
		this.ActualVector, err = parser.Parse(_unmarshalled.ActualVector)
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
