//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package server

import (
	"strconv"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

const (
	CPUPROFILE            = "cpuprofile"
	DEBUG                 = "debug"
	KEEPALIVELENGTH       = "keep-alive-length"
	LOGLEVEL              = "loglevel"
	MAXPARALLELISM        = "max-parallelism"
	MEMPROFILE            = "memprofile"
	REQUESTSIZECAP        = "request-size-cap"
	PIPELINEBATCH         = "pipeline-batch"
	PIPELINECAP           = "pipeline-cap"
	SCANCAP               = "scan-cap"
	SERVICERS             = "servicers"
	PLUSSERVICERS         = "plus-servicers"
	TIMEOUTSETTING        = "timeout"
	CMPOBJECT             = "completed"
	CMPTHRESHOLD          = "completed-threshold"
	CMPLIMIT              = "completed-limit"
	CMPPUSH               = "+completed-limit"
	CMPPOP                = "-completed-limit"
	CMPMAXPLANSIZE        = "completed-max-plan-size"
	CMPSTREAM             = "completed-stream-size"
	PRPLIMIT              = "prepared-limit"
	PRETTY                = "pretty"
	MAXINDEXAPI           = "max-index-api"
	PROFILE               = "profile"
	CONTROLS              = "controls"
	N1QLFEATCTRL          = "n1ql-feat-ctrl"
	AUTOPREPARE           = "auto-prepare"
	MUTEXPROFILE          = "mutexprofile"
	FUNCLIMIT             = "functions-limit"
	TASKLIMIT             = "tasks-limit"
	MEMORYQUOTA           = "memory-quota"
	NODEQUOTA             = "node-quota"
	NODEQUOTAVALPERCENT   = "node-quota-val-percent"
	USECBO                = "use-cbo"
	TXTIMEOUT             = "txtimeout"
	ATRCOLLECTION         = "atrcollection"
	NUMATRS               = "numatrs"
	CLEANUPWINDOW         = "cleanupwindow"
	CLEANUPCLIENTATTEMPTS = "cleanupclientattempts"
	CLEANUPLOSTATTEMPTS   = "cleanuplostattempts"
	GCPERCENT             = "gc-percent"
	REQUESTERRORLIMIT     = "request-error-limit"
	QUERY_TMP_DIR         = "query_tmpspace_dir"
	QUERY_TMP_LIMIT       = "query_tmpspace_limit"
	USEREPLICA            = "use-replica"
	NUM_CPUS              = "num-cpus"
	DURATIONSTYLE         = "duration-style"
	AWR                   = "activity-workload-reporting"
)

type Checker func(interface{}) (bool, errors.Error)

var CHECKERS = map[string]Checker{
	CPUPROFILE:            checkString,
	DEBUG:                 checkBool,
	LOGLEVEL:              checkLogLevel,
	MAXPARALLELISM:        checkNumber,
	MEMPROFILE:            checkString,
	REQUESTSIZECAP:        checkNumber,
	PIPELINEBATCH:         checkNumber,
	PIPELINECAP:           checkNumber,
	SCANCAP:               checkNumber,
	TIMEOUTSETTING:        checkNumber,
	CMPOBJECT:             checkCompleted,
	CMPTHRESHOLD:          checkNumber,
	CMPLIMIT:              checkNumber,
	CMPMAXPLANSIZE:        checkNumber,
	CMPSTREAM:             checkNumber,
	PRETTY:                checkBool,
	MAXINDEXAPI:           checkNumber,
	PROFILE:               checkProfileAdmin,
	CONTROLS:              checkControlsAdmin,
	N1QLFEATCTRL:          checkHexNumber,
	AUTOPREPARE:           checkBool,
	MUTEXPROFILE:          checkBool,
	USECBO:                checkBool,
	TXTIMEOUT:             checkDuration,
	ATRCOLLECTION:         checkPath,
	CLEANUPWINDOW:         checkDuration,
	CLEANUPCLIENTATTEMPTS: checkBool,
	CLEANUPLOSTATTEMPTS:   checkBool,
	GCPERCENT:             checkNumber,
	REQUESTERRORLIMIT:     checkNumber,
	QUERY_TMP_DIR:         checkString,
	QUERY_TMP_LIMIT:       checkNumber,
	USEREPLICA:            checkTristateString,
	NODEQUOTAVALPERCENT:   checkPercent,
	DURATIONSTYLE:         checkDurationStyle,
	AWR:                   checkAWR,
}

var CHECKERS_MIN = map[string]int{
	KEEPALIVELENGTH: KEEP_ALIVE_MIN,
	CMPPUSH:         2,
	CMPPOP:          2,
	SERVICERS:       0,
	PLUSSERVICERS:   0,
	PRPLIMIT:        2,
	FUNCLIMIT:       2,
	TASKLIMIT:       2,
	MEMORYQUOTA:     0,
	NODEQUOTA:       0,
	NUMATRS:         2,
	NUM_CPUS:        0,
}

func checkBool(val interface{}) (bool, errors.Error) {
	_, ok := val.(bool)
	return ok, nil
}

func checkTristateString(val interface{}) (bool, errors.Error) {
	t, ok := val.(string)

	if ok {
		_, ok1 := value.ParseTristateString(t)

		return ok1, nil
	}

	return false, nil
}

func checkNumber(val interface{}) (bool, errors.Error) {
	switch val.(type) {
	case int64:
		return true, nil
	case float64:
		return true, nil
	}
	return false, nil
}

func checkHexNumber(val interface{}) (bool, errors.Error) {
	switch v := val.(type) {
	case int64:
		return true, nil
	case string:
		if _, err := strconv.ParseInt(v, 0, 64); err == nil {
			return true, nil
		}
	}
	return false, nil
}

func checkNumberMin(val interface{}, min int) (bool, errors.Error) {
	switch val := val.(type) {
	case int64:
		return val >= int64(min), nil
	case float64:
		return val >= float64(min), nil
	}
	return false, nil
}

func checkObject(val interface{}) (bool, errors.Error) {
	_, ok := val.(map[string]interface{})
	return ok, nil
}

func checkCompleted(val interface{}) (bool, errors.Error) {
	var tag string

	object, ok := val.(map[string]interface{})
	if !ok {
		return ok, errors.NewAdminSettingTypeError("completed", object)
	}

	if tagVal, ok := object["tag"]; ok {
		tag, ok = object["tag"].(string)
		if !ok {
			return ok, errors.NewAdminSettingTypeError("tag", tagVal)
		}
	}
	for n, v := range object {
		var op RequestsOp

		if n == "tag" {
			continue
		}

		switch n[0] {
		case '+':
			op = CMP_OP_ADD
			n = n[1:]
		case '-':
			op = CMP_OP_DEL
			n = n[1:]
		default:
			op = CMP_OP_UPD
		}
		err := RequestsCheckQualifier(n, op, v, tag)
		if err != nil && op == CMP_OP_UPD && err.Code() == errors.E_COMPLETED_QUALIFIER_NOT_FOUND {
			err = RequestsCheckQualifier(n, CMP_OP_ADD, v, tag)
		}
		if err != nil {
			return false, err
		}
	}
	return ok, nil
}

func checkString(val interface{}) (bool, errors.Error) {
	_, ok := val.(string)
	return ok, nil
}

func checkDuration(val interface{}) (bool, errors.Error) {
	switch val := val.(type) {
	case string:
		if val != "" {
			_, e := util.ParseDurationStyle(val, util.DEFAULT)
			return e == nil, nil
		}
	}
	return false, nil
}

func checkLogLevel(val interface{}) (bool, errors.Error) {
	level, is_string := val.(string)
	if !is_string {
		return false, nil
	}
	_, ok, _ := logging.ParseLevel(level)
	return ok, nil
}

func checkPath(val interface{}) (bool, errors.Error) {
	s, ok := val.(string)
	if ok && s != "" {
		if _, err := algebra.NewVariablePathWithContext(s, "default", ""); err != nil {
			return false, errors.NewAdminSettingTypeError("atrcollection", val)
		}
	}

	return ok, nil
}

func checkPercent(val interface{}) (bool, errors.Error) {
	switch val := val.(type) {
	case int64:
		if val >= 0 && val <= 100 {
			return true, nil
		}
	case float64:
		if val >= 0.0 && val <= 100 {
			return true, nil
		}
	}
	return false, nil
}

func checkDurationStyle(val interface{}) (bool, errors.Error) {
	style, is_string := val.(string)
	if !is_string {
		return false, nil
	}
	s, ok := util.IsDurationStyle(style)
	// permit only styles the UI can parse
	if ok && s != util.LEGACY && s != util.COMPATIBLE {
		ok = false
	}
	return ok, nil
}

func checkAWR(val interface{}) (bool, errors.Error) {
	object, ok := val.(map[string]interface{})
	if !ok {
		return false, errors.NewAdminSettingTypeError(AWR, object)
	}
	err := AwrCB.SetConfig(object, true)
	if err != nil {
		return false, err
	}
	return true, nil
}
