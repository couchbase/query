//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package semantics

import (
	"fmt"
	"strings"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/search"
)

func (this *SemChecker) VisitFunction(expr expression.Function) (interface{}, error) {
	switch nexpr := expr.(type) {
	case algebra.Aggregate:
		return expr, this.visitAggregateFunction(nexpr)
	case *search.Search:
		return expr, this.visitSearchFunction(nexpr)
	case *expression.Advisor:
		return expr, this.visitAdvisorFunction(nexpr)
	case *expression.UserDefinedFunction:
		if this.hasSemFlag(_SEM_TRANSACTION) {
			return expr, errors.NewTranFunctionNotSupportedError(nexpr.Name())
		}
	}
	return expr, expr.MapChildren(this)
}

func (this *SemChecker) visitSearchFunction(search *search.Search) (err error) {
	fnName := strings.ToUpper(search.Name()) + "() function"

	if !this.hasSemFlag(_SEM_WHERE | _SEM_ON) {
		return errors.NewSemanticsError(nil,
			fmt.Sprintf("%s is allowed in WHERE/ON clause(s) only.", fnName))
	}

	if err := search.ValidOperands(); err != nil {
		return errors.NewSemanticsError(err, fmt.Sprintf("%s operands are invalid.", fnName))
	}

	return nil
}

func (this *SemChecker) visitAdvisorFunction(advisor *expression.Advisor) (err error) {
	if !this.hasSemFlag(_SEM_ENTERPRISE) {
		return errors.NewEnterpriseFeature("Advisor Function", "semantics.visit_advisor_function")
	}

	if !this.hasSemFlag(_SEM_PROJECTION) {
		return errors.NewAdvisorProjOnly()
	}

	if this.hasSemFlag(_SEM_TRANSACTION) {
		return errors.NewTranFunctionNotSupportedError(advisor.Name())
	}

	this.setSemFlag(_SEM_ADVISOR_FUNC)
	return nil
}
