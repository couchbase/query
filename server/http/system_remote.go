//  Copyright (c) 2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

// This implements remote system keyspace access for the REST based http package

package http

import (
	"encoding/json"
	goErr "errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/couchbase/query/clustering"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/util"
)

// http implementation of SystemRemoteAccess
type systemRemoteHttp struct {
	localNode   string
	configStore clustering.ConfigurationStore
}

// flags that we've evaluated WhoAmI, and couldn't establish it
const _UNSET = "_"

func NewSystemRemoteAccess(cfgStore clustering.ConfigurationStore) distributed.SystemRemoteAccess {
	return &systemRemoteHttp{
		localNode:   "",
		configStore: cfgStore,
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
	if strings.HasPrefix(key, "[") {
		fields := strings.FieldsFunc(key, func(c rune) bool {
			return c == '[' || c == ']'
		})
		if len(fields) == 2 {
			return fields[0], fields[1]
		}
	}
	return "", key
}

// get remote keys from the specified nodes for the specified endpoint
func (this *systemRemoteHttp) GetRemoteKeys(nodes []string, endpoint string,
	keyFn func(id string), warnFn func(warn errors.Error)) {
	var keys []string

	// not part of a cluster, no keys can be gathered
	if len(this.WhoAmI()) == 0 {
		return
	}

	// no nodes means all nodes
	if len(nodes) == 0 {

		cm := this.configStore.ConfigurationManager()
		clusters, err := cm.GetClusters()
		if err != nil {
			if warnFn != nil {
				warnFn(errors.NewSystemRemoteWarning(err, "scan", endpoint))
			}
			return
		}

		for _, c := range clusters {
			clm := c.ClusterManager()
			queryNodes, err := clm.GetQueryNodes()
			if err != nil {
				if warnFn != nil {
					warnFn(errors.NewSystemRemoteWarning(err, "scan", endpoint))
				}
				continue
			}

			for _, queryNode := range queryNodes {
				node := queryNode.Name()

				// skip ourselves, we will be processed locally
				if node == this.WhoAmI() {
					continue
				}

				body, opErr := doRemoteOp(queryNode, "indexes/"+endpoint, "GET")
				if opErr != nil {
					if warnFn != nil {
						warnFn(errors.NewSystemRemoteWarning(opErr, "scan", endpoint))
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
						keyFn("[" + node + "]" + key)
					}
				}
			}
		}
	} else {

		for _, node := range nodes {

			// skip ourselves, it will be processed locally
			if node == this.WhoAmI() {
				continue
			}

			queryNode, err := getQueryNode(this.configStore, node, "scan", endpoint)
			if err != nil {
				if warnFn != nil {
					warnFn(err)
				}
				continue
			}

			body, opErr := doRemoteOp(queryNode, "indexes/"+endpoint, "GET")
			if opErr != nil {
				if warnFn != nil {
					warnFn(errors.NewSystemRemoteWarning(opErr, "scan", endpoint))
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
					keyFn("[" + node + "]" + key)
				}
			}
		}
	}
}

// get a specified remote document from a remote node
func (this *systemRemoteHttp) GetRemoteDoc(node string, key string, endpoint string, command string,
	docFn func(map[string]interface{}), warnFn func(warn errors.Error)) {
	var body []byte
	var doc map[string]interface{}

	queryNode, err := getQueryNode(this.configStore, node, "fetch", endpoint)
	if err != nil {
		if warnFn != nil {
			warnFn(err)
		}
		return
	}

	body, opErr := doRemoteOp(queryNode, endpoint+"/"+key, command)
	if opErr != nil {
		if warnFn != nil {
			warnFn(errors.NewSystemRemoteWarning(opErr, "fetch", endpoint))
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

	if docFn != nil {
		docFn(doc)
	}
}

// helper for the REST op
func doRemoteOp(node clustering.QueryNode, endpoint string, command string) ([]byte, error) {
	var HTTPTransport = &http.Transport{MaxIdleConnsPerHost: 10} //MaxIdleConnsPerHost}
	var HTTPClient = &http.Client{Transport: HTTPTransport}

	if node == nil {
		return nil, goErr.New("missing remote node")
	}

	fullEndpoint := node.ClusterEndpoint() + "/" + endpoint
	request, _ := http.NewRequest(command, fullEndpoint, nil)
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		_, _ = ioutil.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, err
	}

	return ioutil.ReadAll(resp.Body)
}

// helper to map a node name to a node
func getQueryNode(configStore clustering.ConfigurationStore, node string, op string, endpoint string) (clustering.QueryNode, errors.Error) {
	if configStore == nil {
		return nil, errors.NewSystemRemoteWarning(goErr.New("missing config store"), op, endpoint)
	}

	cm := configStore.ConfigurationManager()
	clusters, err := cm.GetClusters()
	if err != nil {
		return nil, err
	}

	for _, c := range clusters {
		clm := c.ClusterManager()
		queryNodes, err := clm.GetQueryNodes()
		if err != nil {
			continue
		}

		for _, queryNode := range queryNodes {
			if queryNode.Name() == node {
				return queryNode, nil
			}
		}
	}
	return nil, errors.NewSystemRemoteWarning(fmt.Errorf("node %v not found", node), op, endpoint)
}

// returns the local node identity, as known to the cluster
func (this *systemRemoteHttp) WhoAmI() string {

	// There is a reason why we defer determining our own node name
	// to when we actually need it: when the configStore is created
	// at start up time, it may be empty, and we need to give the
	// cluster manager time to populate it, or we will think we are
	// not part of a cluster!

	// This probably ought to be protected by a latch,
	// however, should two requests work it out in parallel, this
	// will just result in temporarily wasted memory.
	if len(this.localNode) == 0 {

		// not part of a cluster if there isn't a configStore
		if this.configStore == nil {
			this.localNode = _UNSET
			return ""
		}

		// first search by IP
		localIp, _ := util.ExternalIP()
		if len(localIp) != 0 {
			if searchName(this.configStore, localIp) {
				this.localNode = localIp
				return localIp
			}

			// reverse lookup and search by name
			localNames, _ := net.LookupAddr(localIp)
			for _, localName := range localNames {
				if searchName(this.configStore, localName) {
					this.localNode = localName
					return localName
				}
			}
		}

		// all else failing, search by hostname
		localName, _ := os.Hostname()
		if searchName(this.configStore, localName) {
			this.localNode = localName
			return localName
		}

		// This is consistent with the /admin/config endpoint:
		// even if we did work out a likely node name, we are not
		// part of a cluster if we don't find ourselves in it.
		this.localNode = _UNSET
		return ""
	} else if this.localNode == _UNSET {
		return ""
	}
	return this.localNode
}

// helper that checks if a given name is a known cluster node
func searchName(configStore clustering.ConfigurationStore, name string) bool {
	cm := configStore.ConfigurationManager()
	clusters, err := cm.GetClusters()
	if err != nil {
		return false
	}

	for _, c := range clusters {
		clm := c.ClusterManager()
		queryNodes, err := clm.GetQueryNodes()
		if err != nil {
			return false
		}

		for _, queryNode := range queryNodes {
			if queryNode.Name() == name {
				return true
			}
		}
	}
	return false
}

func (this *systemRemoteHttp) GetNodeNames() []string {
	var names []string

	if len(this.WhoAmI()) == 0 {
		return names
	}
	cm := this.configStore.ConfigurationManager()
	clusters, err := cm.GetClusters()
	if err != nil {
		return names
	}

	for _, c := range clusters {
		clm := c.ClusterManager()
		queryNodes, err := clm.GetQueryNodes()
		if err != nil {
			continue
		}

		for _, queryNode := range queryNodes {
			names = append(names, queryNode.Name())
		}
	}
	return names
}
