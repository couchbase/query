//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

type statementBase struct {
	stmt Statement
}

/*
Returns all required privileges.
*/
func subqueryPrivileges(exprs expression.Expressions) (*auth.Privileges, errors.Error) {
	subqueries, err := expression.ListSubqueries(exprs, false)
	if err != nil {
		return nil, errors.NewError(err, "")
	}

	privileges := auth.NewPrivileges()
	for _, s := range subqueries {
		sub := s.(*Subquery)
		sp, e := sub.Select().Privileges()
		if e != nil {
			return nil, e
		}

		privileges.AddAll(sp)
	}

	return privileges, nil
}

/*
	unclassified
*/
func (this *statementBase) Type() string {
	return ""
}
