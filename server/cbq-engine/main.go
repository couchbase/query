//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"sync"
	"syscall"
	"time"

	//	go_http "net/http"
	//	_ "net/http/pprof"

	"github.com/couchbase/query/accounting"
	acct_resolver "github.com/couchbase/query/accounting/resolver"
	"github.com/couchbase/query/audit"
	"github.com/couchbase/query/aus"
	config_resolver "github.com/couchbase/query/clustering/resolver"
	"github.com/couchbase/query/datastore"
	datastore_package "github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/resolver"
	"github.com/couchbase/query/datastore/system"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/ffdc"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/functions/constructor"
	"github.com/couchbase/query/functions/storage"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/logging/event"
	log_resolver "github.com/couchbase/query/logging/resolver"
	"github.com/couchbase/query/memory"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/scheduler"
	server_package "github.com/couchbase/query/server"
	control "github.com/couchbase/query/server/control/couchbase"
	"github.com/couchbase/query/server/http"
	queryMetakv "github.com/couchbase/query/server/settings/couchbase"
	stats "github.com/couchbase/query/system"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

// this function must be the first executed so as to clearly delineate cbq-engine start in the query.log
func init() {
	os.Stderr.WriteString("\n")
	logging.Infoa(func() string {
		return fmt.Sprintf("cbq-engine starting version=%v go-version=%s pid=%d", util.VERSION, runtime.Version(), os.Getpid())
	})
	debug.SetGCPercent(_GOGC_PERCENT_DEFAULT)
	setOpenFilesLimit()
}

const (
	_DEF_HTTP                   = ":8093"
	_DEF_HTTPS                  = ":18093"
	_DEF_REQUEST_CAP            = 256
	_DEF_SCAN_CAP               = 512
	_DEF_PIPELINE_CAP           = 512
	_DEF_PIPELINE_BATCH         = 16
	_DEF_COMPLETED_THRESHOLD    = 1000
	_DEF_COMPLETED_THRESHOLD_SL = 5000
	_DEF_COMPLETED_LIMIT        = 4000
	_DEF_PREPARED_LIMIT         = 16384
	_DEF_FUNCTIONS_LIMIT        = 16384
	_DEF_DICTIONARY_CACHE_LIMIT = 16384
	_DEF_TASKS_LIMIT            = 16384
	_DEF_MEMORY_QUOTA           = 0
	_DEF_NODE_QUOTA             = 0
	_DEF_NODE_QUOTA_VAL_PERCENT = 67
	_DEF_CE_MAXCPUS             = 4
	_DEF_REQUEST_ERROR_LIMIT    = errors.DEFAULT_REQUEST_ERROR_LIMIT
	_DEF_SEQSCAN_KEYS           = 10000
)

var DATASTORE = flag.String("datastore", "", "Datastore address (http://URL or dir:PATH or mock:)")
var CONFIGSTORE = flag.String("configstore", "stub:", "Configuration store address (http://URL or stub:)")
var ACCTSTORE = flag.String("acctstore", "gometrics:", "Accounting store address (http://URL or stub:)")
var NAMESPACE = flag.String("namespace", "default", "Default namespace")
var TIMEOUT = flag.Duration("timeout", 0*time.Second,
	"Server execution timeout, e.g. 500ms or 2s; use zero or negative value to disable")
var TXTIMEOUT = flag.Duration("txtimeout", 0*time.Second,
	"Maximum Transaction timeout, e.g. 2m or 2s; use zero or negative to use request level value")
var READONLY = flag.Bool("readonly", false, "Read-only mode")
var SIGNATURE = flag.Bool("signature", true, "Whether to provide signature")
var METRICS = flag.Bool("metrics", true, "Whether to provide metrics")
var PRETTY = flag.Bool("pretty", false, "Pretty output")
var REQUEST_CAP = flag.Int("request-cap", _DEF_REQUEST_CAP, "Maximum number of queued requests per logical CPU")
var REQUEST_SIZE_CAP = flag.Int("request-size-cap", server_package.MAX_REQUEST_SIZE, "Maximum size of a request")
var REQUEST_ERROR_LIMIT = flag.Int("request-error-limit", _DEF_REQUEST_ERROR_LIMIT,
	"Maximum number of errors to accumulate before aborting a request")
