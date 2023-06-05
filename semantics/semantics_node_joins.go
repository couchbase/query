//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package semantics

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
)

func (this *SemChecker) visitJoin(left algebra.FromTerm, right algebra.SimpleFromTerm) error {
	_, err := left.Accept(this)
	if err != nil {
		return err
	}
	_, err = right.Accept(this)
	if err != nil {
		return err
	}
	return nil
}

func (this *SemChecker) VisitJoin(node *algebra.Join) (interface{}, error) {
	left := skipUnnest(node.Left())
	switch left := left.(type) {
	case *algebra.AnsiJoin:
		return nil, errors.NewMixedJoinError("ANSI JOIN", left.Alias(), "non ANSI JOIN",
			node.Alias(), "semantics.visit_join.ansi_mixed_join")
	case *algebra.AnsiNest:
		return nil, errors.NewMixedJoinError("ANSI NEST", left.Alias(), "non ANSI JOIN",
			node.Alias(), "semantics.visit_join.ansi_mixed_join")
	}

	right := node.Right()
	if right.JoinHint() != algebra.JOIN_HINT_NONE {
		return nil, errors.NewJoinNestNoJoinHintError("JOIN", right.Alias(), "semantics.visit_join.no_join_hint")
	}
	if right.Keys() != nil {
		return nil, errors.NewJoinNestNoUseKeysError("JOIN", right.Alias(), "semantics.visit_join.no_use_keys")
	}
	if right.Indexes() != nil {
		return nil, errors.NewJoinNestNoUseIndexError("JOIN", right.Alias(), "semantics.visit_join.no_use_index")
	}

	if this.hasSemFlag(_SEM_WITH_RECURSIVE) && node.Outer() {
		// LEFT, RIGHT,Outer JOIN not allowed as CTE can become infinite recursion
		return nil, errors.NewRecurionUnsupportedError("OUTER JOIN", "may lead to potential infinite recursion")
	}

	return nil, this.visitJoin(node.Left(), node.Right())
}

func (this *SemChecker) VisitIndexJoin(node *algebra.IndexJoin) (interface{}, error) {
	left := skipUnnest(node.Left())
	switch left := left.(type) {
	case *algebra.AnsiJoin:
		return nil, errors.NewMixedJoinError("ANSI JOIN", left.Alias(), "non ANSI JOIN",
			node.Alias(), "semantics.visit_index_join.ansi_mixed_join")
	case *algebra.AnsiNest:
		return nil, errors.NewMixedJoinError("ANSI NEST", left.Alias(), "non ANSI JOIN",
			node.Alias(), "semantics.visit_index_join.ansi_mixed_join")
	}

	right := node.Right()
	if right.JoinHint() != algebra.JOIN_HINT_NONE {
		return nil, errors.NewJoinNestNoJoinHintError("JOIN", right.Alias(), "semantics.visit_index_join.no_join_hint")
	}
	if right.Keys() != nil {
		return nil, errors.NewJoinNestNoUseKeysError("JOIN", right.Alias(), "semantics.visit_index_join.no_use_keys")
	}
	if right.Indexes() != nil {
		return nil, errors.NewJoinNestNoUseIndexError("JOIN", right.Alias(), "semantics.visit_index_join.no_use_index")
	}

	if this.hasSemFlag(_SEM_WITH_RECURSIVE) && node.Outer() {
		// // LEFT, RIGHT,Outer JOIN not allowed as CTE can become infinite recursion
		return nil, errors.NewRecurionUnsupportedError("OUTER JOIN", "may lead to potential infinite recursion")

	}

	return nil, this.visitJoin(node.Left(), node.Right())
}

func (this *SemChecker) VisitAnsiJoin(node *algebra.AnsiJoin) (r interface{}, err error) {
	left := skipUnnest(node.Left())
	switch left := left.(type) {
	case *algebra.Join, *algebra.IndexJoin:
		return nil, errors.NewMixedJoinError("non ANSI JOIN", left.Alias(), "ANSI JOIN",
			node.Alias(), "semantics.visit_ansi_join.ansi_mixed_join")
	case *algebra.Nest, *algebra.IndexNest:
		return nil, errors.NewMixedJoinError("non ANSI NEST", left.Alias(), "ANSI JOIN",
			node.Alias(), "semantics.visit_ansi_join.ansi_mixed_join")
	}

	if err = this.visitJoin(node.Left(), node.Right()); err != nil {
		return nil, err
	}

	if !node.IsCommaJoin() {
		this.setSemFlag(_SEM_ON)
		_, err = this.Map(node.Onclause())
		this.unsetSemFlag(_SEM_ON)
	}

	if this.hasSemFlag(_SEM_WITH_RECURSIVE) && node.Outer() {
		// // LEFT, RIGHT,Outer JOIN not allowed as CTE can become infinite recursion
		return nil, errors.NewRecurionUnsupportedError("OUTER JOIN", "may lead to potential infinite recursion")
	}

	return nil, err
}

