//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package util

import (
	atomic "github.com/couchbase/go-couchbase/platform"
)

var MaxIndexApi atomic.AlignedInt64
var N1qlFeatureControl atomic.AlignedInt64
var UseCBO bool

func SetMaxIndexAPI(apiVersion int) {
	atomic.StoreInt64(&MaxIndexApi, int64(apiVersion))
}

func GetMaxIndexAPI() int {
	return int(atomic.LoadInt64(&MaxIndexApi))
}

const (
	N1QL_GROUPAGG_PUSHDOWN uint64 = 1 << iota
	N1QL_HASH_JOIN
	N1QL_ENCODED_PLAN
	N1QL_GOLANG_UDF
	N1QL_CBO
	N1QL_FLEXINDEX
	N1QL_CBO_NEW
	N1QL_DISABLE_PWD_BKT
	N1QL_READ_FROM_REPLICA
	N1QL_IMPLICIT_ARRAY_COVER
	N1QL_JOIN_ENUMERATION
	N1QL_ALL_BITS // Add anything above this. This needs to be last one
)

const DEF_N1QL_FEAT_CTRL = (N1QL_ENCODED_PLAN | N1QL_GOLANG_UDF | N1QL_CBO_NEW | N1QL_READ_FROM_REPLICA)
const CE_N1QL_FEAT_CTRL = (N1QL_GROUPAGG_PUSHDOWN | N1QL_HASH_JOIN | N1QL_ENCODED_PLAN | N1QL_GOLANG_UDF | N1QL_CBO | N1QL_FLEXINDEX | N1QL_CBO_NEW | N1QL_READ_FROM_REPLICA)

func SetN1qlFeatureControl(control uint64) {
	atomic.StoreInt64(&N1qlFeatureControl, int64(control))
}

func GetN1qlFeatureControl() uint64 {
	return uint64(atomic.LoadInt64(&N1qlFeatureControl))
}

func IsFeatureEnabled(control, feature uint64) bool {
	return (control & feature) == 0
}

const DEF_USE_CBO = true
const CE_USE_CBO = false

func GetUseCBO() bool {
	return UseCBO && IsFeatureEnabled(GetN1qlFeatureControl(), N1QL_CBO)
}

func SetUseCBO(useCBO bool) {
	// use-cbo can only be set if CBO is not turned off in N1qlFeatureControl
	if IsFeatureEnabled(GetN1qlFeatureControl(), N1QL_CBO) {
		UseCBO = useCBO
	}
}
