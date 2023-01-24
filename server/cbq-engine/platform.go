// Copyright 2013-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
// file, in accordance with the Business Source License, use of this software
// will be governed by the Apache License, Version 2.0, included in the file
// licenses/APL2.txt.

//go:build !windows
// +build !windows

package main

import (
	"syscall"

	"github.com/couchbase/query/logging"
)

func HideConsole(_ bool) {
}

func setOpenFilesLimit() {
	var lim syscall.Rlimit
	var err error
	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &lim)
	if err == nil {
		logging.Infof("Current nofiles rlimit: %v (max: %v)", lim.Cur, lim.Max)
		if lim.Max != lim.Cur {
			lim.Cur = lim.Max
			err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &lim)
			if err != nil && lim.Cur > 1048576 {
				lim.Cur = 1048576
				err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &lim)
			}
			if err != nil && lim.Cur > 12288 {
				lim.Cur = 12288
				err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &lim)
			}
			if err != nil {
				logging.Warnf("Unable to increase nofiles limit to %v: %v", lim.Cur, err)
			} else {
				logging.Infof("nofiles limit set to: %v", lim.Cur)
			}
		}
	} else {
		logging.Warnf("Unable to query current nofiles limit: %v", err)
	}
}
