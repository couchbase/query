//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
