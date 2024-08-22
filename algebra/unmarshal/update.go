//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package unmarshal

import (
	"encoding/json"
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/expression/unmarshal"
)

/*
Unmarshals byte array.
*/
func UnmarshalSetTerms(body []byte) (algebra.SetTerms, error) {
	var _unmarshalled []struct {
		Meta  string          `json:"meta"`
		Path  string          `json:"path"`
		Value string          `json:"value"`
		For   json.RawMessage `json:"path_for"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return nil, err
	}

	terms := make(algebra.SetTerms, len(_unmarshalled))
	for i, term := range _unmarshalled {
		var metaExpr expression.Expression

		if term.Meta != "" {
			metaExpr, err = parser.Parse(term.Meta)
			if err != nil {
				return nil, err
			}
		}

		path_expr, err := parser.Parse(term.Path)
		if err != nil {
			return nil, err
		}

		path, is_path := path_expr.(expression.Path)
		if !is_path {
			return nil, fmt.Errorf("UnmarshalSetTerms: cannot resolve path expression from %s",
				term.Path)
		}

		value, err := parser.Parse(term.Value)
		if err != nil {
			return nil, err
		}

		var updateFor *algebra.UpdateFor
		if len(term.For) > 0 {
			updateFor, err = UnmarshalUpdateFor(term.For)
			if err != nil {
				return nil, err
			}
		}

		terms[i] = algebra.NewSetTerm(path, value, updateFor, metaExpr)
	}

	return terms, nil
}

/*
Unmarshals byte array.
*/
func UnmarshalUnsetTerm(body []byte) (*algebra.UnsetTerm, error) {
	var _unmarshalled struct {
		Path string          `json:"path"`
		For  json.RawMessage `json:"path_for"`
		Meta string          `json:"meta"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return nil, err
	}

	path_expr, err := parser.Parse(_unmarshalled.Path)
	if err != nil {
		return nil, err
	}

	path, is_path := path_expr.(expression.Path)
	if !is_path {
		return nil, fmt.Errorf("UnmarshalUnsetTerm: cannot resolve path expression from %s",
			_unmarshalled.Path)
	}

	var updateFor *algebra.UpdateFor
	if len(_unmarshalled.For) > 0 {
		updateFor, err = UnmarshalUpdateFor(_unmarshalled.For)
		if err != nil {
			return nil, err
		}
	}

	var metaExpr expression.Expression
	if _unmarshalled.Meta != "" {
		metaExpr, err = parser.Parse(_unmarshalled.Meta)
		if err != nil {
			return nil, err
		}
	}

	return algebra.NewUnsetTerm(path, updateFor, metaExpr), nil
}

/*
Unmarshals byte array.
*/
func UnmarshalUnsetTerms(body []byte) (algebra.UnsetTerms, error) {
	var _unmarshalled []struct {
		Path string          `json:"path"`
		For  json.RawMessage `json:"path_for"`
		Meta string          `json:"meta"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return nil, err
	}

	terms := make(algebra.UnsetTerms, len(_unmarshalled))
	for i, term := range _unmarshalled {
		path_expr, err := parser.Parse(term.Path)
		if err != nil {
			return nil, err
		}

		path, is_path := path_expr.(expression.Path)
		if !is_path {
			return nil, fmt.Errorf("UnmarshalUnsetTerms: cannot resolve path expression from %s",
				term.Path)
		}

		var updateFor *algebra.UpdateFor
		if len(term.For) > 0 {
			updateFor, err = UnmarshalUpdateFor(term.For)
			if err != nil {
				return nil, err
			}
		}

		var metaExpr expression.Expression
		if term.Meta != "" {
			metaExpr, err = parser.Parse(term.Meta)
			if err != nil {
				return nil, err
			}
		}

		terms[i] = algebra.NewUnsetTerm(path, updateFor, metaExpr)
	}

	return terms, nil
}

/*
Unmarshals byte array.
*/
func UnmarshalUpdateFor(body []byte) (*algebra.UpdateFor, error) {
	var _unmarshalled struct {
		Bindings json.RawMessage `json:"bindings"`
		When     string          `json:"when"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return nil, err
	}

	bindings, err := unmarshal.UnmarshalDimensions(_unmarshalled.Bindings)
	if err != nil {
		return nil, err
	}

	var when expression.Expression
	if _unmarshalled.When != "" {
		when, err = parser.Parse(_unmarshalled.When)
		if err != nil {
			return nil, err
		}
	}

	return algebra.NewUpdateFor(bindings, when), nil
}
