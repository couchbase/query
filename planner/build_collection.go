//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
)

func getScope(parts ...string) (datastore.Scope, errors.Error) {
	if len(parts) != 4 {
		return nil, errors.NewDatastoreInvalidPathPartsError(parts...)
	}
	return datastore.GetScope(parts[0:3]...)
}

func (this *builder) VisitCreateCollection(stmt *algebra.CreateCollection) (interface{}, error) {
	scope, err := getScope(stmt.Keyspace().Path().Parts()...)
	if err != nil {
		return nil, err
	}
	return plan.NewCreateCollection(scope, stmt), nil
}

func (this *builder) VisitDropCollection(stmt *algebra.DropCollection) (interface{}, error) {
	scope, err := getScope(stmt.Keyspace().Path().Parts()...)
	if err != nil {
		return nil, err
	}
	return plan.NewDropCollection(scope, stmt), nil
}

func (this *builder) VisitFlushCollection(stmt *algebra.FlushCollection) (interface{}, error) {
	ksref := stmt.Keyspace()
	keyspace, err := this.getNameKeyspace(ksref, false)
	if err != nil {
		return nil, err
	}
	return plan.NewFlushCollection(keyspace, stmt), nil
}
