//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"github.com/couchbase/query/expression"
)

func ReplaceProjectionAlias(exprs expression.Expressions, projection *Projection) (
	expression.Expressions, bool, error) {

	newExprs := make(expression.Expressions, len(exprs))
	found := false
	paReplacer := newProjAliasReplacer(projection)
	for i, expr := range exprs {
		paReplacer.found = false
		newExpr, err := paReplacer.Map(expr.Copy())
		if err != nil {
			return nil, false, err
		}
		newExprs[i] = newExpr
		found = found || paReplacer.found
	}

	return newExprs, found, nil
}

type projAliasReplacer struct {
	expression.MapperBase

	projection *Projection
	found      bool
}

func newProjAliasReplacer(projection *Projection) *projAliasReplacer {
	rv := &projAliasReplacer{
		projection: projection,
	}

	rv.SetMapper(rv)
	return rv
}

func (this *projAliasReplacer) VisitIdentifier(expr *expression.Identifier) (interface{}, error) {
	if expr.IsProjectionAlias() {
		this.found = true
		if this.projection != nil {
			for _, term := range this.projection.terms {
				if expr.Identifier() == term.alias {
					return term.expr, nil
				}
			}
		}
	}

	return expr, nil
}
