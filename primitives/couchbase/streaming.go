//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// package couchbase provides low level access to the KV store and the orchestrator
package couchbase

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
)

// Use this client for reading from streams that should be open for an extended duration.
var HTTPClientForStreaming = &http.Client{Transport: HTTPTransport, Timeout: 0}
var clientForStreaming *http.Client

// This version of doHTTPRequest is for requests where the response connection is held open
// for an extended duration since line is a new and significant output.
//
// The ordinary version of this method expects the results to arrive promptly, and
// therefore use an HTTP client with a timeout. This client is not suitable
// for streaming use.
func doHTTPRequestForStreaming(req *http.Request) (*http.Response, error) {
	var err error
	var res *http.Response

	// we need a client that ignores certificate errors, since we self-sign
	// our certs
	if clientForStreaming == nil && req.URL.Scheme == "https" {
		var tr *http.Transport

		if skipVerify {
			tr = &http.Transport{
				TLSClientConfig:     &tls.Config{},
				MaxIdleConnsPerHost: MaxIdleConnsPerHost,
			}

			tr.TLSClientConfig.InsecureSkipVerify = skipVerify
		} else {
			// Handle cases with cert
			cfg, err := ClientConfigForX509(caFile, certFile, keyFile, privateKeyPassphrase)
			if err != nil {
				return nil, err
			}

			tr = &http.Transport{
				TLSClientConfig:     cfg,
				MaxIdleConnsPerHost: MaxIdleConnsPerHost,
			}
		}

		clientForStreaming = &http.Client{Transport: tr, Timeout: 0}

	} else if clientForStreaming == nil {
		clientForStreaming = HTTPClientForStreaming
	}

	for i := 1; i <= HTTP_MAX_RETRY; i++ {
		res, err = clientForStreaming.Do(req)
		if err != nil && isHttpConnError(err) {
			// exclude first and last
			if i > 1 && i < HTTP_MAX_RETRY {
				time.Sleep(HTTP_RETRY_PERIOD)
			}
			continue
		}
		break
	}

	if err != nil {
		return nil, err
	}

	return res, err
}

func (c *Client) ProcessStream(path string, callb func(interface{}) error, data interface{}) error {
	return c.processStream(c.BaseURL, path, c.ah, callb, data)
}

// Based on code in http://src.couchbase.org/source/xref/trunk/goproj/src/github.com/couchbase/indexing/secondary/dcp/pools.go#309
func (c *Client) processStream(baseURL *url.URL, path string, authHandler AuthHandler, callb func(interface{}) error,
	data interface{}) error {

	var requestUrl string

	if q := strings.Index(path, "?"); q > 0 {
		requestUrl = baseURL.Scheme + "://" + baseURL.Host + path[:q] + "?" + path[q+1:]
	} else {
		requestUrl = baseURL.Scheme + "://" + baseURL.Host + path
	}

	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", USER_AGENT)

	err = maybeAddAuth(req, authHandler)
	if err != nil {
		return err
	}

	res, err := doHTTPRequestForStreaming(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		bod, _ := ioutil.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("HTTP error %v getting %q: %s",
			res.Status, requestUrl, bod)
	}

	reader := bufio.NewReader(res.Body)
	for {
		bs, err := reader.ReadBytes('\n')
		if err != nil {
			return err
		}
		if len(bs) == 1 && bs[0] == '\n' {
			continue
		}

		err = json.Unmarshal(bs, data)
		if err != nil {
			return err
		}
		err = callb(data)
		if err != nil {
			return err
		}
	}
	return nil
}

// Bucket auto-updater gets the latest version of the bucket config from
// the server. If the configuration has changed then updated the local
// bucket information. If the bucket has been deleted then notify anyone
// who is holding a reference to this bucket

const MAX_RETRY_COUNT = 5
const DISCONNECT_PERIOD = 120 * time.Second
const STREAM_RETRY_PERIOD = 100 * time.Millisecond

type NotifyFn func(bucket string, err error)
type StreamingFn func(bucket *Bucket, msgPrefix string) bool

// Use TCP keepalive to detect half close sockets
var updaterTransport http.RoundTripper = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	Dial: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).Dial,
}

var updaterHTTPClient = &http.Client{Transport: updaterTransport}

