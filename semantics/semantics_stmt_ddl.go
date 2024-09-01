//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package semantics

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

func (this *SemChecker) VisitCreatePrimaryIndex(stmt *algebra.CreatePrimaryIndex) (interface{}, error) {
	if stmt.Using() == datastore.FTS {
		return nil, errors.NewIndexNotAllowed("Primary index with USING FTS", "")
	}
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitCreateIndex(stmt *algebra.CreateIndex) (interface{}, error) {
	gsi := stmt.Using() == datastore.GSI || stmt.Using() == datastore.DEFAULT
	if !gsi && stmt.Partition() != nil {
		return nil, errors.NewIndexNotAllowed("PARTITION BY USING FTS", "")
	}

	for _, expr := range stmt.Expressions() {
		if !expr.Indexable() || expr.Value() != nil {
			return nil, errors.NewCreateIndexNotIndexable(expr.String(), expr.ErrorContext())
		}
	}

	nvectors := 0
	nkeys := 0
	for i, term := range stmt.Keys() {
		expr := term.Expression()
		if _, ok := expr.(*expression.Self); ok {
			return nil, errors.NewCreateIndexSelf(expr.String(), expr.ErrorContext())
		}
		all, ok := expr.(*expression.All)
		if !gsi {
			if term.HasAttribute(algebra.IK_MISSING | algebra.IK_ASC | algebra.IK_DESC | algebra.IK_VECTOR) {
				return nil, errors.NewIndexNotAllowed("Index attributes USING FTS", "")
			} else if ok {
				return nil, errors.NewIndexNotAllowed("Array Index USING FTS", "")
			}
		} else if term.HasAttribute(algebra.IK_VECTOR) {
			nvectors++
			indexKey := expr.String()
			if term.HasAttribute(algebra.IK_MISSING) {
				return nil, errors.NewVectorIndexAttrError("INCLUDE MISSING", indexKey)
			}
			if term.HasAttribute(algebra.IK_DESC) {
				return nil, errors.NewVectorIndexAttrError("DESC", indexKey)
			}
			if ok && all.Distinct() {
				return nil, errors.NewVectorDistinctArrayKey()
			}
			switch expr.(type) {
			case *expression.ObjectConstruct, *expression.ArrayConstruct:
				return nil, errors.NewVectorConstantIndexKey(expr.String())
			}
		}

		if ok && all.Flatten() {
			if term.Attributes() != 0 {
				return nil, errors.NewCreateIndexAttribute(expr.String(), expr.ErrorContext())
			}

			fk := all.FlattenKeys()
			for pos, fke := range fk.Operands() {
				nkeys++
				if !fke.Indexable() || fke.Value() != nil {
					return nil, errors.NewCreateIndexNotIndexable(fke.String(), fke.ErrorContext())
				}
				if fk.HasMissing(pos) && (i > 0 || pos > 0 || !gsi) {
					return nil, errors.NewCreateIndexAttributeMissing(fke.String(), fke.ErrorContext())
				}
				if fk.HasVector(pos) {
					return nil, errors.NewIndexNotAllowed("Array Index with FLATTEN_KEYS using Vector Index Key", fke.String())
				}
			}
		} else {
			nkeys++
			if term.HasAttribute(algebra.IK_VECTOR) && ok {
				return nil, errors.NewIndexNotAllowed("Array Index using Vector Index Key", expr.String())
			}

		}
		if term.HasAttribute(algebra.IK_MISSING) && (i > 0 || !gsi) {
			return nil, errors.NewCreateIndexAttributeMissing(expr.String(), expr.ErrorContext())
		}
		if nvectors > 1 {
			return nil, errors.NewVectorIndexSingleVector(stmt.Name())
		}
	}

	if gsi && stmt.Vector() {
		if nkeys > 1 {
			return nil, errors.NewVectorIndexSingleKey(stmt.Name())
		}
		if nvectors == 0 {
			return nil, errors.NewVectorIndexNoVector(stmt.Name())
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

func (this *SemChecker) VisitCreateSequence(stmt *algebra.CreateSequence) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitDropSequence(stmt *algebra.DropSequence) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitAlterSequence(stmt *algebra.AlterSequence) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}
