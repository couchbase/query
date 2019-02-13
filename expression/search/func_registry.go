//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
	"search_query": &FTSQuery{},
	"search":       &Search{},
	"search_meta":  &SearchMeta{},
	"search_score": &SearchScore{},
}
