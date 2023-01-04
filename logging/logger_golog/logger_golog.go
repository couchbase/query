//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package logger_golog

import (
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

func NewLogger(out io.Writer, lvl logging.Level) *goLogger {
	logger := &goLogger{
		logger: log.New(out, "", 0),
		level:  lvl,
	}
	logger.entryFormatter = &standardFormatter{}
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
	gl.logger.Print(gl.str(level, rlevel, msg))
}

func (gl *goLogger) str(level logging.Level, rlevel logging.Level, msg string) string {
	tm := time.Now().Format("2006-01-02T15:04:05.000-07:00") // time.RFC3339 with milliseconds
	return gl.entryFormatter.format(tm, level, rlevel, msg)
}

func (gl *goLogger) Stringf(level logging.Level, format string, args ...interface{}) string {
	return gl.str(level, logging.NONE, fmt.Sprintf(format, args...))
}

type formatter interface {
	format(string, logging.Level, logging.Level, string) string
}

type standardFormatter struct {
}

func (*standardFormatter) format(tm string, level logging.Level, rlevel logging.Level, msg string) string {
	var b strings.Builder
	b.Grow(len(tm) + len(msg) + 32)
	b.WriteString(tm)
	b.WriteString(" [")
	b.WriteString(level.String())
	if rlevel != logging.NONE {
		b.WriteRune(rune(','))
		b.WriteString(rlevel.String())
	}
	b.WriteString("] ")
	b.WriteString(strings.TrimSpace(msg))
	b.WriteRune(rune('\n'))
	return b.String()
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
