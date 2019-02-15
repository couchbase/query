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

	gsi "github.com/couchbase/indexing/secondary/queryport/n1ql"
	//	ftsclient "github.com/couchbase/n1fty"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/prepareds"
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
	TIMEOUTSETTING: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		s.SetTimeout(time.Duration(value))
		return nil
	},
	CMPTHRESHOLD: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		return RequestsUpdateQualifier("threshold", int(value))
	},
	CMPLIMIT: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		RequestsSetLimit(int(value))
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
			util.SetN1qlFeatureControl(uint64(value) | util.CE_N1QL_FEAT_CTRL)
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

func setCompleted(s *Server, o interface{}) errors.Error {
	var res errors.Error

	for n, v := range o.(map[string]interface{}) {
		res = nil
		switch n[0] {
		case '+':
			n = n[1:len(n)]
			res = RequestsAddQualifier(n, v)
		case '-':
			n = n[1:len(n)]
			res = RequestsRemoveQualifier(n, v)
		default:
			res = RequestsUpdateQualifier(n, v)
			if res != nil {
				switch res.Code() {
				case errors.ADMIN_QUALIFIER_NOT_UNIQUE:
					RequestsRemoveQualifier(n, nil)
					res = RequestsAddQualifier(n, v)
				case errors.ADMIN_QUALIFIER_NOT_SET:
					res = RequestsAddQualifier(n, v)
				}
			}
		}
		if res != nil {
			return res
		}
	}
	return nil
}

func ProcessSettings(settings map[string]interface{}, srvr *Server) errors.Error {
	for setting, value := range settings {
		if check_it, ok := CHECKERS[setting]; !ok {
			return errors.NewAdminUnknownSettingError(setting)
		} else {
			ok, err := check_it(value)
			if !ok {
				if err == nil {
					return errors.NewAdminSettingTypeError(setting, value)
				} else {
					return err
				}
			}
		}
	}
	for setting, value := range settings {
		set_it := _SETTERS[setting]
		err := set_it(srvr, value)
		if err == nil {
			logging.Infof("Query Configuration changed for %v. New value is %v", setting, value)
		} else {
			logging.Infof("Could not change query Configuration %v to %v: %v", setting, value, err)
		}
	}
	return nil
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
				logging.Infof("New Value for curl whitelist <ud>%v</ud>", val)
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

		/*
			idxConfig, err = ftsclient.GetConfig()
			if err != nil {
				logging.Errorf(" Cannot get n1fty index config :: %v", err.Error())
			} else if err = idxConfig.SetConfig(idxrSettings); err != nil {
				logging.Errorf(" Could not set n1fty indexer settings (%v) :: %v",
					idxrSettings, err.Error())
			} else {
				logging.Infof(" n1fty indexer settings have been updated %v", idxrSettings)
			}
		*/

	}

	if len(querySettings) > 0 {
		// Set the query values
		ProcessSettings(querySettings, srvr)
	}
}

// FTS MetakvNotifier notifies the FTS client about any metakv changes it subscribed for.

func N1ftyMetakvNotifier(path string, val []byte, rev interface{}) error {
	/*
		configs := map[string]interface{}{path: val}
		idxConfig, err := ftsclient.GetConfig()
		if err != nil {
			logging.Errorf(" Cannot get n1fty index config :: %v", err.Error())
		} else if err = idxConfig.SetConfig(configs); err != nil {
			logging.Errorf(" Could not set n1fty indexer settings :: %v", err.Error())
		}
		return err
	*/
	return nil
}
