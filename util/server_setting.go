//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

import (
	"fmt"
	"sort"

	atomic "github.com/couchbase/go-couchbase/platform"
)

var MaxIndexApi atomic.AlignedInt64
var n1qlFeatureControl atomic.AlignedInt64
var UseCBO bool

func SetMaxIndexAPI(apiVersion int) {
	atomic.StoreInt64(&MaxIndexApi, int64(apiVersion))
}

func GetMaxIndexAPI() int {
	return int(atomic.LoadInt64(&MaxIndexApi))
}

const (
	N1QL_GROUPAGG_PUSHDOWN     uint64 = 1 << iota // 0x0000000001
	N1QL_HASH_JOIN                                // 0x0000000002
	N1QL_ENCODED_PLAN                             // 0x0000000004
	N1QL_GOLANG_UDF                               // 0x0000000008
	N1QL_CBO                                      // 0x0000000010
	N1QL_FLEXINDEX                                // 0x0000000020
	N1QL_CBO_NEW                                  // 0x0000000040
	N1QL_PASSWORDLESS_BKT                         // 0x0000000080
	N1QL_READ_FROM_REPLICA_OFF                    // 0x0000000100
	N1QL_IMPLICIT_ARRAY_COVER                     // 0x0000000200
	N1QL_JOIN_ENUMERATION                         // 0x0000000400
	N1QL_INDEX_MISSING                            // 0x0000000800
	N1QL_NL_PRIMARYSCAN                           // 0x0000001000
	N1QL_EARLY_ORDER                              // 0x0000002000
	N1QL_SEQ_SCAN                                 // 0x0000004000
	N1QL_SPILL_TO_DISK                            // 0x0000008000
	N1QL_ALL_BITS                                 // Add anything above this. This needs to be last one
)

var N1Ql_Features = map[uint64]string{
	N1QL_GROUPAGG_PUSHDOWN: fmt.Sprintf("Index Grouping and Aggregate Pushdown (%#x)", N1QL_GROUPAGG_PUSHDOWN),
	N1QL_HASH_JOIN:         fmt.Sprintf("Hash Join (%#x)", N1QL_HASH_JOIN),
	N1QL_ENCODED_PLAN:      fmt.Sprintf("Encoded Plans (%#x)", N1QL_ENCODED_PLAN),
	N1QL_GOLANG_UDF:        fmt.Sprintf("Golang UDFs (%#x)", N1QL_GOLANG_UDF),
	N1QL_CBO:               fmt.Sprintf("CBO (%#x)", N1QL_CBO),
	N1QL_FLEXINDEX:         fmt.Sprintf("Flex Index (%#x)", N1QL_FLEXINDEX),
	//N1QL_CBO_NEW:               fmt.Sprintf("CBO New (%#x)", N1QL_CBO_NEW),	// To-Do : un-comment when CBO New is supported
	N1QL_PASSWORDLESS_BKT:      fmt.Sprintf("Allow Password-less Buckets (%#x)", N1QL_PASSWORDLESS_BKT),
	N1QL_READ_FROM_REPLICA_OFF: fmt.Sprintf("Read From Active v-Bucket Only (%#x)", N1QL_READ_FROM_REPLICA_OFF),
	N1QL_IMPLICIT_ARRAY_COVER:  fmt.Sprintf("Implicit Covering Array Index (%#x)", N1QL_IMPLICIT_ARRAY_COVER),
	N1QL_JOIN_ENUMERATION:      fmt.Sprintf("Join Enumeration (%#x)", N1QL_JOIN_ENUMERATION),
	N1QL_INDEX_MISSING:         fmt.Sprintf("Leading Index Key INCLUDE MISSING entries(%#x)", N1QL_INDEX_MISSING),
	N1QL_NL_PRIMARYSCAN:        fmt.Sprintf("Prevent Primary Scan on Inner Side of Nested Loop Join (%#x)", N1QL_NL_PRIMARYSCAN),
	N1QL_EARLY_ORDER:           fmt.Sprintf("Early Order (%#x)", N1QL_EARLY_ORDER),
	N1QL_SEQ_SCAN:              fmt.Sprintf("Sequential Scans (%#x)", N1QL_SEQ_SCAN),
	N1QL_SPILL_TO_DISK:         fmt.Sprintf("Spill To Disk (%#x)", N1QL_SPILL_TO_DISK),
}

const DEF_N1QL_FEAT_CTRL = (N1QL_ENCODED_PLAN | N1QL_GOLANG_UDF | N1QL_CBO_NEW)
const CE_N1QL_FEAT_CTRL = (N1QL_GROUPAGG_PUSHDOWN | N1QL_HASH_JOIN | N1QL_ENCODED_PLAN | N1QL_GOLANG_UDF |
	N1QL_CBO | N1QL_FLEXINDEX | N1QL_CBO_NEW)

func SetN1qlFeatureControl(control uint64) uint64 {
	return uint64(atomic.SwapInt64(&n1qlFeatureControl, int64(control)))
}

func GetN1qlFeatureControl() uint64 {
	return uint64(atomic.LoadInt64(&n1qlFeatureControl))
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

// Get a list of all the features that have been disabled in the 'control' bitset
func DisabledFeatures(control uint64) []string {
	disabled := make([]string, 0)

	for flag, feat := range N1Ql_Features {
		if (control & flag) != 0 { // feature is disabled
			disabled = append(disabled, feat)
		}
	}

	return disabled
}

// Get the features that have changed ( either been enabled or disabled )
// from the 'prev' bitset to the 'new' bitset
func ChangedFeatures(prev uint64, new uint64) ([]string, []string) {
	difference := prev ^ new // the difference between the prev and new n1ql-feat-ctrl bitsets

	enabled := make([]string, 0)
	disabled := make([]string, 0)

	if difference != 0 { // there is a change in the feature bitset
		flags := make([]uint64, 0, len(N1Ql_Features))

		for f := range N1Ql_Features {
			flags = append(flags, f)
		}

		sort.Slice(flags, func(i, j int) bool {
			return flags[i] < flags[j]
		})

		for _, flag := range flags {
			feat := N1Ql_Features[flag]
			old := prev & flag

			if old != (new & flag) { // the feature bit has changed
				if old != 0 { // the feature bit was 1 i.e the feature used to be DISABLED .. hence in the new bitset it is now ENABLED
					enabled = append(enabled, feat)
				} else {
					disabled = append(disabled, feat)
				}
			}
		}
	}

	return enabled, disabled

}
