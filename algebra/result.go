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
	"fmt"
	"strconv"

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
Represents the select clause. Type Projection is a struct
that contains fields mapping to each expression in the
select clause. Distinct and raw are boolean values that
represent if the keywords DISTINCT and RAW are used in the
query. Terms represent the result expression.
*/
type Projection struct {
	distinct bool        `json:"distinct"`
	raw      bool        `json:"raw"`
	terms    ResultTerms `json:"terms"`
	estSize  int64       `json:"est_size"`
}

/*
The function NewProjection returns a pointer to the Projection
struct by assigning the input attributes to the fields of the
struct, and setting raw to false. This is for select clauses
without the RAW keyword specified. Call setAliases() to set
the alias string.
*/
func NewProjection(distinct bool, terms ResultTerms) *Projection {
	rv := &Projection{
		distinct: distinct,
		raw:      false,
		terms:    terms,
	}

	rv.setAliases()
	return rv
}

/*
The function NewRawProjection returns a pointer to the Projection
struct by assigning the input attributes to the fields of the
struct, and setting raw to true. This is for select clauses with
the RAW keyword specified. Call setAliases() to set the alias
string.
*/
func NewRawProjection(distinct bool, expr expression.Expression, as string) *Projection {
	rv := &Projection{
		distinct: distinct,
		raw:      true,
		terms:    ResultTerms{NewResultTerm(expr, false, as)},
	}

	rv.setAliases()
	return rv
}

/*
Returns the shapeof the result expression. If raw is true
return the first expression type as string value, as the
signature. If raw is false, then create a map, range over
the result terms and check if star is set to true to set
the alias key to the the expression type. Return this map.
*/
func (this *Projection) Signature() value.Value {
	if this.raw {
		return value.NewValue(this.terms[0].expr.Type().String())
	}

	rv := value.NewValue(make(map[string]interface{}, len(this.terms)))
	for _, term := range this.terms {
		if term.star {
			rv.SetField("*", "*")
		} else {
			rv.SetField(term.alias, term.expr.Type().String())
		}
	}

	return rv
}

/*
This method maps the result expressions.
*/
func (this *Projection) MapExpressions(mapper expression.Mapper) (err error) {
	for _, term := range this.terms {
		err = term.MapExpression(mapper)
		if err != nil {
			return
		}
	}

	return
}

/*
   Returns all contained Expressions.
*/
func (this *Projection) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, len(this.terms))

	for _, term := range this.terms {
		if term.expr != nil {
			exprs = append(exprs, term.expr)
		}
	}

	return exprs
}

/*
   Representation as a N1QL string.
*/
func (this *Projection) String() string {
	s := ""

	if this.distinct {
		s += "distinct "
	}

	if this.raw {
		s += "raw "
	}

	for i, term := range this.terms {
		if i > 0 {
			s += ", "
		}

		s += term.String()
	}

	return s
}

/*
This method fully qualifies the identifiers for each term
in the result expression. It disallows duplicate alias and
exempts explicit aliases from being formalized.
*/
func (this *Projection) Formalize(in *expression.Formalizer) (f *expression.Formalizer, err error) {
	// Disallow duplicate aliases
	aliases := make(map[string]bool, len(this.terms))
	for _, term := range this.terms {
		if term.alias == "" {
			continue
		}

		if aliases[term.alias] {
			return nil, fmt.Errorf("Duplicate result alias %s.", term.alias)
		}

		aliases[term.alias] = true
	}

	err = this.MapExpressions(in)
	if err != nil {
		return
	}

	if len(aliases) > 0 {
		f = in.Copy()
	} else {
		f = in
	}

	// Exempt explicit aliases from being formalized
	for _, term := range this.terms {
		if term.as != "" {
			f.SetAllowedAlias(term.as, false)
		}
	}

	return
}

/*
Return true if select clause in the query contains the
distinct keyword.
*/
func (this *Projection) Distinct() bool {
	return this.distinct
}

/*
Return true if select clause in the query contains the
raw keyword.
*/
func (this *Projection) Raw() bool {
	return this.raw
}

