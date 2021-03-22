//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"syscall"
	"time"

	//	go_http "net/http"
	//	_ "net/http/pprof"

	"github.com/couchbase/query/accounting"
	acct_resolver "github.com/couchbase/query/accounting/resolver"
	"github.com/couchbase/query/audit"
	config_resolver "github.com/couchbase/query/clustering/resolver"
	datastore_package "github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/resolver"
	"github.com/couchbase/query/datastore/system"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/functions/constructor"
	"github.com/couchbase/query/logging"
	log_resolver "github.com/couchbase/query/logging/resolver"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/scheduler"
	server_package "github.com/couchbase/query/server"
	"github.com/couchbase/query/server/http"
	"github.com/couchbase/query/util"
)

const (
	_DEF_HTTP                   = ":8093"
	_DEF_HTTPS                  = ":18093"
	_DEF_REQUEST_CAP            = 256
	_DEF_SCAN_CAP               = 512
	_DEF_PIPELINE_CAP           = 512
	_DEF_PIPELINE_BATCH         = 16
	_DEF_COMPLETED_THRESHOLD    = 1000
	_DEF_COMPLETED_LIMIT        = 4000
	_DEF_PREPARED_LIMIT         = 16384
	_DEF_FUNCTIONS_LIMIT        = 16384
	_DEF_DICTIONARY_CACHE_LIMIT = 16384
	_DEF_TASKS_LIMIT            = 16384
	_DEF_MEMORY_QUOTA           = 0
)

var DATASTORE = flag.String("datastore", "", "Datastore address (http://URL or dir:PATH or mock:)")
var CONFIGSTORE = flag.String("configstore", "stub:", "Configuration store address (http://URL or stub:)")
var ACCTSTORE = flag.String("acctstore", "gometrics:", "Accounting store address (http://URL or stub:)")
var NAMESPACE = flag.String("namespace", "default", "Default namespace")
var TIMEOUT = flag.Duration("timeout", 0*time.Second, "Server execution timeout, e.g. 500ms or 2s; use zero or negative value to disable")
var TXTIMEOUT = flag.Duration("txtimeout", 0*time.Second, "Maximum Transaction timeout, e.g. 2m or 2s; use zero or negative to use request level value")
var READONLY = flag.Bool("readonly", false, "Read-only mode")
var SIGNATURE = flag.Bool("signature", true, "Whether to provide signature")
var METRICS = flag.Bool("metrics", true, "Whether to provide metrics")
var PRETTY = flag.Bool("pretty", false, "Pretty output")
var REQUEST_CAP = flag.Int("request-cap", _DEF_REQUEST_CAP, "Maximum number of queued requests per logical CPU")
var REQUEST_SIZE_CAP = flag.Int("request-size-cap", server_package.MAX_REQUEST_SIZE, "Maximum size of a request")
var SCAN_CAP = flag.Int64("scan-cap", _DEF_SCAN_CAP, "Maximum buffer size for index scans; use zero or negative value to disable")
var SERVICERS = flag.Int("servicers", 4*runtime.NumCPU(), "Servicer count")
var PLUS_SERVICERS = flag.Int("plus-servicers", 16*runtime.NumCPU(), "Plus servicer count")
var MAX_PARALLELISM = flag.Int("max-parallelism", 1, "Maximum parallelism per query; use zero or negative value to use maximum")
var ORDER_LIMIT = flag.Int64("order-limit", 0, "Maximum LIMIT for ORDER BY clauses; use zero or negative value to disable")
var MUTATION_LIMIT = flag.Int64("mutation-limit", 0, "Maximum LIMIT for data modification statements; use zero or negative value to disable")
var HTTP_ADDR = flag.String("http", _DEF_HTTP, "HTTP service address")
var HTTPS_ADDR = flag.String("https", _DEF_HTTPS, "HTTPS service address")
var CERT_FILE = flag.String("certfile", "", "HTTPS certificate file")
var KEY_FILE = flag.String("keyfile", "", "HTTPS private key file")
var IPv6 = flag.String("ipv6", server_package.TCP_OPT, "Query is IPv6 compliant")
var IPv4 = flag.String("ipv4", server_package.TCP_REQ, "Query uses IPv4 listeners only")

