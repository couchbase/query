//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package plan

import (
	"encoding/json"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type Join struct {
	readonly
	optEstimate
	keyspace datastore.Keyspace
	term     *algebra.KeyspaceTerm
	outer    bool
	onFilter expression.Expression
}

func NewJoin(keyspace datastore.Keyspace, join *algebra.Join, cost, cardinality float64,
	size int64, frCost float64) *Join {
	rv := &Join{
		keyspace: keyspace,
		term:     join.Right(),
		outer:    join.Outer(),
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func NewJoinFromAnsi(keyspace datastore.Keyspace, term *algebra.KeyspaceTerm, outer bool,
	onFilter expression.Expression, cost, cardinality float64,
	size int64, frCost float64) *Join {
	rv := &Join{
		keyspace: keyspace,
		term:     term,
		outer:    outer,
		onFilter: onFilter,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *Join) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitJoin(this)
}

func (this *Join) New() Operator {
	return &Join{}
}

func (this *Join) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *Join) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *Join) Outer() bool {
	return this.outer
}

func (this *Join) OnFilter() expression.Expression {
	return this.onFilter
}

func (this *Join) SetOnFilter(onFilter expression.Expression) {
	this.onFilter = onFilter
}

func (this *Join) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Join) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Join"}
	this.term.MarshalKeyspace(r)
	r["on_keys"] = expression.NewStringer().Visit(this.term.JoinKeys())

	if this.outer {
		r["outer"] = this.outer
	}

	if this.term.As() != "" {
		r["as"] = this.term.As()
	}

	if this.onFilter != nil {
		r["on_filter"] = expression.NewStringer().Visit(this.onFilter)
	}

	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *Join) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string                 `json:"#operator"`
		Namespace   string                 `json:"namespace"`
		Bucket      string                 `json:"bucket"`
		Scope       string                 `json:"scope"`
		Keyspace    string                 `json:"keyspace"`
		On          string                 `json:"on_keys"`
		Outer       bool                   `json:"outer"`
		As          string                 `json:"as"`
		OnFilter    string                 `json:"on_filter"`
		OptEstimate map[string]interface{} `json:"optimizer_estimates"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	var keys_expr expression.Expression
	if _unmarshalled.On != "" {
		keys_expr, err = parser.Parse(_unmarshalled.On)
		if err != nil {
			return err
		}
	}

	this.outer = _unmarshalled.Outer
	this.term = algebra.NewKeyspaceTermFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), _unmarshalled.As, nil, nil)
	this.term.SetJoinKeys(keys_expr)
	this.keyspace, err = datastore.GetKeyspace(this.term.Path().Parts()...)
	if err != nil {
		return err
	}

	if _unmarshalled.OnFilter != "" {
		this.onFilter, err = parser.Parse(_unmarshalled.OnFilter)
		if err != nil {
			return err
		}
	}

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	return nil
}

func (this *Join) verify(prepared *Prepared) bool {
	var res bool

	this.keyspace, res = verifyKeyspace(this.keyspace, prepared)
	return res
}
