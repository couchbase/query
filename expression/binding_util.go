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

func (this *exprBindings) VisitAdd(expr *Add) (interface{}, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitDiv(expr *Div) (interface{}, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitMod(expr *Mod) (interface{}, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitMult(expr *Mult) (interface{}, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitNeg(expr *Neg) (interface{}, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitSub(expr *Sub) (interface{}, error) {
	return this.visit(expr)
}

// Case

func (this *exprBindings) VisitSearchedCase(expr *SearchedCase) (interface{}, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitSimpleCase(expr *SimpleCase) (interface{}, error) {
	return this.visit(expr)
}

// Collection

func (this *exprBindings) VisitArray(expr *Array) (interface{}, error) {
	return nil, nil
}

func (this *exprBindings) VisitExists(expr *Exists) (interface{}, error) {
	return nil, nil
}

func (this *exprBindings) VisitFirst(expr *First) (interface{}, error) {
	return nil, nil
}

func (this *exprBindings) VisitObject(expr *Object) (interface{}, error) {
	return nil, nil
}

func (this *exprBindings) VisitIn(expr *In) (interface{}, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitWithin(expr *Within) (interface{}, error) {
	return nil, nil
}

// Comparison

func (this *exprBindings) VisitBetween(expr *Between) (interface{}, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitEq(expr *Eq) (interface{}, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitLE(expr *LE) (interface{}, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitLike(expr *Like) (interface{}, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitLT(expr *LT) (interface{}, error) {
	return this.visit(expr)
}

func (this *exprBindings) VisitIsMissing(expr *IsMissing) (interface{}, error) {
	return nil, nil
}

func (this *exprBindings) VisitIsNotMissing(expr *IsNotMissing) (interface{}, error) {
	return nil, nil
}

func (this *exprBindings) VisitIsNotNull(expr *IsNotNull) (interface{}, error) {
	return nil, nil
}

func (this *exprBindings) VisitIsNotValued(expr *IsNotValued) (interface{}, error) {
	return nil, nil
}

func (this *exprBindings) VisitIsNull(expr *IsNull) (interface{}, error) {
	return nil, nil
}

func (this *exprBindings) VisitIsValued(expr *IsValued) (interface{}, error) {
	return nil, nil
}

// Concat
func (this *exprBindings) VisitConcat(expr *Concat) (interface{}, error) {
	return nil, nil
}

// Constant
func (this *exprBindings) VisitConstant(expr *Constant) (interface{}, error) {
	return nil, nil
}

// Identifier
func (this *exprBindings) VisitIdentifier(expr *Identifier) (interface{}, error) {
	return nil, nil
}

// Construction

func (this *exprBindings) VisitArrayConstruct(expr *ArrayConstruct) (interface{}, error) {
	return nil, nil
}

func (this *exprBindings) VisitObjectConstruct(expr *ObjectConstruct) (interface{}, error) {
	return nil, nil
}

func (this *exprBindings) VisitNot(expr *Not) (interface{}, error) {
	return this.visit(expr)
}

// Navigation

func (this *exprBindings) VisitElement(expr *Element) (interface{}, error) {
	return nil, nil
}

func (this *exprBindings) VisitField(expr *Field) (interface{}, error) {
	return nil, nil
}

func (this *exprBindings) VisitFieldName(expr *FieldName) (interface{}, error) {
	return nil, nil
}

func (this *exprBindings) VisitSlice(expr *Slice) (interface{}, error) {
	return nil, nil
}

// Self
func (this *exprBindings) VisitSelf(expr *Self) (interface{}, error) {
	return nil, nil
}

// Function
func (this *exprBindings) VisitFunction(expr Function) (interface{}, error) {
	return this.visit(expr)
}

// Subquery
func (this *exprBindings) VisitSubquery(expr Subquery) (interface{}, error) {
	return nil, nil
}

// InferUnderParenthesis
func (this *exprBindings) VisitParenInfer(expr ParenInfer) (interface{}, error) {
	return nil, nil
}

// NamedParameter
func (this *exprBindings) VisitNamedParameter(expr NamedParameter) (interface{}, error) {
	return nil, nil
}

// PositionalParameter
func (this *exprBindings) VisitPositionalParameter(expr PositionalParameter) (interface{}, error) {
	return nil, nil
}

// Cover
func (this *exprBindings) VisitCover(expr *Cover) (interface{}, error) {
	return nil, nil
}

// All
func (this *exprBindings) VisitAll(expr *All) (interface{}, error) {
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

func (this *exprBindings) VisitAny(expr *Any) (interface{}, error) {
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

func (this *exprBindings) VisitAnyEvery(expr *AnyEvery) (interface{}, error) {
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

func (this *exprBindings) VisitEvery(expr *Every) (interface{}, error) {
	return nil, nil
}

// For OR, return the intersection over the children
func (this *exprBindings) VisitOr(expr *Or) (interface{}, error) {
	return this.visit(expr)
}

// For AND, return the union over the children
func (this *exprBindings) VisitAnd(expr *And) (interface{}, error) {
	return this.visit(expr)
}

func (this *exprBindings) visit(expr Expression) (interface{}, error) {
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