var LOGGER = flag.String("logger", "", "Logger implementation")
var LOG_LEVEL = flag.String("loglevel", "info", "Log level: debug, trace, info, warn, error, severe, none")
var DEBUG = flag.Bool("debug", false, "Debug mode")
var KEEP_ALIVE_LENGTH = flag.Int("keep-alive-length", server_package.KEEP_ALIVE_DEFAULT, "maximum size of buffered result")
var STATIC_PATH = flag.String("static-path", "static", "Path to static content")
var PIPELINE_CAP = flag.Int64("pipeline-cap", _DEF_PIPELINE_CAP, "Maximum number of items each execution operator can buffer")
var PIPELINE_BATCH = flag.Int("pipeline-batch", _DEF_PIPELINE_BATCH, "Number of items execution operators can batch")
var ENTERPRISE = flag.Bool("enterprise", true, "Enterprise mode")
var MAX_INDEX_API = flag.Int("max-index-api", datastore_package.INDEX_API_MAX, "Max Index API")
var N1QL_FEAT_CTRL = flag.Uint64("n1ql-feat-ctrl", util.DEF_N1QL_FEAT_CTRL, "N1QL Feature Controls")
var MEMORY_QUOTA = flag.Uint64("memory-quota", _DEF_MEMORY_QUOTA, "Maximum amount of document memory allowed per request, in MB")

//cpu and memory profiling flags
var CPU_PROFILE = flag.String("cpuprofile", "", "write cpu profile to file")
var MEM_PROFILE = flag.String("memprofile", "", "write memory profile to this file")

// Monitoring API
var COMPLETED_THRESHOLD = flag.Int("completed-threshold", _DEF_COMPLETED_THRESHOLD, "cache completed query lasting longer than this many milliseconds")
var COMPLETED_LIMIT = flag.Int("completed-limit", _DEF_COMPLETED_LIMIT, "maximum number of completed requests")

var PREPARED_LIMIT = flag.Int("prepared-limit", _DEF_PREPARED_LIMIT, "maximum number of prepared statements")
var AUTO_PREPARE = flag.Bool("auto-prepare", false, "Silently prepare ad hoc statements if possible")

var FUNCTIONS_LIMIT = flag.Int("functions-limit", _DEF_FUNCTIONS_LIMIT, "maximum number of cached functions")
var TASKS_LIMIT = flag.Int("tasks-limit", _DEF_TASKS_LIMIT, "maximum number of cached tasks")

// GOGC
var _GOGC_PERCENT = 150

// profiler, to use instead of the REST endpoint if needed
// var PROFILER_PORT = flag.Int("profiler-port", 6060, "profiler listening port")

// Dictionary Cache
var DICTIONARY_CACHE_LIMIT = flag.Int("dictionary-cache-limit", _DEF_DICTIONARY_CACHE_LIMIT, "maximum number of entries in dictionary cache")

func init() {
	debug.SetGCPercent(_GOGC_PERCENT)
}

