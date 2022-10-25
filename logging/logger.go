//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package logging

import (
	fmtpkg "fmt"
	"path"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
)

type Level int

const (
	NONE    = Level(iota) // Disable all logging
	FATAL                 // System is in severe error state and has to terminate
	SEVERE                // System is in severe error state and cannot recover reliably
	ERROR                 // System is in error state but can recover and continue reliably
	WARN                  // System approaching error state, or is in a correct but undesirable state
	INFO                  // System-level events and status, in correct states
	REQUEST               // Request-level events, with request-specific rlevel
	DEBUG                 // Debug
	TRACE                 // Trace detailed system execution, e.g. function entry / exit
)

func (level Level) String() string {
	return _LEVEL_NAMES[level]
}

var _LEVEL_NAMES = []string{
	DEBUG:   "DEBUG",
	TRACE:   "TRACE",
	REQUEST: "REQUEST",
	INFO:    "INFO",
	WARN:    "WARN",
	ERROR:   "ERROR",
	SEVERE:  "SEVERE",
	FATAL:   "FATAL",
	NONE:    "NONE",
}

var _LEVEL_MAP = map[string]Level{
	"debug":   DEBUG,
	"trace":   TRACE,
	"request": REQUEST,
	"info":    INFO,
	"warn":    WARN,
	"error":   ERROR,
	"severe":  SEVERE,
	"fatal":   FATAL,
	"none":    NONE,
}

// cache logging enablement to improve runtime performance (reduces from multiple tests to a single test on each call)
var (
	cachedDebug   bool
	cachedTrace   bool
	cachedRequest bool
	cachedInfo    bool
	cachedWarn    bool
	cachedError   bool
	cachedSevere  bool
	cachedFatal   bool
	cachedAudit   bool
)

// maintain the cached logging state
func cacheLoggingChange() {
	cachedDebug = !skipLogging(DEBUG)
	cachedTrace = !skipLogging(TRACE)
	cachedRequest = !skipLogging(REQUEST)
	cachedInfo = !skipLogging(INFO)
	cachedWarn = !skipLogging(WARN)
	cachedError = !skipLogging(ERROR)
	cachedSevere = !skipLogging(SEVERE)
	cachedFatal = !skipLogging(FATAL)
}

func ParseLevel(name string) (level Level, ok bool, filter string) {
	level, ok = _LEVEL_MAP[strings.ToLower(name)]
	if !ok {
		if strings.HasPrefix(strings.ToUpper(name), _LEVEL_NAMES[DEBUG]+":") {
			n := len(_LEVEL_NAMES[DEBUG])
			filter = name[n+1:]
			name = name[:n]
			level, ok = _LEVEL_MAP[strings.ToLower(name)]
		} else if strings.HasPrefix(strings.ToUpper(name), _LEVEL_NAMES[TRACE]+":") {
			n := len(_LEVEL_NAMES[TRACE])
			filter = name[n+1:]
			name = name[:n]
			level, ok = _LEVEL_MAP[strings.ToLower(name)]
		}
	}
	return
}

// Logger provides a common interface for logging libraries
type Logger interface {
	// Higher performance
	Loga(level Level, f func() string)
	Debuga(f func() string)
	Tracea(f func() string)
	Requesta(rlevel Level, f func() string)
	Infoa(f func() string)
	Warna(f func() string)
	Errora(f func() string)
	Severea(f func() string)
	Fatala(f func() string)
	Audita(f func() string)

	// Printf style
	Logf(level Level, fmt string, args ...interface{})
	Debugf(fmt string, args ...interface{})
	Tracef(fmt string, args ...interface{})
	Requestf(rlevel Level, fmt string, args ...interface{})
	Infof(fmt string, args ...interface{})
	Warnf(fmt string, args ...interface{})
	Errorf(fmt string, args ...interface{})
	Severef(fmt string, args ...interface{})
	Fatalf(fmt string, args ...interface{})
	Auditf(fmt string, args ...interface{})

	Stringf(level Level, format string, args ...interface{}) string

	/*
		These APIs control the logging level
	*/
	SetLevel(Level) // Set the logging level
	Level() Level   // Get the current logging level
}

