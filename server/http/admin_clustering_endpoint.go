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
	"bytes"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/couchbase/query/clustering"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/server"
	"github.com/gorilla/mux"
)

const (
	clustersPrefix = adminPrefix + "/clusters"
)

func (this *HttpEndpoint) registerClusterHandlers() {
	pingHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doPing)
	}
	configHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doConfig)
	}
	sslCertHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doSslCert)
	}
	clustersHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doClusters)
	}
	clusterHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doCluster)
	}
	nodesHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doNodes)
	}
	nodeHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doNode)
	}
	settingsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doSettings)
	}
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
	}

	for route, h := range routeMap {
		this.mux.HandleFunc(route, h.handler).Methods(h.methods...)
	}

}

func (this *HttpEndpoint) doConfigStore() (clustering.ConfigurationStore, errors.Error) {
	configStore := this.server.ConfigurationStore()
	if configStore == nil {
		return nil, errors.NewAdminAuthError(nil, "Failed to connect to Configuration Store")
	}
	return configStore, nil
}

func (this *HttpEndpoint) hasAdminAuth(req *http.Request) errors.Error {
	// retrieve the credentials from the request; the credentials must be specified
	// using basic authorization format. An error is returned if there is a step that
	// prevents retrieval of the credentials.
	authHdr := req.Header["Authorization"]
	if len(authHdr) == 0 {
		return errors.NewAdminAuthError(nil, "basic authorization required")
	}

	auth := authHdr[0]
	basicPrefix := "Basic "
	if !strings.HasPrefix(auth, basicPrefix) {
		return errors.NewAdminAuthError(nil, "basic authorization required")
	}

	decoded, err := base64.StdEncoding.DecodeString(auth[len(basicPrefix):])
	if err != nil {
		return errors.NewAdminDecodingError(err)
	}

	colonIndex := bytes.IndexByte(decoded, ':')
	if colonIndex == -1 {
		return errors.NewAdminAuthError(nil, "incorrect authorization header")
	}

	user := string(decoded[:colonIndex])
	password := string(decoded[colonIndex+1:])
	creds := map[string]string{user: password}

	// Attempt authorization with the cluster
	configstore, configErr := this.doConfigStore()
	if configErr != nil {
		return configErr
	}
	sslPrivs := []clustering.Privilege{clustering.PRIV_SYS_ADMIN}
	authErr := configstore.Authorize(creds, sslPrivs)
	if authErr != nil {
		return authErr
	}

	return nil
}

var pingStatus = struct {
	status string `json:"status"`
}{
	"ok",
}

func doPing(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	return &pingStatus, nil
}

var localConfig struct {
	sync.Mutex
	name     string
	myConfig clustering.QueryNode
}

