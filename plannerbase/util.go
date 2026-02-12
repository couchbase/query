//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plannerbase

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

func GetStaticInt(expr expression.Expression) (int64, bool) {
	if expr != nil {
		expVal := expr.Value()
		if expVal != nil {
			switch evt := expVal.Actual().(type) {
			case float64:
				return int64(evt), true
			}
		}
	}

	return 0, false
}

func ReplaceParameters(pred expression.Expression, namedArgs map[string]value.Value,
	positionalArgs value.Values) (expression.Expression, error) {

	if pred == nil || (len(namedArgs) == 0 && len(positionalArgs) == 0) {
		return pred, nil
	}

	var err error
	var replaced, repl bool

	pred = pred.Copy()

	for name, value := range namedArgs {
		nameExpr := algebra.NewNamedParameter(name)
		valueExpr := expression.NewConstant(value)
		pred, repl, err = expression.ReplaceExpr(pred, nameExpr, valueExpr)
		if err != nil {
			return nil, err
		}
		replaced = replaced || repl
	}

	for pos, value := range positionalArgs {
		posExpr := algebra.NewPositionalParameter(pos + 1)
		valueExpr := expression.NewConstant(value)
		pred, repl, err = expression.ReplaceExpr(pred, posExpr, valueExpr)
		if err != nil {
			return nil, err
		}
		replaced = replaced || repl
	}

	if likeFunc, ok := pred.(expression.LikeFunction); ok && replaced {
		pred = likeFunc.Constructor()(likeFunc.Operands()...)
	}

	return pred, nil
}

func IsDerivedExpr(expr expression.Expression) bool {
	return expr != nil && expr.HasExprFlag(expression.EXPR_DERIVED_FROM_LIKE|expression.EXPR_DERIVED_RANGE1|
		expression.EXPR_DERIVED_RANGE2|expression.EXPR_DERIVED_FROM_ISOBJECT)
}

func IgnoreFilter(fl *Filter) bool {
	fltrExpr := fl.fltrExpr
	origExpr := fl.origExpr

	exprFlags := uint64(expression.EXPR_UNNEST_NOT_MISSING | expression.EXPR_UNNEST_ISARRAY | expression.EXPR_JOIN_NOT_NULL)
	if origExpr != nil && origExpr.HasExprFlag(exprFlags) {
		return true
	}

	return fltrExpr.HasExprFlag(exprFlags)
}