var SCAN_CAP = flag.Int64("scan-cap", _DEF_SCAN_CAP, "Maximum buffer size for index scans; use zero or negative value to disable")
var SERVICERS = flag.Int("servicers", 0, "Servicer count")
var PLUS_SERVICERS = flag.Int("plus-servicers", 0, "Plus servicer count")
var MAX_PARALLELISM = flag.Int("max-parallelism", 1, "Maximum parallelism per query; use zero or negative value to use maximum")
var ORDER_LIMIT = flag.Int64("order-limit", 0, "Maximum LIMIT for ORDER BY clauses; use zero or negative value to disable")
var MUTATION_LIMIT = flag.Int64("mutation-limit", 0,
	"Maximum LIMIT for data modification statements; use zero or negative value to disable")
var HTTP_ADDR = flag.String("http", _DEF_HTTP, "HTTP service address")
var HTTPS_ADDR = flag.String("https", _DEF_HTTPS, "HTTPS service address")

var CA_FILE = flag.String("cafile", "", "HTTPS CA certificates")
var CERT_FILE = flag.String("certfile", "", "HTTPS certificate chain file")
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
var NODE_QUOTA = flag.Uint64("node-quota", _DEF_NODE_QUOTA, "Soft memory limit per node, in MB")
var USE_REPLICA = flag.String("use-replica", value.TRISTATE_NAMES[value.NONE], "Allow reading from replica vBuckets")
var NODE_QUOTA_VAL_PERCENT = flag.Uint("node-quota-val-percent", _DEF_NODE_QUOTA_VAL_PERCENT,
	"Percentage of node quota reserved for value memory (0-100)")

// cpu and memory profiling flags
var CPU_PROFILE = flag.String("cpuprofile", "", "write cpu profile to file")
var MEM_PROFILE = flag.String("memprofile", "", "write memory profile to this file")

// Monitoring API
var COMPLETED_THRESHOLD = flag.Int("completed-threshold", -1,
	"cache completed query lasting longer than this many milliseconds")
var COMPLETED_LIMIT = flag.Int("completed-limit", _DEF_COMPLETED_LIMIT, "maximum number of completed requests")

var PREPARED_LIMIT = flag.Int("prepared-limit", _DEF_PREPARED_LIMIT, "maximum number of prepared statements")
var AUTO_PREPARE = flag.Bool("auto-prepare", false, "Silently prepare ad hoc statements if possible")

var FUNCTIONS_LIMIT = flag.Int("functions-limit", _DEF_FUNCTIONS_LIMIT, "maximum number of cached functions")
var TASKS_LIMIT = flag.Int("tasks-limit", _DEF_TASKS_LIMIT, "maximum number of cached tasks")

// GOGC
var _GOGC_PERCENT_DEFAULT = 200
var _GOGC_PERCENT = flag.Int("gc-percent", _GOGC_PERCENT_DEFAULT, "Go runtime garbage collection target percentage")

var UUID = flag.String("uuid", "", "Node UUID.")

var INTERNAL_CLIENT_CERT = flag.String("clientCertFile", "", "Internal communications certificate")
var INTERNAL_CLIENT_KEY = flag.String("clientKeyFile", "", "Internal communications private key")

// profiler, to use instead of the REST endpoint if needed
// var PROFILER_PORT = flag.Int("profiler-port", 6060, "profiler listening port")

// Dictionary Cache
var DICTIONARY_CACHE_LIMIT = flag.Int("dictionary-cache-limit", _DEF_DICTIONARY_CACHE_LIMIT,
	"maximum number of entries in dictionary cache")

var DEPLOYMENT_MODEL = flag.String("deploymentModel", "default", "Deployment Model: default, serverless, provisioned")

var REGULATOR_SETTINGS_FILE = flag.String("regulatorSettingsFile", "", "Regulator settings file")

