//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package clustering_cb

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/clustering"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/primitives/couchbase"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/server/http"
	"github.com/couchbase/query/util"
)

const _PREFIX = "couchbase:"

const _GRACE_PERIOD = time.Second

const _POLL_INTERVAL = 10 * time.Second

///////// Notes about Couchbase implementation of Clustering API
//
// clustering_cb (this package) -> primitives/couchbase -> couchbase cluster
//
// pool is a synonym for cluster
//

// cbConfigStore implements clustering.ConfigurationStore
type cbConfigStore struct {
	sync.RWMutex
	adminUrl     string
	ourPorts     map[string]int
	maybeManaged bool
	noMoreChecks bool
	poolName     string
	poolSrvRev   int
	whoAmI       string
	state        clustering.Mode
	cbConn       *couchbase.Client
	clusterIds   []string
	clusters     map[string]*cbCluster
	uuid         string
}

// create a cbConfigStore given the path to a couchbase instance
func NewConfigstore(path string, uuid string) (clustering.ConfigurationStore, errors.Error) {
	if strings.HasPrefix(path, _PREFIX) {
		path = path[len(_PREFIX):]
	}
	c, err := couchbase.ConnectWithAuth(path, cbauth.NewAuthHandler(nil), couchbase.USER_AGENT)
	if err != nil {
		return nil, errors.NewAdminConnectionError(err, path)
	}
	rv := &cbConfigStore{
		adminUrl:     path,
		ourPorts:     map[string]int{},
		noMoreChecks: false,
		poolSrvRev:   -999,
		cbConn:       &c,
		clusters:     make(map[string]*cbCluster, 1),
		uuid:         uuid,
	}

	// pool names are set once and for all on connection,
	// so we might just allocate them straight away
	for _, pool := range rv.getPools() {
		rv.clusterIds = append(rv.clusterIds, pool.Name)
	}
	return rv, nil
}

// Implement Stringer interface
func (this *cbConfigStore) String() string {
	return fmt.Sprintf("url=%v", this.adminUrl)
}

// Implement clustering.ConfigurationStore interface
func (this *cbConfigStore) Id() string {
	return this.URL()
}

func (this *cbConfigStore) URL() string {
	return this.adminUrl
}

func (this *cbConfigStore) SetOptions(httpAddr, httpsAddr string,
	maybeManaged bool) errors.Error {
	if httpAddr != "" {
		_, port := server.HostNameandPort(httpAddr)
		if port != "" {
			portNum, err := strconv.Atoi(port)
			if err == nil && portNum > 0 {
				this.ourPorts[_HTTP] = portNum
			} else {
				return errors.NewAdminBadServicePort(port)
			}
		} else {
			return errors.NewAdminBadServicePort("<no port>")
		}
	}
	if httpsAddr != "" {
		_, port := server.HostNameandPort(httpsAddr)
		if port != "" {
			portNum, err := strconv.Atoi(port)
			if err == nil && portNum > 0 {
				this.ourPorts[_HTTPS] = portNum
			} else {
				return errors.NewAdminBadServicePort(port)
			}
		} else {
			return errors.NewAdminBadServicePort("<no port>")
		}
	}
	this.maybeManaged = maybeManaged
	pollStdin.Do(func() { go doPollStdin() })
	return nil
}

func (this *cbConfigStore) ClusterNames() ([]string, errors.Error) {
	return this.clusterIds, nil
}

func (this *cbConfigStore) ClusterByName(name string) (clustering.Cluster, errors.Error) {
	this.RLock()
	c, found := this.clusters[name]
	this.RUnlock()
	if found {
		return c, nil
	}
	for p, _ := range this.clusterIds {
		found = this.clusterIds[p] == name
		if found {
			break
		}
	}
	if !found {
		return nil, errors.NewAdminGetClusterError(fmt.Errorf("No such pool"), name)
	}
	c = &cbCluster{
		configStore:    this,
		ClusterName:    name,
		ConfigstoreURI: this.URL(),
		DatastoreURI:   this.URL(),
	}
	this.Lock()
	this.clusters[name] = c
	this.Unlock()
	return c, nil
}

