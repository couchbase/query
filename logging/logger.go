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
	"log"
	"path"
	"regexp"
	"runtime"
	"runtime/debug"
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

func (level Level) Abbreviation() string {
	return _ABBREVIATED_LEVEL_NAMES[level]
}

func (level Level) FunctionName() string {
	n := _LEVEL_NAMES[level]
	n = n[:1] + strings.ToLower(n[1:])
	return n
}

var _LEVEL_NAMES = []string{
	DEBUG:  "DEBUG",
	TRACE:  "TRACE",
	INFO:   "INFO",
	WARN:   "WARN",
	ERROR:  "ERROR",
	SEVERE: "SEVERE",
	FATAL:  "FATAL",
	NONE:   "NONE",
}

var _ABBREVIATED_LEVEL_NAMES = []string{
	DEBUG:  " D ",
	TRACE:  " T ",
	INFO:   " I ",
	WARN:   " W ",
	ERROR:  " E ",
	SEVERE: " S ",
	FATAL:  " F ",
	NONE:   " N ",
}

var _LEVEL_MAP = map[string]Level{
	"debug":  DEBUG,
	"trace":  TRACE,
	"info":   INFO,
	"warn":   WARN,
	"error":  ERROR,
	"severe": SEVERE,
	"fatal":  FATAL,
	"none":   NONE,
}

const FULL_TIMESTAMP_FORMAT = "2006-01-02T15:04:05.000-07:00" // time.RFC3339 with milliseconds
const SHORT_TIMESTAMP_FORMAT = "2006-01-02T15:04:05.000"

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
type Log interface {
	// Higher performance
	Loga(level Level, f func() string)
	Debuga(f func() string)
	Tracea(f func() string)
	Infoa(f func() string)
	Warna(f func() string)
	Errora(f func() string)
	Severea(f func() string)
	Fatala(f func() string)

	// Printf style
	Logf(level Level, fmt string, args ...interface{})
	Debugf(fmt string, args ...interface{})
	Tracef(fmt string, args ...interface{})
	Infof(fmt string, args ...interface{})
	Warnf(fmt string, args ...interface{})
	Errorf(fmt string, args ...interface{})
	Severef(fmt string, args ...interface{})
	Fatalf(fmt string, args ...interface{})
}

type Logger interface {
	Log

	Stringf(level Level, format string, args ...interface{}) string

	/*
		These APIs control the logging level
	*/
	SetLevel(Level) // Set the logging level
	Level() Level   // Get the current logging level
}

type RequestLogger interface {
	Logger
	SetRequestId(string)
	Foreach(func(string) bool) bool
	Close()
}

var logger Logger = nil
var curLevel Level = DEBUG // initially set to never skip

type _filter struct {
	re      *regexp.Regexp
	exclude bool
}

