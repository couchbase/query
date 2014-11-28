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
	"github.com/couchbaselabs/query/value"
)

/*
Expression path. The type Paths is a slice of Path.
*/
type Paths []Path

/*
Path is of type interface that inherits Expression.
It also contains 2 methods Set and Unset. They take
as input an item, value or type Value, and a context
and return a boolean value that depicts if the path
was set or unset.
*/
type Path interface {
	Expression
	Set(item, val value.Value, context Context) bool
	Unset(item value.Value, context Context) bool
}
