//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package expression

func MarkKeyspace(keyspace string, bindVars []string, expr Expression) {
	km := newKeyspaceMarker(keyspace, bindVars)
	_, _ = km.Map(expr)
	return
}

// keyspaceMarker is used to mark an identifier as keyspace identifier
type keyspaceMarker struct {
	MapperBase

	keyspace string
	bindVars []string
}

func newKeyspaceMarker(keyspace string, bindVars []string) *keyspaceMarker {
	rv := &keyspaceMarker{
		keyspace: keyspace,
		bindVars: bindVars,
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
	} else if len(this.bindVars) > 0 {
		for _, v := range this.bindVars {
			if expr.identifier == v {
				expr.SetBindingVariable(true)
				break
			}
		}
	}
	return expr, nil
}