// Debug filters are optional regular expressions which are matched against a string constructed from the colon separated path name
// and line number of the calling function.  These allow filtering of DEBUG level messages to avoid flooding the output and thereby
// making DEBUG level logging easier to process (you can concentrate on just the files you're interested in, or exclude ones you're
// not).
// Debug filters are specified as part of the LOGLEVEL=debug value but can only be set via an individual node's REST API; none of
// the cluster tools (UI, couchbase-cli, etc.) will accept debug filters in the LOGLEVEL specification.
// To specify an include filter, list the pattern after "debug:" in the LOGLEVEL setting value.  To specify an exclude filter, begin
// the filter with a minus sign (to start an include pattern with a minus sign, use backslash to escape it).
// Multiple semi-colon separated filters can be specified, and the specification of include & exclude filters can be mixed but only
// the first matching filter is applied.
// In addition to regular filters, commands may be embedded in the debug specification.  Commands are specified just like filters
// but their first character must be an exclamation point; they do not filter DEBUG level messages.
// INFO level messages are logged when filters are added (no messages are logged when they're removed; the LOGLEVEL change messages
// imply a reset of the filters).
//
//
// Examples:
//
// LOGLEVEL=debug
//
// Regular common level setting with no filters.
//
// LOGLEVEL=debug:memcached
//
// Only output where the file info matches "memcached".  e.g. a message at "/path/to/serverstats.go:264" will not be logged since
// the pattern won't match but "/path/to/go_memcached_test.go:123" will.
//
// LOGLEVEL=debug:memcached;-test
//
// Only output where the file info matches "memcached" or the file info does not match "test".  e.g. a message at
// "/path/to/memcached_test.go:123" *will* be logged - only the first matching filter applies - but "/path/to/ffdc_test.go:123"
// will not.
//
// LOGLEVEL=debug:-test;memcached
//
// Only output where the file info the file info does not match "test" or matches "memcached".  e.g. a message at
// "/path/to/memcached_test.go:123" will *not* be logged, neither will one at "/path/to/server.go:123", but one at
// "/path/to/memcached.go:123" will.
//
// LOGLEVEL=debug:/ffdc;serverstats.go:123
//
// Only output where the file info matches "/ffdc" or "serverstats.go:123".  e.g. a message at "/path/to/serverstats.go:1230" will
// be logged, as will a message at "/path/to/ffdc.go:123" but one at "/path/to/server_ffdc.go:1034" will not nor will one at
// "/path/to/serverstats.go:456".
//
// LOGLEVEL=debug:-manager;!+dumpcore
//
// output as long as the file info doesn't match "manager" plus turn on core dumping.  e.g. a message at
// "/path/to/service_manager.go:123" will be suppressed by this filter
//
// LOGLEVEL=debug:!-dumpcore
//
// No filtering (level set to debug) plus disable core dumping.

var debugFilter []_filter

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
	logger = newLogger
	if logger == nil {
		curLevel = NONE
	} else {
		curLevel = newLogger.Level()
	}
	cacheLoggingChange()
	loggerMutex.Unlock()
}

func Loga(level Level, f func() string, args ...interface{}) {
	if len(args) > 0 {
		if l, ok := args[0].(Log); ok {
			l.Loga(level, f)
		}
	}
	if skipLogging(level) {
		return
	} else if (level == DEBUG || level == TRACE) && !filterDebug() {
		return
	}
	loggerMutex.Lock()
	logger.Loga(level, f)
	loggerMutex.Unlock()
}

func Debuga(f func() string, args ...interface{}) {
	if len(args) > 0 {
		if l, ok := args[0].(Log); ok {
			l.Debuga(f)
		}
	}
	if !cachedDebug || !filterDebug() {
		return
	}
	fl := getFileLine(1)
	loggerMutex.Lock()
	if fl != "" {
		logger.Debuga(func() string { return f() + fl })
	} else {
		logger.Debuga(f)
	}
	loggerMutex.Unlock()
}

func Tracea(f func() string, args ...interface{}) {
	if len(args) > 0 {
		if l, ok := args[0].(Log); ok {
			l.Tracea(f)
		}
	}
	if !cachedTrace || !filterDebug() {
		return
	}
	fl := getFileLine(1)
	loggerMutex.Lock()
	if fl != "" {
		logger.Tracea(func() string { return f() + fl })
	} else {
		logger.Tracea(f)
	}
	loggerMutex.Unlock()
}

func Infoa(f func() string, args ...interface{}) {
	if len(args) > 0 {
		if l, ok := args[0].(Log); ok {
			l.Infoa(f)
		}
	}
	if !cachedInfo {
		return
	}
	loggerMutex.Lock()
	logger.Infoa(f)
	loggerMutex.Unlock()
}

func Warna(f func() string, args ...interface{}) {
	if len(args) > 0 {
		if l, ok := args[0].(Log); ok {
			l.Warna(f)
		}
	}
	if !cachedWarn {
		return
	}
	loggerMutex.Lock()
	logger.Warna(f)
	loggerMutex.Unlock()
}

func Errora(f func() string, args ...interface{}) {
	if len(args) > 0 {
		if l, ok := args[0].(Log); ok {
			l.Errora(f)
		}
	}
	if !cachedError {
		return
	}
	loggerMutex.Lock()
	logger.Errora(f)
	loggerMutex.Unlock()
}

