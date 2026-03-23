//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

// This function is used to parse expressions, and if there are subquery plans available when
// unmarshalling prepared statements, it handles the matching of parsed subquery expression and
// unmarshalled subquery plan
func parseWithContext(s string, plContext *planContext) (expression.Expression, error) {
	expr, err := parser.Parse(s)
	if expr == nil || err != nil {
		return expr, err
	}
	if plContext != nil && plContext.hasSubqueryMap() {
		expr, err = plContext.checkSubqueryMap(expr)
	}
	return expr, err
}
