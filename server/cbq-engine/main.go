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
	"runtime"
	"strconv"
	"time"

	"github.com/couchbaselabs/query/accounting"
	acct_resolver "github.com/couchbaselabs/query/accounting/resolver"
	config_resolver "github.com/couchbaselabs/query/clustering/resolver"
	datastore_package "github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/datastore/resolver"
	"github.com/couchbaselabs/query/logging"
	log_resolver "github.com/couchbaselabs/query/logging/resolver"
	"github.com/couchbaselabs/query/server"
	"github.com/couchbaselabs/query/server/http"
	"github.com/couchbaselabs/query/util"
)

var VERSION = "0.7.0" // Build-time overriddable.

var DATASTORE = flag.String("datastore", "", "Datastore address (http://URL or dir:PATH or mock:)")
var CONFIGSTORE = flag.String("configstore", "stub:", "Configuration store address (http://URL or stub:)")
var ACCTSTORE = flag.String("acctstore", "gometrics:", "Accounting store address (http://URL or stub:)")
var NAMESPACE = flag.String("namespace", "default", "Default namespace")
var TIMEOUT = flag.Duration("timeout", 0*time.Second, "Server execution timeout, e.g. 500ms or 2s; use zero or negative value to disable")
var READONLY = flag.Bool("readonly", false, "Read-only mode")
var SIGNATURE = flag.Bool("signature", true, "Whether to provide signature")
var METRICS = flag.Bool("metrics", true, "Whether to provide metrics")
var REQUEST_CAP = flag.Int("request-cap", runtime.NumCPU()<<16, "Maximum number of queued requests")
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

func main() {
	flag.Parse()

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
	server, err := server.NewServer(datastore, configstore, acctstore, *NAMESPACE, *READONLY, channel,
		*THREAD_COUNT, *TIMEOUT, *SIGNATURE, *METRICS, keep_alive_length)
	if err != nil {
		logging.Errorp(err.Error())
		os.Exit(1)
	}

	go server.Serve()

	logging.Infop("cbq-engine started",
		logging.Pair{"version", VERSION},
		logging.Pair{"datastore", *DATASTORE},
	)

	endpoint := http.NewServiceEndpoint(server, *METRICS, *HTTP_ADDR)
	er := endpoint.ListenAndServe()
	if er != nil {
		logging.Errorf("cbq-engine exiting with error: %v", er)
		os.Exit(1)
	}
}
