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
Represents range transform OBJECT, that allow you to map and filter
the elements of a collection or object.
*/
type Object struct {
	collMapBase
}

func NewObject(nameMapping, valueMapping Expression, bindings Bindings, when Expression) Expression {
	rv := &Object{
		collMapBase: collMapBase{
			nameMapping:  nameMapping,
			valueMapping: valueMapping,
			bindings:     bindings,
			when:         when,
		},
	}

	rv.expr = rv
	return rv
}

func (this *Object) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitObject(this)
}

func (this *Object) Type() value.Type { return value.OBJECT }

func (this *Object) Evaluate(item value.Value, context Context) (value.Value, error) {
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

	rv := make(map[string]interface{}, n)
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

		if this.when != nil {
			wv, e := this.when.Evaluate(av, context)
			if e != nil {
				av.Recycle()
				return nil, e
			}

			if !wv.Truth() {
				av.Recycle()
				continue
			}
		}

		nv, e := this.nameMapping.Evaluate(av, context)
		if e != nil {
			av.Recycle()
			return nil, e
		}

		switch nv.Type() {
		case value.STRING:
			// Do nothing
		case value.MISSING:
			av.Recycle()
			return value.MISSING_VALUE, nil
		default:
			av.Recycle()
			return value.NULL_VALUE, nil
		}

		vv, e := this.valueMapping.Evaluate(av, context)
		av.Recycle()
		if e != nil {
			return nil, e
		}

		if vv.Type() != value.MISSING {
			rv[nv.ToString()] = vv
		}
	}

	return value.NewValue(rv), nil
}

func (this *Object) Copy() Expression {
	rv := NewObject(this.nameMapping.Copy(), this.valueMapping.Copy(),
		this.bindings.Copy(), Copy(this.when))
	rv.BaseCopy(this)
	return rv
}
