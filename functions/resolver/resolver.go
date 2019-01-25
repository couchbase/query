//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package resolver

import (
	"encoding/json"
	go_errors "errors"
	"fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/functions/inline"
)

func MakeBody(name string, bytes []byte) (functions.FunctionBody, errors.Error) {
	var language_type struct {
		Language string `json:"#language"`
	}

	err := json.Unmarshal(bytes, &language_type)
	if err != nil {
		return nil, errors.NewFunctionEncodingError("decode body", name, err)
	}
	switch language_type.Language {
	case "inline":

		var expr expression.Expression
		var _unmarshalled struct {
			_          string   `json:"#language"`
			Parameters []string `json:"parameters"`
			Expression string   `json:"expression"`
		}
		err := json.Unmarshal(bytes, &_unmarshalled)
		if err != nil {
			return nil, errors.NewFunctionEncodingError("decode body", name, err)
		}
		if _unmarshalled.Expression != "" {
			expr, err = parser.Parse(_unmarshalled.Expression)
			if err != nil {
				return nil, errors.NewFunctionEncodingError("decode body", name, err)
			}
		} else {
			return nil, errors.NewFunctionEncodingError("decode body", name, go_errors.New("expression is missing"))
		}
		body, newErr := inline.NewInlineBody(expr)
		if body != nil {
			body.SetVarNames(_unmarshalled.Parameters)
		}
		return body, newErr

	default:
		return nil, errors.NewFunctionEncodingError("decode body", "unknown", fmt.Errorf("unknown language %v", language_type.Language))
	}
}
