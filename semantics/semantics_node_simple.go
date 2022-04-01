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

func (this *SemChecker) VisitKeyspaceTerm(node *algebra.KeyspaceTerm) (interface{}, error) {
	if this.hasSemFlag(_SEM_FROM) && !node.IsAnsiJoinOp() && !node.HasTransferJoinHint() &&
		node.JoinHint() != algebra.JOIN_HINT_NONE {
		return nil, errors.NewFirstTermJoinHintError(node.Alias())
	}
	return nil, node.MapExpressions(this)
}

func (this *SemChecker) VisitExpressionTerm(node *algebra.ExpressionTerm) (interface{}, error) {
	if node.IsKeyspace() {
		return node.KeyspaceTerm().Accept(this)
	}
	if this.hasSemFlag(_SEM_FROM) && !node.IsAnsiJoinOp() && !node.HasTransferJoinHint() &&
		node.JoinHint() != algebra.JOIN_HINT_NONE {
		return nil, errors.NewFirstTermJoinHintError(node.Alias())
	}
	return node.ExpressionTerm().Accept(this)
}

func (this *SemChecker) VisitSubqueryTerm(node *algebra.SubqueryTerm) (interface{}, error) {
	if this.hasSemFlag(_SEM_FROM) && !node.IsAnsiJoinOp() && !node.HasTransferJoinHint() &&
		node.JoinHint() != algebra.JOIN_HINT_NONE {
		return nil, errors.NewFirstTermJoinHintError(node.Alias())
	}
	return node.Subquery().Accept(this)
}
