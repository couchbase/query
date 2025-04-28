//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package http

import (
	"net/http"
	"net/http/pprof"
	"strconv"
	"sync"
	"time"

	json "github.com/couchbase/go_json"
	"github.com/couchbase/query/audit"
	"github.com/couchbase/query/clustering"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/server/http/router"
)

const (
	clustersPrefix   = adminPrefix + "/clusters"
	profilePrivilege = clustering.PRIV_READ
)

func (this *HttpEndpoint) registerClusterHandlers() {
	pingHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doPing, false)
	}
	configHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doConfig, false)
	}
	sslCertHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doSslCert, false)
	}
	clustersHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doClusters, false)
	}
	clusterHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doCluster, false)
	}
	nodesHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doNodes, false)
	}
	nodeHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doNode, false)
	}
	settingsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doSettings, false)
	}
	shutdownHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doShutdown, false)
	}
	indexHandler := this.wrapHandlerFuncWithAdminAuth(pprof.Index)
	profileHandler := this.wrapHandlerFuncWithAdminAuth(pprof.Profile)
	routeMap := map[string]struct {
		handler handlerFunc
		methods []string
	}{
		adminPrefix + "/ping":                      {handler: pingHandler, methods: []string{"GET"}},
		adminPrefix + "/config":                    {handler: configHandler, methods: []string{"GET"}},
		adminPrefix + "/ssl_cert":                  {handler: sslCertHandler, methods: []string{"POST"}},
		adminPrefix + "/settings":                  {handler: settingsHandler, methods: []string{"GET", "POST"}},
		clustersPrefix:                             {handler: clustersHandler, methods: []string{"GET", "POST"}},
		clustersPrefix + "/{cluster}":              {handler: clusterHandler, methods: []string{"GET", "PUT", "DELETE"}},
		clustersPrefix + "/{cluster}/nodes":        {handler: nodesHandler, methods: []string{"GET", "POST"}},
		clustersPrefix + "/{cluster}/nodes/{node}": {handler: nodeHandler, methods: []string{"GET", "PUT", "DELETE"}},
		"/debug/pprof/":                            {handler: indexHandler, methods: []string{"GET"}},
		"/debug/pprof/profile":                     {handler: profileHandler, methods: []string{"GET"}},
		adminPrefix + "/shutdown":                  {handler: shutdownHandler, methods: []string{"GET", "POST"}},
	}

	for route, h := range routeMap {
		this.router.Map(route, h.handler, h.methods...)
	}
	this.router.Map("/debug/pprof/block", newAdminAuthHandlerWrapper(this, pprof.Handler("block")).ServeHTTP)
	this.router.Map("/debug/pprof/goroutine", newAdminAuthHandlerWrapper(this, pprof.Handler("goroutine")).ServeHTTP)
	this.router.Map("/debug/pprof/threadcreate", newAdminAuthHandlerWrapper(this, pprof.Handler("threadcreate")).ServeHTTP)
	this.router.Map("/debug/pprof/heap", newAdminAuthHandlerWrapper(this, pprof.Handler("heap")).ServeHTTP)
	this.router.Map("/debug/pprof/mutex", newAdminAuthHandlerWrapper(this, pprof.Handler("mutex")).ServeHTTP)

}

func (this *HttpEndpoint) wrapHandlerFuncWithAdminAuth(f func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter,
	*http.Request) {

	return func(writer http.ResponseWriter, request *http.Request) {
		authErr := this.hasAdminAuth(request, profilePrivilege)
		if authErr != nil {
			writeError(writer, authErr)
			return
		}
		f(writer, request)
	}
}

func newAdminAuthHandlerWrapper(endpoint *HttpEndpoint, baseHandler http.Handler) http.Handler {
	return &adminAuthHandlerWrapper{baseHandler: baseHandler, endpoint: endpoint}
}

type adminAuthHandlerWrapper struct {
	baseHandler http.Handler
	endpoint    *HttpEndpoint
}

func (wrapper *adminAuthHandlerWrapper) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	authErr := wrapper.endpoint.hasAdminAuth(r, profilePrivilege)
	if authErr != nil {
		writeError(rw, authErr)
		return
	}
	wrapper.baseHandler.ServeHTTP(rw, r)
}

