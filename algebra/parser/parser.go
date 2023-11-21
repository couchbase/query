//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

/*
Package parser parses optimizer hint strings into algebra.OptimHints.
*/
package parser

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/parser/n1ql"
)

// s is expected to start with a PLUS sign, e.g. "+ INDEX(default ix1)"
func Parse(s string) *algebra.OptimHints {
	return n1ql.ParseOptimHints(s)
}
