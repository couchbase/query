//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// This implements remote system keyspace access for the REST based http package

//go:build enterprise

package http

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	goErr "errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/couchbase/cbauth"
	ntls "github.com/couchbase/goutils/tls"
	"github.com/couchbase/query/clustering"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/primitives/couchbase"
	"github.com/couchbase/query/tenant"
)

// http implementation of SystemRemoteAccess
type systemRemoteHttp struct {
	state       clustering.Mode
	configStore clustering.ConfigurationStore

	// use getCommParams() and setCommParams() to get this pointer
	commParams unsafe.Pointer // *commParameters
}

type commParameters struct {
	client        *http.Client
	useSecurePort bool
}

// Returns nil if SetConnectionSecurityConfig has never been called.
func (this *systemRemoteHttp) getCommParams() *commParameters {
	curCommParameters := atomic.LoadPointer(&(this.commParams))

	return ((*commParameters)(curCommParameters))
}

func (this *systemRemoteHttp) setCommParams(cp *commParameters) {
	curCommParameters := unsafe.Pointer(cp)
	atomic.StorePointer(&(this.commParams), curCommParameters)
}

func (this *systemRemoteHttp) SetConnectionSecurityConfig(caFile string, certFile string, encryptNodeToNodeComms bool,
	clientCertAuthMandatory bool, internalClientCertFile string, internalClientKeyFile string,
	internalClientPrivateKeyPassphrase []byte) {

	var cp *commParameters
	if !encryptNodeToNodeComms {
		cp = &commParameters{
			client: &http.Client{
				Transport: &http.Transport{MaxIdleConnsPerHost: 10},
				Timeout:   5 * time.Second,
			},
			useSecurePort: false,
		}
	} else {
		if len(caFile) > 0 {
			certFile = caFile
		}
		serverCert, err := ioutil.ReadFile(certFile)
		if err != nil {
			logging.Errorf("SystemRemoteHttp.SetCommunictionSecurityConfig: Unable to read cert file %s:%v", certFile, err)
			return
		}
		caPool := x509.NewCertPool()
		caPool.AppendCertsFromPEM(serverCert)
		tlsConfig := &tls.Config{RootCAs: caPool}

		// MB-52102: Include the internal client cert if n2n encryption is enabled
		// and client certificate authentication is mandatory.
		if clientCertAuthMandatory {
			internalTlSCert, err := ntls.LoadX509KeyPair(internalClientCertFile, internalClientKeyFile,
				internalClientPrivateKeyPassphrase)
			if err != nil {
				logging.Errorf("SystemRemoteHttp.SetCommunictionSecurityConfig: internal client certificate refresh failed: %v",
					err)
				return
			}

			tlsConfig.Certificates = []tls.Certificate{internalTlSCert}
		}

		cp = &commParameters{
			client: &http.Client{
				Transport: &http.Transport{
					TLSClientConfig:     tlsConfig,
					MaxIdleConnsPerHost: 10,
				},
				Timeout: 5 * time.Second,
			},
			useSecurePort: true,
		}
	}
	existingCommParams := atomic.SwapPointer(&this.commParams, unsafe.Pointer(cp))
	var ecp *commParameters = ((*commParameters)(existingCommParams))
	transport, ok := ecp.client.Transport.(*http.Transport)
	if ok {
		transport.CloseIdleConnections()
	}
}

func NewSystemRemoteAccess(cfgStore clustering.ConfigurationStore) distributed.SystemRemoteAccess {
	return &systemRemoteHttp{
		configStore: cfgStore,
		commParams: unsafe.Pointer(&commParameters{
			client: &http.Client{
				Transport: &http.Transport{MaxIdleConnsPerHost: 10},
				Timeout:   5 * time.Second,
			},
			useSecurePort: false,
		}),
	}
}

// construct a key from node name and local key
func (this *systemRemoteHttp) MakeKey(node string, key string) string {
	if node == "" {
		return key
	} else {
		return "[" + node + "]" + key
	}
}

