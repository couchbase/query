//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package plannerbase

import (
	"github.com/couchbase/query/expression"
)

func LessThan(expr1, expr2 expression.Expression) bool {
	value1 := expr1.Value()
	value2 := expr2.Value()

	return value1 != nil && value2 != nil &&
		value1.Collate(value2) < 0
}

func LessThanOrEquals(expr1, expr2 expression.Expression) bool {
	return LessThan(expr1, expr2) || expr1.EquivalentTo(expr2)
}
