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
	"runtime"
	"strings"
	"time"

	"github.com/couchbaselabs/query/accounting"
	"github.com/couchbaselabs/query/accounting/logger_retriever"
	acct_resolver "github.com/couchbaselabs/query/accounting/resolver"
	cfg_resolver "github.com/couchbaselabs/query/clustering/resolver"
	"github.com/couchbaselabs/query/datastore/resolver"
	"github.com/couchbaselabs/query/querylog"
	"github.com/couchbaselabs/query/server"
	"github.com/couchbaselabs/query/server/http"
)

var VERSION = "0.7.0" // Build-time overriddable.

var DATASTORE = flag.String("datastore", "", "Datastore address (http://URL or dir:PATH or mock:)")
var CONFIGSTORE = flag.String("configstore", "stub:", "Configuration store address (zookeeper:hosts or stub:)")
var ACCTSTORE = flag.String("acctstore", "stub:", "Accounting store address (stub:)")
var CLUSTER = flag.String("cluster", "default", "Default cluster")
var CREATE_CLUSTER = flag.Bool("create-cluster", false, "Whether to create the cluster")
var JOIN_CLUSTER = flag.Bool("join-cluster", true, "Whether to join the cluster")
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
var HTTPS_ADDR = flag.String("https", ":8094", "HTTPS service address")
var CERT_FILE = flag.String("certfile", "", "HTTPS certificate file")
var KEY_FILE = flag.String("keyfile", "", "HTTPS private key file")
var LOG_KEYS = flag.String("log", "", "Log keywords, comma separated")
var DEV_MODE = flag.Bool("dev", false, "Developer Mode")
var ADMIN_ADDR = flag.String("admin", ":9093", "HTTP admin service address")

var devModeDefaultLogKeys = []string{querylog.HTTP, querylog.SCAN, querylog.OPTIMIZER,
	querylog.PLANNER, querylog.PARSER, querylog.COMPILER, querylog.PIPELINE,
	querylog.ALGEBRA, querylog.DATASTORE}

var lw *logger_retriever.RetrieverLogger

func main() {
	flag.Parse()

	lw = logger_retriever.NewRetrieverLogger(devModeDefaultLogKeys)
	if lw == nil {
		fmt.Sprintf("Unable initialize default logger")
	}

	if *DEV_MODE {
		lw.SetLevel(accounting.Debug)
		lw.Debug("Developer mode enabled ")
	} else {
		// set log level to info : TODO change to warning
		// sometime before release
		lw.SetLevel(accounting.Info)
	}

	if *LOG_KEYS != "" {
		lw = logger_retriever.NewRetrieverLogger(strings.Split(*LOG_KEYS, ","))
	}

	datastore, err := resolver.NewDatastore(*DATASTORE)
	if err != nil {
		lw.Error("Error starting cbq-engine: %v", err)
		return
	}

	configstore, err := cfg_resolver.NewConfigstore(*CONFIGSTORE)
	if err != nil {
		lw.Error("Could not connect to config store: %v", err)
	}

	acctstore, err := acct_resolver.NewAcctstore(*ACCTSTORE)
	if err != nil {
		lw.Error("Could not connect to accounting store: %v", err)
	}

	clusterConfig, err := cfg_resolver.NewClusterConfig(*CONFIGSTORE, *CLUSTER, configstore, datastore, acctstore)
	lw.Info("cluster config: %s", clusterConfig)
	queryNodeConfig, err := cfg_resolver.NewQueryNodeConfig(*CONFIGSTORE, *CLUSTER, VERSION, *HTTP_ADDR, *ADMIN_ADDR, configstore, datastore, acctstore)
	lw.Info("query node config: %s", queryNodeConfig)

	if *CREATE_CLUSTER && configstore != nil { // TODO: "configstore != nil" is code for "connected to configstore". Encapsulate this.
		// Create the cluster
		cfm := configstore.ConfigurationManager()
		clusterConfig, err = cfm.AddCluster(clusterConfig)
		if err != nil {
			lw.Error("Could not add cluster: %v", err)
		} else {
			lw.Info("Created cluster %s", clusterConfig)
		}
	}

	if *JOIN_CLUSTER && configstore != nil && clusterConfig != nil { // TODO: "configstore != nil && clusterConfig != nil" is code for "connected to configstore". Encapsulate this.
		// Attempt to join the cluster
		cm := clusterConfig.ClusterManager()
		queryNodeConfig, err = cm.AddQueryNode(queryNodeConfig)
		if err != nil {
			lw.Error("Could not add query node: %v", err)
		} else {
			lw.Info("Created query node %s", queryNodeConfig)
		}
	}

	channel := make(server.RequestChannel, *REQUEST_CAP)
	server, err := server.NewServer(datastore, *NAMESPACE, *READONLY, channel,
		*THREAD_COUNT, *TIMEOUT, *SIGNATURE, *METRICS)
	if err != nil {
		lw.Error("Error starting cbq-engine: %v", err)
		return
	}

	go server.Serve()

	lw.Info("cbq-engine started...")
	lw.Info("version: %s", VERSION)
	lw.Info("datastore: %s", *DATASTORE)
	lw.Info("http address: %s", *HTTP_ADDR)

	endpoint := http.NewHttpEndpoint(server, *METRICS, *HTTP_ADDR)
	er := endpoint.ListenAndServe()
	if er != nil {
		lw.Error("cbq-engine exiting with error: %v", er)
	}
}