func (this *cbConfigStore) ConfigurationManager() clustering.ConfigurationManager {
	return this
}

// Helper method to retrieve all pools
func (this *cbConfigStore) getPools() []couchbase.RestPool {
	return this.cbConn.Info.Pools
}

// Helper method to retrieve Couchbase services data (/pools/default/nodeServices)
// and Couchbase pool (cluster) data (/pools/default)
//
func (this *cbConfigStore) getPoolServices(name string) (*couchbase.Pool, *couchbase.PoolServices, errors.Error) {
	nodeServices, err := this.cbConn.GetPoolServices(name)
	if err != nil {
		return nil, nil, errors.NewAdminGetClusterError(err, name)
	}

	pool, err := this.cbConn.GetPool(name)
	if err != nil {
		return nil, nil, errors.NewAdminGetClusterError(err, name)
	}

	return &pool, &nodeServices, nil
}

// cbConfigStore also implements clustering.ConfigurationManager interface
func (this *cbConfigStore) ConfigurationStore() clustering.ConfigurationStore {
	return this
}

func (this *cbConfigStore) AddCluster(l clustering.Cluster) (clustering.Cluster, errors.Error) {
	// NOP. This is a read-only implementation
	return nil, nil
}

func (this *cbConfigStore) RemoveCluster(l clustering.Cluster) (bool, errors.Error) {
	// NOP. This is a read-only implementation
	return false, nil
}

func (this *cbConfigStore) RemoveClusterByName(name string) (bool, errors.Error) {
	// NOP. This is a read-only implementation
	return false, nil
}

func (this *cbConfigStore) GetClusters() ([]clustering.Cluster, errors.Error) {
	clusters := []clustering.Cluster{}
	clusterNames, _ := this.ClusterNames()
	for _, name := range clusterNames {
		cluster, err := this.ClusterByName(name)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, cluster)
	}
	return clusters, nil
}

func (this *cbConfigStore) Authorize(credentials map[string]string, privileges []clustering.Privilege) errors.Error {
	if len(credentials) == 0 {
		return errors.NewAdminAuthError(nil, "no credentials provided")
	}

	for username, password := range credentials {
		auth, err := cbauth.Auth(username, password)
		if err != nil {
			return errors.NewAdminAuthError(err, "unable to authenticate with given credential")
		}
		for _, requested := range privileges {
			switch requested {
			case clustering.PRIV_SYS_ADMIN:
				isAdmin, err := auth.IsAllowed("cluster.settings!write")
				if err != nil {
					return errors.NewAdminAuthError(err, "")
				}
				if isAdmin {
					return nil
				}
				return errors.NewAdminAuthError(nil, "sys admin requires administrator credentials")
			case clustering.PRIV_READ:
				isPermitted, err := auth.IsAllowed("cluster.settings!read")
				if err != nil {
					return errors.NewAdminAuthError(err, "")
				}
				if isPermitted {
					return nil
				}
				return errors.NewAdminAuthError(nil, "read not authorized")
			default:
				return errors.NewAdminAuthError(nil, fmt.Sprintf("unexpected authorization %v", requested))
			}
		}
	}
	return errors.NewAdminAuthError(nil, "unrecognized authorization request")
}

const n1qlService = "n1ql"

func (this *cbConfigStore) WhoAmI() (string, errors.Error) {
	name, state, err := this.doNameState()
	if err != nil {
		return "", err
	}
	if state != clustering.STANDALONE {
		return name, nil
	}
	return "", nil
}

func (this *cbConfigStore) State() (clustering.Mode, errors.Error) {
	_, state, err := this.doNameState()
	if err != nil {
		return "", err
	}
	return state, nil
}

