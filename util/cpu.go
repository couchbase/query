//  Copyright (c) 2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

// +build !windows,!solaris

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
