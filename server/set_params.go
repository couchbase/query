//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package server

import (
	"strings"
	"time"

	"github.com/couchbase/cbauth/metakv"
	gsi "github.com/couchbase/indexing/secondary/queryport/n1ql"
	ftsclient "github.com/couchbase/n1fty"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/scheduler"
	queryMetakv "github.com/couchbase/query/server/settings/couchbase"
	"github.com/couchbase/query/util"
)

type Setter func(*Server, interface{}) errors.Error

var _SETTERS = map[string]Setter{
	CPUPROFILE: func(s *Server, o interface{}) errors.Error {
		value, _ := o.(string)
		s.SetCpuProfile(value)
		return nil
	},
	DEBUG: func(s *Server, o interface{}) errors.Error {
		value, _ := o.(bool)
		s.SetDebug(value)
		return nil
	},
	KEEPALIVELENGTH: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		s.SetKeepAlive(int(value))
		s.SettingsCallback()(KEEPALIVELENGTH, int(value))
		return nil
	},
	LOGLEVEL: func(s *Server, o interface{}) errors.Error {
		value, _ := o.(string)
		s.SetLogLevel(value)
		return nil
	},
	MAXPARALLELISM: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		s.SetMaxParallelism(int(value))
		return nil
	},
	MEMPROFILE: func(s *Server, o interface{}) errors.Error {
		value, _ := o.(string)
		s.SetMemProfile(value)
		return nil
	},
	PIPELINECAP: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		s.SetPipelineCap(int64(value))
		return nil
	},
	PIPELINEBATCH: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		s.SetPipelineBatch(int(value))
		return nil
	},
	REQUESTSIZECAP: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		s.SetRequestSizeCap(int(value))
		return nil
	},
	SCANCAP: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		s.SetScanCap(int64(value))
		return nil
	},
	SERVICERS: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		s.SetServicers(int(value))
		return nil
	},
	PLUSSERVICERS: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		s.SetPlusServicers(int(value))
		return nil
	},
	TIMEOUTSETTING: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		s.SetTimeout(time.Duration(value))
		return nil
	},
	CMPTHRESHOLD: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		return RequestsUpdateQualifier("threshold", int(value), "")
	},
	CMPLIMIT: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		RequestsSetLimit(int(value), CMP_OP_UPD)
		return nil
	},
	CMPPUSH: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		RequestsSetLimit(int(value), CMP_OP_ADD)
		return nil
	},
	CMPPOP: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		RequestsSetLimit(int(value), CMP_OP_DEL)
		return nil
	},
	CMPOBJECT: setCompleted,
	PRPLIMIT: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		prepareds.PreparedsSetLimit(int(value))
		return nil
	},
	PRETTY: func(s *Server, o interface{}) errors.Error {
		value, _ := o.(bool)
		s.SetPretty(value)
		return nil
	},
	MAXINDEXAPI: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		s.SetMaxIndexAPI(int(value))
		return nil
	},
	PROFILE:  setProfileAdmin,
	CONTROLS: setControlsAdmin,
	N1QLFEATCTRL: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		if s.enterprise {
			util.SetN1qlFeatureControl(uint64(value))
		} else {
			util.SetN1qlFeatureControl(uint64(value) | (util.CE_N1QL_FEAT_CTRL & ^util.N1QL_ENCODED_PLAN))
		}
		return nil
	},
	AUTOPREPARE: func(s *Server, o interface{}) errors.Error {
		value, _ := o.(bool)
		s.SetAutoPrepare(value)
		return nil
	},
	MUTEXPROFILE: func(s *Server, o interface{}) errors.Error {
		value, _ := o.(bool)
		s.SetMutexProfile(value)
		return nil
	},
	FUNCLIMIT: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		functions.FunctionsSetLimit(int(value))
		return nil
	},
	TASKLIMIT: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		scheduler.SchedulerSetLimit(int(value))
		return nil
	},
	MEMORYQUOTA: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		s.SetMemoryQuota(uint64(value))
		return nil
	},
	USECBO: func(s *Server, o interface{}) errors.Error {
		value, _ := o.(bool)
		s.SetUseCBO(value)
		return nil
	},
	TXTIMEOUT: func(s *Server, o interface{}) errors.Error {
		s.SetTxTimeout(getDuration(o))
		return nil
	},
	ATRCOLLECTION: func(s *Server, o interface{}) errors.Error {
		if value, ok := o.(string); ok {
			s.SetAtrCollection(value)
		}
		return nil
	},
	NUMATRS: func(s *Server, o interface{}) errors.Error {
		s.SetNumAtrs(int(getNumber(o)))
		return nil
	},
	CLEANUPWINDOW: func(s *Server, o interface{}) errors.Error {
		datastore.GetTransactionSettings().SetCleanupWindow(getDuration(o))
		return nil
	},
	CLEANUPCLIENTATTEMPTS: func(s *Server, o interface{}) errors.Error {
		if value, ok := o.(bool); ok {
			datastore.GetTransactionSettings().SetCleanupClientAttempts(value)
		}
		return nil
	},
	CLEANUPLOSTATTEMPTS: func(s *Server, o interface{}) errors.Error {
		if value, ok := o.(bool); ok {
			datastore.GetTransactionSettings().SetCleanupLostAttempts(value)
		}
		return nil
	},
	GCPERCENT: func(s *Server, o interface{}) errors.Error {
		if err := s.SetGCPercent(int(getNumber(o))); err != nil {
			return errors.NewServiceErrorBadValue(err, "settings")
		}
		return nil
	},
}

