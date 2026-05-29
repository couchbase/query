//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

// GetModelProvidersFunc is set by the natural package at init time to avoid an import cycle.
// expression → natural → execution → expression
var GetModelProvidersFunc func(nlCred, nlOrgId string, enabledOnly bool) (interface{}, errors.Error)

///////////////////////////////////////////////////
//
// ModelProviders
//
///////////////////////////////////////////////////

// ModelProviders returns the list of enabled AI model providers for the
// organization associated with the current natural language credentials.
// Usage: SELECT RAW model_providers()
type ModelProviders struct {
	FunctionBase
}

func NewModelProviders(operands ...Expression) Function {
	rv := &ModelProviders{}
	rv.Init("model_providers", operands...)
	rv.setVolatile()
	rv.expr = rv
	return rv
}

func (this *ModelProviders) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ModelProviders) Type() value.Type { return value.ARRAY }

func (this *ModelProviders) Evaluate(item value.Value, context Context) (value.Value, error) {

	nlcred, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	nlorgId, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	if nlcred.Type() == value.MISSING || nlorgId.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	if nlcred.Type() != value.STRING || nlorgId.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	enabledOnly := false
	if len(this.operands) > 2 {
		enabledOnlyVal, err := this.operands[2].Evaluate(item, context)
		if err != nil {
			return nil, err
		}
		if enabledOnlyVal.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		}
		if enabledOnlyVal.Type() != value.BOOLEAN {
			return value.NULL_VALUE, nil
		}
		enabledOnly = enabledOnlyVal.Truth()
	}

	result, err := GetModelProvidersFunc(nlcred.ToString(), nlorgId.ToString(), enabledOnly)
	if err != nil {
		return nil, err
	}
	return value.NewValue(result), nil
}

func (this *ModelProviders) MinArgs() int { return 2 }

func (this *ModelProviders) MaxArgs() int { return 3 }

func (this *ModelProviders) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewModelProviders(operands...)
	}
}
