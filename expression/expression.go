//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

/*
Package expression provides expression evaluation for query and
indexing.
*/
package expression

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/value"
)

/*
The type Expressions is defined as a slice of Expression. The
type CompositeExpressions is defined as a slice of Expressions.
*/
type Expressions []Expression
type CompositeExpressions []Expressions

type HasExpressions interface {
	Expressions() Expressions
	MapExpressions(mapper Mapper) error
}

/*
The Expression interface represents N1QL expressions.
*/
type Expression interface {
	fmt.Stringer
	json.Marshaler

	/*
	   Visitor pattern.
	*/
	Accept(visitor Visitor) (interface{}, error)

	/*
	   Type() returns the N1QL data type of the result of this
	   Expression. Type() allows you to infer the schema or shape
	   of query results before actually evaluating the query.
	*/
	Type() value.Type

	/*
	   Evaluate the expression for a given input and a particular
	   context.
	*/
	Evaluate(item value.Value, context Context) (value.Value, error)

	/*
	   Evaluate the expression for an indexing context. Support
	   multiple return values for array indexing.
	*/
	EvaluateForIndex(item value.Value, context Context) (value.Value, value.Values, error)

	/*
	   This method indicates if the expression is an array index
	   key, and if so, whether it is distinct, flatten.
	*/
	IsArrayIndexKey() (bool, bool, bool)

	/*
	   Value() returns the static / constant value of this
	   Expression, or nil. Expressions that depend on data,
	   clocks, or randomness must return nil. Used in index
	   selection.
	*/
	Value() value.Value

	/*
	   Static() returns the static / constant equivalent of this
	   Expression, or nil. Expressions that depend on data or
	   randomness must return nil. Used in index selection.
	*/
	Static() Expression

	/*
	   As per the N1QL specs this function returns the terminal
	   identifier in the case the expression is a path. It can
	   be thought of an expression alias. For example if for the
	   following select statement, b is the Alias. Select a.b.
	*/
	Alias() string

	/*
	   This method indicates if the expression can be used as a
	   secondary index key.
	*/
	Indexable() bool

	/*
	   True iff this Expression always returns MISSING if any of
	   its inputs are MISSING. This test is used in index
	   selection when an index contains the clause WHERE expr IS
	   NOT MISSING. False negatives are allowed.
	*/
	PropagatesMissing() bool

	/*
	   True iff this Expression always returns NULL if any of its
	   inputs is NULL. This test is used in index selection when
	   an index contains the clause WHERE expr IS NOT NULL or the
	   clause WHERE expr IS VALUED. False negatives are allowed.
	*/
	PropagatesNull() bool

	/*
	   Indicates if this expression is equivalent to the other
	   expression.  False negatives are allowed. Used in index
	   selection.
	*/
	EquivalentTo(other Expression) bool

	/*
	   Indicates if this expression depends on the other
	   expression.  False negatives are allowed. Used in index
	   selection.
	*/
	DependsOn(other Expression) bool

	/*
	   Indicates if this expression is based on the keyspace and
	   is covered by the list of expressions; that is, this
	   expression does not depend on any stored data beyond the
	   expressions.
	*/
	CoveredBy(keyspace string, exprs Expressions, options CoveredOptions) Covered

	/*
	   If this expression is in the WHERE clause of a partial
	   index, lists the Expressions that are implicitly covered.
	*/
	FilterCovers(covers map[string]value.Value) map[string]value.Value

	/* Same as FilterCovers() instead of string use expression.
	   In expression package we will not able to parse from expression.
	   Used in CoveredBy() to build ALL index key WHEN clause after rename binding
	   variables from ANY clause
	*/
	FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value

	/*
	   Utility function that returns the children of the
	   expression. For expression a+b, a and b are the children.
	*/
	Children() Expressions

	/*
	   Utility function that takes in as input parameter a mapper
	   that maps this Expression and its child expressions to
	   expressions.  If there is an error during the mapping, an
	   error is returned.
	*/
	MapChildren(mapper Mapper) error

	/*
	   Deep copy.
	*/
	Copy() Expression

	/*
	   base copy.
	*/
	BaseCopy(Expression)

	/*
	   Indicates if this expression may produce multiple
	   overlapping spans during index sarging. For example, a < 5
	   OR a < 10 produces overlapping spans. False positives are
	   ok.
	*/
	MayOverlapSpans() bool

	/*
	   Indicates if this expression may be a projection or
	   ordering term in an aggregation query.
	*/
	SurvivesGrouping(groupKeys Expressions, allowed *value.ScopeValue) (bool, Expression)

	/*
	  Returns the privileges required to execute a statement containing this expression.
	  Returns a non-nil pointer to a Privileges structure.
	*/
	Privileges() *auth.Privileges

	/*
	   Indicates if this expression can be used in index group/aggregates
	*/
	IndexAggregatable() bool

	/*
	   Gets the xattr parent path. For meta().xattr._sync, it gives _sync.
	*/
	FieldNames(base Expression, names map[string]bool) (present bool)

	/*
	   Reset the value field of the expression
	*/
	ResetValue()

	/*
	   Check expression flag
	*/
	HasExprFlag(flag uint64) bool

	/*
	   Set expression flag
	*/
	SetExprFlag(flag uint64)

	/*
	   Unset expression flag
	*/
	UnsetExprFlag(flag uint64)

	/*
	   Enable in-list evaluation optimization (using hash table)
	*/
	EnableInlistHash(context Context)

	/*
	   Reset runtime-alloated memory
	*/
	ResetMemory(context Context)

	/*
	   Set identifier flags
	*/
	SetIdentFlags(aliases map[string]bool, flags uint32)

	/*
	   ExpressionBase
	*/
	ExprBase() *ExpressionBase

	/*
	   Parser error context
	*/
	ErrorContext() string

	GetErrorContext() (int, int)

	/*
	   Contains volatile expression
	*/
	HasVolatileExpr() bool

	/*
	   Contains static variable
	*/
	HasStaticVariable() bool
}

