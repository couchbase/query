//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
	switch left := node.Left().(type) {
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

	return nil, this.visitJoin(node.Left(), node.Right())
}

func (this *SemChecker) VisitIndexJoin(node *algebra.IndexJoin) (interface{}, error) {
	switch left := node.Left().(type) {
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

	return nil, this.visitJoin(node.Left(), node.Right())
}

func (this *SemChecker) VisitAnsiJoin(node *algebra.AnsiJoin) (r interface{}, err error) {
	switch left := node.Left().(type) {
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

	this.setSemFlag(_SEM_ON)
	_, err = this.Map(node.Onclause())
	this.unsetSemFlag(_SEM_ON)

	return nil, err
}

func (this *SemChecker) VisitNest(node *algebra.Nest) (interface{}, error) {
	switch left := node.Left().(type) {
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
	switch left := node.Left().(type) {
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
	switch left := node.Left().(type) {
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
