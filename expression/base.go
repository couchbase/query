//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"encoding/json"
	"reflect"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

/* expression flags */
const (
	EXPR_IS_CONDITIONAL = 1 << iota
	EXPR_IS_VOLATILE
	EXPR_VALUE_MISSING
	EXPR_VALUE_NULL
	EXPR_DYNAMIC_IN
	EXPR_CAN_FLATTEN
	EXPR_OR_FROM_NE
	EXPR_DERIVED_RANGE
	EXPR_DERIVED_RANGE1
	EXPR_DERIVED_RANGE2
	EXPR_DERIVED_FROM_LIKE
	EXPR_ANY_FOR_UNNEST
	EXPR_UNNEST_NOT_MISSING
	EXPR_DEFAULT_LIKE
	EXPR_UNNEST_ISARRAY
	EXPR_FLATTEN_KEYS
	EXPR_ARRAY_IS_SET
	EXPR_VALIDATE_KEYS
	EXPR_JOIN_NOT_NULL
	EXPR_ORDER_BY
	EXPR_ANN_MOD_METRIC
)

/*
ExpressionBase is a base class for all expressions.
*/
type ExpressionBase struct {
	expr              Expression
	value             *value.Value
	exprFlags         uint64
	errorContext      ErrorContext
	aliasErrorContext ErrorContext
}

var _NIL_VALUE value.Value

func (this *ExpressionBase) SetErrorContext(line, column int) {
	this.errorContext.Set(line, column)
}

func (this *ExpressionBase) SetAliasErrorContext(line, column int) {
	this.aliasErrorContext.Set(line, column)
}

func (this *ExpressionBase) GetErrorContext() (int, int) {
	if this == nil {
		return 0, 0
	}
	return this.errorContext.Get()
}

func (this *ExpressionBase) ErrorContext() string {
	if this != nil {
		return this.errorContext.String()
	}
	return ""
}

func (this *ExpressionBase) AliasErrorContext() string {
	if this != nil {
		return this.aliasErrorContext.String()
	}
	return ""
}

func (this *ExpressionBase) String() string {
	return NewStringer().Visit(this.expr)
}

func (this *ExpressionBase) MarshalJSON() ([]byte, error) {
	s := NewStringer().Visit(this.expr)
	return json.Marshal(s)
}

/*
Make sure expression flags are copied when copying expression
*/
func (this *ExpressionBase) BaseCopy(oldExpr Expression) {
	this.setExprFlags(oldExpr.ExprBase().getExprFlags())
	this.errorContext = oldExpr.ExprBase().errorContext
	this.aliasErrorContext = oldExpr.ExprBase().aliasErrorContext
}

/*
Evaluate the expression for an indexing context. Support multiple
return values for array indexing.

By default, just call Evaluate().
*/
func (this *ExpressionBase) EvaluateForIndex(item value.Value, context Context) (
	value.Value, value.Values, error) {
	val, err := this.expr.Evaluate(item, context)
	return val, nil, err
}

/*
This method indicates if the expression is an array index key, and
if so, whether it is distinct.
*/

func (this *ExpressionBase) IsArrayIndexKey() (bool, bool, bool) {
	return false, false, false
}

func (this *ExpressionBase) getExprFlags() uint64 {
	return this.exprFlags
}

func (this *ExpressionBase) setExprFlags(flags uint64) {
	this.exprFlags = flags
}

func (this *ExpressionBase) HasExprFlag(flag uint64) bool {
	return (this.exprFlags & flag) != 0
}

func (this *ExpressionBase) SetExprFlag(flag uint64) {
	this.exprFlags |= flag
}

func (this *ExpressionBase) UnsetExprFlag(flag uint64) {
	this.exprFlags &^= flag
}

func (this *ExpressionBase) volatile() bool {
	return (this.exprFlags & EXPR_IS_VOLATILE) != 0
}

func (this *ExpressionBase) setVolatile() {
	this.exprFlags |= EXPR_IS_VOLATILE
}

func (this *ExpressionBase) unsetVolatile() {
	this.exprFlags &^= EXPR_IS_VOLATILE
}

func (this *ExpressionBase) conditional() bool {
	return (this.exprFlags & EXPR_IS_CONDITIONAL) != 0
}

func (this *ExpressionBase) setConditional() {
	this.exprFlags |= EXPR_IS_CONDITIONAL
}

