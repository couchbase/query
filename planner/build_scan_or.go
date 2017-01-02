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

func (this *builder) buildOrScan(keyspace datastore.Keyspace, node *algebra.KeyspaceTerm,
	id expression.Expression, pred *expression.Or, limit expression.Expression,
	indexes []datastore.Index, primaryKey expression.Expressions, formalizer *expression.Formalizer) (
	plan.Operator, int, error) {

	where := this.where
	defer func() {
		this.where = where
	}()

	this.cover = nil
	this.resetCountMin()

	if this.order != nil {
		this.resetOrderLimit()
		limit = nil
	} else {
		this.order = nil
	}

	minSargLength := 0
	scans := make([]plan.Operator, 0, len(pred.Operands()))

	for _, op := range pred.Operands() {
		this.where = op
		scan, termSargLength, err := this.buildTermScan(keyspace, node, id, op, limit, indexes, primaryKey, formalizer)
		if scan == nil || err != nil {
			return nil, 0, err
		}

		scans = append(scans, scan)

		if minSargLength == 0 || minSargLength > termSargLength {
			minSargLength = termSargLength
		}
	}

	return plan.NewUnionScan(scans...), minSargLength, nil
}
