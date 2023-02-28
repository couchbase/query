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

type SendDelete struct {
	dml
	optEstimate
	keyspace     datastore.Keyspace
	term         *algebra.KeyspaceRef
	alias        string
	limit        expression.Expression
	validateKeys bool
	fastDiscard  bool // if the execution phase should discard items without sending them downstream
}

func NewSendDelete(keyspace datastore.Keyspace, ksref *algebra.KeyspaceRef, limit expression.Expression,
	cost, cardinality float64, size int64, frCost float64, fastDiscard bool) *SendDelete {
	rv := &SendDelete{
		keyspace:    keyspace,
		term:        ksref,
		alias:       ksref.Alias(),
		limit:       limit,
		fastDiscard: fastDiscard,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *SendDelete) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendDelete(this)
}

func (this *SendDelete) New() Operator {
	return &SendDelete{}
}

func (this *SendDelete) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *SendDelete) SetValidateKeys(on bool) {
	this.validateKeys = on
}

func (this *SendDelete) ValidateKeys() bool {
	return this.validateKeys
}

func (this *SendDelete) Term() *algebra.KeyspaceRef {
	return this.term
}

func (this *SendDelete) Alias() string {
	return this.alias
}

func (this *SendDelete) Limit() expression.Expression {
	return this.limit
}

func (this *SendDelete) FastDiscard() bool {
	return this.fastDiscard
}

func (this *SendDelete) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *SendDelete) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "SendDelete"}
	this.term.MarshalKeyspace(r)
	r["alias"] = this.alias

	if this.limit != nil {
		r["limit"] = this.limit
	}

	if this.validateKeys {
		r["validate_keys"] = this.validateKeys
	}

	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}

	r["fast_discard"] = this.fastDiscard

	if f != nil {
		f(r)
	}
	return r
}

func (this *SendDelete) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_            string                 `json:"#operator"`
		Namespace    string                 `json:"namespace"`
		Bucket       string                 `json:"bucket"`
		Scope        string                 `json:"scope"`
		Keyspace     string                 `json:"keyspace"`
		Expr         string                 `json:"expr"`
		As           string                 `json:"as"`
		Alias        string                 `json:"alias"`
		Limit        string                 `json:"limit"`
		OptEstimate  map[string]interface{} `json:"optimizer_estimates"`
		ValidateKeys bool                   `json:"validate_keys"`
		FastDiscard  bool                   `json:"fast_discard"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.alias = _unmarshalled.Alias
	this.validateKeys = _unmarshalled.ValidateKeys
	this.fastDiscard = _unmarshalled.FastDiscard

	if _unmarshalled.Limit != "" {
		this.limit, err = parser.Parse(_unmarshalled.Limit)
		if err != nil {
			return err
		}
	}

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	if _unmarshalled.Expr != "" {
		var expr expression.Expression
		expr, err = parser.Parse(_unmarshalled.Expr)
		if err == nil {
			this.term = algebra.NewKeyspaceRefFromExpression(expr, _unmarshalled.As)
		}
	} else {
		this.term = algebra.NewKeyspaceRefFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
			_unmarshalled.Scope, _unmarshalled.Keyspace), _unmarshalled.As)
		this.keyspace, err = datastore.GetKeyspace(this.term.Path().Parts()...)
	}

	return err
}

func (this *SendDelete) verify(prepared *Prepared) bool {
	var res bool

	this.keyspace, res = verifyKeyspace(this.keyspace, prepared)
	return res
}
