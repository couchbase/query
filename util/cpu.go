//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

import (
	"os"
	"runtime"
	"strconv"

	"github.com/couchbase/query/logging"
)

func NumCPU() int {
	return runtime.GOMAXPROCS(0)
}

func SetNumCPUs(max int, setting int, serverless bool) int {
	if mproc := os.Getenv("GOQUERY_GOMAXPROCS"); mproc != "" {
		if n, err := strconv.Atoi(mproc); err == nil {
			if n < max && n > 0 {
				max = n
				logging.Infof("CPU limit: %d (GOQUERY_GOMAXPROCS)", n)
			}
		} else {
			logging.Warnf("Invalid GOQUERY_GOMAXPROCS setting: %v", mproc)
		}
	} else if mproc := os.Getenv("GOMAXPROCS"); mproc != "" {
		if n, err := strconv.Atoi(mproc); err == nil {
			if n < max && n > 0 {
				max = n
				logging.Infof("CPU limit: %d (GOMAXPROCS)", max)
			}
		} else {
			logging.Warnf("Invalid GOMAXPROCS setting: %v", mproc)
		}
	} else if serverless {
		max = int(float64(max) * 0.8)
	}
	numCPUs := max
	if setting > 0 && setting < max {
		numCPUs = setting
	}
	if numCPUs < 1 {
		numCPUs = 1
	}
	runtime.GOMAXPROCS(numCPUs)
	return NumCPU()
}
