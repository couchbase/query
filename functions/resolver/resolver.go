//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package resolver

import (
	"encoding/json"
	go_errors "errors"
	"fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/functions/golang"
	"github.com/couchbase/query/functions/inline"
	"github.com/couchbase/query/functions/javascript"
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
			newErr = body.SetVarNames(_unmarshalled.Parameters)
		}
		return body, newErr

	case "golang":

		var _unmarshalled struct {
			_          string   `json:"#language"`
			Parameters []string `json:"parameters"`
			Library    string   `json:"library"`
			Object     string   `json:"object"`
		}
		err := json.Unmarshal(bytes, &_unmarshalled)
		if err != nil {
			return nil, errors.NewFunctionEncodingError("decode body", name, err)
		}
		if _unmarshalled.Object == "" || _unmarshalled.Library == "" {
			return nil, errors.NewFunctionEncodingError("decode body", name, go_errors.New("object is missing"))
		}
		body, newErr := golang.NewGolangBody(_unmarshalled.Library, _unmarshalled.Object)
		if body != nil {
			newErr = body.SetVarNames(_unmarshalled.Parameters)
		}
		return body, newErr

	case "javascript":

		var _unmarshalled struct {
			_          string   `json:"#language"`
			Parameters []string `json:"parameters"`
			Library    string   `json:"library"`
			Object     string   `json:"object"`
		}
		err := json.Unmarshal(bytes, &_unmarshalled)
		if err != nil {
			return nil, errors.NewFunctionEncodingError("decode body", name, err)
		}
		if _unmarshalled.Object == "" || _unmarshalled.Library == "" {
			return nil, errors.NewFunctionEncodingError("decode body", name, go_errors.New("object is missing"))
		}
		body, newErr := javascript.NewJavascriptBody(_unmarshalled.Library, _unmarshalled.Object)
		if body != nil {
			newErr = body.SetVarNames(_unmarshalled.Parameters)
		}
		return body, newErr

	default:
		return nil, errors.NewFunctionEncodingError("decode body", "unknown", fmt.Errorf("unknown language %v", language_type.Language))
	}
}
