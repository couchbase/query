//  Copyright 2026-Present Couchbase, Inc.
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
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

// ExternalScan is used for scanning external collections (e.g., Iceberg tables).
type ExternalScan struct {
	readonly
	optEstimate
	keyspace       datastore.Keyspace
	term           *algebra.KeyspaceTerm
	subPaths       []string
	projection     []string
	filter         expression.Expression
	snapshotIdExpr expression.Expression
	snapshotTsExpr expression.Expression
	nested_loop    bool
	countOnly      bool
}

func NewExternalScan(keyspace datastore.Keyspace, term *algebra.KeyspaceTerm, subPaths []string,
	filter, snapshotIdExpr, snapshotTsExpr expression.Expression, nested_loop bool,
	cost, cardinality float64, size int64, frCost float64) *ExternalScan {
	rv := &ExternalScan{
		keyspace:       keyspace,
		term:           term,
		subPaths:       subPaths,
		filter:         filter,
		snapshotIdExpr: snapshotIdExpr,
		snapshotTsExpr: snapshotTsExpr,
		nested_loop:    nested_loop,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *ExternalScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExternalScan(this)
}

func (this *ExternalScan) New() Operator {
	return &ExternalScan{}
}

func (this *ExternalScan) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *ExternalScan) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *ExternalScan) SubPaths() []string {
	return this.subPaths
}

func (this *ExternalScan) EarlyProjection() []string {
	return this.projection
}

func (this *ExternalScan) SetEarlyProjection(projection []string) {
	this.projection = projection
}

func (this *ExternalScan) Filter() expression.Expression {
	return this.filter
}

func (this *ExternalScan) SetFilter(filter expression.Expression) {
	this.filter = filter
}

func (this *ExternalScan) SnapshotIdExpr() expression.Expression {
	return this.snapshotIdExpr
}

func (this *ExternalScan) SnapshotTimestampExpr() expression.Expression {
	return this.snapshotTsExpr
}

func (this *ExternalScan) IsUnderNL() bool {
	return this.nested_loop
}

// CountOnly reports whether this scan only needs to produce the total row
// count (SELECT COUNT(*) with no predicate), letting the datastore answer
// from catalog/file metadata instead of reading and converting every row.
func (this *ExternalScan) CountOnly() bool {
	return this.countOnly
}

func (this *ExternalScan) SetCountOnly(countOnly bool) {
	this.countOnly = countOnly
}

func (this *ExternalScan) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *ExternalScan) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "ExternalScan"}
	this.term.MarshalKeyspace(r)
	if this.term.As() != "" {
		r["as"] = this.term.As()
	}
	if len(this.subPaths) > 0 {
		r["subpaths"] = this.subPaths
	}
	if len(this.projection) > 0 {
		r["early_projection"] = this.projection
	}
	if this.filter != nil {
		r["filter"] = this.filter.String()
	}
	if this.snapshotIdExpr != nil {
		r["snapshot_id"] = this.snapshotIdExpr.String()
	}
	if this.snapshotTsExpr != nil {
		r["snapshot_timestamp"] = this.snapshotTsExpr.String()
	}
	if this.nested_loop {
		r["nested_loop"] = this.nested_loop
	}
	if this.countOnly {
		r["count_only"] = this.countOnly
	}
	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}
	if f != nil {
		f(r)
	}
	return r
}

func (this *ExternalScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_                 string                 `json:"#operator"`
		Namespace         string                 `json:"namespace"`
		Bucket            string                 `json:"bucket"`
		Scope             string                 `json:"scope"`
		Keyspace          string                 `json:"keyspace"`
		As                string                 `json:"as"`
		SubPaths          []string               `json:"subpaths"`
		Projection        []string               `json:"early_projection"`
		Filter            string                 `json:"filter"`
		SnapshotId        string                 `json:"snapshot_id"`
		SnapshotTimestamp string                 `json:"snapshot_timestamp"`
		NestedLoop        bool                   `json:"nested_loop"`
		CountOnly         bool                   `json:"count_only"`
		OptEstimate       map[string]interface{} `json:"optimizer_estimates"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.subPaths = _unmarshalled.SubPaths
	this.projection = _unmarshalled.Projection
	this.nested_loop = _unmarshalled.NestedLoop
	this.countOnly = _unmarshalled.CountOnly

	this.term = algebra.NewKeyspaceTermFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), _unmarshalled.As, nil, nil)
	this.keyspace, err = datastore.GetKeyspace(this.term.Path().Parts()...)
	if err != nil {
		return err
	}

	if _unmarshalled.Filter != "" {
		this.filter, err = parser.Parse(_unmarshalled.Filter)
		if err != nil {
			return err
		}
	}

	if _unmarshalled.SnapshotId != "" {
		this.snapshotIdExpr, err = parser.Parse(_unmarshalled.SnapshotId)
		if err != nil {
			return err
		}
	}

	if _unmarshalled.SnapshotTimestamp != "" {
		this.snapshotTsExpr, err = parser.Parse(_unmarshalled.SnapshotTimestamp)
		if err != nil {
			return err
		}
	}

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	return nil
}