var logger Logger = nil
var curLevel Level = DEBUG // initially set to never skip
var debugFilter []*regexp.Regexp

var loggerMutex sync.RWMutex

// All the methods below first acquire the mutex (mostly in exclusive mode)
// and only then check if logging at the current level is enabled.
// This introduces a fair bottleneck for those log entries that should be
// skipped (the majority, at INFO or below levels)
// We try to predict here if we should lock the mutex at all by caching
// the current log level: while dynamically changing logger, there might
// be the odd entry skipped as the new level is cached.
// Since we seem to never change the logger, this is not an issue.
func skipLogging(level Level) bool {
	if logger == nil {
		return true
	}
	return level > curLevel
}

func SetLogger(newLogger Logger) {
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logger = newLogger
	if logger == nil {
		curLevel = NONE
	} else {
		curLevel = newLogger.Level()
	}
	cacheLoggingChange()
}

// we are using deferred unlocking here throughout as we have to do this
// for the anonymous function variants even though it would be more efficient
// to not do this for the printf style variants
// anonymous function variants

func Loga(level Level, f func() string) {
	if skipLogging(level) {
		return
	} else if (level == DEBUG || level == TRACE) && !filterDebug() {
		return
	}
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logger.Loga(level, f)
}

func Debuga(f func() string) {
	if !cachedDebug || !filterDebug() {
		return
	}
	pc, fname, lineno, ok := runtime.Caller(1)
	if ok {
		fnc := runtime.FuncForPC(pc)
		var fl string
		if fnc != nil {
			n := fnc.Name()
			i := strings.LastIndexByte(n, '(')
			if i == -1 {
				i = strings.LastIndexByte(n, '.')
				if i != -1 {
					i++
				}
			}
			if i < 0 {
				i = 0
			}
			fl = fmtpkg.Sprintf(" (%s|%s:%d)", n[i:], path.Base(fname), lineno)
		} else {
			fl = fmtpkg.Sprintf(" (%s:%d)", path.Base(fname), lineno)
		}
		loggerMutex.Lock()
		defer loggerMutex.Unlock()
		logger.Debuga(func() string { return f() + fl })
	} else {
		loggerMutex.Lock()
		defer loggerMutex.Unlock()
		logger.Debuga(f)
	}
}

func Tracea(f func() string) {
	if !cachedTrace || !filterDebug() {
		return
	}
	pc, fname, lineno, ok := runtime.Caller(1)
	if ok {
		fnc := runtime.FuncForPC(pc)
		var fl string
		if fnc != nil {
			n := fnc.Name()
			i := strings.LastIndexByte(n, '(')
			if i == -1 {
				i = strings.LastIndexByte(n, '.')
				if i != -1 {
					i++
				}
			}
			if i < 0 {
				i = 0
			}
			fl = fmtpkg.Sprintf(" (%s|%s:%d)", n[i:], path.Base(fname), lineno)
		} else {
			fl = fmtpkg.Sprintf(" (%s:%d)", path.Base(fname), lineno)
		}
		loggerMutex.Lock()
		defer loggerMutex.Unlock()
		logger.Tracea(func() string { return f() + fl })
	} else {
		loggerMutex.Lock()
		defer loggerMutex.Unlock()
		logger.Tracea(f)
	}
}

func Requesta(rlevel Level, f func() string) {
	if !cachedRequest {
		return
	}
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logger.Requesta(rlevel, f)
}

func Infoa(f func() string) {
	if !cachedInfo {
		return
	}
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logger.Infoa(f)
}

func Warna(f func() string) {
	if !cachedWarn {
		return
	}
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logger.Warna(f)
}

func Errora(f func() string) {
	if !cachedError {
		return
	}
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logger.Errora(f)
}

func Severea(f func() string) {
	if !cachedSevere {
		return
	}
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logger.Severea(f)
}

func Fatala(f func() string) {
	if !cachedFatal {
		return
	}
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logger.Fatala(f)
}

