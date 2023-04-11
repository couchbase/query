//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build !windows && !solaris

package util

import (
	"syscall"

	"github.com/couchbase/query/logging"
)

func CpuTimes() (int64, int64) {
	ru := syscall.Rusage{}
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		logging.Errorf(err.Error())
		return int64(0), int64(0)
	}

	newUtime := int64(ru.Utime.Nano())
	newStime := int64(ru.Stime.Nano())
	return newUtime, newStime
}
