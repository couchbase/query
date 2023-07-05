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
	Actual() interface{}
	ToString() string
	Truth() bool
	Recycle()
	Track()
}

type Error interface {
	error
	Object() map[string]interface{}
}

type Context interface {
	NewValue(val interface{}) interface{}
	CopyValue(val interface{}) interface{}
	StoreValue(key string, val interface{})
	RetrieveValue(key string) interface{}
	ReleaseValue(key string)
	CompareValues(val1, val2 interface{}) (int, bool)
	ExecuteStatement(statement string, namedArgs map[string]interface{}, positionalArgs []interface{}) (interface{}, uint64, error)
	OpenStatement(statement string, namedArgs map[string]interface{}, positionalArgs []interface{}) (interface{}, error)
	Log(fmt string, args ...interface{})
	NestingLevel() int
	StorageContext() string
}

type Handle interface {
	Type() string
	Mutations() uint64
	Results() (interface{}, uint64, error)
	Complete() (uint64, error)
	NextDocument() (interface{}, error)
	Cancel()
}

func Args(args interface{}, context interface{}) (Value, Context, error) {
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
	Log(fmt string, args ...interface{})
}