func main() {

	HideConsole(true)
	defer HideConsole(false)
	flag.Parse()

	initialCfg, num_cpus := waitForInitialSettings()

	// many Init() depend on this

	tenant.Init(*DEPLOYMENT_MODEL == datastore.DEPLOYMENT_MODEL_SERVERLESS)

	memory.SetMemoryLimitFunction(setMemoryLimit)
	memory.Config(*NODE_QUOTA, *NODE_QUOTA_VAL_PERCENT, []int{*SERVICERS, *PLUS_SERVICERS})
	tenant.Config(memory.Quota())

	max_cpus := runtime.NumCPU()
	if !*ENTERPRISE {
		max_cpus = _DEF_CE_MAXCPUS
	}
	numProcs := util.SetNumCPUs(max_cpus, num_cpus, tenant.IsServerless())

	ffdc.Init()

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
		filter := ""

		if *LOG_LEVEL != "" {
			lvl, ok, f := logging.ParseLevel(*LOG_LEVEL)
			if ok {
				level = lvl
				filter = f
			}
		}

		logging.SetLevel(level)
		logging.SetDebugFilter(filter)
	}

	resolver.SetDeploymentModel(*DATASTORE, *DEPLOYMENT_MODEL)
	// default until settings adjust
	util.SetTemp(os.TempDir(), 0)

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
	configstore, err := config_resolver.NewConfigstore(*CONFIGSTORE, *UUID)
	if err != nil {
		logging.Errorf("Could not connect to configstore: %v", err)
	}
	// configstore options must be set immediately after creation as other start-up operations will depend on them being set
	configstore.SetOptions(*HTTP_ADDR, *HTTPS_ADDR, (*HTTP_ADDR == _DEF_HTTP && *HTTPS_ADDR == _DEF_HTTPS))

	// ditto for distributed access for monitoring
	// also distributed is used by many init() functions and should be done as early as possible
	prof, ctrl, err := monitoringInit(configstore)
	if err != nil {
		logging.Errorf("%v", err.Error())
		fmt.Printf("\n%v\n", err)
		os.Exit(1)
	}

	// needs to be before any events may be generated
	event.Init(configstore)

	acctstore, err := acct_resolver.NewAcctstore(*ACCTSTORE)
	if err != nil {
		logging.Errorf("Could not connect to acctstore: %v", err)
	} else {
		// Create the metrics we are interested in
		accounting.RegisterMetrics(acctstore)
		// Make metrics available
		acctstore.MetricReporter().Start(1, 1)
	}

	// Start the completed requests log
	if *COMPLETED_THRESHOLD == -1 {
		if tenant.IsServerless() {
			*COMPLETED_THRESHOLD = _DEF_COMPLETED_THRESHOLD_SL
		} else {
			*COMPLETED_THRESHOLD = _DEF_COMPLETED_THRESHOLD
		}
	}
	server_package.RequestsInit(*COMPLETED_THRESHOLD, *COMPLETED_LIMIT, _DEF_SEQSCAN_KEYS)

	// Initialized the prepared statement cache
	if *PREPARED_LIMIT <= 0 {
		logging.Errorf("Ignoring invalid prepared statement cache size: %v", *PREPARED_LIMIT)
		*PREPARED_LIMIT = _DEF_PREPARED_LIMIT
	}
	prepareds.PreparedsInit(*PREPARED_LIMIT)
	functions.FunctionsInit(*FUNCTIONS_LIMIT, storage.UseSystemStorage)
	scheduler.SchedulerSetLimit(*TASKS_LIMIT)

	if *DICTIONARY_CACHE_LIMIT <= 0 {
		logging.Errorf("Ignoring invalid dictionary cache size: %v", *DICTIONARY_CACHE_LIMIT)
		*DICTIONARY_CACHE_LIMIT = _DEF_DICTIONARY_CACHE_LIMIT
	}

	// Initialize dictionary cache
	server_package.InitDictionaryCache(*DICTIONARY_CACHE_LIMIT)

	sys, err := system.NewDatastore(datastore, acctstore)
	if err != nil {
		logging.Errorf("%v", err.Error())
		os.Exit(1)
	}

	/*
	 * Do not adjust cluster settings after the server has been created.  Adjust initialCfg beforehand instead if necessary.
	 */
	server, err := server_package.NewServer(datastore, sys, configstore, acctstore, *NAMESPACE,
		*READONLY, *REQUEST_CAP*numProcs, *REQUEST_CAP*numProcs, *SERVICERS, *PLUS_SERVICERS,
		*MAX_PARALLELISM, *TIMEOUT, *SIGNATURE, *METRICS, *ENTERPRISE,
		*PRETTY, prof, ctrl, initialCfg)
	if err != nil {
		logging.Errorf("%v", err.Error())
		os.Exit(1)
	}

	memory.Config(memory.NodeQuota(), memory.ValPercent(), []int{server.Servicers(), server.PlusServicers()})
	datastore_package.SetSystemstore(server.Systemstore())
	prepareds.PreparedsReprepareInit(datastore, sys)

	// only non cluster-setting options
	server.SetCpuProfile(*CPU_PROFILE)
	server.SetKeepAlive(*KEEP_ALIVE_LENGTH)
	server.SetMemProfile(*MEM_PROFILE)
	server.SetRequestSizeCap(*REQUEST_SIZE_CAP)
	server.SetMaxIndexAPI(*MAX_INDEX_API)
	server.SetAutoPrepare(*AUTO_PREPARE)
	server.SetGCPercent(*_GOGC_PERCENT)
	server.SetRequestErrorLimit(*REQUEST_ERROR_LIMIT)

	audit.StartAuditService(*DATASTORE, server.Servicers()+server.PlusServicers())

	// report any non cluster-setting options
	logging.Infoa(func() string {
		return fmt.Sprintf("cbq-engine started"+
			" version=%v"+
			" ds_version=%v"+
			" datastore=%v"+
			" max-concurrency=%v"+
			" servicers=%v"+
			" plus-servicers=%v"+
			" request-cap=%v"+
			" request-size-cap=%v"+
			" max-index-api=%v"+
			" gc-percent=%v",
			util.VERSION,
			datastore.Info().Version(),
			*DATASTORE,
			numProcs,
			server.Servicers(),
			server.PlusServicers(),
			*REQUEST_CAP,
			server.RequestSizeCap(),
			server.MaxIndexAPI(),
			*_GOGC_PERCENT,
		)
	})

	server_package.InitAWR() // start before endpoints but after server init

	// Create http endpoint (but don't start it yet)
	endpoint := http.NewServiceEndpoint(server, *STATIC_PATH, *METRICS,
		*HTTP_ADDR, *HTTPS_ADDR, *CA_FILE, *CERT_FILE, *KEY_FILE, *INTERNAL_CLIENT_CERT, *INTERNAL_CLIENT_KEY)

	ffdc.Set(ffdc.Completed, http.CaptureCompletedRequests)
	ffdc.Set(ffdc.Active, func(w io.Writer) error {
		return http.CaptureActiveRequests(endpoint, w)
	})
	ffdc.Set(ffdc.Vitals, func(w io.Writer) error {
		return http.CaptureVitals(endpoint, w)
	})

	server.SetSettingsCallback(endpoint.SettingsCallback)

	constructor.Init(endpoint.Router(), server.Servicers(), "")
	tenant.Start(endpoint, *UUID, *REGULATOR_SETTINGS_FILE)

	// Since TLS listener has already been started by NewServiceEndpoint
	// So not starting here
	// Check later for enterprise -
	// server.Enterprise() && *CERT_FILE != "" && *KEY_FILE != ""

	// start the endpoint once all initialisation is complete
	er := endpoint.Listen()
	if er != nil {
		logging.Errorf("cbq-engine (HTTP_ADDR %v) exiting with error: %v", *HTTP_ADDR, er)
		os.Exit(1)
	}

	er = endpoint.SetupSSL()
	if er != nil {
		logging.Errorf("Error with Setting up SSL endpoints : %v", err.Error())
		os.Exit(1)
	}

	// topology awareness - after listeners are ready to handle requests
	_ = control.NewManager(*UUID)

	// Now that we are up and running, try to prime the prepareds cache
	prepareds.PreparedsRemotePrime()

	// migrations (functions storage and CBO stats) last
	storage.Migrate()
	server_package.MigrateDictionary()

	// Initialize configurations for AUS
	aus.InitAus(server)

	signalCatcher(server, endpoint)
}