/*
Set the raw value
*/
func (this *Projection) SetRaw(raw bool) {
	this.raw = raw
}

/*
Return the result expression terms.
*/
func (this *Projection) Terms() ResultTerms {
	return this.terms
}

/*
Return the estimated result size.
*/
func (this *Projection) EstSize() int64 {
	return this.estSize
}

/*
Set the estimated result size.
*/
func (this *Projection) SetEstSize(estSize int64) {
	this.estSize = estSize
}

/*
Set the result term alias by calling setAlias for
each term.
*/
func (this *Projection) setAliases() {
	a := 1
	for _, term := range this.terms {
		a = term.setAlias(a)
	}
}

/*
Marshal input into byte array.
*/
func (this *Projection) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "projection"}
	r["distinct"] = this.distinct
	r["raw"] = this.raw
	r["terms"] = this.terms
	return json.Marshal(r)
}

/*
Type ResultTerms represents multiple ResultTerm
(result expressions).
*/
type ResultTerms []*ResultTerm

/*
This represents the result expression in a select clause.
Type ResultTerm is a struct that contains fields mapping
to the different expressions in the result-expr. The
expr maps to the input expression, star is a boolean
value that is true when * is used in the path. Both as
and alias represent the result expression alias. The
alias string is the path (a.b, alias = b) if no AS clause
is present, and if an alias is defined using the AS
clause in the result expr both alias and as are the
defined alias.
*/
type ResultTerm struct {
	expr  expression.Expression `json:"expr"`
	star  bool                  `json:"star"`
	as    string                `json:"as"`
	alias string                `json:"_"`
}

/*
The function NewResultTerm returns a pointer to the
ResultTerm struct by assigning the input attributes
to the fields of the struct. The value of alias string
is not set here.
*/
func NewResultTerm(expr expression.Expression, star bool, as string) *ResultTerm {
	if expr == nil {
		expr = expression.SELF
	}

	return &ResultTerm{
		expr: expr,
		star: star,
		as:   as,
	}
}

/*
Map the input expression of the result expr.
*/
func (this *ResultTerm) MapExpression(mapper expression.Mapper) (err error) {
	if this.expr != nil {
		this.expr, err = mapper.Map(this.expr)
	}

	return
}

/*
   Representation as a N1QL string.
*/
func (this *ResultTerm) String() string {
	s := ""

	if this.expr != nil {
		s = this.expr.String()
	}

	if this.star {
		if s == "" {
			s = "*"
		} else {
			s += ".*"
		}
	}

	if this.as != "" {
		s += " as `" + this.as + "`"
	}

	return s
}

/*
Return the input expression.
*/
func (this *ResultTerm) Expression() expression.Expression {
	return this.expr
}

/*
Return boolean value based on the presence
of * in the result expr.
*/
func (this *ResultTerm) Star() bool {
	return this.star
}

/*
Return the alias string defined by AS if present.
*/
func (this *ResultTerm) As() string {
	return this.as
}

/*
Return the alias string.
*/
func (this *ResultTerm) Alias() string {
	return this.alias
}

/*
Set the terms alias string. If star is true then
return the input integer as is. If the as string
is not empty set alias to that value, and if it
is then set it to the expr Alias (path). If the
expression isnt nil and the alias string is empty
then set the alias to "$a", where a represents
the input integer.
*/
func (this *ResultTerm) setAlias(a int) int {
	if this.star {
		return a
	}

	if this.as != "" {
		this.alias = this.as
	} else {
		this.alias = this.expr.Alias()
	}

	if this.expr != nil && this.alias == "" {
		this.alias = "$" + strconv.Itoa(a)
		a++
	}

	return a

}

/*
Marshal input ResultTerm into byte array.
*/
func (this *ResultTerm) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "resultTerm"}
	r["alias"] = this.alias
	r["as"] = this.as
	if this.expr != nil {
		r["expr"] = expression.NewStringer().Visit(this.expr)
	}
	r["star"] = this.star
	return json.Marshal(r)
}
