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
	"github.com/gorilla/mux"
)

type HttpEndpoint struct {
	server      *server.Server
	metrics     bool
	bufpool     BufferPool
	listener    net.Listener
	listenerTLS net.Listener
	mux         *mux.Router
}

const (
	servicePrefix = "/query/service"
)

func NewServiceEndpoint(server *server.Server, staticPath string, metrics bool) *HttpEndpoint {
	rv := &HttpEndpoint{
		server:  server,
		metrics: metrics,
		bufpool: NewSyncPool(server.KeepAlive()),
	}

	rv.registerHandlers(staticPath)
	return rv
}

func (this *HttpEndpoint) Listen(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err == nil {
		this.listener = ln
		go http.Serve(ln, this.mux)
	}
	return err
}

func (this *HttpEndpoint) ListenTLS(addr, certFile, keyFile string) error {
	// create tls configuration
	tlsCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", addr)
	if err == nil {
		cfg := &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
			ClientAuth:   tls.NoClientCert,
		}
		tls_ln := tls.NewListener(ln, cfg)
		this.listener = tls_ln
		go http.Serve(tls_ln, this.mux)
	}
	return err
}

// If the server channel is full and we are unable to queue a request,
// we respond with a timeout status.
func (this *HttpEndpoint) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
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

func (this *HttpEndpoint) Close() {
	if this.listener != nil {
		this.listener.Close()
		logging.Infop("HttpEndpoint.Close()", logging.Pair{"Address", this.listener.Addr()})
	}
	if this.listenerTLS != nil {
		this.listenerTLS.Close()
		logging.Infop("HttpEndpoint.Close()", logging.Pair{"Address", this.listener.Addr()})
	}
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

	registerClusterHandlers(this.mux, this.server)
	registerAccountingHandlers(this.mux, this.server)
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
