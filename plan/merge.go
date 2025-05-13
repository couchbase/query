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

type Merge struct {
	dml
	optEstimate
	keyspace     datastore.Keyspace
	ref          *algebra.KeyspaceRef
	key          expression.Expression
	canSpill     bool
	fastDiscard  bool
	limit        expression.Expression
	update       Operator
	updateFilter expression.Expression
	delete       Operator
	deleteFilter expression.Expression
	insert       Operator
	insertFilter expression.Expression
	let          Operator
}

func NewMerge(keyspace datastore.Keyspace, ref *algebra.KeyspaceRef, key expression.Expression,
	canSpill, fastDiscard bool, limit expression.Expression,
	update Operator, updateFilter expression.Expression,
	delete Operator, deleteFilter expression.Expression,
	insert Operator, insertFilter expression.Expression,
	cost, cardinality float64, size int64, frCost float64) *Merge {

	rv := &Merge{
		keyspace:     keyspace,
		ref:          ref,
		key:          key,
		canSpill:     canSpill,
		fastDiscard:  fastDiscard,
		limit:        limit,
		update:       update,
		updateFilter: updateFilter,
		delete:       delete,
		deleteFilter: deleteFilter,
		insert:       insert,
		insertFilter: insertFilter,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *Merge) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMerge(this)
}

func (this *Merge) New() Operator {
	return &Merge{}
}

func (this *Merge) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *Merge) KeyspaceRef() *algebra.KeyspaceRef {
	return this.ref
}

func (this *Merge) Key() expression.Expression {
	return this.key
}

func (this *Merge) IsOnKey() bool {
	return this.key != nil
}

func (this *Merge) CanSpill() bool {
	return this.canSpill
}

func (this *Merge) FastDiscard() bool {
	return this.fastDiscard
}

func (this *Merge) Limit() expression.Expression {
	return this.limit
}

func (this *Merge) Update() Operator {
	return this.update
}

func (this *Merge) Delete() Operator {
	return this.delete
}

func (this *Merge) Insert() Operator {
	return this.insert
}

func (this *Merge) UpdateFilter() expression.Expression {
	return this.updateFilter
}

func (this *Merge) DeleteFilter() expression.Expression {
	return this.deleteFilter
}

func (this *Merge) InsertFilter() expression.Expression {
	return this.insertFilter
}

func (this *Merge) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Merge) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Merge"}
	this.ref.MarshalKeyspace(r)

	if this.key != nil {
		r["key"] = this.key.String()
	}

	if this.ref.As() != "" {
		r["as"] = this.ref.As()
	}

	if this.canSpill {
		r["can_spill"] = this.canSpill
	}

	if this.fastDiscard {
		r["fast_discard"] = this.fastDiscard
	}

	if this.limit != nil {
		r["limit"] = this.limit
	}

	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}

	if f != nil {
		f(r)
	} else {
		if this.update != nil {
			r["update"] = this.update
		}
		if this.updateFilter != nil {
			r["update_filter"] = this.updateFilter
		}
		if this.delete != nil {
			r["delete"] = this.delete
		}
		if this.deleteFilter != nil {
			r["delete_filter"] = this.deleteFilter
		}
		if this.insert != nil {
			r["insert"] = this.insert
		}
		if this.insertFilter != nil {
			r["insert_filter"] = this.insertFilter
		}
	}
	return r
}

func (this *Merge) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_            string                 `json:"#operator"`
		Namespace    string                 `json:"namespace"`
		Bucket       string                 `json:"bucket"`
		Scope        string                 `json:"scope"`
		Keyspace     string                 `json:"keyspace"`
		As           string                 `json:"as"`
		Key          string                 `json:"key"`
		CanSpill     bool                   `json:"can_spill"`
		FastDiscard  bool                   `json:"fast_discard"`
		Limit        string                 `json:"limit"`
		Update       json.RawMessage        `json:"update"`
		Delete       json.RawMessage        `json:"delete"`
		Insert       json.RawMessage        `json:"insert"`
		UpdateFilter string                 `json:"update_filter"`
		DeleteFilter string                 `json:"delete_filter"`
		InsertFilter string                 `json:"insert_filter"`
		OptEstimate  map[string]interface{} `json:"optimizer_estimates"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.ref = algebra.NewKeyspaceRefFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), _unmarshalled.As)

	this.keyspace, err = datastore.GetKeyspace(this.ref.Path().Parts()...)
	if err != nil {
		return err
	}

	if _unmarshalled.Key != "" {
		this.key, err = parser.Parse(_unmarshalled.Key)
		if err != nil {
			return err
		}
	}

	this.canSpill = _unmarshalled.CanSpill
	this.fastDiscard = _unmarshalled.FastDiscard

	if _unmarshalled.Limit != "" {
		this.limit, err = parser.Parse(_unmarshalled.Limit)
		if err != nil {
			return err
		}
	}

	planContext := this.PlanContext()

	ops := []json.RawMessage{
		_unmarshalled.Update,
		_unmarshalled.Delete,
		_unmarshalled.Insert,
	}

	for i, child := range ops {
		if len(child) == 0 {
			continue
		}

		var op_type struct {
			Operator string `json:"#operator"`
		}

		err = json.Unmarshal(child, &op_type)
		if err != nil {
			return err
		}

		switch i {
		case 0:
			this.update, err = MakeOperator(op_type.Operator, child, planContext)
		case 1:
			this.delete, err = MakeOperator(op_type.Operator, child, planContext)
		case 2:
			this.insert, err = MakeOperator(op_type.Operator, child, planContext)
		}

		if err != nil {
			return err
		}
	}

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	if _unmarshalled.UpdateFilter != "" {
		if this.updateFilter, err = parser.Parse(_unmarshalled.UpdateFilter); err != nil {
			return err
		}
	}

	if _unmarshalled.DeleteFilter != "" {
		if this.deleteFilter, err = parser.Parse(_unmarshalled.DeleteFilter); err != nil {
			return err
		}
	}

	if _unmarshalled.InsertFilter != "" {
		if this.insertFilter, err = parser.Parse(_unmarshalled.InsertFilter); err != nil {
			return err
		}
	}

	if planContext != nil {
		if this.limit != nil {
			_, err = planContext.Map(this.limit)
			if err != nil {
				return err
			}
		}
		if this.key != nil {
			_, err = planContext.Map(this.key)
			if err != nil {
				return err
			}
			// legacy style, need to add target here
			planContext.addKeyspaceAlias(this.ref.Alias())
		}
		if this.updateFilter != nil {
			_, err = planContext.Map(this.updateFilter)
			if err != nil {
				return err
			}
		}
		if this.deleteFilter != nil {
			_, err = planContext.Map(this.deleteFilter)
			if err != nil {
				return err
			}
		}
		if this.insertFilter != nil {
			_, err = planContext.Map(this.insertFilter)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (this *Merge) verify(prepared *Prepared) bool {
	var result bool

	this.keyspace, result = verifyKeyspace(this.keyspace, prepared)
	if result && this.insert != nil {
		result = this.insert.verify(prepared)
	}
	if result && this.delete != nil {
		result = this.delete.verify(prepared)
	}
	if result && this.update != nil {
		result = this.update.verify(prepared)
	}
	return result
}
