//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

/*
Package parser parses N1QL strings into expressions.
*/
package parser

import (
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/parser/n1ql"
)

func Parse(s string) (expression.Expression, error) {
	return n1ql.ParseExpression(s)
}

func ParseUdf(s string) (expression.Expression, error) {
	return n1ql.ParseExpressionUdf(s)
}
