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

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type Fetch struct {
	readonly
	optEstimate
	keyspace datastore.Keyspace
	term     *algebra.KeyspaceTerm
	subPaths []string
}

func NewFetch(keyspace datastore.Keyspace, term *algebra.KeyspaceTerm, subPaths []string,
	cost, cardinality float64, size int64, frCost float64) *Fetch {
	rv := &Fetch{
		keyspace: keyspace,
		term:     term,
		subPaths: subPaths,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *Fetch) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFetch(this)
}

func (this *Fetch) New() Operator {
	return &Fetch{}
}

func (this *Fetch) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *Fetch) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *Fetch) SubPaths() []string {
	return this.subPaths
}

func (this *Fetch) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Fetch) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Fetch"}
	this.term.MarshalKeyspace(r)
	if len(this.subPaths) > 0 {
		r["subpaths"] = this.subPaths
	}

	if this.term.As() != "" {
		r["as"] = this.term.As()
	}
	if this.term.IsUnderNL() {
		r["nested_loop"] = this.term.IsUnderNL()
	}

	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *Fetch) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string                 `json:"#operator"`
		Namespace   string                 `json:"namespace"`
		Bucket      string                 `json:"bucket"`
		Scope       string                 `json:"scope"`
		Keyspace    string                 `json:"keyspace"`
		FromExpr    string                 `json:"fromExpr"`
		As          string                 `json:"as"`
		UnderNL     bool                   `json:"nested_loop"`
		OptEstimate map[string]interface{} `json:"optimizer_estimates"`
		SubPaths    []string               `json:"subpaths"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.subPaths = _unmarshalled.SubPaths

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	if _unmarshalled.FromExpr != "" {
		var expr expression.Expression
		expr, err = parser.Parse(_unmarshalled.FromExpr)
		if err == nil {
			this.term = algebra.NewKeyspaceTermFromExpression(expr, _unmarshalled.As, nil, nil, 0)
		}
	} else {
		this.term = algebra.NewKeyspaceTermFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
			_unmarshalled.Scope, _unmarshalled.Keyspace), _unmarshalled.As, nil, nil)
		this.keyspace, err = datastore.GetKeyspace(this.term.Path().Parts()...)
	}
	if err == nil && _unmarshalled.UnderNL {
		this.term.SetUnderNL()
	}

	return err
}

func (this *Fetch) verify(prepared *Prepared) bool {
	var res bool

	this.keyspace, res = verifyKeyspace(this.keyspace, prepared)
	return res
}

type DummyFetch struct {
	readonly
	optEstimate
	keyspace datastore.Keyspace
	term     *algebra.KeyspaceTerm
}

func NewDummyFetch(keyspace datastore.Keyspace, term *algebra.KeyspaceTerm,
	cost, cardinality float64, size int64, frCost float64) *DummyFetch {
	rv := &DummyFetch{
		keyspace: keyspace,
		term:     term,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *DummyFetch) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDummyFetch(this)
}

func (this *DummyFetch) New() Operator {
	return &DummyFetch{}
}

func (this *DummyFetch) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *DummyFetch) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *DummyFetch) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *DummyFetch) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "DummyFetch"}
	this.term.MarshalKeyspace(r)
	if this.term.As() != "" {
		r["as"] = this.term.As()
	}
	if this.term.IsUnderNL() {
		r["nested_loop"] = this.term.IsUnderNL()
	}

	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *DummyFetch) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string                 `json:"#operator"`
		Namespace   string                 `json:"namespace"`
		Bucket      string                 `json:"bucket"`
		Scope       string                 `json:"scope"`
		Keyspace    string                 `json:"keyspace"`
		FromExpr    string                 `json:"fromExpr"`
		As          string                 `json:"as"`
		UnderNL     bool                   `json:"nested_loop"`
		OptEstimate map[string]interface{} `json:"optimizer_estimates"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	if _unmarshalled.FromExpr != "" {
		expr, err1 := parser.Parse(_unmarshalled.FromExpr)
		if err1 != nil {
			return err1
		}
		this.term = algebra.NewKeyspaceTermFromExpression(expr, _unmarshalled.As, nil, nil, 0)
	} else {
		this.term = algebra.NewKeyspaceTermFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
			_unmarshalled.Scope, _unmarshalled.Keyspace), _unmarshalled.As, nil, nil)
		this.keyspace, err = datastore.GetKeyspace(this.term.Path().Parts()...)
	}

	if err == nil && _unmarshalled.UnderNL {
		this.term.SetUnderNL()
	}
	return err
}
