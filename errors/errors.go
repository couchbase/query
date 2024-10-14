//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

/*
Package err provides user-visible errors and warnings. These errors
include error codes and will eventually provide multi-language
messages.
*/
package errors

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

const (
	EXCEPTION = iota
	ERROR
	WARNING
	NOTICE
	INFO
	LOG
	DEBUG
)

var levelNames = map[int]string{
	EXCEPTION: "exception",
	ERROR:     "error",
	WARNING:   "warning",
	NOTICE:    "notice",
	INFO:      "info",
	LOG:       "log",
	DEBUG:     "debug",
}

type ErrorCode int32

type Errors []Error

type Tristate int

const (
	NONE Tristate = iota
	FALSE
	TRUE
)

func ToBool(t Tristate) bool {
	return t == TRUE
}

const (
	DEFAULT_REQUEST_ERROR_LIMIT = 16
)

// Error will eventually include code, message key, and internal error
// object (cause) and message
type Error interface {
	error
	Code() ErrorCode
	TranslationKey() string
	SetTranslationKey(s string)
	GetICause() error
	Level() int
	LevelString() string
	IsFatal() bool
	IsWarning() bool
	OnceOnly() bool
	Object() map[string]interface{}
	Retry() Tristate
	Cause() interface{}
	SetCause(cause interface{})
	ContainsText(text string) bool
	HasCause(ErrorCode) bool
	HasICause(ErrorCode) bool
	ExtractLineAndColumn(map[string]interface{})
	AddErrorContext(ctx string)
	Repeat()
	GetRepeats() int
	GetErrorCause() Error
}

type AbortError struct {
	e string
}

// dummy method to make AbortError and error not equivalent
func (e *AbortError) Error() string {
	return e.e
}

// to make abort error non equivalent to any other error
func (e *AbortError) dummyMethod() {
}

func (e *AbortError) MarshalText() ([]byte, error) {
	return []byte(e.e), nil
}

func NewAbortError(e string) *AbortError {
	return &AbortError{e}
}

type ErrorChannel chan Error

var gsiPatterns map[string]*regexp.Regexp

func init() {
	gsiPatterns = make(map[string]*regexp.Regexp)
	gsiPatterns["enterprise"] = regexp.MustCompile("(.*) not supported in non-Enterprise Edition")
	gsiPatterns["exist"] = regexp.MustCompile("Index(.*) already exist")
	gsiPatterns["reason"] = regexp.MustCompile("(.*)( Reason: | Error=| Error: )(.*)")
	gsiPatterns["tempfile"] = regexp.MustCompile("(.*) temp file size exceeded limit ([0-9]+), ([0-9]+)")
}

func NewError(e error, internalMsg string) Error {
	switch e := e.(type) {
	case Error: // if given error is already an Error, just return it:
		return e
	}

	var cause interface{}

	// Provide meaningful additional information for where possible for GSI errors
	isGsi, caller := GSICaller()
	if isGsi {
		errText := ""
		if e != nil {
			errText = strings.TrimSpace(e.Error())
			if errText == internalMsg {
				e = nil
			}
		} else {
			errText = strings.TrimSpace(internalMsg)
		}
		if res := gsiPatterns["enterprise"].FindSubmatch([]byte(errText)); res != nil {
			cause = NewEnterpriseFeature(string(res[1]), "indexing.enterprise_only_feature")
		} else if res = gsiPatterns["exist"].FindSubmatch([]byte(errText)); res != nil {
			cause = NewIndexAlreadyExistsError(string(res[1]))
		} else {
			code := E_GSI
			key := "indexing.error"
			level := EXCEPTION
			m := make(map[string]interface{}, 5)
			if strings.HasPrefix(internalMsg, "GSI ") {
				m["source"] = internalMsg[4:]
			}
			if strings.Contains(errText, "Encountered transient error") {
				m["error"] = errText
				code = W_GSI_TRANSIENT
				key = "indexing.transient_error"
				level = WARNING
			} else if res = gsiPatterns["tempfile"].FindSubmatch([]byte(errText)); res != nil {
				m["request"] = strings.TrimSpace(string(res[1]))
				m["limit"], _ = strconv.Atoi(string(res[2]))
				m["size"], _ = strconv.Atoi(string(res[3]))
				m["user_action"] = "Check queryTmpSpaceDir and queryTmpSpaceSize settings."
				code = E_GSI_TEMP_FILE_SIZE
				key = "indexing.temp_file_size"
			} else {
				if res = gsiPatterns["reason"].FindSubmatch([]byte(errText)); res != nil {
					m["error"] = strings.TrimSpace(string(res[1]))
					m["reason"] = strings.TrimSpace(string(res[3]))
				} else {
					m["error"] = errText
				}
			}
			cause = &err{level: level, ICode: code, IKey: key, InternalMsg: "GSI error", InternalCaller: caller, cause: m}
		}
	}

	return &err{level: EXCEPTION, ICode: E_INTERNAL, IKey: "Internal Error", ICause: e, InternalMsg: internalMsg,
		InternalCaller: caller, cause: cause}
}

