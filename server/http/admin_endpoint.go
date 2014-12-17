//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package http

import (
	"fmt"
	"net/http"

	"github.com/couchbaselabs/query/clustering"
	"github.com/couchbaselabs/query/server"
	"github.com/gorilla/mux"
)

const (
	adminPrefix    = "/admin"
	clustersPrefix = "/admin/clusters"
)

// admin_endpoint

func registerAdminHandlers(server *server.Server) {
	r := mux.NewRouter()
	r.HandleFunc(adminPrefix+"/ping", pingHandler).
		Methods("GET")

	r.HandleFunc(adminPrefix+"/config", func(w http.ResponseWriter, req *http.Request) {
		doConfig(server.ConfigurationStore(), w, req)
	}).
		Methods("GET")

	r.HandleFunc(clustersPrefix, func(w http.ResponseWriter, req *http.Request) {
		doClusters(server.ConfigurationStore(), w, req)
	}).
		Methods("GET")

	r.HandleFunc(clustersPrefix+"/{cluster}", func(w http.ResponseWriter, req *http.Request) {
		doCluster(server.ConfigurationStore(), w, req)
	}).
		Methods("GET")

	r.HandleFunc(clustersPrefix+"/{cluster}/nodes", func(w http.ResponseWriter, req *http.Request) {
		doNodes(server.ConfigurationStore(), w, req)
	}).
		Methods("GET")

	r.HandleFunc(clustersPrefix+"/{cluster}/nodes/{node}", func(w http.ResponseWriter, req *http.Request) {
		doNode(server.ConfigurationStore(), w, req)
	}).
		Methods("GET")

	http.Handle(adminPrefix+"/", r)
}

func pingHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "ok")
}

func doConfig(cfgStore clustering.ConfigurationStore, w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "/admin/config using %s\n", cfgStore.Id())
}

func doClusters(cfgStore clustering.ConfigurationStore, w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "/admin/clusters using %s\n", cfgStore.Id())
}

func doCluster(cfgStore clustering.ConfigurationStore, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	cluster := vars["cluster"]
	fmt.Fprintf(w, "/admin/clusters/%s using %s\n", cluster, cfgStore.Id())
}

func doNodes(cfgStore clustering.ConfigurationStore, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	cluster := vars["cluster"]
	fmt.Fprintf(w, "/admin/cluster/%s/nodes using %s\n", cluster, cfgStore.Id())
}

func doNode(cfgStore clustering.ConfigurationStore, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	cluster := vars["cluster"]
	node := vars["node"]
	fmt.Fprintf(w, "/admin/cluster/%s/nodes/%s using %s\n", cluster, node, cfgStore.Id())
}