func main() {

	HideConsole(true)
	defer HideConsole(false)
	flag.Parse()

	// Use the IPv4/IPv6 flags to setup listener bool value
	// This is for external interfaces / listeners
	// localhost represents IPv4. This is always true inless IPv6 is required.
	listener := false

	if *IPv6 == server_package.TCP_REQ {
		listener = true
	}

	// Use the datastore/configstore and accountingstore values
	// setup localhost bool value.
	// This is for IPv6 support for internal interfaces
	localhost6 := false // ipv4 endpoints

	// Check if file path.

	bval1, err := server_package.CheckURL(*HTTP_ADDR, "http addr")
	if err != nil {
		bval1, err = server_package.CheckURL(*DATASTORE, "datastore")
		if err != nil {
			bval1, err = server_package.CheckURL(*CONFIGSTORE, "configstore")
			if err != nil {
				bval1, err = server_package.CheckURL(*ACCTSTORE, "accounting store")
				if err != nil {
					// Its not a valid url but it could be a filepath for filestore access
					if _, err1 := os.Stat(*DATASTORE); os.IsNotExist(err1) {
						fmt.Printf("ERROR: %s\n", err)
						os.Exit(1)
					} else {
						// Set IPV6 as false
						bval1 = false
					}
				}
			}
		}
	}

	localhost6 = bval1

	err = server_package.SetIP(*IPv4, *IPv6, localhost6, listener)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		os.Exit(1)
	}

	// useful for getting list of go-routines
	// localhost needs to refer to either 127.0.0.1 or [::1]
	// to be used instead of the REST endpoint if ever needed
	// var profilerPort string
	//
	// if *PROFILER_PORT <= 0 || *PROFILER_PORT > 65535 {
	// 	profilerPort = ":6060"
	// } else {
	// 	profilerPort = fmt.Sprintf(":%d", *PROFILER_PORT)
	// }
	// urlV := server.GetIP(true) + profilerPort
	// go go_http.ListenAndServe(urlV, nil)

	if *LOGGER != "" {
		logger, _ := log_resolver.NewLogger(*LOGGER)
		if logger == nil {
			fmt.Printf("Invalid logger: %s\n", *LOGGER)
			os.Exit(1)
		}

		logging.SetLogger(logger)
	}

	if *DEBUG {
		logging.SetLevel(logging.DEBUG)
	} else {
		level := logging.INFO

		if *LOG_LEVEL != "" {
			lvl, ok := logging.ParseLevel(*LOG_LEVEL)
			if ok {
				level = lvl
			}
		}

		logging.SetLevel(level)
	}

	datastore, err := resolver.NewDatastore(*DATASTORE)
	if err != nil {
		logging.Errorf("%v", err.Error())
		logging.Errorf("Shutting down.")
		os.Exit(1)
	}
	datastore_package.SetDatastore(datastore)

	nullSecurityConfig := &datastore_package.ConnectionSecurityConfig{}
	datastore.SetConnectionSecurityConfig(nullSecurityConfig)

	// configstore should be set before the system datastore
	configstore, err := config_resolver.NewConfigstore(*CONFIGSTORE)
	if err != nil {
		logging.Errorf("Could not connect to configstore: %v", err)
	}

	configstore.SetOptions(*HTTP_ADDR, *HTTPS_ADDR, (*HTTP_ADDR == _DEF_HTTP && *HTTPS_ADDR == _DEF_HTTPS))

	// ditto for distributed access for monitoring
	// also distributed is used by many init() functions and should be done as early as possible
	prof, ctrl, err := monitoringInit(configstore)
	if err != nil {
		logging.Errorf("%v", err.Error())
		fmt.Printf("\n%v\n", err)
		os.Exit(1)
	}

	acctstore, err := acct_resolver.NewAcctstore(*ACCTSTORE)
	if err != nil {
		logging.Errorf("Could not connect to acctstore: %v", err)
	} else {
		// Create the metrics we are interested in
		accounting.RegisterMetrics(acctstore)
		// Make metrics available
		acctstore.MetricReporter().Start(1, 1)
	}

	numCPU := runtime.NumCPU()
	if *ENTERPRISE && os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(numCPU)
	}

	if !*ENTERPRISE {
		if os.Getenv("GOMAXPROCS") != "" {
			numCPU = runtime.GOMAXPROCS(0)
		}

		// Use at most 4 cpus in non-enterprise mode
		runtime.GOMAXPROCS(util.MinInt(numCPU, 4))
	}

	// Start the completed requests log
	server_package.RequestsInit(*COMPLETED_THRESHOLD, *COMPLETED_LIMIT)

	// Initialized the prepared statement cache
	if *PREPARED_LIMIT <= 0 {
		logging.Errorf("Ignoring invalid prepared statement cache size: %v", *PREPARED_LIMIT)
		*PREPARED_LIMIT = _DEF_PREPARED_LIMIT
	}
	prepareds.PreparedsInit(*PREPARED_LIMIT)
	functions.FunctionsSetLimit(*FUNCTIONS_LIMIT)
	scheduler.SchedulerSetLimit(*TASKS_LIMIT)

	if *DICTIONARY_CACHE_LIMIT <= 0 {
		logging.Errorf("Ignoring invalid dictionary cache size: %v", *DICTIONARY_CACHE_LIMIT)
		*DICTIONARY_CACHE_LIMIT = _DEF_DICTIONARY_CACHE_LIMIT
	}
	// Initialize dictionary cache
	server_package.InitDictionaryCache(*DICTIONARY_CACHE_LIMIT)

	numProcs := runtime.GOMAXPROCS(0)

	sys, err := system.NewDatastore(datastore)
	if err != nil {
		logging.Errorf("%v", err.Error())
		os.Exit(1)
	}

	server, err := server_package.NewServer(datastore, sys, configstore, acctstore, *NAMESPACE,
		*READONLY, *REQUEST_CAP*numProcs, *REQUEST_CAP*numProcs, *SERVICERS, *PLUS_SERVICERS,
		*MAX_PARALLELISM, *TIMEOUT, *SIGNATURE, *METRICS, *ENTERPRISE,
		*PRETTY, prof, ctrl)
	if err != nil {
		logging.Errorf("%v", err.Error())
		os.Exit(1)
	}

	datastore_package.SetSystemstore(server.Systemstore())
	prepareds.PreparedsReprepareInit(datastore, sys)

	server.SetCpuProfile(*CPU_PROFILE)
	server.SetKeepAlive(*KEEP_ALIVE_LENGTH)
	server.SetMemProfile(*MEM_PROFILE)
	server.SetScanCap(*SCAN_CAP)
	server.SetPipelineCap(*PIPELINE_CAP)
	server.SetPipelineBatch(*PIPELINE_BATCH)
	server.SetRequestSizeCap(*REQUEST_SIZE_CAP)
	server.SetScanCap(*SCAN_CAP)
	server.SetMaxIndexAPI(*MAX_INDEX_API)
	server.SetAutoPrepare(*AUTO_PREPARE)
	server.SetTxTimeout(*TXTIMEOUT)
	if *ENTERPRISE {
		util.SetN1qlFeatureControl(*N1QL_FEAT_CTRL)
		util.SetUseCBO(util.DEF_USE_CBO)
	} else {
		util.SetN1qlFeatureControl(*N1QL_FEAT_CTRL | util.CE_N1QL_FEAT_CTRL)
		util.SetUseCBO(util.CE_USE_CBO)
	}
	server.SetMemoryQuota(*MEMORY_QUOTA)

	audit.StartAuditService(*DATASTORE, *SERVICERS+*PLUS_SERVICERS)

	ll := logging.LogLevel().String() // extract first
	logging.Infoa(func() string {
		return fmt.Sprintf("cbq-engine started"+
			" version=%v"+
			" datastore=%v"+
			" max-concurrency=%v"+
			" loglevel=%v"+
			" servicers=%v"+
			" plus-servicers=%v"+
			" scan-cap=%v"+
			" pipeline-cap=%v"+
			" pipeline-batch=%v"+
			" request-cap=%v"+
			" request-size-cap=%v"+
			" max-index-api=%v"+
			" max-parallelism=%v"+
			" n1ql-feat-ctrl=%v"+
			" use-cbo=%v"+
			" timeout=%v"+
			" txtimeout=%v",
			util.VERSION,
			*DATASTORE,
			numProcs,
			ll,
			server.Servicers(),
			server.PlusServicers(),
			server.ScanCap(),
			server.PipelineCap(),
			server.PipelineBatch(),
			*REQUEST_CAP,
			server.RequestSizeCap(),
			server.MaxIndexAPI(),
			server.MaxParallelism(),
			util.GetN1qlFeatureControl(),
			util.GetUseCBO(),
			server.Timeout(),
			server.TxTimeout(),
		)
	})

	// Create http endpoint
	endpoint := http.NewServiceEndpoint(server, *STATIC_PATH, *METRICS,
		*HTTP_ADDR, *HTTPS_ADDR, *CERT_FILE, *KEY_FILE)
	er := endpoint.Listen()
	if er != nil {
		logging.Errorf("cbq-engine (HTTP_ADDR %v) exiting with error: %v", *HTTP_ADDR, er)
		os.Exit(1)
	}
	server.SetSettingsCallback(endpoint.SettingsCallback)
	constructor.Init(endpoint.Mux())

	// Now that we are up and running, try to prime the prepareds cache
	prepareds.PreparedsRemotePrime()

	// Since TLS listener has already been started by NewServiceEndpoint
	// So not starting here
	// Check later for enterprise -
	// server.Enterprise() && *CERT_FILE != "" && *KEY_FILE != ""

	signalCatcher(server, endpoint)
}

