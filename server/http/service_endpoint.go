//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"golang.org/x/net/http2"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	gsi "github.com/couchbase/indexing/secondary/queryport/n1ql"

	"github.com/couchbase/cbauth"
	ntls "github.com/couchbase/goutils/tls"
	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/audit"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/util"
	"github.com/gorilla/mux"
)

type userMetrics struct {
	uuid                 string
	userRequestRateLimit float64
	userPayloadLimit     float64
	userOutputLimit      float64
	userRequestsLimit    int32
	activeRequests       int32
	requestMeter         util.Meter
	payloadMeter         util.Meter
	outputMeter          util.Meter
	requestsFailures     int64
	requestRateFailures  int64
	payloadRateFailures  int64
	outputRateFailures   int64
	limitsVersion        string
}

type HttpEndpoint struct {
	server              *server.Server
	metrics             bool
	httpAddr            string
	httpsAddr           string
	cafile              string
	certFile            string
	keyFile             string
	bufpool             BufferPool
	listener            map[string]net.Listener
	listenerTLS         map[string]net.Listener
	localListener       bool
	mux                 *mux.Router
	actives             server.ActiveRequests
	options             server.ServerOptions
	connSecConfig       datastore.ConnectionSecurityConfig
	internalUser        string
	trackUsers          bool
	trackedUsers        map[string]*userMetrics
	trackedUsersVersion string
	usersLock           sync.RWMutex
}

const (
	servicePrefix   = "/query/service"
	gsiPrefix       = "/gsi"
	_MAXRETRIES     = 3
	_LISTENINTERVAL = 100 * time.Millisecond
)

var _ENDPOINT *HttpEndpoint

// surprisingly FastPool is faster than LocklessPool
var requestPool util.FastPool

func init() {
	util.NewFastPool(&requestPool, func() interface{} {
		return &httpRequest{}
	})
}

func NewServiceEndpoint(srv *server.Server, staticPath string, metrics bool,
	httpAddr, httpsAddr, caFile, certFile, keyFile string) *HttpEndpoint {
	rv := &HttpEndpoint{
		server:    srv,
		metrics:   metrics,
		httpAddr:  httpAddr,
		httpsAddr: httpsAddr,
		cafile:    caFile,
		certFile:  certFile,
		keyFile:   keyFile,
		bufpool:   NewSyncPool(srv.KeepAlive()),
		actives:   NewActiveRequests(srv),
		options:   NewHttpOptions(srv),
	}

	rv.listener = make(map[string]net.Listener, 2)
	rv.listenerTLS = make(map[string]net.Listener, 2)

	rv.connSecConfig.CertFile = certFile
	rv.connSecConfig.KeyFile = keyFile
	rv.connSecConfig.CAFile = caFile

	server.SetActives(rv.actives)
	server.SetOptions(rv.options)

	rv.registerHandlers(staticPath)
	_ENDPOINT = rv
	return rv
}

func (this *HttpEndpoint) Mux() *mux.Router {
	return this.mux
}

func (this *HttpEndpoint) SettingsCallback(f string, v interface{}) {
	switch f {
	case server.KEEPALIVELENGTH:
		val, ok := v.(int)
		if ok {
			this.bufpool.SetBufferCapacity(val)
		}
	}
}

func getNetwProtocol() map[string]string {
	protocol := make(map[string]string)

	val := server.IsIPv6()
	if val != server.TCP_OFF {
		protocol["tcp6"] = val
	}

	val = server.IsIPv4()
	if val != server.TCP_OFF {
		protocol["tcp4"] = val
	}
	return protocol
}

