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
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
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
	NODEQUOTAVALPERCENT:   checkPercent,
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
}

func checkBool(val interface{}) (bool, errors.Error) {
	_, ok := val.(bool)
	return ok, nil
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
			// valid units are "ns", "us", "ms", "s", "m", "h"
			lc := val[len(val)-1]
			if lc == 's' || lc == 'm' || lc == 'h' {
				_, e := time.ParseDuration(val)
				return e == nil, nil
			}
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