func (this *cbConfigStore) Cluster() (clustering.Cluster, errors.Error) {
	_, _, err := this.doNameState()
	if err != nil {
		return nil, err
	}
	return this.ClusterByName(this.poolName)
}

func (this *cbConfigStore) doNameState() (string, clustering.Mode, errors.Error) {

	// once things get to a certain state, no changes are possible
	// hence we can skip the tests and we don't even require a lock
	if this.noMoreChecks {
		return this.whoAmI, this.state, nil
	}

	// this will exhaust all possibilities in the hope of
	// finding a good name and only return an error
	// if we could not find a name at all
	var err errors.Error

	this.RLock()

	// have we been here before?
	if this.poolName != "" {
		pool, poolServices, newErr := this.getPoolServices(this.poolName)
		if newErr != nil {
			err = errors.NewAdminConnectionError(newErr, this.poolName)
		} else {
			defer pool.Close()

			// If pool services rev matches the cluster's rev, nothing has changed
			if poolServices.Rev == this.poolSrvRev {
				defer this.RUnlock()
				return this.whoAmI, this.state, nil
			}

			// check that things are still valid
			whoAmI, state, newErr := this.checkPoolServices(pool, poolServices)
			if newErr == nil && state != "" {

				// promote the lock for the update
				// (this may be wasteful, but we have no other choice)
				this.RUnlock()
				this.Lock()
				defer this.Unlock()

				// we got here first, update
				if poolServices.Rev > this.poolSrvRev {
					this.whoAmI = whoAmI
					this.state = state

					hostName, _ := server.HostNameandPort(whoAmI)

					// no more changes will happen if we are clustered and have a FQDN
					// (name will not fall back to 127.0.0.1), or we are standalone
					this.noMoreChecks = (state == clustering.STANDALONE ||
						(state == clustering.CLUSTERED && hostName != server.GetIP(false)))

				}
				return this.whoAmI, this.state, nil
			}
		}
	}

	// Either things went badly wrong, or we are here for the first time
	// We have to work out things from first principles, which requires
	// promoting the lock
	// Somebody might have done the work while we were waiting, if so
	// we will take advantage of that
	this.RUnlock()
	this.Lock()
	defer this.Unlock()

	if this.noMoreChecks {
		return this.whoAmI, this.state, nil
	}

	if this.poolName != "" {
		pool, poolServices, newErr := this.getPoolServices(this.poolName)
		if newErr != nil {
			err = errors.NewAdminConnectionError(newErr, this.poolName)
		} else {
			defer pool.Close()

			if poolServices.Rev == this.poolSrvRev {
				return this.whoAmI, this.state, nil
			}

			whoAmI, state, newErr := this.checkPoolServices(pool, poolServices)
			if newErr == nil && state != "" {
				this.whoAmI = whoAmI
				this.state = state

				hostName, _ := server.HostNameandPort(whoAmI)

				this.noMoreChecks = (state == clustering.STANDALONE ||
					(state == clustering.CLUSTERED && hostName != server.GetIP(false)))
				return this.whoAmI, this.state, nil
			} else if err == nil {
				err = newErr
			}
		}
	}

	// nope - start from scratch
	this.whoAmI = ""
	this.state = ""
	this.poolName = ""
	this.noMoreChecks = false

	// same process, but scan all pools now
	for _, p := range this.getPools() {
		pool, poolServices, newErr := this.getPoolServices(p.Name)
		if newErr != nil {
			logging.Tracef("%p.getPoolServices(%v) failed: %v", this, p.Name, newErr)
			if err != nil {
				err = newErr
			}
			if pool != nil {
				pool.Close()
			}
			continue
		}
		whoAmI, state, newErr := this.checkPoolServices(pool, poolServices)
		pool.Close()
		if newErr != nil {
			logging.Tracef("%p.checkPoolServices() (pool name: %v) failed: %v", this, p.Name, newErr)
			if err == nil {
				err = newErr
			}
			continue
		}

		// not in this pool
		if state == "" {
			continue
		}

		this.poolName = p.Name
		this.whoAmI = whoAmI
		this.state = state

		hostName, _ := server.HostNameandPort(whoAmI)

		this.noMoreChecks = (state == clustering.STANDALONE ||
			(state == clustering.CLUSTERED && hostName != server.GetIP(false)))
		return this.whoAmI, this.state, nil
	}

	// We haven't found ourselves in there.
	// It could be we are not part of a cluster.
	// It could be ns_server is not yet listing us
	// Either way, we can't cache anything.
	return "", clustering.STARTING, err
}

