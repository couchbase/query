//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plannerbase

import (
	"github.com/couchbase/query/expression"
)

func (this *subset) VisitEvery(expr *expression.Every) (interface{}, error) {
	switch expr2 := this.expr2.(type) {
	case *expression.Every:
		return expr.Bindings().SubsetOf(expr2.Bindings()) &&
			SubsetOf(expr.Satisfies(), expr2.Satisfies()), nil
	case *expression.AnyEvery:
		return expr.Bindings().SubsetOf(expr2.Bindings()) &&
			SubsetOf(expr.Satisfies(), expr2.Satisfies()), nil
	default:
		return this.visitDefault(expr)
	}
}
