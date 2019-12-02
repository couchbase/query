//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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

	return algebra.NewUnsetTerm(path, updateFor), nil
}

/*
Unmarshals byte array.
*/
func UnmarshalUnsetTerms(body []byte) (algebra.UnsetTerms, error) {
	var _unmarshalled []struct {
		Path string          `json:"path"`
		For  json.RawMessage `json:"path_for"`
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

		terms[i] = algebra.NewUnsetTerm(path, updateFor)
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
