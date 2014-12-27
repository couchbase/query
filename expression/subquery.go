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
Used to implement subqueries. Type Subquery is
an interface that inherits from Expression. It
also implements a method Formalize that takes
as input a type Formalizer, and returns an error.
*/
type Subquery interface {
	Expression

	Formalize(parent *Formalizer) error
}
