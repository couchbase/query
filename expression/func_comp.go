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
	"github.com/couchbaselabs/query/value"
)

type Greatest struct {
	nAryBase
}

func NewGreatest(args Expressions) Function {
	return &Greatest{
		nAryBase{
			operands: args,
		},
	}
}

func (this *Greatest) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Greatest) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Greatest) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Greatest) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Greatest) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Greatest) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Greatest) eval(args value.Values) (value.Value, error) {
	rv := value.NULL_VALUE
	for _, a := range args {
		if a.Type() <= value.NULL {
			continue
		} else if rv == value.NULL_VALUE {
			rv = a
		} else if rv.Type() != a.Type() {
			return value.NULL_VALUE, nil
		} else if a.Collate(rv) > 0 {
			rv = a
		}
	}

	return rv, nil
}

func (this *Greatest) MinArgs() int { return 1 }

func (this *Greatest) Constructor() FunctionConstructor { return NewGreatest }

type Least struct {
	nAryBase
}

func NewLeast(args Expressions) Function {
	return &Least{
		nAryBase{
			operands: args,
		},
	}
}

func (this *Least) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Least) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Least) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Least) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Least) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Least) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Least) eval(args value.Values) (value.Value, error) {
	rv := value.NULL_VALUE
	for _, a := range args {
		if a.Type() <= value.NULL {
			continue
		} else if rv == value.NULL_VALUE {
			rv = a
		} else if rv.Type() != a.Type() {
			return value.NULL_VALUE, nil
		} else if a.Collate(rv) < 0 {
			rv = a
		}
	}

	return rv, nil
}

func (this *Least) MinArgs() int { return 1 }

func (this *Least) Constructor() FunctionConstructor { return NewLeast }