func (this *SemChecker) VisitNest(node *algebra.Nest) (interface{}, error) {
	left := skipUnnest(node.Left())
	switch left := left.(type) {
	case *algebra.AnsiJoin:
		return nil, errors.NewMixedJoinError("ANSI JOIN", left.Alias(), "non ANSI NEST",
			node.Alias(), "semantics.visit_nest.ansi_mixed_join")
	case *algebra.AnsiNest:
		return nil, errors.NewMixedJoinError("ANSI NEST", left.Alias(), "non ANSI NEST",
			node.Alias(), "semantics.visit_nest.ansi_mixed_join")
	}

	right := node.Right()
	if right.JoinHint() != algebra.JOIN_HINT_NONE {
		return nil, errors.NewJoinNestNoJoinHintError("NEST", right.Alias(), "semantics.visit_nest.no_join_hint")
	}
	if right.Keys() != nil {
		return nil, errors.NewJoinNestNoUseKeysError("NEST", right.Alias(), "semantics.visit_nest.no_use_keys")
	}
	if right.Indexes() != nil {
		return nil, errors.NewJoinNestNoUseIndexError("NEST", right.Alias(), "semantics.visit_nest.no_use_index")
	}

	return nil, this.visitJoin(node.Left(), node.Right())
}

func (this *SemChecker) VisitIndexNest(node *algebra.IndexNest) (interface{}, error) {
	left := skipUnnest(node.Left())
	switch left := left.(type) {
	case *algebra.AnsiJoin:
		return nil, errors.NewMixedJoinError("ANSI JOIN", left.Alias(), "non ANSI NEST",
			node.Alias(), "semantics.visit_index_nest.ansi_mixed_join")
	case *algebra.AnsiNest:
		return nil, errors.NewMixedJoinError("ANSI NEST", left.Alias(), "non ANSI NEST",
			node.Alias(), "semantics.visit_index_nest.ansi_mixed_join")
	}

	right := node.Right()
	if right.JoinHint() != algebra.JOIN_HINT_NONE {
		return nil, errors.NewJoinNestNoJoinHintError("NEST", right.Alias(), "semantics.visit_index_nest.no_join_hint")
	}
	if right.Keys() != nil {
		return nil, errors.NewJoinNestNoUseKeysError("NEST", right.Alias(), "semantics.visit_index_nest.no_use_keys")
	}
	if right.Indexes() != nil {
		return nil, errors.NewJoinNestNoUseIndexError("NEST", right.Alias(), "semantics.visit_index_nest.no_use_index")
	}

	return nil, this.visitJoin(node.Left(), node.Right())
}

func (this *SemChecker) VisitAnsiNest(node *algebra.AnsiNest) (r interface{}, err error) {
	left := skipUnnest(node.Left())
	switch left := left.(type) {
	case *algebra.Join, *algebra.IndexJoin:
		return nil, errors.NewMixedJoinError("non ANSI JOIN", left.Alias(), "ANSI NEST",
			node.Alias(), "semantics.visit_ansi_nest.ansi_mixed_join")
	case *algebra.Nest, *algebra.IndexNest:
		return nil, errors.NewMixedJoinError("non ANSI NEST", left.Alias(), "ANSI NEST",
			node.Alias(), "semantics.visit_ansi_nest.ansi_mixed_join")
	}

	if err = this.visitJoin(node.Left(), node.Right()); err != nil {
		return nil, err
	}

	this.setSemFlag(_SEM_ON)
	_, err = this.Map(node.Onclause())
	this.unsetSemFlag(_SEM_ON)

	return nil, err
}

func (this *SemChecker) VisitUnnest(node *algebra.Unnest) (interface{}, error) {
	_, err := node.Left().Accept(this)
	if err != nil {
		return nil, err
	}
	_, err = this.Map(node.Expression())
	return nil, err
}

func skipUnnest(node algebra.FromTerm) algebra.FromTerm {
	for {
		if unnest, ok := node.(*algebra.Unnest); ok {
			node = unnest.Left()
		} else {
			return node
		}
	}

	return node
}
