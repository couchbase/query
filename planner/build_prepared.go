//  Copyright (c) 2014 Couchbase, Inc.
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
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

func BuildPrepared(stmt algebra.Statement, datastore, systemstore datastore.Datastore,
	namespace string, subquery, stream bool, namedArgs map[string]value.Value, positionalArgs value.Values,
	indexApiVersion int, featureControls uint64) (*plan.Prepared, error) {
	operator, err := Build(stmt, datastore, systemstore, namespace, subquery, stream, namedArgs, positionalArgs,
		indexApiVersion, featureControls)
	if err != nil {
		return nil, err
	}

	signature := stmt.Signature()
	return plan.NewPrepared(operator, signature), nil
}
