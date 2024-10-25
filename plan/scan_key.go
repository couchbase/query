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

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

// KeyScan is used for USE KEYS clauses.
type KeyScan struct {
	readonly
	optEstimate
	keys     expression.Expression
	distinct bool
}

func NewKeyScan(keys expression.Expression, distinct bool, cost, cardinality float64,
	size int64, frCost float64) *KeyScan {
	keys.SetExprFlag(expression.EXPR_CAN_FLATTEN)
	rv := &KeyScan{
		keys:     keys,
		distinct: distinct,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
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

func (this *KeyScan) Distinct() bool {
	return this.distinct
}

func (this *KeyScan) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *KeyScan) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "KeyScan"}
	r["keys"] = this.keys.String()
	if this.distinct {
		r["distinct"] = this.distinct
	}
	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}
	if f != nil {
		f(r)
	}
	return r
}

func (this *KeyScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string                 `json:"#operator"`
		Keys        string                 `json:"keys"`
		Distinct    bool                   `json:"distinct"`
		OptEstimate map[string]interface{} `json:"optimizer_estimates"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.Keys != "" {
		this.keys, err = parser.Parse(_unmarshalled.Keys)
		if this.keys != nil {
			this.keys.SetExprFlag(expression.EXPR_CAN_FLATTEN)
		}
		if err != nil {
			return err
		}
	}

	this.distinct = _unmarshalled.Distinct

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	return nil
}
