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

/*
Type ExpressionBase is defined as an empty struct. 
*/
type ExpressionBase struct {
}

/*
It returns an empty string for the terminal identifier of
the expression.
*/
func (this *ExpressionBase) Alias() string {
	return ""
}

/*
Range over the children of the expression, and check if each
child is indexable. If not then return false as the expression
is not indexable. If all children are indexable, then return 
true.
*/
func (this *ExpressionBase) indexable(expr Expression) bool {
	for _, child := range expr.Children() {
		if !child.Indexable() {
			return false
		}
	}

	return true
}

/*
Check if two expressions are equivalent. First compare the dynamic
type information of the two expressions, using reflect.TypeOf. If 
it is not the same, then return false. Compare the length of the 
two expressions. If they are not the same, then not equal, hence
return false. If the lengths are equal, range through the children 
and check if they are equivalent by calling the EquivalentTo method
for each set of children and return false if not equal. If the 
method hasnt returned till this point, then the expressions are
equal and return true.
*/
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

/*
This has not been implemented and calls EquivalentTo.
*/
func (this *ExpressionBase) subsetOf(expr, other Expression) bool {
	return expr.EquivalentTo(other)
}
