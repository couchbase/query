//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package logger_golog

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/couchbase/query/logging"
)

type goLogger struct {
	logger         *log.Logger
	level          logging.Level
	entryFormatter formatter
}

const (
	_LEVEL  = "_level"
	_MSG    = "_msg"
	_TIME   = "_time"
	_RLEVEL = "_rlevel"
)

func NewLogger(out io.Writer, lvl logging.Level, jsonLogging bool) *goLogger {
	logger := &goLogger{
		logger: log.New(out, "", 0),
		level:  lvl,
	}
	if jsonLogging {
		logger.entryFormatter = &jsonFormatter{}
	} else {
		logger.entryFormatter = &textFormatter{}
	}
	return logger
}

// anonymous function variants

func (gl *goLogger) Loga(level logging.Level, f func() string) {
	if gl.logger == nil {
		return
	}
	if level <= gl.level {
		gl.log(level, logging.NONE, f())
	}
}
func (gl *goLogger) Debuga(f func() string) {
	gl.Loga(logging.DEBUG, f)
}

func (gl *goLogger) Tracea(f func() string) {
	gl.Loga(logging.TRACE, f)
}

func (gl *goLogger) Requesta(rlevel logging.Level, f func() string) {
	if gl.logger == nil {
		return
	}
	if logging.REQUEST <= gl.level {
		gl.log(logging.REQUEST, rlevel, f())
	}
}

func (gl *goLogger) Infoa(f func() string) {
	gl.Loga(logging.INFO, f)
}

func (gl *goLogger) Warna(f func() string) {
	gl.Loga(logging.WARN, f)
}

func (gl *goLogger) Errora(f func() string) {
	gl.Loga(logging.ERROR, f)
}

func (gl *goLogger) Severea(f func() string) {
	gl.Loga(logging.SEVERE, f)
}

func (gl *goLogger) Fatala(f func() string) {
	gl.Loga(logging.FATAL, f)
}

// printf-style variants

func (gl *goLogger) Logf(level logging.Level, format string, args ...interface{}) {
	if gl.logger == nil {
		return
	}
	if level <= gl.level {
		gl.log(level, logging.NONE, fmt.Sprintf(format, args...))
	}
}

func (gl *goLogger) Debugf(format string, args ...interface{}) {
	gl.Logf(logging.DEBUG, format, args...)
}

func (gl *goLogger) Tracef(format string, args ...interface{}) {
	gl.Logf(logging.TRACE, format, args...)
}

func (gl *goLogger) Requestf(rlevel logging.Level, format string, args ...interface{}) {
	if gl.logger == nil {
		return
	}
	if logging.REQUEST <= gl.level {
		gl.log(logging.REQUEST, rlevel, fmt.Sprintf(format, args...))
	}
}

func (gl *goLogger) Infof(format string, args ...interface{}) {
	gl.Logf(logging.INFO, format, args...)
}

func (gl *goLogger) Warnf(format string, args ...interface{}) {
	gl.Logf(logging.WARN, format, args...)
}

func (gl *goLogger) Errorf(format string, args ...interface{}) {
	gl.Logf(logging.ERROR, format, args...)
}

func (gl *goLogger) Severef(format string, args ...interface{}) {
	gl.Logf(logging.SEVERE, format, args...)
}

func (gl *goLogger) Fatalf(format string, args ...interface{}) {
	gl.Logf(logging.FATAL, format, args...)
}

func (gl *goLogger) Level() logging.Level {
	return gl.level
}

func (gl *goLogger) SetLevel(level logging.Level) {
	gl.level = level
}

func (gl *goLogger) log(level logging.Level, rlevel logging.Level, msg string) {
	tm := time.Now().Format("2006-01-02T15:04:05.000-07:00") // time.RFC3339 with milliseconds
	gl.logger.Print(gl.entryFormatter.format(tm, level, rlevel, msg))
}

type formatter interface {
	format(string, logging.Level, logging.Level, string) string
}

type textFormatter struct {
}

func (*textFormatter) format(tm string, level logging.Level, rlevel logging.Level, msg string) string {
	b := &strings.Builder{}
	appendKeyValue(b, _TIME, tm)
	appendKeyValue(b, _LEVEL, level.String())
	if rlevel != logging.NONE {
		appendKeyValue(b, _RLEVEL, rlevel.String())
	}
	appendKeyValue(b, _MSG, msg)
	b.WriteByte('\n')
	return b.String()
}

func appendKeyValue(b *strings.Builder, key, value interface{}) {
	if _, ok := value.(string); ok {
		fmt.Fprintf(b, "%v=%s ", key, value)
	} else {
		fmt.Fprintf(b, "%v=%v ", key, value)
	}
}

type jsonFormatter struct {
}

func (*jsonFormatter) format(tm string, level logging.Level, rlevel logging.Level, msg string) string {
	data := make(map[string]interface{}, 4)
	data[_TIME] = tm
	data[_LEVEL] = level.String()
	if rlevel != logging.NONE {
		data[_RLEVEL] = rlevel.String()
	}
	data[_MSG] = msg
	serialized, _ := json.Marshal(data)
	var b strings.Builder
	b.Write(serialized)
	b.WriteByte('\n')
	return b.String()
}