func NewWarning(internalMsg string) Error {
	return &err{level: WARNING, ICode: W_GENERIC, InternalMsg: internalMsg, InternalCaller: CallerN(1)}
}

func NewNotice(internalMsg string) Error {
	return &err{level: NOTICE, InternalMsg: internalMsg, InternalCaller: CallerN(1)}
}

func NewInfo(internalMsg string) Error {
	return &err{level: INFO, InternalMsg: internalMsg, InternalCaller: CallerN(1)}
}

func NewLog(internalMsg string) Error {
	return &err{level: LOG, InternalMsg: internalMsg, InternalCaller: CallerN(1)}
}

func NewDebug(internalMsg string) Error {
	return &err{level: DEBUG, InternalMsg: internalMsg, InternalCaller: CallerN(1)}
}

func NewErrors(es []error, internalMsg string) (errs Errors) {
	for _, e := range es {
		errs = append(errs, NewError(e, internalMsg))
	}
	return errs
}

type err struct {
	ICode          ErrorCode
	IKey           string
	ICause         error
	InternalMsg    string
	InternalCaller string
	level          int
	onceOnly       bool
	retry          Tristate // Retrying this query might be useful.
	cause          interface{}
	repeats        int
}

func (e *err) Error() string {
	switch {
	default:
		return "Unspecified error."
	case e.InternalMsg != "" && e.ICause != nil:
		return e.InternalMsg + " - cause: " + e.ICause.Error()
	case e.InternalMsg != "":
		return e.InternalMsg
	case e.ICause != nil:
		return e.ICause.Error()
	case e.cause != nil: // only as a last resort if InternalMsg & ICause aren't set
		return fmt.Sprintf("%v", e.cause)
	}
}

func (e *err) Object() map[string]interface{} {
	m := map[string]interface{}{
		// only use standard data types in the object
		"code":    int32(e.ICode),
		"key":     e.IKey,
		"message": e.InternalMsg,
		"caller":  e.InternalCaller,
		"_level":  e.LevelString(),
	}
	if e.ICause != nil {
		m["icause"] = e.ICause.Error()
	}
	if e.retry != NONE {
		m["retry"] = ToBool(e.retry)
	}
	if e.cause != nil {
		// ensure m["cause"] contains only basic types
		m["cause"] = processValue(e.cause)
	}
	e.ExtractLineAndColumn(m)
	if e.repeats > 0 {
		m["repeats"] = e.repeats
	}
	return m
}

func processValue(v interface{}) interface{} {
	switch vt := v.(type) {
	case map[string]interface{}:
		return processMap(vt)
	case interface{ Object() map[string]interface{} }:
		return vt.Object()
	case interface{ Error() string }:
		return vt.Error()
	case interface{ String() string }:
		return vt.String()
	case *AbortError:
		return vt.e
	case []interface{}:
		return vt
	case []string:
		return vt
	case uint64:
		return vt
	case int64:
		return vt
	case int:
		return vt
	case float64:
		return vt
	case string:
		return vt
	case nil:
		return vt
	case bool:
		return vt
	default:
		return fmt.Sprintf("%v", v)
	}
}

func processMap(m map[string]interface{}) map[string]interface{} {
	rv := make(map[string]interface{})
	for k, v := range m {
		rv[k] = processValue(v)
	}
	return rv
}

func (e *err) MarshalJSON() ([]byte, error) {
	m := e.Object()
	if e.InternalCaller != "" &&
		!strings.HasPrefix("e.InternalCaller", "unknown:") {
		m["caller"] = e.InternalCaller
	}
	var bb bytes.Buffer
	enc := json.NewEncoder(&bb)
	enc.SetEscapeHTML(false)
	err := enc.Encode(m)
	if err != nil {
		return nil, err
	}
	return bb.Bytes(), nil
}

