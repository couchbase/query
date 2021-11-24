//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package search

import (
	"strings"

	"github.com/couchbase/query/expression"
)

/*
This method is used to retrieve a function by the parser.
Based on the input string name it looks through a map and
retrieves the function that corresponds to it. If the
function exists it returns true and the function. While
looking into the map, convert the string name to lower
case.
*/
func GetSearchFunction(name string) (expression.Function, bool) {
	rv, ok := _FUNCTIONS[strings.ToLower(name)]
	return rv, ok
}

/*
The variable _FUNCTIONS represents a map from string to
Function. Each string returns a pointer to that function.
*/
var _FUNCTIONS = map[string]expression.Function{
	"search":       &Search{},
	"search_meta":  &SearchMeta{},
	"search_score": &SearchScore{},
}
