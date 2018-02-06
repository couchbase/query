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
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/prepareds"
	paramSettings "github.com/couchbase/query/server/settings"
	queryMetakv "github.com/couchbase/query/server/settings/couchbase"
	"github.com/couchbase/query/util"
)

type Setter func(*Server, interface{})

var _SETTERS = map[string]Setter{
	paramSettings.CPUPROFILE: func(s *Server, o interface{}) {
		value, _ := o.(string)
		s.SetCpuProfile(value)
	},
	paramSettings.DEBUG: func(s *Server, o interface{}) {
		value, _ := o.(bool)
		s.SetDebug(value)
	},
	paramSettings.KEEPALIVELENGTH: func(s *Server, o interface{}) {
		value, _ := o.(float64)
		s.SetKeepAlive(int(value))
	},
	paramSettings.LOGLEVEL: func(s *Server, o interface{}) {
		value, _ := o.(string)
		s.SetLogLevel(value)
	},
	paramSettings.MAXPARALLELISM: func(s *Server, o interface{}) {
		value, _ := o.(float64)
		s.SetMaxParallelism(int(value))
	},
	paramSettings.MEMPROFILE: func(s *Server, o interface{}) {
		value, _ := o.(string)
		s.SetMemProfile(value)
	},
	paramSettings.PIPELINECAP: func(s *Server, o interface{}) {
		value, _ := o.(float64)
		s.SetPipelineCap(int64(value))
	},
	paramSettings.PIPELINEBATCH: func(s *Server, o interface{}) {
		value, _ := o.(float64)
		s.SetPipelineBatch(int(value))
	},
	paramSettings.REQUESTSIZECAP: func(s *Server, o interface{}) {
		value, _ := o.(float64)
		s.SetRequestSizeCap(int(value))
	},
	paramSettings.SCANCAP: func(s *Server, o interface{}) {
		value, _ := o.(float64)
		s.SetScanCap(int64(value))
	},
	paramSettings.SERVICERS: func(s *Server, o interface{}) {
		value, _ := o.(float64)
		s.SetServicers(int(value))
	},
	paramSettings.TIMEOUTSETTING: func(s *Server, o interface{}) {
		value, _ := o.(float64)
		s.SetTimeout(time.Duration(value))
	},
	paramSettings.CMPTHRESHOLD: func(s *Server, o interface{}) {
		value, _ := o.(float64)
		_ = RequestsUpdateQualifier("threshold", int(value))
	},
	paramSettings.CMPLIMIT: func(s *Server, o interface{}) {
		value, _ := o.(float64)
		RequestsSetLimit(int(value))
	},
	paramSettings.PRPLIMIT: func(s *Server, o interface{}) {
		value, _ := o.(float64)
		prepareds.PreparedsSetLimit(int(value))
	},
	paramSettings.PRETTY: func(s *Server, o interface{}) {
		value, _ := o.(bool)
		s.SetPretty(value)
	},
	paramSettings.MAXINDEXAPI: func(s *Server, o interface{}) {
		value, _ := o.(float64)
		s.SetMaxIndexAPI(int(value))
	},
	paramSettings.PROFILE:  setProfileAdmin,
	paramSettings.CONTROLS: setControlsAdmin,
	paramSettings.N1QLFEATCTRL: func(s *Server, o interface{}) {
		value, _ := o.(float64)
		if s.enterprise {
			util.SetN1qlFeatureControl(uint64(value))
		} else {
			util.SetN1qlFeatureControl(uint64(value) | util.CE_N1QL_FEAT_CTRL)
		}
	},
}

func ProcessSettings(settings map[string]interface{}, srvr *Server) errors.Error {
	for setting, value := range settings {
		if check_it, ok := paramSettings.CHECKERS[setting]; !ok {
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
		set_it(srvr, value)
		logging.Infof("Query Configuration changed for %v. New value is %v", setting, value)
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
			querySettings[key] = val
		}
	}

	if len(idxrSettings) > 0 {
		// Call a global function defined by indexer
		var idxConfig datastore.IndexConfig
		var err error
		if idxConfig, err = gsi.GetIndexConfig(); err == nil {
			err = idxConfig.SetConfig(idxrSettings)
			if err != nil {
				//log failure to set values
				logging.Infof(" ERROR: Could not set indexer settings :: %v", idxrSettings)
			}
			logging.Infof(" Indexer settings have been updated %v", idxrSettings)
		} else {
			logging.Infof(" ERROR: Cannot get index config :: %v", err.Error())

		}
	}

	if len(querySettings) > 0 {
		// Set the query values
		ProcessSettings(querySettings, srvr)
	}
}
