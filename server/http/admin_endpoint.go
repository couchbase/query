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
	"encoding/json"
	"net/http"
	"sync"

	"github.com/couchbaselabs/query/clustering"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/server"
	"github.com/couchbaselabs/query/util"
	"github.com/gorilla/mux"
)

const (
	adminPrefix    = "/admin"
	clustersPrefix = "/admin/clusters"
)

type apiFunc func(clustering.ConfigurationStore, http.ResponseWriter, *http.Request) (interface{}, errors.Error)

type handlerFunc func(http.ResponseWriter, *http.Request)

// admin_endpoint

func registerAdminHandlers(server *server.Server) {
	r := mux.NewRouter()

	pingHandler := func(w http.ResponseWriter, req *http.Request) {
		wrapAPI(server.ConfigurationStore(), w, req, doPing)
	}
	configHandler := func(w http.ResponseWriter, req *http.Request) {
		wrapAPI(server.ConfigurationStore(), w, req, doConfig)
	}
	clustersHandler := func(w http.ResponseWriter, req *http.Request) {
		wrapAPI(server.ConfigurationStore(), w, req, doClusters)
	}
	clusterHandler := func(w http.ResponseWriter, req *http.Request) {
		wrapAPI(server.ConfigurationStore(), w, req, doCluster)
	}
	nodesHandler := func(w http.ResponseWriter, req *http.Request) {
		wrapAPI(server.ConfigurationStore(), w, req, doNodes)
	}
	nodeHandler := func(w http.ResponseWriter, req *http.Request) {
		wrapAPI(server.ConfigurationStore(), w, req, doNode)
	}
	routeMap := map[string]struct {
		handler handlerFunc
		methods []string
	}{
		adminPrefix + "/ping":                      {handler: pingHandler, methods: []string{"GET"}},
		adminPrefix + "/config":                    {handler: configHandler, methods: []string{"GET"}},
		clustersPrefix:                             {handler: clustersHandler, methods: []string{"GET", "POST"}},
		clustersPrefix + "/{cluster}":              {handler: clusterHandler, methods: []string{"GET", "PUT", "DELETE"}},
		clustersPrefix + "/{cluster}/nodes":        {handler: nodesHandler, methods: []string{"GET", "POST"}},
		clustersPrefix + "/{cluster}/nodes/{node}": {handler: nodeHandler, methods: []string{"GET", "PUT", "DELETE"}},
	}

	for route, h := range routeMap {
		r.HandleFunc(route, h.handler).Methods(h.methods...)
	}

	http.Handle(adminPrefix+"/", r)
}

func wrapAPI(cfgStore clustering.ConfigurationStore, w http.ResponseWriter, req *http.Request, f apiFunc) {
	obj, err := f(cfgStore, w, req)
	if err != nil {
		writeError(w, err)
		return
	}

	if obj == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	buf, json_err := json.Marshal(obj)
	if json_err != nil {
		writeError(w, errors.NewError(json_err, ""))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func writeError(w http.ResponseWriter, err errors.Error) {
	w.Header().Set("Content-Type", "application/json")
	buf, er := json.Marshal(err)
	if er != nil {
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
	status := mapErrorToHttpStatus(err)
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func doPing(cfgStore clustering.ConfigurationStore, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	return &struct {
		status string `json:"status"`
	}{
		"ok",
	}, nil
}

var localConfig struct {
	sync.Mutex
	myConfig clustering.QueryNode
}

func doConfig(cfgStore clustering.ConfigurationStore, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	if localConfig.myConfig != nil {
		return localConfig.myConfig, nil
	}
	var self clustering.QueryNode
	name, err := util.ExternalIP()
	if err != nil {
		return nil, err
	}

	cm := cfgStore.ConfigurationManager()
	clusters, err := cm.GetClusters()
	if err != nil {
		return nil, err
	}

	for _, c := range clusters {
		clm := c.ClusterManager()
		queryNodes, err := clm.GetQueryNodes()
		if err != nil {
			return nil, err
		}

		for _, qryNode := range queryNodes {
			if qryNode.Name() == name {
				self = qryNode
				break
			}
		}
	}
	localConfig.Lock()
	defer localConfig.Unlock()
	localConfig.myConfig = self
	return localConfig.myConfig, nil
}

func doClusters(cfgStore clustering.ConfigurationStore, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	cm := cfgStore.ConfigurationManager()
	switch req.Method {
	case "GET":
		return cm.GetClusters()
	case "POST":
		cluster, err := getClusterFromRequest(req)
		if err != nil {
			return nil, errors.NewError(err, "")
		}
		return cfgStore.ConfigurationManager().AddCluster(cluster)
	default:
		return nil, nil
	}
}

func doCluster(cfgStore clustering.ConfigurationStore, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	name := vars["cluster"]
	cluster, err := cfgStore.ClusterByName(name)
	if err != nil {
		return nil, err
	}

	switch req.Method {
	case "GET":
		return cluster, nil
	case "DELETE":
		return cfgStore.ConfigurationManager().RemoveCluster(cluster)
	default:
		return nil, nil
	}
}

func doNodes(cfgStore clustering.ConfigurationStore, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	name := vars["cluster"]
	cluster, err := cfgStore.ClusterByName(name)
	if err != nil || cluster == nil {
		return cluster, err
	}
	switch req.Method {
	case "GET":
		return cluster.ClusterManager().GetQueryNodes()
	case "POST":
		node, err := getNodeFromRequest(req)
		if err != nil {
			return nil, errors.NewError(err, "")
		}
		return cluster.ClusterManager().AddQueryNode(node)
	default:
		return nil, nil
	}
}

func doNode(cfgStore clustering.ConfigurationStore, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	node := vars["node"]
	name := vars["cluster"]
	cluster, err := cfgStore.ClusterByName(name)
	if err != nil || cluster == nil {
		return cluster, err
	}

	switch req.Method {
	case "GET":
		return cluster.QueryNodeByName(node)
	case "DELETE":
		return cluster.ClusterManager().RemoveQueryNodeByName(node)
	default:
		return nil, nil
	}
}

func getClusterFromRequest(req *http.Request) (clustering.Cluster, error) {
	var cluster clustering.Cluster
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&cluster)
	return cluster, err
}

func getNodeFromRequest(req *http.Request) (clustering.QueryNode, error) {
	var node clustering.QueryNode
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&node)
	return node, err
}

func mapErrorToHttpStatus(err errors.Error) int {
	return http.StatusInternalServerError
}
