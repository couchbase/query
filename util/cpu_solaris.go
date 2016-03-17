//  Copyright (c) 2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
