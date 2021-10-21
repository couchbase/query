//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package semantics

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

func (this *SemChecker) VisitCreatePrimaryIndex(stmt *algebra.CreatePrimaryIndex) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitCreateIndex(stmt *algebra.CreateIndex) (interface{}, error) {
	gsi := stmt.Using() == datastore.GSI || stmt.Using() == datastore.DEFAULT
	for _, expr := range stmt.Expressions() {
		if !expr.Indexable() || expr.Value() != nil {
			return nil, errors.NewCreateIndexNotIndexable(expr.String(), expr.ErrorContext())
		}
	}

	for i, term := range stmt.Keys() {
		expr := term.Expression()
		if all, ok := expr.(*expression.All); ok && all.Flatten() {
			if term.Attributes() != 0 {
				return nil, errors.NewCreateIndexAttribute(expr.String(), expr.ErrorContext())
			}

			fk := all.FlattenKeys()
			for pos, fke := range fk.Operands() {
				if !fke.Indexable() || fke.Value() != nil {
					return nil, errors.NewCreateIndexNotIndexable(fke.String(), fke.ErrorContext())
				}
				if fk.HasMissing(pos) && (i > 0 || pos > 0 || !gsi) {
					return nil, errors.NewCreateIndexAttributeMissing(fke.String(), fke.ErrorContext())
				}
			}
		}
		if term.HasAttribute(algebra.IK_MISSING) && (i > 0 || !gsi) {
			return nil, errors.NewCreateIndexAttributeMissing(expr.String(), expr.ErrorContext())
		}
	}

	if err := semCheckFlattenKeys(stmt.Expressions()); err != nil {
		return nil, err
	}

	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitDropIndex(stmt *algebra.DropIndex) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitAlterIndex(stmt *algebra.AlterIndex) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitBuildIndexes(stmt *algebra.BuildIndexes) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitCreateScope(stmt *algebra.CreateScope) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitDropScope(stmt *algebra.DropScope) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitCreateCollection(stmt *algebra.CreateCollection) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitDropCollection(stmt *algebra.DropCollection) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitFlushCollection(stmt *algebra.FlushCollection) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

type CheckFlattenKeys struct {
	expression.MapperBase
	flattenKeys expression.Expression
}

/* FLATTEN_KEYS() function allowed only in
   -   Array indexing key deepest value mapping
   -   Not surounded by any function
   -   No recursive
*/

func NewCheckFlattenKeys() *CheckFlattenKeys {
	rv := &CheckFlattenKeys{}
	rv.SetMapper(rv)
	rv.SetMapFunc(func(expr expression.Expression) (expression.Expression, error) {
		if _, ok := expr.(*expression.FlattenKeys); ok && rv.flattenKeys != expr {
			return expr, errors.NewFlattenKeys(expr.String(), expr.ErrorContext())
		}
		return expr, expr.MapChildren(rv)
	})
	return rv
}

func semCheckFlattenKeys(exprs expression.Expressions) (err error) {
	cfk := NewCheckFlattenKeys()
	for _, expr := range exprs {
		if all, ok := expr.(*expression.All); ok && all.Flatten() {
			cfk.flattenKeys = all.FlattenKeys()
		} else {
			cfk.flattenKeys = nil
		}

		if _, err = cfk.Map(expr); err != nil {
			return err
		}
	}

	return err
}
