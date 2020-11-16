//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plannerbase

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

func GetStaticInt(expr expression.Expression) (int64, bool) {
	if expr != nil {
		expVal := expr.Value()
		if expVal != nil {
			switch evt := expVal.Actual().(type) {
			case float64:
				return int64(evt), true
			}
		}
	}

	return 0, false
}

func ReplaceParameters(pred expression.Expression, namedArgs map[string]value.Value,
	positionalArgs value.Values) (expression.Expression, error) {

	if pred == nil || (len(namedArgs) == 0 && len(positionalArgs) == 0) {
		return pred, nil
	}

	var err error

	pred = pred.Copy()

	for name, value := range namedArgs {
		nameExpr := algebra.NewNamedParameter(name)
		valueExpr := expression.NewConstant(value)
		pred, err = expression.ReplaceExpr(pred, nameExpr, valueExpr)
		if err != nil {
			return nil, err
		}
	}

	for pos, value := range positionalArgs {
		posExpr := algebra.NewPositionalParameter(pos + 1)
		valueExpr := expression.NewConstant(value)
		pred, err = expression.ReplaceExpr(pred, posExpr, valueExpr)
		if err != nil {
			return nil, err
		}
	}

	return pred, nil
}
