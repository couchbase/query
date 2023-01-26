//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package logging

import (
	"bufio"
	"bytes"
	fmtpkg "fmt"
	"os"
	"sync"
	"time"
)

const _MAX_TRACE_SIZE = 64 * 1024 * 1024

func splitOnNUL(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, 0); i >= 0 {
		return i + 1, data[0:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

type TempFileLogger struct {
	sync.Mutex
	logLevel    Level
	file        *os.File
	debugFilter []_filter
	requestId   string
}

func (this *TempFileLogger) SetRequestId(id string) {
	this.requestId = id
}

func (this *TempFileLogger) Level() Level {
	return this.logLevel
}

func (this *TempFileLogger) SetLevel(l Level) {
	this.logLevel = l
}

func (this *TempFileLogger) Stringf(l Level, format string, args ...interface{}) string {
	var fl string
	if l == DEBUG || l == TRACE {
		fl = getFileLine(1)
	}
	return time.Now().Format(SHORT_TIMESTAMP_FORMAT) + string(l.Abbreviation()) + fmtpkg.Sprintf(format, args...) + fl
}

func (this *TempFileLogger) log(l Level, fn func() string) {
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
		f, err := os.CreateTemp("", "log_"+this.requestId+"_*")
		if err != nil {
			this.Unlock()
			return
		}
		os.Remove(f.Name()) // automatically clean-up
		this.file = f
	}
	n, _ := this.file.Seek(0, os.SEEK_END)
	if n > _MAX_TRACE_SIZE {
		// this is a blunt clearing of the file and could mean somewhat less that the maximum ends up being reported
		this.file.Truncate(0)
		this.file.Seek(0, os.SEEK_END)
		this.file.WriteString("... truncated ...")
		this.file.Write([]byte{0})
	}
	this.file.WriteString(now.Format(SHORT_TIMESTAMP_FORMAT))
	this.file.WriteString(l.Abbreviation())
	this.file.WriteString(fn())
	if len(fl) > 0 {
		this.file.WriteString(fl)
	}
	this.file.Write([]byte{0})
	this.Unlock()
}

func (this *TempFileLogger) Foreach(f func(text string) bool) bool {
	this.Lock()
	if this.file != nil {
		_, err := this.file.Seek(0, os.SEEK_SET)
		if err == nil {
			s := bufio.NewScanner(this.file)
			if s == nil {
				this.Unlock()
				return false
			}
			s.Split(splitOnNUL)
			for {
				if !s.Scan() {
					break
				}
				if !f(s.Text()) {
					this.Unlock()
					return false
				}
			}
		}
	}
	this.Unlock()
	return true
}

func (this *TempFileLogger) Close() {
	if this.file != nil {
		this.file.Truncate(0)
		this.file.Close()
		this.file = nil
	}
}

func (this *TempFileLogger) SetDebugFilter(s string) {
	setDebugFilter(this, &this.debugFilter, s, this.Infof)
}

func (this *TempFileLogger) Loga(level Level, f func() string) {
	this.log(level, f)
}

func (this *TempFileLogger) Debuga(f func() string) {
	this.log(DEBUG, f)
}

func (this *TempFileLogger) Tracea(f func() string) {
	this.log(TRACE, f)
}

func (this *TempFileLogger) Infoa(f func() string) {
	this.log(INFO, f)
}

func (this *TempFileLogger) Warna(f func() string) {
	this.log(WARN, f)
}

func (this *TempFileLogger) Errora(f func() string) {
	this.log(ERROR, f)
}

func (this *TempFileLogger) Severea(f func() string) {
	this.log(SEVERE, f)
}

func (this *TempFileLogger) Fatala(f func() string) {
	this.log(FATAL, f)
}

func (this *TempFileLogger) Logf(level Level, f string, args ...interface{}) {
	this.log(level, func() string { return fmtpkg.Sprintf(f, args...) })
}

func (this *TempFileLogger) Debugf(f string, args ...interface{}) {
	this.log(DEBUG, func() string { return fmtpkg.Sprintf(f, args...) })
}

func (this *TempFileLogger) Tracef(f string, args ...interface{}) {
	this.log(TRACE, func() string { return fmtpkg.Sprintf(f, args...) })
}

func (this *TempFileLogger) Infof(f string, args ...interface{}) {
	this.log(INFO, func() string { return fmtpkg.Sprintf(f, args...) })
}

func (this *TempFileLogger) Warnf(f string, args ...interface{}) {
	this.log(WARN, func() string { return fmtpkg.Sprintf(f, args...) })
}

func (this *TempFileLogger) Errorf(f string, args ...interface{}) {
	this.log(ERROR, func() string { return fmtpkg.Sprintf(f, args...) })
}

func (this *TempFileLogger) Severef(f string, args ...interface{}) {
	this.log(SEVERE, func() string { return fmtpkg.Sprintf(f, args...) })
}

func (this *TempFileLogger) Fatalf(f string, args ...interface{}) {
	this.log(FATAL, func() string { return fmtpkg.Sprintf(f, args...) })
}
