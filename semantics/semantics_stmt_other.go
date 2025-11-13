//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package semantics

import (
	"strings"

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
				case roles[i].IsScope: // role can be granted/revoked at the scope or bucket (all scopes in the bucket) level
					if typ == _KEYSPACE_TARGET {
						return errors.NewRoleIncorrectLevelError(r, "collection")
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

func validateGroups(groups []string) errors.Error {
	if len(groups) == 0 {
		return nil
	}
	ds := datastore.GetDatastore()
	val, err := ds.GroupInfo()
	if err != nil {
		return err
	}
	gm := make(map[string]bool)
	for _, sg := range val.Actual().([]interface{}) {
		if g, ok := sg.(map[string]interface{}); ok {
			gm[g["id"].(string)] = true
		}
	}
	for _, g := range groups {
		if _, ok := gm[g]; !ok {
			return errors.NewGroupNotFoundError(g)
		}
	}
	return nil
}

func validateGroupRoles(roles []string) errors.Error {
	if len(roles) == 0 {
		return nil
	}
	ds := datastore.GetDatastore()
	all, err := ds.GetRolesAll()
	if err != nil {
		return err
	}

	for _, r := range roles {
		var parts []string
		i := strings.IndexRune(r, '[')
		if i != -1 {
			q := r[i+1 : len(r)-1]
			r = r[:i]
			if q != "" {
				parts = strings.Split(q, ":")
			}
		}
		found := false
		for i := range all {
			if all[i].Name == r {
				found = true
				if all[i].Target == "" && len(parts) != 0 {
					return errors.NewRoleTakesNoKeyspaceError(auth.RoleToAlias(r))
				} else if all[i].Target != "" && len(parts) == 0 {
					return errors.NewRoleRequiresKeyspaceError(auth.RoleToAlias(r))
				}
				break
			}
		}
		if !found {
			return errors.NewRoleNotFoundError(auth.RoleToAlias(r))
		}
	}

	return nil
}

func (this *SemChecker) VisitCreateUser(stmt *algebra.CreateUser) (interface{}, error) {
	parts := strings.Split(stmt.User(), ":")
	if len(parts) == 1 || parts[0] == "local" {
		if _, ok := stmt.Password(); !ok {
			return nil, errors.NewUserAttributeError("local", "password", "required")
		}
	} else if parts[0] == "external" {
		if _, ok := stmt.Password(); ok {
			return nil, errors.NewUserAttributeError("external", "password", "not supported")
		}
	}
	if g, ok := stmt.Groups(); ok {
		return nil, validateGroups(g)
	} else {
		_, p := stmt.Password()
		_, n := stmt.Name()
		if !p && !n {
			return nil, errors.NewMissingAttributesError("user")
		}
	}
	return nil, nil
}

func (this *SemChecker) VisitAlterUser(stmt *algebra.AlterUser) (interface{}, error) {
	parts := strings.Split(stmt.User(), ":")
	if len(parts) > 1 && parts[0] == "external" {
		if _, ok := stmt.Password(); ok {
			return nil, errors.NewUserAttributeError("external", "password", "not supported")
		}
	}
	if g, ok := stmt.Groups(); ok {
		return nil, validateGroups(g)
	} else {
		_, p := stmt.Password()
		_, n := stmt.Name()
		if !p && !n {
			return nil, errors.NewMissingAttributesError("user")
		}
	}
	return nil, nil
}

func (this *SemChecker) VisitDropUser(stmt *algebra.DropUser) (interface{}, error) {
	return nil, nil
}

func (this *SemChecker) VisitCreateGroup(stmt *algebra.CreateGroup) (interface{}, error) {
	if r, ok := stmt.Roles(); ok {
		return nil, validateGroupRoles(r)
	} else {
		_, d := stmt.Desc()
		if !d {
			return nil, errors.NewMissingAttributesError("group")
		}
	}
	return nil, nil
}

func (this *SemChecker) VisitAlterGroup(stmt *algebra.AlterGroup) (interface{}, error) {
	if r, ok := stmt.Roles(); ok {
		return nil, validateGroupRoles(r)
	} else {
		_, d := stmt.Desc()
		if !d {
			return nil, errors.NewMissingAttributesError("group")
		}
	}
	return nil, nil
}

func (this *SemChecker) VisitDropGroup(stmt *algebra.DropGroup) (interface{}, error) {
	return nil, nil
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
	if stmt.Save() && !this.hasSemFlag(_SEM_ENTERPRISE) {
		return nil, errors.NewEnterpriseFeature("SAVE option for PREPARE statement", "semantics_visit_prepare")
	}
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
