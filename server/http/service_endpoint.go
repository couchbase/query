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
	"net"
	"net/http"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/audit"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/util"
	"github.com/gorilla/mux"
)

type HttpEndpoint struct {
	server      *server.Server
	metrics     bool
	httpAddr    string
	httpsAddr   string
	certFile    string
	keyFile     string
	bufpool     BufferPool
	listener    net.Listener
	listenerTLS net.Listener
	mux         *mux.Router
	actives     server.ActiveRequests
	options     server.ServerOptions
}

const (
	servicePrefix = "/query/service"
)

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

	server.SetActives(rv.actives)
	server.SetOptions(rv.options)

	rv.registerHandlers(staticPath)
	return rv
}

func (this *HttpEndpoint) Listen() error {
	ln, err := net.Listen("tcp", this.httpAddr)
	if err == nil {
		this.listener = ln
		go http.Serve(ln, this.mux)
		logging.Infop("HttpEndpoint: Listen", logging.Pair{"Address", ln.Addr()})
	}
	return err
}

func (this *HttpEndpoint) ListenTLS() error {
	// create tls configuration
	tlsCert, err := tls.LoadX509KeyPair(this.certFile, this.keyFile)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", this.httpsAddr)
	if err == nil {
		cfg := &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
			ClientAuth:   tls.NoClientCert,
			MinVersion:   cbauth.MinTLSVersion(),
			CipherSuites: cbauth.CipherSuites(),
			NextProtos:   []string{"h2", "http/1.1"},
		}
		tls_ln := tls.NewListener(ln, cfg)
		this.listenerTLS = tls_ln
		go http.Serve(tls_ln, this.mux)
		logging.Infop("HttpEndpoint: ListenTLS", logging.Pair{"Address", ln.Addr()})
	}
	return err
}

// If the server channel is full and we are unable to queue a request,
// we respond with a timeout status.
func (this *HttpEndpoint) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	request := newHttpRequest(resp, req, this.bufpool, this.server.RequestSizeCap())

	this.actives.Put(request)
	defer this.actives.Delete(request.Id().String(), false)

	defer this.doStats(request, this.server)

	if request.State() == server.FATAL {
		// There was problems creating the request: Fail it and return
		request.Failed(this.server)
		return
	}

	if request.ScanConsistency() == datastore.UNBOUNDED {
		select {
		case this.server.Channel() <- request:
			// Wait until the request exits.
			<-request.CloseNotify()
		default:
			// Buffer is full.
			resp.WriteHeader(http.StatusServiceUnavailable)
		}
	} else {
		select {
		case this.server.PlusChannel() <- request:
			// Wait until the request exits.
			<-request.CloseNotify()
		default:
			// Buffer is full.
			resp.WriteHeader(http.StatusServiceUnavailable)
		}
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

func (this *HttpEndpoint) doStats(request *httpRequest, srvr *server.Server) {

	// Update metrics:
	service_time := request.executionTime
	request_time := request.elapsedTime
	acctstore := this.server.AccountingStore()
	prepared := request.Prepared() != nil

	prepareds.RecordPreparedMetrics(request.Prepared(), request_time, service_time)
	accounting.RecordMetrics(acctstore, request_time, service_time, request.resultCount,
		request.resultSize, request.errorCount, request.warningCount, request.Type(),
		prepared, (request.State() != server.COMPLETED),
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
	this.cache.Add(http_req, http_req.Id().String(), nil)
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
