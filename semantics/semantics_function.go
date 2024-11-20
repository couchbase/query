//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
	case *expression.TimeSeries, *expression.Knn:
		if ve, ok := nexpr.(interface{ ValidOperands() error }); ok {
			err := ve.ValidOperands()
			if err != nil {
				return expr, errors.NewSemanticsWithCauseError(err,
					fmt.Sprintf("%s() function operands are invalid.", strings.ToUpper(nexpr.Name())))
			}
		} else {
			return expr, errors.NewSemanticsInternalError(
				fmt.Sprintf("%s() function does not have ValidOperands() function.",
					strings.ToUpper(nexpr.Name())))
		}
		return expr, nil
	case *expression.FlattenKeys:
		if this.stmtType != "CREATE_INDEX" && this.stmtType != "UPDATE_STATISTICS" {
			return expr, errors.NewFlattenKeys(nexpr.String(), nexpr.ErrorContext())
		}
		/*	case *expression.UserDefinedFunction:
			if this.hasSemFlag(_SEM_TRANSACTION) {
				return expr, errors.NewTranFunctionNotSupportedError(nexpr.Name())
			}
		*/
	case *expression.SequenceOperation:
		if !nexpr.IsNameValid() {
			return expr, errors.NewSequenceError(errors.E_SEQUENCE_NAME_PARTS, nexpr.FullName(), nexpr.ErrorContext())
		}
		if this.hasSemFlag(_SEM_WHERE | _SEM_ON) {
			return nil, errors.NewSemanticsError(nil, "Sequence operations are not allowed in WHERE/ON clauses")
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

func (this *SemChecker) VisitAll(expr *expression.All) (interface{}, error) {

	if this.stmtType != "CREATE_INDEX" && this.stmtType != "UPDATE_STATISTICS" {
		return expr, errors.NewAllDistinctNotAllowed(expr.String(), expr.ErrorContext())
	}
	return expr, expr.MapChildren(this)
}
