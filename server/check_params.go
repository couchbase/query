//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package server

import (
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
	USECBO                = "use-cbo"
	TXTIMEOUT             = "txtimeout"
	ATRCOLLECTION         = "atrcollection"
	NUMATRS               = "numatrs"
	CLEANUPWINDOW         = "cleanupwindow"
	CLEANUPCLIENTATTEMPTS = "cleanupclientattempts"
	CLEANUPLOSTATTEMPTS   = "cleanuplostattempts"
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
	N1QLFEATCTRL:          checkNumber,
	AUTOPREPARE:           checkBool,
	MUTEXPROFILE:          checkBool,
	USECBO:                checkBool,
	TXTIMEOUT:             checkDuration,
	ATRCOLLECTION:         checkPath,
	CLEANUPWINDOW:         checkDuration,
	CLEANUPCLIENTATTEMPTS: checkBool,
	CLEANUPLOSTATTEMPTS:   checkBool,
}

var CHECKERS_MIN = map[string]int{
	KEEPALIVELENGTH: KEEP_ALIVE_MIN,
	CMPPUSH:         2,
	CMPPOP:          2,
	SERVICERS:       1,
	PLUSSERVICERS:   1,
	PRPLIMIT:        2,
	FUNCLIMIT:       2,
	TASKLIMIT:       2,
	MEMORYQUOTA:     0,
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
		if err != nil && op == CMP_OP_UPD && err.Code() == errors.ADMIN_QUALIFIER_NOT_SET {
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
	_, ok := logging.ParseLevel(level)
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
