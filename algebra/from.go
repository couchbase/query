//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

/*
Represents the from clause in a select statement.
*/
type FromTerm interface {
	/*
	   Represents the Node interface.
	*/
	Node

	/*
	   Apply a Mapper to all the expressions in this statement
	*/
	MapExpressions(mapper expression.Mapper) error

	/*
	   Returns all contained Expressions.
	*/
	Expressions() expression.Expressions

	/*
	   Returns all required privileges.
	*/
	Privileges() (*auth.Privileges, errors.Error)

	/*
	   Representation as a N1QL string.
	*/
	String() string

	/*
	   Qualify all identifiers for the parent expression.
	*/
	Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error)

	/*
	   Represents the primary (first) term of this FROM term.
	*/
	PrimaryTerm() SimpleFromTerm

	/*
	   Represents alias string.
	*/
	Alias() string

	/*
	   Contains correlation reference?
	*/
	IsCorrelated() bool

	/*
	   Get correlation references
	*/
	GetCorrelation() map[string]uint32
}

type SimpleFromTerm interface {
	FromTerm
	SetAnsiJoin()
	SetAnsiNest()
	IsAnsiJoin() bool
	IsAnsiNest() bool
	IsAnsiJoinOp() bool
	SetCommaJoin()
	IsCommaJoin() bool
	JoinHint() JoinHint
	SetJoinHint(joinHint JoinHint)
	PreferHash() bool
	PreferNL() bool
	UnsetJoinProps() uint32
	SetJoinProps(joinProps uint32)
	HasInferJoinHint() bool
	SetInferJoinHint()
	HasTransferJoinHint() bool
	SetTransferJoinHint()
	IsLateralJoin() bool
	SetLateralJoin()
	UnsetLateralJoin()
}

type JoinTerm interface {
	FromTerm
	Left() FromTerm
	// Right() function returns different type for ANSI JOIN and non-ANSI JOIN
	Outer() bool
}

func GetKeyspaceTerm(term SimpleFromTerm) *KeyspaceTerm {
	if term == nil {
		return nil
	}

	switch term := term.(type) {
	case *KeyspaceTerm:
		return term
	case *ExpressionTerm:
		if term.IsKeyspace() {
			return term.KeyspaceTerm()
		}
		return nil
	default:
		return nil
	}
}

func addSimpleTermCorrelation(curCorrelation, newCorrelation map[string]uint32, join bool,
	parent *expression.Formalizer) map[string]uint32 {
	if curCorrelation == nil {
		curCorrelation = make(map[string]uint32, len(newCorrelation))
	}
	for k, v := range newCorrelation {
		// differentiate lateral correlation with nested correlation
		// if the correlation is lateral (with a previous keyspace)
		// then this correlation should not be propagated up
		lateral := false
		if join {
			lateral = !parent.CheckCorrelation(k)
		}
		if lateral {
			v |= expression.IDENT_IS_LATERAL_CORR
		}
		curCorrelation[k] |= v
	}
	return curCorrelation
}

func joinCorrelated(left, right FromTerm) bool {
	if left.IsCorrelated() {
		return true
	}
	if right.IsCorrelated() {
		for _, v := range right.GetCorrelation() {
			// skip lateral correlation
			if (v & expression.IDENT_IS_LATERAL_CORR) == 0 {
				return true
			}
		}
	}
	return false
}

func getJoinCorrelation(left, right FromTerm) map[string]uint32 {
	leftCorrelation := left.GetCorrelation()
	rightCorrelation := right.GetCorrelation()
	correlation := make(map[string]uint32, len(leftCorrelation)+len(rightCorrelation))
	for k, v := range leftCorrelation {
		correlation[k] |= v
	}
	for k, v := range rightCorrelation {
		// skip lateral correlation
		if (v & expression.IDENT_IS_LATERAL_CORR) == 0 {
			correlation[k] |= v
		}
	}
	return correlation
}

func checkLateralCorrelation(term SimpleFromTerm) {
	for _, v := range term.GetCorrelation() {
		if (v & expression.IDENT_IS_LATERAL_CORR) != 0 {
			term.SetLateralJoin()
			return
		}
	}
	term.UnsetLateralJoin()
}
