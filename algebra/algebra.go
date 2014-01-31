//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

Package algebra provides a language-independent algebra. Any language
flavor or syntax that can be converted to this algebra can then be
processed by the query engine.

*/
package algebra

import (
	_ "fmt"
	"time"

	"github.com/couchbaselabs/query/value"
)

type Node interface {
	//fmt.Stringer
	Accept(visitor Visitor) (interface{}, error)
}

type Context interface {
	Now() time.Time
	Argument(parameter string) value.Value
	EvaluateSubquery(query *Select, item value.Value) (value.Value, error)
}

type ResultTerm struct {
	star bool       `json:"star"`
	expr Expression `json:"expr"`
	as   string     `json:"as"`
}

type ResultTermList []*ResultTerm
