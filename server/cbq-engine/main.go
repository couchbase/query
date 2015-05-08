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
	"runtime/pprof"
	"strconv"
	"syscall"
	"time"

	"github.com/couchbase/query/accounting"
	acct_resolver "github.com/couchbase/query/accounting/resolver"
	config_resolver "github.com/couchbase/query/clustering/resolver"
	datastore_package "github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/resolver"
	"github.com/couchbase/query/logging"
	log_resolver "github.com/couchbase/query/logging/resolver"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/server/http"
	"github.com/couchbase/query/util"
)

var DATASTORE = flag.String("datastore", "", "Datastore address (http://URL or dir:PATH or mock:)")
var CONFIGSTORE = flag.String("configstore", "stub:", "Configuration store address (http://URL or stub:)")
var ACCTSTORE = flag.String("acctstore", "gometrics:", "Accounting store address (http://URL or stub:)")
var NAMESPACE = flag.String("namespace", "default", "Default namespace")
var TIMEOUT = flag.Duration("timeout", 0*time.Second, "Server execution timeout, e.g. 500ms or 2s; use zero or negative value to disable")
var READONLY = flag.Bool("readonly", false, "Read-only mode")
var SIGNATURE = flag.Bool("signature", true, "Whether to provide signature")
var METRICS = flag.Bool("metrics", true, "Whether to provide metrics")
var REQUEST_CAP = flag.Int("request-cap", runtime.NumCPU()<<16, "Maximum number of queued requests")
var REQUEST_SIZE_CAP = flag.String("request-size-cap", http.MAX_REQUEST_SIZE, "Maximum size of a request")
var SCAN_CAP = flag.Int64("scan-cap", 0, "Maximum buffer size for primary index scans; use zero or negative value to disable")
var THREAD_COUNT = flag.Int("threads", runtime.NumCPU()<<6, "Thread count")
var ORDER_LIMIT = flag.Int64("order-limit", 0, "Maximum LIMIT for ORDER BY clauses; use zero or negative value to disable")
var MUTATION_LIMIT = flag.Int64("mutation-limit", 0, "Maximum LIMIT for data modification statements; use zero or negative value to disable")
var HTTP_ADDR = flag.String("http", ":8093", "HTTP service address")
var HTTPS_ADDR = flag.String("https", ":18093", "HTTPS service address")
var CERT_FILE = flag.String("certfile", "", "HTTPS certificate file")
var KEY_FILE = flag.String("keyfile", "", "HTTPS private key file")
var LOGGER = flag.String("logger", "", "Logger implementation")
var DEBUG = flag.Bool("debug", false, "Debug mode")
var KEEP_ALIVE_LENGTH = flag.String("keep-alive-length", strconv.Itoa(server.KEEP_ALIVE_DEFAULT), "maximum size of buffered result")
var STATIC_PATH = flag.String("static-path", "static", "Path to static content")

//cpu and memory profiling flags
var CPU_PROFILE = flag.String("cpuprofile", "", "write cpu profile to file")
var MEM_PROFILE = flag.String("memprofile", "", "write memory profile to this file")