func (this *cbConfigStore) checkPoolServices(pool *couchbase.Pool, poolServices *couchbase.PoolServices) (string, clustering.Mode, errors.Error) {
	if poolServices == nil {
		return "", "", errors.NewAdminConnectionError(nil, this.poolName)
	}
	for _, node := range poolServices.NodesExt {

		// the assumption is that a n1ql node is started by the local mgmt service
		// so we only have to have a look at ThisNode
		if !node.ThisNode {
			continue
		}

		// In the node services endpoint, nodes will either have a fully-qualified
		// domain name or the hostname will not be provided indicating that the
		// hostname is 127.0.0.1
		hostname := node.Hostname
		if hostname == "" {
			// For constructing URLs with raw IPv6 addresses- the IPv6 address
			// must be enclosed within ‘[‘ and ‘]’ brackets.
			hostname = server.GetIP(true)
		}
		ip := net.ParseIP(hostname)
		if ip != nil && ip.To4() == nil && ip.To16() != nil { // IPv6
			hostname = "[" + hostname + "]"
		}

		mgmtPort := node.Services["mgmt"]
		if mgmtPort == 0 {

			// shouldn't happen, there should always be a mgmt port on each node
			// we should return an error
			msg := fmt.Sprintf("NodeServices does not report mgmt endpoint for "+
				"this node: %v", node)
			return "", "", errors.NewAdminGetNodeError(nil, msg)
		}

		found := 0
		// now that we have identified the node, is n1ql actually running?
		for serv, proto := range n1qlProtocols {
			port, ok := node.Services[serv]
			ourPort, ook := this.ourPorts[proto]

			// ports matching, good
			// port not listed, skip
			// we are not listening or ports mismatching, standalone
			if ok {
				if ook && port == ourPort {
					found++
				} else if len(this.uuid) == 0 {
					return "", clustering.STANDALONE, nil
				}
			}
		}

		// We don't assume that there is precisely one query node per host.
		// Query nodes are unique per mgmt endpoint, so we add the mgmt
		// port to the whoami string to uniquely identify the query node.
		whoAmI := hostname + ":" + strconv.Itoa(mgmtPort)
		if found != 0 {
			return whoAmI, clustering.CLUSTERED, nil
		} else {

			// we found no n1ql service port - is n1ql provisioned in this node?
			for _, node := range pool.Nodes {
				if !node.ThisNode || node.NodeUUID != this.uuid {
					continue
				}
				for _, s := range node.Services {

					// yes, but clearly, not yet advertised
					// place ourselves in a holding pattern
					if s == n1qlService {

						// if we had been signalled that the couchbase orchestrator may
						// have started us, the fact that we find a n1ql service on our
						// node means that's us, but we haven't yet been reballanced in
						if this.maybeManaged {
							return whoAmI, clustering.STARTING, nil
						} else {
							return "", clustering.STARTING, nil
						}
					}
				}
			}
		}
	}
	return "", "", nil
}

// Type services associates a protocol with a port number
type services map[string]int

const (
	_HTTP  = "http"
	_HTTPS = "https"
)

// n1qlProtocols associates Couchbase query service names with a protocol
var n1qlProtocols = map[string]string{
	"n1ql":    _HTTP,
	"n1qlSSL": _HTTPS,
}