func Severea(f func() string, args ...interface{}) {
	if len(args) > 0 {
		if l, ok := args[0].(Log); ok {
			l.Severea(f)
		}
	}
	if !cachedSevere {
		return
	}
	loggerMutex.Lock()
	logger.Severea(f)
	loggerMutex.Unlock()
}

func Fatala(f func() string, args ...interface{}) {
	if len(args) > 0 {
		if l, ok := args[0].(Log); ok {
			l.Fatala(f)
		}
	}
	if !cachedFatal {
		return
	}
	loggerMutex.Lock()
	logger.Fatala(f)
	loggerMutex.Unlock()
}

// printf-style variants

func Logf(level Level, fmt string, args ...interface{}) {
	n := len(args)
	if n > 0 {
		if l, ok := args[n-1].(Log); ok {
			n--
			l.Logf(level, fmt, args[:n]...)
		}
	}
	if skipLogging(level) {
		return
	} else if (level == DEBUG || level == TRACE) && !filterDebug() {
		return
	}
	loggerMutex.Lock()
	logger.Logf(level, fmt, args[:n]...)
	loggerMutex.Unlock()
}

func Debugf(fmt string, args ...interface{}) {
	n := len(args)
	if n > 0 {
		if l, ok := args[n-1].(Log); ok {
			n--
			l.Debugf(fmt, args[:n]...)
		}
	}
	if !cachedDebug || !filterDebug() {
		return
	}
	fmt += getFileLine(1)
	loggerMutex.Lock()
	logger.Debugf(fmt, args[:n]...)
	loggerMutex.Unlock()
}

func Tracef(fmt string, args ...interface{}) {
	n := len(args)
	if n > 0 {
		if l, ok := args[n-1].(Log); ok {
			n--
			l.Tracef(fmt, args[:n]...)
		}
	}
	if !cachedTrace || !filterDebug() {
		return
	}
	fmt += getFileLine(1)
	loggerMutex.Lock()
	logger.Tracef(fmt, args[:n]...)
	loggerMutex.Unlock()
}

func Infof(fmt string, args ...interface{}) {
	n := len(args)
	if n > 0 {
		if l, ok := args[n-1].(Log); ok {
			n--
			l.Infof(fmt, args[:n]...)
		}
	}
	if !cachedInfo {
		return
	}
	loggerMutex.Lock()
	logger.Infof(fmt, args[:n]...)
	loggerMutex.Unlock()
}

func Warnf(fmt string, args ...interface{}) {
	n := len(args)
	if n > 0 {
		if l, ok := args[n-1].(Log); ok {
			n--
			l.Warnf(fmt, args[:n]...)
		}
	}
	if !cachedWarn {
		return
	}
	loggerMutex.Lock()
	logger.Warnf(fmt, args[:n]...)
	loggerMutex.Unlock()
}

func Errorf(fmt string, args ...interface{}) {
	n := len(args)
	if n > 0 {
		if l, ok := args[n-1].(Log); ok {
			n--
			l.Errorf(fmt, args[:n]...)
		}
	}
	if !cachedError {
		return
	}
	loggerMutex.Lock()
	logger.Errorf(fmt, args[:n]...)
	loggerMutex.Unlock()
}

func Severef(fmt string, args ...interface{}) {
	n := len(args)
	if n > 0 {
		if l, ok := args[n-1].(Log); ok {
			n--
			l.Severef(fmt, args[:n]...)
		}
	}
	if !cachedSevere {
		return
	}
	loggerMutex.Lock()
	logger.Severef(fmt, args[:n]...)
	loggerMutex.Unlock()
}

func Fatalf(fmt string, args ...interface{}) {
	n := len(args)
	if n > 0 {
		if l, ok := args[n-1].(Log); ok {
			n--
			l.Fatalf(fmt, args[:n]...)
		}
	}
	if !cachedFatal {
		return
	}
	loggerMutex.Lock()
	logger.Fatalf(fmt, args[:n]...)
	loggerMutex.Unlock()
}