/*
Value() returns the static / constant value of this Expression, or
nil. Expressions that depend on data, clocks, or random numbers must
return nil.
*/
func (this *ExpressionBase) Value() value.Value {
	if this.value != nil {
		return *this.value
	}

	if this.volatile() {
		this.value = &_NIL_VALUE
		return nil
	}

	propMissing := this.expr.PropagatesMissing()
	propNull := this.expr.PropagatesNull()

	for _, child := range this.expr.Children() {
		cv := child.Value()
		if cv == nil {
			if this.value == nil {
				this.value = &_NIL_VALUE
			}

			continue
		}

		if propMissing && cv.Type() == value.MISSING {
			this.SetExprFlag(EXPR_VALUE_MISSING)
			this.value = &cv
			return *this.value
		}

		if propNull && cv.Type() == value.NULL {
			this.SetExprFlag(EXPR_VALUE_NULL)
			this.value = &cv
		}
	}

	if this.value != nil {
		return *this.value
	}

	defer func() {
		err := recover()
		if err != nil {
			this.value = &_NIL_VALUE
			logging.Stackf(logging.DEBUG, "Panic during evaluation: %v", err)
		}
	}()

	val, err := this.expr.Evaluate(nil, nil)
	if err != nil {
		this.value = &_NIL_VALUE
		return nil
	}

	if val != nil {
		if val.Type() == value.MISSING {
			this.SetExprFlag(EXPR_VALUE_MISSING)
		} else if val.Type() == value.NULL {
			this.SetExprFlag(EXPR_VALUE_NULL)
		}
	}

	this.value = &val
	return *this.value
}

func (this *ExpressionBase) Static() Expression {
	for _, child := range this.expr.Children() {
		if child.Static() == nil {
			return nil
		}
	}

	return this.expr
}

/*
It returns an empty string or the terminal identifier of
the expression.
*/
func (this *ExpressionBase) Alias() string {
	return ""
}

/*
Range over the children of the expression, and check if each
child is indexable. If not then return false as the expression
is not indexable. If all children are indexable, then return
true.
*/
func (this *ExpressionBase) Indexable() bool {
	for _, child := range this.expr.Children() {
		if !child.Indexable() {
			return false
		}
	}

	return true
}

/*
Returns false if any child's PropagatesMissing() returns false.
*/
func (this *ExpressionBase) PropagatesMissing() bool {
	if this.conditional() {
		return false
	}

	for _, child := range this.expr.Children() {
		if !child.PropagatesMissing() {
			return false
		}
	}

	return true
}

/*
Returns false if any child's PropagatesNull() returns false.
*/
func (this *ExpressionBase) PropagatesNull() bool {
	if this.conditional() {
		return false
	}

	for _, child := range this.expr.Children() {
		if !child.PropagatesNull() {
			return false
		}
	}

	return true
}

/*
Indicates if this expression is equivalent to the other expression.
False negatives are allowed. Used in index selection.
*/
func (this *ExpressionBase) EquivalentTo(other Expression) bool {
	if this.valueEquivalentTo(other) {
		return true
	}

	if reflect.TypeOf(this.expr) != reflect.TypeOf(other) {
		return false
	}

	ours := this.expr.Children()
	theirs := other.Children()

	if len(ours) != len(theirs) {
		return false
	}

	for i, child := range ours {
		if !child.EquivalentTo(theirs[i]) {
			return false
		}
	}

	return true
}

/*
Indicates if this expression depends on the other expression.  False
negatives are allowed. Used in index selection.
*/
func (this *ExpressionBase) DependsOn(other Expression) bool {
	if this.conditional() || other.Value() != nil {
		return false
	}
	return this.dependsOn(other)
}

func (this *ExpressionBase) dependsOn(other Expression) bool {
	if this.expr.EquivalentTo(other) {
		return true
	}

	for _, child := range this.expr.Children() {
		if child.DependsOn(other) {
			return true
		}
	}

	return false
}

/*
Indicates if this expression is based on the keyspace and is covered
by the list of expressions; that is, this expression does not depend
on any stored data beyond the expressions.
*/
func (this *ExpressionBase) CoveredBy(keyspace string, exprs Expressions, options CoveredOptions) Covered {
	var rv Covered
	for _, expr := range exprs {
		if this.expr.EquivalentTo(expr) {
			return CoveredEquiv
		}

		// special handling of array index expression
		if options.hasCoverArrayKeyOptions() {
			if all, ok := expr.(*All); ok {
				rv = chkArrayKeyCover(this.expr, keyspace, exprs, all, options)
				if rv == CoveredTrue || rv == CoveredEquiv {
					return rv
				}
				switch this.expr.(type) {
				case *AnyEvery, *Every:
					return rv
				}

			}
		}
	}

	children := this.expr.Children()
	rv = CoveredTrue

	// MB-22112: we treat the special case where a keyspace is part of the projection list
	// a keyspace as a single term does not cover by definition
	// a keyspace as part of a field or a path does cover to delay the decision in terms
	// further down the path
	for _, child := range children {
		switch child.CoveredBy(keyspace, exprs, options) {
		case CoveredFalse:
			return CoveredFalse

		// MB-25317: ignore expressions not related to this keyspace
		case CoveredSkip:
			options.setCoverSkip()

			// MB-30350 trickle down CoveredSkip to outermost field
			if options.hasCoverTrickle() {
				rv = CoveredSkip
			}

		// MB-25560: this subexpression is already covered, no need to check subsequent terms
		case CoveredEquiv:
			options.setCoverSkip()

			// trickle down CoveredEquiv to outermost field
			if options.hasCoverTrickle() {
				rv = CoveredEquiv
			}
		}
	}

	return rv
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.
*/
func (this *ExpressionBase) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	return covers
}

func (this *ExpressionBase) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	return covers
}

