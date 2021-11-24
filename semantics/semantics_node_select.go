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
	"github.com/couchbase/query/expression"
)

func (this *SemChecker) VisitSelectTerm(node *algebra.SelectTerm) (interface{}, error) {
	return node.Select().Accept(this)
}

func (this *SemChecker) VisitSubselect(node *algebra.Subselect) (r interface{}, err error) {
	saveSemFlag := this.semFlag
	defer func() { this.semFlag = saveSemFlag }()
	this.unsetSemFlag(_SEM_WHERE | _SEM_ON | _SEM_PROJECTION | _SEM_ADVISOR_FUNC)
	if node.With() != nil {
		if err = node.With().MapExpressions(this); err != nil {
			return nil, err
		}
	}

	if node.From() != nil {
		if r, err = node.From().Accept(this); err != nil {
			return r, err
		}
	}

	if node.Let() != nil {
		if err = node.Let().MapExpressions(this); err != nil {
			return nil, err
		}
	}

	if node.Where() != nil {
		this.setSemFlag(_SEM_WHERE)
		_, err = this.Map(node.Where())
		this.unsetSemFlag(_SEM_WHERE)
		if err != nil {
			return nil, err
		}
	}

	if node.Group() != nil {
		if err = node.Group().MapExpressions(this); err != nil {
			return nil, err
		}
	}

	this.setSemFlag(_SEM_PROJECTION)
	err = node.Projection().MapExpressions(this)
	this.unsetSemFlag(_SEM_PROJECTION)
	if err != nil {
		return nil, err
	}

	if this.hasSemFlag(_SEM_ADVISOR_FUNC) {
		if node.From() != nil {
			return nil, errors.NewAdvisorNoFrom()
		}
	}

	return nil, nil
}

func (this *SemChecker) VisitSubquery(expr expression.Subquery) (r interface{}, err error) {
	if node, ok := expr.(*algebra.Subquery); ok {
		_, err = node.Select().Accept(this)
	}
	return expr, err
}
