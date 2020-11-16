//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"github.com/couchbase/query/expression"
	base "github.com/couchbase/query/plannerbase"
)

type sarg struct {
	key             expression.Expression
	baseKeyspace    *base.BaseKeyspace
	keyspaceNames   map[string]string
	isJoin          bool
	doSelec         bool
	advisorValidate bool
	context         *PrepareContext
}

func (this *sarg) getSarg(pred expression.Expression) expression.Expression {
	if pred == nil {
		return nil
	}

	cpred := pred.Static()
	if cpred != nil || !this.isJoin {
		return cpred
	}

	if pred.Indexable() {
		// make sure the expression does NOT reference current keyspace
		keyspaceNames := make(map[string]string, 1+len(this.baseKeyspace.GetUnnests()))
		keyspaceNames[this.baseKeyspace.Name()] = this.baseKeyspace.Keyspace()
		for a, k := range this.baseKeyspace.GetUnnests() {
			keyspaceNames[a] = k
		}
		if !expression.HasKeyspaceReferences(pred, keyspaceNames) {
			return pred.Copy()
		}
	}

	return nil
}

func (this *sarg) getSelec(pred expression.Expression) float64 {
	if !this.doSelec {
		return OPT_SELEC_NOT_AVAIL
	}

	var array bool
	switch pred.(type) {
	case *expression.Any, *expression.AnyEvery, *expression.Every:
		array = true
	}

	for _, fl := range this.baseKeyspace.Filters() {
		if pred.EquivalentTo(fl.FltrExpr()) {
			if array {
				return fl.ArraySelec()
			}
			return fl.Selec()
		}
	}

	// if this is a subterm of an OR, it won't be in filters
	var keyspaces map[string]string
	var err error
	if this.isJoin {
		keyspaces, err = expression.CountKeySpaces(pred, this.keyspaceNames)
		if err != nil {
			return OPT_SELEC_NOT_AVAIL
		}
	} else {
		keyspaces = make(map[string]string, 1)
		keyspaces[this.baseKeyspace.Name()] = this.baseKeyspace.Keyspace()
	}
	sel, arrSel := optExprSelec(keyspaces, pred, this.advisorValidate, this.context)

	if array {
		return arrSel
	}
	return sel
}

// Arithmetic

func (this *sarg) VisitAdd(pred *expression.Add) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitDiv(pred *expression.Div) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitMod(pred *expression.Mod) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitMult(pred *expression.Mult) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitNeg(pred *expression.Neg) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitSub(pred *expression.Sub) (interface{}, error) {
	return this.visitDefault(pred)
}

// Case

func (this *sarg) VisitSearchedCase(pred *expression.SearchedCase) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitSimpleCase(pred *expression.SimpleCase) (interface{}, error) {
	return this.visitDefault(pred)
}

// Collection

func (this *sarg) VisitArray(pred *expression.Array) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitEvery(pred *expression.Every) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitExists(pred *expression.Exists) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitFirst(pred *expression.First) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitObject(pred *expression.Object) (interface{}, error) {
	return this.visitDefault(pred)
}

// Comparison

func (this *sarg) VisitBetween(pred *expression.Between) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitLike(pred *expression.Like) (interface{}, error) {
	return this.visitLike(pred)
}

func (this *sarg) VisitIsNotValued(pred *expression.IsNotValued) (interface{}, error) {
	return this.visitDefault(pred)
}

// Concat
func (this *sarg) VisitConcat(pred *expression.Concat) (interface{}, error) {
	return this.visitDefault(pred)
}

// Constant
func (this *sarg) VisitConstant(pred *expression.Constant) (interface{}, error) {
	return this.visitDefault(pred)
}

// Identifier
func (this *sarg) VisitIdentifier(pred *expression.Identifier) (interface{}, error) {
	return this.visitDefault(pred)
}

// Construction

func (this *sarg) VisitArrayConstruct(pred *expression.ArrayConstruct) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitObjectConstruct(pred *expression.ObjectConstruct) (interface{}, error) {
	return this.visitDefault(pred)
}

// Logic

func (this *sarg) VisitNot(pred *expression.Not) (interface{}, error) {
	return this.visitDefault(pred)
}

// Navigation

func (this *sarg) VisitElement(pred *expression.Element) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitField(pred *expression.Field) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitFieldName(pred *expression.FieldName) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitSlice(pred *expression.Slice) (interface{}, error) {
	return this.visitDefault(pred)
}

// Self
func (this *sarg) VisitSelf(pred *expression.Self) (interface{}, error) {
	return this.visitDefault(pred)
}

// Function
func (this *sarg) VisitFunction(pred expression.Function) (interface{}, error) {
	switch pred := pred.(type) {
	case *expression.RegexpLike:
		return this.visitLike(pred)
	}

	return this.visitDefault(pred)
}

// Subquery
func (this *sarg) VisitSubquery(pred expression.Subquery) (interface{}, error) {
	return this.visitDefault(pred)
}

// NamedParameter
func (this *sarg) VisitNamedParameter(pred expression.NamedParameter) (interface{}, error) {
	return this.visitDefault(pred)
}

// PositionalParameter
func (this *sarg) VisitPositionalParameter(pred expression.PositionalParameter) (interface{}, error) {
	return this.visitDefault(pred)
}

// Cover
func (this *sarg) VisitCover(pred *expression.Cover) (interface{}, error) {
	return pred.Covered().Accept(this)
}

// All
func (this *sarg) VisitAll(pred *expression.All) (interface{}, error) {
	return pred.Array().Accept(this)
}
