//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"github.com/couchbaselabs/query/expression"
)

/*
Type Exists is a struct that inherits from expression.Exists to 
set LIMIT 1 on subqueries.
*/
type Exists struct {
	expression.Exists
}

/*
The function NewExists uses the NewExists method to 
create a new Exists function with one operand. If that
operand is a subquery, that has no limit defined, set it
to one expression (defined in expressions).
*/
func NewExists(operand expression.Expression) *Exists {
	rv := &Exists{
		*expression.NewExists(operand),
	}

	switch o := operand.(type) {
	case *Subquery:
		if o.query.Limit() == nil {
			o.query.SetLimit(expression.ONE_EXPR)
		}
	}

	return rv
}
