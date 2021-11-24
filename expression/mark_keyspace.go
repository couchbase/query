//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

func MarkKeyspace(keyspace string, expr Expression) {
	km := newKeyspaceMarker(keyspace)
	_, _ = km.Map(expr)
	return
}

// keyspaceMarker is used to mark an identifier as keyspace identifier
type keyspaceMarker struct {
	MapperBase

	keyspace string
}

func newKeyspaceMarker(keyspace string) *keyspaceMarker {
	rv := &keyspaceMarker{
		keyspace: keyspace,
	}

	rv.mapFunc = func(expr Expression) (Expression, error) {
		if keyspace == "" {
			return expr, nil
		} else {
			return expr, expr.MapChildren(rv)
		}
	}

	rv.mapper = rv
	return rv
}

func (this *keyspaceMarker) VisitIdentifier(expr *Identifier) (interface{}, error) {
	if expr.identifier == this.keyspace {
		expr.SetKeyspaceAlias(true)
	}
	return expr, nil
}