func doHTTPRequestForUpdate(req *http.Request) (*http.Response, error) {

	var err error
	var res *http.Response

	for i := 1; i <= HTTP_MAX_RETRY; i++ {
		res, err = updaterHTTPClient.Do(req)
		if err != nil && isHttpConnError(err) {
			// exclude first and last
			if i > 1 && i < HTTP_MAX_RETRY {
				time.Sleep(HTTP_RETRY_PERIOD)
			}
			continue
		}
		break
	}

	if err != nil {
		return nil, err
	}

	return res, err
}

var updaterId int32

func (b *Bucket) RunBucketUpdater2(streamingFn StreamingFn, notify NotifyFn) bool {

	b.Lock()
	rv := !b.closed
	if b.updater != nil {
		b.updater.Close()
		b.updater = nil
	}
	b.Unlock()
	if rv {
		go func() {
			id := atomic.AddInt32(&updaterId, 1) & 0xffff
			name := b.GetName()
			abName := name
			if len(abName) > 8 {
				abName = abName[0:4] + abName[len(abName)-4:]
			} else {
				abName = abName + "________"
			}
			msgPrefix := fmt.Sprintf("[%p:%.8s:%s:%04x] Updater:", b, abName, b.GetAbbreviatedUUID(), id)
			err := b.UpdateBucket2(msgPrefix, streamingFn)
			if err != nil {
				if notify != nil {
					notify(name, err)

					// MB-49772 get rid of the deleted bucket
					p := b.pool
					b.Close()
					p.Lock()
					p.BucketMap[name] = nil
					delete(p.BucketMap, name)
					p.Unlock()
				}
				if err.Code() != errors.E_BUCKET_UPDATER_EP_NOT_FOUND {
					logging.Errorf("%s (%s) exited with: %v", msgPrefix, name, err)
				}
			}
		}()
	}
	return rv
}

func (b *Bucket) replaceConnPools2(with []*connectionPool, bucketLocked bool) {
	if !bucketLocked {
		b.Lock()
		defer b.Unlock()
	}
	old := b.connPools
	b.connPools = unsafe.Pointer(&with)
	if old != nil {
		for _, pool := range *(*[]*connectionPool)(old) {
			if pool != nil && pool.inUse == false {
				pool.Close()
			}
		}
	}
	return
}