// cbCluster implements clustering.Cluster
type cbCluster struct {
	sync.RWMutex
	configStore       clustering.ConfigurationStore `json:"-"`
	dataStore         datastore.Datastore           `json:"-"`
	acctStore         accounting.AccountingStore    `json:"-"`
	ClusterName       string                        `json:"name"`
	DatastoreURI      string                        `json:"datastore"`
	ConfigstoreURI    string                        `json:"configstore"`
	AccountingURI     string                        `json:"accountstore"`
	version           clustering.Version            `json:"-"`
	VersionString     string                        `json:"version"`
	queryNodeNames    []string                      `json:"-"`
	queryNodeServices map[string]services           `json:"-"`
	queryNodes        map[string]*cbQueryNodeConfig `json:"-"`
	queryNodeUUIDs    map[string]string             `json:"-"`
	queryNodeHealthy  map[string]bool               `json:"-"`
	capabilities      map[string]bool               `json:"-"`
	poolSrvRev        int                           `json:"-"`
	lastCheck         util.Time                     `json:"-"`
}

// Create a new cbCluster instance
func NewCluster(name string,
	version clustering.Version,
	configstore clustering.ConfigurationStore,
	datastore datastore.Datastore,
	acctstore accounting.AccountingStore) (clustering.Cluster, errors.Error) {
	c := makeCbCluster(name, version, configstore, datastore, acctstore)
	return c, nil
}

func makeCbCluster(name string,
	version clustering.Version,
	cs clustering.ConfigurationStore,
	ds datastore.Datastore,
	as accounting.AccountingStore) clustering.Cluster {
	cluster := cbCluster{
		configStore:    cs,
		dataStore:      ds,
		acctStore:      as,
		ClusterName:    name,
		DatastoreURI:   ds.URL(),
		ConfigstoreURI: cs.URL(),
		AccountingURI:  as.URL(),
		version:        version,
		VersionString:  version.String(),
		queryNodeNames: []string{},
		poolSrvRev:     -999,
	}
	return &cluster
}

// cbCluster implements Stringer interface
func (this *cbCluster) String() string {
	return getJsonString(this)
}

// cbCluster implements clustering.Cluster interface
func (this *cbCluster) ConfigurationStoreId() string {
	return this.configStore.Id()
}

func (this *cbCluster) Name() string {
	return this.ClusterName
}

