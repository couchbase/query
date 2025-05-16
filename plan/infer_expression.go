//  Copyright 2021-Present Couchbase, Inc.
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
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/value"
)

type InferExpression struct {
	execution
	node *algebra.InferExpression
}

func NewInferExpression(node *algebra.InferExpression) *InferExpression {
	return &InferExpression{
		node: node,
	}
}

func (this *InferExpression) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInferExpression(this)
}

func (this *InferExpression) New() Operator {
	return &InferExpression{}
}

func (this *InferExpression) Node() *algebra.InferExpression {
	return this.node
}

func (this *InferExpression) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *InferExpression) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "InferExpression"}
	r["expr"] = this.node.Expression()
	r["using"] = this.node.Using()
	if this.node.With() != nil {
		r["with"] = this.node.With()
	}
	if f != nil {
		f(r)
	}
	return r
}

func (this *InferExpression) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string                  `json:"#operator"`
		Expr  string                  `json:"expr"`
		Using datastore.InferenceType `json:"using"`
		With  json.RawMessage         `json:"with"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	expr, err := parser.Parse(_unmarshalled.Expr)
	if err != nil || len(_unmarshalled.Expr) == 0 {
		return err
	}

	var with value.Value
	if len(_unmarshalled.With) > 0 {
		with = value.NewValue([]byte(_unmarshalled.With))
	}

	this.node = algebra.NewInferExpression(expr, _unmarshalled.Using, with)
	return nil
}
