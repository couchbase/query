//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"strings"
)

// retrieve identifier expression for a keyspace
func GetKeyspaceIdentifier(alias string, expr Expression) *Identifier {
	retriever := NewKSIdentRetriever(alias)
	_, err := expr.Accept(retriever)
	if err != nil {
		return nil
	}
	return retriever.ident
}

type ksIdentRetriever struct {
	TraverserBase

	keyspace string
	ident    *Identifier
}

func NewKSIdentRetriever(keyspace string) *ksIdentRetriever {
	rv := &ksIdentRetriever{
		keyspace: keyspace,
	}

	rv.traverser = rv
	return rv
}

func (this *ksIdentRetriever) VisitIdentifier(ident *Identifier) (interface{}, error) {
	if this.ident == nil {
		if this.keyspace == ident.Identifier() ||
			(ident.CaseInsensitive() && strings.ToLower(this.keyspace) == strings.ToLower(ident.Identifier())) {

			this.ident = ident
		}
	}
	return nil, nil
}