// split global key into name and local key
func (this *systemRemoteHttp) SplitKey(key string) (string, string) {
	bytes := []byte(key)
	l := len(bytes)
	o := 0

	// skip spaces
	for o < l && bytes[o] == ' ' {
		o++
	}

	// if no square brackets or a single character, no node name
	if o >= l-1 || bytes[o] != '[' {
		return "", key
	}

	o++
	start := o
	brackets := 1

	// two consecutive square brackets mean IPv6
	if bytes[o] == '[' {
		brackets++
	}

	// scan the string and look for the other side
	for o < l {
		if bytes[o] == ']' {
			brackets--

			// yay, found the node
			if brackets == 0 {

				// if there's characters after the last bracket, all good
				if o < l-1 {
					return string(bytes[start:o]), string(bytes[o+1 : l])
				} else {

					// node but no document key?
					break
				}
			}
		}
		o++
	}

	// couldn't make sense of anything
	return "", key
}

// get remote keys from the specified nodes for the specified endpoint
// Note - "nodes[]" must be a list of node names
func (this *systemRemoteHttp) GetRemoteKeys(nodes []string, endpoint string,
	keyFn func(id string) bool, warnFn func(warn errors.Error), creds distributed.Creds, authToken string) {
	var keys []string

	// now that the local node name can change, use a consistent one across the scan
	whoAmI := this.WhoAmI()

	// not part of a cluster, no keys can be gathered
	if len(whoAmI) == 0 {
		return
	}

	cp := this.getCommParams()
	// no nodes means all nodes
	if len(nodes) == 0 {

		clusters, err := this.configStore.ClusterNames()
		if err != nil {
			if warnFn != nil {
				warnFn(errors.NewSystemRemoteWarning(err, "scan", endpoint))
			}
			return
		}

		for c, _ := range clusters {
			cl, err := this.configStore.ClusterByName(clusters[c])
			if err != nil {
				if warnFn != nil {
					warnFn(errors.NewSystemRemoteWarning(err, "scan", endpoint))
				}
				continue
			}

			queryNodeNames, err := cl.QueryNodeNames()
			if err != nil {
				if warnFn != nil {
					warnFn(errors.NewSystemRemoteWarning(err, "scan", endpoint))
				}
				continue
			}

			for n, _ := range queryNodeNames {
				node := queryNodeNames[n]
				nodeName := tenant.EncodeNodeName(node)

				// skip ourselves, we will be processed locally
				if node == whoAmI {
					continue
				}
				queryNode, err := this.getQueryNode(node, "scan", endpoint)
				if err != nil {
					if warnFn != nil {
						warnFn(err)
					}
					continue
				}
				if !queryNode.Healthy() {
					if warnFn != nil {
						warnFn(errors.NewSystemRemoteNodeSkippedWarning(nodeName, "scan", endpoint))
					}
					continue
				}

				body, opErr := this.doRemoteOp(queryNode, "indexes/"+endpoint, "GET", "", "scan", creds, "", cp)
				if opErr != nil {
					if warnFn != nil {
						warnFn(opErr)
					}
					continue
				}

				jErr := json.Unmarshal(body, &keys)
				if jErr != nil {
					if warnFn != nil {
						warnFn(errors.NewSystemRemoteWarning(jErr, "scan", endpoint))
					}
					continue
				}

				if keyFn != nil {
					for _, key := range keys {
						if !keyFn("[" + nodeName + "]" + key) {
							return
						}
					}
				}
			}
		}
	} else {

		for _, node := range nodes {
			// skip ourselves, it will be processed locally
			if node == whoAmI {
				continue
			}
			nodeName := tenant.EncodeNodeName(node)
			queryNode, err := this.getQueryNode(node, "scan", endpoint)
			if err != nil {
				if warnFn != nil {
					warnFn(err)
				}
				continue
			}
			if !queryNode.Healthy() {
				if warnFn != nil {
					warnFn(errors.NewSystemRemoteNodeSkippedWarning(nodeName, "scan", endpoint))
				}
				continue
			}

			body, opErr := this.doRemoteOp(queryNode, "indexes/"+endpoint, "GET", "", "scan", creds, "", cp)
			if opErr != nil {
				if warnFn != nil {
					warnFn(opErr)
				}
				continue
			}
			jErr := json.Unmarshal(body, &keys)
			if jErr != nil {
				if warnFn != nil {
					warnFn(errors.NewSystemRemoteWarning(jErr, "scan", endpoint))
				}
				continue
			}
			if keyFn != nil {
				for _, key := range keys {
					if !keyFn("[" + nodeName + "]" + key) {
						return
					}
				}
			}
		}
	}
}

