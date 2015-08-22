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
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
)

func (this *builder) beginMutate(keyspace datastore.Keyspace, ksref *algebra.KeyspaceRef,
	keys expression.Expression, indexes algebra.IndexRefs, limit expression.Expression) error {
	ksref.SetDefaultNamespace(this.namespace)
	term := algebra.NewKeyspaceTerm(ksref.Namespace(), ksref.Keyspace(), nil, ksref.As(), keys, indexes)

	this.children = make([]plan.Operator, 0, 8)
	this.subChildren = make([]plan.Operator, 0, 8)

	if this.where != nil {
		limit = nil
	}

	scan, err := this.selectScan(keyspace, term, limit)
	if err != nil {
		return err
	}

	this.children = append(this.children, scan)

	if this.coveringScan != nil {
		coverer := expression.NewCoverer(this.coveringScan.Covers())
		err = this.cover.MapExpressions(coverer)
		if err != nil {
			return err
		}

		if this.where != nil {
			this.where, err = coverer.Map(this.where)
			if err != nil {
				return err
			}
		}
	} else {
		fetch := plan.NewFetch(keyspace, term)
		this.subChildren = append(this.subChildren, fetch)
	}

	if this.where != nil {
		this.subChildren = append(this.subChildren, plan.NewFilter(this.where))
	}

	return nil
}
