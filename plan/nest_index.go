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

type IndexNest struct {
	readonly
	optEstimate
	keyspace datastore.Keyspace
	term     *algebra.KeyspaceTerm
	outer    bool
	keyFor   string
	subPaths []string
	idExpr   expression.Expression
	index    datastore.Index
	indexer  datastore.Indexer
}

func NewIndexNest(keyspace datastore.Keyspace, nest *algebra.IndexNest, index datastore.Index,
	subPaths []string, cost, cardinality float64, size int64, frCost float64) *IndexNest {
	rv := &IndexNest{
		keyspace: keyspace,
		term:     nest.Right(),
		outer:    nest.Outer(),
		keyFor:   nest.For(),
		index:    index,
		indexer:  index.Indexer(),
		subPaths: subPaths,
	}

	rv.idExpr = expression.NewField(
		expression.NewMeta(expression.NewIdentifier(rv.keyFor)),
		expression.NewFieldName("id", false))
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *IndexNest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexNest(this)
}

func (this *IndexNest) New() Operator {
	return &IndexNest{}
}

func (this *IndexNest) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *IndexNest) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *IndexNest) Outer() bool {
	return this.outer
}

func (this *IndexNest) For() string {
	return this.keyFor
}

func (this *IndexNest) IdExpr() expression.Expression {
	return this.idExpr
}

func (this *IndexNest) Index() datastore.Index {
	return this.index
}

func (this *IndexNest) SubPaths() []string {
	return this.subPaths
}

func (this *IndexNest) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IndexNest) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "IndexNest"}
	this.term.MarshalKeyspace(r)
	r["on_key"] = this.term.JoinKeys().String()
	r["for"] = this.keyFor
	if len(this.subPaths) > 0 {
		r["subpaths"] = this.subPaths
	}

	if this.outer {
		r["outer"] = this.outer
	}

	if this.term.As() != "" {
		r["as"] = this.term.As()
	}

	scan := map[string]interface{}{
		"index":    this.index.Name(),
		"index_id": this.index.Id(),
		"using":    this.index.Type(),
	}

	r["scan"] = scan

	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}

	if this.term.ValidateKeys() {
		r["validate_keys"] = true
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *IndexNest) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string `json:"#operator"`
		Namespace string `json:"namespace"`
		Bucket    string `json:"bucket"`
		Scope     string `json:"scope"`
		Keyspace  string `json:"keyspace"`
		On        string `json:"on_key"`
		Outer     bool   `json:"outer"`
		As        string `json:"as"`
		For       string `json:"for"`
		Scan      struct {
			Index   string              `json:"index"`
			IndexId string              `json:"index_id"`
			Using   datastore.IndexType `json:"using"`
		} `json:"scan"`
		SubPaths     []string               `json:"subpaths"`
		OptEstimate  map[string]interface{} `json:"optimizer_estimates"`
		ValidateKeys bool                   `json:"validate_keys"`
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
	this.keyFor = _unmarshalled.For
	this.subPaths = _unmarshalled.SubPaths
	this.idExpr = expression.NewField(
		expression.NewMeta(expression.NewIdentifier(this.keyFor)),
		expression.NewFieldName("id", false))
	this.term = algebra.NewKeyspaceTermFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), _unmarshalled.As, nil, nil)
	this.term.SetJoinKeys(keys_expr)
	this.term.SetValidateKeys(_unmarshalled.ValidateKeys)
	this.keyspace, err = datastore.GetKeyspace(this.term.Path().Parts()...)
	if err != nil {
		return err
	}

	this.indexer, err = this.keyspace.Indexer(_unmarshalled.Scan.Using)
	if err != nil {
		return err
	}

	this.index, err = this.indexer.IndexById(_unmarshalled.Scan.IndexId)
	if err != nil {
		return err
	}

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	planContext := this.PlanContext()
	if planContext != nil {
		if this.term.JoinKeys() != nil {
			_, err = planContext.Map(this.term.JoinKeys())
			if err != nil {
				return err
			}
		}
		planContext.addKeyspaceAlias(this.term.Alias())
	}

	return nil
}

func (this *IndexNest) verify(prepared *Prepared) bool {
	return verifyIndex(this.index, this.indexer, nil, prepared)
}