// signalCatcher blocks until a signal is received and then takes appropriate action
func signalCatcher(server *server_package.Server, endpoint *http.HttpEndpoint) {
	sig_chan := make(chan os.Signal, 4)
	signal.Notify(sig_chan, os.Interrupt, syscall.SIGTERM, util.SIGCONT)

	var s os.Signal
	for {
		select {
		case s = <-sig_chan:
			if scs, ok := s.(syscall.Signal); ok {
				logging.Infof("Received signal: %d - %s", scs, scs.String())
			} else {
				logging.Infof("Received signal: %v", s)
			}
		}
		if s == util.SIGCONT {
			util.ResyncTime()
			logging.Infof("Resuming cbq-engine process")
		} else {
			break
		}
	}
	if server.CpuProfile() != "" {
		logging.Infof("Stopping CPU profile")
		pprof.StopCPUProfile()
	}
	if server.MemProfile() != "" {
		f, err := os.Create(server.MemProfile())
		if err != nil {
			logging.Errorf("Cannot create memory profile file: %v", err)
		} else {
			logging.Infof("Writing Memory profile")
			pprof.WriteHeapProfile(f)
			f.Close()
		}
	}
	if s == os.Interrupt {
		// Interrupt (ctrl-C) => Immediate (ungraceful) exit
		logging.Infof("Shutting down immediately")
		os.Exit(0)
	}
	ffdc.Capture(ffdc.SigTerm)
	// graceful shutdown on SIGTERM
	server.InitiateShutdownAndWait("SIGTERM")
	os.Exit(0)
}

