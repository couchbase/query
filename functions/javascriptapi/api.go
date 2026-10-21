//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package javascriptapi

import "fmt"

type Value interface {
	String() string
	MarshalJSON() ([]byte, error)
	Actual() any
	ToString() string
	Truth() bool
	Recycle()
	Track()
}

type Error interface {
	error
	Object() map[string]any
}

type Context interface {
	NewValue(val any) any
	CopyValue(val any) any
	StoreValue(key string, val any)
	RetrieveValue(key string) any
	ReleaseValue(key string)
	CompareValues(val1, val2 any) (int, bool)
	ExecuteStatement(statement string, namedArgs map[string]any, positionalArgs []any) (any, uint64, error)
	OpenStatement(statement string, namedArgs map[string]any, positionalArgs []any) (any, error)
	Log(fmt string, args ...any)
	NestingLevel() int
	StorageContext() string
}

type Handle interface {
	Type() string
	Mutations() uint64
	Results() (any, uint64, error)
	Complete() (uint64, error)
	NextDocument() (any, error)
	Cancel()
}

func Args(args any, context any) (Value, Context, error) {
	a, ok := args.(Value)
	if !ok {
		return nil, nil, fmt.Errorf("invalid function arguments type %T", args)
	}
	c, ok := context.(Context)
	if !ok {
		return nil, nil, fmt.Errorf("invalid function context type %T", context)
	}
	return a, c, nil
}

// BasicContext interface exposes a limited set of methods to js-evaluator
type BasicContext interface {
	StorageContext() string
	Log(fmt string, args ...any)
}
