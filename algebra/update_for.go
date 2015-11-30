//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"encoding/json"

	"github.com/couchbase/query/expression"
)

/*
Represents the FOR clause in UPDATE SET/UNSET.
*/
type UpdateFor struct {
	bindings expression.Bindings   `json:"bindings"`
	when     expression.Expression `json:"when"`
}

func NewUpdateFor(bindings expression.Bindings, when expression.Expression) *UpdateFor {
	return &UpdateFor{bindings, when}
}

/*
Apply mapper to expressions in the WHEN clause and bindings.
*/
func (this *UpdateFor) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.bindings.MapExpressions(mapper)
	if err != nil {
		return
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
	exprs := this.bindings.Expressions()

	if this.when != nil {
		exprs = append(exprs, this.when)
	}

	return exprs
}

/*
Returns the expression bindings for the UPDATE-FOR clause.
*/
func (this *UpdateFor) Bindings() expression.Bindings {
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
		r["when"] = expression.NewStringer().Visit(this.when)
	}

	return json.Marshal(r)
}
