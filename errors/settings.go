//  Copyright 2025-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package errors

import (
	"fmt"
)

func NewSettingsError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SETTINGS, IKey: "settings_error",
		ICause: e, InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewSettingsMetaKVError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SETTINGS_METAKV, IKey: "settings_metakv_error",
		ICause: e, InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewSettingsInvalidType(settings string, actual interface{}) Error {
	return &err{level: EXCEPTION, ICode: E_SETTINGS_INVALID_TYPE, IKey: "settings_invalid_type",
		InternalMsg:    fmt.Sprintf("Invalid type (%T) specified for %s", actual, settings),
		InternalCaller: CallerN(1)}
}

func NewSettingsInvalidValue(setting, expected string, value interface{}) Error {
	var msg, tmsg string
	if value != nil {
		tmsg = fmt.Sprintf("(%v) of type %T", value, value)
	}
	if expected != "" {
		msg = fmt.Sprintf("Invalid value %s specified for setting %s, %s expected", tmsg, setting, expected)
	} else {
		msg = fmt.Sprintf("Invalid value %s specified for setting %s", tmsg, setting)
	}
	return &err{level: EXCEPTION, ICode: E_SETTINGS_INVALID_VALUE, IKey: "settings_invalid_value",
		InternalMsg:    msg,
		InternalCaller: CallerN(1)}
}