func (this *cbCluster) QueryNodeNames() ([]string, errors.Error) {

	// Get a handle of the couchbase connection:
	configStore, ok := this.configStore.(*cbConfigStore)
	if !ok {
		return nil, errors.NewAdminConnectionError(nil, this.ConfigurationStoreId())
	}

	if util.Since(this.lastCheck) <= _GRACE_PERIOD {
		return this.queryNodeNames, nil
	}

	poolServices, err := configStore.cbConn.GetPoolServices(this.ClusterName)
	if err != nil {
		return nil, errors.NewAdminConnectionError(err, this.ConfigurationStoreId())
	}

	if poolServices.Rev == this.poolSrvRev {
		this.Lock()
		this.lastCheck = util.Now()
		this.Unlock()
		return this.queryNodeNames, nil
	}

	// If pool services and cluster rev do not match, update the cluster's rev and query node data:
	queryNodeNames := []string{}
	queryNodeServices := map[string]services{}
	queryNodeUUIDs := make(map[string]string)
	queryNodeHealthy := make(map[string]bool)
	pool, err := configStore.cbConn.GetPool(configStore.poolName)
	if err != nil {
		return nil, errors.NewAdminGetNodeError(err, configStore.poolName)
	}
	for _, nodeServices := range poolServices.NodesExt {
		var queryServices services
		for name, protocol := range n1qlProtocols {
			if nodeServices.Services[name] != 0 {
				if queryServices == nil {
					queryServices = services{}
				}
				queryServices[protocol] = nodeServices.Services[name]
			}
		}

		if len(queryServices) == 0 { // no n1ql service at this node
			continue
		}

		hostname := nodeServices.Hostname

		// nodeServices.Hostname is either a fully-qualified domain name or
		// the empty string - which indicates 127.0.0.1
		// For constructing URLs with raw IPv6 addresses- the IPv6 address
		// must be enclosed within ‘[‘ and ‘]’ brackets.
		if hostname == "" {
			hostname = server.GetIP(true)
		} else {
			hostname, _ = server.HostNameandPort(hostname)
		}
		ip := net.ParseIP(hostname)
		if ip != nil && ip.To4() == nil && ip.To16() != nil { // IPv6
			hostname = "[" + hostname + "]"
		}

		mgmtPort := nodeServices.Services["mgmt"]
		if mgmtPort == 0 {

			// shouldn't happen; all nodes should have a mgmt port
			// should probably log a warning and this node gets ignored
			// TODO: log warning (when signature has warnings)
			continue
		}

		// Query nodes are unique per mgmt endpoint - which means they are unique
		// per Couchbase Server node instance - so we give query nodes an ID
		// which reflects that. Note that in particular query nodes are not
		// guaranteed to be unique per host.
		nodeId := hostname + ":" + strconv.Itoa(mgmtPort)

		queryNodeNames = append(queryNodeNames, nodeId)
		queryNodeServices[nodeId] = queryServices

		// Get NodeUUID
		for _, n := range pool.Nodes {
			if n.Hostname == nodeId {
				queryNodeUUIDs[nodeId] = n.NodeUUID

				// just in case we are querying an old node that doesn't report the status
				queryNodeHealthy[nodeId] = n.Status != "unhealthy"
				break
			}
		}
	}

	var capabilities map[string]bool
	if len(poolServices.Capabilities) > 0 {
		var caps map[string]interface{}

		err := json.Unmarshal(poolServices.Capabilities, &caps)
		if err == nil {
			n1ql := caps[n1qlService]
			if n1ql != nil {
				capList, ok := n1ql.([]interface{})
				if ok {
					capabilities = make(map[string]bool, len(capList))
					for i, _ := range capList {
						name, ok := capList[i].(string)
						if ok {
							capabilities[name] = true
						}
					}
				}
			}
		}
	}

	this.Lock()
	defer this.Unlock()
	this.queryNodeNames = queryNodeNames
	this.queryNodeUUIDs = queryNodeUUIDs
	this.queryNodeHealthy = queryNodeHealthy
	this.queryNodeServices = queryNodeServices
	this.queryNodes = make(map[string]*cbQueryNodeConfig)
	this.capabilities = capabilities

	this.poolSrvRev = poolServices.Rev
	this.lastCheck = util.Now()
	return this.queryNodeNames, nil
}

func (this *cbCluster) QueryNodeByName(name string) (clustering.QueryNode, errors.Error) {
	qryNodeNames, err := this.QueryNodeNames()
	if err != nil {
		return nil, err
	}

	this.RLock()
	rv, ok := this.queryNodes[name]
	this.RUnlock()
	if ok {
		return rv, nil
	}

	qryNodeName := ""
	for _, q := range qryNodeNames {
		if name == q {
			qryNodeName = q
			break
		}
	}
	if qryNodeName == "" {
		return nil, errors.NewAdminNoNodeError(name)
	}

	uuid, _ := this.queryNodeUUIDs[name]
	healthy, ok := this.queryNodeHealthy[name]

	qryNode := &cbQueryNodeConfig{
		ClusterName:   this.Name(),
		QueryNodeName: qryNodeName,
		QueryNodeUUID: uuid,
		NodeHealthy:   healthy || !ok, // if a node isn't there don't assume is unhealthy
	}

	// We find the host based on query name
	queryHost, _ := server.HostNameandPort(qryNodeName)

	// Since we are using it in the URL
	if strings.Contains(queryHost, ":") {
		queryHost = "[" + queryHost + "]"
	}

	for protocol, port := range this.queryNodeServices[qryNodeName] {
		switch protocol {
		case _HTTP:
			qryNode.Query = makeURL(protocol, queryHost, port, http.ServicePrefix())
			qryNode.Admin = makeURL(protocol, queryHost, port, http.AdminPrefix())
		case _HTTPS:
			qryNode.QuerySSL = makeURL(protocol, queryHost, port, http.ServicePrefix())
			qryNode.AdminSSL = makeURL(protocol, queryHost, port, http.AdminPrefix())
		}
	}

	this.Lock()
	this.queryNodes[name] = qryNode
	this.Unlock()
	return qryNode, nil
}