func (this *HttpEndpoint) doConfigStore() (clustering.ConfigurationStore, errors.Error) {
	configStore := this.server.ConfigurationStore()
	if configStore == nil {
		return nil, errors.NewAdminAuthError(nil, "Failed to connect to Configuration Store")
	}
	return configStore, nil
}

func (this *HttpEndpoint) hasAdminAuth(req *http.Request, privilege clustering.Privilege) errors.Error {
	// Attempt authorization with the cluster
	configstore, configErr := this.doConfigStore()
	if configErr != nil {
		return configErr
	}
	sslPrivs := []clustering.Privilege{privilege}
	authErr := configstore.Authorize(req, sslPrivs)
	if authErr != nil {
		return authErr
	}

	return nil
}

var pingStatus = struct {
	Status string `json:"status"`
}{
	"OK",
}

func doPing(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_ADMIN_PING
	if endpoint.server.IsHealthy() {
		return &pingStatus, nil
	} else {
		return nil, errors.NewServiceUnavailableError()
	}
}

var localConfig struct {
	sync.Mutex
	name     string
	myConfig clustering.QueryNode
}

func doConfig(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_ADMIN_CONFIG
	var self clustering.QueryNode

	cfgStore, cfgErr := endpoint.doConfigStore()
	if cfgErr != nil {
		return nil, cfgErr
	}
	name, er := cfgStore.WhoAmI()
	if er != nil {
		return nil, errors.NewAdminGetNodeError(er, server.GetIP(false))
	}
	if localConfig.myConfig != nil && name == localConfig.name {
		return localConfig.myConfig, nil
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
	localConfig.name = name
	return localConfig.myConfig, nil
}

func doClusters(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_ADMIN_CLUSTERS
	cfgStore, cfgErr := endpoint.doConfigStore()
	if cfgErr != nil {
		return nil, cfgErr
	}
	cm := cfgStore.ConfigurationManager()
	switch req.Method {
	case "GET":
		return cm.GetClusters()
	case "POST":
		cluster, err := getClusterFromRequest(req)
		if err != nil {
			return nil, err
		}
		af.Body = cluster
		return cfgStore.ConfigurationManager().AddCluster(cluster)
	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doCluster(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_ADMIN_CLUSTERS
	_, name := router.RequestValue(req, "cluster")
	af.Cluster = name
	cfgStore, cfgErr := endpoint.doConfigStore()
	if cfgErr != nil {
		return nil, cfgErr
	}
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
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doNodes(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_ADMIN_CLUSTERS
	_, name := router.RequestValue(req, "cluster")
	af.Cluster = name
	cfgStore, cfgErr := endpoint.doConfigStore()
	if cfgErr != nil {
		return nil, cfgErr
	}
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
			return nil, err
		}
		af.Body = node
		return cluster.ClusterManager().AddQueryNode(node)
	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doNode(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	_, name := router.RequestValue(req, "cluster")
	_, node := router.RequestValue(req, "node")

	af.EventTypeId = audit.API_ADMIN_CLUSTERS
	af.Node = node
	af.Cluster = name

	cfgStore, cfgErr := endpoint.doConfigStore()
	if cfgErr != nil {
		return nil, cfgErr
	}
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
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

// reload the ssl certificate. Only performed if the server is running https and
// the request contains basic authorization credentials that can be successfully
// authorized against the configuration store.
func doSslCert(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_ADMIN_SSL_CERT
	if endpoint.httpsAddr == "" {
		return nil, errors.NewAdminNotSSLEnabledError()
	}

	err := endpoint.hasAdminAuth(req, clustering.PRIV_SYS_ADMIN)
	if err != nil {
		return nil, err
	}

	// Reload the SSL certificate
	tlsErr := endpoint.ListenTLS()

	if tlsErr != nil {
		logging.Errorf("Error reloading the certificate: %s", tlsErr.Error())
		return nil, errors.NewAdminEndpointError(tlsErr, "Error reloading the certificate.")
	}

	// response payload
	sslStatus := map[string]string{}
	sslStatus["status"] = "ok"
	sslStatus["keyfile"] = endpoint.keyFile
	sslStatus["certfile"] = endpoint.certFile
	sslStatus["cafile"] = endpoint.cafile

	return sslStatus, nil
}

func doSettings(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_ADMIN_SETTINGS

	// Admin auth required
	privilege := clustering.PRIV_SYS_ADMIN
	if req.Method == "GET" {
		privilege = clustering.PRIV_READ
	}
	err := endpoint.hasAdminAuth(req, privilege)
	if err != nil {
		return nil, err
	}

	settings := map[string]interface{}{}
	srvr := endpoint.server
	switch req.Method {
	case "GET":
		return server.FillSettings(settings, srvr), nil
	case "POST":
		decoder, e := getJsonDecoder(req.Body)
		if e != nil {
			return nil, e
		}
		err := decoder.Decode(&settings)
		if err != nil {
			return nil, errors.NewAdminDecodingError(err)
		}

		errP := settingsWorkHorse(settings, srvr)
		af.Values = settings
		if errP != nil {
			return nil, errP
		}
		settings = map[string]interface{}{}
		return server.FillSettings(settings, srvr), nil
	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func settingsWorkHorse(settings map[string]interface{}, srvr *server.Server) errors.Error {
	distribute := settings["distribute"]
	if distribute != nil {
		delete(settings, "distribute")
	}

	if errP := server.ProcessSettings(settings, srvr); errP != nil {
		return errP
	}

	if distribute != nil {
		body, _ := json.Marshal(settings)
		go distributed.RemoteAccess().DoRemoteOps([]string{}, "settings", "POST", "", string(body),
			func(warn errors.Error) {
				if warn != nil {
					logging.Infof("failed to distribute settings <ud>%v</ud>", settings)
				}
			}, distributed.NO_CREDS, "")
	}
	return nil
}

func getClusterFromRequest(req *http.Request) (clustering.Cluster, errors.Error) {
	var cluster clustering.Cluster
	decoder, e := getJsonDecoder(req.Body)
	if e != nil {
		return nil, e
	}
	err := decoder.Decode(&cluster)
	if err != nil {
		return nil, errors.NewAdminDecodingError(err)
	}
	return cluster, nil
}

func getNodeFromRequest(req *http.Request) (clustering.QueryNode, errors.Error) {
	var node clustering.QueryNode
	decoder, e := getJsonDecoder(req.Body)
	if e != nil {
		return nil, e
	}
	err := decoder.Decode(&node)
	if err != nil {
		return nil, errors.NewAdminDecodingError(err)
	}
	return node, nil
}

func doShutdown(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_ADMIN_SHUTDOWN

	// only permit internal callers
	err, isInternal := endpoint.verifyCredentialsFromRequest(nil, req, af)

	if err != nil {
		return nil, errors.NewAdminAuthError(err, "")
	}

	if !isInternal {
		return nil, errors.NewAdminAuthError(nil, "")
	}

	switch req.Method {
	case "POST":
		timeout := time.Duration(0)
		deadline, e := strconv.ParseInt(req.FormValue("deadline"), 10, 64)
		if e != nil {
			timeout, e = time.ParseDuration(req.FormValue("timeout"))
		} else {
			timeout = time.Unix(deadline, 0).Sub(time.Now())
		}
		cancel := req.FormValue("cancel")
		if !endpoint.server.ShutDown() {
			if cancel == "" {
				endpoint.server.InitiateShutdown(timeout, "shutdown instruction")
				return textPlain("shutdown requested\n"), nil
			} else if endpoint.server.ShuttingDown() {
				endpoint.server.CancelShutdown()
				return textPlain("shutdown cancelled\n"), nil
			}
		} else {
			return errors.NewServiceShuttingDownError(), nil
		}
	case "GET":
		if endpoint.server.ShutDown() {
			return errors.NewServiceShutDownError(), nil
		} else if endpoint.server.ShuttingDown() {
			return errors.NewServiceShuttingDownError(), nil
		}
	}
	return nil, errors.NewAdminEndpointError(nil, "Invalid method:"+req.Method)
}
