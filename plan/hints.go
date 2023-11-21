//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"encoding/json"
	"strings"

	"github.com/couchbase/query/algebra"
	hintParser "github.com/couchbase/query/algebra/parser"
	exprParser "github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/value"
)

func unmarshalOptimHints(body []byte) (*algebra.OptimHints, error) {
	var _unmarshalled struct {
		Followed       []json.RawMessage `json:"hints_followed"`
		NotFollowed    []json.RawMessage `json:"hints_not_followed"`
		WithError      []json.RawMessage `json:"hints_with_error"`
		Invalid        []json.RawMessage `json:"invalid_hints"`
		Unknown        []json.RawMessage `json:"hints_status_unknown"`
		FromSubqueries []json.RawMessage `json:"~from_clause_subqueries"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return nil, err
	}

	length := len(_unmarshalled.Followed) + len(_unmarshalled.NotFollowed) +
		len(_unmarshalled.WithError) + len(_unmarshalled.Invalid) + len(_unmarshalled.Unknown)
	newHints := make([]algebra.OptimHint, 0, length)
	optimHints := algebra.NewOptimHints(newHints, false)

	err = unmarshalHintArray(_unmarshalled.Followed, algebra.HINT_STATE_FOLLOWED, optimHints)
	if err != nil {
		return nil, err
	}

	err = unmarshalHintArray(_unmarshalled.NotFollowed, algebra.HINT_STATE_NOT_FOLLOWED, optimHints)
	if err != nil {
		return nil, err
	}

	err = unmarshalHintArray(_unmarshalled.WithError, algebra.HINT_STATE_ERROR, optimHints)
	if err != nil {
		return nil, err
	}

	err = unmarshalHintArray(_unmarshalled.Invalid, algebra.HINT_STATE_INVALID, optimHints)
	if err != nil {
		return nil, err
	}

	err = unmarshalHintArray(_unmarshalled.Unknown, algebra.HINT_STATE_UNKNOWN, optimHints)
	if err != nil {
		return nil, err
	}

	if len(_unmarshalled.FromSubqueries) > 0 {
		subqTermHints := make([]*algebra.SubqOptimHints, len(_unmarshalled.FromSubqueries))
		for i := range _unmarshalled.FromSubqueries {
			subqTermHints[i], err = unmarshalSubqTermHints(_unmarshalled.FromSubqueries[i])
			if err != nil {
				return nil, err
			}
		}
		optimHints.AddSubqTermHints(subqTermHints)
	}

	return optimHints, nil
}

func unmarshalHintArray(hints []json.RawMessage, state algebra.HintState, optimHints *algebra.OptimHints) error {
	for _, hint := range hints {
		valStr := value.NewValue(hint).Actual().(string)
		valExpr, err := exprParser.Parse(valStr)
		if err != nil {
			return err
		} else if valExpr == nil {
			continue
		}
		val := valExpr.Value()
		if val == nil {
			continue
		}

		var newHints []algebra.OptimHint
		var hintStr, errStr string
		if val.Type() == value.OBJECT {
			hintVal, ok := val.Field("hint")
			if !ok || hintVal == nil {
				continue
			}
			hintStr = hintVal.Actual().(string)

			hintsErr, ok := val.Field("error")
			if ok {
				errStr = hintsErr.Actual().(string)
			}
			optimHints.SetJSONStyle()
		} else if val.Type() == value.STRING {
			str := val.Actual().(string)
			parts := strings.Split(str, ":")
			if len(parts) > 1 {
				hintStr = parts[0]
				errStr = parts[1]
			} else if len(parts) > 0 {
				hintStr = parts[0]
			}
		}

		if len(hintStr) == 0 {
			continue
		}

		newOptimHints := hintParser.Parse("+ " + hintStr)
		if newOptimHints == nil {
			continue
		}
		newHints = newOptimHints.Hints()

		if len(newHints) != 1 {
			continue
		}
		newHint := newHints[0]

		switch state {
		case algebra.HINT_STATE_FOLLOWED:
			newHint.SetFollowed()
		case algebra.HINT_STATE_NOT_FOLLOWED:
			newHint.SetNotFollowed()
		case algebra.HINT_STATE_ERROR:
			newHint.SetError(errStr)
		case algebra.HINT_STATE_INVALID:
			// no-op, should have been parsed to INVALID already
		case algebra.HINT_STATE_UNKNOWN:
			// no-op, default state
		}
		optimHints.AddHints(newHints)
	}
	return nil
}

func unmarshalSubqTermHints(body []byte) (*algebra.SubqOptimHints, error) {
	var _unmarshalled struct {
		Alias string          `json:"alias"`
		Hints json.RawMessage `json:"optimizer_hints"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return nil, err
	}

	alias := _unmarshalled.Alias
	hints, err := unmarshalOptimHints(_unmarshalled.Hints)
	if err != nil {
		return nil, err
	}
	return algebra.NewSubqOptimHints(alias, hints), nil
}
