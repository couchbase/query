//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build debug

package logging

// Helper functions for instrumentation - not built into released product

import (
	fmtpkg "fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"
)

func DBG(fmt string, args ...interface{}) {
	if skipLogging(INFO) {
		return
	}
	Infof("DBG:"+fmt+getFileLine(1), args...)
}

func DBGSTK(fmt string, args ...interface{}) {
	if skipLogging(INFO) {
		return
	}
	var sb strings.Builder
	pc := make([]uintptr, 8)
	n := runtime.Callers(2, pc)
	if n > 0 {
		frames := runtime.CallersFrames(pc)
		ok := true
		var frame runtime.Frame
		for ok {
			frame, ok = frames.Next()
			if sb.Len() > 0 {
				sb.WriteRune('|')
			}
			sb.WriteString(path.Base(frame.File))
			sb.WriteRune(':')
			sb.WriteString(fmtpkg.Sprintf("%d", frame.Line))
		}
	}
	if sb.Len() > 0 {
		fmt += " (" + sb.String() + ")"
	}
	Infof("DBG:"+fmt, args...)
}

var m sync.Mutex

func DBGF(fmt string, args ...interface{}) {
	m.Lock()
	f, err := os.OpenFile("/tmp/debug.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err == nil {
		f.WriteString(time.Now().Format("15:04:05.000 "))
		f.WriteString(fmtpkg.Sprintf(fmt, args...))
		f.WriteString(getFileLine(1) + "\n")
		f.Close()
	}
	m.Unlock()
}
