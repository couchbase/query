//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

/*
This function returns a Function that represents the
greater than equal to comparison expression. It is
implemented using the NewLE function with its input
operand expressions switched. (The first greater than
equal to the second operand is the same as the second
less than equal to the first operand.). NewLE implements
a function thatn represents the less than equal to
comparison expression.
*/
func NewGE(first, second Expression) Function {
	return NewLE(second, first)
}
