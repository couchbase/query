//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*
Package algebra provides a syntax-independent algebra. Any language
flavor or syntax that can be converted to this algebra can then be
processed by the query engine.
*/
package algebra

import (
	_ "github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type Statement interface {
	Accept(visitor Visitor) (interface{}, error)
	Signature() value.Value
	Formalize() error
	//MapExpressions(mapper expression.Mapper) error
}

type Node interface {
	Accept(visitor NodeVisitor) (interface{}, error)
}
