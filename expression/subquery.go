//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

/*
Used to implement subqueries. Type Subquery is
an interface that inherits from Expression. It
also implements a method Formalize that takes
as input a type Formalizer, and returns an error.
*/
type Subquery interface {
	Expression

	Formalize(parent *Formalizer) error
	IsCorrelated() bool
	GetCorrelation() map[string]uint32
}

/*
SubqueryLister is a Visitor for enumerating subqueries within an
expression tree.
*/
type SubqueryLister struct {
	TraverserBase

	subqueries []Subquery
	descend    bool
}

func NewSubqueryLister(descend bool) *SubqueryLister {
	rv := &SubqueryLister{
		subqueries: make([]Subquery, 0, 8),
		descend:    descend,
	}

	rv.traverser = rv
	return rv
}

func (this *SubqueryLister) VisitSubquery(expr Subquery) (interface{}, error) {
	this.subqueries = append(this.subqueries, expr)

	if this.descend {
		return nil, this.TraverseList(expr.Children())
	}

	return nil, nil
}

func (this *SubqueryLister) Subqueries() []Subquery {
	return this.subqueries
}

func ListSubqueries(exprs Expressions, descend bool) ([]Subquery, error) {
	lister := NewSubqueryLister(descend)

	for _, expr := range exprs {
		err := lister.Traverse(expr)
		if err != nil {
			return nil, err
		}
	}

	return lister.Subqueries(), nil
}

// Returns if the expression contains subqueries
func ContainsSubquery(expr Expression) bool {
	if _, ok := expr.(Subquery); ok {
		return true
	} else {
		for _, expr := range expr.Children() {
			if ContainsSubquery(expr) {
				return true
			}
		}
	}

	return false
}
