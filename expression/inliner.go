//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

/*
Inliner is a mapper that inlines bindings, e.g. of a LET clause.
*/
type Inliner struct {
	MapperBase
	mappings map[string]Expression
	modified bool
}

func NewInliner(mappings map[string]Expression) *Inliner {
	rv := &Inliner{
		mappings: mappings,
	}

	rv.mapper = rv
	return rv
}

func (this *Inliner) VisitIdentifier(id *Identifier) (interface{}, error) {
	repl, ok := this.mappings[id.Identifier()]
	if ok {
		this.modified = true
		return repl, nil
	} else {
		return id, nil
	}
}

func (this *Inliner) IsModified() bool {
	return this.modified
}

func (this *Inliner) Reset() {
	this.modified = false
}
