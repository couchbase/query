//  Copyright 2016-Present Couchbase, Inc.
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
Represents range predicate ANY AND EVERY, that allow testing of a bool
condition over the elements of a collection or object.
*/
type AnyEvery struct {
	collPredBase
}

func NewAnyEvery(bindings Bindings, satisfies Expression) Expression {
	rv := &AnyEvery{
		collPredBase: collPredBase{
			bindings:  bindings,
			satisfies: satisfies,
		},
	}

	rv.expr = rv
	return rv
}

func (this *AnyEvery) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAnyEvery(this)
}

func (this *AnyEvery) Type() value.Type { return value.BOOLEAN }

func (this *AnyEvery) Evaluate(item value.Value, context Context) (value.Value, error) {
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
		if item != nil {
			if ai, ok := item.(value.AnnotatedValue); ok {
				av.CopyAnnotations(ai)
			}
		}

		sv, e := this.satisfies.Evaluate(av, context)
		av.Recycle()
		if e != nil {
			return nil, e
		}

		if !sv.Truth() {
			return value.FALSE_VALUE, nil
		}
	}

	return value.NewValue(n > 0), nil
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For ANY AND EVERY, simply list this expression.
*/
func (this *AnyEvery) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *AnyEvery) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	covers[this] = value.TRUE_VALUE
	return covers
}

func (this *AnyEvery) Copy() Expression {
	rv := NewAnyEvery(this.bindings.Copy(), Copy(this.satisfies)).(*AnyEvery)
	rv.arrayId = this.arrayId
	rv.BaseCopy(this)
	return rv
}
