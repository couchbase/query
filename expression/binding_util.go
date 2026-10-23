//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

// Get the bindings of the expression
func getExprBindings(expr Expression) (bindings Bindings) {
	if expr != nil {
		switch other := expr.(type) {
		case *All:
			if array, ok := other.array.(*Array); ok {
				return array.Bindings()
			}
		case *Any:
			return other.Bindings()
		case *AnyEvery:
			return other.Bindings()
		}
	}
	return
}

// If conflict or error or all same return false.
func HasRenameableBindings(from, to Expression, aliases map[string]bool) BindingVarOptions {
	// top level bindings
	toBindings := getExprBindings(to)
	if from == nil || len(toBindings) == 0 {
		return BINDING_VARS_SAME
	}

	// get all nested from bindings
	fromBindings, err := BindingsFor(from)
	if err != nil || len(fromBindings) == 0 {
		return BINDING_VARS_SAME
	}

	// check what are the new variables, remove if same variable present in the same position
	rv, names := fromBindings[0].RenameVariables(toBindings)
	if rv == BINDING_VARS_CONFLICT || len(names) == 0 {
		return rv
	}

	// conflicts with existing aliases (keyspace, LET, WITH)
	for a, _ := range aliases {
		if _, ok := names[a]; ok {
			return BINDING_VARS_CONFLICT
		}
	}

	// conflicts with existing nested variables
	for _, b := range fromBindings[1:] {
		if b.DuplicateVariable(names) {
			return BINDING_VARS_CONFLICT
		}
	}

	// no conflicts
	return rv
}

// Collect all the bindings (including nested)

func BindingsFor(expr Expression) ([]Bindings, error) {
	cov := &exprBindings{}
	rv, err := expr.Accept(cov)
	if err != nil || rv == nil {
		return nil, err
	}
	rvb, _ := rv.([]Bindings)
	return rvb, nil
}

type exprBindings struct {
}

// Arithmetic

