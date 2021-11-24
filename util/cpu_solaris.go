//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

/*
#include <string.h>
#include <errno.h>
#include <sys/resource.h>

const char *get_error() { return strerror(errno); }
*/
import "C"

import (
	"github.com/couchbase/query/logging"
)

func CpuTimes() (int64, int64) {
	ru := C.struct_rusage{}
	if ret := C.getrusage(C.RUSAGE_SELF, &ru); ret != 0 {
		logging.Errorf(C.GoString(C.get_error()))
		return int64(0), int64(0)
	}

	newUtime := int64(ru.ru_utime.tv_usec) * 1000
	newStime := int64(ru.ru_stime.tv_usec) * 1000
	return newUtime, newStime
}