func (this *HttpEndpoint) localhostListen() error {
	netWs := getNetwProtocol()
	if len(netWs) == 0 {
		// Both values were set to off so we fail here.
		return fmt.Errorf(" Failed to start service: Both IPv4 and IPv6 flags were not set.")
	}

	srv := &http.Server{
		Handler:           this.mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	_, port, _ := net.SplitHostPort(this.httpAddr)
	if port == "" {
		port = "8093"
	}
	this.localListener = true
	IPv4 := server.IsIPv4()
	if IPv4 != server.TCP_OFF {
		err := this.serve(srv, "tcp4", IPv4, "127.0.0.1:"+port)
		if err != nil {
			return err
		}
	}
	IPv6 := server.IsIPv6()
	if IPv6 != server.TCP_OFF {
		err := this.serve(srv, "tcp6", IPv6, "[::1]:"+port)
		if err != nil {
			return err
		}
	}
	return nil
}

// Ipv4 and IPv6 are tri value flags that take required optional or off as values.
// Fail only in the case the listener doesnt come up when flag value is required
func (this *HttpEndpoint) serve(srv *http.Server, protocol, required, httpAddr string) error {
	ln, err := net.Listen(protocol, httpAddr)

	if err != nil {
		if required == server.TCP_REQ {
			return fmt.Errorf("Failed to start service: %v", err.Error())
		} else {
			logging.Infof("Failed to start service: %v", err.Error())
		}
	} else {
		this.listener[protocol] = ln
		go srv.Serve(ln)
		logging.Infoa(func() string { return fmt.Sprintf("HttpEndpoint: Listen Address - %v", ln.Addr()) })
	}
	return nil
}

func (this *HttpEndpoint) Listen() error {
	netWs := getNetwProtocol()
	if len(netWs) == 0 {
		// Both values were set to off so we fail here.
		return fmt.Errorf(" Failed to start service: Both IPv4 and IPv6 flags were not set.")
	}

	srv := &http.Server{
		Handler:           this.mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	for netW, val := range netWs {
		err := this.serve(srv, netW, val, this.httpAddr)
		if err != nil {
			return err
		}
	}

	return nil
}

func (this *HttpEndpoint) ListenTLS() error {

	netWs := getNetwProtocol()

	if len(netWs) == 0 {
		// Both values were set to off so we fail here.
		return fmt.Errorf(" Failed to start service: Both IPv4 and IPv6 flags were not set.")
	}

	// create tls configuration
	if (this.certFile == "" || this.cafile == "") && this.keyFile == "" {
		logging.Errorf("No certificate passed. Secure listener not brought up.")
		return nil
	}

	cbauthTLSsettings, err1 := cbauth.GetTLSConfig()
	if err1 != nil {
		return fmt.Errorf("Failed to get cbauth tls config: %v", err1.Error())
	}

	this.connSecConfig.TLSConfig = cbauthTLSsettings

	tlsCert, err := ntls.LoadX509KeyPair(this.certFile, this.keyFile,
		this.connSecConfig.TLSConfig.PrivateKeyPassphrase)
	if err != nil {
		return err
	}

	cfg := &tls.Config{
		Certificates:             []tls.Certificate{tlsCert},
		ClientAuth:               cbauthTLSsettings.ClientAuthType,
		MinVersion:               cbauthTLSsettings.MinVersion,
		CipherSuites:             cbauthTLSsettings.CipherSuites,
		PreferServerCipherSuites: cbauthTLSsettings.PreferServerCipherSuites,
	}

	if cbauthTLSsettings.ClientAuthType != tls.NoClientCert {
		caCert, err := ioutil.ReadFile(this.cafile)
		if err != nil {
			return fmt.Errorf(" Error in reading cacert file, err: %v", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		cfg.ClientCAs = caCertPool
	}

	// In the interest of allowing Go to correctly configure our HTTP2 setup,
	// we create a false server object and then configure it on that.  This
	// enables us to get an early warning if our TLS configuration is not
	// compatible with HTTP2 or could cause TLS negotiation failures.
	http2Srv := http.Server{TLSConfig: cfg}
	err2 := http2.ConfigureServer(&http2Srv, nil)
	if err2 != nil {
		logging.Errorf(" Error configuring http2, err: %v", err2)
	} else {
		cfg = http2Srv.TLSConfig
	}

	srv := &http.Server{
		Handler:           this.mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	for netW, val := range netWs {
		var ln net.Listener
		var err error

		for i := 0; i < _MAXRETRIES; i++ {
			if i != 0 {
				time.Sleep(_LISTENINTERVAL)
			}

			ln, err = net.Listen(netW, this.httpsAddr)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), "bind address already in use") {
				break
			}
		}

		if err != nil {
			if val == server.TCP_REQ {
				return fmt.Errorf("Failed to start service: %v", err.Error())
			} else {
				logging.Infof("Failed to start service: %v", err.Error())
			}
		} else {
			tls_ln := tls.NewListener(ln, cfg)
			this.listenerTLS[netW] = tls_ln
			go srv.Serve(tls_ln)
			logging.Infoa(func() string { return fmt.Sprintf("HttpEndpoint: ListenTLS Address - %v", ln.Addr()) })
		}

	}

	return nil
}

// If the server channel is full and we are unable to queue a request,
// we respond with a timeout status.
func (this *HttpEndpoint) ServeHTTP(resp http.ResponseWriter, req *http.Request) {

	// ESCAPE analysis workaround
	request := requestPool.Get().(*httpRequest)
	*request = httpRequest{}
	newHttpRequest(request, resp, req, this.bufpool, this.server.RequestSizeCap(), this.server.Namespace(), this.trackUsers)
	defer func() {
		requestPool.Put(request)
	}()

	if this.trackUsers {
		request.Loga(logging.INFO, func() string { return "Tracking user" })
		userName, domain := datastore.FirstCred(request.Credentials())
		this.usersLock.Lock()
		user := this.trackedUsers[userName]
		request.SetTracked()
		if user == nil {
			user = &userMetrics{
				uuid:           datastore.GetUserUUID(request.Credentials()),
				activeRequests: 1,
				requestMeter:   util.NewMeter(time.Minute, time.Minute),
				payloadMeter:   util.NewMeter(time.Minute, time.Minute),
				outputMeter:    util.NewMeter(time.Minute, time.Minute),
			}
			user.uuid = strings.Replace(user.uuid, "-", "_", -1)
			limits, err := cbauth.GetUserLimits(userName, domain, "query")
			if err != nil {
				s := fmt.Sprintf("No user limits found for user <ud>%v</ud> - limits not enforced", userName)
				logging.Infof(s)
				request.Loga(logging.INFO, func() string { return s })
			} else {
				user.amendLimits(limits)
				user.limitsVersion = this.trackedUsersVersion
			}
			if this.trackedUsers == nil {
				this.trackedUsers = make(map[string]*userMetrics)
			}
			this.trackedUsers[userName] = user
			this.usersLock.Unlock()
		} else {
			if this.trackedUsersVersion != user.limitsVersion {
				limits, err := cbauth.GetUserLimits(userName, domain, "query")
				if err != nil {
					s := fmt.Sprintf("No user limits found for user <ud>%v</ud> - limits not changed", userName)
					logging.Infof(s)
					request.Loga(logging.INFO, func() string { return s })
				} else {
					user.amendLimits(limits)
					user.limitsVersion = this.trackedUsersVersion
				}
			}
			this.usersLock.Unlock()
			atomic.AddInt32(&user.activeRequests, 1)
		}
		reqSize := req.ContentLength

		// if we didn't have a request size we approximate to the statement length
		if reqSize <= 0 {
			reqSize = int64(len(request.Statement()))
		}
		user.requestMeter.Mark(1, request.RequestTime())
		user.payloadMeter.Mark(reqSize, request.RequestTime())
		if user.userRequestsLimit > 0 && user.activeRequests > user.userRequestsLimit {
			atomic.AddInt64(&user.requestsFailures, 1)
			atomic.AddInt32(&user.activeRequests, -1)
			request.Fail(errors.NewServiceUserRequestExceededError())
			request.Failed(this.server)
			return
		} else if user.userRequestRateLimit > 0 && user.requestMeter.Rate() > user.userRequestRateLimit {
			atomic.AddInt64(&user.requestRateFailures, 1)
			atomic.AddInt32(&user.activeRequests, -1)
			request.Fail(errors.NewServiceUserRequestRateExceededError())
			request.Failed(this.server)
			return
		} else if user.userPayloadLimit > 0 && user.payloadMeter.Rate() > user.userPayloadLimit {
			atomic.AddInt64(&user.payloadRateFailures, 1)
			atomic.AddInt32(&user.activeRequests, -1)
			request.Fail(errors.NewServiceUserRequestSizeExceededError())
			request.Failed(this.server)
			return
		} else if user.userOutputLimit > 0 && user.outputMeter.Rate() > user.userOutputLimit {
			atomic.AddInt64(&user.outputRateFailures, 1)
			atomic.AddInt32(&user.activeRequests, -1)
			request.Fail(errors.NewServiceUserResultsSizeExceededError())
			request.Failed(this.server)
			return
		}
	} else if tenant.IsServerless() {

		// TODO TENANT throttling here is easy but may result in double waiting, once
		// for throttling and once for servicers availability.
		// A better choice would be near server.go:service_request(), after the queueing
		// due to servicers load has happened.
		// this would shorten the overall wait due to throttling, but has the major disadvantage
		// that it is quite complicated to code and might badly affect queue throughput.
		// Explore if this code is acceptable or conversely throttling after queuing can be
		// coded efficiently.
		bucket := ""
		path := algebra.ParseQueryContext(request.QueryContext())
		if len(path) > 1 {
			bucket = path[1]
		}
		userName, _ := datastore.FirstCred(request.Credentials())
		request.Loga(logging.INFO, func() string { return fmt.Sprintf("Checking throttling for %v", userName) })
		ctx, d, err := tenant.Throttle(datastore.IsAdmin(request.Credentials()), userName, bucket,
			datastore.GetUserBuckets(request.Credentials()), this.server.RequestTimeout(request.Timeout()))
		request.SetThrottleTime(d)
		request.Loga(logging.INFO, func() string { return fmt.Sprintf("Throttle time: %v", d) })
		if err != nil {
			request.Fail(err)
			request.Failed(this.server)
			return
		}
		if !request.Alive() {
			request.Fail(errors.NewServiceNoClientError())
			request.Failed(this.server)
			return
		}
		request.SetTenantCtx(ctx)
	}

	defer this.doStats(request, this.server)

	// check and act on this first to keep possible load as low as possible in the event of unmetered repeat attempts
	// new transactions can't be started as the BEGIN doesn't carry a transaction ID and is ejected here
	// invalid transaction IDs do pass through here but will be caught in server processing
	if this.server.ShuttingDown() && request.TxId() == "" {
		logging.Infof("Incoming request from '%v' rejected during service shutdown.", req.RemoteAddr)
		if this.server.ShutDown() {
			request.Fail(errors.NewServiceShutDownError())
		} else {
			request.Fail(errors.NewServiceShuttingDownError())
		}
		request.Failed(this.server)
		return
	}

	this.actives.Put(request)
	defer this.actives.Delete(request.Id().String(), false, nil)
	request.Loga(logging.INFO, func() string { return "Request active" })

	if request.State() == server.FATAL {

		// There was problems creating the request: Fail it and return
		request.Failed(this.server)
		return
	}

	var res bool
	if request.ScanConsistency() == datastore.UNBOUNDED && request.TxId() == "" {
		res = this.server.ServiceRequest(request)
	} else {
		res = this.server.PlusServiceRequest(request)
	}

	if !res {
		resp.WriteHeader(http.StatusServiceUnavailable)
	}
}

func (this *HttpEndpoint) Close() error {
	serr := []error{}
	for netW, listener := range this.listener {
		if listener != nil {
			err := this.closeListener(listener)
			if err != nil {
				serr = append(serr, err)
			} else {
				this.listener[netW] = nil
				delete(this.listener, netW)
			}
		}
	}
	this.localListener = false
	if len(serr) != 0 {
		return fmt.Errorf("HTTP Listener errors: %v", serr)
	}
	return nil
}

func (this *HttpEndpoint) CloseTLS() error {
	serr := []error{}
	for netW, listener := range this.listenerTLS {
		if listener != nil {
			err := this.closeListener(listener)
			if err != nil {
				serr = append(serr, err)
			} else {
				this.listenerTLS[netW] = nil
				delete(this.listenerTLS, netW)
			}
		}
	}
	if len(serr) != 0 {
		return fmt.Errorf("TLS Listener errors: %v", serr)
	}
	return nil
}

func (this *HttpEndpoint) closeListener(l net.Listener) error {
	var err error

	if l != nil {
		for i := 0; i < _MAXRETRIES; i++ {
			if i != 0 {
				time.Sleep(_LISTENINTERVAL)
			}
			err = l.Close()
			if err == nil {
				break
			}
		}
		logging.Infoa(func() string {
			return fmt.Sprintf("HttpEndpoint: close listener, Address - %v, Error - %v", l.Addr(), err)
		})
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

	this.mux.Handle(gsiPrefix+"/getInternalVersion", newAdminAuthHandlerWrapper(this, gsi.NewInternalVersionHandler()))
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

func (this *HttpEndpoint) SetupSSL() error {

	err := cbauth.RegisterConfigRefreshCallback(func(configChange uint64) error {
		// Both flags could be set here.
		settingsUpdated := false
		if (configChange & cbauth.CFG_CHANGE_CERTS_TLSCONFIG) != 0 {
			logging.Infof(" Certificates have been refreshed by ns server ")
			closeErr := this.CloseTLS()

			if closeErr != nil && !strings.Contains(strings.ToLower(closeErr.Error()), "closed network connection & use") {
				logging.Errora(func() string {
					return fmt.Sprintf("ERROR: Closing TLS listener - %s", closeErr.Error())
				})
				return errors.NewAdminEndpointError(closeErr, "error closing tls listenener")
			}

			tlsErr := this.ListenTLS()
			if tlsErr != nil {
				logging.Errora(func() string {
					return fmt.Sprintf("ERROR: Starting TLS listener - %s", tlsErr.Error())
				})
				return errors.NewAdminEndpointError(tlsErr, "error starting tls listener")

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

			// For strict TLS, stop the non encrypted listeners and restart on localhost only
			// For non strict, stop the localhost listeners, if any, and restart on all addresses
			if this.connSecConfig.ClusterEncryptionConfig.DisableNonSSLPorts == true {
				closeErr := this.Close()
				if closeErr != nil {
					logging.Errora(func() string {
						return fmt.Sprintf("Closing HTTP listener failed- %s", closeErr.Error())
					})
				}
				listenErr := this.localhostListen()
				if listenErr != nil {
					logging.Errora(func() string {
						return fmt.Sprintf("Starting localhost HTTP listener failed - %s", listenErr.Error())
					})
				}
			} else {
				if this.localListener {
					closeErr := this.Close()
					if closeErr != nil {
						logging.Errora(func() string {
							return fmt.Sprintf("Closing HTTP listener failed- %s", closeErr.Error())
						})
					}
				}
				if len(this.listener) == 0 {
					listenErr := this.Listen()
					if listenErr != nil {
						logging.Errora(func() string {
							return fmt.Sprintf("Starting HTTP listener failed - %s", listenErr.Error())
						})
					}
				}
			}

			// Temporary log message.
			logging.Infof("Updating node-to-node encryption level: %+v", cryptoConfig)
			settingsUpdated = true

		}

		if (configChange & cbauth.CFG_CHANGE_USER_LIMITS) != 0 {
			limitsConfig, err := cbauth.GetLimitsConfig()
			if err != nil {
				logging.Errorf("unable to retrieve User Limits settings: %v", err)
				return errors.NewAdminEndpointError(err, "unable to retrieve User Limits settings")
			}
			this.trackUsers = limitsConfig.EnforceLimits
			if this.trackUsers {
				this.usersLock.Lock()
				this.trackedUsersVersion = limitsConfig.UserLimitsVersion
				this.usersLock.Unlock()
			} else {

				// get rid of user cache:
				this.usersLock.Lock()
				for u, _ := range this.trackedUsers {
					delete(this.trackedUsers, u)
				}
				this.usersLock.Unlock()
			}
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
			distributed.RemoteAccess().SetConnectionSecurityConfig(this.connSecConfig.CAFile, this.connSecConfig.CertFile,
				this.connSecConfig.ClusterEncryptionConfig.EncryptData)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (this *HttpEndpoint) doStats(request *httpRequest, srvr *server.Server) {

	// Update metrics:
	service_time := request.executionTime
	request_time := request.elapsedTime
	transaction_time := request.transactionElapsedTime
	prepared := request.Prepared() != nil

	prepareds.RecordPreparedMetrics(request.Prepared(), request_time, service_time)
	accounting.RecordMetrics(request_time, service_time, transaction_time, request.resultCount,
		request.resultSize, request.GetErrorCount(), request.GetWarningCount(), request.Type(),
		prepared, (request.State() != server.COMPLETED),
		int(request.PhaseOperator(execution.INDEX_SCAN)),
		int(request.PhaseOperator(execution.PRIMARY_SCAN)),
		string(request.ScanConsistency()))

	request.CompleteRequest(request_time, service_time, transaction_time, request.resultCount,
		request.resultSize, request.GetErrorCount(), request.req, srvr)
	if this.trackUsers {
		userName, _ := datastore.FirstCred(request.Credentials())
		this.usersLock.RLock()
		user := this.trackedUsers[userName]
		this.usersLock.RUnlock()
		if user != nil {
			atomic.AddInt32(&user.activeRequests, -1)
			user.outputMeter.Mark(int64(request.resultSize), request.RequestTime())
		}
	}

	audit.Submit(request)
}

func ServicePrefix() string {
	return servicePrefix
}

func (this *userMetrics) amendLimits(vars map[string]int) {
	val, ok := vars["num_concurrent_requests"]
	if ok && val >= 0 {
		this.userRequestsLimit = int32(val)
	} else {
		this.userRequestsLimit = 0
	}
	val, ok = vars["num_queries_per_min"]
	if ok && val >= 0 {
		userRequestRateLimit := float64(val)
		if userRequestRateLimit != this.userRequestRateLimit {
			this.userRequestRateLimit = userRequestRateLimit
			this.requestMeter.Reset()
		}
	} else {
		this.userRequestRateLimit = 0
	}
	val, ok = vars["ingress_mib_per_min"]
	if ok && val >= 0 {
		userPayloadLimit := float64(val) * util.MiB
		if userPayloadLimit != this.userPayloadLimit {
			this.userPayloadLimit = userPayloadLimit
			this.payloadMeter.Reset()
		}
	} else {
		this.userPayloadLimit = 0
	}
	val, ok = vars["egress_mib_per_min"]
	if ok && val >= 0 {
		userOutputLimit := float64(val) * util.MiB
		if userOutputLimit != this.userOutputLimit {
			this.userOutputLimit = userOutputLimit
			this.outputMeter.Reset()
		}
	} else {
		this.userOutputLimit = 0
	}
}

// activeHttpRequests implements server.ActiveRequests for http requests
type activeHttpRequests struct {
	cache  *util.GenCache
	server *server.Server
}

func NewActiveRequests(server *server.Server) server.ActiveRequests {
	return &activeHttpRequests{
		cache:  util.NewGenCache(-1),
		server: server,
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

func (this *activeHttpRequests) Delete(id string, stop bool, f func(r server.Request) bool) bool {
	this.cache.DeleteWithCheck(id, func(e interface{}) bool {
		r := e.(server.Request)
		if f != nil && !f(r) {
			return false
		}
		if stop {
			req := e.(*httpRequest)
			req.Stop(server.STOPPED)
		}
		return true
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

func (this *activeHttpRequests) Load() int {
	return this.server.Load()
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