func main() {
	flag.Parse()

	var f *os.File
	if *MEM_PROFILE != "" {
		var err error
		f, err = os.Create(*MEM_PROFILE)
		if err != nil {
			fmt.Printf("Cannot start mem profiler %v\n", err)
		}
	}

	if *CPU_PROFILE != "" {
		f, err := os.Create(*CPU_PROFILE)
		if err != nil {
			fmt.Printf("Cannot start cpu profiler %v\n", err)
		} else {
			pprof.StartCPUProfile(f)
		}
	}

	if *LOGGER != "" {
		logger, _ := log_resolver.NewLogger(*LOGGER)
		if logger == nil {
			fmt.Printf("Invalid logger: %s\n", *LOGGER)
			os.Exit(1)
		}

		logging.SetLogger(logger)
	}

	if *DEBUG {
		logging.SetLevel(logging.Debug)
		logging.Debugp("Debug mode enabled")
	} else {
		logging.SetLevel(logging.Info)
	}

	datastore, err := resolver.NewDatastore(*DATASTORE)
	if err != nil {
		logging.Errorp(err.Error())
		os.Exit(1)
	}
	datastore_package.SetDatastore(datastore)
	datastore_package.SetScanCap(*SCAN_CAP)

	configstore, err := config_resolver.NewConfigstore(*CONFIGSTORE)
	if err != nil {
		logging.Errorp("Could not connect to configstore",
			logging.Pair{"error", err},
		)
	}
	acctstore, err := acct_resolver.NewAcctstore(*ACCTSTORE)
	if err != nil {
		logging.Errorp("Could not connect to acctstore",
			logging.Pair{"error", err},
		)
	} else {
		// Create the metrics we are interested in
		accounting.RegisterMetrics(acctstore)
		// Make metrics available
		acctstore.MetricReporter().Start(1, 1)
	}

	keep_alive_length, e := util.ParseQuantity(*KEEP_ALIVE_LENGTH)

	if e != nil {
		logging.Errorp("Error parsing keep alive length; reverting to default",
			logging.Pair{"keep alive length", *KEEP_ALIVE_LENGTH},
			logging.Pair{"error", e},
			logging.Pair{"default", server.KEEP_ALIVE_DEFAULT},
		)
	}

	if e == nil && keep_alive_length < 1 {
		logging.Infop("Negative or zero keep alive length; reverting to default",
			logging.Pair{"keep alive length", *KEEP_ALIVE_LENGTH},
			logging.Pair{"default", server.KEEP_ALIVE_DEFAULT},
		)
	}

	channel := make(server.RequestChannel, *REQUEST_CAP)
	http.SetRequestSizeCap(*REQUEST_SIZE_CAP)
	server, err := server.NewServer(datastore, configstore, acctstore, *NAMESPACE, *READONLY, channel,
		*THREAD_COUNT, *TIMEOUT, *SIGNATURE, *METRICS, keep_alive_length)
	if err != nil {
		logging.Errorp(err.Error())
		os.Exit(1)
	}

	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	go server.Serve()

	logging.Infop("cbq-engine started",
		logging.Pair{"version", util.VERSION},
		logging.Pair{"datastore", *DATASTORE},
		logging.Pair{"threads", runtime.GOMAXPROCS(0)},
	)

	// Create http endpoint
	endpoint := http.NewServiceEndpoint(server, *STATIC_PATH, *METRICS,
		*HTTP_ADDR, *HTTPS_ADDR, *CERT_FILE, *KEY_FILE)
	er := endpoint.Listen()
	if er != nil {
		logging.Errorp("cbq-engine exiting with error",
			logging.Pair{"error", er},
			logging.Pair{"HTTP_ADDR", *HTTP_ADDR},
		)
		os.Exit(1)
	}
	if *CERT_FILE != "" && *KEY_FILE != "" {
		er := endpoint.ListenTLS()
		if er != nil {
			logging.Errorp("cbq-engine exiting with error",
				logging.Pair{"error", er},
				logging.Pair{"HTTPS_ADDR", *HTTPS_ADDR},
			)
			os.Exit(1)
		}
	}
	signalCatcher(server, endpoint, *CPU_PROFILE != "", f)
}

// signalCatcher blocks until a signal is recieved and then takes appropriate action
func signalCatcher(server *server.Server, endpoint *http.HttpEndpoint, writeCPUprof bool, f *os.File) {
	sig_chan := make(chan os.Signal, 4)
	signal.Notify(sig_chan, os.Interrupt, syscall.SIGTERM)

	var s os.Signal
	select {
	case s = <-sig_chan:
	}
	if writeCPUprof {
		logging.Infop("Stopping CPU profiling...")
		pprof.StopCPUProfile()
	}
	if f != nil {
		logging.Infop("Stopping Memory profiling...")
		pprof.WriteHeapProfile(f)
		f.Close()
	}
	if s == os.Interrupt {
		// Interrupt (ctrl-C) => Immediate (ungraceful) exit
		logging.Infop("cbq-engine shutting down immediately...")
		os.Exit(0)
	}
	logging.Infop("cbq-engine attempting graceful...")
	// Stop accepting new requests
	err := endpoint.Close()
	if err != nil {
		logging.Errorp("error closing http listener", logging.Pair{"err", err})
	}
	err = endpoint.CloseTLS()
	if err != nil {
		logging.Errorp("error closing https listener", logging.Pair{"err", err})
	}
	// TODO: wait until server requests have all completed
}
