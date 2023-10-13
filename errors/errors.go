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
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"runtime"
	"strings"

	"github.com/couchbase/query/value"
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

type ErrorCode int32

type Errors []Error

// Error will eventually include code, message key, and internal error
// object (cause) and message
type Error interface {
	error
	Code() ErrorCode
	TranslationKey() string
	SetTranslationKey(s string)
	GetICause() error
	Level() int
	IsFatal() bool
	IsWarning() bool
	OnceOnly() bool
	Object() map[string]interface{}
	Retry() value.Tristate
	Cause() interface{}
	HasCause(ErrorCode) bool
	HasICause(ErrorCode) bool
	SetCause(cause interface{})
	ContainsText(text string) bool
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

func NewError(e error, internalMsg string) Error {
	switch e := e.(type) {
	case Error: // if given error is already an Error, just return it:
		return e
	default:
		return &err{level: EXCEPTION, ICode: E_INTERNAL, IKey: "Internal Error", ICause: e,
			InternalMsg: internalMsg, InternalCaller: CallerN(1)}
	}
}

func NewWarning(internalMsg string) Error {
	return &err{level: WARNING, InternalMsg: internalMsg, InternalCaller: CallerN(1)}
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
	retry          value.Tristate // Retrying this query might be useful.
	cause          interface{}
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
	}
	if e.ICause != nil {
		m["icause"] = e.ICause.Error()
	}
	if e.retry != value.NONE {
		m["retry"] = value.ToBool(e.retry)
	}
	if e.cause != nil {
		// ensure m["cause"] contains only basic types
		m["cause"] = processValue(e.cause)
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
	return json.Marshal(m)
}

func (e *err) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Caller  string         `json:"caller"`
		Code    int32          `json:"code"`
		ICause  string         `json:"icause"`
		Key     string         `json:"key"`
		Message string         `json:"message"`
		Retry   value.Tristate `json:"retry"`
		Cause   interface{}    `json:"cause"`
	}

	unmarshalErr := json.Unmarshal(body, &_unmarshalled)
	if unmarshalErr != nil {
		return unmarshalErr
	}

	e.ICode = ErrorCode(_unmarshalled.Code)
	e.IKey = _unmarshalled.Key
	e.InternalMsg = _unmarshalled.Message
	e.InternalCaller = _unmarshalled.Caller
	e.retry = _unmarshalled.Retry
	e.cause = _unmarshalled.Cause
	if _unmarshalled.ICause != "" {
		e.ICause = errors.New(_unmarshalled.ICause)
	}
	return nil
}

func (e *err) Level() int {
	return e.level
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

func (e *err) Retry() value.Tristate {
	return e.retry
}

func (e *err) Cause() interface{} {
	return e.cause
}

func (e *err) SetCause(cause interface{}) {
	e.cause = cause
}

// only put errors in the reserved range here (7000-9999)
func NewNotImplemented(feature string) Error {
	return &err{level: EXCEPTION, ICode: 9999, IKey: "not_implemented", InternalMsg: fmt.Sprintf("Not available: %v", feature), InternalCaller: CallerN(1)}
}

// Returns "FileName:LineNum" of caller.
func Caller() string {
	return CallerN(1)
}

// Returns "FileName:LineNum" of the Nth caller on the call stack,
// where level of 0 is the caller of CallerN.
func CallerN(level int) string {
	_, fname, lineno, ok := runtime.Caller(1 + level)
	if !ok {
		return "unknown:0"
	}
	return fmt.Sprintf("%s:%d",
		strings.Split(path.Base(fname), ".")[0], lineno)
}

// In the future we should be able to check error codes or keys rather than matching error text, or even base it on error type but
// for now we can only check the text
func IsExistsError(object string, e error) bool {
	re := regexp.MustCompile(object + ".*already exists")
	return re.Match([]byte(e.Error()))
}

func IsNotFoundError(object string, e error) bool {
	re := regexp.MustCompile(object + ".*not found")
	return re.Match([]byte(e.Error()))
}

func IsIndexExistsError(e error) bool {
	return IsExistsError("Index", e)
}

func IsScopeExistsError(e error) bool {
	return IsExistsError("Scope", e)
}

func IsCollectionExistsError(e error) bool {
	return IsExistsError("Collection", e)
}

func IsIndexNotFoundError(e error) bool {
	return IsNotFoundError("Index", e)
}

func IsScopeNotFoundError(e error) bool {
	return IsNotFoundError("Scope", e)
}

func IsCollectionNotFoundError(e error) bool {
	return IsNotFoundError("Collection", e)
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
