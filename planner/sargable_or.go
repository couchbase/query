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

func (this *sargable) VisitOr(pred *expression.Or) (interface{}, error) {
	if base.SubsetOf(pred, this.key.Expr) {
		return true, nil
	}

	keys := datastore.IndexKeys{this.key}
	isArrays := []bool{this.array}
	for _, child := range pred.Operands() {
		if min, _, _, _ := SargableFor(child, keys, this.missing, this.gsi, isArrays,
			this.context, this.aliases); min <= 0 {
			return false, nil
		}
	}

	return true, nil
}
