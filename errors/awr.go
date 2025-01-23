//  Copyright 2024-Present Couchbase, Inc.
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

var _awr = map[ErrorCode][2]string{
	E_AWR_START:   {"start", "Failed to start workload reporting"},
	E_AWR_CONFIG:  {"config", "Error processing workload configuration"},
	E_AWR_DISTRIB: {"distribution", "Error distributing workload settings"},
}

func getAWRSchemaHelp(setting string) string {
	var help string
	switch setting {
	case "threshold":
		help = "A valid duration string. The duration must be at least 0 seconds. (e.g. \"1m30s\")"
	case "interval":
		help = "A valid duration string. The duration must be at least 1 minute. (e.g. \"1m30s\")"
	case "queue_len", "num_statements":
		help = "A positive integer."
	case "enabled":
		help = "A boolean value."
	case "location":
		help = "A string representating a syntactically valid path to a bucket or collection." +
			" The only permitted namespace is \"default\". (e.g. \"default:bucket1.scope1.collection1\")"
	default:
		return ""
	}

	return fmt.Sprintf("Setting '%s' must be: %s", setting, help)
}

func NewAWRError(code ErrorCode, args ...interface{}) Error {
	e := &err{level: EXCEPTION, ICode: code, InternalCaller: CallerN(1),
		IKey: "service.workload_report." + _awr[code][0], InternalMsg: _awr[code][1]}
	var fmtArgs []interface{}
	for _, a := range args {
		switch a := a.(type) {
		case string:
			fmtArgs = append(fmtArgs, a)
		case Error:
			e.cause = a
		case error:
			e.cause = a.Error()
		default:
			panic("invalid arguments to NewAWRError")
		}
	}
	if len(fmtArgs) > 0 {
		e.InternalMsg = fmt.Sprintf(e.InternalMsg, fmtArgs...)
	}
	return e
}

func NewAWRInvalidSettingError(setting string, value interface{}, cause error) Error {
	c := make(map[string]interface{}, 2)
	if help := getAWRSchemaHelp(setting); help != "" {
		c["help"] = help
	}

	if cause != nil {
		c["cause"] = cause.Error()
	}

	return &err{level: EXCEPTION, ICode: E_AWR_SETTING, InternalCaller: CallerN(1), IKey: "service.workload_report.setting",
		InternalMsg: fmt.Sprintf(" Invalid value '%v' for workload setting '%s'.", value, setting), cause: c}
}
