//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"github.com/couchbase/query/expression"
)

/*
Type Exists is a struct that inherits from expression.Exists to
set LIMIT 1 on subqueries.
*/
type Exists struct {
	expression.Exists
}

/*
The function NewExists uses the NewExists method to
create a new Exists function with one operand. If that
operand is a subquery, that has no limit defined, set it
to one expression (defined in expressions).
*/
func NewExists(operand expression.Expression) *Exists {
	rv := &Exists{
		*expression.NewExists(operand),
	}

	switch o := operand.(type) {
	case *Subquery:
		if o.query.Limit() == nil {
			o.query.SetLimit(expression.ONE_EXPR)
		}
	}

	return rv
}
