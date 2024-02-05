//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	base "github.com/couchbase/query/plannerbase"
)

func (this *sargable) VisitAnd(pred *expression.And) (interface{}, error) {
	if !this.vector && base.SubsetOf(pred, this.key) {
		return true, nil
	}

	attrs := datastore.IK_NONE
	if this.vector {
		attrs |= datastore.IK_VECTOR
	}
	keys := datastore.IndexKeys{&datastore.IndexKey{this.key, attrs}}
	isArrays := []bool{this.array}
	for _, child := range pred.Operands() {
		var min int
		if this.vector {
			min, _, _, _ = SargableFor(nil, child, this.index, keys, this.missing, this.gsi, isArrays,
				this.context, this.aliases)
		} else {
			min, _, _, _ = SargableFor(child, nil, this.index, keys, this.missing, this.gsi, isArrays,
				this.context, this.aliases)
		}
		if min > 0 {
			return true, nil
		}
	}

	return false, nil
}
