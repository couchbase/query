//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package unmarshal

import (
	"encoding/json"

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

func UnmarshalBinding(body []byte) (*expression.Binding, error) {
	var _unmarshalled struct {
		NameVar string `json:"name_var"`
		Var     string `json:"var"`
		Expr    string `json:"expr"`
		Desc    bool   `json:"desc"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return nil, err
	}

	expr, err := parser.Parse(_unmarshalled.Expr)
	if err != nil {
		return nil, err
	}

	return expression.NewBinding(_unmarshalled.NameVar, _unmarshalled.Var, expr, _unmarshalled.Desc), nil
}

func UnmarshalBindings(body []byte) (expression.Bindings, error) {
	var _unmarshalled []struct {
		NameVar string `json:"name_var"`
		Var     string `json:"var"`
		Expr    string `json:"expr"`
		Desc    bool   `json:"desc"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return nil, err
	}

	bindings := make(expression.Bindings, len(_unmarshalled))
	for i, binding := range _unmarshalled {
		expr, err := parser.Parse(binding.Expr)
		if err != nil {
			return nil, err
		}

		bindings[i] = expression.NewBinding(binding.NameVar, binding.Var, expr, binding.Desc)
	}

	return bindings, nil
}

func UnmarshalDimensions(body []byte) ([]expression.Bindings, error) {
	var _unmarshalled [][]struct {
		NameVar string `json:"name_var"`
		Var     string `json:"var"`
		Expr    string `json:"expr"`
		Desc    bool   `json:"desc"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return nil, err
	}

	dimensions := make([]expression.Bindings, len(_unmarshalled))
	for i, u := range _unmarshalled {
		dimension := make(expression.Bindings, len(u))
		for j, binding := range u {
			expr, err := parser.Parse(binding.Expr)
			if err != nil {
				return nil, err
			}

			dimension[j] = expression.NewBinding(binding.NameVar, binding.Var, expr, binding.Desc)
		}

		dimensions[i] = dimension
	}

	return dimensions, nil
}
