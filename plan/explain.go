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
	op         Operator
	text       string
	optimHints *algebra.OptimHints
}

func NewExplain(op Operator, text string, optimHints *algebra.OptimHints) *Explain {
	return &Explain{
		op:         op,
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

func (this *Explain) Operator() Operator {
	return this.op
}

func (this *Explain) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Explain) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := make(map[string]interface{}, 2)
	r["text"] = this.text
	if this.op != nil {
		if this.op.Cost() > 0.0 {
			r["cost"] = this.op.Cost()
		}
		if this.op.Cardinality() > 0.0 {
			r["cardinality"] = this.op.Cardinality()
		}
	}
	if f != nil {
		f(r)
	} else {
		r["plan"] = this.op
		if this.optimHints != nil {
			r["optimizer_hints"] = this.optimHints
		}
	}
	return r
}

func (this *Explain) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Op          json.RawMessage `json:"plan"`
		Text        string          `json:"text"`
		Cost        float64         `json:"cost"`
		Cardinality float64         `json:"cardinality"`
		OptimHints  json.RawMessage `json:"optimizer_hints"`
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

	this.op, err = MakeOperator(op_type.Operator, _unmarshalled.Op)
	return err
}
