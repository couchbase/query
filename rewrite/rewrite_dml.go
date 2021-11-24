//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package rewrite

import (
	"github.com/couchbase/query/algebra"
)

func (this *Rewrite) VisitSelect(stmt *algebra.Select) (r interface{}, err error) {
	windowTerms := this.windowTerms

	defer func() {
		this.windowTerms = windowTerms
	}()

	sel, ok := stmt.Subresult().(*algebra.Subselect)
	if ok {
		this.windowTerms = sel.Window()
	} else {
		this.windowTerms = nil
	}

	if this.windowTerms != nil {
		if err = this.windowTerms.ValidateWindowTerms(); err != nil {
			return stmt, err
		}
	}

	if r, err = stmt.Subresult().Accept(this); err != nil {
		return r, err
	}

	if stmt.Order() != nil {
		if err = stmt.Order().MapExpressions(this); err != nil {
			return stmt, err
		}
	}

	if stmt.Offset() != nil {
		if _, err = this.Map(stmt.Offset()); err != nil {
			return stmt, err
		}
	}

	if stmt.Limit() != nil {
		if _, err = this.Map(stmt.Limit()); err != nil {
			return stmt, err
		}
	}

	if ok {
		sel.ResetWindow()
	}

	return stmt, nil
}

func (this *Rewrite) VisitKeyspaceTerm(node *algebra.KeyspaceTerm) (interface{}, error) {
	return node, node.MapExpressions(this)
}

func (this *Rewrite) VisitExpressionTerm(node *algebra.ExpressionTerm) (interface{}, error) {
	if node.IsKeyspace() {
		return node.KeyspaceTerm().Accept(this)
	}
	return node.ExpressionTerm().Accept(this)
}

func (this *Rewrite) VisitSubqueryTerm(node *algebra.SubqueryTerm) (interface{}, error) {
	return node.Subquery().Accept(this)
}

func (this *Rewrite) VisitInsert(stmt *algebra.Insert) (interface{}, error) {
	if stmt.Select() != nil {
		if r, err := stmt.Select().Accept(this); err != nil {
			return r, err
		}
	}

	return stmt, stmt.MapExpressionsNoSelect(this)
}

func (this *Rewrite) VisitUpsert(stmt *algebra.Upsert) (interface{}, error) {
	if stmt.Select() != nil {
		if r, err := stmt.Select().Accept(this); err != nil {
			return r, err
		}
	}

	return stmt, stmt.MapExpressionsNoSelect(this)
}

func (this *Rewrite) VisitDelete(stmt *algebra.Delete) (r interface{}, err error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitUpdate(stmt *algebra.Update) (r interface{}, err error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitMerge(stmt *algebra.Merge) (r interface{}, err error) {
	source := stmt.Source()
	if source.SubqueryTerm() != nil {
		r, err = source.SubqueryTerm().Accept(this)
	} else if source.ExpressionTerm() != nil {
		r, err = source.ExpressionTerm().Accept(this)
	} else if source.From() != nil {
		r, err = source.From().Accept(this)
	}

	if err != nil {
		return stmt, err
	}

	return stmt, stmt.MapExpressions(this)
}
