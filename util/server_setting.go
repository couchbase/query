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
	"strings"

	atomic "github.com/couchbase/go-couchbase/platform"
)

const FULL_SPAN_FANOUT = 8192
const INCLUDE_SPAN_FANOUT = 128

var MaxIndexApi atomic.AlignedInt64
var n1qlFeatureControl atomic.AlignedInt64
var UseCBO bool

func SetMaxIndexAPI(apiVersion int) {
	atomic.StoreInt64(&MaxIndexApi, int64(apiVersion))
}

func GetMaxIndexAPI() int {
	return int(atomic.LoadInt64(&MaxIndexApi))
}

// Note:
// Feature bits always indicate DISABLING the feature so constants should be named for the default operation which may mean giving
// a negative name here.  (If posible avoid negative naming and label the alternative.)
const (
	N1QL_GROUPAGG_PUSHDOWN    uint64 = 1 << iota // 0x0000000001
	N1QL_HASH_JOIN                               // 0x0000000002
	N1QL_ENCODED_PLAN                            // 0x0000000004
	N1QL_GOLANG_UDF                              // 0x0000000008
	N1QL_CBO                                     // 0x0000000010
	N1QL_FLEXINDEX                               // 0x0000000020
	N1QL_CBO_NEW                                 // 0x0000000040
	_RETIRED_DONT_USE_1                          // 0x0000000080 N1QL_PASSWORDLESS_BKT (MB-39484)
	_RETIRED_DONT_USE_2                          // 0x0000000100 N1QL_READ_FROM_REPLICA_OFF
	N1QL_IMPLICIT_ARRAY_COVER                    // 0x0000000200
	N1QL_JOIN_ENUMERATION                        // 0x0000000400
	N1QL_INDEX_MISSING                           // 0x0000000800
	N1QL_NL_PRIMARYSCAN                          // 0x0000001000
	N1QL_EARLY_ORDER                             // 0x0000002000
	N1QL_SEQ_SCAN                                // 0x0000004000
	N1QL_SPILL_TO_DISK                           // 0x0000008000
	N1QL_PART_GRACEFUL                           // 0x0000010000
	N1QL_FULL_GET                                // 0x0000020000 controls use of sub-doc API
	N1QL_RANDOM_SCAN                             // 0x0000040000
	N1QL_NEW_MERGE                               // 0x0000080000
	N1QL_NO_DATE_WARNINGS                        // 0x0000100000
	N1QL_USE_SYS_FREE_MEM                        // 0x0000200000
	N1QL_ADMISSION_CONTROL                       // 0x0000400000
	N1QL_IGNORE_IDXR_META                        // 0x0000800000
	N1QL_NATURAL_LANG_REQ                        // 0x0001000000
	N1QL_FULL_SPAN_FANOUT                        // 0x0002000000
)

// Care should be taken that the descriptions accept "disabled" being appended when the bit is set (and "enabled" when not).
// Don't include retired values - they will just be ignored when describing the feature controls setting.
var _N1QL_Features = map[uint64]string{
	N1QL_GROUPAGG_PUSHDOWN:    "Index grouping and aggregate pushdown",
	N1QL_HASH_JOIN:            "Hash join",
	N1QL_ENCODED_PLAN:         "Encoded plans",
	N1QL_GOLANG_UDF:           "Golang UDFs",
	N1QL_CBO:                  "CBO",
	N1QL_FLEXINDEX:            "Flex index",
	N1QL_CBO_NEW:              "(Reserved for future use)",
	N1QL_IMPLICIT_ARRAY_COVER: "Implicit covering array index",
	N1QL_JOIN_ENUMERATION:     "Join enumeration",
	N1QL_INDEX_MISSING:        "Include MISSING entries in leading index key",
	N1QL_NL_PRIMARYSCAN:       "Prevent primary scan on inner side of nested loop join",
	N1QL_EARLY_ORDER:          "Early order",
	N1QL_SEQ_SCAN:             "Sequential scans",
	N1QL_SPILL_TO_DISK:        "Spill to disk",
	N1QL_PART_GRACEFUL:        "Partial graceful shutdown",
	N1QL_FULL_GET:             "Only fetch full documents",
	N1QL_RANDOM_SCAN:          "Random scans",
	N1QL_NEW_MERGE:            "New MERGE",
	N1QL_NO_DATE_WARNINGS:     "Date warning suppression",
	N1QL_USE_SYS_FREE_MEM:     "Allow system free memory use",
	N1QL_ADMISSION_CONTROL:    "Admission control",
	N1QL_IGNORE_IDXR_META:     "Ignore indexer metadata changes for prepared statements",
	N1QL_NATURAL_LANG_REQ:     "Natural Language Request",
	N1QL_FULL_SPAN_FANOUT:     "Spans Fanout to 8192",
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

	for flag, feat := range _N1QL_Features {
		if (control & flag) != 0 { // feature is disabled
			disabled = append(disabled, fmt.Sprintf("%s (%#x)", feat, flag))
		}
	}
	sort.Strings(disabled)

	return disabled
}

// Get the features that have changed ( either been enabled or disabled ) from the 'prev' bitset to the 'new' bitset
// Used for logging changes
func DescribeChangedFeatures(prev uint64, new uint64) string {
	if prev == new {
		return " No Changes"
	}

	// there is a difference between the feature bitsets
	changes := strings.Builder{}
	changes.WriteString(fmt.Sprintf(" (%#x)", new))
	flags := make([]uint64, 0, len(_N1QL_Features))

	for f := range _N1QL_Features {
		flags = append(flags, f)
	}

	sort.Slice(flags, func(i, j int) bool {
		return flags[i] < flags[j]
	})

	for _, flag := range flags {
		feat := _N1QL_Features[flag]
		old := prev & flag

		if old != (new & flag) { // the feature bit has changed
			changes.WriteString(fmt.Sprintf(", %s (%#x) ", feat, flag))
			if old != 0 { // the feature bit was 1 i.e the feature used to be DISABLED hence in the new bitset it is now ENABLED
				changes.WriteString("enabled")
			} else {
				changes.WriteString("disabled")
			}
		}
	}

	return changes.String()
}

func FullSpanFanout(isInclude bool) int {
	if IsFeatureEnabled(GetN1qlFeatureControl(), N1QL_FULL_SPAN_FANOUT) {
		if isInclude {
			return INCLUDE_SPAN_FANOUT
		}
		return FULL_SPAN_FANOUT
	} else if isInclude {
		return 4 * INCLUDE_SPAN_FANOUT
	}
	return 4 * FULL_SPAN_FANOUT
}
