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

type statementBase struct {
	stmt       Statement
	paramCount int
}

/*
Return the statement subqueries.
*/
func (this *statementBase) Subqueries() ([]*Subquery, errors.Error) {
	return listSubqueries(this.stmt.Expressions())
}

func listSubqueries(exprs expression.Expressions) ([]*Subquery, errors.Error) {
	subqs, err := expression.ListSubqueries(exprs, false)
	if err != nil {
		return nil, errors.NewListSubqueryError(err)
	}
	subqueries := make([]*Subquery, 0, len(subqs))
	for _, subq := range subqs {
		subqueries = append(subqueries, subq.(*Subquery))
	}
	return subqueries, nil
}

/*
Returns all required privileges.
*/
func subqueryPrivileges(exprs expression.Expressions) (*auth.Privileges, errors.Error) {
	subqueries, err := listSubqueries(exprs)
	if err != nil {
		return nil, err
	}
	privileges := auth.NewPrivileges()
	for _, sub := range subqueries {
		sp, e := sub.Select().Privileges()
		if e != nil {
			return nil, e
		}

		privileges.AddAll(sp)
	}

	return privileges, nil
}

/*
unclassified statement
*/
func (this *statementBase) Type() string {
	return ""
}

/*
track parameters
*/
func (this *statementBase) SetParamsCount(params int) {
	this.paramCount = params
}

/*
does it have parameters?
*/
func (this *statementBase) Params() int {
	return this.paramCount
}

/*
Get optimier hints
*/
func (this *statementBase) OptimHints() *OptimHints {
	return nil
}
