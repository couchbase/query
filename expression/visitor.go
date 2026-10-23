//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

/*
The type of Visitor is an interface with a list of methods that are
implemented in Stringer.go. The general class of methods fall under
are Arithmetic and Case Expressions, collections, concat, constant,
identifier, construction, logic, navigation, and function all of
which take an input parameter of that type and return an interface
and an error.
*/
type Visitor interface {
	/*
	   Arithmetic expressions.
	*/
	VisitAdd(expr *Add) (any, error)
	VisitDiv(expr *Div) (any, error)
	VisitMod(expr *Mod) (any, error)
	VisitMult(expr *Mult) (any, error)
	VisitNeg(expr *Neg) (any, error)
	VisitSub(expr *Sub) (any, error)

	/*
	   Case Expressions. There are two types, searched case and
	   simple case expressions. Refer to N1QL specs.
	*/
	VisitSearchedCase(expr *SearchedCase) (any, error)
	VisitSimpleCase(expr *SimpleCase) (any, error)

	/*
	   Collections. (A collection is an ordered group of elements).
	   Refer to the N1QL specs for the list of supported
	   collections.
	*/
	VisitAny(expr *Any) (any, error)
	VisitEvery(expr *Every) (any, error)
	VisitAnyEvery(expr *AnyEvery) (any, error)
	VisitArray(expr *Array) (any, error)
	VisitFirst(expr *First) (any, error)
	VisitObject(expr *Object) (any, error)
	VisitExists(expr *Exists) (any, error)
	VisitIn(expr *In) (any, error)
	VisitWithin(expr *Within) (any, error)

	/*
	   Comparison terms hwlp compare two or more expressions.
	   Refer to the N1QL specs for the list of supported
	   comparison terms.
	*/
	VisitBetween(expr *Between) (any, error)
	VisitEq(expr *Eq) (any, error)
	VisitLE(expr *LE) (any, error)
	VisitLike(expr *Like) (any, error)
	VisitLT(expr *LT) (any, error)
	VisitIsMissing(expr *IsMissing) (any, error)
	VisitIsNotMissing(expr *IsNotMissing) (any, error)
	VisitIsNotNull(expr *IsNotNull) (any, error)
	VisitIsNotValued(expr *IsNotValued) (any, error)
	VisitIsNull(expr *IsNull) (any, error)
	VisitIsValued(expr *IsValued) (any, error)

	/*
	   Concat. Both expressions need to be strings, else
	   returns null.
	*/
	VisitConcat(expr *Concat) (any, error)

	/*
	   Visit a Constant expression.
	*/
	VisitConstant(expr *Constant) (any, error)

	/*
	   Identifier. They can be of two types, escaped and unescaped.
	   Refer to the N1QL specs.
	*/
	VisitIdentifier(expr *Identifier) (any, error)

	/*
	   Construction. As per the N1ql specs, objects and arrays can
	   be constructed with arbitrary structure, nesting, and
	   embedded expressions.
	*/
	VisitArrayConstruct(expr *ArrayConstruct) (any, error)
	VisitObjectConstruct(expr *ObjectConstruct) (any, error)

	/*
	   Logical Terms use boolean logic. Standard operators.
	*/
	VisitAnd(expr *And) (any, error)
	VisitNot(expr *Not) (any, error)
	VisitOr(expr *Or) (any, error)

	/*
	   Navigation. Used to navigate through objects and
	   arrays.
	*/
	VisitElement(expr *Element) (any, error)
	VisitField(expr *Field) (any, error)
	VisitFieldName(expr *FieldName) (any, error)
	VisitSlice(expr *Slice) (any, error)

	/*
	   Self.
	*/
	VisitSelf(expr *Self) (any, error)

	/*
	   Function. Refer to N1QL for a list of supported functions.
	*/
	VisitFunction(expr Function) (any, error)

	/*
	   Subquery. Returns the subquery string.
	*/
	VisitSubquery(expr Subquery) (any, error)

	/*
		InferUnderParenthesis
	*/
	VisitParenInfer(expr ParenInfer) (any, error)

	/*
	   Parameters. There are 2 types, named and positional.
	   It allows passing of parameters into the query using
	   position or name.
	*/
	VisitNamedParameter(expr NamedParameter) (any, error)
	VisitPositionalParameter(expr PositionalParameter) (any, error)

	/*
	   Indexing support.
	*/
	VisitCover(expr *Cover) (any, error)
	VisitAll(expr *All) (any, error)
}
