//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package logging

import (
	fmtpkg "fmt"
	"os"
	"sync"
	"time"
)

type FileLogger struct {
	sync.Mutex
	logLevel    Level
	file        *os.File
	debugFilter []_filter
	requestId   string
}

func (this *FileLogger) SetRequestId(id string) {
	this.requestId = " " + id
}

func (this *FileLogger) Level() Level {
	return this.logLevel
}

func (this *FileLogger) SetLevel(l Level) {
	this.logLevel = l
}

func (this *FileLogger) Stringf(l Level, format string, args ...interface{}) string {
	var fl string
	if l == DEBUG || l == TRACE {
		fl = getFileLine(1)
	}
	return time.Now().Format(SHORT_TIMESTAMP_FORMAT) + string(l.Abbreviation()) + fmtpkg.Sprintf(format, args...) + fl
}

func (this *FileLogger) log(l Level, fn func() string) {
	if l < this.logLevel {
		return
	}
	now := time.Now()
	var fl string
	if l == DEBUG || l == TRACE {
		fl = findCaller(l)
		if len(this.debugFilter) > 0 && len(fl) > 0 {
			this.Lock()
			df := this.debugFilter
			this.Unlock()
			ok := false
			for _, p := range df {
				if p.re.MatchString(fl) {
					// first match applies
					ok = !p.exclude
					break
				}
			}
			if !ok {
				return
			}
		}
	}
	this.Lock()
	if this.file == nil {
		// Use OS temp location
		// Single possible file so at most _MAX_TRACE_SIZE space used (space shouldn't be exhausted)
		f, err := os.Create(fmtpkg.Sprintf("%s%ccb_query_request.log", os.TempDir(), os.PathSeparator))
		if err != nil {
			this.Unlock()
			return
		}
		this.file = f
	}
	n, _ := this.file.Seek(0, os.SEEK_END)
	if n > _MAX_TRACE_SIZE {
		// this is a blunt clearing of the file and could mean somewhat less that the maximum ends up being reported
		this.file.Truncate(0)
		this.file.Seek(0, os.SEEK_END)
		this.file.WriteString("... truncated ...\n")
	}
	this.file.WriteString(now.Format(SHORT_TIMESTAMP_FORMAT))
	this.file.WriteString(this.requestId)
	this.file.WriteString(l.Abbreviation())
	this.file.WriteString(fn())
	if len(fl) > 0 {
		this.file.WriteString(fl)
	}
	this.file.WriteString("\n")
	this.Unlock()
}

func (this *FileLogger) Foreach(f func(text string) bool) bool {
	this.Lock()
	if this.file != nil {
		if !f(this.file.Name()) {
			this.Unlock()
			return false
		}
	}
	this.Unlock()
	return true
}

func (this *FileLogger) Close() {
	if this.file != nil {
		this.file.Close()
		this.file = nil
	}
}

func (this *FileLogger) SetDebugFilter(s string) {
	setDebugFilter(this, &this.debugFilter, s, this.Infof)
}

func (this *FileLogger) Loga(level Level, f func() string) {
	this.log(level, f)
}

func (this *FileLogger) Debuga(f func() string) {
	this.log(DEBUG, f)
}

func (this *FileLogger) Tracea(f func() string) {
	this.log(TRACE, f)
}

func (this *FileLogger) Infoa(f func() string) {
	this.log(INFO, f)
}

func (this *FileLogger) Warna(f func() string) {
	this.log(WARN, f)
}

func (this *FileLogger) Errora(f func() string) {
	this.log(ERROR, f)
}

func (this *FileLogger) Severea(f func() string) {
	this.log(SEVERE, f)
}

func (this *FileLogger) Fatala(f func() string) {
	this.log(FATAL, f)
}

func (this *FileLogger) Logf(level Level, f string, args ...interface{}) {
	this.log(level, func() string { return fmtpkg.Sprintf(f, args...) })
}

func (this *FileLogger) Debugf(f string, args ...interface{}) {
	this.log(DEBUG, func() string { return fmtpkg.Sprintf(f, args...) })
}

func (this *FileLogger) Tracef(f string, args ...interface{}) {
	this.log(TRACE, func() string { return fmtpkg.Sprintf(f, args...) })
}

func (this *FileLogger) Infof(f string, args ...interface{}) {
	this.log(INFO, func() string { return fmtpkg.Sprintf(f, args...) })
}

func (this *FileLogger) Warnf(f string, args ...interface{}) {
	this.log(WARN, func() string { return fmtpkg.Sprintf(f, args...) })
}

func (this *FileLogger) Errorf(f string, args ...interface{}) {
	this.log(ERROR, func() string { return fmtpkg.Sprintf(f, args...) })
}

func (this *FileLogger) Severef(f string, args ...interface{}) {
	this.log(SEVERE, func() string { return fmtpkg.Sprintf(f, args...) })
}

func (this *FileLogger) Fatalf(f string, args ...interface{}) {
	this.log(FATAL, func() string { return fmtpkg.Sprintf(f, args...) })
}