func doConfig(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
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

func doClusters(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
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
		return cfgStore.ConfigurationManager().AddCluster(cluster)
	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doCluster(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	name := vars["cluster"]
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

func doNodes(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	name := vars["cluster"]
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
		return cluster.ClusterManager().AddQueryNode(node)
	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doNode(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	node := vars["node"]
	name := vars["cluster"]
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
func doSslCert(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	if endpoint.httpsAddr == "" {
		return nil, errors.NewAdminNotSSLEnabledError()
	}

	err := endpoint.hasAdminAuth(req)
	if err != nil {
		return nil, err
	}

	// Auth clear: restart TLS listener to reload the SSL cert.
	closeErr := endpoint.CloseTLS()
	if closeErr != nil {
		return nil, errors.NewAdminEndpointError(closeErr, "error closing tls listenener")
	}

	tlsErr := endpoint.ListenTLS()
	if tlsErr != nil {
		return nil, errors.NewAdminEndpointError(tlsErr, "error starting tls listenener")
	}

	// response payload
	sslStatus := map[string]string{}
	sslStatus["status"] = "ok"
	sslStatus["keyfile"] = endpoint.keyFile
	sslStatus["certfile"] = endpoint.certFile

	return sslStatus, nil
}

const (
	_CPUPROFILE      = "cpuprofile"
	_DEBUG           = "debug"
	_KEEPALIVELENGTH = "keep-alive-length"
	_LOGLEVEL        = "loglevel"
	_MAXPARALLELISM  = "max-parallelism"
	_MEMPROFILE      = "memprofile"
	_REQUESTSIZECAP  = "request-size-cap"
	_PIPELINEBATCH   = "pipeline-batch"
	_PIPELINECAP     = "pipeline-cap"
	_SCANCAP         = "scan-cap"
	_SERVICERS       = "servicers"
	_TIMEOUT         = "timeout"
	_CMPTHRESHOLD    = "completed-threshold"
	_CMPLIMIT        = "completed-limit"
	_PRPLIMIT        = "prepared-limit"
	_PRETTY          = "pretty"
	_PROFILE         = "profile"
	_CONTROLS        = "controls"
	_MAXINDEXAPI     = "max-index-api"
)

type checker func(interface{}) (bool, errors.Error)

func checkBool(val interface{}) (bool, errors.Error) {
	_, ok := val.(bool)
	return ok, nil
}

func checkNumber(val interface{}) (bool, errors.Error) {
	_, ok := val.(float64)
	return ok, nil
}

func checkPositiveInteger(val interface{}) (bool, errors.Error) {
	v, ok := val.(float64)

	// we are getting floats here - val doesn't cast to ints
	// and we want a cache, however small
	return ok && (v > 1), nil
}

func checkString(val interface{}) (bool, errors.Error) {
	_, ok := val.(string)
	return ok, nil
}

func checkLogLevel(val interface{}) (bool, errors.Error) {
	level, is_string := val.(string)
	if !is_string {
		return false, nil
	}
	_, ok := logging.ParseLevel(level)
	return ok, nil
}

var _CHECKERS = map[string]checker{
	_CPUPROFILE:      checkString,
	_DEBUG:           checkBool,
	_KEEPALIVELENGTH: checkNumber,
	_LOGLEVEL:        checkLogLevel,
	_MAXPARALLELISM:  checkNumber,
	_MEMPROFILE:      checkString,
	_REQUESTSIZECAP:  checkNumber,
	_PIPELINEBATCH:   checkNumber,
	_PIPELINECAP:     checkNumber,
	_SCANCAP:         checkNumber,
	_SERVICERS:       checkNumber,
	_TIMEOUT:         checkNumber,
	_CMPTHRESHOLD:    checkNumber,
	_CMPLIMIT:        checkNumber,
	_PRPLIMIT:        checkPositiveInteger,
	_PRETTY:          checkBool,
	_PROFILE:         checkProfileAdmin,
	_CONTROLS:        checkControlsAdmin,
	_MAXINDEXAPI:     checkNumber,
}

type setter func(*server.Server, interface{})

var _SETTERS = map[string]setter{
	_CPUPROFILE: func(s *server.Server, o interface{}) {
		value, _ := o.(string)
		s.SetCpuProfile(value)
	},
	_DEBUG: func(s *server.Server, o interface{}) {
		value, _ := o.(bool)
		s.SetDebug(value)
	},
	_KEEPALIVELENGTH: func(s *server.Server, o interface{}) {
		value, _ := o.(float64)
		s.SetKeepAlive(int(value))
	},
	_LOGLEVEL: func(s *server.Server, o interface{}) {
		value, _ := o.(string)
		s.SetLogLevel(value)
	},
	_MAXPARALLELISM: func(s *server.Server, o interface{}) {
		value, _ := o.(float64)
		s.SetMaxParallelism(int(value))
	},
	_MEMPROFILE: func(s *server.Server, o interface{}) {
		value, _ := o.(string)
		s.SetMemProfile(value)
	},
	_PIPELINECAP: func(s *server.Server, o interface{}) {
		value, _ := o.(float64)
		s.SetPipelineCap(int64(value))
	},
	_PIPELINEBATCH: func(s *server.Server, o interface{}) {
		value, _ := o.(float64)
		s.SetPipelineBatch(int(value))
	},
	_REQUESTSIZECAP: func(s *server.Server, o interface{}) {
		value, _ := o.(float64)
		s.SetRequestSizeCap(int(value))
	},
	_SCANCAP: func(s *server.Server, o interface{}) {
		value, _ := o.(float64)
		s.SetScanCap(int64(value))
	},
	_SERVICERS: func(s *server.Server, o interface{}) {
		value, _ := o.(float64)
		s.SetServicers(int(value))
	},
	_TIMEOUT: func(s *server.Server, o interface{}) {
		value, _ := o.(float64)
		s.SetTimeout(time.Duration(value))
	},
	_CMPTHRESHOLD: func(s *server.Server, o interface{}) {
		value, _ := o.(float64)
		_ = server.RequestsUpdateQualifier("threshold", int(value))
	},
	_CMPLIMIT: func(s *server.Server, o interface{}) {
		value, _ := o.(float64)
		server.RequestsSetLimit(int(value))
	},
	_PRPLIMIT: func(s *server.Server, o interface{}) {
		value, _ := o.(float64)
		plan.PreparedsSetLimit(int(value))
	},
	_PRETTY: func(s *server.Server, o interface{}) {
		value, _ := o.(bool)
		s.SetPretty(value)
	},
	_PROFILE:  setProfileAdmin,
	_CONTROLS: setControlsAdmin,
	_MAXINDEXAPI: func(s *server.Server, o interface{}) {
		value, _ := o.(float64)
		s.SetMaxIndexAPI(int(value))
	},
}

func doSettings(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	// Admin auth required
	err := endpoint.hasAdminAuth(req)
	if err != nil {
		return nil, err
	}

	settings := map[string]interface{}{}
	srvr := endpoint.server
	switch req.Method {
	case "GET":
		return fillSettings(settings, srvr), nil
	case "POST":
		decoder, e := getJsonDecoder(req.Body)
		if e != nil {
			return nil, e
		}
		err := decoder.Decode(&settings)
		if err != nil {
			return nil, errors.NewAdminDecodingError(err)
		}
		for setting, value := range settings {
			if check_it, ok := _CHECKERS[setting]; !ok {
				return nil, errors.NewAdminUnknownSettingError(setting)
			} else {
				ok, err := check_it(value)
				if !ok {
					if err == nil {
						return nil, errors.NewAdminSettingTypeError(setting, value)
					} else {
						return nil, err
					}
				}
			}
		}
		for setting, value := range settings {
			set_it := _SETTERS[setting]
			set_it(srvr, value)
		}
		return fillSettings(settings, srvr), nil
	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func fillSettings(settings map[string]interface{}, srvr *server.Server) map[string]interface{} {
	settings[_CPUPROFILE] = srvr.CpuProfile()
	settings[_MEMPROFILE] = srvr.MemProfile()
	settings[_SERVICERS] = srvr.Servicers()
	settings[_SCANCAP] = srvr.ScanCap()
	settings[_REQUESTSIZECAP] = srvr.RequestSizeCap()
	settings[_DEBUG] = srvr.Debug()
	settings[_PIPELINEBATCH] = srvr.PipelineBatch()
	settings[_PIPELINECAP] = srvr.PipelineCap()
	settings[_MAXPARALLELISM] = srvr.MaxParallelism()
	settings[_TIMEOUT] = srvr.Timeout()
	settings[_KEEPALIVELENGTH] = srvr.KeepAlive()
	settings[_LOGLEVEL] = srvr.LogLevel()
	threshold, _ := server.RequestsGetQualifier("threshold")
	settings[_CMPTHRESHOLD] = threshold
	settings[_CMPLIMIT] = server.RequestsLimit()
	settings[_PRPLIMIT] = plan.PreparedsLimit()
	settings[_PRETTY] = srvr.Pretty()
	settings[_MAXINDEXAPI] = srvr.MaxIndexAPI()
	settings = getProfileAdmin(settings, srvr)
	settings = getControlsAdmin(settings, srvr)
	return settings
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
