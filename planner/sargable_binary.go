//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/expression"
)

func (this *sargable) visitBinary(pred expression.BinaryFunction) (bool, error) {
	key := this.key.Expr
	return pred.First().EquivalentTo(key) ||
			pred.Second().EquivalentTo(key) ||
			this.defaultSargable(pred),
		nil
}
