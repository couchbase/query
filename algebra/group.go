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
	"github.com/couchbase/query/expression"
)

/*
This represents the Group by clause. Type Group is a
struct that contains group by expression 'by', the
letting clause and the having clause represented by
expression bindings and expressions respectively.
Aliases in the LETTING clause create new names that
may be referred to in the HAVING, SELECT, and ORDER
BY clauses. Having specifies a condition.
*/
type Group struct {
	by      expression.Expressions `json:by`
	letting expression.Bindings    `json:"letting"`
	having  expression.Expression  `json:"having"`
}

/*
The function NewGroup returns a pointer to the Group
struct that has its field sort terms set to the input
argument expressions.
*/
func NewGroup(by expression.Expressions, letting expression.Bindings, having expression.Expression) *Group {
	return &Group{
		by:      by,
		letting: letting,
		having:  having,
	}
}

/*
This method qualifies identifiers for all the constituent clauses,
namely the by, letting and having expressions by mapping them.
*/
func (this *Group) Formalize(f *expression.Formalizer) error {
	var err error

	if this.by != nil {
		for i, b := range this.by {
			this.by[i], err = f.Map(b)
			if err != nil {
				return err
			}
		}
	}

	if this.letting != nil {
		_, err = f.PushBindings(this.letting, false)
		if err != nil {
			return err
		}
	}

	if this.having != nil {
		this.having, err = f.Map(this.having)
		if err != nil {
			return err
		}
	}

	return nil
}

/*
This method maps all the constituent clauses, namely the
by, letting and having within a group by clause.
*/
func (this *Group) MapExpressions(mapper expression.Mapper) (err error) {
	if this.by != nil {
		err = this.by.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.letting != nil {
		err = this.letting.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.having != nil {
		this.having, err = mapper.Map(this.having)
	}

	return
}

/*
   Returns all contained Expressions.
*/
func (this *Group) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 16)

	if this.by != nil {
		exprs = append(exprs, this.by...)
	}

	if this.letting != nil {
		exprs = append(exprs, this.letting.Expressions()...)
	}

	if this.having != nil {
		exprs = append(exprs, this.having)
	}

	return exprs
}

/*
   Representation as a N1QL string.
*/
func (this *Group) String() string {
	s := ""

	if this.by != nil {
		s += " group by "

		for i, b := range this.by {
			if i > 0 {
				s += ", "
			}

			s += b.String()
		}
	}

	if this.letting != nil {
		s += " letting " + stringBindings(this.letting)
	}

	if this.having != nil {
		s += " having " + this.having.String()
	}

	return s
}

/*
Returns the Group by expression.
*/
func (this *Group) By() expression.Expressions {
	return this.by
}

/*
Returns the letting expression bindings.
*/
func (this *Group) Letting() expression.Bindings {
	return this.letting
}

/*
Returns the having condition expression.
*/
func (this *Group) Having() expression.Expression {
	return this.having
}
