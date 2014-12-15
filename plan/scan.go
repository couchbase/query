//  Copyright (c) 2014 Couchbase, Inc.
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

	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/expression/parser"
	"github.com/couchbaselabs/query/planner"
)

type PrimaryScan struct {
	readonly
	index datastore.PrimaryIndex
	term  *algebra.KeyspaceTerm
}

func NewPrimaryScan(index datastore.PrimaryIndex, term *algebra.KeyspaceTerm) *PrimaryScan {
	return &PrimaryScan{
		index: index,
		term:  term,
	}
}

func (this *PrimaryScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitPrimaryScan(this)
}

func (this *PrimaryScan) New() Operator {
	return &PrimaryScan{}
}

func (this *PrimaryScan) Index() datastore.PrimaryIndex {
	return this.index
}

func (this *PrimaryScan) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *PrimaryScan) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "PrimaryScan"}
	r["index"] = this.index.Name()
	r["namespace"] = this.term.Namespace()
	r["keyspace"] = this.term.Keyspace()
	return json.Marshal(r)
}

func (this *PrimaryScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string `json:"#operator"`
		Index string `json:"index"`
		Names string `json:"namespace"`
		Keys  string `json:"keyspace"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	k, err := datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)
	if err != nil {
		return err
	}

	this.term = algebra.NewKeyspaceTerm(
		_unmarshalled.Names, _unmarshalled.Keys,
		nil, "", nil)
	this.index, err = k.IndexByPrimary()

	return err
}

type IndexScan struct {
	readonly
	index    datastore.Index
	term     *algebra.KeyspaceTerm
	spans    planner.Spans
	distinct bool
	limit    int64
}

func NewIndexScan(index datastore.Index, term *algebra.KeyspaceTerm,
	spans planner.Spans, distinct bool, limit int64) *IndexScan {
	return &IndexScan{
		index:    index,
		term:     term,
		spans:    spans,
		distinct: distinct,
		limit:    limit,
	}
}

func (this *IndexScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexScan(this)
}

func (this *IndexScan) New() Operator {
	return &IndexScan{}
}

func (this *IndexScan) Index() datastore.Index {
	return this.index
}

func (this *IndexScan) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *IndexScan) Spans() planner.Spans {
	return this.spans
}

func (this *IndexScan) Distinct() bool {
	return this.distinct
}

func (this *IndexScan) Limit() int64 {
	return this.limit
}

func (this *IndexScan) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "IndexScan"}
	r["index"] = this.index.Name()
	r["namespace"] = this.term.Namespace()
	r["keyspace"] = this.term.Keyspace()

	// FIXME
	r["spans"] = this.spans

	if this.distinct {
		r["distinct"] = this.distinct
	}

	if this.limit >= 0 {
		r["limit"] = this.limit
	}

	return json.Marshal(r)
}

func (this *IndexScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_        string        `json:"#operator"`
		Index    string        `json:"index"`
		Names    string        `json:"namespace"`
		Keys     string        `json:"keyspace"`
		Spans    planner.Spans `json:"spans"`
		Distinct bool          `json:"distinct"`
		Limit    int64         `json:"limit"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	k, err := datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)
	if err != nil {
		return err
	}

	this.term = algebra.NewKeyspaceTerm(
		_unmarshalled.Names, _unmarshalled.Keys,
		nil, "", nil)
	this.spans = _unmarshalled.Spans
	this.distinct = _unmarshalled.Distinct
	this.limit = _unmarshalled.Limit
	this.index, err = k.IndexByName(_unmarshalled.Index)
	return err
}

// KeyScan is used for KEYS clauses (except after JOIN / NEST).
type KeyScan struct {
	readonly
	keys expression.Expression
}

func NewKeyScan(keys expression.Expression) *KeyScan {
	return &KeyScan{
		keys: keys,
	}
}

func (this *KeyScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitKeyScan(this)
}

func (this *KeyScan) New() Operator {
	return &KeyScan{}
}

func (this *KeyScan) Keys() expression.Expression {
	return this.keys
}

func (this *KeyScan) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "KeyScan"}
	r["keys"] = expression.NewStringer().Visit(this.keys)
	return json.Marshal(r)
}

func (this *KeyScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_    string `json:"#operator"`
		Keys string `json:"keys"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	keys_expr, err := parser.Parse(_unmarshalled.Keys)
	this.keys = keys_expr

	return err
}

// ParentScan is used for UNNEST subqueries.
type ParentScan struct {
	readonly
}

func NewParentScan() *ParentScan {
	return &ParentScan{}
}

func (this *ParentScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitParentScan(this)
}

func (this *ParentScan) New() Operator {
	return &ParentScan{}
}

func (this *ParentScan) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "ParentScan"}
	return json.Marshal(r)
}

func (this *ParentScan) UnmarshalJSON([]byte) error {
	// NOP: ParentScan has no data structure
	return nil
}

// ValueScan is used for VALUES clauses, e.g. in INSERTs.
type ValueScan struct {
	readonly
	values expression.Expression
}

func NewValueScan(values expression.Expression) *ValueScan {
	return &ValueScan{
		values: values,
	}
}

func (this *ValueScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitValueScan(this)
}

func (this *ValueScan) New() Operator {
	return &ValueScan{}
}

func (this *ValueScan) Values() expression.Expression {
	return this.values
}

func (this *ValueScan) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "ValueScan"}
	r["values"] = expression.NewStringer().Visit(this.values)
	return json.Marshal(r)
}

func (this *ValueScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_      string `json:"#operator"`
		Values string `json:"values"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.values, err = parser.Parse(_unmarshalled.Values)

	return err
}

