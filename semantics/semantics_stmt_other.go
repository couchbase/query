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
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

type roleOp interface {
	Roles() []string
	Keyspaces() []*algebra.KeyspaceRef
}

const (
	_NO_TARGET = 1 << iota
	_KEYSPACE_TARGET
	_SCOPE_TARGET
	_BUCKET_TARGET
)

func validateRoles(op roleOp) errors.Error {
	ds := datastore.GetDatastore()
	roles, err := ds.GetRolesAll()
	if err != nil {
		return err
	}

	typ := 0
	ks := op.Keyspaces()
	if len(ks) == 0 {
		typ = _NO_TARGET
	}
	for i := range ks {
		p := algebra.ParsePath(ks[i].FullName())

		if len(p) == 2 {
			typ |= _BUCKET_TARGET
		} else if len(p) == 3 {
			typ |= _SCOPE_TARGET
		} else {
			typ |= _KEYSPACE_TARGET
		}
	}

outer:
	for _, r := range auth.NormalizeRoleNames(op.Roles()) {
		for i := range roles {
			if roles[i].Name == r {
				switch {
				case roles[i].Target == "": // global role
					if typ != _NO_TARGET {
						return errors.NewRoleTakesNoKeyspaceError(r)
					}
				case roles[i].IsScope: // role can be granted/ revoked at the Scope or Bucket ( all scopes in the bucket ) level
					{
						if typ == _KEYSPACE_TARGET {
							return errors.NewRoleIncorrectLevelError(r, "collection")
						}
					}
				default: // keyspace (at least) required for this role
					if typ == _NO_TARGET {
						return errors.NewRoleRequiresKeyspaceError(r)
					}
				}
				continue outer
			}
		}
		return errors.NewRoleNotFoundError(r)
	}
	return nil
}

func (this *SemChecker) VisitGrantRole(stmt *algebra.GrantRole) (interface{}, error) {
	err := validateRoles(stmt)
	if err != nil {
		return nil, err
	}
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitRevokeRole(stmt *algebra.RevokeRole) (interface{}, error) {
	err := validateRoles(stmt)
	if err != nil {
		return nil, err
	}
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitExplain(stmt *algebra.Explain) (interface{}, error) {
	saveStmtType := stmt.Type()
	defer func() { this.stmtType = saveStmtType }()
	this.stmtType = stmt.Statement().Type()

	return stmt.Statement().Accept(this)
}

func (this *SemChecker) VisitExplainFunction(stmt *algebra.ExplainFunction) (interface{}, error) {
	return nil, nil
}

func (this *SemChecker) VisitAdvise(stmt *algebra.Advise) (interface{}, error) {
	if !this.hasSemFlag(_SEM_ENTERPRISE) {
		return nil, errors.NewEnterpriseFeature("Advise", "semantics.visit_advise")
	}

	saveStmtType := stmt.Type()
	defer func() { this.stmtType = saveStmtType }()
	this.stmtType = stmt.Statement().Type()

	switch stmt.Statement().Type() {
	case "SELECT", "DELETE", "MERGE", "UPDATE":
		return stmt.Statement().Accept(this)
	default:
		return nil, errors.NewAdviseUnsupportedStmtError("semantics.visit_advise")
	}
}

func (this *SemChecker) VisitPrepare(stmt *algebra.Prepare) (interface{}, error) {
	saveStmtType := stmt.Type()
	defer func() { this.stmtType = saveStmtType }()
	this.stmtType = stmt.Statement().Type()
	return stmt.Statement().Accept(this)
}

func (this *SemChecker) VisitExecute(stmt *algebra.Execute) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitInferKeyspace(stmt *algebra.InferKeyspace) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitInferExpression(stmt *algebra.InferExpression) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitUpdateStatistics(stmt *algebra.UpdateStatistics) (interface{}, error) {
	if !this.hasSemFlag(_SEM_ENTERPRISE) {
		return nil, errors.NewEnterpriseFeature("Update Statistics", "semantics.visit_update_statistics")
	}

	for _, expr := range stmt.Terms() {
		if _, ok := expr.(*expression.Self); ok {
			return nil, errors.NewUpdateStatSelf(expr.String(), expr.ErrorContext())
		}
	}

	if err := semCheckFlattenKeys(stmt.Terms()); err != nil {
		return nil, err
	}

	if (stmt.IndexAll() || len(stmt.Indexes()) > 0) &&
		(stmt.Using() != datastore.GSI && stmt.Using() != datastore.DEFAULT) {
		return nil, errors.NewUpdateStatInvalidIndexTypeError()
	}
	if stmt.IndexAll() && !stmt.Keyspace().Path().IsCollection() {
		return nil, errors.NewUpdateStatIndexAllCollectionOnly()
	}
	return nil, stmt.MapExpressions(this)
}
