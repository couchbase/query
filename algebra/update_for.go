//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"encoding/json"

	"github.com/couchbase/query/expression"
)

/*
Represents the FOR clause in UPDATE SET/UNSET.
*/
type UpdateFor struct {
	bindings []expression.Bindings `json:"bindings"`
	when     expression.Expression `json:"when"`
}

func NewUpdateFor(bindings []expression.Bindings, when expression.Expression) *UpdateFor {
	return &UpdateFor{bindings, when}
}

/*
Apply mapper to expressions in the WHEN clause and bindings.
*/
func (this *UpdateFor) MapExpressions(mapper expression.Mapper) (err error) {
	for _, b := range this.bindings {
		err = b.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.when != nil {
		this.when, err = mapper.Map(this.when)
		if err != nil {
			return
		}
	}

	return
}

/*
Returns all contained Expressions.
*/
func (this *UpdateFor) Expressions() expression.Expressions {
	exprs := this.bindings[0].Expressions()
	for _, b := range this.bindings[1:] {
		exprs = append(exprs, b.Expressions()...)
	}

	if this.when != nil {
		exprs = append(exprs, this.when)
	}

	return exprs
}

/*
Returns the expression bindings for the UPDATE-FOR clause.
*/
func (this *UpdateFor) Bindings() []expression.Bindings {
	return this.bindings
}

/*
Returns the when expression for the WHEN clause in the
UPDATE-FOR clause.
*/
func (this *UpdateFor) When() expression.Expression {
	return this.when
}

/*
Marshals input into byte array.
*/
func (this *UpdateFor) MarshalJSON() ([]byte, error) {
	r := make(map[string]interface{}, 2)
	r["bindings"] = this.bindings
	if this.when != nil {
		r["when"] = this.when.String()
	}

	return json.Marshal(r)
}