func (b *Bucket) UpdateBucket2(msgPrefix string, streamingFn StreamingFn) errors.Error {
	var failures int
	var returnErr error
	var poolServices PoolServices
	var updater io.ReadCloser

	defer func() {
		log := false
		b.Lock()
		if b.updater == updater {
			b.updater = nil
			log = true
		}
		b.Unlock()
		if log {
			logging.Debugf("%s Resetting b.updater (%p) on exit", msgPrefix, updater)
		}
	}()

	for {

		if failures == MAX_RETRY_COUNT {
			return errors.NewBucketUpdaterMaxErrors(returnErr)
		}

		nodes := b.Nodes()
		if len(nodes) < 1 {
			return errors.NewBucketUpdaterNoHealthyNodesFound()
		}

		streamUrl := b.pool.client.BaseURL.JoinPath("/pools/default/bucketsStreaming/", uriAdj(b.GetName())).String()
		logging.Infof("%s Streaming %s", msgPrefix, streamUrl)
		logging.Debuga(func() string {
			p := "<nil>"
			if updater != nil {
				p = fmt.Sprintf("%p", updater)
			}
			return fmt.Sprintf("%s updater:%s failures:%d", msgPrefix, p, failures)
		})
		req, err := http.NewRequest("GET", streamUrl, nil)
		if err != nil {
			logging.Infof("%s Error creating request: %v", msgPrefix, err)
			return errors.NewBucketUpdaterStreamingError(err)
		}
		req.Header.Set("User-Agent", USER_AGENT)

		// Lock here to avoid having pool closed under us.
		b.RLock()
		err = maybeAddAuth(req, b.pool.client.ah)
		b.RUnlock()
		if err != nil {
			logging.Infof("%s Error setting request auth: %v", msgPrefix, err)
			return errors.NewBucketUpdaterAuthError(err)
		}

		res, err := doHTTPRequestForUpdate(req)
		if err != nil {
			if isConnError(err) {
				failures++
				if failures < MAX_RETRY_COUNT {
					logging.Infof("%s %v (Retrying %v)", msgPrefix, err, failures)
					time.Sleep(time.Duration(failures) * STREAM_RETRY_PERIOD)
				} else {
					returnErr = errors.NewBucketUpdaterStreamingError(err)
				}
				continue
			}
			return errors.NewBucketUpdaterStreamingError(err)
		} else if res.StatusCode == http.StatusNotFound {
			// bucket has been removed, shut down
			logging.Infof("%s Streaming endpoint not found. Exiting.", msgPrefix)
			return errors.NewBucketUpdaterEndpointNotFoundError()
		} else if res.StatusCode != http.StatusOK {
			bod, _ := ioutil.ReadAll(io.LimitReader(res.Body, 512))
			logging.Errorf("%s Status %v - %s", msgPrefix, res.StatusCode, bod)
			res.Body.Close()
			returnErr = errors.NewBucketUpdaterFailedToConnectToHost(res.StatusCode, bod)
			failures++
			continue
		}

		b.Lock()
		if b.updater != updater {
			// another updater is running and we should exit cleanly
			b.Unlock()
			res.Body.Close()
			logging.Infof("%s New updater found", msgPrefix)
			return nil
		} else if b.closed {
			b.Unlock()
			res.Body.Close()
			logging.Infof("%s Bucket closed", msgPrefix)
			return nil
		}
		b.updater = res.Body
		updater = b.updater
		b.Unlock()

		dec := json.NewDecoder(res.Body)

		tmpb := &Bucket{}
		for {
			b.RLock()
			terminate := b.updater != updater || b.closed
			b.RUnlock()
			if terminate {
				res.Body.Close()
				logging.Infof("%s Stopping (changed:%v,closed:%v)", msgPrefix, b.updater != updater, b.closed)
				return nil
			}

			err := dec.Decode(&tmpb)

			b.RLock()
			terminate = b.updater != updater || b.closed
			b.RUnlock()
			if terminate {
				logging.Infof("%s Stopping (changed:%v,closed:%v)", msgPrefix, b.updater != updater, b.closed)
				return nil
			}

			if err != nil {
				logging.Debugf("%s Decode error: %v", msgPrefix, err)
				returnErr = err
				res.Body.Close()
				break
			}

			// if we got here, reset failure count
			failures = 0

			if b.pool.client.tlsConfig != nil {
				poolServices, err = b.pool.client.GetPoolServices("default")
				if err != nil {
					returnErr = err
					res.Body.Close()
					break
				}
			}

			b.Lock()

			// mark all the old connection pools for deletion
			pools := b.getConnPools(true /* already locked */)
			for _, pool := range pools {
				if pool != nil {
					pool.inUse = false
				}
			}

			newcps := make([]*connectionPool, len(tmpb.VBSMJson.ServerList))
			for i := range newcps {
				// get the old connection pool and check if it is still valid
				pool := b.getConnPoolByHost(tmpb.VBSMJson.ServerList[i], true /* bucket already locked */)
				if pool != nil && pool.inUse == false && pool.tlsConfig == b.pool.client.tlsConfig {
					// if the hostname and index is unchanged then reuse this pool
					newcps[i] = pool
					pool.inUse = true
					continue
				}
				// else create a new pool
				var encrypted bool
				hostport := tmpb.VBSMJson.ServerList[i]
				if b.pool.client.tlsConfig != nil {
					hostport, encrypted, err = MapKVtoSSLExt(hostport, &poolServices, b.pool.client.disableNonSSLPorts)
					if err != nil {
						b.Unlock()
						return errors.NewBucketUpdaterMappingError(err)
					}
				}
				if b.ah != nil {
					newcps[i] = newConnectionPool(hostport,
						b.ah, AsynchronousCloser, PoolSize, PoolOverflow, b.pool.client.tlsConfig, b.Name, encrypted)

				} else {
					newcps[i] = newConnectionPool(hostport,
						b.authHandler(true /* bucket already locked */),
						AsynchronousCloser, PoolSize, PoolOverflow, b.pool.client.tlsConfig, b.Name, encrypted)
				}
			}

			b.replaceConnPools2(newcps, true /* bucket already locked */)

			tmpb.ah = b.ah
			b.vBucketServerMap = unsafe.Pointer(&tmpb.VBSMJson)
			b.nodeList = unsafe.Pointer(&tmpb.NodesJSON)
			b.Unlock()

			if streamingFn != nil {
				if !streamingFn(tmpb, msgPrefix) {
					return nil
				}
			}
			logging.Debugf("%s Got new configuration", msgPrefix)

		}
		// we are here because of an error
		failures++
		continue

	}
	return nil
}
