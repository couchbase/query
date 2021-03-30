//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package expression

/*
Renamer is used to rename binding variables, but is a generic
expression renamer.
*/
type Renamer struct {
	MapperBase

	names map[string]*Identifier
}

func NewRenamer(from, to Bindings) *Renamer {
	var names map[string]*Identifier

	if from.SubsetOf(to) {
		for i, f := range from {
			t := to[i]

			if f.variable != t.variable {
				if names == nil {
					names = make(map[string]*Identifier, len(from))
				}

				ident := NewIdentifier(t.variable)
				ident.SetBindingVariable(true)
				names[f.variable] = ident
			}

			if f.nameVariable != t.nameVariable {
				if names == nil {
					names = make(map[string]*Identifier, len(from))
				}

				ident := NewIdentifier(t.nameVariable)
				ident.SetBindingVariable(true)
				names[f.nameVariable] = ident
			}
		}
	}

	rv := &Renamer{
		names: names,
	}

	rv.mapFunc = func(expr Expression) (Expression, error) {
		if len(names) == 0 {
			return expr, nil
		} else {
			return expr, expr.MapChildren(rv)
		}
	}

	rv.mapper = rv
	return rv
}

func (this *Renamer) VisitIdentifier(expr *Identifier) (interface{}, error) {
	name, ok := this.names[expr.identifier]
	if ok {
		return name, nil
	} else {
		return expr, nil
	}
}
