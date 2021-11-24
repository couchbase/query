//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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

func (this *keyspaceTrimmer) VisitFunction(expr Function) (interface{}, error) {
	err := expr.MapChildren(this)
	if err != nil {
		return expr, err
	}

	// special handling of Meta() function
	if meta, ok := expr.(*Meta); ok {
		if len(meta.operands) > 0 && meta.operands[0] == MISSING_EXPR {
			meta.operands = meta.operands[:0]
		}
	}

	return expr, nil
}

func (this *keyspaceTrimmer) VisitSubquery(expr Subquery) (interface{}, error) {
	// since a Subquery expression is not copied via Copy() call,
	// do not traverse inside the subquery
	return expr, nil
}