// DummyScan is used for SELECTs with no FROM clause.
type DummyScan struct {
	readonly
}

func NewDummyScan() *DummyScan {
	return &DummyScan{}
}

func (this *DummyScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDummyScan(this)
}

func (this *DummyScan) New() Operator {
	return &DummyScan{}
}

func (this *DummyScan) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{"#operator": "DummyScan"})
}

func (this *DummyScan) UnmarshalJSON([]byte) error {
	// NOP: DummyScan has no data structure
	return nil
}

// CountScan is used for SELECT COUNT(*) with no WHERE clause.
type CountScan struct {
	readonly
	keyspace datastore.Keyspace
	term     *algebra.KeyspaceTerm
}

func NewCountScan(keyspace datastore.Keyspace, term *algebra.KeyspaceTerm) *CountScan {
	return &CountScan{
		keyspace: keyspace,
		term:     term,
	}
}

func (this *CountScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCountScan(this)
}

func (this *CountScan) New() Operator {
	return &CountScan{}
}

func (this *CountScan) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *CountScan) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *CountScan) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "CountScan"}
	r["namespace"] = this.term.Namespace()
	r["keyspace"] = this.term.Keyspace()
	return json.Marshal(r)
}

func (this *CountScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string `json:"#operator"`
		Names string `json:"namespace"`
		Keys  string `json:"keyspace"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)

	return err
}

// IntersectScan scans multiple indexes and intersects the results.
type IntersectScan struct {
	readonly
	scans []Operator
}

func NewIntersectScan(scans ...Operator) *IntersectScan {
	return &IntersectScan{
		scans: scans,
	}
}

func (this *IntersectScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIntersectScan(this)
}

func (this *IntersectScan) New() Operator {
	return &IntersectScan{}
}

func (this *IntersectScan) Scans() []Operator {
	return this.scans
}

func (this *IntersectScan) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "IntersectScan"}

	// FIXME
	r["scans"] = this.scans

	return json.Marshal(r)
}

func (this *IntersectScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string            `json:"#operator"`
		Scans []json.RawMessage `json:"scans"`
	}
	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.scans = []Operator{}

	for _, raw_scan := range _unmarshalled.Scans {
		var scan_type struct {
			Operator string `json:"#operator"`
		}
		var read_only struct {
			Readonly bool `json:"readonly"`
		}
		err = json.Unmarshal(raw_scan, &scan_type)
		if err != nil {
			return err
		}

		if scan_type.Operator == "" {
			err = json.Unmarshal(raw_scan, &read_only)
			if err != nil {
				return err
			} else {
				// This should be a readonly object
			}
		} else {
			scan_op, err := MakeOperator(scan_type.Operator, raw_scan)
			if err != nil {
				return err
			}

			this.scans = append(this.scans, scan_op)
		}
	}

	return err
}

// UnionScan scans multiple indexes and unions the results.
type UnionScan struct {
	readonly
	scans []Operator
}

func NewUnionScan(scans ...Operator) *UnionScan {
	return &UnionScan{
		scans: scans,
	}
}

func (this *UnionScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnionScan(this)
}

func (this *UnionScan) New() Operator {
	return &UnionScan{}
}

func (this *UnionScan) Scans() []Operator {
	return this.scans
}

func (this *UnionScan) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "UnionScan"}

	// FIXME
	r["scans"] = this.scans

	return json.Marshal(r)
}

func (this *UnionScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string            `json:"#operator"`
		Scans []json.RawMessage `json:"scans"`
	}
	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.scans = []Operator{}

	for _, raw_scan := range _unmarshalled.Scans {
		var scan_type struct {
			Operator string `json:"#operator"`
		}
		var read_only struct {
			Readonly bool `json:"readonly"`
		}
		err = json.Unmarshal(raw_scan, &scan_type)
		if err != nil {
			return err
		}

		if scan_type.Operator == "" {
			err = json.Unmarshal(raw_scan, &read_only)
			if err != nil {
				return err
			} else {
				// This should be a readonly object
			}
		} else {
			scan_op, err := MakeOperator(scan_type.Operator, raw_scan)
			if err != nil {
				return err
			}

			this.scans = append(this.scans, scan_op)
		}
	}

	return err
}