func SetLevel(level Level) {
	loggerMutex.Lock()
	logger.SetLevel(level)
	curLevel = level
	cacheLoggingChange()
	loggerMutex.Unlock()
}

func LogLevel() Level {
	loggerMutex.RLock()
	rv := NONE
	if logger != nil {
		rv = logger.Level()
	}
	loggerMutex.RUnlock()
	return rv
}

func Logging(l Level) bool {
	return !skipLogging(l)
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
	logger.Logf(level, fmt, args...)
	logger.Logf(level, s)
	loggerMutex.Unlock()
}

func Stringf(level Level, fmt string, args ...interface{}) string {
	loggerMutex.RLock()
	rv := logger.Stringf(level, fmt, args...)
	loggerMutex.RUnlock()
	return rv
}

func SetDebugFilter(s string) {
	setDebugFilter(&loggerMutex, &debugFilter, s, Infof)
}

func setDebugFilter(pMutex sync.Locker, pFilters *[]_filter, s string, log func(string, ...interface{})) {
	if s == "" {
		if *pFilters != nil {
			pMutex.Lock()
			*pFilters = nil
			pMutex.Unlock()
		}
		return
	}
	pats := strings.Split(s, ";")
	df := make([]_filter, 0, len(pats))
	hasInclude := false
	for _, p := range pats {
		if len(p) == 0 {
			continue
		}
		e := false
		if p[0] == '!' { // command, not a filter
			switch p[1:] {
			case "+dumpcore":
				fallthrough
			case "dumpcore":
				p = ""
				debug.SetTraceback("crash")
				setCoreLimit()
				log("Core dumping enabled.")
			case "-dumpcore":
				p = ""
				debug.SetTraceback("single")
				log("Core dumping disabled.")
			}
		} else if p[0] == '-' {
			e = true
			p = p[1:]
		} else {
			hasInclude = true
			if strings.HasPrefix(p, "\\-") {
				p = p[1:]
			}
		}
		if p != "" {
			f, err := regexp.Compile(p)
			if err == nil {
				df = append(df, _filter{f, e})
				if e {
					log("Added debug logging exclude filter: '%s'", p)
				} else {
					log("Added debug logging include filter: '%s'", p)
				}
			}
		}
	}
	if len(df) == 0 {
		df = nil
	} else if !hasInclude {
		// only exclude specified; needs an include filter to so include everything not excluded
		f, _ := regexp.Compile(".")
		df = append(df, _filter{f, false})
	}
	pMutex.Lock()
	*pFilters = df
	pMutex.Unlock()
}

func filterDebug() bool {
	if debugFilter == nil || len(debugFilter) == 0 {
		return true
	}

	_, pathname, line, ok := runtime.Caller(2)
	if !ok {
		return false
	}
	pat := fmtpkg.Sprintf("%s:%d", pathname, line)
	loggerMutex.RLock()
	df := debugFilter
	loggerMutex.RUnlock()
	for _, p := range df {
		if p.re.MatchString(pat) {
			// first match applies
			return !p.exclude
		}
	}
	return false
}

func debugFilterString() string {
	loggerMutex.RLock()
	df := debugFilter
	loggerMutex.RUnlock()
	s := ":"
	for _, f := range df {
		p := f.re.String()
		if f.exclude {
			s += "-"
		} else if p[0] == '-' {
			p = "\\" + p
		}
		s += p + ";"
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
	buf := make([]byte, 20*MiB)
	copy(buf, msg+"\n")
	n := runtime.Stack(buf[len(msg)+1:], true)
	s := string(buf[:n+len(msg)+1])
	Logf(level, s)
}

func getFileLine(caller int) string {
	pc, fname, lineno, ok := runtime.Caller(caller + 1)
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
			return fmtpkg.Sprintf(" (%s|%s:%d)", n[i:], path.Base(fname), lineno)
		} else {
			return fmtpkg.Sprintf(" (%s:%d)", path.Base(fname), lineno)
		}
	}
	return ""
}