func (this *cbCluster) Datastore() datastore.Datastore {
	return this.dataStore
}

func (this *cbCluster) AccountingStore() accounting.AccountingStore {
	return this.acctStore
}

func (this *cbCluster) ConfigurationStore() clustering.ConfigurationStore {
	return this.configStore
}

func (this *cbCluster) Version() clustering.Version {
	if this.version == nil {
		this.version = clustering.NewVersion(this.VersionString)
	}
	return this.version
}

func (this *cbCluster) ClusterManager() clustering.ClusterManager {
	return this
}

func (this *cbCluster) Capability(name string) bool {

	// dirty trick to refresh the cluster and load the capabilities
	_, err := this.QueryNodeNames()
	if err != nil {
		return false
	}

	this.RLock()
	rv := this.capabilities[name]
	this.RUnlock()
	return rv
}

func (this *cbCluster) Settings() (map[string]interface{}, errors.Error) {
	pool, err := this.configStore.(*cbConfigStore).cbConn.GetPool(this.ClusterName)
	if err != nil {
		return nil, errors.NewAdminGetClusterError(err, this.ClusterName)
	}

	out := map[string]interface{}{"memory": map[string]interface{}{
		"kv":       pool.MemoryQuota,
		"cbas":     pool.CbasMemoryQuota,
		"eventing": pool.EventingMemoryQuota,
		"fts":      pool.FtsMemoryQuota,
		"index":    pool.IndexMemoryQuota,
	},
	}
	return out, nil
}

// cbCluster implements clustering.ClusterManager interface
func (this *cbCluster) Cluster() clustering.Cluster {
	return this
}

func (this *cbCluster) AddQueryNode(n clustering.QueryNode) (clustering.QueryNode, errors.Error) {
	// NOP. This is a read-only implementation
	return nil, nil
}

func (this *cbCluster) RemoveQueryNode(n clustering.QueryNode) (clustering.QueryNode, errors.Error) {
	return this.RemoveQueryNodeByName(n.Name())
}

func (this *cbCluster) RemoveQueryNodeByName(name string) (clustering.QueryNode, errors.Error) {
	// NOP. This is a read-only implementation
	return nil, nil
}

func (this *cbCluster) GetQueryNodes() ([]clustering.QueryNode, errors.Error) {
	qryNodes := []clustering.QueryNode{}
	names, err := this.QueryNodeNames()
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		qryNode, err := this.QueryNodeByName(name)
		if err != nil {
			return nil, err
		}
		qryNodes = append(qryNodes, qryNode)
	}
	return qryNodes, nil
}

const _SYSTEM_LOG_PATH = "/_event"
const _NUM_RETRIES = 5

func (this *cbCluster) ReportEventAsync(event string) {
	// local recovery as faults may be reported as events
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		logging.Stackf(logging.ERROR, "Failed to report event: %v. Event: %v", r, event)
	}()
	hostport := strings.TrimPrefix(strings.TrimPrefix(this.DatastoreURI, "http://"), "https://")
	u, p, err := cbauth.Default.GetHTTPServiceAuth(hostport)
	if err != nil {
		logging.Errorf("Failed to obtain credentials for %v for event logging: %v. Event: %v", hostport, err, event)
		return
	}
	url := this.DatastoreURI + _SYSTEM_LOG_PATH

	go func() {
		defer func() {
			r := recover()
			if r == nil {
				return
			}
			logging.Stackf(logging.ERROR, "Failed to report event: %v. Event: %v", r, event)
		}()

		_, err := couchbase.InvokeEndpointWithRetry(url, u, p, "POST", "application/json", event, _NUM_RETRIES)
		if err != nil {
			logging.Errora(func() string {
				return fmt.Sprintf("Failed to report event: %v. URL: %v Event: %v", err, url, event)
			})
		}
	}()
}

