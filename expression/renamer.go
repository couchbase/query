//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"fmt"
)

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

func (this *Renamer) VisitAll(expr *All) (interface{}, error) {
	if array, ok := expr.array.(*Array); ok {
		for _, b := range array.Bindings() {
			if name, ok := this.names[b.variable]; ok {
				b.variable = name.Alias()
			}
			if name, ok := this.names[b.nameVariable]; ok {
				b.nameVariable = name.Alias()
			}
		}
	}
	return expr, expr.MapChildren(this)
}

func (this *Renamer) VisitAny(expr *Any) (interface{}, error) {
	for _, b := range expr.Bindings() {
		if name, ok := this.names[b.variable]; ok {
			b.variable = name.Alias()
		}
		if name, ok := this.names[b.nameVariable]; ok {
			b.nameVariable = name.Alias()
		}
	}
	return expr, expr.MapChildren(this)
}

func (this *Renamer) VisitAnyEvery(expr *AnyEvery) (interface{}, error) {
	for _, b := range expr.Bindings() {
		if name, ok := this.names[b.variable]; ok {
			b.variable = name.Alias()
		}
		if name, ok := this.names[b.nameVariable]; ok {
			b.nameVariable = name.Alias()
		}
	}
	return expr, expr.MapChildren(this)
}

// Rename ANY (nested level too) clause binding variables with arrayKey binding variables

type AnyRenamer struct {
	MapperBase
	arrayKey *All
}

func NewAnyRenamer(arrayKey *All) *AnyRenamer {
	rv := &AnyRenamer{
		arrayKey: arrayKey,
	}

	rv.mapFunc = func(expr Expression) (Expression, error) {
		if rv.arrayKey == nil {
			return expr, nil
		} else {
			return expr, expr.MapChildren(rv)
		}
	}

	rv.mapper = rv
	return rv
}

func (this *AnyRenamer) VisitAny(expr *Any) (interface{}, error) {
	array, ok := this.arrayKey.Array().(*Array)
	if ok && equivalentBindingsWithExpression(expr.Bindings(), array.Bindings(), nil, nil) {
		arrayKey := this.arrayKey
		cnflict, _, nExpr := renameBindings(expr, this.arrayKey, false)
		expr, ok = nExpr.(*Any)
		if cnflict || !ok {
			return nil, fmt.Errorf("Binding variable conflict")
		}

		defer func() {
			this.arrayKey = arrayKey
		}()

		this.arrayKey, _ = array.valueMapping.(*All)
	}

	return expr, expr.MapChildren(this)
}

func RenameAnyExpr(expr Expression, arrayKey *All) (Expression, error) {
	if arrayKey != nil && expr != nil {
		rv := NewAnyRenamer(arrayKey)
		return rv.Map(expr)
	}
	return expr, nil
}
