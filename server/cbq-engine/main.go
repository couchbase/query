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
	"time"

	config_resolver "github.com/couchbaselabs/query/clustering/resolver"
	"github.com/couchbaselabs/query/datastore/resolver"
	"github.com/couchbaselabs/query/logging"
	log_resolver "github.com/couchbaselabs/query/logging/resolver"
	"github.com/couchbaselabs/query/server"
	"github.com/couchbaselabs/query/server/http"
)

var VERSION = "0.7.0" // Build-time overriddable.

var DATASTORE = flag.String("datastore", "", "Datastore address (http://URL or dir:PATH or mock:)")
var CONFIGSTORE = flag.String("configstore", "stub:", "Configuration store address (http://URL or stub:)")
var NAMESPACE = flag.String("namespace", "default", "Default namespace")
var TIMEOUT = flag.Duration("timeout", 0*time.Second, "Server execution timeout; use zero or negative value to disable")
var READONLY = flag.Bool("readonly", false, "Read-only mode")
var SIGNATURE = flag.Bool("signature", true, "Whether to provide signature")
var METRICS = flag.Bool("metrics", true, "Whether to provide metrics")
var REQUEST_CAP = flag.Int("request-cap", runtime.NumCPU()<<16, "Maximum number of queued requests")
var THREAD_COUNT = flag.Int("threads", runtime.NumCPU()<<6, "Thread count")
var ORDER_LIMIT = flag.Int64("order-limit", 0, "Maximum LIMIT for ORDER BY clauses; use zero or negative value to disable")
var UPDATE_LIMIT = flag.Int64("update-limit", 0, "Maximum LIMIT for data modification statements; use zero or negative value to disable")
var HTTP_ADDR = flag.String("http", ":8093", "HTTP service address")
var HTTPS_ADDR = flag.String("https", ":18093", "HTTPS service address")
var CERT_FILE = flag.String("certfile", "", "HTTPS certificate file")
var KEY_FILE = flag.String("keyfile", "", "HTTPS private key file")
var LOGGER = flag.String("logger", "", "Logger implementation")
var DEBUG = flag.Bool("debug", false, "Debug mode")

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

	configstore, err := config_resolver.NewConfigstore(*CONFIGSTORE)
	if err != nil {
		logging.Errorp("Could not connect to configstore",
			logging.Pair{"error", err},
		)
	}

	channel := make(server.RequestChannel, *REQUEST_CAP)
	server, err := server.NewServer(datastore, configstore, *NAMESPACE, *READONLY, channel,
		*THREAD_COUNT, *TIMEOUT, *SIGNATURE, *METRICS)
	if err != nil {
		logging.Errorp(err.Error())
		os.Exit(1)
	}

	go server.Serve()

	logging.Infop("cbq-engine started",
		logging.Pair{"version", VERSION},
		logging.Pair{"datastore", *DATASTORE},
	)

	endpoint := http.NewHttpEndpoint(server, *METRICS, *HTTP_ADDR)
	er := endpoint.ListenAndServe()
	if er != nil {
		logging.Errorf("cbq-engine exiting with error: %v", er)
		os.Exit(1)
	}
}
