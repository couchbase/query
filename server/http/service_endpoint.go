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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/audit"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/util"
	"github.com/gorilla/mux"
)

type HttpEndpoint struct {
	server        *server.Server
	metrics       bool
	httpAddr      string
	httpsAddr     string
	certFile      string
	keyFile       string
	bufpool       BufferPool
	listener      net.Listener
	listenerTLS   net.Listener
	mux           *mux.Router
	actives       server.ActiveRequests
	options       server.ServerOptions
	connSecConfig datastore.ConnectionSecurityConfig
}

const (
	servicePrefix = "/query/service"
)

var _ENDPOINT *HttpEndpoint

func NewServiceEndpoint(srv *server.Server, staticPath string, metrics bool,
	httpAddr, httpsAddr, certFile, keyFile string) *HttpEndpoint {
	rv := &HttpEndpoint{
		server:    srv,
		metrics:   metrics,
		httpAddr:  httpAddr,
		httpsAddr: httpsAddr,
		certFile:  certFile,
		keyFile:   keyFile,
		bufpool:   NewSyncPool(srv.KeepAlive()),
		actives:   NewActiveRequests(),
		options:   NewHttpOptions(srv),
	}

	rv.connSecConfig.CertFile = certFile
	rv.connSecConfig.KeyFile = keyFile

	server.SetActives(rv.actives)
	server.SetOptions(rv.options)

	rv.setupSSL()
	rv.registerHandlers(staticPath)
	_ENDPOINT = rv
	return rv
}

func (this *HttpEndpoint) Mux() *mux.Router {
	return this.mux
}

func (this *HttpEndpoint) Listen() error {
	ln, err := net.Listen("tcp", this.httpAddr)
	if err == nil {
		this.listener = ln
		srv := &http.Server{
			Handler:           this.mux,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       30 * time.Second,
		}
		go srv.Serve(ln)
		logging.Infop("HttpEndpoint: Listen", logging.Pair{"Address", ln.Addr()})
	}
	return err
}

func (this *HttpEndpoint) ListenTLS() error {
	// create tls configuration
	if this.certFile == "" {
		logging.Errorf("No certificate passed. Secure listener not brought up.")
		return nil
	}
	tlsCert, err := tls.LoadX509KeyPair(this.certFile, this.keyFile)
	if err != nil {
		return err
	}

	cbauthTLSsettings, err1 := cbauth.GetTLSConfig()
	if err1 != nil {
		return fmt.Errorf("Failed to get cbauth tls config: %v", err.Error())
	}
	this.connSecConfig.TLSConfig = cbauthTLSsettings

	ln, err := net.Listen("tcp", this.httpsAddr)

	if err == nil {
		cfg := &tls.Config{
			Certificates:             []tls.Certificate{tlsCert},
			ClientAuth:               cbauthTLSsettings.ClientAuthType,
			MinVersion:               cbauthTLSsettings.MinVersion,
			CipherSuites:             cbauthTLSsettings.CipherSuites,
			PreferServerCipherSuites: cbauthTLSsettings.PreferServerCipherSuites,
			NextProtos:               []string{"h2", "http/1.1"},
		}

		if cbauthTLSsettings.ClientAuthType != tls.NoClientCert {
			caCert, err := ioutil.ReadFile(this.certFile)
			if err != nil {
				return fmt.Errorf(" Error in reading cacert file, err: %v", err)
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			cfg.ClientCAs = caCertPool
		}

		tls_ln := tls.NewListener(ln, cfg)
		this.listenerTLS = tls_ln
		srv := &http.Server{
			Handler:           this.mux,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       30 * time.Second,
		}
		go srv.Serve(tls_ln)
		logging.Infop("HttpEndpoint: ListenTLS", logging.Pair{"Address", ln.Addr()})
	}

	return err
}

// If the server channel is full and we are unable to queue a request,
// we respond with a timeout status.
func (this *HttpEndpoint) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	// avoid GC load. this works because ServeHTTP is alive for the whole
	// duration of the request and it's safe to reference its frame
	var httpRequest httpRequest

	request := newHttpRequest(&httpRequest, resp, req, this.bufpool, this.server.RequestSizeCap())

	this.actives.Put(request)
	defer this.actives.Delete(request.Id().String(), false)

	defer this.doStats(request, this.server)

	if request.State() == server.FATAL {
		// There was problems creating the request: Fail it and return
		request.Failed(this.server)
		return
	}

	var res bool
	if request.ScanConsistency() == datastore.UNBOUNDED {
		res = this.server.ServiceRequest(request)

	} else {
		res = this.server.PlusServiceRequest(request)
	}

	if !res {
		resp.WriteHeader(http.StatusServiceUnavailable)
	}
}