// get a specified remote document from a remote node
// Note - the remote "node" must be the node name
func (this *systemRemoteHttp) GetRemoteDoc(node string, key string, endpoint string, command string,
	docFn func(map[string]interface{}), warnFn func(warn errors.Error), creds distributed.Creds, authToken string,
	formData map[string]interface{}) {

	var loc string
	var body []byte
	var doc map[string]interface{}

	queryNode, err := this.getQueryNode(node, "fetch", endpoint)
	if err != nil {
		if warnFn != nil {
			warnFn(err)
		}
		return
	}

	cp := this.getCommParams()
	if key != "" {
		loc = endpoint + "/" + key
	} else {
		loc = endpoint
	}

	data := ""
	if len(formData) > 0 {
		form := url.Values{}
		for k, v := range formData {
			form.Set(k, fmt.Sprintf("%v", v))
		}
		switch command {
		case http.MethodPost:
			data = form.Encode()
		case http.MethodGet:
			loc += "?" + form.Encode()
		}
	}

	body, opErr := this.doRemoteOp(queryNode, loc, command, data, "fetch", creds, authToken, cp)
	if opErr != nil {
		if warnFn != nil {
			warnFn(opErr)
		}
		return
	}

	jErr := json.Unmarshal(body, &doc)
	if jErr != nil {
		if warnFn != nil {
			errors.NewSystemRemoteWarning(jErr, "fetch", endpoint)
		}
		return
	}

	if docFn != nil && doc != nil {
		docFn(doc)
	}
}

// perform operation on key on the specified nodes for the specified endpoint
func (this *systemRemoteHttp) DoRemoteOps(nodes []string, endpoint string, command string, key string, data string,
	warnFn func(warn errors.Error), creds distributed.Creds, authToken string) {

	// now that the local node name can change, use a consistent one across the scan
	whoAmI := this.WhoAmI()

	// not part of a cluster, no node to operate against
	if len(whoAmI) == 0 {
		return
	}

	if key != "" {
		endpoint = endpoint + "/" + key
	}

	cp := this.getCommParams()
	// no nodes means all nodes
	if len(nodes) == 0 {

		clusters, err := this.configStore.ClusterNames()
		if err != nil {
			if warnFn != nil {
				warnFn(errors.NewSystemRemoteWarning(err, "scan", endpoint))
			}
			return
		}

		for c, _ := range clusters {
			cl, err := this.configStore.ClusterByName(clusters[c])
			if err != nil {
				if warnFn != nil {
					warnFn(errors.NewSystemRemoteWarning(err, "scan", endpoint))
				}
				continue
			}
			queryNodeNames, err := cl.QueryNodeNames()
			if err != nil {
				if warnFn != nil {
					warnFn(errors.NewSystemRemoteWarning(err, "scan", endpoint))
				}
				continue
			}

			for n, _ := range queryNodeNames {
				node := queryNodeNames[n]

				// skip ourselves, we will be processed locally
				if node == whoAmI {
					continue
				}

				queryNode, err := this.getQueryNode(node, "scan", endpoint)
				if err != nil {
					if warnFn != nil {
						warnFn(err)
					}
					continue
				}
				_, opErr := this.doRemoteOp(queryNode, endpoint, command, data, command, creds, authToken, cp)
				if warnFn != nil && opErr != nil {
					warnFn(opErr)
				}

			}
		}
	} else {

		for _, node := range nodes {

			// skip ourselves, it will be processed locally
			if node == whoAmI {
				continue
			}

			queryNode, err := this.getQueryNode(node, "scan", endpoint)
			if err != nil {
				if warnFn != nil {
					warnFn(err)
				}
				continue
			}

			_, opErr := this.doRemoteOp(queryNode, endpoint, command, data, command, creds, authToken, cp)
			if warnFn != nil && opErr != nil {
				warnFn(opErr)
			}
		}
	}
}