func (e *err) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Caller  string      `json:"caller"`
		Code    int32       `json:"code"`
		ICause  string      `json:"icause"`
		Key     string      `json:"key"`
		Message string      `json:"message"`
		Retry   Tristate    `json:"retry"`
		Cause   interface{} `json:"cause"`
		Repeats int         `json:"repeats"`
		Level   string      `json:"_level"`
	}

	unmarshalErr := json.Unmarshal(body, &_unmarshalled)
	if unmarshalErr != nil {
		return unmarshalErr
	}

	for k, v := range levelNames {
		if v == _unmarshalled.Level {
			e.level = k
		}
	}
	e.ICode = ErrorCode(_unmarshalled.Code)
	e.IKey = _unmarshalled.Key
	e.InternalMsg = _unmarshalled.Message
	e.InternalCaller = _unmarshalled.Caller
	e.retry = _unmarshalled.Retry
	e.cause = _unmarshalled.Cause
	e.repeats = _unmarshalled.Repeats
	if _unmarshalled.ICause != "" {
		e.ICause = errors.New(_unmarshalled.ICause)
	}
	return nil
}

func (e *err) Level() int {
	return e.level
}

func (e *err) LevelString() string {
	s, ok := levelNames[e.level]
	if !ok {
		s = levelNames[EXCEPTION]
	}
	return s
}

func (e *err) IsFatal() bool {
	return e.level == EXCEPTION
}

func (e *err) IsWarning() bool {
	return e.level == WARNING
}

func (e *err) Code() ErrorCode {
	return e.ICode
}

func (e *err) TranslationKey() string {
	return e.IKey
}

func (e *err) SetTranslationKey(s string) {
	e.IKey = s
}

func (e *err) GetICause() error {
	return e.ICause
}

func (e *err) OnceOnly() bool {
	return e.onceOnly
}

func (e *err) Retry() Tristate {
	return e.retry
}

func (e *err) Cause() interface{} {
	if e.cause == nil {
		return nil
	}

	switch t := e.cause.(type) {
	case Error:
		return t
	case error: // prevent the cause from being marshalled to an empty object
		return t.Error()
	}
	return e.cause
}

func (e *err) SetCause(cause interface{}) {
	e.cause = cause
}

var extractRe = regexp.MustCompile("line ([0-9]+), column ([0-9]+)")

func (e *err) ExtractLineAndColumn(m map[string]interface{}) {
	if m == nil {
		return
	}
	err := e.Error()
	if matches := extractRe.FindStringSubmatch(err); matches != nil {
		m["line"], _ = strconv.Atoi(matches[1])
		m["column"], _ = strconv.Atoi(matches[2])
	}
}

func (e *err) AddErrorContext(ctx string) {
	if ctx == "" {
		return
	}
	err := e.Error()
	if extractRe.FindStringSubmatch(err) == nil {
		e.InternalMsg += ctx
	}
}

func (e *err) Repeat() {
	e.repeats++
}

func (e *err) GetRepeats() int {
	return e.repeats
}

// only put errors in the reserved range here (7000-9999)
func NewNotImplemented(feature string) Error {
	return &err{level: EXCEPTION, ICode: 9999, IKey: "not_implemented", InternalMsg: fmt.Sprintf("Not available: %v", feature),
		InternalCaller: CallerN(1)}
}

// Returns "FileName:LineNum" of caller.
func Caller() string {
	return CallerN(1)
}

// Returns "FileName:LineNum" of the Nth caller on the call stack,
// where level of 0 is the caller of CallerN.
func CallerN(level int) string {
	_, file, line, ok := runtime.Caller(1 + level)
	if !ok {
		return "unknown:0"
	}
	return fmt.Sprintf("%s:%d", strings.Split(path.Base(file), ".")[0], line)
}

func GSICaller() (bool, string) {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return false, "unknown:0"
	}
	return strings.Index(file, "/indexing/secondary/") != -1, fmt.Sprintf("%s:%d", strings.Split(path.Base(file), ".")[0], line)
}

// In the future we should be able to check error codes or keys rather than matching error text, or even base it on error type but
// for now we can only check the text
func IsExistsError(object string, e error) bool {
	re := regexp.MustCompile(object + ".*already exist")
	return re.Match([]byte(e.Error()))
}

func IsNotFoundError(object string, e error) bool {
	re := regexp.MustCompile(object + ".*not found")
	return re.Match([]byte(e.Error()))
}

func IsIndexExistsError(e error) bool {
	if err, ok := e.(Error); ok && err.HasCause(E_INDEX_ALREADY_EXISTS) {
		return true
	}
	return false
}

