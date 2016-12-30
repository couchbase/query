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
	"strings"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
)

func Build(stmt algebra.Statement, datastore, systemstore datastore.Datastore,
	namespace string, subquery bool) (plan.Operator, error) {
	builder := newBuilder(datastore, systemstore, namespace, subquery)
	o, err := stmt.Accept(builder)

	if err != nil {
		return nil, err
	}

	op := o.(plan.Operator)
	_, is_prepared := o.(*plan.Prepared)

	if !subquery && !is_prepared {
		privs, er := stmt.Privileges()
		if er != nil {
			return nil, er
		}

		if len(privs) > 0 {
			op = plan.NewAuthorize(privs, op)
		}

		return plan.NewSequence(op, plan.NewStream()), nil
	} else {
		return op, nil
	}
}

type builder struct {
	datastore       datastore.Datastore
	systemstore     datastore.Datastore
	namespace       string
	subquery        bool
	correlated      bool
	maxParallelism  int
	delayProjection bool                  // Used to allow ORDER BY non-projected expressions
	from            algebra.FromTerm      // Used for index selection
	where           expression.Expression // Used for index selection
	order           *algebra.Order        // Used to collect aggregates from ORDER BY, and for ORDER pushdown
	limit           expression.Expression // Used for LIMIT pushdown
	countAgg        *algebra.Count        // Used for COUNT() pushdown to IndexCountScan
	minAgg          *algebra.Min          // Used for MIN() pushdown to IndexScan
	distinct        bool
	children        []plan.Operator
	subChildren     []plan.Operator
	cover           expression.HasExpressions
	coveringScans   []plan.Operator
	coveredUnnests  map[*algebra.Unnest]bool
	coveredLets     expression.Expressions
	countScan       *plan.IndexCountScan
}

func newBuilder(datastore, systemstore datastore.Datastore, namespace string, subquery bool) *builder {
	rv := &builder{
		datastore:       datastore,
		systemstore:     systemstore,
		namespace:       namespace,
		subquery:        subquery,
		delayProjection: false,
	}

	return rv
}

func (this *builder) getTermKeyspace(node *algebra.KeyspaceTerm) (datastore.Keyspace, error) {
	node.SetDefaultNamespace(this.namespace)
	ns := node.Namespace()

	datastore := this.datastore
	if strings.ToLower(ns) == "#system" {
		datastore = this.systemstore
	}

	namespace, err := datastore.NamespaceByName(ns)
	if err != nil {
		return nil, err
	}

	keyspace, err := namespace.KeyspaceByName(node.Keyspace())
	if err != nil {
		return nil, err
	}

	return keyspace, nil
}
