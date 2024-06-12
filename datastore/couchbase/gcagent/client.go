//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package gcagent

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/gocbcore/v10"
	"github.com/couchbase/gocbcore/v10/connstr"
	ntls "github.com/couchbase/goutils/tls"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/primitives/couchbase"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/util"
)

const (
	_CONNECTTIMEOUT = 10000 * time.Millisecond
	_KVTIMEOUT      = 2500 * time.Millisecond
	_WARMUPTIMEOUT  = 1000 * time.Millisecond
	_WARMUP         = false
	_CLOSEWAIT      = 2 * time.Minute
	_MINQUEUES      = 4
	_MAXQUEUES      = 16
	_QUEUESIZE      = 32 * 1024
	_KVBUFFERSIZE   = 16 * 1024
)

type MemcachedAuthProvider struct {
	c *Client
}

func (auth *MemcachedAuthProvider) Credentials(req gocbcore.AuthCredsRequest) (
	[]gocbcore.UserPassPair, error) {
	endpoint := req.Endpoint

	// get rid of the http:// or https:// prefix from the endpoint
	endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")
	username, password, err := cbauth.GetMemcachedServiceAuth(endpoint)
	if err != nil {
		return []gocbcore.UserPassPair{{}}, err
	}

	return []gocbcore.UserPassPair{{
		Username: username,
		Password: password,
	}}, nil
}

func (auth *MemcachedAuthProvider) SupportsNonTLS() bool {
	return true
}

func (auth *MemcachedAuthProvider) SupportsTLS() bool {
	return true
}

func (auth *MemcachedAuthProvider) Certificate(req gocbcore.AuthCertRequest) (*tls.Certificate, error) {
	// If the internal client certificate has been set, use it for client authentication.
	auth.c.mutex.RLock()
	defer auth.c.mutex.RUnlock()
	return auth.c.internalClientCert, nil
}

// Call this method with a TLS certificate file name to make communication
type Client struct {
	config             *gocbcore.AgentConfig
	transactions       *gocbcore.TransactionsManager
	rootCAs            *x509.CertPool
	mutex              sync.RWMutex
	atrLocations       map[string]gocbcore.TransactionLostATRLocation
	certs              *tls.Certificate
	agentProviders     map[string]*AgentProvider
	internalClientCert *tls.Certificate
}

func NewClient(url string, caFile, certFile, keyFile string, passphrase []byte, encryptNodeToNodeComms bool,
	clientCertAuthMandatory bool, internalClientFile, internalClientKey string, internalClientPassphrase []byte) (
	rv *Client, err error) {
	rv = &Client{}

	// network=default use internal (vs  alternative) addresses
	// http bootstrap is faster
	nurl := strings.Replace(url, "http://", "ns_server://", 1)
	options := "?network=default&bootstrap_on=http"
	if rv.config, err = agentConfig(nurl, options, rv); err != nil {
		return nil, err
	}

	if certFile != "" || caFile != "" || keyFile != "" || internalClientFile != "" || internalClientKey != "" {
		if err = rv.InitTLS(caFile, certFile, keyFile, passphrase, clientCertAuthMandatory,
			internalClientFile, internalClientKey, internalClientPassphrase); err != nil {
			return nil, err
		}
	}

	// generic provider
	rv.atrLocations = make(map[string]gocbcore.TransactionLostATRLocation, 32)
	rv.agentProviders = make(map[string]*AgentProvider, 32)

	return rv, err
}

func agentConfig(url, options string, rv *Client) (*gocbcore.AgentConfig, error) {
	config := &gocbcore.AgentConfig{}
	config.UserAgent = couchbase.USER_AGENT
	config.DefaultRetryStrategy = gocbcore.NewBestEffortRetryStrategy(nil)
	config.KVConfig.ConnectTimeout = _CONNECTTIMEOUT
	config.KVConfig.ConnectionBufferSize = _KVBUFFERSIZE
	// queue size per kv node
	config.KVConfig.MaxQueueSize = _QUEUESIZE
	// number of the queues per kv node
	config.KVConfig.PoolSize = int((util.NumCPU() + 1) / 2)
	if config.KVConfig.PoolSize < _MINQUEUES {
		config.KVConfig.MaxQueueSize += (_MINQUEUES - config.KVConfig.PoolSize) * _QUEUESIZE
	} else if config.KVConfig.PoolSize > _MAXQUEUES {
		// Limit PoolSize. If more CPU, there will be more kv nodes in cluster.
		config.KVConfig.PoolSize = _MAXQUEUES
	}
	config.IoConfig.UseCollections = true
	config.IoConfig.NetworkType = "network"
	config.SecurityConfig.Auth = &MemcachedAuthProvider{rv}
	config.SecurityConfig.TLSRootCAProvider = func() *x509.CertPool {
		return rv.TLSRootCAs()
	}
	config.SecurityConfig.NoTLSSeedNode = true
	config.SecurityConfig.UseTLS = (rv.TLSRootCAs() != nil)

	config.InternalConfig.EnableResourceUnitsTrackingHello = tenant.IsServerless()

	connSpec, err := connstr.Parse(url + options)
	if err == nil {
		err = config.FromConnStr(connSpec.String())
	}

	return config, err
}