// helper for the REST op
func (this *systemRemoteHttp) doRemoteOp(node clustering.QueryNode, endpoint string, command string, data string, op string,
	creds distributed.Creds, authToken string, cp *commParameters) ([]byte, errors.Error) {

	if node == nil {
		return nil, errors.NewSystemRemoteWarning(goErr.New("missing node"), op, endpoint)
	}
	fullEndpoint, u, p := this.getFullEndpoint(node, endpoint, creds, cp)

	return this.doRemoteEndpointOp(fullEndpoint, command, data, op, creds, authToken, cp, u, p)
}

func (this *systemRemoteHttp) getFullEndpoint(node clustering.QueryNode, endpoint string, creds distributed.Creds,
	cp *commParameters) (string, string, string) {

	var fullEndpoint string
	if cp.useSecurePort {
		fullEndpoint = node.ClusterSecure() + "/" + endpoint
	} else {
		fullEndpoint = node.ClusterEndpoint() + "/" + endpoint
	}
	if creds != "" {
		fullEndpoint += "?impersonate=" + string(creds)
	}

	// Here, I'm leveraging the fact that the node name is the host:port of the mgmt
	// endpoint associated with the node. This is the same hostport pair that allows us
	// to access the admin creds for that node.
	authenticator := cbauth.Default
	u, p, _ := authenticator.GetHTTPServiceAuth(node.Name())

	return fullEndpoint, u, p
}

func (this *systemRemoteHttp) doRemoteEndpointOp(fullEndpoint string, command string, data string, op string,
	creds distributed.Creds, authToken string, cp *commParameters, u string, p string) ([]byte, errors.Error) {

	var reader io.Reader
	if data != "" {
		reader = strings.NewReader(data)
	}

	request, _ := http.NewRequest(command, fullEndpoint, reader)
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(u, p)
	request.Header.Set("User-Agent", couchbase.USER_AGENT)

	resp, err := cp.client.Do(request)
	if err != nil {
		ep := ""
		u, e := url.Parse(fullEndpoint)
		if e == nil {
			ep = u.Path
		}
		return nil, errors.NewSystemRemoteWarning(err, op, ep)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		ep := ""
		u, e := url.Parse(fullEndpoint)
		if e == nil {
			ep = u.Path
		}
		return nil, errors.NewSystemRemoteWarning(err, op, ep)
	}

	// we got a response, but the operation failed: extract the error
	if resp.StatusCode != 200 {
		var opErr errors.Error

		err = json.Unmarshal(body, &opErr)
		if err != nil || len(body) == 0 || strings.TrimSpace(opErr.Error()) == "" {

			// MB-28264 we could not unmarshal an error from a remote node
			// just create an error from the body
			ep := ""
			u, e := url.Parse(fullEndpoint)
			if e == nil {
				ep = u.Path
			}
			errText := string(body)
			if strings.TrimSpace(errText) == "" {
				if resp.StatusCode == http.StatusForbidden {
					errText = "no permissions"
				} else if resp.StatusCode == http.StatusNotFound {
					errText = "object not found"
				} else {
					errText = "no data received"
				}
			}

			return nil, errors.NewSystemRemoteWarning(goErr.New(errText), op, ep)
		}
		return nil, opErr
	}

	return body, nil
}

// helper to map a node name to a node
func (this *systemRemoteHttp) getQueryNode(node string, op string, endpoint string) (clustering.QueryNode, errors.Error) {
	if this.configStore == nil {
		return nil, errors.NewSystemRemoteWarning(goErr.New("missing config store"), op, endpoint)
	}

	clusters, err := this.configStore.ClusterNames()
	if err != nil {
		return nil, err
	}

	for c, _ := range clusters {
		cl, err := this.configStore.ClusterByName(clusters[c])
		if err != nil {
			return nil, errors.NewSystemRemoteWarning(err, op, endpoint)
		}
		queryNode, err := cl.QueryNodeByName(node)
		if queryNode != nil && err == nil {
			return queryNode, nil
		}
	}
	return nil, errors.NewSystemRemoteWarning(errors.NewSystemRemoteNodeNotFoundWarning(node), op, endpoint)
}

