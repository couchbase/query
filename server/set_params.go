//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package server

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/couchbase/cbauth/metakv"
	gsi "github.com/couchbase/indexing/secondary/queryport/n1ql"
	ftsclient "github.com/couchbase/n1fty"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/logging/event"
	"github.com/couchbase/query/memory"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/scheduler"
	queryMetakv "github.com/couchbase/query/server/settings/couchbase"
	"github.com/couchbase/query/sort"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
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
		memory.Config(memory.NodeQuota(), memory.ValPercent(), []int{s.Servicers(), s.PlusServicers()})
		return nil
	},
	PLUSSERVICERS: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		s.SetPlusServicers(int(value))
		memory.Config(memory.NodeQuota(), memory.ValPercent(), []int{s.Servicers(), s.PlusServicers()})
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
	CMPMAXPLANSIZE: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		RequestsSetMaxPlanSize(int(value))
		return nil
	},
	CMPSTREAM: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		RequestsSetFileStreamSize(int64(value))
		return nil
	},
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
		value := getHexNumber(o)
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
	NODEQUOTA: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		memory.Config(uint64(value), memory.ValPercent(), []int{s.Servicers(), s.PlusServicers()})
		tenant.Config(memory.Quota())
		return nil
	},
	NODEQUOTAVALPERCENT: func(s *Server, o interface{}) errors.Error {
		value := getNumber(o)
		memory.Config(memory.NodeQuota(), uint(value), []int{s.Servicers(), s.PlusServicers()})
		tenant.Config(memory.Quota())
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
	REQUESTERRORLIMIT: func(s *Server, o interface{}) errors.Error {
		if err := s.SetRequestErrorLimit(int(getNumber(o))); err != nil {
			return errors.NewServiceErrorBadValue(err, "settings")
		}
		return nil
	},
	DURATIONSTYLE: func(s *Server, o interface{}) errors.Error {
		var ok bool
		var str string
		var style util.DurationStyle
		if str, ok = o.(string); ok {
			if style, ok = util.IsDurationStyle(str); ok {
				// permit only styles that the UI can parse as the default
				if style == util.LEGACY || style == util.COMPATIBLE {
					util.SetDurationStyle(style)
				} else {
					ok = false
				}
			}
		}
		if !ok {
			return errors.NewServiceErrorBadValue(nil, "settings")
		}
		return nil
	},
	/*
	   	"enforce_limits": func(s *Server, o interface{}) errors.Error {
	                   s.SettingsCallback()("enforce_limits", o)
	                   return nil
	*/
	QUERY_TMP_DIR: func(s *Server, o interface{}) errors.Error {
		if err := util.SetTempDir(o.(string)); err != nil {
			return errors.NewServiceErrorBadValue(err, "settings")
		}
		return nil
	},
	QUERY_TMP_LIMIT: func(s *Server, o interface{}) errors.Error {
		if err := util.SetTempQuota(int64(getNumber(o)) * util.MiB); err != nil {
			return errors.NewServiceErrorBadValue(err, "settings")
		}
		return nil
	},
	USEREPLICA: func(s *Server, o interface{}) errors.Error {
		ur, ok := o.(string)

		if ok {
			urv, ok1 := value.ParseTristateString(ur)

			if ok1 {
				s.SetUseReplica(urv)
			}
		}

		return nil
	},
	NUM_CPUS: func(s *Server, o interface{}) errors.Error {
		logging.Infoa(func() string {
			return fmt.Sprintf("%s updated to %d.  Change will take place on restart.", NUM_CPUS, int(getNumber(o)))
		})
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

func getHexNumber(o interface{}) int64 {
	switch o := o.(type) {
	case int64:
		return o
	case uint64:
		return int64(o)
	case string:
		if v, err := strconv.ParseInt(o, 0, 64); err == nil {
			return v
		}
	}
	return -1
}

func getDuration(o interface{}) time.Duration {
	switch o := o.(type) {
	case string:
		if d, e := util.ParseDurationStyle(o, util.DEFAULT); e == nil {
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
				case errors.E_COMPLETED_QUALIFIER_NOT_UNIQUE:
					RequestsRemoveQualifier(n, nil, tag)
					res = RequestsAddQualifier(n, v, tag)
				case errors.E_COMPLETED_QUALIFIER_NOT_FOUND:
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

var reportAllInitially = true

func ProcessSettings(settings map[string]interface{}, srvr *Server) (err errors.Error) {
	prev := make(map[string]interface{})
	prev = FillSettings(prev, srvr)

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
			if serr != nil {
				logging.Infof("Could not change query Configuration %v to %v: %v", s, value, serr)
				if err == nil {
					err = serr
				}
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

	current := make(map[string]interface{})
	current = FillSettings(current, srvr)
	reportChangedValues(prev, current, reportAllInitially)
	reportAllInitially = false

	return err
}

func compare(a, b interface{}) bool {
	switch av := a.(type) {
	case map[string]interface{}:
		if bv, ok := b.(map[string]interface{}); ok {
			if len(av) != len(bv) {
				return false
			}
			for k, v := range av {
				bvv, ok := bv[k]
				if !ok || !compare(v, bvv) {
					return false
				}
			}
			return true
		}
		return false
	case []interface{}:
		if bv, ok := b.([]interface{}); ok {
			if len(av) != len(bv) {
				return false
			}
			for i := range av {
				if !compare(av[i], bv[i]) {
					return false
				}
			}
			return true
		}
		return false
	case interface{ Equals(interface{}) bool }:
		return av.Equals(b)
	default:
		return a == b
	}
}

func reportChangedValues(prev map[string]interface{}, current map[string]interface{}, all bool) {
	changed := make([]interface{}, 0, len(current))

	names := make([]string, 0, len(current))
	for setting, _ := range current {
		names = append(names, setting)
	}
	sort.Strings(names)

	for _, k := range names {
		v := current[k]
		p, ok := prev[k]
		same := false
		if ok && reflect.TypeOf(v) == reflect.TypeOf(p) {
			switch vt := v.(type) {
			case map[string]interface{}:
				pt := p.(map[string]interface{})
				same = compare(vt, pt)
			case []interface{}:
				pt := p.([]interface{})
				if len(vt) == len(pt) {
					same = true
					for i := 0; i < len(vt) && same == true; i++ {
						if reflect.TypeOf(vt[i]) == reflect.TypeOf(pt[i]) {
							switch vtt := vt[i].(type) {
							case map[string]interface{}:
								ptt := pt[i].(map[string]interface{})
								same = compare(vtt, ptt)
							default:
								same = vt[i] == pt[i]
							}
						} else {
							same = false
						}
					}
				}
			default:
				same = p == v
			}
		}
		if !same || all {
			changed = append(changed, k)
			changeRecord := make(map[string]interface{})
			changeRecord["from"] = p
			changeRecord["to"] = v
			changed = append(changed, changeRecord)
			if _, ok := v.(map[string]interface{}); !ok {
				var extra string
				// log what features have changed in the new n1ql-feat-ctrl bitset
				if k == N1QLFEATCTRL {
					prevVal := uint64(0) // so on start-up all disabled features are logged
					currVal := uint64(getHexNumber(v))

					if !same && !all {
						prevVal = uint64(getHexNumber(p))
					} else {
						same = true
					}
					extra = util.DescribeChangedFeatures(prevVal, currVal)
				} else if k == "completed-threshold" {
					if p == nil {
						p = "nil"
					} else {
						p = p.(time.Duration) * time.Millisecond
					}
					if v == nil {
						v = "nil"
					} else {
						v = v.(time.Duration) * time.Millisecond
					}
				}

				if !same {
					logging.Infof("Query Configuration changed for %v from %v to %v%s", k, p, v, extra)
				} else {
					logging.Infof("Query Configuration changed for %v. New value is %v%s", k, v, extra)
				}
			}
		}
	}
	if len(changed) > 0 {
		event.Report(event.CONFIG_CHANGE, event.INFO, changed...)
	}
}

func FillSettings(settings map[string]interface{}, srvr *Server) map[string]interface{} {
	settings[CPUPROFILE] = srvr.CpuProfile()
	settings[MEMPROFILE] = srvr.MemProfile()
	settings[SERVICERS] = srvr.Servicers()
	settings[PLUSSERVICERS] = srvr.PlusServicers()
	settings[SCANCAP] = srvr.ScanCap()
	settings[REQUESTSIZECAP] = srvr.RequestSizeCap()
	settings[DEBUG] = srvr.Debug()
	settings[PIPELINEBATCH] = srvr.PipelineBatch()
	settings[PIPELINECAP] = srvr.PipelineCap()
	settings[MAXPARALLELISM] = srvr.MaxParallelism()
	settings[TIMEOUTSETTING] = srvr.Timeout()
	settings[KEEPALIVELENGTH] = srvr.KeepAlive()
	settings[LOGLEVEL] = srvr.LogLevel()
	threshold, _ := RequestsGetQualifier("threshold", "")
	settings[CMPTHRESHOLD] = threshold
	settings[CMPLIMIT] = RequestsLimit()
	settings[CMPOBJECT] = RequestsGetQualifiers()
	settings[CMPMAXPLANSIZE] = RequestsMaxPlanSize()
	settings[CMPSTREAM] = RequestsFileStreamSize()
	settings[PRPLIMIT] = prepareds.PreparedsLimit()
	settings[PRETTY] = srvr.Pretty()
	settings[MAXINDEXAPI] = srvr.MaxIndexAPI()
	settings[N1QLFEATCTRL] = util.GetN1qlFeatureControl()
	settings[TXTIMEOUT] = util.OutputDuration(srvr.TxTimeout())
	settings = GetProfileAdmin(settings, srvr)
	settings = GetControlsAdmin(settings, srvr)
	settings[AUTOPREPARE] = srvr.AutoPrepare()
	settings[MUTEXPROFILE] = srvr.MutexProfile()
	settings[FUNCLIMIT] = functions.FunctionsLimit()
	settings[MEMORYQUOTA] = srvr.MemoryQuota()
	settings[NODEQUOTA] = memory.NodeQuota()
	settings[NODEQUOTAVALPERCENT] = memory.ValPercent()
	settings[USECBO] = srvr.UseCBO()
	settings[ATRCOLLECTION] = srvr.AtrCollection()
	settings[NUMATRS] = srvr.NumAtrs()

	tranSettings := datastore.GetTransactionSettings()
	settings[CLEANUPWINDOW] = tranSettings.CleanupWindow().String()
	settings[CLEANUPCLIENTATTEMPTS] = tranSettings.CleanupClientAttempts()
	settings[CLEANUPLOSTATTEMPTS] = tranSettings.CleanupLostAttempts()
	settings[GCPERCENT] = srvr.GCPercent()
	settings[REQUESTERRORLIMIT] = srvr.RequestErrorLimit()
	settings[USEREPLICA] = srvr.UseReplicaToString()
	settings[NUM_CPUS] = util.NumCPU()
	settings[DURATIONSTYLE] = util.GetDurationStyle().String()
	return settings
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
			querySettings[paramName] = val
		} else {
			// QUERY PARAM
			paramName, ok := queryMetakv.GLOBALPARAM[key]
			if ok && (paramName == "curl_whitelist" || paramName == "curl_allowedlist") {
				// Set the allowlist value to pass to context

				// Create a new map just for comparison between new and existing values for this parameter
				// This new map excludes the allowed and disallowed URL object fields
				// this is because the value package cannot handle url.Url object type
				serverAllowList := make(map[string]interface{}, len(srvr.GetAllowlist()))
				for k, v := range srvr.GetAllowlist() {
					if k == "allowed_transformed_urls" || k == "disallowed_transformed_urls" {
						continue
					}
					serverAllowList[k] = v
				}

				al := value.NewValue(serverAllowList)
				nal := value.NewValue(val)

				if !al.Equals(nal).Truth() {
					srvr.SetAllowlist(val.(map[string]interface{}))
					logging.Infof("New Value for curl allowed list <ud>%v</ud>", val)
				}
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
