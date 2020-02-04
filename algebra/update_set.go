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
Represents the SET clause in the UPDATE statement.
*/
type Set struct {
	terms SetTerms
}

func NewSet(terms SetTerms) *Set {
	return &Set{terms}
}

/*
Applies mapper to all the terms in the setTerms.
*/
func (this *Set) MapExpressions(mapper expression.Mapper) (err error) {
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
func (this *Set) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 16)
	for _, term := range this.terms {
		exprs = append(exprs, term.Expressions()...)
	}

	return exprs
}

func (this *Set) NonMutatedExpressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 16)
	for _, term := range this.terms {
		exprs = append(exprs, term.NonMutatedExpressions()...)
	}

	return exprs
}

/*
Fully qualify identifiers for each term in the set terms.
*/
func (this *Set) Formalize(f *expression.Formalizer) (err error) {
	for _, term := range this.terms {
		err = term.Formalize(f)
		if err != nil {
			return
		}
	}

	return
}

/*
Returns the terms in the SET clause defined by
setTerms.
*/
func (this *Set) Terms() SetTerms {
	return this.terms
}

type SetTerms []*SetTerm

type SetTerm struct {
	meta      expression.Expression `json:"meta"`
	path      expression.Path       `json:"path"`
	value     expression.Expression `json:"value"`
	updateFor *UpdateFor            `json:"path_for"`
}

func NewSetTerm(path expression.Path, value expression.Expression, updateFor *UpdateFor,
	meta expression.Expression) *SetTerm {
	return &SetTerm{meta, path, value, updateFor}
}

var _MUTATE_META_PATHS = []string{"expiration"}

func IsValidMetaMutatePath(path expression.Expression) bool {
	if alias, path, err := expression.PathString(path); err == nil && path == "" {
		for _, s := range _MUTATE_META_PATHS {
			if s == alias {
				return true
			}
		}
	}
	return false
}

/*
Applies mapper to the path and value expressions, and update-for
in the set term.
*/
func (this *SetTerm) MapExpressions(mapper expression.Mapper) (err error) {
	if this.meta != nil {
		this.meta, err = mapper.Map(this.meta)
		if err != nil {
			return err
		}
	}

	path, err1 := mapper.Map(this.path)
	if err1 != nil {
		return err1
	}

	this.path = path.(expression.Path)
	this.value, err = mapper.Map(this.value)
	if err != nil {
		return
	}

	if this.updateFor != nil {
		err = this.updateFor.MapExpressions(mapper)
	}

	return
}

/*
Returns all contained Expressions.
*/
func (this *SetTerm) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 8)
	exprs = append(exprs, this.path, this.value)

	if this.updateFor != nil {
		exprs = append(exprs, this.updateFor.Expressions()...)
	}

	return exprs
}

func (this *SetTerm) NonMutatedExpressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 8)
	exprs = append(exprs, this.value)

	if this.updateFor != nil {
		exprs = append(exprs, this.updateFor.Expressions()...)
	}

	return exprs
}

/*
Fully qualify identifiers for the update-for clause, the path
and value expressions in the SET clause.
*/
func (this *SetTerm) Formalize(f *expression.Formalizer) (err error) {
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

	if this.meta != nil {
		// if meta is present don't formalize the path
		this.meta, err = f.Map(this.meta)
		if err != nil {
			return err
		}
	} else {
		path, err := f.Map(this.path)
		if err != nil {
			return err
		}
		this.path = path.(expression.Path)
	}

	this.value, err = f.Map(this.value)
	return
}

/*
Returns the path expression in the SET clause.
*/
func (this *SetTerm) Path() expression.Path {
	return this.path
}

/*
Returns the Meta portion of expression in the SET clause.
*/
func (this *SetTerm) Meta() expression.Expression {
	return this.meta
}

/*
Returns the value expression in the SET clause.
*/
func (this *SetTerm) Value() expression.Expression {
	return this.value
}

/*
Returns the update-for clause in the SET clause.
*/
func (this *SetTerm) UpdateFor() *UpdateFor {
	return this.updateFor
}

/*
Marshals input into byte array.
*/
func (this *SetTerm) MarshalJSON() ([]byte, error) {
	r := make(map[string]interface{}, 3)
	stringer := expression.NewStringer()
	if this.meta != nil {
		r["meta"] = stringer.Visit(this.meta)
	}
	r["path"] = stringer.Visit(this.path)
	r["value"] = stringer.Visit(this.value)
	if this.updateFor != nil {
		r["path_for"] = this.updateFor
	}

	return json.Marshal(r)
}
