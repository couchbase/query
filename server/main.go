//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package server

import (
	"flag"
	"fmt"
	"runtime"
	"time"

	"github.com/couchbaselabs/clog"
	"github.com/couchbaselabs/query/datastore/resolver"
)

var VERSION = "0.7.0" // Build-time overriddable.

var DATASTORE = flag.String("datastore", "", "Datastore address (http://...) or dir:PATH")
var NAMESPACE = flag.String("namespace", "default", "Default namespace")
var TIMEOUT = flag.Duration("timeout", 0*time.Second, "Server timeout; zero or negative value disables server timeout")
var QUEUE_MAX = flag.Int("queue", runtime.NumCPU()<<16, "Maximum number of queued requests")
var THREAD_COUNT = flag.Int("threads", runtime.NumCPU()<<6, "Thread count")
var READONLY = flag.Bool("readonly", false, "Read-only mode")
var HTTP_ADDR = flag.String("http", ":8093", "HTTP listen address")
var HTTPS_ADDR = flag.String("https", ":8094", "HTTPS listen address")
var CERT_FILE = flag.String("certfile", "", "HTTPS certificate file")
var KEY_FILE = flag.String("keyfile", "", "HTTPS private key file")

func main() {
	flag.Parse()
	store, err := resolver.NewDatastore(*DATASTORE)
	if err != nil {
		clog.Log(fmt.Sprintf("Error starting cbq-engine: %v", err))
		return
	}

	channel := make(RequestChannel, *QUEUE_MAX)
	server, err := NewServer(store, *NAMESPACE, *READONLY, channel, *THREAD_COUNT, *TIMEOUT)
	if err != nil {
		clog.Log(fmt.Sprintf("Error starting cbq-engine: %v", err))
		return
	}

	server.Serve()

	clog.Log("cbq-engine started...")
	clog.Log("version: %s", VERSION)
	clog.Log("datastore: %s", *DATASTORE)

	/*
		receptor := NewHttpReceptor(server, *HTTP_ADDR)
		receptor.ListenAndServe()

	*/
}
