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
				TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
				MaxIdleConnsPerHost: MaxIdleConnsPerHost,
			}
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
func (c *Client) processStream(baseURL *url.URL, path string, authHandler AuthHandler, callb func(interface{}) error, data interface{}) error {
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
type StreamingFn func(bucket *Bucket)

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

func (b *Bucket) RunBucketUpdater2(streamingFn StreamingFn, notify NotifyFn) {

	b.Lock()
	if b.updater != nil {
		bu := b.updater
		b.updater = nil
		bu.Close()
	}
	b.Unlock()
	go func() {
		err := b.UpdateBucket2(streamingFn)
		if err != nil {
			if notify != nil {
				name := b.GetName()
				notify(name, err)

				// MB-49772 get rid of the deleted bucket
				p := b.pool
				b.Close()
				p.Lock()
				p.BucketMap[name] = nil
				delete(p.BucketMap, name)
				p.Unlock()
			}
			logging.Errorf("Bucket Updater exited with err %v", err)
		}
	}()
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

func (b *Bucket) UpdateBucket2(streamingFn StreamingFn) error {
	var failures int
	var returnErr error
	var poolServices PoolServices
	var updater io.ReadCloser

	defer func() {
		b.Lock()
		if b.updater == updater {
			b.updater = nil
		}
		b.Unlock()
	}()

	for {

		if failures == MAX_RETRY_COUNT {
			return errors.NewBucketUpdaterMaxErrors(returnErr)
		}

		nodes := b.Nodes()
		if len(nodes) < 1 {
			return errors.NewBucketUpdaterNoHealthyNodesFound()
		}

		streamUrl := fmt.Sprintf("%s/pools/default/bucketsStreaming/%s", b.pool.client.BaseURL, uriAdj(b.GetName()))
		logging.Infof("Bucket updater: Trying with %s", streamUrl)
		req, err := http.NewRequest("GET", streamUrl, nil)
		if err != nil {
			if isConnError(err) {
				failures++
				if failures < MAX_RETRY_COUNT {
					logging.Infof("Bucket updater : %v (Retrying %v)", err, failures)
					time.Sleep(time.Duration(failures) * STREAM_RETRY_PERIOD)
				} else {
					returnErr = errors.NewBucketUpdaterStreamingError(err)
				}
				continue
			}
			return errors.NewBucketUpdaterStreamingError(err)
		}
		req.Header.Set("User-Agent", USER_AGENT)

		// Lock here to avoid having pool closed under us.
		b.RLock()
		err = maybeAddAuth(req, b.pool.client.ah)
		b.RUnlock()
		if err != nil {
			return errors.NewBucketUpdaterAuthError(err)
		}

		res, err := doHTTPRequestForUpdate(req)
		if err != nil {
			if isConnError(err) {
				failures++
				if failures < MAX_RETRY_COUNT {
					logging.Infof("Bucket updater : %v (Retrying %v)", err, failures)
					time.Sleep(time.Duration(failures) * STREAM_RETRY_PERIOD)
				} else {
					returnErr = errors.NewBucketUpdaterStreamingError(err)
				}
				continue
			}
			return errors.NewBucketUpdaterStreamingError(err)
		}

		if res.StatusCode != 200 {
			bod, _ := ioutil.ReadAll(io.LimitReader(res.Body, 512))
			logging.Errorf("Bucket updater: Failed to connect to host, unexpected status code: %v. Body %s", res.StatusCode, bod)
			res.Body.Close()
			returnErr = errors.NewBucketUpdaterFailedToConnectToHost(res.StatusCode, bod)
			failures++
			continue
		}
		b.Lock()
		if b.updater == updater {
			b.updater = res.Body
			updater = b.updater
		} else {
			// another updater is running and we should exit cleanly
			b.Unlock()
			res.Body.Close()
			logging.Debugf("Bucket updater: New updater found for bucket: %v", b.GetName())
			return nil
		}
		b.Unlock()

		dec := json.NewDecoder(res.Body)

		tmpb := &Bucket{}
		for {
			b.Lock()
			bu := b.updater
			b.Unlock()
			if bu != updater {
				res.Body.Close()
				logging.Debugf("Bucket updater: Detected new updater for bucket: %v", b.GetName())
				return nil
			}

			err := dec.Decode(&tmpb)

			b.Lock()
			bu = b.updater
			b.Unlock()
			if bu != updater {
				logging.Debugf("Bucket updater: Detected new updater for bucket: %v", b.GetName())
				return nil
			}

			if err != nil {
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
						b.ah, false, PoolSize, PoolOverflow, b.pool.client.tlsConfig, b.Name, encrypted)

				} else {
					newcps[i] = newConnectionPool(hostport,
						b.authHandler(true /* bucket already locked */),
						false, PoolSize, PoolOverflow, b.pool.client.tlsConfig, b.Name, encrypted)
				}
			}

			b.replaceConnPools2(newcps, true /* bucket already locked */)

			tmpb.ah = b.ah
			b.vBucketServerMap = unsafe.Pointer(&tmpb.VBSMJson)
			b.nodeList = unsafe.Pointer(&tmpb.NodesJSON)
			b.Unlock()

			if streamingFn != nil {
				streamingFn(tmpb)
			}
			logging.Debugf("Bucket updater: Got new configuration for bucket %s", b.GetName())

		}
		// we are here because of an error
		failures++
		continue

	}
	return nil
}
