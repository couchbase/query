//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build !enterprise

package tenant

import (
	"time"

	"github.com/couchbase/query/errors"
)

type Unit uint64
type Service int
type Services [_SIZER]Unit
type ResourceManager func(string)

type Context interface{}
type Endpoint interface{}

const (
	QUERY_CU = Service(iota)
	JS_CU
	GSI_RU
	FTS_RU
	KV_RU
	KV_WU
	_SIZER
)

func Init(serverless bool) {
}

func Start(endpoint Endpoint, nodeid string, regulatorsettingsfile string) {
}

func RegisterResourceManager(m ResourceManager) {
}

func IsServerless() bool {
	return false
}

func AddUnit(dest *Unit, u Unit) {
}

func (this Unit) String() string {
	return "\"\""
}

func (this Unit) NonZero() bool {
	return false
}

func Throttle(isAdmin bool, user, bucket string, buckets []string, timeout time.Duration) (Context, errors.Error) {
	return new(Context), nil
}

func Bucket(ctx Context) string {
	return ""
}

func RecordCU(ctx Context, d time.Duration, m uint64) Unit {
	return 0
}

func RecordJsCU(ctx Context, d time.Duration, m uint64) Unit {
	return 0
}

func RefundUnits(ctx Context, units Services) error {
	return nil
}

func Units2Map(serv Services) map[string]interface{} {
	return nil
}

func EncodeNodeName(name string) string {
	return name
}

func DecodeNodeName(name string) string {
	return name
}

func Suspend(bucket string, delay time.Duration) {
}