func (this *cbCluster) NodeUUID(host string) (string, errors.Error) {
	qni, err := this.QueryNodeByName(host)
	if err != nil {
		return "", err
	}
	return qni.NodeUUID(), nil
}

func (this *cbCluster) UUIDToHost(uuid string) (string, errors.Error) {
	_, err := this.QueryNodeNames()
	if err != nil {
		return "", err
	}
	for h, u := range this.queryNodeUUIDs {
		if u == uuid {
			return h, nil
		}
	}
	return "", errors.NewAdminNoNodeError(uuid)
}

// cbQueryNodeConfig implements clustering.QueryNode
type cbQueryNodeConfig struct {
	ClusterName   string                    `json:"cluster"`
	QueryNodeName string                    `json:"name"`
	QueryNodeUUID string                    `json:"uuid"`
	NodeHealthy   bool                      `json:"-"`
	Query         string                    `json:"queryEndpoint,omitempty"`
	Admin         string                    `json:"adminEndpoint,omitempty"`
	QuerySSL      string                    `json:"querySecure,omitempty"`
	AdminSSL      string                    `json:"adminSecure,omitempty"`
	ClusterRef    *cbCluster                `json:"-"`
	StandaloneRef *clustering.StdStandalone `json:"-"`
	OptionsCL     *clustering.ClOptions     `json:"options"`
}

// cbQueryNodeConfig implements Stringer interface
func (this *cbQueryNodeConfig) String() string {
	return getJsonString(this)
}

// cbQueryNodeConfig implements clustering.QueryNode interface
func (this *cbQueryNodeConfig) Cluster() clustering.Cluster {
	return this.ClusterRef
}

func (this *cbQueryNodeConfig) Name() string {
	return this.QueryNodeName
}

func (this *cbQueryNodeConfig) QueryEndpoint() string {
	return this.Query
}

func (this *cbQueryNodeConfig) ClusterEndpoint() string {
	return this.Admin
}

func (this *cbQueryNodeConfig) QuerySecure() string {
	return this.QuerySSL
}

func (this *cbQueryNodeConfig) ClusterSecure() string {
	return this.AdminSSL
}

func (this *cbQueryNodeConfig) Standalone() clustering.Standalone {
	return this.StandaloneRef
}

func (this *cbQueryNodeConfig) Options() clustering.QueryNodeOptions {
	return this.OptionsCL
}

func (this *cbQueryNodeConfig) NodeUUID() string {
	return this.QueryNodeUUID
}

func (this *cbQueryNodeConfig) Healthy() bool {
	return this.NodeHealthy
}

func getJsonString(i interface{}) string {
	serialized, _ := json.Marshal(i)
	s := bytes.NewBuffer(append(serialized, '\n'))
	return s.String()
}

var pollStdin util.Once

func doPollStdin() {
	reader := bufio.NewReader(os.Stdin)
	for {
		ch, err := reader.ReadByte()
		if err == io.EOF || (err == nil && (ch == '\n' || ch == '\r')) {
			logging.Infof("Received EOF or EOL.")
			os.Exit(0)
		} else if err != nil {
			logging.Errorf("Unexpected error polling stdin: %v", err)
			os.Exit(1)
		}
	}
}

func makeURL(protocol string, host string, port int, endpoint string) string {
	if port == 0 {
		return ""
	}
	urlParts := []string{protocol, "://", host, ":", strconv.Itoa(port), endpoint}
	return strings.Join(urlParts, "")
}
