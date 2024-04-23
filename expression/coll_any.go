//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"github.com/couchbase/query/value"
)

/*
Represents range predicate ANY, that allow testing of a bool condition
over the elements of a collection or object.
*/
type Any struct {
	collPredBase
}

func NewAny(bindings Bindings, satisfies Expression) Expression {
	rv := &Any{
		collPredBase: collPredBase{
			bindings:  bindings,
			satisfies: satisfies,
		},
	}

	rv.expr = rv
	return rv
}

func (this *Any) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAny(this)
}

func (this *Any) Type() value.Type { return value.BOOLEAN }

func (this *Any) Evaluate(item value.Value, context Context) (value.Value, error) {
	bvals, buffers, bpairs, n, missing, null, err := collEval(this.bindings, item, context)
	defer collReleaseBuffers(bvals, buffers, bpairs)
	if err != nil {
		return nil, err
	}

	if missing {
		return value.MISSING_VALUE, nil
	}

	if null {
		return value.NULL_VALUE, nil
	}

	for i := 0; i < n; i++ {
		cv := value.NewScopeValue(make(map[string]interface{}, len(this.bindings)), item)
		for j, b := range this.bindings {
			if b.NameVariable() == "" {
				cv.SetField(b.Variable(), bvals[j][i])
			} else {
				pair := bpairs[j][i]
				cv.SetField(b.NameVariable(), pair.Name)
				cv.SetField(b.Variable(), pair.Value)
			}
		}

		av := value.NewAnnotatedValue(cv)
		if ai, ok := item.(value.AnnotatedValue); ok {
			av.CopyAnnotations(ai)
		}

		sv, err := this.satisfies.Evaluate(av, context)
		av.Recycle()
		if err != nil {
			return nil, err
		}

		if sv.Truth() {
			return value.TRUE_VALUE, nil
		}
	}

	return value.FALSE_VALUE, nil
}

func (this *Any) CoveredBy(keyspace string, exprs Expressions, options CoveredOptions) Covered {

	nExprs := make(Expressions, 0, len(exprs))
	var all *All

	for _, expr := range exprs {
		if aexpr, ok := expr.(*All); ok {
			all = aexpr
		} else {
			nExprs = append(nExprs, expr)
		}
	}

	if options.hasCoverImplicitArrayKey() && all != nil && all.Flatten() {
		cnflict, _, expr := renameBindings(all, this, true)
		if cnflict {
			return CoveredFalse
		}
		if aexpr, ok := expr.(*All); ok {
			all = aexpr
		}
		options.setCoverArrayKeyOptions()
	}

	if all != nil {
		nExprs = append(nExprs, all)
	}
	return this.coveredBy(keyspace, nExprs, options)
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For ANY, simply list this expression.
*/
func (this *Any) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *Any) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	covers[this] = value.TRUE_VALUE
	return covers
}

func (this *Any) Copy() Expression {
	rv := NewAny(this.bindings.Copy(), Copy(this.satisfies)).(*Any)
	rv.arrayId = this.arrayId
	rv.BaseCopy(this)
	return rv
}
