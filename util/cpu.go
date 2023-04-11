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

var maxCPUs int
var warned bool

func SetMaxCPUs(max int) {
	maxCPUs = max
}

func SetNumCPUs(setting int, serverless bool) int {
	explicit := false
	numCPUs := runtime.NumCPU()
	if setting > 0 && setting < numCPUs {
		numCPUs = setting
		explicit = true
	}
	if maxCPUs > 0 && numCPUs > maxCPUs {
		numCPUs = maxCPUs
	}
	if !explicit {
		if mproc := os.Getenv("GOQUERY_GOMAXPROCS"); mproc != "" {
			if n, err := strconv.Atoi(mproc); err == nil {
				if n < numCPUs && n > 0 {
					numCPUs = n
				}
			} else if !warned {
				logging.Warnf("Invalid GOQUERY_GOMAXPROCS setting: %v", mproc)
				warned = true
			}
		} else if mproc := os.Getenv("GOMAXPROCS"); mproc != "" {
			if n, err := strconv.Atoi(mproc); err == nil {
				if n < numCPUs && n > 0 {
					numCPUs = n
				} else if serverless && n == numCPUs {
					numCPUs = int(float64(n) * 0.8)
				}
			} else if !warned {
				logging.Warnf("Invalid GOMAXPROCS setting: %v", mproc)
				warned = true
			}
		} else if serverless {
			numCPUs = int(float64(numCPUs) * 0.8)
		}
		if numCPUs < 1 {
			numCPUs = 1
		}
	}
	runtime.GOMAXPROCS(numCPUs)
	return NumCPU()
}
