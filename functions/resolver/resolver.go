//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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

func MakePath(bytes []byte) ([]string, errors.Error) {
	var name_type struct {
		Type string `json:"type"`
	}

	err := json.Unmarshal(bytes, &name_type)
	if err != nil {
		return nil, errors.NewFunctionEncodingError("decode name", "unknown", err)
	}
	switch name_type.Type {
	case "global":
		var _unmarshalled struct {
			_         string `json:"type"`
			Namespace string `json:"namespace"`
			Name      string `json:"name"`
		}
		err := json.Unmarshal(bytes, &_unmarshalled)
		if err != nil {
			return nil, errors.NewFunctionEncodingError("decode name", "unknown", err)
		}
		if _unmarshalled.Namespace == "" || _unmarshalled.Name == "" {
			return nil, errors.NewFunctionEncodingError("decode name", "unknown", go_errors.New("incomplete function name"))
		}
		return []string{_unmarshalled.Namespace, _unmarshalled.Name}, nil
	case "scope":
		var _unmarshalled struct {
			_         string `json:"type"`
			Namespace string `json:"namespace"`
			Bucket    string `json:"bucket"`
			Scope     string `json:"scope"`
			Name      string `json:"name"`
		}
		err := json.Unmarshal(bytes, &_unmarshalled)
		if err != nil {
			return nil, errors.NewFunctionEncodingError("decode name", "unknown", err)
		}
		if _unmarshalled.Namespace == "" || _unmarshalled.Bucket == "" || _unmarshalled.Scope == "" || _unmarshalled.Name == "" {
			return nil, errors.NewFunctionEncodingError("decode name", "unknown", go_errors.New("incomplete function name"))
		}
		return []string{_unmarshalled.Namespace, _unmarshalled.Bucket, _unmarshalled.Scope, _unmarshalled.Name}, nil
	default:
		return nil, errors.NewFunctionEncodingError("decode name", "unknown", fmt.Errorf("unknown type %v", name_type.Type))
	}
}

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
			Text       string   `json:"text"`
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
		if len(_unmarshalled.Text) == 0 {
			_unmarshalled.Text = expr.String()
		}
		body, newErr := inline.NewInlineBody(expr, _unmarshalled.Text)
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
			Prefix     string   `json:"prefix"`
			Name       string   `json:"name"`
			Text       string   `json:"text"`
		}
		err := json.Unmarshal(bytes, &_unmarshalled)
		if err != nil {
			return nil, errors.NewFunctionEncodingError("decode body", name, err)
		}

		// Check if the function body is a valid combination of library, object and text
		// This is to ensure that the function body is valid to be considered as either Internal JS or External JS function
		if !(_unmarshalled.Text == "" && _unmarshalled.Object != "" && _unmarshalled.Library != "") &&
			!(_unmarshalled.Text != "" && _unmarshalled.Object == "" && _unmarshalled.Library == "") {

			return nil, errors.NewFunctionEncodingError("decode body", name, go_errors.New("invalid function definition"))
		}
		body, newErr := javascript.NewJavascriptBodyWithDetails(_unmarshalled.Library, _unmarshalled.Object,
			_unmarshalled.Prefix, _unmarshalled.Name, _unmarshalled.Text)
		if body != nil {
			newErr = body.SetVarNames(_unmarshalled.Parameters)
		}
		return body, newErr

	default:
		return nil, errors.NewFunctionEncodingError("decode body", "unknown",
			fmt.Errorf("unknown language %v", language_type.Language))
	}
}
