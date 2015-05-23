//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plan

import (
	"math"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/planner"
	"github.com/couchbase/query/util"
)

func (this *builder) selectScan(keyspace datastore.Keyspace,
	node *algebra.KeyspaceTerm) (op Operator, err error) {
	keys := node.Keys()
	if keys != nil {
		switch keys := keys.(type) {
		case *expression.ArrayConstruct:
			this.maxParallelism = util.MaxInt(1, len(keys.Operands()))
		case *algebra.NamedParameter, *algebra.PositionalParameter:
			this.maxParallelism = 0
		default:
			this.maxParallelism = 1
		}

		scan := NewKeyScan(keys)
		return scan, nil
	}

	this.maxParallelism = 0 // Default behavior for index scans

	secondary, primary, err := planner.BuildScan(keyspace, node, this.where)
	if err != nil {
		return nil, err
	}

	if primary != nil {
		return NewPrimaryScan(primary, keyspace, node), nil
	}

	scans := make([]Operator, 0, len(secondary))
	var scan Operator
	for index, spans := range secondary {
		scan = NewIndexScan(index, node, spans, false, math.MaxInt64)
		if len(spans) > 1 {
			// Use UnionScan to de-dup multiple spans
			scan = NewUnionScan(scan)
		}

		scans = append(scans, scan)
	}

	if len(scans) > 1 {
		return NewIntersectScan(scans...), nil
	} else {
		return scans[0], nil
	}
}
