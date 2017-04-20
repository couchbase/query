//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package logger_golog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

func (gl *goLogger) Logp(level logging.Level, msg string, kv ...logging.Pair) {
	if gl.logger == nil {
		return
	}
	if level <= gl.level {
		e := newLogEntry(msg, level)
		copyPairs(e, kv)
		gl.log(e)
	}
}

func (gl *goLogger) Debugp(msg string, kv ...logging.Pair) {
	gl.Logp(logging.DEBUG, msg, kv...)
}

func (gl *goLogger) Tracep(msg string, kv ...logging.Pair) {
	gl.Logp(logging.TRACE, msg, kv...)
}

func (gl *goLogger) Requestp(rlevel logging.Level, msg string, kv ...logging.Pair) {
	if gl.logger == nil {
		return
	}
	if logging.REQUEST <= gl.level {
		e := newLogEntry(msg, logging.REQUEST)
		e.Rlevel = rlevel
		copyPairs(e, kv)
		gl.log(e)
	}
}

func (gl *goLogger) Infop(msg string, kv ...logging.Pair) {
	gl.Logp(logging.INFO, msg, kv...)
}

func (gl *goLogger) Warnp(msg string, kv ...logging.Pair) {
	gl.Logp(logging.WARN, msg, kv...)
}

func (gl *goLogger) Errorp(msg string, kv ...logging.Pair) {
	gl.Logp(logging.ERROR, msg, kv...)
}

func (gl *goLogger) Severep(msg string, kv ...logging.Pair) {
	gl.Logp(logging.SEVERE, msg, kv...)
}

func (gl *goLogger) Fatalp(msg string, kv ...logging.Pair) {
	gl.Logp(logging.FATAL, msg, kv...)
}

func (gl *goLogger) Logm(level logging.Level, msg string, kv logging.Map) {
	if gl.logger == nil {
		return
	}
	if level <= gl.level {
		e := newLogEntry(msg, level)
		e.Data = kv
		gl.log(e)
	}
}

func (gl *goLogger) Debugm(msg string, kv logging.Map) {
	gl.Logm(logging.DEBUG, msg, kv)
}

func (gl *goLogger) Tracem(msg string, kv logging.Map) {
	gl.Logm(logging.TRACE, msg, kv)
}

func (gl *goLogger) Requestm(rlevel logging.Level, msg string, kv logging.Map) {
	if gl.logger == nil {
		return
	}
	if logging.REQUEST <= gl.level {
		e := newLogEntry(msg, logging.REQUEST)
		e.Rlevel = rlevel
		e.Data = kv
		gl.log(e)
	}
}

func (gl *goLogger) Infom(msg string, kv logging.Map) {
	gl.Logm(logging.INFO, msg, kv)
}

func (gl *goLogger) Warnm(msg string, kv logging.Map) {
	gl.Logm(logging.WARN, msg, kv)
}

func (gl *goLogger) Errorm(msg string, kv logging.Map) {
	gl.Logm(logging.ERROR, msg, kv)
}

func (gl *goLogger) Severem(msg string, kv logging.Map) {
	gl.Logm(logging.SEVERE, msg, kv)
}

func (gl *goLogger) Fatalm(msg string, kv logging.Map) {
	gl.Logm(logging.FATAL, msg, kv)
}

func (gl *goLogger) Logf(level logging.Level, format string, args ...interface{}) {
	if gl.logger == nil {
		return
	}
	if level <= gl.level {
		e := newLogEntry(fmt.Sprintf(format, args...), level)
		gl.log(e)
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
		e := newLogEntry(fmt.Sprintf(format, args...), logging.REQUEST)
		e.Rlevel = rlevel
		gl.log(e)
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

func (gl *goLogger) log(newEntry *logEntry) {
	s := gl.entryFormatter.format(newEntry)
	gl.logger.Print(s)
}

type logEntry struct {
	Time    string
	Level   logging.Level
	Rlevel  logging.Level
	Message string
	Data    logging.Map
}

func newLogEntry(msg string, level logging.Level) *logEntry {
	return &logEntry{
		Time:    time.Now().Format("2006-01-02T15:04:05.000-07:00"), // time.RFC3339 with milliseconds
		Level:   level,
		Rlevel:  logging.NONE,
		Message: msg,
	}
}

func copyPairs(newEntry *logEntry, pairs []logging.Pair) {
	newEntry.Data = make(logging.Map, len(pairs))
	for _, p := range pairs {
		newEntry.Data[p.Name] = p.Value
	}
}

type formatter interface {
	format(*logEntry) string
}

type textFormatter struct {
}

func (*textFormatter) format(newEntry *logEntry) string {
	b := &bytes.Buffer{}
	appendKeyValue(b, _TIME, newEntry.Time)
	appendKeyValue(b, _LEVEL, newEntry.Level.String())
	if newEntry.Rlevel != logging.NONE {
		appendKeyValue(b, _RLEVEL, newEntry.Rlevel.String())
	}
	appendKeyValue(b, _MSG, newEntry.Message)
	for key, value := range newEntry.Data {
		appendKeyValue(b, key, value)
	}
	b.WriteByte('\n')
	s := bytes.NewBuffer(b.Bytes())
	return s.String()
}

func appendKeyValue(b *bytes.Buffer, key, value interface{}) {
	if _, ok := value.(string); ok {
		fmt.Fprintf(b, "%v=%s ", key, value)
	} else {
		fmt.Fprintf(b, "%v=%v ", key, value)
	}
}

type jsonFormatter struct {
}

func (*jsonFormatter) format(newEntry *logEntry) string {
	if newEntry.Data == nil {
		newEntry.Data = make(logging.Map, 5)
	}
	newEntry.Data[_TIME] = newEntry.Time
	newEntry.Data[_LEVEL] = newEntry.Level.String()
	if newEntry.Rlevel != logging.NONE {
		newEntry.Data[_RLEVEL] = newEntry.Rlevel.String()
	}
	newEntry.Data[_MSG] = newEntry.Message
	serialized, _ := json.Marshal(newEntry.Data)
	s := bytes.NewBuffer(append(serialized, '\n'))
	return s.String()
}
