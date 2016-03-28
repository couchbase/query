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
Represents range transform First, that allow you to map and
filter the elements or attributes of a collection or object(s).
FIRST evaluates to a single element based on the operand expression.
Type First is a struct that implements collMap.
*/
type First struct {
	collMapBase
}

/*
This method returns a pointer to the First struct that has the
bindings,mapping and when fields populated by the input args
bindings and expression when/mapping.
*/
func NewFirst(mapping Expression, bindings Bindings, when Expression) Expression {
	rv := &First{
		collMapBase: collMapBase{
			mapping:  mapping,
			bindings: bindings,
			when:     when,
		},
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFirst method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *First) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFirst(this)
}

/*
It returns a value that is the receivers mapping type. This is
because First evaluates to a single element based on the operand
expression.
*/
func (this *First) Type() value.Type { return this.mapping.Type() }

func (this *First) Evaluate(item value.Value, context Context) (value.Value, error) {
	bvals, bpairs, n, missing, null, err := collEval(this.bindings, item, context)
	defer collReleaseBuffers(bvals, bpairs)

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

		if this.when != nil {
			wv, e := this.when.Evaluate(cv, context)
			if e != nil {
				return nil, e
			}

			if !wv.Truth() {
				continue
			}
		}

		mv, e := this.mapping.Evaluate(cv, context)
		if e != nil {
			return nil, e
		}

		return mv, nil
	}

	return value.MISSING_VALUE, nil
}

func (this *First) Copy() Expression {
	return NewFirst(this.mapping.Copy(), this.bindings.Copy(), Copy(this.when))
}
