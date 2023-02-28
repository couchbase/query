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

type SendUpsert struct {
	dml
	optEstimate
	keyspace    datastore.Keyspace
	term        *algebra.KeyspaceRef
	alias       string
	key         expression.Expression
	value       expression.Expression
	options     expression.Expression
	skipNewKeys bool
	fastDiscard bool // if the execution phase should discard items without sending them downstream
}

func NewSendUpsert(keyspace datastore.Keyspace, ksref *algebra.KeyspaceRef,
	key, value, options expression.Expression, cost, cardinality float64,
	size int64, frCost float64, skipNewKeys bool, fastDiscard bool) *SendUpsert {
	rv := &SendUpsert{
		keyspace:    keyspace,
		term:        ksref,
		alias:       ksref.Alias(),
		key:         key,
		value:       value,
		options:     options,
		skipNewKeys: skipNewKeys,
		fastDiscard: fastDiscard,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *SendUpsert) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendUpsert(this)
}

func (this *SendUpsert) New() Operator {
	return &SendUpsert{}
}

func (this *SendUpsert) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *SendUpsert) Term() *algebra.KeyspaceRef {
	return this.term
}

func (this *SendUpsert) Alias() string {
	return this.alias
}

func (this *SendUpsert) Key() expression.Expression {
	return this.key
}

func (this *SendUpsert) Value() expression.Expression {
	return this.value
}

func (this *SendUpsert) Options() expression.Expression {
	return this.options
}

func (this *SendUpsert) SkipNewKeys() bool {
	return this.skipNewKeys
}

func (this *SendUpsert) FastDiscard() bool {
	return this.fastDiscard
}

func (this *SendUpsert) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *SendUpsert) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "SendUpsert"}
	this.term.MarshalKeyspace(r)
	r["alias"] = this.alias

	if this.key != nil {
		r["key"] = this.key.String()
	}

	if this.value != nil {
		r["value"] = this.value.String()
	}

	if this.options != nil {
		r["options"] = this.options.String()
	}

	if this.skipNewKeys {
		r["skip_new_keys"] = this.skipNewKeys
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

func (this *SendUpsert) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string                 `json:"#operator"`
		KeyExpr     string                 `json:"key"`
		ValueExpr   string                 `json:"value"`
		OptionsExpr string                 `json:"options"`
		Namespace   string                 `json:"namespace"`
		Bucket      string                 `json:"bucket"`
		Scope       string                 `json:"scope"`
		Keyspace    string                 `json:"keyspace"`
		Expr        string                 `json:"expr"`
		As          string                 `json:"as"`
		Alias       string                 `json:"alias"`
		OptEstimate map[string]interface{} `json:"optimizer_estimates"`
		SkipNewKeys bool                   `json:"skip_new_keys"`
		FastDiscard bool                   `json:"fast_discard"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.KeyExpr != "" {
		this.key, err = parser.Parse(_unmarshalled.KeyExpr)
		if err != nil {
			return err
		}
	}

	if _unmarshalled.ValueExpr != "" {
		this.value, err = parser.Parse(_unmarshalled.ValueExpr)
		if err != nil {
			return err
		}
	}

	if _unmarshalled.OptionsExpr != "" {
		this.options, err = parser.Parse(_unmarshalled.OptionsExpr)
		if err != nil {
			return err
		}
	}

	this.alias = _unmarshalled.Alias
	this.skipNewKeys = _unmarshalled.SkipNewKeys
	this.fastDiscard = _unmarshalled.FastDiscard

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

func (this *SendUpsert) verify(prepared *Prepared) bool {
	var res bool

	this.keyspace, res = verifyKeyspace(this.keyspace, prepared)
	return res
}
