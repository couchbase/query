//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

import (
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/couchbase/query/logging"
)

type tempInfoT struct {
	loc   string
	quota int64
	inuse int64
	hwm   int64
}

var tempInfo tempInfoT
var tempMutex sync.RWMutex

func SetTemp(loc string, quota int64) error {
	tempMutex.Lock()
	if quota < 0 {
		quota = 0
	}
	if !path.IsAbs(loc) {
		logging.Errorf("Attempt to set relative temporary path: %v", loc)
		tempMutex.Unlock()
		return fmt.Errorf("Attempt to set relative temporary path")
	} else if _, err := os.Stat(loc); err != nil {
		logging.Errorf("Attempt to set invalid or inaccessible temporary path: %v (%v)", loc, err)
		tempMutex.Unlock()
		return fmt.Errorf("Attempt to set invalid or inaccessible temporary path")
	}
	if tempInfo.loc != loc {
		tempInfo.inuse = 0
	}
	if tempInfo.loc != loc || tempInfo.quota != quota {
		tempInfo.loc = loc
		tempInfo.quota = quota
		logging.Infof("Temporary file path set to: %v, quota: %v", loc, logging.HumanReadableSize(quota, true))
	}
	tempMutex.Unlock()
	return nil
}

func SetTempDir(loc string) error {
	return SetTemp(loc, TempQuota())
}

func SetTempQuota(q int64) error {
	return SetTemp(TempLocation(), q)
}

func TempLocation() string {
	tempMutex.RLock()
	rv := tempInfo.loc
	tempMutex.RUnlock()
	return rv
}

func TempQuota() int64 {
	tempMutex.RLock()
	rv := tempInfo.quota
	tempMutex.RUnlock()
	return rv
}

func CreateTemp(pattern string, autoRemove bool) (*os.File, error) {
	f, err := os.CreateTemp(TempLocation(), pattern)
	if autoRemove && err == nil {
		os.Remove(f.Name())
	}
	return f, err
}

func UseTemp(pathname string, sz int64) bool {
	rv := true
	loc := path.Dir(pathname)
	tempMutex.Lock()
	if tempInfo.quota > 0 && (pathname == "" || loc == tempInfo.loc) {
		tempInfo.inuse += sz
		if tempInfo.inuse > tempInfo.quota {
			tempInfo.inuse -= sz
			rv = false
		} else if tempInfo.inuse > tempInfo.hwm {
			tempInfo.hwm = tempInfo.inuse
		}
	}
	tempMutex.Unlock()
	return rv
}

func ReleaseTemp(pathname string, sz int64) {
	loc := path.Dir(pathname)
	tempMutex.Lock()
	if tempInfo.quota > 0 && (pathname == "" || loc == tempInfo.loc) {
		tempInfo.inuse -= sz
		if tempInfo.inuse < 0 {
			logging.Debugf("Error in temp space accounting for %v: inuse=%v, size=%v", tempInfo.loc, tempInfo.inuse, sz)
			tempInfo.inuse = 0
		}
	}
	tempMutex.Unlock()
}

func TempStats() (int64, int64) {
	tempMutex.Lock()
	c := tempInfo.inuse
	h := tempInfo.hwm
	tempMutex.Unlock()
	return c, h
}