func (this *HttpEndpoint) Close() error {
	return this.closeListener(this.listener)
}

func (this *HttpEndpoint) CloseTLS() error {
	return this.closeListener(this.listenerTLS)
}

func (this *HttpEndpoint) closeListener(l net.Listener) error {
	var err error
	if l != nil {
		err = l.Close()
		logging.Infop("HttpEndpoint: close listener ", logging.Pair{"Address", l.Addr()}, logging.Pair{"err", err})
	}
	return err
}

func (this *HttpEndpoint) registerHandlers(staticPath string) {
	this.mux = mux.NewRouter()

	this.mux.Handle(servicePrefix, this).
		Methods("GET", "POST")

	// TODO: Deprecate (remove) this binding
	this.mux.Handle("/query", this).
		Methods("GET", "POST")

	this.registerClusterHandlers()
	this.registerAccountingHandlers()
	this.registerStaticHandlers(staticPath)
}

func (this *HttpEndpoint) registerStaticHandlers(staticPath string) {
	this.mux.Handle("/", http.FileServer(http.Dir(staticPath)))
	pathPrefix := "/tutorial/"
	pathValue := staticPath + pathPrefix
	this.mux.PathPrefix(pathPrefix).Handler(http.StripPrefix(pathPrefix,
		http.FileServer(http.Dir(pathValue))))
}

// Reconfigure the node-to-node encryption.
func (this *HttpEndpoint) UpdateNodeToNodeEncryptionLevel() {
}

func (this *HttpEndpoint) setupSSL() {

	err := cbauth.RegisterConfigRefreshCallback(func(configChange uint64) error {
		// Both flags could be set here.
		settingsUpdated := false
		if (configChange & cbauth.CFG_CHANGE_CERTS_TLSCONFIG) != 0 {
			logging.Infof(" Certificates have been refreshed by ns server ")
			closeErr := this.CloseTLS()
			if closeErr != nil && !strings.ContainsAny(strings.ToLower(closeErr.Error()), "closed network connection & use") {
				logging.Infof("ERROR: Closing TLS listener - %s", closeErr.Error())
				return errors.NewAdminEndpointError(closeErr, "error closing tls listenener")
			}

			tlsErr := this.ListenTLS()
			if tlsErr != nil {
				if strings.ContainsAny(strings.ToLower(tlsErr.Error()), "bind address & already in use") {
					time.Sleep(100 * time.Millisecond)
				}
				logging.Infof("ERROR: Starting TLS listener - %s", tlsErr.Error())
				return errors.NewAdminEndpointError(tlsErr, "error starting tls listenener")
			}
			settingsUpdated = true
		}

		if (configChange & cbauth.CFG_CHANGE_CLUSTER_ENCRYPTION) != 0 {
			cryptoConfig, err := cbauth.GetClusterEncryptionConfig()
			if err != nil {
				logging.Errorf("unable to retrieve node-to-node encryption settings: %v", err)
				return errors.NewAdminEndpointError(err, "unable to retrieve node-to-node encryption settings")
			}
			this.connSecConfig.ClusterEncryptionConfig = cryptoConfig

			// Temporary log message.
			logging.Errorf("Updating node-to-node encryption level: %+v", cryptoConfig)
			settingsUpdated = true
		}

		if settingsUpdated {
			ds := datastore.GetDatastore()
			if ds == nil {
				logging.Warnf("No datastore configured. Unable to update connection security settings.")
			} else {
				ds.SetConnectionSecurityConfig(&(this.connSecConfig))
			}
			sds := datastore.GetSystemstore()
			if sds == nil {
				logging.Warnf("No system datastore configured. Unable to update connection security settings.")
			} else {
				sds.SetConnectionSecurityConfig(&(this.connSecConfig))
			}
			distributed.RemoteAccess().SetConnectionSecurityConfig(this.connSecConfig.CertFile,
				this.connSecConfig.ClusterEncryptionConfig.EncryptData)
		}
		return nil
	})
	if err != nil {
		logging.Infof("Error with refreshing client certificate : %v", err.Error())
	}
}

