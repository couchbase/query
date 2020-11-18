//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

Package err provides user-visible errors and warnings. These errors
include error codes and will eventually provide multi-language
messages.

*/
package errors

import (
	"encoding/json"
	"fmt"
	"path"
	"runtime"
	"strings"
)

const (
	EXCEPTION = iota
	WARNING
	NOTICE
	INFO
	LOG
	DEBUG
)

type Errors []Error

// Error will eventually include code, message key, and internal error
// object (cause) and message
type Error interface {
	error
	Code() int32
	TranslationKey() string
	GetICause() error
	Level() int
	IsFatal() bool
	IsWarning() bool
	OnceOnly() bool
	Object() map[string]interface{}
	Retry() bool
	Cause() interface{}
	SetCause(cause interface{})
}

type ErrorChannel chan Error

func NewError(e error, internalMsg string) Error {
	switch e := e.(type) {
	case Error: // if given error is already an Error, just return it:
		return e
	default:
		return &err{level: EXCEPTION, ICode: 5000, IKey: "Internal Error", ICause: e,
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
	ICode          int32
	IKey           string
	ICause         error
	InternalMsg    string
	InternalCaller string
	level          int
	onceOnly       bool
	retry          bool // Retrying this query might be useful.
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
	}
}

func (e *err) Object() map[string]interface{} {
	m := map[string]interface{}{
		"code":    e.ICode,
		"key":     e.IKey,
		"message": e.InternalMsg,
	}
	if e.ICause != nil {
		m["icause"] = e.ICause.Error()
	}
	if e.retry {
		m["retry"] = true
	}
	if e.cause != nil {
		m["cause"] = e.cause
	}
	return m
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
		Caller  string      `json:"caller"`
		Code    int32       `json:"code"`
		ICause  string      `json:"icasue"`
		Key     string      `json:"key"`
		Message string      `json:"message"`
		Retry   bool        `json:"retry"`
		Cause   interface{} `json:"cause"`
	}

	unmarshalErr := json.Unmarshal(body, &_unmarshalled)
	if unmarshalErr != nil {
		return unmarshalErr
	}

	e.ICode = _unmarshalled.Code
	e.IKey = _unmarshalled.Key
	e.InternalMsg = _unmarshalled.Message
	e.InternalCaller = _unmarshalled.Caller
	e.retry = _unmarshalled.Retry
	e.cause = _unmarshalled.Cause
	if _unmarshalled.ICause != "" {
		e.ICause = fmt.Errorf("%v", _unmarshalled.ICause)
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

func (e *err) Code() int32 {
	return e.ICode
}

func (e *err) TranslationKey() string {
	return e.IKey
}

func (e *err) GetICause() error {
	return e.ICause
}

func (e *err) OnceOnly() bool {
	return e.onceOnly
}

func (e *err) Retry() bool {
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
	return &err{level: EXCEPTION, ICode: 9999, IKey: "not_implemented", InternalMsg: fmt.Sprintf("Not yet implemented: %v", feature), InternalCaller: CallerN(1)}
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