func (this *exprBindings) VisitAdd(expr *Add) (any, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitDiv(expr *Div) (any, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitMod(expr *Mod) (any, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitMult(expr *Mult) (any, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitNeg(expr *Neg) (any, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitSub(expr *Sub) (any, error) {
	return this.visit(expr)
}

// Case

func (this *exprBindings) VisitSearchedCase(expr *SearchedCase) (any, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitSimpleCase(expr *SimpleCase) (any, error) {
	return this.visit(expr)
}

// Collection

func (this *exprBindings) VisitArray(expr *Array) (any, error) {
	return nil, nil
}

func (this *exprBindings) VisitExists(expr *Exists) (any, error) {
	return nil, nil
}

func (this *exprBindings) VisitFirst(expr *First) (any, error) {
	return nil, nil
}

func (this *exprBindings) VisitObject(expr *Object) (any, error) {
	return nil, nil
}

func (this *exprBindings) VisitIn(expr *In) (any, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitWithin(expr *Within) (any, error) {
	return nil, nil
}

// Comparison

func (this *exprBindings) VisitBetween(expr *Between) (any, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitEq(expr *Eq) (any, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitLE(expr *LE) (any, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitLike(expr *Like) (any, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitLT(expr *LT) (any, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitIsMissing(expr *IsMissing) (any, error) {
	return nil, nil
}

func (this *exprBindings) VisitIsNotMissing(expr *IsNotMissing) (any, error) {
	return nil, nil
}

func (this *exprBindings) VisitIsNotNull(expr *IsNotNull) (any, error) {
	return nil, nil
}

func (this *exprBindings) VisitIsNotValued(expr *IsNotValued) (any, error) {
	return nil, nil
}

func (this *exprBindings) VisitIsNull(expr *IsNull) (any, error) {
	return nil, nil
}

func (this *exprBindings) VisitIsValued(expr *IsValued) (any, error) {
	return nil, nil
}

// Concat
func (this *exprBindings) VisitConcat(expr *Concat) (any, error) {
	return nil, nil
}

// Constant
func (this *exprBindings) VisitConstant(expr *Constant) (any, error) {
	return nil, nil
}

// Identifier
func (this *exprBindings) VisitIdentifier(expr *Identifier) (any, error) {
	return nil, nil
}

// Construction

func (this *exprBindings) VisitArrayConstruct(expr *ArrayConstruct) (any, error) {
	return nil, nil
}

func (this *exprBindings) VisitObjectConstruct(expr *ObjectConstruct) (any, error) {
	return nil, nil
}

func (this *exprBindings) VisitNot(expr *Not) (any, error) {
	return this.visit(expr)
}

// Navigation

func (this *exprBindings) VisitElement(expr *Element) (any, error) {
	return nil, nil
}

func (this *exprBindings) VisitField(expr *Field) (any, error) {
	return nil, nil
}

func (this *exprBindings) VisitFieldName(expr *FieldName) (any, error) {
	return nil, nil
}

func (this *exprBindings) VisitSlice(expr *Slice) (any, error) {
	return nil, nil
}

// Self
func (this *exprBindings) VisitSelf(expr *Self) (any, error) {
	return nil, nil
}

// Function
func (this *exprBindings) VisitFunction(expr Function) (any, error) {
	return this.visit(expr)
}

// Subquery
func (this *exprBindings) VisitSubquery(expr Subquery) (any, error) {
	return nil, nil
}

// InferUnderParenthesis
func (this *exprBindings) VisitParenInfer(expr ParenInfer) (any, error) {
	return nil, nil
}

// NamedParameter
func (this *exprBindings) VisitNamedParameter(expr NamedParameter) (any, error) {
	return nil, nil
}

// PositionalParameter
func (this *exprBindings) VisitPositionalParameter(expr PositionalParameter) (any, error) {
	return nil, nil
}

// Cover
func (this *exprBindings) VisitCover(expr *Cover) (any, error) {
	return nil, nil
}

// All
func (this *exprBindings) VisitAll(expr *All) (any, error) {
	var rv []Bindings
	all := expr
	for {
		switch array := all.Array().(type) {
		case *Array:
			rv = append(rv, array.Bindings())
			switch valMapping := array.ValueMapping().(type) {
			case *All:
				all = valMapping
			default:
				return rv, nil
			}
		default:
			return rv, nil
		}
	}
	return rv, nil
}

func (this *exprBindings) VisitAny(expr *Any) (any, error) {
	var rv []Bindings
	rv = append(rv, expr.Bindings())
	sv, err := expr.Satisfies().Accept(this)
	if err != nil {
		return nil, err
	}
	svb, _ := sv.([]Bindings)
	if len(svb) > 0 {
		rv = append(rv, svb...)
	}
	return rv, err
}

func (this *exprBindings) VisitAnyEvery(expr *AnyEvery) (any, error) {
	var rv []Bindings
	rv = append(rv, expr.Bindings())
	sv, err := expr.Satisfies().Accept(this)
	if err != nil {
		return nil, err
	}
	svb, _ := sv.([]Bindings)
	if len(svb) > 0 {
		rv = append(rv, svb...)
	}
	return rv, err
}

func (this *exprBindings) VisitEvery(expr *Every) (any, error) {
	return nil, nil
}

// For OR, return the intersection over the children
func (this *exprBindings) VisitOr(expr *Or) (any, error) {
	return this.visit(expr)
}

// For AND, return the union over the children
func (this *exprBindings) VisitAnd(expr *And) (any, error) {
	return this.visit(expr)
}

func (this *exprBindings) visit(expr Expression) (any, error) {
	var rv []Bindings
	for _, op := range expr.Children() {
		cv, err := op.Accept(this)
		if err != nil {
			return nil, err
		}
		cvb, _ := cv.([]Bindings)

		if len(cvb) > 0 {
			rv = append(rv, cvb...)
		}
	}
	return rv, nil
}
