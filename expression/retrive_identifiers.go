//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

// retrieve identifier expressions

func GetIdentifiers(expr Expression) map[string]Expression {
	rv := &IdentifiersRetriever{names: make(map[string]Expression)}
	rv.mapFunc = func(expr Expression) (Expression, error) {
		return expr, expr.MapChildren(rv)
	}
	rv.mapper = rv

	if _, err := expr.Accept(rv); err != nil {
		return nil
	}

	return rv.names
}

type IdentifiersRetriever struct {
	MapperBase
	names map[string]Expression
}

func (this *IdentifiersRetriever) VisitIdentifier(expr *Identifier) (interface{}, error) {
	this.names[expr.Alias()] = expr
	return expr, nil
}