func (c *Client) InitTransactions(txConfig *gocbcore.TransactionsConfig) (err error) {
	c.AddAtrLocation(&txConfig.CustomATRLocation)
	txConfig.LostCleanupATRLocationProvider = func() (lostAtrLocations []gocbcore.TransactionLostATRLocation, cerr error) {
		c.mutex.RLock()
		defer c.mutex.RUnlock()
		lostAtrLocations = make([]gocbcore.TransactionLostATRLocation, 0, len(c.atrLocations))
		for _, atrl := range c.atrLocations {
			lostAtrLocations = append(lostAtrLocations, atrl)
		}
		return
	}

	c.transactions, err = gocbcore.InitTransactions(txConfig)
	return err
}

func (c *Client) AddAtrLocation(atrLocation *gocbcore.TransactionATRLocation) (err error) {
	if atrLocation != nil && atrLocation.Agent != nil && atrLocation.Agent.BucketName() != "" {
		lostAtr := gocbcore.TransactionLostATRLocation{BucketName: atrLocation.Agent.BucketName(),
			ScopeName:      "_default",
			CollectionName: "_default"}

		if atrLocation.ScopeName != "" {
			lostAtr.ScopeName = atrLocation.ScopeName
		}
		if atrLocation.CollectionName != "" {
			lostAtr.CollectionName = atrLocation.CollectionName
		}
		s := lostAtr.BucketName + "." + lostAtr.ScopeName + "." + lostAtr.CollectionName
		c.mutex.Lock()
		defer c.mutex.Unlock()
		if _, ok := c.atrLocations[s]; !ok {
			c.atrLocations[s] = lostAtr
		}
	}
	return
}

func (c *Client) RemoveAtrLocation(bucketName string) (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for s, atrl := range c.atrLocations {
		if atrl.BucketName == bucketName {
			delete(c.atrLocations, s)
		}
	}
	return
}

func (c *Client) CreateAgentProvider(bucketName string) (*AgentProvider, error) {
	c.mutex.RLock()
	ap, ok := c.agentProviders[bucketName]
	c.mutex.RUnlock()
	if ok {
		return ap, nil
	}

	ap = &AgentProvider{client: c, bucketName: bucketName}
	useTLS := c.rootCAs != nil
	err := ap.CreateAgent()
	if err == nil && useTLS != (c.rootCAs != nil) {
		err = ap.Refresh() // refresh if TLS changed while creation
	}
	if err == nil {
		c.mutex.Lock()
		c.agentProviders[bucketName] = ap
		c.mutex.Unlock()
	}

	return ap, err
}

func (c *Client) Close() {
	if c.transactions != nil {
		c.transactions.Close()
	}
	c.transactions = nil
	c.mutex.Lock()
	c.rootCAs = nil
	c.internalClientCert = nil
	for n, ap := range c.agentProviders {
		delete(c.agentProviders, n)
		for s, atrl := range c.atrLocations {
			if atrl.BucketName == n {
				delete(c.atrLocations, s)
			}
		}
		ap.Agent().Close()
	}
	c.mutex.Unlock()
}

// with the KV engine encrypted.
func (c *Client) InitTLS(caFile, certFile, keyFile string, passphrase []byte, clientCertAuthMandatory bool,
	internalClientCertFile, internalClientKeyFile string, internalClientPrivateKeyPassphrase []byte) error {
	certs, err := ntls.LoadX509KeyPair(certFile, keyFile, passphrase)
	if err != nil {
		logging.Errorf("Transaction client certificates refresh failed: %v", err)
		return err
	}

	if len(caFile) == 0 {
		caFile = certFile
	}

	serverCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		logging.Errorf("Transaction client CA root certificate refresh failed: %v", err)
		return err
	}

	CA_Pool := x509.NewCertPool()
	CA_Pool.AppendCertsFromPEM(serverCert)

	// MB-52102: Include the internal client cert if n2n encryption is enabled and client certificate authentication is mandatory.
	var internalClientCert tls.Certificate
	if clientCertAuthMandatory {
		internalClientCert, err = ntls.LoadX509KeyPair(internalClientCertFile, internalClientKeyFile,
			internalClientPrivateKeyPassphrase)
		if err != nil {
			logging.Errorf("Transaction client internal client certificate refresh failed: %v", err)
			return err
		}
	}

	c.mutex.Lock()
	// Set values for certs and passphrase
	c.certs = &certs
	c.rootCAs = CA_Pool

	if clientCertAuthMandatory {
		c.internalClientCert = &internalClientCert
	}
	c.mutex.Unlock()
	logging.Infof("Transaction client certificates have been refreshed")
	return nil
}

func (c *Client) ClearTLS() {
	c.mutex.Lock()
	c.rootCAs = nil
	c.certs = nil
	c.internalClientCert = nil
	c.mutex.Unlock()
	logging.Infof("Transaction client certificates have been reset")
}

func (c *Client) TLSRootCAs() *x509.CertPool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.rootCAs
}

func (c *Client) Transactions() *gocbcore.TransactionsManager {
	return c.transactions
}
