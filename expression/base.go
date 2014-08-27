//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"reflect"
)

type ExpressionBase struct {
}

func (this *ExpressionBase) Alias() string {
	return ""
}

func (this *ExpressionBase) equivalentTo(expr, other Expression) bool {
	if reflect.TypeOf(expr) != reflect.TypeOf(other) {
		return false
	}

	ours := expr.Children()
	theirs := other.Children()

	if len(ours) != len(theirs) {
		return false
	}

	for i, child := range ours {
		if !child.EquivalentTo(theirs[i]) {
			return false
		}
	}

	return true
}

func (this *ExpressionBase) subsetOf(expr, other Expression) bool {
	return expr.EquivalentTo(other)
}