func (this *HttpEndpoint) doStats(request *httpRequest, srvr *server.Server) {

	// Update metrics:
	service_time := request.executionTime
	request_time := request.elapsedTime
	prepared := request.Prepared() != nil

	prepareds.RecordPreparedMetrics(request.Prepared(), request_time, service_time)
	accounting.RecordMetrics(request_time, service_time, request.resultCount,
		request.resultSize, request.errorCount, request.warningCount, request.Type(),
		prepared, (request.State() != server.COMPLETED),
		int(request.PhaseOperator(execution.INDEX_SCAN)),
		int(request.PhaseOperator(execution.PRIMARY_SCAN)),
		string(request.ScanConsistency()))

	request.CompleteRequest(request_time, service_time, request.resultCount,
		request.resultSize, request.errorCount, request.req, srvr)

	audit.Submit(request)
}

func ServicePrefix() string {
	return servicePrefix
}

// activeHttpRequests implements server.ActiveRequests for http requests
type activeHttpRequests struct {
	cache *util.GenCache
}

func NewActiveRequests() server.ActiveRequests {
	return &activeHttpRequests{
		cache: util.NewGenCache(-1),
	}
}

func (this *activeHttpRequests) Put(req server.Request) errors.Error {
	http_req, is_http := req.(*httpRequest)
	if !is_http {
		return errors.NewServiceErrorHttpReq(req.Id().String())
	}
	this.cache.FastAdd(http_req, http_req.Id().String())
	return nil
}

func (this *activeHttpRequests) Get(id string, f func(server.Request)) errors.Error {
	var dummyF func(interface{}) = nil

	if f != nil {
		dummyF = func(e interface{}) {
			r := e.(*httpRequest)
			f(r)
		}
	}
	_ = this.cache.Get(id, dummyF)
	return nil
}

func (this *activeHttpRequests) Delete(id string, stop bool) bool {
	this.cache.Delete(id, func(e interface{}) {
		if stop {
			req := e.(*httpRequest)
			req.Stop(server.STOPPED)
		}
	})

	return true
}

func (this *activeHttpRequests) Count() (int, errors.Error) {
	return this.cache.Size(), nil
}

func (this *activeHttpRequests) ForEach(nonBlocking func(string, server.Request) bool, blocking func() bool) {
	dummyF := func(id string, r interface{}) bool {
		return nonBlocking(id, r.(*httpRequest))
	}
	this.cache.ForEach(dummyF, blocking)
}

// httpOptions implements server.ServerOptions for http servers
type httpOptions struct {
	server *server.Server
}

func NewHttpOptions(server *server.Server) server.ServerOptions {
	return &httpOptions{
		server: server,
	}
}

func (this *httpOptions) Controls() bool {
	return this.server.Controls()
}

func (this *httpOptions) Profile() server.Profile {
	return this.server.Profile()
}
