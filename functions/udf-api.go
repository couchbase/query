//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package functions

import (
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

type UdfContext struct {
	context Context
}

type UdfHandle struct {
	handle interface {
		NextDocument() (value.Value, error)
		Cancel()
	}
}

func NewUdfContext(context Context) *UdfContext {
	return &UdfContext{context}
}

func (this *UdfContext) NewValue(val interface{}) interface{} {
	return value.NewValue(val)
}

func (this *UdfContext) CopyValue(val interface{}) interface{} {
	v, ok := val.(value.Value)
	if ok {
		return v.CopyForUpdate()
	}
	return nil
}

func (this *UdfContext) StoreValue(key string, val interface{}) {
	this.context.StoreValue(key, val)
}

func (this *UdfContext) RetrieveValue(key string) interface{} {
	return this.context.RetrieveValue(key)
}

func (this *UdfContext) ReleaseValue(key string) {
	this.context.ReleaseValue(key)
}

func (this *UdfContext) CompareValues(val1, val2 interface{}) (int, bool) {
	v1, ok1 := val1.(value.Value)
	v2, ok2 := val1.(value.Value)
	if !ok1 || !ok2 {
		return 0, true
	}
	res := v1.Compare(v2)
	i, ok := res.Actual().(int)
	return i, (!ok || res.Type() == value.NULL || res.Type() == value.MISSING)
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

func (this *UdfContext) OpenStatement(statement string, namedArgs map[string]interface{}, positionalArgs []interface{}) (interface{}, error) {
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
	handle, err := this.context.OpenStatement(statement, named, positional, false, this.context.Readonly())
	if err != nil {
		return nil, err
	}
	return &UdfHandle{handle}, nil
}

func (this *UdfContext) Log(fmt string, args ...interface{}) {
	logging.Infof(fmt, args...)
}

func (this *UdfHandle) NextDocument() (interface{}, error) {
	return this.handle.NextDocument()
}

func (this *UdfHandle) Cancel() {
	this.handle.Cancel()
}
