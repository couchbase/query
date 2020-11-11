//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
	default:
		return expr, expr.MapChildren(this)
	}
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