// returns the local node identity, as known to the cluster
func (this *systemRemoteHttp) WhoAmI() string {

	// when clustered operations begin, we'll determine
	// if we are part of a cluster.
	// if yes, we'll refresh our node name from clustering
	// at every call, if not, we turn off clustering for good
	if this.state == "" {

		// not part of a cluster if there isn't a configStore
		if this.configStore == nil {
			this.state = clustering.STANDALONE
			return ""
		}

		state, err := this.configStore.State()
		if err != nil {
			this.state = clustering.STANDALONE
			return ""
		}
		this.state = state

		if this.state == clustering.STANDALONE {
			return ""
		}

		// not part of a cluster if we can't work out our own name
		localNode, err := this.configStore.WhoAmI()
		if err != nil {
			this.state = clustering.STANDALONE
			return ""
		}
		return localNode
	} else if this.state == clustering.STANDALONE {
		return ""
	}

	localNode, _ := this.configStore.WhoAmI()
	return localNode
}

func (this *systemRemoteHttp) NodeUUID(host string) string {
	cluster, err := this.configStore.Cluster()
	if err != nil {
		return ""
	}
	uuid, _ := cluster.NodeUUID(host)
	return uuid
}

func (this *systemRemoteHttp) UUIDToHost(uuid string) string {
	cluster, err := this.configStore.Cluster()
	if err != nil {
		return ""
	}
	host, _ := cluster.UUIDToHost(uuid)
	return host
}

func (this *systemRemoteHttp) Starting() bool {
	this.doState()
	return this.state == clustering.STARTING
}

func (this *systemRemoteHttp) Clustered() bool {
	this.doState()
	return this.state == clustering.CLUSTERED
}

func (this *systemRemoteHttp) StandAlone() bool {
	this.doState()
	return this.state == clustering.STANDALONE
}

func (this *systemRemoteHttp) doState() {

	// not part of a cluster if there isn't a configStore
	if this.configStore == nil {
		this.state = clustering.STANDALONE
		return
	}
	if this.state == clustering.STANDALONE {
		return
	}
	state, err := this.configStore.State()
	if err != nil {
		this.state = clustering.STANDALONE
		return
	}
	this.state = state
}

func (this *systemRemoteHttp) GetNodeNames() []string {
	var names []string

	if len(this.WhoAmI()) == 0 {
		return names
	}
	clusters, err := this.configStore.ClusterNames()
	if err != nil {
		return names
	}

	for c, _ := range clusters {
		cl, err := this.configStore.ClusterByName(clusters[c])
		if err != nil {
			continue
		}
		queryNodeNames, err := cl.QueryNodeNames()
		if err != nil {
			continue
		}

		for n, _ := range queryNodeNames {
			names = append(names, queryNodeNames[n])
		}
	}
	return names
}

var capabilities = map[distributed.Capability]string{
	distributed.NEW_PREPAREDS:            "enhancedPreparedStatements",
	distributed.NEW_OPTIMIZER:            "costBasedOptimizer",
	distributed.NEW_INDEXADVISOR:         "indexAdvisor",
	distributed.NEW_INLINE_FUNCTIONS:     "inlineFunctions",
	distributed.NEW_JAVASCRIPT_FUNCTIONS: "javaScriptFunctions",
	distributed.KV_RANGE_SCANS:           "kv.rangeScans",
	distributed.READ_FROM_REPLICA:        "readFromReplica",
}

// a capability is enabled if we are part of a cluster and we find it enabled
// on each cluster that's reachable
func (this *systemRemoteHttp) Enabled(capability distributed.Capability) bool {

	// if we are running standalone, enable all features, as we don't have to contend with
	// any other node, and we don't have a cluster manager to ask anyway
	if this.state == clustering.STANDALONE ||

		// work out state id WhoAmI() had never been called
		(this.state == "" && this.WhoAmI() == "") {
		return true
	}
	clusters, err := this.configStore.ClusterNames()
	if err != nil {
		return false
	}

	for c, _ := range clusters {
		cl, err := this.configStore.ClusterByName(clusters[c])
		if err != nil {
			continue
		}
		if !cl.Capability(capabilities[capability]) {
			return false
		}
	}
	return true
}

// dynamically change settings
func (this *systemRemoteHttp) Settings(settings map[string]interface{}) errors.Error {
	return settingsWorkHorse(settings, _ENDPOINT.server)
}