const _MEMORY_LIMIT = 0.9
const _MAX_MEMORY_ABOVE_LIMIT = 8 * util.GiB
const _PER_SERVICER_MIN_MEMORY = 128 * util.MiB
const _MIN_MEMORY_LIMIT = util.GiB

func setMemoryLimit(ml int64) {
	var extra string
	var oml int64

	// the minimum and maximum permitted can be overridden by using the GOMEMLIMIT environment variable
	max := int64(math.MaxInt64)
	// this is based on the default number of servicers and doesn't change with updates to the Server object's settings
	min := int64(util.NumCPU() * server_package.SERVICERS_MULTIPLIER * _PER_SERVICER_MIN_MEMORY)
	if min < _MIN_MEMORY_LIMIT {
		min = _MIN_MEMORY_LIMIT
	}
	if min > max {
		max = min
	}

	ss, err := stats.NewSystemStats()
	if err == nil {
		defer ss.Close()
		t, err := ss.SystemTotalMem()
		if err == nil {
			max = int64(float64(t) * _MEMORY_LIMIT)
			if int64(t)-max > _MAX_MEMORY_ABOVE_LIMIT {
				max = int64(t) - _MAX_MEMORY_ABOVE_LIMIT
				extra = fmt.Sprintf("(%.0f%% of total)", (float64(max)/float64(t))*100)
			} else {
				extra = fmt.Sprintf("(%.0f%% of total)", _MEMORY_LIMIT*100)
			}
		}
	}

	if os.Getenv("GOMEMLIMIT") != "" {
		extra = "(GOMEMLIMIT)"
		oml = -1
		ml = debug.SetMemoryLimit(-1)
	} else if ml > 0 {
		if ml > max {
			ml = max
			extra = "(NODE QUOTA - LIMITED)"
		} else if ml < min {
			ml = min
			extra = "(NODE QUOTA - LIMITED)"
		} else {
			extra = "(NODE QUOTA)"
		}
		oml = debug.SetMemoryLimit(ml)
	} else {
		ml = max
		oml = debug.SetMemoryLimit(ml)
	}
	if oml != ml {
		logging.Infoa(func() string {
			return fmt.Sprintf("Soft memory limit: %s %v", logging.HumanReadableSize(ml, false), extra)
		})
	}
}

func waitForInitialSettings() (queryMetakv.Config, int) {
	var cfg queryMetakv.Config
	var wg sync.WaitGroup
	num_cpus := 0

	stop := make(chan struct{})
	callb := func(arg queryMetakv.Config) {
		cfg = arg
		wg.Done()
		close(stop)
	}
	wg.Add(1)
	logging.Infof("Waiting for initial settings")
	queryMetakv.SetupSettingsNotifier(callb, stop)
	wg.Wait()
	logging.Infof("Initial settings received")

	if v, ok := cfg.Field("num-cpus"); ok {
		num_cpus = int(value.AsNumberValue(v).Int64())
	}

	return cfg, num_cpus
}
