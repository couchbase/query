//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package functions

import (
	"github.com/couchbase/query/value"
)

type UdfContext struct {
	context Context
}

func NewUdfContext(context Context) *UdfContext {
	return &UdfContext{context}
}

func (this *UdfContext) NewValue(val interface{}) interface{} {
	return value.NewValue(val)
}

func (this *UdfContext) ExecuteStatement(statement string, namedArgs map[string]interface{}, positionalArgs []interface{}) (interface{}, uint64, error) {
	var named map[string]value.Value
	var positional []value.Value

	if len(namedArgs) > 0 {
		named = make(map[string]value.Value, len(namedArgs))
		for n, v := range namedArgs {
			named[n] = value.NewValue(v)
		}
	}
	if len(positionalArgs) > 0 {
		positional = make([]value.Value, len(positionalArgs))
		for i, v := range positionalArgs {
			positional[i] = value.NewValue(v)
		}
	}
	return this.context.EvaluateStatement(statement, named, positional, false, this.context.Readonly())
}