func IsScopeExistsError(e error) bool {
	return IsExistsError("Scope", e)
}

func IsCollectionExistsError(e error) bool {
	return IsExistsError("Collection", e)
}

func IsSequenceExistsError(e error) bool {
	if err, ok := e.(Error); ok && err.Code() == E_SEQUENCE_ALREADY_EXISTS {
		return true
	}
	return IsExistsError("Sequence", e)
}

func IsIndexNotFoundError(e error) bool {
	if err, ok := e.(Error); ok && (err.Code() == E_INDEX_NOT_FOUND || err.Code() == E_CB_INDEX_NOT_FOUND) {
		return true
	}
	return IsNotFoundError("Index", e)
}

func IsScopeNotFoundError(e error) bool {
	if err, ok := e.(Error); ok && err.Code() == E_CB_SCOPE_NOT_FOUND {
		return true
	}
	return IsNotFoundError("Scope", e)
}

func IsCollectionNotFoundError(e error) bool {
	return IsNotFoundError("Collection", e)
}

func IsSequenceNotFoundError(e error) bool {
	if err, ok := e.(Error); ok && err.Code() == E_SEQUENCE_NOT_FOUND {
		return true
	}
	return IsNotFoundError("Sequence", e)
}

// search initial error text and all cause nesting levels for the given string
func (e *err) ContainsText(text string) bool {
	s := e.Error()
	if strings.Contains(s, text) {
		return true
	}
	// search causes
	eo := e.Object()
	for {
		if cause, ok := eo["cause"]; ok {
			s = fmt.Sprintf("%v", cause)
			if strings.Contains(s, text) {
				return true
			}
			if cwo, ok := cause.(interface{ Object() map[string]interface{} }); ok {
				eo = cwo.Object()
			} else {
				return false
			}
		} else {
			return false
		}
	}
}

func (e *err) HasCause(code ErrorCode) bool {
	if e.Code() == code {
		return true
	}
	c := e.Cause()
	for c != nil {
		switch cse := c.(type) {
		case Error:
			if cse.Code() == code {
				return true
			}
			c = cse.Cause()
		case map[string]interface{}:
			cde, ok := cse["code"]
			if ok {
				switch cde := cde.(type) {
				case int32:
					if cde == int32(code) {
						return true
					}
				case ErrorCode:
					if cde == code {
						return true
					}
				}
			}
			c, _ = cse["cause"]
		default:
			c = nil
		}
	}
	return false
}

func (e *err) HasICause(code ErrorCode) bool {
	c := e.ICause

	for c != nil {
		switch icse := c.(type) {
		case Error:
			if icse.Code() == code {
				return true
			}
			c = icse.GetICause()
		default:
			c = nil
		}
	}

	return false
}

func NewTempFileQuotaExceededError() Error {
	return &err{level: EXCEPTION, ICode: E_TEMP_FILE_QUOTA, IKey: "quota.temp_file.exceeded", InternalCaller: CallerN(1),
		InternalMsg: "Temporary file quota exceeded"}
}

func getErrorForCause(e error) interface{} {
	switch e := e.(type) {
	case Error:
		return e
	case interface{ MarshalJSON() ([]byte, error) }:
		return e
	case error:
		s := e.Error()
		var i interface{}
		if json.Unmarshal([]byte(s), &i) == nil {
			return i
		}
		return s
	default:
		return e
	}
}

func FromObject(m map[string]interface{}) Error {
	b, e := json.Marshal(m)
	if e != nil {
		return nil
	}
	return FromBytes(b)
}

func FromBytes(b []byte) Error {
	rv := &err{}
	if e := rv.UnmarshalJSON(b); e != nil {
		return nil
	}
	return rv
}

func (this *err) GetErrorCause() Error {
	if c, ok := this.cause.(map[string]interface{}); ok {
		if _, ok := c["code"]; ok {
			rv := FromObject(c)
			if rv != nil {
				return rv
			}
		}
		if e, ok := c["cause"]; ok {
			switch t := e.(type) {
			case map[string]interface{}:
				return FromObject(t)
			case string:
				return FromBytes([]byte(t))
			case []byte:
				return FromBytes(t)
			}
		}
		if e, ok := c["error"]; ok {
			switch t := e.(type) {
			case map[string]interface{}:
				return FromObject(t)
			case string:
				return FromBytes([]byte(t))
			case []byte:
				return FromBytes(t)
			}
		}
	}
	return nil
}