func (this Expressions) MapExpressions(mapper Mapper) (err error) {
	for i, e := range this {
		expr, err := mapper.Map(e)
		if err != nil {
			return err
		}

		this[i] = expr
	}

	return
}

// Expressions implements Stringer() API.
func (this Expressions) String() string {
	var exprText bytes.Buffer
	exprText.WriteString("[")
	for i, expr := range this {
		if i > 0 {
			exprText.WriteString(", ")
		}
		exprText.WriteString(expr.String())
	}
	exprText.WriteString("]")
	return exprText.String()
}

func (this Expressions) Copy() Expressions {
	rv := make(Expressions, len(this))
	for i, expr := range this {
		rv[i] = Copy(expr)
	}
	return rv
}

func (this Expressions) EquivalentTo(other Expressions) bool {
	if len(this) != len(other) {
		return false
	}

	for i, expr := range this {
		if !Equivalent(expr, other[i]) {
			return false
		}
	}

	return true
}

func Copy(expr Expression) Expression {
	if expr == nil {
		return nil
	}

	return expr.Copy()
}

func CopyExpressions(exprs Expressions) Expressions {
	if exprs == nil {
		return nil
	}

	return exprs.Copy()
}

func Equivalent(expr1, expr2 Expression) bool {
	return (expr1 == nil && expr2 == nil) ||
		(expr1 != nil && expr2 != nil && expr1.EquivalentTo(expr2))
}

func Equivalents(exprs1, exprs2 Expressions) bool {
	if len(exprs1) != len(exprs2) {
		return false
	}

	for i, expr1 := range exprs1 {
		if !Equivalent(expr1, exprs2[i]) {
			return false
		}
	}

	return true
}

func GetFlattenKeys(keys Expressions) Expressions {
	for i, key := range keys {
		if all, ok := key.(*All); ok && all.Flatten() {
			fkeys := make(Expressions, 0, len(keys)+all.FlattenSize()-1)
			if i > 0 {
				fkeys = append(fkeys, keys[0:i]...)
			}
			for _, fk := range all.FlattenKeys().Operands() {
				aKey := all.Copy().(*All)
				aKey.SetFlattenValueMapping(fk.Copy())
				aKey.SetExprFlag(EXPR_FLATTEN_KEYS)
				fkeys = append(fkeys, aKey)
			}
			if i <= len(keys)-1 {
				fkeys = append(fkeys, keys[i+1:]...)
			}
			return fkeys
		}
	}
	return keys
}
