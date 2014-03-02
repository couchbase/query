//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

type collMap struct {
	ExpressionBase
	mapping  Expression
	bindings Bindings
	when     Expression
}

func (this *collMap) Children() Expressions {
	d := make(Expressions, 0, 2+len(this.bindings))
	d = append(d, this.mapping)

	for _, b := range this.bindings {
		d = append(d, b.Expression())
	}

	if this.when != nil {
		d = append(d, this.when)
	}

	return d
}

func (this *collMap) VisitChildren(visitor Visitor) (Expression, error) {
	var e error
	this.mapping, e = visitor.Visit(this.mapping)
	if e != nil {
		return nil, e
	}

	for _, b := range this.bindings {
		_, e = b.Accept(visitor)
		if e != nil {
			return nil, e
		}
	}

	if this.when != nil {
		this.when, e = visitor.Visit(this.when)
		if e != nil {
			return nil, e
		}
	}

	return this, nil
}

type collPred struct {
	ExpressionBase
	bindings  Bindings
	satisfies Expression
}

func (this *collPred) Children() Expressions {
	d := make(Expressions, 0, 1+len(this.bindings))
	for _, b := range this.bindings {
		d = append(d, b.Expression())
	}

	d = append(d, this.satisfies)
	return d
}

func (this *collPred) VisitChildren(visitor Visitor) (Expression, error) {
	var e error
	for _, b := range this.bindings {
		_, e = b.Accept(visitor)
		if e != nil {
			return nil, e
		}
	}

	this.satisfies, e = visitor.Visit(this.satisfies)
	if e != nil {
		return nil, e
	}

	return this, nil
}
