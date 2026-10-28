//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build !race

package value

/*
Tests asserting exact zero allocations (testing.AllocsPerRun == 0) skip when
raceEnabled. They guard against allocation regressions, but the counts depend
on the compiler's escape analysis and inlining, so a Go toolchain upgrade or
other build flags (e.g. -gcflags=-N) can change them without a code change.
*/
const raceEnabled = false
