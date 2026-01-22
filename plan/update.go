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
	"github.com/couchbase/query/algebra/unmarshal"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

// Enable copy-before-write, so that all reads use old values
type Clone struct {
	readonly
	optEstimate
	alias string
}

func NewClone(alias string, cost, cardinality float64, size int64, frCost float64) *Clone {
	rv := &Clone{
		alias: alias,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *Clone) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitClone(this)
}

func (this *Clone) New() Operator {
	return &Clone{}
}

func (this *Clone) Alias() string {
	return this.alias
}

func (this *Clone) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Clone) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Clone"}
	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}
	if f != nil {
		f(r)
	}
	return r
}

func (this *Clone) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		OptEstimate map[string]interface{} `json:"optimizer_estimates"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	return nil
}

// Write to copy
type Set struct {
	readonly
	optEstimate
	node *algebra.Set
}

func NewSet(node *algebra.Set, cost, cardinality float64, size int64, frCost float64) *Set {
	rv := &Set{
		node: node,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *Set) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSet(this)
}

func (this *Set) New() Operator {
	return &Set{}
}

func (this *Set) Node() *algebra.Set {
	return this.node
}

func (this *Set) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Set) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Set"}
	r["set_terms"] = this.node.Terms()
	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}
	if f != nil {
		f(r)
	}
	return r
}

func (this *Set) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string                 `json:"#operator"`
		SetTerms    json.RawMessage        `json:"set_terms"`
		OptEstimate map[string]interface{} `json:"optimizer_estimates"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	terms, err := unmarshal.UnmarshalSetTerms(_unmarshalled.SetTerms)
	if err != nil {
		return err
	}

	this.node = algebra.NewSet(terms)

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	planContext := this.PlanContext()
	if planContext != nil {
		err = this.node.MapExpressions(planContext)
		if err != nil {
			return err
		}
	}

	return nil
}

// Write to copy
type Unset struct {
	readonly
	optEstimate
	node *algebra.Unset
}

func NewUnset(node *algebra.Unset, cost, cardinality float64, size int64, frCost float64) *Unset {
	rv := &Unset{
		node: node,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *Unset) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnset(this)
}

func (this *Unset) New() Operator {
	return &Unset{}
}

func (this *Unset) Node() *algebra.Unset {
	return this.node
}

func (this *Unset) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Unset) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Unset"}
	r["unset_terms"] = this.node.Terms()
	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}
	if f != nil {
		f(r)
	}
	return r
}

func (this *Unset) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string                 `json:"#operator"`
		UnsetTerms  json.RawMessage        `json:"unset_terms"`
		OptEstimate map[string]interface{} `json:"optimizer_estimates"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	terms, err := unmarshal.UnmarshalUnsetTerms(_unmarshalled.UnsetTerms)
	if err != nil {
		return err
	}

	this.node = algebra.NewUnset(terms)

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	planContext := this.PlanContext()
	if planContext != nil {
		err = this.node.MapExpressions(planContext)
		if err != nil {
			return err
		}
	}

	return nil
}

// Send to keyspace
type SendUpdate struct {
	dml
	optEstimate
	keyspace    datastore.Keyspace
	term        *algebra.KeyspaceRef
	alias       string
	limit       expression.Expression
	fastDiscard bool // if the execution phase should discard items without sending them downstream
}

func NewSendUpdate(keyspace datastore.Keyspace, ksref *algebra.KeyspaceRef,
	limit expression.Expression, cost, cardinality float64, size int64, frCost float64, fastDiscard bool) *SendUpdate {
	rv := &SendUpdate{
		keyspace:    keyspace,
		term:        ksref,
		alias:       ksref.Alias(),
		limit:       limit,
		fastDiscard: fastDiscard,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *SendUpdate) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendUpdate(this)
}

func (this *SendUpdate) New() Operator {
	return &SendUpdate{}
}

func (this *SendUpdate) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *SendUpdate) Term() *algebra.KeyspaceRef {
	return this.term
}

func (this *SendUpdate) Alias() string {
	return this.alias
}

func (this *SendUpdate) Limit() expression.Expression {
	return this.limit
}

func (this *SendUpdate) FastDiscard() bool {
	return this.fastDiscard
}

func (this *SendUpdate) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *SendUpdate) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "SendUpdate"}
	this.term.MarshalKeyspace(r)
	r["alias"] = this.alias

	if this.limit != nil {
		r["limit"] = this.limit
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

func (this *SendUpdate) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string                 `json:"#operator"`
		Namespace   string                 `json:"namespace"`
		Bucket      string                 `json:"bucket"`
		Scope       string                 `json:"scope"`
		Keyspace    string                 `json:"keyspace"`
		Expr        string                 `json:"expr"`
		As          string                 `json:"as"`
		Alias       string                 `json:"alias"`
		Limit       string                 `json:"limit"`
		OptEstimate map[string]interface{} `json:"optimizer_estimates"`
		FastDiscard bool                   `json:"fast_discard"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.alias = _unmarshalled.Alias
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
	if err != nil {
		return err
	}

	planContext := this.PlanContext()
	if planContext != nil {
		if this.limit != nil {
			_, err = planContext.Map(this.limit)
			if err != nil {
				return err
			}
		}
		planContext.addKeyspaceAlias(this.term.Alias())
	}

	return nil
}

func (this *SendUpdate) verify(prepared *Prepared) errors.Error {
	var err errors.Error

	this.keyspace, err = verifyKeyspace(this.keyspace, prepared)
	return err
}

func (this *SendUpdate) keyspaceReferences(prepared *Prepared) {
	prepared.addKeyspaceReference(this.keyspace)
}