// signalCatcher blocks until a signal is received and then takes appropriate action
func signalCatcher(server *server_package.Server, endpoint *http.HttpEndpoint) {
	sig_chan := make(chan os.Signal, 4)
	signal.Notify(sig_chan, os.Interrupt, syscall.SIGTERM)

	var s os.Signal
	select {
	case s = <-sig_chan:
	}
	if server.CpuProfile() != "" {
		logging.Infof("Stopping CPU profile")
		pprof.StopCPUProfile()
	}
	if server.MemProfile() != "" {
		f, err := os.Create(server.MemProfile())
		if err != nil {
			logging.Errorf("Cannot create memory profile file@ %v", err)
		} else {

			logging.Infof("Writing  Memory profile")
			pprof.WriteHeapProfile(f)
			f.Close()
		}
	}
	if s == os.Interrupt {
		// Interrupt (ctrl-C) => Immediate (ungraceful) exit
		logging.Infof("Shutting down immediately")
		os.Exit(0)
	}
	logging.Infof("Attempting graceful exit")
	// Stop accepting new requests
	err := endpoint.Close()
	if err != nil {
		logging.Errorf("error closing http listener: %v", err)
	}
	err = endpoint.CloseTLS()
	if err != nil {
		logging.Errorf("error closing https listener: %v", err)
	}
}
