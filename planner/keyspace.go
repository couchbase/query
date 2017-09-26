//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"github.com/couchbase/query/expression"
)

type baseKeyspace struct {
	name        string
	filters     Filters
	joinfilters Filters
	dnfPred     expression.Expression
	origPred    expression.Expression
}

func newBaseKeyspace(keyspace string) *baseKeyspace {
	rv := &baseKeyspace{
		name: keyspace,
	}

	return rv
}

// get a map of baseKeysapces for a single keyspace name
func getOneBaseKeyspaces(keyspaceName string) map[string]*baseKeyspace {
	dest := make(map[string]*baseKeyspace, 1)
	dest[keyspaceName] = newBaseKeyspace(keyspaceName)

	return dest
}
