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
	"strconv"
	"strings"
	"time"

	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/logging"
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
}

const (
	servicePrefix = "/query/service"
)

func NewServiceEndpoint(server *server.Server, staticPath string, metrics bool,
	httpAddr, httpsAddr, certFile, keyFile string) *HttpEndpoint {
	rv := &HttpEndpoint{
		server:    server,
		metrics:   metrics,
		httpAddr:  httpAddr,
		httpsAddr: httpsAddr,
		certFile:  certFile,
		keyFile:   keyFile,
		bufpool:   NewSyncPool(server.KeepAlive()),
	}

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
	// Content negotiation
	if !this.doContentNegotiation(resp, req) {
		resp.WriteHeader(http.StatusNotAcceptable)
		return
	}

	request := newHttpRequest(resp, req, this.bufpool)

	defer this.doStats(request)

	if request.State() == server.FATAL {
		// There was problems creating the request: Fail it and return
		request.Failed(this.server)
		return
	}

	select {
	case this.server.Channel() <- request:
		// Wait until the request exits.
		<-request.CloseNotify()
	default:
		// Timeout.
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

	// Handle static endpoint
	this.mux.Handle("/", (http.FileServer(http.Dir(staticPath))))

	this.mux.Handle(servicePrefix, this).
		Methods("GET", "POST")

	// TODO: Deprecate (remove) this binding
	this.mux.Handle("/query", this).
		Methods("GET", "POST")

	this.registerClusterHandlers()
	this.registerAccountingHandlers()
}

func (this *HttpEndpoint) doStats(request *httpRequest) {
	// Update metrics:
	service_time := time.Since(request.ServiceTime())
	request_time := time.Since(request.RequestTime())
	acctstore := this.server.AccountingStore()
	accounting.RecordMetrics(acctstore, request_time, service_time, request.resultCount,
		request.resultSize, request.errorCount, request.warningCount, request.Statement())
}

func GetServiceURL(host string, port int) string {
	urlParts := []string{"http://", host, ":", strconv.Itoa(port), servicePrefix}
	return strings.Join(urlParts, "")
}

const acceptType = "application/json"
const versionTag = "version="
const version = acceptType + "; " + versionTag + util.VERSION

func (this *HttpEndpoint) doContentNegotiation(resp http.ResponseWriter, req *http.Request) bool {
	// set content type to current version
	resp.Header().Set("Content-Type", version)
	accept := req.Header["Accept"]
	// if no media type specified, default to current version
	if accept == nil || accept[0] == "*/*" {
		return true
	}
	desiredContent := accept[0]
	// media type must be application/json at least
	if !strings.HasPrefix(desiredContent, acceptType) {
		return false
	}
	versionIndex := strings.Index(desiredContent, versionTag)
	// no version specified, default to current version
	if versionIndex == -1 {
		return true
	}
	// check if requested version is supported
	requestVersion := desiredContent[versionIndex+len(versionTag):]
	if requestVersion >= util.MIN_VERSION && requestVersion <= util.VERSION {
		resp.Header().Set("Content-Type", desiredContent)
		return true
	}
	return false
}
