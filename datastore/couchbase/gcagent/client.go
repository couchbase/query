//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

// +build enterprise

package gcagent

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/gocbcore/v9"
	"github.com/couchbase/gocbcore/v9/connstr"
	gctx "github.com/couchbaselabs/gocbcore-transactions"
)

const (
	_CONNECTTIMEOUT   = 10000 * time.Millisecond
	_KVCONNECTTIMEOUT = 7000 * time.Millisecond
	_KVTIMEOUT        = 2500 * time.Millisecond
	_WARMUPTIMEOUT    = 1000 * time.Millisecond
	_WARMUP           = false
	_CLOSEWAIT        = 2 * time.Minute
	_kVPOOLSIZE       = 8
	_MAXQUEUESIZE     = 32 * 1024
)

type MemcachedAuthProvider struct {
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
}

// Call this method with a TLS certificate file name to make communication
type Client struct {
	config        *gocbcore.AgentConfig
	sslConfig     *gocbcore.AgentConfig
	transactions  *gctx.Transactions
	rootCAs       *x509.CertPool
	agentProvider *AgentProvider
	mutex         sync.RWMutex
}

func NewClient(url, certFile string) (rv *Client, err error) {
	var connSpec *connstr.ConnSpec

	rv = &Client{}
	if rv.config, connSpec, err = agentConfig(url); err != nil {
		return nil, err
	}

	// create SSL agent config file
	if len(connSpec.Addresses) > 0 {
		surl := "couchbases://" + connSpec.Addresses[0].Host
		if rv.sslConfig, _, err = agentConfig(surl); err != nil {
			return nil, err
		}
	}

	if certFile != "" {
		if err = rv.InitTLS(certFile); err != nil {
			return nil, err
		}
	}

	// generic provider
	rv.agentProvider, err = rv.CreateAgentProvider("")

	return rv, err
}

func agentConfig(url string) (config *gocbcore.AgentConfig, cspec *connstr.ConnSpec, err error) {
	config = &gocbcore.AgentConfig{
		ConnectTimeout:       _CONNECTTIMEOUT,
		KVConnectTimeout:     _KVCONNECTTIMEOUT,
		UseCollections:       true,
		KvPoolSize:           _kVPOOLSIZE,
		MaxQueueSize:         _MAXQUEUESIZE,
		Auth:                 &MemcachedAuthProvider{},
		DefaultRetryStrategy: gocbcore.NewBestEffortRetryStrategy(nil),
	}

	var connSpec connstr.ConnSpec
	if connSpec, err = connstr.Parse(url); err == nil {
		err = config.FromConnStr(connSpec.String())
	}

	return config, &connSpec, err
}

func (c *Client) InitTransactions(txConfig *gctx.Config) (err error) {
	c.transactions, err = gctx.Init(txConfig)
	return err
}

func (c *Client) CreateAgentProvider(bucketName string) (*AgentProvider, error) {
	ap := &AgentProvider{client: c, bucketName: bucketName}
	err := ap.CreateOrRefreshAgent()
	return ap, err
}

func (c *Client) AgentProvider() *AgentProvider {
	return c.agentProvider
}

func (c *Client) Agent() *gocbcore.Agent {
	return c.agentProvider.Agent()
}

func (c *Client) Close() {
	if c.agentProvider != nil {
		c.agentProvider.Close()
	}
	if c.transactions != nil {
		c.transactions.Close()
	}
	c.agentProvider = nil
	c.transactions = nil
	c.mutex.Lock()
	c.rootCAs = nil
	c.mutex.Unlock()
}

// with the KV engine encrypted.
func (c *Client) InitTLS(certFile string) error {
	serverCert, err := ioutil.ReadFile(certFile)
	if err != nil {
		return err
	}
	CA_Pool := x509.NewCertPool()
	CA_Pool.AppendCertsFromPEM(serverCert)
	c.mutex.Lock()
	c.rootCAs = CA_Pool
	c.mutex.Unlock()
	if c.agentProvider != nil {
		return c.agentProvider.Refresh()
	}
	return nil
}

func (c *Client) ClearTLS() {
	c.mutex.Lock()
	c.rootCAs = nil
	c.mutex.Unlock()
	if c.agentProvider != nil {
		c.agentProvider.Refresh()
	}
}

func (c *Client) TLSRootCAs() *x509.CertPool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.rootCAs
}

func (c *Client) Transactions() *gctx.Transactions {
	return c.transactions
}
