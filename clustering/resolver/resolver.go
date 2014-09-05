//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package resolver

import (
	"fmt"
	"net"
	"strings"

	"github.com/couchbaselabs/query/accounting"
	"github.com/couchbaselabs/query/clustering"
	"github.com/couchbaselabs/query/clustering/stub"
	"github.com/couchbaselabs/query/clustering/zookeeper"
	"github.com/couchbaselabs/query/datastore"

	"github.com/couchbaselabs/query/errors"
)

func NewConfigstore(uri string) (clustering.ConfigurationStore, errors.Error) {
	if strings.HasPrefix(uri, "zookeeper:") {
		return clustering_zk.NewConfigstore(uri)
	}

	if strings.HasPrefix(uri, "stub:") {
		return clustering_stub.NewConfigurationStore()
	}

	return nil, errors.NewError(nil, fmt.Sprintf("Invalid configstore uri: %s", uri))
}

func NewClusterConfig(uri string, cluster_name string, configstore clustering.ConfigurationStore, datastore datastore.Datastore, acctstore accounting.AccountingStore) (clustering.Cluster, errors.Error) {
	if strings.HasPrefix(uri, "zookeeper:") {
		return clustering_zk.NewCluster(cluster_name, configstore, datastore, acctstore), nil
	}

	if strings.HasPrefix(uri, "stub:") {
		return clustering_stub.ClusterStub{}, nil
	}

	return nil, errors.NewError(nil, fmt.Sprintf("Invalid configstore uri: %s", uri))
}

func NewQueryNodeConfig(uri string, cluster_name string, version string, query_addr string, admin_addr string, configstore clustering.ConfigurationStore, datastore datastore.Datastore, acctstore accounting.AccountingStore) (clustering.QueryNode, errors.Error) {
	ip_addr, err := externalIP()
	if err != nil {
		ip_addr = "127.0.0.1"
	}

	if strings.HasPrefix(uri, "zookeeper:") {
		queryName := ip_addr + query_addr                 // Construct query node name from ip addr and http_addr. Assumption that this will be unique
		queryEndpoint := "http://" + queryName            // TODO : protocol specification: how do we know it will be http?
		adminEndpoint := "http://" + ip_addr + admin_addr // TODO: protocol specification: how to specify http cleanly?
		return clustering_zk.NewQueryNode(cluster_name, queryName, version, queryEndpoint, adminEndpoint, configstore, datastore, acctstore), nil
	}

	if strings.HasPrefix(uri, "stub:") {
		return clustering_stub.QueryNodeStub{}, nil
	}

	return nil, errors.NewError(nil, fmt.Sprintf("Invalid configstore uri: %s", uri))
}

func externalIP() (string, errors.Error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", errors.NewError(err, "")
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", errors.NewError(err, "")
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.NewError(nil, "Not connected to the network")
}