func (this *ExpressionBase) valueEquivalentTo(other Expression) bool {
	thisValue := this.expr.Value()
	otherValue := other.Value()

	return thisValue != nil && otherValue != nil &&
		thisValue.EquivalentTo(otherValue)
}

/*
Set the receiver expression to the input expression.
*/
func (this *ExpressionBase) SetExpr(expr Expression) {
	if this.expr == nil {
		this.expr = expr
	}
}

/*
Return TRUE if any child may overlap spans.
*/
func (this *ExpressionBase) MayOverlapSpans() bool {
	for _, child := range this.expr.Children() {
		if child.MayOverlapSpans() {
			return true
		}
	}

	return false
}

func (this *ExpressionBase) SurvivesGrouping(groupKeys Expressions, allowed *value.ScopeValue) (
	bool, Expression) {
	for _, key := range groupKeys {
		if this.expr.EquivalentTo(key) {
			return true, nil
		}
	}

	for _, child := range this.expr.Children() {
		ok, expr := child.SurvivesGrouping(groupKeys, allowed)
		if !ok {
			return ok, expr
		}
	}

	return true, nil
}

func (this *ExpressionBase) Privileges() *auth.Privileges {
	// By default, the privileges required for an expression are the union
	// of the privilges required for the children of the expression.
	children := this.expr.Children()
	if len(children) == 0 {
		return auth.NewPrivileges()
	} else if len(children) == 1 {
		return children[0].Privileges()
	}

	// We want to be careful here to avoid unnecessary allocation of auth.Privileges records.
	privilegeList := make([]*auth.Privileges, len(children))
	for i, child := range children {
		privilegeList[i] = child.Privileges()
	}

	totalPrivileges := 0
	for _, privs := range privilegeList {
		totalPrivileges += privs.Num()
	}

	if totalPrivileges == 0 {
		return privilegeList[0] // will be empty
	}

	unionPrivileges := auth.NewPrivileges()
	for _, privs := range privilegeList {
		unionPrivileges.AddAll(privs)
	}
	return unionPrivileges
}

/*
Return FALSE if any child is not IndexAggregatable()
*/

func (this *ExpressionBase) IndexAggregatable() bool {
	for _, child := range this.expr.Children() {
		if !child.IndexAggregatable() {
			return false
		}
	}

	return true
}

/*
Used for Xattr paths
*/

func (this *ExpressionBase) FieldNames(base Expression, names map[string]bool) (present bool) {
	present = false
	if Equivalent(base, this.expr) {
		return true
	}

	for _, child := range this.expr.Children() {
		if child.FieldNames(base, names) {
			present = true
		}
	}

	return present
}

func XattrsNames(exprs Expressions, alias string) (present bool, names []string) {
	present = false
	var xattrs Expression
	if alias == "" {
		xattrs = NewField(NewMeta(), NewFieldName("xattrs", false))
	} else {
		xattrs = NewField(NewMeta(NewIdentifier(alias)),
			NewFieldName("xattrs", false))
	}

	mNames := make(map[string]bool, 5)
	for _, expr := range exprs {
		if expr.FieldNames(xattrs, mNames) {
			present = true
		}
	}
	if len(mNames) > 0 {
		names = make([]string, 0, len(mNames))
		for s, _ := range mNames {
			// "$document", "$document.exptime" are not allowed and caller will raise error
			if s == "$document" || s == "$document.exptime" {
				names = append([]string{s}, names...)
			} else {
				names = append(names, s)
			}
		}
		return present, names
	}

	return present, nil
}

func MetaExpiration(exprs Expressions, alias string) (present bool, names []string) {
	var base Expression
	if alias == "" {
		base = NewMeta()
	} else {
		base = NewMeta(NewIdentifier(alias))
	}

	mNames := make(map[string]bool, 5)
	for _, expr := range exprs {
		expr.FieldNames(base, mNames)
	}

	for s, _ := range mNames {
		if s == "expiration" {
			return true, []string{"$document.exptime"}
		}
	}

	return false, nil
}

func (this *ExpressionBase) ResetValue() {
	this.value = nil
	for _, child := range this.expr.Children() {
		child.ResetValue()
	}
}

/*
Enable in-list evaluation optimization (using a hash table)
*/
func (this *ExpressionBase) EnableInlistHash(context Context) {
	for _, child := range this.expr.Children() {
		child.EnableInlistHash(context)
	}
}

func (this *ExpressionBase) ResetMemory(context Context) {
	for _, child := range this.expr.Children() {
		child.ResetMemory(context)
	}
}

func (this *ExpressionBase) SetIdentFlags(aliases map[string]bool, flags uint32) {
	for _, child := range this.expr.Children() {
		child.SetIdentFlags(aliases, flags)
	}
}

func (this *ExpressionBase) ExprBase() *ExpressionBase {
	return this
}

func (this *ExpressionBase) HasVolatileExpr() bool {
	if this.volatile() {
		return true
	}
	for _, child := range this.expr.Children() {
		if child.HasVolatileExpr() {
			return true
		}
	}
	return false
}
