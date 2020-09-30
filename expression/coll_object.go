//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
		if ai, ok := item.(value.AnnotatedValue); ok {
			av.CopyAnnotations(ai)
		}

		if this.when != nil {
			wv, e := this.when.Evaluate(av, context)
			if e != nil {
				return nil, e
			}

			if !wv.Truth() {
				continue
			}
		}

		nv, e := this.nameMapping.Evaluate(av, context)
		if e != nil {
			return nil, e
		}

		switch nv.Type() {
		case value.STRING:
			// Do nothing
		case value.MISSING:
			return value.MISSING_VALUE, nil
		default:
			return value.NULL_VALUE, nil
		}

		vv, e := this.valueMapping.Evaluate(av, context)
		if e != nil {
			return nil, e
		}

		if vv.Type() != value.MISSING {
			rv[nv.Actual().(string)] = vv
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