func getNumber(o interface{}) float64 {
	switch o := o.(type) {
	case int64:
		return float64(o)
	case float64:
		return o
	}
	return -1
}

func getDuration(o interface{}) time.Duration {
	switch o := o.(type) {
	case string:
		if d, e := time.ParseDuration(o); e == nil {
			return d
		}
	}
	return 0
}

func setCompleted(s *Server, o interface{}) errors.Error {
	var res errors.Error
	var tag string

	object := o.(map[string]interface{})
	if tagVal, ok := object["tag"]; ok {
		tag, ok = tagVal.(string)
		if !ok {
			return errors.NewAdminSettingTypeError("tag", tagVal)
		}
	}
	for n, v := range object {
		if n == "tag" {
			continue
		}
		res = nil
		switch n[0] {
		case '+':
			n = n[1:len(n)]
			res = RequestsAddQualifier(n, v, tag)
		case '-':
			n = n[1:len(n)]
			res = RequestsRemoveQualifier(n, v, tag)
		default:
			res = RequestsUpdateQualifier(n, v, tag)
			if res != nil {
				switch res.Code() {
				case errors.ADMIN_QUALIFIER_NOT_UNIQUE:
					RequestsRemoveQualifier(n, nil, tag)
					res = RequestsAddQualifier(n, v, tag)
				case errors.ADMIN_QUALIFIER_NOT_SET:
					res = RequestsAddQualifier(n, v, tag)
				}
			}
		}
		if res != nil {
			return res
		}
	}
	return nil
}

func ProcessSettings(settings map[string]interface{}, srvr *Server) (err errors.Error) {
	for setting, value := range settings {
		var cerr errors.Error

		s := strings.ToLower(setting)
		ok := false
		check_it, found := CHECKERS[s]
		if found {
			ok, cerr = check_it(value)
		} else {
			var min int

			min, found = CHECKERS_MIN[s]
			if found {
				ok, cerr = checkNumberMin(value, min)
			}
		}
		if found && ok {
			set_it := _SETTERS[s]
			serr := set_it(srvr, value)
			if serr == nil {
				logging.Infof("Query Configuration changed for %v. New value is %v", s, value)
			} else {
				logging.Infof("Could not change query Configuration %v to %v: %v", s, value, serr)
			}
		} else {
			if !found {
				cerr = errors.NewAdminUnknownSettingError(setting)
				logging.Infof("Query Configuration: %v", cerr.Error())
			} else {
				if cerr == nil {
					cerr = errors.NewAdminSettingTypeError(setting, value)
					logging.Infof("Query Configuration: %v", cerr.Error())
				} else {
					logging.Infof("Query Configuration: Incorrect value %v for setting: %s, error: %v ", value, s, cerr)
				}
			}

			if err == nil {
				err = cerr
			}
		}
	}

	return err
}

func SetParamValuesForAll(cfg queryMetakv.Config, srvr *Server) {
	// Convert value.Value - type OBJECT to map[string]interface{}
	// Range through the config changes and put together 2 lists.
	// List 1 : Indexer settings
	var idxrSettings = map[string]interface{}{}

	// List 2 : Query settings
	var querySettings = map[string]interface{}{}

	configValues := cfg.Fields()
	for key, val := range configValues {
		// INDEXER PARAM
		paramName, ok := queryMetakv.INDEXERPARAM[key]
		if ok {
			idxrSettings[paramName] = val
		} else {
			// QUERY PARAM
			paramName, ok := queryMetakv.GLOBALPARAM[key]
			if ok && paramName == "curl_whitelist" {
				// Set the whitelist value to pass to context
				srvr.SetWhitelist(val.(map[string]interface{}))
				logging.Infof("New Value for curl allowedlist <ud>%v</ud>", val)
			} else if ok && paramName == "curl_allowedlist" {
				srvr.SetWhitelist(val.(map[string]interface{}))
				logging.Infof("New Value for curl allowedlist <ud>%v</ud>", val)
			} else {
				querySettings[key] = val
			}
		}
	}

	if len(idxrSettings) > 0 {
		// Call a global function defined by indexer
		idxConfig, err := gsi.GetIndexConfig()
		if err != nil {
			logging.Errorf(" Cannot get gsi index config :: %v", err.Error())
		} else if err = idxConfig.SetConfig(idxrSettings); err != nil {
			logging.Errorf(" Could not set GSI indexer settings (%v) :: %v", idxrSettings, err.Error())
		} else {
			logging.Infof(" GSI indexer settings have been updated %v", idxrSettings)
		}

		idxConfig, err = ftsclient.GetConfig()
		if err != nil {
			logging.Errorf(" Cannot get n1fty index config :: %v", err.Error())
		} else if err = idxConfig.SetConfig(idxrSettings); err != nil {
			logging.Errorf(" Could not set n1fty indexer settings (%v) :: %v",
				idxrSettings, err.Error())
		} else {
			logging.Infof(" n1fty indexer settings have been updated %v", idxrSettings)
		}

	}

	if len(querySettings) > 0 {
		// Set the query values
		ProcessSettings(querySettings, srvr)
	}
}

// FTS MetakvNotifier notifies the FTS client about any metakv changes it subscribed for.

func N1ftyMetakvNotifier(kve metakv.KVEntry) error {
	configs := map[string]interface{}{kve.Path: kve.Value}
	idxConfig, err := ftsclient.GetConfig()
	if err != nil {
		logging.Errorf(" Cannot get n1fty index config :: %v", err.Error())
	} else if err = idxConfig.SetConfig(configs); err != nil {
		logging.Errorf(" Could not set n1fty indexer settings :: %v", err.Error())
	}
	return err
}
