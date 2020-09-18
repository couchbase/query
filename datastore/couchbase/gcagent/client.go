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
	_CLEANUPWINDOW    = 2500 * time.Millisecond
	_WARMUP           = false
	_WARMUPTIMEOUT    = 1000 * time.Millisecond
	_kVDURABLETIMEOUT = _KVTIMEOUT
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
	transactions  *gctx.Transactions
	rootCAs       *x509.CertPool
	agentProvider *AgentProvider
}

func NewClient(url, certFile string, defExpirationTime time.Duration) (rv *Client, err error) {
	auth := &MemcachedAuthProvider{}
	config := &gocbcore.AgentConfig{
		ConnectTimeout:       _CONNECTTIMEOUT,
		KVConnectTimeout:     _KVCONNECTTIMEOUT,
		UseCollections:       true,
		KvPoolSize:           _kVPOOLSIZE,
		MaxQueueSize:         _MAXQUEUESIZE,
		Auth:                 auth,
		DefaultRetryStrategy: gocbcore.NewBestEffortRetryStrategy(nil),
		//UseTLS:           true,
	}

	var connSpec connstr.ConnSpec
	if connSpec, err = connstr.Parse(url); err == nil {
		err = config.FromConnStr(connSpec.String())
	}

	if err != nil {
		return
	}

	txConfig := &gctx.Config{ExpirationTime: defExpirationTime,
		CleanupWindow:         _CLEANUPWINDOW,
		CleanupClientAttempts: true,
		CleanupLostAttempts:   true,
	}

	rv = &Client{config: config}
	if certFile != "" {
		err = rv.InitTLS(certFile)
	}

	if err == nil {
		rv.transactions, err = gctx.Init(txConfig)
	}

	if err == nil {
		// generic provider
		rv.agentProvider, err = rv.CreateAgentProvider("")
	}
	return
}

func (c *Client) CreateAgentProvider(bucketName string) (*AgentProvider, error) {
	config := *c.config
	config.UserAgent = bucketName
	config.BucketName = bucketName
	config.TLSRootCAProvider = func() *x509.CertPool {
		return c.rootCAs
	}
	agent, err := gocbcore.CreateAgent(&config)
	if err != nil {
		return nil, err
	}

	if _WARMUP && bucketName != "" {
		// Warm up by calling wait until ready
		warmWaitCh := make(chan struct{}, 1)
		if _, werr := agent.WaitUntilReady(
			time.Now().Add(_WARMUPTIMEOUT),
			gocbcore.WaitUntilReadyOptions{},
			func(result *gocbcore.WaitUntilReadyResult, cerr error) {
				if cerr != nil {
					err = cerr
				}
				warmWaitCh <- struct{}{}
			}); werr != nil && err == nil {
			err = werr
		}
		<-warmWaitCh
	}

	return &AgentProvider{provider: agent}, err
}

func (c *Client) AgentProvider() *AgentProvider {
	return c.agentProvider
}

func (c *Client) Agent() *gocbcore.Agent {
	return c.agentProvider.provider
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
	c.ClearTLS()
}

// with the KV engine encrypted.
func (c *Client) InitTLS(certFile string) error {
	serverCert, err := ioutil.ReadFile(certFile)
	if err != nil {
		return err
	}
	CA_Pool := x509.NewCertPool()
	CA_Pool.AppendCertsFromPEM(serverCert)
	c.rootCAs = CA_Pool
	return nil
}

func (c *Client) ClearTLS() {
	c.rootCAs = nil
}

func (c *Client) Transactions() *gctx.Transactions {
	return c.transactions
}
