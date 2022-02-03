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
	gctx "github.com/couchbase/gocbcore-transactions"
	"github.com/couchbase/gocbcore/v10"
	"github.com/couchbase/gocbcore/v10/connstr"
	ntls "github.com/couchbase/goutils/tls"
	"github.com/couchbase/query/logging"
)

const (
	_CONNECTTIMEOUT = 10000 * time.Millisecond
	_KVTIMEOUT      = 2500 * time.Millisecond
	_WARMUPTIMEOUT  = 1000 * time.Millisecond
	_WARMUP         = false
	_CLOSEWAIT      = 2 * time.Minute
	_kVPOOLSIZE     = 8
	_MAXQUEUESIZE   = 32 * 1024
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
	return nil, nil
	// At present when we act as client we use Credentials, not certificates.
	// return auth.c.certs, nil
}

// Call this method with a TLS certificate file name to make communication
type Client struct {
	config         *gocbcore.AgentConfig
	transactions   *gctx.Manager
	rootCAs        *x509.CertPool
	mutex          sync.RWMutex
	atrLocations   map[string]gctx.LostATRLocation
	certs          *tls.Certificate
	agentProviders map[string]*AgentProvider
}

func NewClient(url string, caFile, certFile, keyFile string, passphrase []byte) (rv *Client, err error) {
	rv = &Client{}

	// network=default use internal (vs  alternative) addresses
	// http bootstrap is faster
	nurl := strings.Replace(url, "http://", "ns_server://", 1)
	options := "?network=default&bootstrap_on=http"
	if rv.config, err = agentConfig(nurl, options, rv); err != nil {
		return nil, err
	}

	if certFile != "" || caFile != "" || keyFile != "" {
		if err = rv.InitTLS(caFile, certFile, keyFile, passphrase); err != nil {
			return nil, err
		}
	}

	// generic provider
	rv.atrLocations = make(map[string]gctx.LostATRLocation, 32)
	rv.agentProviders = make(map[string]*AgentProvider, 32)

	return rv, err
}

func agentConfig(url, options string, rv *Client) (*gocbcore.AgentConfig, error) {
	config := &gocbcore.AgentConfig{}
	config.DefaultRetryStrategy = gocbcore.NewBestEffortRetryStrategy(nil)
	config.KVConfig.ConnectTimeout = _CONNECTTIMEOUT
	config.KVConfig.PoolSize = _kVPOOLSIZE
	config.KVConfig.MaxQueueSize = _MAXQUEUESIZE
	config.IoConfig.UseCollections = true
	config.IoConfig.NetworkType = "network"
	config.SecurityConfig.Auth = &MemcachedAuthProvider{rv}
	config.SecurityConfig.TLSRootCAProvider = func() *x509.CertPool {
		return rv.TLSRootCAs()
	}
	config.SecurityConfig.NoTLSSeedNode = true
	config.SecurityConfig.UseTLS = (rv.TLSRootCAs() != nil)

	connSpec, err := connstr.Parse(url + options)
	if err == nil {
		err = config.FromConnStr(connSpec.String())
	}

	return config, err
}

func (c *Client) InitTransactions(txConfig *gctx.Config) (err error) {
	c.AddAtrLocation(&txConfig.CustomATRLocation)
	txConfig.LostCleanupATRLocationProvider = func() (lostAtrLocations []gctx.LostATRLocation, cerr error) {
		c.mutex.RLock()
		defer c.mutex.RUnlock()
		lostAtrLocations = make([]gctx.LostATRLocation, 0, len(c.atrLocations))
		for _, atrl := range c.atrLocations {
			lostAtrLocations = append(lostAtrLocations, atrl)
		}
		return
	}

	c.transactions, err = gctx.Init(txConfig)
	return err
}

func (c *Client) AddAtrLocation(atrLocation *gctx.ATRLocation) (err error) {
	if atrLocation != nil && atrLocation.Agent != nil && atrLocation.Agent.BucketName() != "" {
		lostAtr := gctx.LostATRLocation{BucketName: atrLocation.Agent.BucketName(),
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
func (c *Client) InitTLS(caFile, certFile, keyFile string, passphrase []byte) error {
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
	c.mutex.Lock()
	// Set values for certs and passphrase
	c.certs = &certs
	c.rootCAs = CA_Pool
	c.mutex.Unlock()
	logging.Infof("Transaction client certificates have been refreshed")
	return nil
}

func (c *Client) ClearTLS() {
	c.mutex.Lock()
	c.rootCAs = nil
	c.certs = nil
	c.mutex.Unlock()
	logging.Infof("Transaction client certificates have been reset")
}

func (c *Client) TLSRootCAs() *x509.CertPool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.rootCAs
}

func (c *Client) Transactions() *gctx.Manager {
	return c.transactions
}
