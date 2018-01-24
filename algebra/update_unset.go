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
Represents the UNSET clause in the UPDATE statement.
*/
type Unset struct {
	terms UnsetTerms
}

func NewUnset(terms UnsetTerms) *Unset {
	return &Unset{terms}
}

/*
Applies mapper to all the terms in the UnsetTerms.
*/
func (this *Unset) MapExpressions(mapper expression.Mapper) (err error) {
	for _, term := range this.terms {
		err = term.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	return
}

/*
Returns all contained Expressions.
*/
func (this *Unset) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 16)
	for _, term := range this.terms {
		exprs = append(exprs, term.Expressions()...)
	}

	return exprs
}

/*
Fully qualify identifiers for each term in the Unset terms.
*/
func (this *Unset) Formalize(f *expression.Formalizer) (err error) {
	for _, term := range this.terms {
		err = term.Formalize(f)
		if err != nil {
			return
		}
	}

	return
}

/*
Returns the terms in the UNSET clause defined by
UnsetTerms.
*/
func (this *Unset) Terms() UnsetTerms {
	return this.terms
}

type UnsetTerms []*UnsetTerm

type UnsetTerm struct {
	path      expression.Path `json:"path"`
	updateFor *UpdateFor      `json:"path_for"`
}

func NewUnsetTerm(path expression.Path, updateFor *UpdateFor) *UnsetTerm {
	return &UnsetTerm{path, updateFor}
}

/*
Applies mapper to the path expressions and update-for in
the unset Term.
*/
func (this *UnsetTerm) MapExpressions(mapper expression.Mapper) (err error) {
	path, err := mapper.Map(this.path)
	if err != nil {
		return err
	}

	this.path = path.(expression.Path)

	if this.updateFor != nil {
		err = this.updateFor.MapExpressions(mapper)
	}

	return
}

/*
Returns all contained Expressions.
*/
func (this *UnsetTerm) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 8)
	exprs = append(exprs, this.path)

	if this.updateFor != nil {
		exprs = append(exprs, this.updateFor.Expressions()...)
	}

	return exprs
}

/*
Fully qualify identifiers for the update-for clause and the path
expression in the unset clause.
*/
func (this *UnsetTerm) Formalize(f *expression.Formalizer) (err error) {
	if this.updateFor != nil {
		for _, b := range this.updateFor.bindings {
			err := f.PushBindings(b, true)
			if err != nil {
				return err
			}

			defer f.PopBindings()
		}

		if this.updateFor.when != nil {
			this.updateFor.when, err = f.Map(this.updateFor.when)
			if err != nil {
				return err
			}
		}
	}

	path, err := f.Map(this.path)
	if err != nil {
		return err
	}

	this.path = path.(expression.Path)
	return
}

/*
Returns the path expression in the UNSET clause.
*/
func (this *UnsetTerm) Path() expression.Path {
	return this.path
}

/*
Returns the update-for clause in the UNSET clause.
*/
func (this *UnsetTerm) UpdateFor() *UpdateFor {
	return this.updateFor
}

/*
Marshals input into byte array.
*/
func (this *UnsetTerm) MarshalJSON() ([]byte, error) {
	r := make(map[string]interface{}, 2)
	r["path"] = expression.NewStringer().Visit(this.path)
	if this.updateFor != nil {
		r["path_for"] = this.updateFor
	}

	return json.Marshal(r)
}
