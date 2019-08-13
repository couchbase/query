//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

func TrimKeyspace(expr Expression, keyspace string) (Expression, error) {
	ksTrimmer := newKeyspaceTrimmer(keyspace)
	newExpr, err := ksTrimmer.Map(expr.Copy())
	if err != nil {
		return nil, err
	}

	return newExpr, nil
}

type keyspaceTrimmer struct {
	MapperBase

	keyspace string
}

func newKeyspaceTrimmer(keyspace string) *keyspaceTrimmer {
	rv := &keyspaceTrimmer{
		keyspace: keyspace,
	}

	rv.SetMapper(rv)
	return rv
}

func (this *keyspaceTrimmer) VisitIdentifier(expr *Identifier) (interface{}, error) {
	ident := expr.Identifier()
	if ident == this.keyspace {
		// cannot return a nil pointer for mapper, use MISSING_EXPR instead
		return MISSING_EXPR, nil
	}
	return expr, nil
}

func (this *keyspaceTrimmer) VisitField(expr *Field) (interface{}, error) {
	first, err := this.Map(expr.First())
	if err != nil {
		return nil, err
	}

	second := expr.Second()

	if first != MISSING_EXPR {
		if first.EquivalentTo(expr.First()) {
			return expr, nil
		}

		rv := NewField(first, second)
		rv.BaseCopy(expr)
		return rv, nil
	}

	if fn, ok := second.(*FieldName); ok {
		return NewIdentifier(fn.Alias()), nil
	}

	return second, nil
}
