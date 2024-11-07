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
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/couchbase/query/logging"
)

type TempFile struct {
	os.File
}

// On *nix based systems we could just unlink the name immediately after creation but on others we can't, so we wrap the Close
// method so that we can clean-up automatically (during normal operations).  In the event of an abnormal process exit, the
// lingering temp files will be cleaned-up when the temporary location is configured again on restart.  This does leave the
// potential for "lost" files if there is a configuration change after an abnormal exit and before the next restart (highly
// unlikely); these would have to be cleaned up manually.
func (this *TempFile) Close() error {
	err := this.File.Close()
	if err == nil {
		os.Remove(this.Name())
		runtime.SetFinalizer(this, nil)
	}
	return err
}

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
	loc = filepath.Clean(loc)
	if !filepath.IsAbs(loc) {
		tempMutex.Unlock()
		logging.Errorf("Attempt to set relative temporary path: %v", loc)
		return fmt.Errorf("Attempt to set relative temporary path")
	} else if _, err := os.Stat(loc); err != nil {
		tempMutex.Unlock()
		logging.Errorf("Attempt to set invalid or inaccessible temporary path: %v (%v)", loc, err)
		return fmt.Errorf("Attempt to set invalid or inaccessible temporary path")
	}
	if tempInfo.loc != loc {
		tempInfo.inuse = 0
		cleanup(loc) // clean-up the new path before we start, if we're just changing the quota the leave the contents alone
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

func CreateTemp(pattern string) (*TempFile, error) {
	tf := &TempFile{}
	f, err := os.CreateTemp(TempLocation(), pattern)
	tf.File = *f
	if logging.LogLevel() == logging.DEBUG {
		runtime.SetFinalizer(tf, func(i interface{}) {
			if tf, ok := i.(*TempFile); ok {
				logging.Debugf("Temp file finaliser for %s called.  Missing close.", tf.Name())
				tf.Close()
			}
		})
	}
	return tf, err
}

func UseTemp(pathname string, sz int64) bool {
	rv := true
	loc := filepath.Dir(pathname)
	tempMutex.Lock()
	if tempInfo.quota > 0 && (pathname == "" || loc == tempInfo.loc) {
		tempInfo.inuse += sz
		if tempInfo.inuse > tempInfo.quota {
			tempInfo.inuse -= sz
			rv = false
		} else if tempInfo.inuse > tempInfo.hwm {
			tempInfo.hwm = tempInfo.inuse
		}
	} else if tempInfo.quota > 0 && logging.LogLevel() == logging.DEBUG {
		logging.Debugf("UseTemp(%s, %d) - %s not in temp path: %s", pathname, sz, loc, tempInfo.loc)
	}
	tempMutex.Unlock()
	return rv
}

func ReleaseTemp(pathname string, sz int64) {
	loc := filepath.Dir(pathname)
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

var prefixes []string

// remove files with registered prefixes from the supplied path
func cleanup(path string) {
	if len(prefixes) == 0 || path == "" {
		return
	}
	d, err := os.Open(path)
	if err != nil {
		logging.Debugf("%v", err)
		return
	}
	n := 0
	sz := int64(0)
	for {
		ents, err := d.ReadDir(10)
		if err != nil {
			break
		}
		for i := range ents {
			logging.Debugf("%s", ents[i].Name())
			if ents[i].IsDir() {
				continue
			}
			for j := range prefixes {
				if strings.HasPrefix(ents[i].Name(), prefixes[j]) {
					if info, err := ents[i].Info(); err == nil {
						sz += info.Size()
					}
					pathname := filepath.Join(path, ents[i].Name())
					logging.Debugf("%s", pathname)
					os.Remove(pathname)
					n++
					break
				}
			}
		}
		if len(ents) < 10 {
			break
		}
	}
	d.Close()
	logging.Infof("Cleaned up %s in %d temporary file(s) from: %s", logging.HumanReadableSize(sz, false), n, path)
}

func RegisterTempPattern(pattern string) {
	prefixes = append(prefixes, strings.TrimSuffix(pattern, "*"))
}