func findCaller(l Level) string {
	pc := make([]uintptr, 32)
	n := runtime.Callers(2, pc)
	if n == 0 {
		return ""
	}
	pc = pc[:n]
	frames := runtime.CallersFrames(pc)
	nf := l.FunctionName() + "f"
	na := l.FunctionName() + "a"
	for {
		frame, more := frames.Next()
		if frame.Function != "" &&
			!strings.HasSuffix(frame.Function, ".Loga") &&
			!strings.HasSuffix(frame.Function, ".Logf") &&
			!strings.HasSuffix(frame.Function, nf) &&
			!strings.HasSuffix(frame.Function, na) &&
			!strings.Contains(frame.Function, "/logging.") {

			n := frame.Function
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
			return fmtpkg.Sprintf(" (%s|%s:%d)", n[i:], path.Base(frame.File), frame.Line)
		}
		if !more {
			break
		}
	}
	return ""
}

var NULL_LOG Logger = &nullLogImpl{}

type nullLogImpl struct{}

func (this *nullLogImpl) Loga(level Level, f func() string)                              {}
func (this *nullLogImpl) Debuga(f func() string)                                         {}
func (this *nullLogImpl) Tracea(f func() string)                                         {}
func (this *nullLogImpl) Infoa(f func() string)                                          {}
func (this *nullLogImpl) Warna(f func() string)                                          {}
func (this *nullLogImpl) Errora(f func() string)                                         {}
func (this *nullLogImpl) Severea(f func() string)                                        {}
func (this *nullLogImpl) Fatala(f func() string)                                         {}
func (this *nullLogImpl) Logf(level Level, f string, args ...interface{})                {}
func (this *nullLogImpl) Debugf(f string, args ...interface{})                           {}
func (this *nullLogImpl) Tracef(f string, args ...interface{})                           {}
func (this *nullLogImpl) Infof(f string, args ...interface{})                            {}
func (this *nullLogImpl) Warnf(f string, args ...interface{})                            {}
func (this *nullLogImpl) Errorf(f string, args ...interface{})                           {}
func (this *nullLogImpl) Severef(f string, args ...interface{})                          {}
func (this *nullLogImpl) Fatalf(f string, args ...interface{})                           {}
func (this *nullLogImpl) Stringf(level Level, format string, args ...interface{}) string { return "" }
func (this *nullLogImpl) SetLevel(Level)                                                 {}
func (this *nullLogImpl) Level() Level                                                   { return NONE }

const (
	KiB = (1 << 10)
	MiB = (1 << 20)
	GiB = (1 << 30)
)

func HumanReadableSize(sz int64, includeSource bool) string {
	var s float64
	var suffix string
	if sz >= GiB {
		s = float64(sz) / float64(GiB)
		suffix = "GiB"
	} else if sz >= MiB {
		s = float64(sz) / float64(MiB)
		suffix = "MiB"
	} else if sz >= KiB {
		s = float64(sz) / float64(KiB)
		suffix = "KiB"
	} else if includeSource {
		return fmtpkg.Sprintf("%v", sz)
	} else if sz == 1 {
		return "1 byte"
	} else {
		return fmtpkg.Sprintf("%v bytes", sz)
	}
	num := fmtpkg.Sprintf("%.3f", s)
	num = strings.TrimSuffix(strings.TrimSuffix(num, "0"), "0")
	if includeSource {
		return fmtpkg.Sprintf("%v (%s %s)", sz, num, suffix)
	} else {
		return fmtpkg.Sprintf("%s %s", num, suffix)
	}
}

// Wraps the log package default logger so logging is consistent
var _wrapper = &wrapper{}

type wrapper struct {
}

func (this *wrapper) Write(p []byte) (n int, err error) {
	Infof(string(p))
	return len(p), nil
}

func init() {
	log.Default().SetOutput(_wrapper)
	log.Default().SetPrefix("")
	log.Default().SetFlags(0)
}
