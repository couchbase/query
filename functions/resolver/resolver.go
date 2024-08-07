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
	entry, er := MakeBodyEntry(name, bytes)
	if er != nil {
		return nil, er
	}
	var err error
	language, ok := entry["#language"].(string)
	if !ok {
		return nil, errors.NewFunctionEncodingError("decode body", "unknown",
			fmt.Errorf("language is not string"))
	}
	switch language {
	case "inline":
		var expr expression.Expression
		expression := entry["expression"].(string)
		if expression != "" {
			expr, err = parser.Parse(expression)
			if err != nil {
				return nil, errors.NewFunctionEncodingError("decode body", name, err)
			}
		} else {
			return nil, errors.NewFunctionEncodingError("decode body", name, go_errors.New("expression is missing"))
		}
		text := entry["text"].(string)
		if len(text) == 0 {
			text = expr.String()
		}
		body, newErr := inline.NewInlineBody(expr, text)
		if body != nil {
			newErr = body.SetVarNames(entry["parameters"].([]string))
		}
		return body, newErr

	case "golang":
		object := entry["object"].(string)
		library := entry["library"].(string)
		if object == "" || library == "" {
			return nil, errors.NewFunctionEncodingError("decode body", name, go_errors.New("object is missing"))
		}
		body, newErr := golang.NewGolangBody(library, object)
		if body != nil {
			newErr = body.SetVarNames(entry["parameters"].([]string))
		}
		return body, newErr

	case "javascript":

		// Check if the function body is a valid combination of library, object and text
		// This is to ensure that the function body is valid to be considered as either Internal JS or External JS function
		var text, object, library, prefix, libName string
		if t, ok := entry["text"].(string); ok {
			text = t
		}
		if o, ok := entry["object"].(string); ok {
			object = o
		}
		if l, ok := entry["library"].(string); ok {
			library = l
		}
		if p, ok := entry["prefix"].(string); ok {
			prefix = p
		}
		if n, ok := entry["libName"].(string); ok {
			libName = n
		}
		if !(text == "" && object != "" && library != "") &&
			!(text != "" && object == "" && library == "") {

			return nil, errors.NewFunctionEncodingError("decode body", name, go_errors.New("invalid function definition"))
		}
		body, newErr := javascript.NewJavascriptBodyWithDetails(library, object, prefix, libName, text)
		if body != nil {
			newErr = body.SetVarNames(entry["parameters"].([]string))
		}
		return body, newErr

	default:
		return nil, errors.NewFunctionEncodingError("decode body", "unknown",
			fmt.Errorf("unknown language %v", language))
	}
}

// the "entry" returned here is the same format as what's returned from the "Body()" function
func MakeBodyEntry(name string, bytes []byte) (map[string]interface{}, errors.Error) {
	var language_type struct {
		Language string `json:"#language"`
	}

	err := json.Unmarshal(bytes, &language_type)
	if err != nil {
		return nil, errors.NewFunctionEncodingError("decode body", name, err)
	}
	switch language_type.Language {
	case "inline":

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
		entry := map[string]interface{}{
			"#language":  "inline",
			"expression": _unmarshalled.Expression,
			"parameters": _unmarshalled.Parameters,
			"text":       _unmarshalled.Text,
		}
		return entry, nil

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
		entry := map[string]interface{}{
			"#language":  "golang",
			"library":    _unmarshalled.Library,
			"object":     _unmarshalled.Object,
			"parameters": _unmarshalled.Parameters,
		}
		return entry, nil

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
		entry := map[string]interface{}{
			"#language":  "javascript",
			"parameters": _unmarshalled.Parameters,
		}
		if _unmarshalled.Text != "" {
			entry["text"] = _unmarshalled.Text
		}
		if _unmarshalled.Library != "" {
			entry["library"] = _unmarshalled.Library
		}
		if _unmarshalled.Object != "" {
			entry["object"] = _unmarshalled.Object
		}
		if _unmarshalled.Prefix != "" {
			entry["prefix"] = _unmarshalled.Prefix
		}
		if _unmarshalled.Name != "" {
			entry["libName"] = _unmarshalled.Name
		}
		return entry, nil

	default:
		return nil, errors.NewFunctionEncodingError("decode body", "unknown",
			fmt.Errorf("unknown language %v", language_type.Language))
	}
}
