//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package util

import (
	atomic "github.com/couchbase/go-couchbase/platform"
)

var MaxIndexApi atomic.AlignedInt64
var N1qlFeatureControl atomic.AlignedInt64

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
	N1QL_ALL_BITS // Add anything above this. This needs to be last one
)

const DEF_N1QL_FEAT_CTRL = (N1QL_ENCODED_PLAN | N1QL_GOLANG_UDF)
const CE_N1QL_FEAT_CTRL = (N1QL_GROUPAGG_PUSHDOWN | N1QL_HASH_JOIN | N1QL_ENCODED_PLAN | N1QL_GOLANG_UDF)

func SetN1qlFeatureControl(control uint64) {
	atomic.StoreInt64(&N1qlFeatureControl, int64(control))
}

func GetN1qlFeatureControl() uint64 {
	return uint64(atomic.LoadInt64(&N1qlFeatureControl))
}

func IsFeatureEnabled(control, feature uint64) bool {
	return (control & feature) == 0
}
