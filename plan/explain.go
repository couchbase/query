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
)

type Explain struct {
	execution
	qp         *QueryPlan
	text       string
	optimHints *algebra.OptimHints
}

func NewExplain(qp *QueryPlan, text string, optimHints *algebra.OptimHints) *Explain {
	return &Explain{
		qp:         qp,
		text:       text,
		optimHints: optimHints,
	}
}

func (this *Explain) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExplain(this)
}

func (this *Explain) New() Operator {
	return &Explain{}
}

func (this *Explain) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Explain) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := make(map[string]interface{}, 2)
	r["text"] = this.text
	op := this.qp.op
	if op != nil {
		if op.Cost() > 0.0 {
			r["cost"] = op.Cost()
		}
		if op.Cardinality() > 0.0 {
			r["cardinality"] = op.Cardinality()
		}
	}
	r["plan"] = op
	if this.optimHints != nil {
		r["optimizer_hints"] = this.optimHints
	}
	subqueries := this.qp.subqueries
	if len(subqueries) > 0 {
		marshalledSubqueries := make([]map[string]interface{}, 0, len(subqueries))
		for t, s := range subqueries {
			subquery := map[string]interface{}{
				"subquery":   t.String(),
				"plan":       s,
				"correlated": t.IsCorrelated(),
			}
			optimHints := t.OptimHints()
			if optimHints != nil {
				subquery["optimizer_hints"] = optimHints
			}
			marshalledSubqueries = append(marshalledSubqueries, subquery)
		}
		r["~subqueries"] = marshalledSubqueries
	}
	if f != nil {
		r["#operator"] = "Explain"
		f(r)
	}
	return r
}

// Note that explain is never prepared nor distributed across nodes,
// so this code is never exercised - it might as well be a noop
func (this *Explain) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Op          json.RawMessage `json:"plan"`
		Text        string          `json:"text"`
		Cost        float64         `json:"cost"`
		Cardinality float64         `json:"cardinality"`
		OptimHints  json.RawMessage `json:"optimizer_hints"`
		Subqueries  json.RawMessage `json:"~subqueries"`
	}

	var op_type struct {
		Operator string `json:"#operator"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	err = json.Unmarshal(_unmarshalled.Op, &op_type)
	if err != nil {
		return err
	}

	this.text = _unmarshalled.Text

	err = json.Unmarshal(_unmarshalled.Op, &op_type)
	if err != nil {
		return err
	}

	// Cost/cardinality is included in the explain plan so it's easy to
	// see the overall cost/cardinality for the entire plan, there is
	// no need to put the info anywhere

	// Optimizer hints is printed in explain for informational purpose only

	// Subqueries is printed in explain for informational purposes only

	op, err := MakeOperator(op_type.Operator, _unmarshalled.Op)
	if err != nil {
		return err
	}
	this.qp = NewQueryPlan(op)
	return nil
}