func Audita(f func() string) {
	if !cachedAudit {
		return
	}
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logger.Audita(f)
}

// printf-style variants

func Logf(level Level, fmt string, args ...interface{}) {
	if skipLogging(level) {
		return
	} else if (level == DEBUG || level == TRACE) && !filterDebug() {
		return
	}
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logger.Logf(level, fmt, args...)
}

func Debugf(fmt string, args ...interface{}) {
	if !cachedDebug || !filterDebug() {
		return
	}
	pc, fname, lineno, ok := runtime.Caller(1)
	if ok {
		fnc := runtime.FuncForPC(pc)
		if fnc != nil {
			n := fnc.Name()
			i := strings.LastIndexByte(n, '(')
			if i == -1 {
				i = strings.LastIndexByte(n, '.')
				if i != -1 {
					i++
				}
			}
			if i < 0 {
				i = 0
			}
			f := fmtpkg.Sprintf(" (%s|%s:%d)", n[i:], path.Base(fname), lineno)
			fmt = fmt + f
		} else {
			f := fmtpkg.Sprintf(" (%s:%d)", path.Base(fname), lineno)
			fmt = fmt + f
		}
	}
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logger.Debugf(fmt, args...)
}

func Tracef(fmt string, args ...interface{}) {
	if !cachedTrace || !filterDebug() {
		return
	}
	pc, fname, lineno, ok := runtime.Caller(1)
	if ok {
		fnc := runtime.FuncForPC(pc)
		if fnc != nil {
			n := fnc.Name()
			i := strings.LastIndexByte(n, '(')
			if i == -1 {
				i = strings.LastIndexByte(n, '.')
				if i != -1 {
					i++
				}
			}
			if i < 0 {
				i = 0
			}
			f := fmtpkg.Sprintf(" (%s|%s:%d)", n[i:], path.Base(fname), lineno)
			fmt = fmt + f
		} else {
			f := fmtpkg.Sprintf(" (%s:%d)", path.Base(fname), lineno)
			fmt = fmt + f
		}
	}
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logger.Tracef(fmt, args...)
}

func Requestf(rlevel Level, fmt string, args ...interface{}) {
	if !cachedRequest {
		return
	}
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logger.Requestf(rlevel, fmt, args...)
}

func Infof(fmt string, args ...interface{}) {
	if !cachedInfo {
		return
	}
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logger.Infof(fmt, args...)
}

func Warnf(fmt string, args ...interface{}) {
	if !cachedWarn {
		return
	}
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logger.Warnf(fmt, args...)
}

func Errorf(fmt string, args ...interface{}) {
	if !cachedError {
		return
	}
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logger.Errorf(fmt, args...)
}

func Severef(fmt string, args ...interface{}) {
	if !cachedSevere {
		return
	}
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logger.Severef(fmt, args...)
}

func Fatalf(fmt string, args ...interface{}) {
	if !cachedFatal {
		return
	}
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logger.Fatalf(fmt, args...)
}

func Auditf(fmt string, args ...interface{}) {
	if !cachedAudit {
		return
	}
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logger.Auditf(fmt, args...)
}

func SetLevel(level Level) {
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logger.SetLevel(level)
	curLevel = level
	cacheLoggingChange()
}

func LogLevel() Level {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()
	if logger == nil {
		return NONE
	}
	return logger.Level()
}

func Stackf(level Level, fmt string, args ...interface{}) {
	if skipLogging(level) {
		return
	} else if level == DEBUG && !filterDebug() {
		return
	}
	buf := make([]byte, 1<<16)
	n := runtime.Stack(buf, false)
	s := string(buf[0:n])
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logger.Logf(level, fmt, args...)
	logger.Logf(level, s)
}

func Stringf(level Level, fmt string, args ...interface{}) string {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()
	return logger.Stringf(level, fmt, args...)
}

func SetDebugFilter(s string) {
	if s == "" {
		if debugFilter != nil {
			loggerMutex.Lock()
			debugFilter = nil
			loggerMutex.Unlock()
		}
		return
	}
	pats := strings.Split(s, ";")
	df := make([]*regexp.Regexp, 0, len(pats))
	for _, p := range pats {
		f, err := regexp.Compile(p)
		if err == nil {
			df = append(df, f)
			Infof("Added debug logging filter: '%s'", p)
		}
	}
	loggerMutex.Lock()
	debugFilter = df
	loggerMutex.Unlock()
}

func filterDebug() bool {
	if debugFilter == nil || len(debugFilter) == 0 {
		return true
	}

	_, pathname, _, ok := runtime.Caller(2)
	if !ok {
		return false
	}
	loggerMutex.RLock()
	df := debugFilter
	loggerMutex.RUnlock()
	for _, p := range df {
		if p.MatchString(pathname) {
			return true
		}
	}
	return false
}

func debugFilterString() string {
	loggerMutex.RLock()
	df := debugFilter
	loggerMutex.RUnlock()
	s := ":"
	for _, e := range df {
		s += e.String() + ";"
	}
	return s[:len(s)-1]
}

func LogLevelString() string {
	l := LogLevel()
	if (l != DEBUG && l != TRACE) || len(debugFilter) == 0 {
		return l.String()
	}
	return l.String() + debugFilterString()
}

func DumpAllStacks(level Level, msg string) {
	if skipLogging(level) {
		return
	}
	stacks := make([]runtime.StackRecord, runtime.NumGoroutine())
	n, ok := runtime.GoroutineProfile(stacks)
	if !ok {
		buf := make([]byte, 20*1024*1024)
		copy(buf, msg+"\n")
		n := runtime.Stack(buf[len(msg)+1:], true)
		s := string(buf[:n+len(msg)+1])
		Logf(level, s)
	} else {
		stacks = stacks[:n]
		cnt := make(map[int]int)
		for i := 0; i < len(stacks); {
			same := false
			for j := 0; j < i; j++ {
				if len(stacks[i].Stack()) == len(stacks[j].Stack()) {
					same = true
					for ii := range stacks[i].Stack0 {
						if stacks[i].Stack0[ii] != stacks[j].Stack0[ii] {
							same = false
							break
						} else if stacks[i].Stack0[ii] == 0 {
							break
						}
					}
					if same {
						cnt[j] = cnt[j] + 1
						copy(stacks[i:], stacks[i+1:])
						stacks = stacks[:len(stacks)-1]
						break
					}
				}
			}
			if !same {
				cnt[i] = 1
				i++
			}
		}
		order := make([][2]int, len(cnt))
		i := 0
		for k, v := range cnt {
			order[i][0] = k
			order[i][1] = v
			i++
		}
		cnt = nil
		sort.Slice(order, func(i int, j int) bool {
			if order[i][1] > order[j][1] {
				return true
			} else if order[i][1] < order[j][1] {
				return false
			}
			return len(stacks[order[i][0]].Stack()) > len(stacks[order[j][0]].Stack())
		})
		var sb strings.Builder
		sb.WriteString(msg)
		sb.WriteRune('\n')
		for i := range order {
			sb.WriteString(fmtpkg.Sprintf("%d @\n", order[i][1]))
			frames := runtime.CallersFrames(stacks[order[i][0]].Stack())
			m := 0
			for more := true; more == true; {
				var frame runtime.Frame
				frame, more = frames.Next()
				if frame.Func != nil {
					l := len(frame.Function)
					if l > m {
						m = l
					}
				}
			}
			frames = runtime.CallersFrames(stacks[order[i][0]].Stack())
			for more := true; more == true; {
				var frame runtime.Frame
				frame, more = frames.Next()
				if frame.Func != nil {
					sb.WriteRune(' ')
					sb.WriteString(frame.Function)
					for n := m - len(frame.Function); n > 0; n-- {
						sb.WriteRune(' ')
					}
					sb.WriteRune(' ')
					sb.WriteString(frame.File)
					sb.WriteRune(':')
					sb.WriteString(fmtpkg.Sprintf("%d", frame.Line))
					sb.WriteRune('\n')
				}
			}
			sb.WriteRune('\n')
		}
		Logf(level, sb.String())
	}
}
