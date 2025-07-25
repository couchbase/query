//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

/*

Package couchbase provides a couchbase-server implementation of the datastore
package.

*/

package couchbase

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/couchbase/cbauth"
	json "github.com/couchbase/go_json"
	"github.com/couchbase/gomemcached"
	memcached "github.com/couchbase/gomemcached/client" // package name is memcached
	gsi "github.com/couchbase/indexing/secondary/queryport/n1ql"
	ftsclient "github.com/couchbase/n1fty"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/couchbase/gcagent"
	"github.com/couchbase/query/datastore/virtual"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/logging"
	cb "github.com/couchbase/query/primitives/couchbase"
	"github.com/couchbase/query/sequences"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/transactions"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
	"github.com/golang/snappy"
)

var REQUIRE_CBAUTH bool           // Connection to authorization system must succeed.
var _SKIP_IMPERSONATE bool = true //  don't send actual user names

// cbPoolMap and cbPoolServices implement a local cache of the datastore's topology
type cbPoolMap struct {
	sync.RWMutex
	poolServices map[string]*cbPoolServices
}

type cbPoolServices struct {
	sync.RWMutex
	name         string
	lastUpdate   util.Time
	rev          int
	pool         cb.Pool
	nodeServices map[string]interface{}
}

const _INFO_INTERVAL = time.Second

var _POOLMAP cbPoolMap

const (
	PRIMARY_INDEX          = "#primary"
	_TRAN_CLEANUP_INTERVAL = 1 * time.Minute
)

type mutationState int // state of the mutation operation

const (
	_MUTATED mutationState = iota // if the key was mutated successfully
	_STOPPED                      // if the mutation operation was stopped
	_FAILED                       // if the mutation failed a reason
	_NONE
)

const (
	_DEFAULT_CONN       = 64
	_SERVERLESS_CONN    = 16
	_OVERFLOW_CONN      = 64
	_DEFAULT_TIMEOUT    = 30 * time.Second
	_SERVERLESS_TIMEOUT = 20 * time.Second
)

// Max number of mutation workers
// 1 routine for every 4 CPU cores
// But, a max of 4 go routines are allowed
var _MAX_MUTATION_ROUTINES = util.MinInt(util.MaxInt(1, int(util.NumCPU()/4)), 4)

// bucket capabilities used for migration
var bucketCapabilities = map[string]datastore.Migration{
	"querySystemCollection": datastore.HAS_SYSTEM_COLLECTION,
}

var migration2Capability = map[datastore.Migration]string{
	datastore.HAS_SYSTEM_COLLECTION: "querySystemCollection",
}

func init() {

	// MB-27415 have a larger overflow pool and close overflow connections asynchronously
	cb.SetConnectionPoolParams(_DEFAULT_CONN, _OVERFLOW_CONN)
	cb.EnableAsynchronousCloser(true, _DEFAULT_TIMEOUT)

	val, err := strconv.ParseBool(os.Getenv("REQUIRE_CBAUTH"))
	if err != nil {
		REQUIRE_CBAUTH = val
	} else {
		REQUIRE_CBAUTH = true // default
	}

	// enable data type response
	cb.EnableDataType = true

	// enable xattrs
	cb.EnableXattr = true

	// start the fetch workers for servicing the BulkGet operations
	cb.InitBulkGet()
	_POOLMAP.poolServices = make(map[string]*cbPoolServices, 1)

	// Enable sync replication (durability)
	cb.EnableSyncReplication = true

	// Enable collections
	cb.EnableCollections = true

	// Enable Preserve Expiry
	cb.EnablePreserveExpiry = true

	// Enable KV Error maps
	cb.EnableXerror = true

	// transaction cache initialization
	transactions.TranContextCacheInit(_TRAN_CLEANUP_INTERVAL)

}

// Pass Deployment Model to gsi+n1fty
func SetDeploymentModel(deploymentModel string) {
	if deploymentModel == datastore.DEPLOYMENT_MODEL_SERVERLESS {
		cb.SetConnectionPoolParams(_SERVERLESS_CONN, _OVERFLOW_CONN)
		cb.EnableAsynchronousCloser(true, _SERVERLESS_TIMEOUT)
	}
	gsi.SetDeploymentModel(deploymentModel)
	ftsclient.SetDeploymentModel(deploymentModel)
}

// store is the root for the couchbase datastore
type store struct {
	client         cb.Client // instance of primitives/couchbase client
	gcClient       *gcagent.Client
	namespaceCache map[string]*namespace // map of pool-names and IDs
	CbAuthInit     bool                  // whether cbAuth is initialized
	inferencer     datastore.Inferencer  // what we use to infer schemas
	statUpdater    datastore.StatUpdater // what we use to update statistics
	connectionUrl  string                // where to contact ns_server
	connSecConfig  *datastore.ConnectionSecurityConfig
	nslock         sync.RWMutex
}

func (s *store) Id() string {
	return s.URL()
}

func (s *store) URL() string {
	return s.client.BaseURL.String()
}

func (s *store) Info() datastore.Info {
	info := &infoImpl{client: &s.client}
	return info
}

type infoImpl struct {
	client *cb.Client
}

func (info *infoImpl) Version() string {
	return info.client.Info.ImplementationVersion
}

func fullhostName(n string) string {
	hostName, portVal := server.HostNameandPort(n)
	if hostName != "" && portVal != "" {
		return n
	}
	if portVal != "" {
		portVal = ":" + portVal
	}
	return server.GetIP(true) + portVal
}

func (info *infoImpl) Topology() ([]string, []errors.Error) {
	var nodes []string

	isReadLock, errs := info.refresh()
	for _, p := range _POOLMAP.poolServices {
		p.RLock()
		for node, _ := range p.nodeServices {
			nodes = append(nodes, node)
		}
		p.RUnlock()
	}
	if isReadLock {
		_POOLMAP.RUnlock()
	} else {
		_POOLMAP.Unlock()
	}
	return nodes, errs
}

func (info *infoImpl) Services(node string) (map[string]interface{}, []errors.Error) {
	isReadLock, errs := info.refresh()

	defer func() {
		if isReadLock {
			_POOLMAP.RUnlock()
		} else {
			_POOLMAP.Unlock()
		}
	}()
	for _, p := range _POOLMAP.poolServices {
		p.RLock()
		n, ok := p.nodeServices[node]
		if ok {
			// Return a safely writeable copy of the information.  This is not a full value copy since the values themselves can
			// typically be changed without issue, but the map can't.
			m := n.(map[string]interface{})
			ret := make(map[string]interface{}, len(m))
			for k, v := range m {
				ret[k] = v
			}
			p.RUnlock()
			return ret, nil
		}
		p.RUnlock()
	}
	return map[string]interface{}{}, errs
}

func (info *infoImpl) refresh() (bool, []errors.Error) {
	var errs []errors.Error

	isReadLock := true
	_POOLMAP.RLock()

	// scan the pools
	for _, p := range info.client.Info.Pools {
		poolEntry, found := _POOLMAP.poolServices[p.Name]
		if found && util.Since(poolEntry.lastUpdate) < _INFO_INTERVAL {
			continue
		}

		pool, err := info.client.GetPool(p.Name)
		poolServices, pErr := info.client.GetPoolServices(p.Name)

		if err == nil && pErr == nil {

			// missing the information, rebuild
			if !found || poolEntry.rev != poolServices.Rev {

				// promote the lock
				if isReadLock {
					var ok bool

					_POOLMAP.RUnlock()
					_POOLMAP.Lock()
					isReadLock = false

					// now that we have promoted the lock, did we get beaten by somebody else to it?
					poolEntry, ok = _POOLMAP.poolServices[p.Name]
					if ok && (poolEntry.rev == poolServices.Rev) {
						continue
					}
				}

				newPoolServices := &cbPoolServices{name: p.Name, rev: poolServices.Rev}
				nodeServices := make(map[string]interface{}, len(pool.Nodes))

				// go through all the nodes in the pool
				for _, n := range pool.Nodes {
					var servicesCopy []interface{}

					newServices := make(map[string]interface{}, 4)
					newServices["name"] = fullhostName(n.Hostname)
					newServices["uuid"] = n.NodeUUID
					newServices["status"] = n.Status
					for _, s := range n.Services {
						servicesCopy = append(servicesCopy, s)
					}
					newServices["services"] = servicesCopy

					// go through all bucket independet services in the pool
					for _, ns := range poolServices.NodesExt {

						mgmtPort := ns.Services["mgmt"]
						if mgmtPort == 0 {

							// shouldn't happen, there should always be a mgmt port on each node
							// we should return an error
							msg := fmt.Sprintf("NodeServices does not report mgmt endpoint for this node: %v", newServices["name"])
							errs = append(errs, errors.NewAdminGetNodeError(nil, msg))
							continue
						}

						nsHostname := ""
						if ns.Hostname != "" {
							nsHostname = ns.Hostname + ":" + strconv.Itoa(mgmtPort)
						}
						// if we can positively match nodeServices and node, add ports
						if n.Hostname == nsHostname ||
							(nsHostname == "" && ns.ThisNode && n.ThisNode) {
							ports := make(map[string]interface{}, len(ns.Services))

							// only add the ports for those services that are advertised
							for _, s := range n.Services {
								for pn, p := range ns.Services {
									if strings.Index(pn, s) == 0 {
										ports[pn] = p
									}
								}
							}
							newServices["ports"] = ports
							break
						}
					}
					nodeServices[fullhostName(n.Hostname)] = newServices
				}
				newPoolServices.nodeServices = nodeServices
				newPoolServices.lastUpdate = util.Now()
				_POOLMAP.poolServices[p.Name] = newPoolServices
			} else {

				// just update the node statuses
				poolEntry.Lock()
				for _, n := range pool.Nodes {
					m := poolEntry.nodeServices[fullhostName(n.Hostname)].(map[string]interface{})
					m["status"] = n.Status
				}
				poolEntry.lastUpdate = util.Now()
				poolEntry.Unlock()
			}
		} else {
			if err != nil {
				errs = append(errs, errors.NewDatastoreClusterError(err, p.Name))
			}
			if pErr != nil {
				errs = append(errs, errors.NewDatastoreClusterError(pErr, p.Name))
			}

			// promote the lock
			if isReadLock {
				_POOLMAP.RUnlock()
				_POOLMAP.Lock()
				isReadLock = false
			}

			// pool not found, remove any previous entry
			delete(_POOLMAP.poolServices, p.Name)
		}
		if err == nil {
			pool.Close()
		}
	}

	// cached pool map differs, cleanup
	if len(_POOLMAP.poolServices) != len(info.client.Info.Pools) {

		// promote the lock
		if isReadLock {
			_POOLMAP.RUnlock()
			_POOLMAP.Lock()
			isReadLock = false
		}

		for e, _ := range _POOLMAP.poolServices {
			found := false
			for _, p := range info.client.Info.Pools {
				if e == p.Name {
					found = true
					break
				}
			}
			if !found {
				delete(_POOLMAP.poolServices, e)
			}
		}
	}
	return isReadLock, errs
}

func (s *store) NamespaceIds() ([]string, errors.Error) {
	return s.NamespaceNames()
}

func (s *store) NamespaceNames() ([]string, errors.Error) {
	return []string{"default"}, nil
}

func (s *store) NamespaceById(id string) (p datastore.Namespace, e errors.Error) {
	return s.NamespaceByName(id)
}

func (s *store) NamespaceByName(name string) (p datastore.Namespace, e errors.Error) {
	p, ok := s.namespaceCache[name]
	if !ok {
		var err errors.Error
		p, err = loadNamespace(s, name)
		if err != nil {
			return nil, err
		}
		s.namespaceCache[name] = p.(*namespace)
	}
	return p, nil
}

func (s *store) ForeachBucket(f func(datastore.ExtendedBucket)) {
	for _, n := range s.namespaceCache {
		for _, k := range n.keyspaceCache {
			if k.cbKeyspace != nil {
				f(k.cbKeyspace)
			}
		}
	}
}

func (s *store) LoadAllBuckets(f func(datastore.ExtendedBucket)) {
	for _, n := range s.namespaceCache {
		n.refresh()
		for k, _ := range n.cbNamespace.BucketMap {
			if b, err := n.KeyspaceByName(k); err == nil && b != nil {
				if eb, ok := b.(datastore.ExtendedBucket); ok && eb != nil {
					f(eb)
				}
			}
		}
	}
}

// The ns_server admin API is open iff we can access the /pools API without a password.
func (s *store) adminIsOpen() bool {
	url := s.connectionUrl + "/pools"
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return false
	}
	return true
}

func (s *store) auth(user, pwd string) (cbauth.Creds, error) {
	return cbauth.Auth(user, pwd)
}

func (s *store) authWebCreds(req *http.Request) (cbauth.Creds, error) {
	return cbauth.AuthWebCreds(req)
}

func (s *store) Authorize(privileges *auth.Privileges, credentials *auth.Credentials) errors.Error {
	if s.CbAuthInit == false {
		// cbauth is not initialized. Access to SASL protected buckets will be
		// denied by the couchbase server
		logging.Warnf("CbAuth not intialized")
		return nil
	}
	return cbAuthorize(s, privileges, credentials, false)
}

func (s *store) AuthorizeInternal(privileges *auth.Privileges, credentials *auth.Credentials) errors.Error {
	if s.CbAuthInit == false {
		// cbauth is not initialized. Access to SASL protected buckets will be
		// denied by the couchbase server
		logging.Warnf("CbAuth not intialized")
		return nil
	}
	return cbAuthorize(s, privileges, credentials, true)
}

func (s *store) AdminUser(node string) (string, string, error) {
	if s.CbAuthInit == false {
		// cbauth is not initialized. Access to SASL protected buckets will be
		// denied by the couchbase server
		logging.Warnf("CbAuth not intialized")
		return "", "", fmt.Errorf("CbAuth not initialized")
	}
	return cbauth.GetHTTPServiceAuth(node)
}

func (s *store) GetUserUUID(creds *auth.Credentials) string {
	if creds != nil && len(creds.CbauthCredentialsList) > 0 {
		res, _ := cbauth.GetUserUuid(creds.CbauthCredentialsList[0].User())
		return res
	}
	return ""
}

func (s *store) GetUserBuckets(creds *auth.Credentials) []string {
	if s.CbAuthInit == false {
		// cbauth is not initialized. Access to SASL protected buckets will be
		// denied by the couchbase server
		logging.Warnf("CbAuth not intialized")
		return []string{}
	}
	if creds == nil || len(creds.CbauthCredentialsList) == 0 {
		return []string{}
	}
	res, _ := creds.CbauthCredentialsList[0].GetBuckets()

	return res
}

func (s *store) GetImpersonateBuckets(user, domain string) []string {
	if s.CbAuthInit == false {
		// cbauth is not initialized. Access to SASL protected buckets will be
		// denied by the couchbase server
		logging.Warnf("CbAuth not intialized")
		return []string{}
	}
	if len(user) == 0 {
		return []string{}
	}
	res, _ := cbauth.GetUserBuckets(user, domain)
	return res
}

func (s *store) PreAuthorize(privileges *auth.Privileges) {
	cbPreAuthorize(privileges)
}

func (s *store) CredsString(creds *auth.Credentials) (string, string) {
	if creds != nil && len(creds.CbauthCredentialsList) > 0 {
		u, d := creds.CbauthCredentialsList[0].User()

		// defensively
		if d == "" {
			d = "local"
		}
		return u, d
	}
	return "", ""
}

func (s *store) SetLogLevel(level logging.Level) {
	for _, n := range s.namespaceCache {
		defer n.lock.Unlock()
		n.lock.Lock()
		for _, k := range n.keyspaceCache {
			if k.cbKeyspace == nil {
				continue
			}
			indexers, _ := k.cbKeyspace.Indexers()
			if len(indexers) > 0 {
				for _, idxr := range indexers {
					idxr.SetLogLevel(level)
				}

				return
			}
		}
	}
}

// Ignore the name parameter for now
func (s *store) Inferencer(name datastore.InferenceType) (datastore.Inferencer, errors.Error) {
	return s.inferencer, nil
}

func (s *store) Inferencers() ([]datastore.Inferencer, errors.Error) {
	return []datastore.Inferencer{s.inferencer}, nil
}

func (s *store) StatUpdater() (datastore.StatUpdater, errors.Error) {
	return s.statUpdater, nil
}

func (s *store) AuditInfo() (*datastore.AuditInfo, errors.Error) {
	auditSpec, err := s.client.GetAuditSpec()
	if err != nil {
		return nil, errors.NewSystemUnableToRetrieveError(err, "audit information")
	}

	users := make(map[datastore.UserInfo]bool, len(auditSpec.DisabledUsers))
	for _, u := range auditSpec.DisabledUsers {
		ui := datastore.UserInfo{Name: u.Name, Domain: u.Domain}
		users[ui] = true
	}

	events := make(map[uint32]bool, len(auditSpec.Disabled))
	for _, eid := range auditSpec.Disabled {
		events[eid] = true
	}

	ret := &datastore.AuditInfo{
		EventDisabled:   events,
		UserAllowlisted: users,
		AuditEnabled:    auditSpec.AuditdEnabled,
		Uid:             auditSpec.Uid,
	}
	return ret, nil
}

func (s *store) ProcessAuditUpdateStream(callb func(uid string) error) errors.Error {
	f := func(data interface{}) error {
		d, ok := data.(*DefaultObject)
		if !ok {
			return fmt.Errorf("Unable to convert received object to proper type: %T", data)
		}
		return callb(d.Uid)
	}
	do := DefaultObject{}
	err := s.client.ProcessStream("/poolsStreaming/default", f, &do)
	if err != nil {
		return errors.NewAuditStreamHandlerFailed(err)
	}
	return nil
}

func (s *store) EnableStorageAudit(val bool) {
	_SKIP_IMPERSONATE = !val
}

type DefaultObject struct {
	Uid string `json:"auditUid"`
}

func (s *store) UserInfo() (value.Value, errors.Error) {
	data, err := s.client.GetUserRoles()
	if err != nil {
		return nil, errors.NewSystemUnableToRetrieveError(err, "user information")
	}
	return value.NewValue(data), nil
}

func (s *store) GetUserInfoAll() ([]datastore.User, errors.Error) {
	sourceUsers, err := s.client.GetUserInfoAll()
	if err != nil {
		return nil, errors.NewSystemUnableToRetrieveError(err, "user information")
	}
	resultUsers := make([]datastore.User, len(sourceUsers))
	for i, u := range sourceUsers {
		resultUsers[i].Name = u.Name
		resultUsers[i].Id = u.Id
		resultUsers[i].Domain = u.Domain
		roles := make([]datastore.Role, len(u.Roles))
		for j, r := range u.Roles {
			roles[j].Name = r.Role
			if r.CollectionName != "" && r.CollectionName != "*" {
				roles[j].Target = r.BucketName + ":" + r.ScopeName + ":" + r.CollectionName
			} else if r.ScopeName != "" && r.ScopeName != "*" {
				roles[j].Target = r.BucketName + ":" + r.ScopeName
			} else if r.BucketName != "" {
				roles[j].Target = r.BucketName
			}
		}
		resultUsers[i].Roles = roles
	}
	return resultUsers, nil
}

func (s *store) PutUserInfo(u *datastore.User) errors.Error {
	var outputUser cb.User
	outputUser.Name = u.Name
	outputUser.Id = u.Id
	outputUser.Roles = make([]cb.Role, len(u.Roles))
	outputUser.Domain = u.Domain
	for i, r := range u.Roles {
		outputUser.Roles[i].Role = r.Name
		if len(r.Target) > 0 {
			outputUser.Roles[i].BucketName = r.Target
		}
	}
	outputUser.Password = u.Password
	outputUser.Groups = u.Groups
	err := s.client.PutUserInfo(&outputUser)
	if err != nil {
		return errors.NewSystemUnableToUpdateError(err, "user")
	}
	return nil
}

func (s *store) DeleteUser(u *datastore.User) errors.Error {
	var outputUser cb.User
	outputUser.Id = u.Id
	outputUser.Domain = u.Domain
	err := s.client.DeleteUser(&outputUser)
	if err != nil {
		return errors.NewSystemUnableToUpdateError(err, "user")
	}
	return nil
}

func (s *store) GetUserInfo(u *datastore.User) errors.Error {
	var outputUser cb.User
	outputUser.Id = u.Id
	outputUser.Domain = u.Domain
	err := s.client.GetUserInfo(&outputUser)
	if err != nil {
		return errors.NewSystemUnableToRetrieveError(err, "user information")
	}
	u.Id = outputUser.Id
	u.Domain = outputUser.Domain
	if len(outputUser.Roles) > 0 {
		u.Roles = make([]datastore.Role, len(outputUser.Roles))
		for i, v := range outputUser.Roles {
			u.Roles[i].Name = v.Role
			u.Roles[i].Target = v.BucketName
			u.Roles[i].IsScope = v.ScopeName != "" && v.CollectionName == ""
		}
	}
	u.Name = outputUser.Name
	u.Groups = outputUser.Groups
	return nil
}

func (s *store) GetRolesAll() ([]datastore.Role, errors.Error) {
	roleDescList, err := s.client.GetRolesAll()
	if err != nil {
		return nil, errors.NewDatastoreUnableToRetrieveRoles(err)
	}
	roles := make([]datastore.Role, len(roleDescList))
	for i, rd := range roleDescList {
		roles[i].Name = rd.Role
		roles[i].Target = rd.BucketName
		roles[i].IsScope = rd.ScopeName != "" && rd.CollectionName == ""
	}
	return roles, nil
}

func (s *store) GetGroupInfo(g *datastore.Group) errors.Error {
	var outputGroup cb.Group
	outputGroup.Id = g.Id
	err := s.client.GetGroupInfo(&outputGroup)
	if err != nil {
		return errors.NewSystemUnableToRetrieveError(err, "group information")
	}
	g.Id = outputGroup.Id
	g.Desc = outputGroup.Desc
	if len(outputGroup.Roles) > 0 {
		g.Roles = make([]datastore.Role, len(outputGroup.Roles))
		for i := range outputGroup.Roles {
			g.Roles[i].Name = outputGroup.Roles[i].Role
			if outputGroup.Roles[i].BucketName != "" {
				g.Roles[i].Target = outputGroup.Roles[i].BucketName
				if outputGroup.Roles[i].ScopeName != "" && outputGroup.Roles[i].ScopeName != "*" {
					g.Roles[i].Target += ":" + outputGroup.Roles[i].ScopeName
					if outputGroup.Roles[i].CollectionName != "" && outputGroup.Roles[i].CollectionName != "*" {
						g.Roles[i].Target += ":" + outputGroup.Roles[i].CollectionName
					}
				}
			}
		}
	} else {
		g.Roles = nil
	}
	return nil
}

func (s *store) PutGroupInfo(g *datastore.Group) errors.Error {
	var outputGroup cb.Group
	outputGroup.Id = g.Id
	outputGroup.Desc = g.Desc
	if len(g.Roles) > 0 {
		outputGroup.Roles = make([]cb.Role, len(g.Roles))
		for i := range g.Roles {
			outputGroup.Roles[i].Role = g.Roles[i].Name
			if len(g.Roles[i].Target) > 0 {
				parts := strings.Split(g.Roles[i].Target, ":")
				outputGroup.Roles[i].BucketName = parts[0]
				if len(parts) > 1 {
					outputGroup.Roles[i].ScopeName = parts[1]
				}
				if len(parts) > 2 {
					outputGroup.Roles[i].CollectionName = parts[2]
				}
			}
		}
	}
	err := s.client.PutGroupInfo(&outputGroup)
	if err != nil {
		return errors.NewSystemUnableToUpdateError(err, "group")
	}
	return nil
}

func (s *store) DeleteGroup(g *datastore.Group) errors.Error {
	var outputGroup cb.Group
	outputGroup.Id = g.Id
	err := s.client.DeleteGroup(&outputGroup)
	if err != nil {
		return errors.NewSystemUnableToUpdateError(err, "group")
	}
	return nil
}

func (s *store) GroupInfo() (value.Value, errors.Error) {
	sourceGroups, err := s.client.GroupInfo()
	if err != nil {
		return nil, errors.NewSystemUnableToRetrieveError(err, "group information")
	}
	return value.NewValue(sourceGroups), nil
}

func (s *store) GetGroupInfoAll() ([]datastore.Group, errors.Error) {
	sourceGroups, err := s.client.GetGroupInfoAll()
	if err != nil {
		return nil, errors.NewSystemUnableToRetrieveError(err, "group information")
	}
	resultGroups := make([]datastore.Group, len(sourceGroups))
	for i, g := range sourceGroups {
		resultGroups[i].Id = g.Id
		resultGroups[i].Desc = g.Desc
		roles := make([]datastore.Role, len(g.Roles))
		for j, r := range g.Roles {
			roles[j].Name = r.Role
			if r.CollectionName != "" && r.CollectionName != "*" {
				roles[j].Target = r.BucketName + ":" + r.ScopeName + ":" + r.CollectionName
			} else if r.ScopeName != "" && r.ScopeName != "*" {
				roles[j].Target = r.BucketName + ":" + r.ScopeName
			} else if r.BucketName != "" {
				roles[j].Target = r.BucketName
			}
		}
		resultGroups[i].Roles = roles
	}
	return resultGroups, nil
}

func (s *store) CreateBucket(name string, with value.Value) errors.Error {
	withArg := with.CopyForUpdate().Actual().(map[string]interface{})
	withArg["name"] = name
	b, _ := json.Marshal(withArg)
	var param map[string]interface{}
	json.Unmarshal(b, &param)
	err := s.client.CreateBucket(param)
	if err != nil {
		return errors.NewSystemUnableToUpdateError(err, "bucket")
	}
	return nil
}

func (s *store) AlterBucket(name string, with value.Value) errors.Error {
	b, _ := json.Marshal(with)
	var param map[string]interface{}
	json.Unmarshal(b, &param)
	err := s.client.AlterBucket(name, param)
	if err != nil {
		return errors.NewSystemUnableToUpdateError(err, "bucket")
	}
	return nil
}

func (s *store) DropBucket(name string) errors.Error {
	err := s.client.DropBucket(name)
	if err != nil {
		return errors.NewSystemUnableToUpdateError(err, "bucket")
	}
	return nil
}

func (s *store) BucketInfo() (value.Value, errors.Error) {
	sourceBuckets, err := s.client.BucketInfo()
	if err != nil {
		return nil, errors.NewSystemUnableToRetrieveError(err, "bucket information")
	}
	return value.NewValue(sourceBuckets), nil
}

func (s *store) SetClientConnectionSecurityConfig() (err error) {
	if s.connSecConfig != nil &&
		s.connSecConfig.ClusterEncryptionConfig.EncryptData {

		// For every initTLS call when info is refreshed pass the
		// cert and key info along with passphrase to client.

		err = s.client.InitTLS(s.connSecConfig.CAFile,
			s.connSecConfig.CertFile,
			s.connSecConfig.KeyFile,
			s.connSecConfig.ClusterEncryptionConfig.DisableNonSSLPorts,
			s.connSecConfig.TLSConfig.PrivateKeyPassphrase,
			s.connSecConfig.TLSConfig.ShouldClientsUseClientCert,
			s.connSecConfig.InternalClientCertFile,
			s.connSecConfig.InternalClientKeyFile,
			s.connSecConfig.TLSConfig.ClientPrivateKeyPassphrase)

		if err == nil && s.gcClient != nil {
			err = s.gcClient.InitTLS(s.connSecConfig.CAFile,
				s.connSecConfig.CertFile,
				s.connSecConfig.KeyFile,
				s.connSecConfig.TLSConfig.PrivateKeyPassphrase,
				s.connSecConfig.TLSConfig.ShouldClientsUseClientCert,
				s.connSecConfig.InternalClientCertFile,
				s.connSecConfig.InternalClientKeyFile,
				s.connSecConfig.TLSConfig.ClientPrivateKeyPassphrase)
		}
		if err != nil {
			if len(s.connSecConfig.CAFile) > 0 {
				err = fmt.Errorf("Unable to initialize TLS using certificate %s. Aborting security update. Error:%v",
					s.connSecConfig.CAFile, err)
			} else {
				err = fmt.Errorf("Unable to initialize TLS using certificate %s. Aborting security update. Error:%v",
					s.connSecConfig.CertFile, err)
			}

			logging.Errorf("%v", err)
			return
		}
	} else {
		s.client.ClearTLS()
		if s.gcClient != nil {
			s.gcClient.ClearTLS()
		}
	}
	return
}

func (s *store) SetConnectionSecurityConfig(connSecConfig *datastore.ConnectionSecurityConfig) {
	s.connSecConfig = connSecConfig
	if err := s.SetClientConnectionSecurityConfig(); err != nil {
		return
	}

	gsi.SetConnectionSecurityConfig(connSecConfig)
	ftsclient.SetConnectionSecurityConfig(connSecConfig)

	// for any active buckets set new security config
	for _, n := range s.namespaceCache {

		// force a full pool refresh
		n.refreshFully()
		n.lock.Lock()
		for _, k := range n.keyspaceCache {
			if k.cbKeyspace == nil {
				continue
			}

			// Make new TLS settings take effect in the buckets.
			if k.cbKeyspace.agentProvider != nil {
				k.cbKeyspace.agentProvider.Refresh()
			}

			// Pass new settings to indexers.
			indexers, _ := k.cbKeyspace.Indexers()
			if len(indexers) > 0 {
				for _, idxr := range indexers {
					idxr.SetConnectionSecurityConfig(connSecConfig)
				}
			}
		}
		n.lock.Unlock()
	}
}

func initCbAuth(url string) (*cb.Client, error) {

	transport := cbauth.WrapHTTPTransport(cb.HTTPTransport, nil)
	cb.HTTPClient.Transport = transport

	client, err := cb.ConnectWithAuth(url, cbauth.NewAuthHandler(nil), cb.USER_AGENT)
	if err != nil {
		return nil, err
	}

	logging.Infof(" Initialization of cbauth succeeded ")

	return &client, nil
}

func parseUrl(u string) (host string, username string, password string, err error) {
	url, err := url.Parse(u)
	if err != nil {
		return "", "", "", err
	}
	if url.User == nil {
		return "", "", "", fmt.Errorf("Unusable url %s. No user information.", u)
	}
	password, _ = url.User.Password()
	if password == "" {
		logging.Warnf("No password found in url <ud>%s</ud>", u)
	}
	if url.User.Username() == "" {
		logging.Warnf("No username found in url <ud>%s</ud>", u)
	}
	return url.Host, url.User.Username(), password, nil
}

// NewStore creates a new Couchbase store for the given url.
// In the main server, and error return here will cause the server to shut down.
func NewDatastore(u string) (s datastore.Datastore, e errors.Error) {
	var client cb.Client
	var cbAuthInit bool
	var err error
	var connectionUrl string

	// initialize cbauth
	c, err := initCbAuth(u)
	if err != nil {
		logging.Errorf("Unable to initialize cbauth. Error %v", err)

		// intialize cb_auth variables manually
		host, username, password, err := parseUrl(u)
		if err != nil {
			logging.Warnf("Unable to parse url <ud>%s</ud>: %v", u, err)
		} else {
			set, err := cbauth.InternalRetryDefaultInit(host, username, password)
			if set == false || err != nil {
				logging.Errorf("Unable to initialize cbauth variables. Error %v", err)
			} else {
				c, err = initCbAuth("http://" + host)
				if err != nil {
					logging.Errorf("Unable to initialize cbauth. Error %v", err)
				} else {
					client = *c
					cbAuthInit = true
					connectionUrl = "http://" + host
				}
			}
		}
	} else {
		client = *c
		cbAuthInit = true
		connectionUrl = u
	}

	if !cbAuthInit {
		if REQUIRE_CBAUTH {
			return nil, errors.NewUnableToInitCbAuthError(err)
		}
		// connect without auth
		logging.Warnf("Unable to initialize cbAuth, access to couchbase buckets may be restricted")
		cb.HTTPClient = &http.Client{}
		client, err = cb.Connect(u, cb.USER_AGENT)
		if err != nil {
			return nil, errors.NewCbConnectionError(err, "url "+u)
		}
	}

	store := &store{
		client:         client,
		namespaceCache: make(map[string]*namespace),
		CbAuthInit:     cbAuthInit,
		connectionUrl:  connectionUrl,
	}

	// get the schema inferencer
	var er errors.Error
	store.inferencer, er = GetDefaultInferencer(store)
	if er != nil {
		return nil, er
	}

	// get statistics updater
	store.statUpdater, er = GetDefaultStatUpdater(store)
	if er != nil {
		return nil, er
	}

	// initialize the default pool.
	// TODO can couchbase server contain more than one pool ?

	defaultPool, er := loadNamespace(store, "default")
	if er != nil {
		logging.Errorf("Cannot connect to default pool: %v", er)
		return nil, er
	}

	store.namespaceCache["default"] = defaultPool
	logging.Infof("New store created with url %s", u)

	tenant.RegisterResourceManager(func(bucket string) { store.manageTenant(bucket) })

	if tenant.IsServerless() {
		cb.EnableComputeUnits = true
		cb.EnableHandleThrottle = true
		cb.Suspend = tenant.Suspend
		cb.IsSuspended = tenant.IsSuspended
	}

	return store, nil
}

func (s *store) manageTenant(bucket string) {
	var cbKeyspace *keyspace

	p := s.namespaceCache["default"]

	// nothing to unload
	if p == nil {
		return
	}
	logging.Infof("Unloading tenant %v", bucket)
	p.lock.Lock()

	ks, ok := p.keyspaceCache[bucket]
	if ok && ks.cbKeyspace != nil {
		cbKeyspace = ks.cbKeyspace
		ks.cbKeyspace.Release(false)
		delete(p.keyspaceCache, bucket)

		// keyspace has been deleted, force full auto reprepare check
		p.version++
	} else {
		logging.Warnf("Keyspace %v not configured on this server", bucket)
	}
	p.lock.Unlock()

	if cbKeyspace != nil {
		if isSysBucket(cbKeyspace.name) {
			DropDictionaryCache()
		} else {
			// clearDictCacheEntries() needs to be called outside p.lock
			// since it'll need to lock it when trying to delete from
			// system collection
			clearDictCacheEntries(cbKeyspace)
		}
	}
}

func loadNamespace(s *store, name string) (*namespace, errors.Error) {
	cbpool, err := s.client.GetPool(name)
	if err != nil {
		if name == "default" {
			// if default pool is not available, try reconnecting to the server
			url := s.URL()

			var client cb.Client

			if s.CbAuthInit == true {
				client, err = cb.ConnectWithAuth(url, cbauth.NewAuthHandler(nil), cb.USER_AGENT)
			} else {
				client, err = cb.Connect(url, cb.USER_AGENT)
			}
			if err != nil {
				return nil, errors.NewCbNamespaceNotFoundError(err, name)
			}
			// check if the default pool exists
			cbpool, err = client.GetPool(name)
			if err != nil {
				return nil, errors.NewCbNamespaceNotFoundError(err, name)
			}
			s.client = client

			err = s.SetClientConnectionSecurityConfig()
			if err != nil {
				return nil, errors.NewCbNamespaceNotFoundError(err, name)
			}
		} else {
			logging.Errorf(" Error while retrieving pool %v", err)
		}
	}

	rv := namespace{
		store:         s,
		name:          name,
		cbNamespace:   &cbpool,
		keyspaceCache: make(map[string]*keyspaceEntry),
	}

	return &rv, nil
}

// full name representation of a bucket / scope / keyspace for error message purposes
func fullName(elems ...string) string {
	switch len(elems) {
	case 1:
		return elems[0]
	case 2:
		return elems[0] + ":" + elems[1]
	default:
		res := elems[0] + ":" + elems[1]
		for i := 2; i < len(elems); i++ {
			res = res + "." + elems[i]
		}
		return res
	}
}

// a namespace represents a couchbase pool
type namespace struct {
	store         *store
	name          string
	cbNamespace   *cb.Pool
	last          util.Time // last time we refreshed the pool
	keyspaceCache map[string]*keyspaceEntry
	version       uint64
	lock          sync.RWMutex // lock to guard the keyspaceCache
	nslock        sync.RWMutex // lock for this structure
}

type keyspaceEntry struct {
	sync.Mutex
	cbKeyspace *keyspace
	errCount   int
	errTime    util.Time
	lastUse    util.Time
}

func (this *keyspaceEntry) recordError() {
	// We try not to flood the log with errors
	if this.errCount == 0 {
		this.errTime = util.Now()
	} else if util.Since(this.errTime) > _MIN_ERR_INTERVAL {
		this.errTime = util.Now()
	}
	this.errCount++
}

const (
	_MIN_ERR_INTERVAL            time.Duration = 5 * time.Second
	_THROTTLING_TIMEOUT          time.Duration = 10 * time.Millisecond
	_CLEANUP_INTERVAL            time.Duration = time.Hour
	_NAMESPACE_REFRESH_THRESHOLD time.Duration = 100 * time.Millisecond
	_STATS_REFRESH_THRESHOLD     time.Duration = 1 * time.Second
)

func (p *namespace) Datastore() datastore.Datastore {
	return p.store
}

func (p *namespace) Id() string {
	return p.Name()
}

func (p *namespace) Name() string {
	return p.name
}

func (p *namespace) KeyspaceIds() ([]string, errors.Error) {
	return p.KeyspaceNames()
}

func (p *namespace) KeyspaceNames() ([]string, errors.Error) {
	p.refresh()
	p.nslock.RLock()
	rv := make([]string, len(p.cbNamespace.BucketMap))
	i := 0
	for name, _ := range p.cbNamespace.BucketMap {
		rv[i] = name
		i++
	}
	p.nslock.RUnlock()
	return rv, nil
}

func (p *namespace) Objects(credentials *auth.Credentials, filter func(string) bool, preload bool) (
	[]datastore.Object, errors.Error) {

	if len(credentials.CbauthCredentialsList) == 0 {
		return nil, errors.NewDatastoreUnableToRetrieveBuckets(fmt.Errorf("empty credentials"))
	}

	b, err := credentials.CbauthCredentialsList[0].GetBuckets()
	if err != nil {
		return nil, errors.NewDatastoreUnableToRetrieveBuckets(err)
	}

	rv := make([]datastore.Object, 0, len(b))
	for i := range b {
		if filter != nil && !filter(b[i]) {
			continue
		}
		rv = append(rv, datastore.Object{b[i], b[i], false, false})
	}

	p.refresh()

	i := 0

	// separate loops because rv might shrink if the entry is nether a bucket nor a keyspace
	for i = 0; i < len(rv); {
		var defaultCollection datastore.Keyspace

		name := rv[i].Name

		p.lock.RLock()
		entry := p.keyspaceCache[name]
		if entry != nil && entry.cbKeyspace != nil {
			defaultCollection = entry.cbKeyspace.defaultCollection
		}
		p.lock.RUnlock()

		if preload && defaultCollection == nil {
			ks, err := p.KeyspaceByName(name)
			if ks != nil && err == nil {
				defaultCollection = ks.(*keyspace).defaultCollection
			}
		}

		// if we have loaded the bucket, check if the bucket has a default collection
		// if we haven't loaded the bucket, see if you can get the default collection id
		// the bucket is a keyspace if the default collection exists
		if defaultCollection != nil {
			switch k := defaultCollection.(type) {
			case *collection:
				rv[i].IsKeyspace = (k != nil)
				rv[i].IsBucket = true
			case *keyspace:
				rv[i].IsKeyspace = (k != nil)
				rv[i].IsBucket = false
			}
		} else if !preload {
			bucket, _ := p.cbNamespace.GetBucket(name)
			if bucket != nil {
				_, _, err := bucket.GetCollectionCID("_default", "_default", time.Time{})
				if err == nil {
					rv[i].IsKeyspace = true
				}
			}
			rv[i].IsBucket = true
		}

		// skip entries that may have been zapped in the interim
		if !rv[i].IsKeyspace && !rv[i].IsBucket {
			if i == len(rv)-1 {
				rv = rv[:i]
			} else {
				rv = append(rv[:i], rv[i+1:]...)
			}
			continue
		}
		i++
	}
	b = nil // to aid the GC
	return rv, nil
}

func (p *namespace) KeyspaceByName(name string) (datastore.Keyspace, errors.Error) {
	rv, err := p.keyspaceByName(name)
	if rv == nil {
		// this returns a detectable nil; if a nil rv (pointer) is returned directly, the result is a non-nil interface type
		return nil, err
	}
	return rv, err
}

func (p *namespace) VirtualKeyspaceByName(path []string) (datastore.Keyspace, errors.Error) {
	return virtual.NewVirtualKeyspace(p, path)
}

func (p *namespace) keyspaceByName(name string) (*keyspace, errors.Error) {
	var keyspace, oldCbKeyspace *keyspace

	// make sure that no one is deleting the keyspace as we check
	p.lock.RLock()
	entry := p.keyspaceCache[name]
	if entry != nil {
		keyspace = entry.cbKeyspace
	}
	p.lock.RUnlock()
	if keyspace != nil {

		// shortcut if good, or only manifest needed
		switch keyspace.flags {
		case _NEEDS_MANIFEST:

			// avoid a race condition where we read a manifest while the uid is increased
			// by the bucket update callback
			for {
				mani, err := keyspace.cbbucket.GetCollectionsManifest()
				if err == nil {

					// see later: another case for shared optimistic locks.
					// only the first one in gets to change scopes, every one else's work is wasted
					scopes, defaultCollection := refreshScopesAndCollections(mani, keyspace)

					// if any other flag has been set in the interim, we go the reload route
					keyspace.Lock()
					if keyspace.flags == _NEEDS_MANIFEST {

						// another manifest arrived in the interim, and we've loaded the old one
						// try again
						if mani.Uid < keyspace.newCollectionsManifestUid {
							keyspace.Unlock()
							continue
						}

						// do not update if somebody has already done it
						if mani.Uid > keyspace.collectionsManifestUid ||
							keyspace.collectionsManifestUid == _INVALID_MANIFEST_UID {
							keyspace.collectionsManifestUid = mani.Uid
							keyspace.scopes = scopes
							logging.Infof("Refreshed manifest for bucket %v id %v", name, mani.Uid)

							// if there's no scopes fall back to bucket access
							if len(scopes) == 0 {
								keyspace.defaultCollection = keyspace
							} else {
								keyspace.defaultCollection = defaultCollection
							}
							keyspace.flags = 0
						}
					}
					keyspace.Unlock()
					if keyspace.flags == 0 {
						return keyspace, nil
					}
				} else {
					logging.Infof("Unable to retrieve collections info for bucket %s: %v", name, err)
				}
				break
			}
		case 0:
			return keyspace, nil
		}
	}

	// MB-19601 we haven't found the keyspace, so we have to load it,
	// however, there might be a flood of other requests coming in, all
	// wanting to do use the same keyspace and all needing to load it.
	// In the previous implementation all requests would first create
	// and refresh the keyspace, refreshing the indexes, etc
	// In YCSB enviroments this resulted in thousends of requests
	// flooding ns_server with buckets and ddocs load at the same time.
	// What we want instead is for one request to do the work, and all the
	// others waiting and benefiting from that work.
	// This is the exact scenario for using Shared Optimistic Locks, but,
	// sadly, they are patented by IBM, so clearly it's no go for us.
	// What we do is create the keyspace entry, and record that we are priming
	// it by locking that entry.
	// Everyone else will have to wait on the lock, and once they get it,
	// they can check on the keyspace again - if all is fine, just continue
	// if not try and load again.
	// Shared Optimistic Locks by stealth, although not as efficient (there
	// might be sequencing of would be loaders on the keyspace lock after
	// the initial keyspace loading has been done).
	// If we fail, again! then there's something wrong with the keyspace,
	// which means that retrying over and over again, we'll be loading ns_server
	// so what we do is throttle the reloads and log errors, so that the
	// powers that be are alerted that there's some resource issue.
	// Finally, since we are having to use two locks rather than one, make sure
	// that the locking sequence is predictable.
	// keyspace lock is always locked outside of the keyspace cache lock.

	// 1) create the entry if necessary, record time of loading attempt
	p.lock.Lock()
	version := p.version // used to detect pool refresh whilst we're busy here

	runCleanup := false
	refresh := false
	entry = p.keyspaceCache[name]
	if entry == nil {

		// adding a new keyspace does not force the namespace version to change
		// all previously prepared statements are still good
		entry = &keyspaceEntry{}
		p.keyspaceCache[name] = entry
		runCleanup = true
	} else if entry.cbKeyspace != nil && entry.cbKeyspace.flags&(_NEEDS_REFRESH|_DELETED) != 0 {

		// a keyspace that has been deleted or needs refreshing causes a
		// version change
		oldCbKeyspace = entry.cbKeyspace
		refresh = true
	}
	entry.lastUse = util.Now()
	p.lock.Unlock()

	// 2) serialize the loading by locking the entry
	entry.Lock()
	defer entry.Unlock()

	// 3) check if somebody has done the job for us in the interim
	if entry.cbKeyspace != nil && (!refresh || entry.cbKeyspace != oldCbKeyspace) {
		return entry.cbKeyspace, nil
	}

	// we may have to retry the loading if a refresh completes whilst we're busy here
	for {
		// 4) if previous loads resulted in errors, throttle requests
		if entry.errCount > 0 && util.Since(entry.lastUse) < _THROTTLING_TIMEOUT {
			time.Sleep(_THROTTLING_TIMEOUT)
		}

		// 5) try the loading
		k, err := newKeyspace(p, name, &version)
		if err != nil {
			entry.recordError()
			// if the bucket was closed under us, then we should retry as the pool has been refreshed
			if err.Code() == errors.E_CB_BUCKET_CLOSED {
				if k != nil && k.cbbucket != nil {
					k.cbbucket.StopUpdater()
				}
				k = nil
				continue // retry
			}
			return nil, err
		}

		// we need to be sure the pool doesn't refresh and close our bucket whilst we're adding this keyspace to the cache so we
		// acquire the lock again then check the version hasn't changed whilst we've been busy
		p.lock.Lock()
		if p.version != version {
			entry.recordError()
			version = p.version
			if k != nil && k.cbbucket != nil {
				k.cbbucket.StopUpdater()
			}
			k = nil
			p.lock.Unlock()
			continue // retry
		}
		entry.errCount = 0
		// this is the only place where entry.cbKeyspace is set
		// it is never unset - so it's safe to test cbKeyspace != nil
		entry.cbKeyspace = k
		p.lock.Unlock()
		// once successfully loaded we can check if we need to clean-up after external actions
		// this is only needed when loading a keyspace that wasn't previously loaded
		// and migration has completed (which populates system collection)
		if runCleanup && datastore.IsMigrationComplete(datastore.HAS_SYSTEM_COLLECTION) {
			go CleanupSystemCollection(p.name, name)
		}
		return k, nil
	}
}

// compare the list of node addresses
// Assumption: the list of node addresses in each list are sorted
func compareNodeAddress(a, b []string) bool {

	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (p *namespace) KeyspaceById(id string) (datastore.Keyspace, errors.Error) {
	rv, err := p.keyspaceByName(id)
	if rv == nil {
		// this returns a detectable nil; if a nil rv (pointer) is returned directly, the result is a non-nil interface type
		return nil, err
	}
	return rv, err
}

// namespace implements KeyspaceMetadata
func (p *namespace) MetadataVersion() uint64 {
	return p.version
}

// ditto
func (p *namespace) MetadataId() string {
	return p.name
}

func (p *namespace) BucketIds() ([]string, errors.Error) {
	return p.KeyspaceIds()
}

func (p *namespace) BucketNames() ([]string, errors.Error) {
	return p.KeyspaceNames()
}

func (p *namespace) BucketById(name string) (datastore.Bucket, errors.Error) {
	rv, err := p.keyspaceByName(name)
	if rv == nil {
		// this returns a detectable nil; if a nil rv (pointer) is returned directly, the result is a non-nil interface type
		return nil, err
	}
	return rv, err
}

func (p *namespace) BucketByName(name string) (datastore.Bucket, errors.Error) {
	rv, err := p.keyspaceByName(name)
	if rv == nil {
		// this returns a detectable nil; if a nil rv (pointer) is returned directly, the result is a non-nil interface type
		return nil, err
	}
	return rv, err
}

func (p *namespace) getPool() *cb.Pool {
	p.nslock.RLock()
	defer p.nslock.RUnlock()
	return p.cbNamespace
}

func (p *namespace) refresh() {
	if util.Since(p.last) < _NAMESPACE_REFRESH_THRESHOLD {
		return
	}
	logging.Debugf("Refreshing pool %s", p.name)
	p.refreshFully()
}

func (p *namespace) refreshFully() {

	// trigger refresh of this pool
	newpool, err := p.store.client.GetPool(p.name)
	if err != nil {
		newpool, err = p.reload1(err)
		if err == nil {
			p.reload2(&newpool, nil)
			p.last = util.Now()
		}
		return
	}

	// MB-36458 do not switch pools as checks are being made
	p.nslock.RLock()
	oldpool := p.cbNamespace
	changed := len(oldpool.BucketMap) != len(newpool.BucketMap)
	if !changed {
		for on, ob := range oldpool.BucketMap {
			nb := newpool.BucketMap[on]
			if nb != nil && ob != nil && nb.UUID == ob.UUID {
				continue
			}
			changed = true
			break
		}
	}
	p.nslock.RUnlock()
	if changed {
		p.reload2(&newpool, nil)
		p.last = util.Now()
		return
	}
	newpool.Close()

	p.lock.Lock()
	for _, ks := range p.keyspaceCache {

		// in case a change has kicked in in between checking bucketMaps and acquiring the lock
		if ks.cbKeyspace == nil {
			continue
		}

		// Not deleted. Check if GSI indexer is available
		if ks.cbKeyspace.gsiIndexer == nil {
			ks.cbKeyspace.refreshGSIIndexer(p.store.URL(), p.Name())
		}

		// Not deleted. Check if FTS indexer is available
		if ks.cbKeyspace.ftsIndexer == nil {
			ks.cbKeyspace.refreshFTSIndexer(p.store.URL(), p.Name())
		}
	}
	p.lock.Unlock()
	p.last = util.Now()
}

func (p *namespace) reload(version *uint64) {
	logging.Debugf("[%p] Reload '%s'", p, p.name)

	newpool, err := p.store.client.GetPool(p.name)
	if err != nil {
		newpool, err = p.reload1(err)
		if err != nil {
			return
		}
	}
	p.reload2(&newpool, version)
}

func (p *namespace) reload1(err error) (cb.Pool, error) {
	var client cb.Client

	logging.Errorf("Error updating pool name <ud>'%s'</ud>: %v", p.name, err)
	url := p.store.URL()

	/*
		transport := cbauth.WrapHTTPTransport(cb.HTTPTransport, nil)
		cb.HTTPClient.Transport = transport
	*/

	if p.store.CbAuthInit == true {
		client, err = cb.ConnectWithAuth(url, cbauth.NewAuthHandler(nil), cb.USER_AGENT)
	} else {
		client, err = cb.Connect(url, cb.USER_AGENT)
	}
	if err != nil {
		logging.Errorf("Error connecting to URL %s: %v", url, err)
		return cb.Pool{}, err
	}
	// check if the default pool exists
	newpool, err := client.GetPool(p.name)
	if err != nil {
		logging.Errorf("Retry Failed Error updating pool name <ud>%s</ud>: %v", p.name, err)
		return newpool, err
	}
	p.store.client = client

	err = p.store.SetClientConnectionSecurityConfig()
	if err != nil {
		return newpool, err
	}

	return newpool, nil
}

func (p *namespace) reload2(newpool *cb.Pool, version *uint64) {
	p.lock.Lock()
	for name, ks := range p.keyspaceCache {
		logging.Debugf("Checking keyspace '%s'", name)
		if ks.cbKeyspace == nil {
			if util.Since(ks.lastUse) > _CLEANUP_INTERVAL {
				delete(p.keyspaceCache, name)
			}
			continue
		}
		newbucket, err := newpool.GetBucket(name)
		if err != nil {
			ks.cbKeyspace.Release(true)
			logging.Errorf("Error retrieving bucket '%s': %v", name, err)
			delete(p.keyspaceCache, name)
		} else if ks.cbKeyspace.cbbucket.UUID != newbucket.UUID {
			logging.Debugf("UUid of keyspace %v uuid now %v", ks.cbKeyspace.cbbucket.UUID, newbucket.UUID)
			// UUID has changed. Update the keyspace struct with the newbucket
			// and release old one
			ks.cbKeyspace.cbbucket.Close()
			ks.cbKeyspace.cbbucket = newbucket
			if !newbucket.RunBucketUpdater2(p.KeyspaceUpdateCallback, p.KeyspaceDeleteCallback) {
				logging.Errorf("[%p] Bucket '%s' is closed.", newbucket, newbucket.Name)
				delete(p.keyspaceCache, name)
			}
		} else {
			// we are reloading, so close old and set new bucket
			ks.cbKeyspace.cbbucket.Close()
			ks.cbKeyspace.cbbucket = newbucket
			if !newbucket.RunBucketUpdater2(p.KeyspaceUpdateCallback, p.KeyspaceDeleteCallback) {
				logging.Errorf("[%p] Bucket '%s' is closed.", newbucket, newbucket.Name)
				delete(p.keyspaceCache, name)
			}
		}

		// Not deleted. Check if GSI indexer is available
		if ks.cbKeyspace.gsiIndexer == nil {
			ks.cbKeyspace.refreshGSIIndexer(p.store.URL(), p.Name())
		}

		// Not deleted. Check if FTS indexer is available
		if ks.cbKeyspace.ftsIndexer == nil {
			ks.cbKeyspace.refreshFTSIndexer(p.store.URL(), p.Name())
		}
	}
	// keyspaces have been reloaded, force full auto reprepare check
	p.version++ // this must be done under the lock so we can detect the refresh in the keyspace addition code
	if version != nil {
		*version = p.version
	}
	p.lock.Unlock()

	// MB-36458 switch pool and...
	p.nslock.Lock()
	oldPool := p.cbNamespace
	p.cbNamespace = newpool
	p.nslock.Unlock()

	// ...MB-33185 let go of old pool when noone is accessing it
	oldPool.Close()
}

const (
	_DELETED        = 1 << iota // this bucket no longer exists
	_NEEDS_REFRESH              // received error that indicates the bucket needs refreshing
	_NEEDS_MANIFEST             // scopes or collections changed
)

const _INVALID_MANIFEST_UID = math.MaxUint64

type keyspace struct {
	sync.RWMutex   // to change flags and manifests in flight
	namespace      *namespace
	name           string
	fullName       string
	uidString      string
	cbbucket       *cb.Bucket
	agentProvider  *gcagent.AgentProvider
	flags          int
	gsiIndexer     datastore.Indexer // GSI index provider
	ftsIndexer     datastore.Indexer // FTS index provider
	ssIndexer      datastore.Indexer // sequential scan provider
	chkIndex       chkIndexDict
	indexersLoaded bool

	collectionsManifestUid    uint64            // current manifest id
	newCollectionsManifestUid uint64            // announced manifest id
	scopes                    map[string]*scope // scopes by id
	defaultCollection         datastore.Keyspace
	last                      util.Time // last refresh
}

var _NO_SCOPES map[string]*scope = map[string]*scope{}

func newKeyspace(p *namespace, name string, version *uint64) (*keyspace, errors.Error) {

	cbNamespace := p.getPool()
	cbbucket, err := cbNamespace.GetBucket(name)

	if err != nil {
		if !strings.Contains(err.Error(), "HTTP error 404") {
			logging.Infof("Bucket %s not found: %v", name, err)
		} else {
			logging.Debugf("Bucket %s not found", name)
		}
		// connect and check if the bucket exists
		if !cbNamespace.BucketExists(name) {
			return nil, errors.NewCbKeyspaceNotFoundError(err, fullName(p.name, name))
		}
		// it does, so we just need to refresh the primitives cache
		p.reload(version)
		cbNamespace = p.getPool()

		// and then check one more time
		logging.Infof("Retrying bucket %s", name)
		cbbucket, err = cbNamespace.GetBucket(name)
		if err != nil {
			// really no such bucket exists
			return nil, errors.NewCbKeyspaceNotFoundError(err, fullName(p.name, name))
		}
	}

	if strings.EqualFold(cbbucket.Type, "memcached") {
		return nil, errors.NewCbBucketTypeNotSupportedError(nil, cbbucket.Type)
	}

	connSecConfig := p.store.connSecConfig
	if connSecConfig == nil {
		return nil, errors.NewCbSecurityConfigNotProvided(fullName(p.name, name))
	}

	if cbbucket.IsClosed() {
		return nil, errors.NewCbBucketClosedError(fullName(p.name, name))
	}

	rv := &keyspace{
		namespace: p,
		name:      name,
		fullName:  p.Name() + ":" + name,
		uidString: cbbucket.UUID,
		cbbucket:  cbbucket,
	}

	rv.scopes = _NO_SCOPES
	mani, err := cbbucket.GetCollectionsManifest()
	if err == nil {
		rv.collectionsManifestUid = mani.Uid
		rv.scopes, rv.defaultCollection = buildScopesAndCollections(mani, rv)
		logging.Infof("Loaded manifest for bucket %v id %v", name, mani.Uid)
	} else {
		level := logging.INFO
		// if we're a bit early and the data service is still starting up we may key a KEY_NOENT result here
		// don't write this to the log so that we don't cause any alarm
		if strings.Contains(err.Error(), "status=KEY_ENOENT, opcode=0x89") {
			level = logging.DEBUG
		}
		logging.Logf(level, "Unable to retrieve collections info for bucket %s: %v", name, err)
		// set collectionsManifestUid to _INVALID_MANIFEST_UID such that if collection becomes
		// available (e.g. after legacy node is removed from cluster during rolling upgrade)
		// it'll trigger a refresh of collection manifest
		rv.collectionsManifestUid = _INVALID_MANIFEST_UID
	}

	// if we don't have any scope (not even default) revert to old style keyspace
	if len(rv.scopes) == 0 {
		rv.defaultCollection = rv
	}

	// Create a bucket updater that will keep the couchbase bucket fresh.
	if !cbbucket.RunBucketUpdater2(p.KeyspaceUpdateCallback, p.KeyspaceDeleteCallback) {
		return nil, errors.NewCbBucketClosedError(fullName(p.name, name))
	} else {
		logging.Infof("Loaded bucket %s (%s)", name, cbbucket.GetAbbreviatedUUID())
	}

	return rv, nil
}

// Called by primitives/couchbase if a configured keyspace is deleted
func (p *namespace) KeyspaceDeleteCallback(name string, err error) {

	var cbKeyspace *keyspace

	p.lock.Lock()

	ks, ok := p.keyspaceCache[name]
	if ok && ks.cbKeyspace != nil {
		logging.Infof("Keyspace %v is being deleted", name)
		cbKeyspace = ks.cbKeyspace
		ks.cbKeyspace.Release(true)
		delete(p.keyspaceCache, name)

		// keyspace has been deleted, force full auto reprepare check
		p.version++
	} else {
		logging.Warnf("Keyspace %v not configured on this server", name)
	}

	p.lock.Unlock()

	if cbKeyspace != nil {
		if isSysBucket(cbKeyspace.name) {
			DropDictionaryCache()
		} else {
			// dropDictCacheEntries() needs to be called outside p.lock
			// since it'll need to lock it when trying to delete from
			// system collection
			dropDictCacheEntries(cbKeyspace)
		}
	}
}

// Called by primitives/couchbase if a configured keyspace is updated
func (p *namespace) KeyspaceUpdateCallback(bucket *cb.Bucket, msgPrefix string) bool {

	ret := true
	checkSysBucket := false

	p.lock.Lock()

	ks, ok := p.keyspaceCache[bucket.Name]
	if ok && ks.cbKeyspace != nil {
		ks.cbKeyspace.Lock()
		uid, _ := strconv.ParseUint(bucket.CollectionsManifestUid, 16, 64)
		if ks.cbKeyspace.collectionsManifestUid != uid {
			if ks.cbKeyspace.collectionsManifestUid == _INVALID_MANIFEST_UID {
				logging.Infof("%s received manifest id %v", msgPrefix, uid)
			} else {
				logging.Infof("%s switching manifest id from %v to %v", msgPrefix, ks.cbKeyspace.collectionsManifestUid, uid)
			}
			ks.cbKeyspace.flags |= _NEEDS_MANIFEST
			ks.cbKeyspace.newCollectionsManifestUid = uid
			if isSysBucket(ks.cbKeyspace.name) {
				checkSysBucket = true
			}
		}

		var missingCapabilities map[string]datastore.Migration
		if len(ks.cbKeyspace.cbbucket.Capabilities) != len(bucket.Capabilities) {
			missingCapabilities = make(map[string]datastore.Migration)
			for _, n := range bucket.Capabilities {
				c, ok := bucketCapabilities[n]
				if ok {
					missingCapabilities[n] = c
				}
			}
			for _, o := range ks.cbKeyspace.cbbucket.Capabilities {
				delete(missingCapabilities, o)
			}
			if len(missingCapabilities) != 0 {
				ks.cbKeyspace.flags |= _NEEDS_REFRESH
			}
		}

		// the KV nodes list has changed, force a refresh on next use
		if ks.cbKeyspace.cbbucket.ChangedVBServerMap(&bucket.VBSMJson) {
			logging.Infof("%s vbMap changed", msgPrefix)
			ks.cbKeyspace.flags |= _NEEDS_REFRESH

			// bucket will be reloaded, we don't need an updater anymore
			ret = false
		}
		ks.cbKeyspace.Unlock()

		// if the bucket capability appears, the bucket is *ready* to be migrated
		// also, for a new bucket, the expectation here is that it will come alive
		// with all the capabilities correctly set, so that it won't be migrated
		for cn, c := range missingCapabilities {
			if !isSysBucket(bucket.Name) || (c != datastore.HAS_SYSTEM_COLLECTION) {
				logging.Infof("%s Starting migration to %v for bucket %s", msgPrefix, cn, bucket.Name)
				go datastore.ExecuteMigrators(bucket.Name, c)
			}
		}
	} else {
		logging.Warnf("Keyspace %v not configured on this server", bucket.Name)
	}

	p.lock.Unlock()

	if checkSysBucket {
		chkSysBucket()
	}

	return ret
}

func (b *keyspace) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.cbbucket)
}

func (b *keyspace) GetIOStats(reset bool, all bool, prometheus bool, serverless bool, times bool) map[string]interface{} {
	return b.cbbucket.GetIOStats(reset, all, prometheus, serverless, times)
}

func (b *keyspace) DurabilityPossible() bool {
	return b.cbbucket.DurabilityPossible()
}

func (b *keyspace) HasCapability(m datastore.Migration) bool {
	b.Lock()
	defer b.Unlock()
	c, ok := migration2Capability[m]
	if ok {
		for _, o := range b.cbbucket.Capabilities {
			if o == c {
				return true
			}
		}
	}
	return false
}

func (b *keyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *keyspace) Namespace() datastore.Namespace {
	return b.namespace
}

func (b *keyspace) Id() string {
	return b.Name()
}

func (b *keyspace) Name() string {
	return b.name
}

func (b *keyspace) Uid() string {
	return b.uidString
}

// keyspace (as a bucket) implements KeyspaceMetadata
func (b *keyspace) MetadataVersion() uint64 {

	// this bucket doesn't exist anymore or it needs a new manifest:
	// fail any quick prepared verify, and force a full one instead
	if b.flags&(_DELETED|_NEEDS_MANIFEST) != 0 {
		return math.MaxUint64
	}
	return b.collectionsManifestUid
}

// ditto
func (b *keyspace) MetadataId() string {
	return b.fullName
}

func (b *keyspace) QualifiedName() string {
	return b.fullName + _DEFAULT_SCOPE_COLLECTION_NAME
}

func (b *keyspace) AuthKey() string {
	return b.name
}

func (b *keyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	return b.count(context)
}

func (b *keyspace) needsTimeRefresh(threshold time.Duration) bool {
	now := util.Now()
	if now.Sub(b.last) < threshold {
		return false
	}
	b.Lock()
	b.last = now
	b.Unlock()
	return true
}

var ds2cb = []cb.BucketStats{
	cb.StatCount,
	cb.StatSize,
	cb.StatMemSize,
}

func (b *keyspace) Stats(context datastore.QueryContext, which []datastore.KeyspaceStats) ([]int64, errors.Error) {
	return b.stats(context, which)
}

func (b *keyspace) stats(context datastore.QueryContext, which []datastore.KeyspaceStats,
	clientContext ...*memcached.ClientContext) ([]int64, errors.Error) {

	cbWhich := make([]cb.BucketStats, len(which))
	for i, f := range which {
		cbWhich[i] = ds2cb[f]
	}
	res, err := b.cbbucket.GetIntStats(b.needsTimeRefresh(_STATS_REFRESH_THRESHOLD), cbWhich, clientContext...)
	if err != nil {
		b.checkRefresh(err)
		return nil, errors.NewCbKeyspaceCountError(err, b.fullName)
	}
	return res, nil
}

func (b *keyspace) count(context datastore.QueryContext, clientContext ...*memcached.ClientContext) (int64, errors.Error) {
	count, err := b.cbbucket.GetIntStats(b.needsTimeRefresh(_STATS_REFRESH_THRESHOLD), []cb.BucketStats{cb.StatCount},
		clientContext...)
	if err != nil {
		b.checkRefresh(err)
		return 0, errors.NewCbKeyspaceCountError(err, b.fullName)
	}
	return count[0], nil
}

func (b *keyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return b.size(context)
}

func (b *keyspace) size(context datastore.QueryContext, clientContext ...*memcached.ClientContext) (int64, errors.Error) {
	size, err := b.cbbucket.GetIntStats(b.needsTimeRefresh(_STATS_REFRESH_THRESHOLD), []cb.BucketStats{cb.StatSize},
		clientContext...)
	if err != nil {
		b.checkRefresh(err)
		return 0, errors.NewCbKeyspaceSizeError(err, b.fullName)
	}
	return size[0], nil
}

func (b *keyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	b.loadIndexes()
	switch name {
	case datastore.GSI, datastore.DEFAULT:
		if b.gsiIndexer != nil {
			return b.gsiIndexer, nil
		}
		return nil, errors.NewCbIndexerNotImplementedError(nil, fmt.Sprintf("GSI may not be enabled"))
	case datastore.FTS:
		if b.ftsIndexer != nil {
			return b.ftsIndexer, nil
		}
		return nil, errors.NewCbIndexerNotImplementedError(nil, fmt.Sprintf("FTS may not be enabled"))
	case datastore.SEQ_SCAN:
		if b.ssIndexer != nil {
			return b.ssIndexer, nil
		}
		return nil, errors.NewCbIndexerNotImplementedError(nil, fmt.Sprintf("Sequential scans may not be enabled"))
	default:
		return nil, errors.NewCbIndexerNotImplementedError(nil, fmt.Sprintf("Type %s", name))
	}
}

func (b *keyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	b.loadIndexes()
	indexers := make([]datastore.Indexer, 0, 3)
	var err errors.Error
	if b.gsiIndexer != nil {
		indexers = append(indexers, b.gsiIndexer)
		err = checkIndexCache(b.QualifiedName(), b.gsiIndexer, &b.chkIndex)
	}

	if b.ftsIndexer != nil {
		indexers = append(indexers, b.ftsIndexer)
	}

	if b.ssIndexer != nil {
		indexers = append(indexers, b.ssIndexer)
	}

	return indexers, err
}

// return a document key free from collection ids
func key(k []byte, clientContext ...*memcached.ClientContext) []byte {
	if len(clientContext) == 0 || len(k) <= 1 {
		return k
	}

	i := 1
	collId := clientContext[0].CollId
	for collId >= 0x80 {
		collId >>= 7
		i++
	}
	if i >= len(k) {
		return []byte("")
	}
	return k[i:]
}

//
// Inferring schemas sometimes requires getting a sample of random documents
// from a keyspace. Ideally this should come through a random traversal of the
// primary index, but until that is available, we need to use the Bucket's
// connection pool of memcached.Clients to request random documents from
// the KV store.
//

func (k *keyspace) GetRandomEntry(xattrs bool, context datastore.QueryContext) (string, value.Value, errors.Error) {
	return k.getRandomEntry(context, "", "", xattrs)
}

func (k *keyspace) getRandomEntry(context datastore.QueryContext, scopeName, collectionName string,
	xattrs bool, clientContext ...*memcached.ClientContext) (string, value.Value, errors.Error) {
	resp, err := k.cbbucket.GetRandomDoc(xattrs, clientContext...)

	if err != nil {
		k.checkRefresh(err)

		// Ignore "Not found" errors
		if isNotFoundError(err) {
			return "", nil, nil
		}
		return "", nil, errors.NewCbGetRandomEntryError(err)
	}
	if len(resp.Key) == 0 {
		logging.Warnf("%v: empty random document key detected", k.name)
		return "", nil, nil
	}
	key := string(key(resp.Key, clientContext...))
	if key == "" {
		logging.Warnf("%v: empty random document key (processed) detected", k.name)
		return "", nil, nil
	}
	doc := doFetch(key, k.fullName, resp, context, nil, xattrs)

	return key, doc, nil
}

func (b *keyspace) Fetch(keys []string, fetchMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string, projection []string, useSubDoc bool) errors.Errors {
	return b.fetch(b.fullName, b.QualifiedName(), "", "", keys, fetchMap, context, subPaths, projection, useSubDoc)
}

func (b *keyspace) fetch(fullName, qualifiedName, scopeName, collectionName string, keys []string,
	fetchMap map[string]value.AnnotatedValue, context datastore.QueryContext, subPaths []string,
	projection []string, useSubDoc bool, clientContext ...*memcached.ClientContext) errors.Errors {

	if txContext, _ := context.GetTxContext().(*transactions.TranContext); txContext != nil {
		collId, user := getCollectionId(clientContext...)
		return b.txFetch(fullName, qualifiedName, scopeName, collectionName, user, collId,
			keys, fetchMap, context, subPaths, false, txContext)
	}

	var noVirtualDocAttr bool
	var bulkResponse map[string]*gomemcached.MCResponse
	var mcr *gomemcached.MCResponse
	var err error

	l := len(keys)
	if l == 0 {
		return nil
	}

	ls := len(subPaths)
	fast := l == 1 && ls == 0
	if fast {
		if !useSubDoc || len(projection) == 0 {
			mcr, err = b.cbbucket.GetsMC(keys[0], context.IsActive, context.GetReqDeadline(), context.KvTimeout(),
				context.UseReplica(), clientContext...)
			useSubDoc = false
		} else {
			mcr, err = b.cbbucket.GetsSubDoc(keys[0], context.GetReqDeadline(), context.KvTimeout(), projection,
				append(clientContext, &memcached.ClientContext{DocumentSubDocPaths: true})...)
		}
	} else {
		if ls > 0 && ls < 15 && (subPaths[0] != "$document" && subPaths[0] != "$document.exptime") {
			subPaths = append([]string{"$document.exptime"}, subPaths...)
			noVirtualDocAttr = true
		}

		if l == 1 {
			mcr, err = b.cbbucket.GetsSubDoc(keys[0], context.GetReqDeadline(), context.KvTimeout(), subPaths, clientContext...)
		} else {
			// TODO TENANT handle refunds on transient failures
			if useSubDoc && ls == 0 && len(projection) > 0 {
				bulkResponse, err = b.cbbucket.GetBulk(keys, context.IsActive, context.GetReqDeadline(), context.KvTimeout(),
					projection, context.UseReplica(), append(clientContext, &memcached.ClientContext{DocumentSubDocPaths: true})...)
			} else {
				bulkResponse, err = b.cbbucket.GetBulk(keys, context.IsActive, context.GetReqDeadline(), context.KvTimeout(),
					subPaths, context.UseReplica(), clientContext...)
				useSubDoc = false
			}
			defer b.cbbucket.ReleaseGetBulkPools(bulkResponse)
		}
	}

	if err != nil {
		b.checkRefresh(err)

		// Ignore "Not found" keys
		if !isNotFoundError(err) {
			if cb.IsReadTimeOutError(err) {
				logging.Errorf(err.Error())
			}

			_, err = processIfMCError(errors.FALSE, err, keys[0], qualifiedName)
			if cb.IsBucketNotFound(err) {
				return []errors.Error{errors.NewCbBulkGetError(err, "", errors.FALSE)}
			} else {
				return []errors.Error{errors.NewCbBulkGetError(err, "", errors.TRUE)}
			}
		}
	}

	if fast {
		if mcr != nil && err == nil {
			if !useSubDoc {
				fetchMap[keys[0]] = doFetch(keys[0], fullName, mcr, context, projection, false)
			} else {
				fetchMap[keys[0]] = getSubDocFetchResults(keys[0], fullName, mcr, projection, context)
			}
		}

	} else if l == 1 {
		if mcr != nil && err == nil {
			fetchMap[keys[0]] = getSubDocXattrFetchResults(keys[0], fullName, mcr, subPaths, noVirtualDocAttr, context)
		}
	} else {
		i := 0
		if ls > 0 {
			for k, v := range bulkResponse {
				fetchMap[k] = getSubDocXattrFetchResults(k, fullName, v, subPaths, noVirtualDocAttr, context)
				i++
			}
		} else if useSubDoc {
			for k, v := range bulkResponse {
				fetchMap[k] = getSubDocFetchResults(k, fullName, v, projection, context)
				i++
			}
			logging.Debugf("(Sub-doc) Requested keys %d Fetched %d keys ", l, i)
		} else {
			for k, v := range bulkResponse {
				fetchMap[k] = doFetch(k, fullName, v, context, projection, false)
				i++
			}
			logging.Debugf("Requested keys %d Fetched %d keys ", l, i)
		}
	}

	return nil
}

func doFetch(k string, fullName string, v *gomemcached.MCResponse, context datastore.QueryContext,
	projection []string, xattrs bool) value.AnnotatedValue {

	var val value.AnnotatedValue
	var raw []byte
	var flags, expiration uint32
	var xattrVal value.Value

	if v.DataType&gomemcached.DatatypeFlagCompressed == 0 {
		raw = v.Body
	} else {
		// Uncomment when needed
		//context.Debugf("Compressed document: %v", k)
		var err error
		raw, err = snappy.Decode(nil, v.Body)
		if err != nil {
			context.Error(errors.NewInvalidCompressedValueError(err, v.Body))
			logging.Severef("Invalid compressed document received: %v - %v", err, v, context)
			return nil
		}
	}
	if xattrs && raw[0] != '{' {
		var ok bool
		raw, xattrVal, ok = cb.ExtractXattrs(raw)
		if !ok {
			logging.Warnf("[%s] Invalid XATTRs", k)
		}
	}
	if len(projection) == 0 {
		val = value.NewAnnotatedValue(value.NewParsedValue(raw, (v.DataType&gomemcached.DatatypeFlagJSON != 0)))
	} else {
		val = value.NewAnnotatedValue(value.NewNestedScopeValue(nil))
		var scan json.ScanState
		json.SetScanState(&scan, raw)
	proj:
		for found := 0; found < len(projection); {
			k, e := scan.ScanKeys()
			if e != nil || k == nil {
				break
			}
			field := util.ByteToString(k)
			for i := 0; i < len(projection); i++ {
				if projection[i] == field {
					v1, e := scan.NextValue()
					if e != nil {
						break proj
					}
					val.SetField(projection[i], value.NewParsedValue(v1, (v.DataType&gomemcached.DatatypeFlagJSON != 0)))
					found++
					break
				}
			}
		}
		scan.Release()
		raw = nil
	}

	if len(v.Extras) >= 8 {
		flags = binary.BigEndian.Uint32(v.Extras[0:4])
		expiration = binary.BigEndian.Uint32(v.Extras[4:8])
	} else if len(v.Extras) >= 4 {
		flags = binary.BigEndian.Uint32(v.Extras[0:4])
	}

	val.SetMetaField(value.META_KEYSPACE, fullName)
	val.SetMetaField(value.META_CAS, v.Cas)
	if val.Type() == value.BINARY {
		val.SetMetaField(value.META_TYPE, "base64")
	} else {
		val.SetMetaField(value.META_TYPE, "json")
	}
	val.SetMetaField(value.META_FLAGS, flags)
	val.SetMetaField(value.META_EXPIRATION, expiration)
	if xattrVal != nil {
		val.SetMetaField(value.META_XATTRS, xattrVal)
	}
	val.SetId(k)

	if tenant.IsServerless() {
		ru, _ := v.ComputeUnits()
		context.RecordKvRU(tenant.Unit(ru))
	}

	// Uncomment when needed
	//logging.Debugf("CAS Value for key %v is %v flags %v", k, uint64(v.Cas), meta_flags)

	return val
}

func getSubDocXattrFetchResults(k string, fullName string, v *gomemcached.MCResponse, subPaths []string, noVirtualDocAttr bool,
	context datastore.QueryContext) value.AnnotatedValue {

	responseIter := 0
	i := 0
	xVal := map[string]interface{}{}

	for i < len(subPaths) {
		// For the xattr contents - $document
		xattrError := gomemcached.Status(binary.BigEndian.Uint16(v.Body[responseIter+0:]))
		xattrValueLen := int(binary.BigEndian.Uint32(v.Body[responseIter+2:]))

		xattrValue := v.Body[responseIter+6 : responseIter+6+xattrValueLen]

		// When xattr value not defined for a doc, set missing
		tmpVal := value.NewValue(xattrValue)

		if xattrError != gomemcached.SUBDOC_PATH_NOT_FOUND {
			xVal[subPaths[i]] = tmpVal.Actual()
		}

		// Calculate actual doc value
		responseIter = responseIter + 6 + xattrValueLen
		i = i + 1
	}

	// For the actual document contents -
	respError := gomemcached.Status(binary.BigEndian.Uint16(v.Body[responseIter+0:]))
	respValueLen := int(binary.BigEndian.Uint32(v.Body[responseIter+2:]))

	respValue := v.Body[responseIter+6 : responseIter+6+respValueLen]

	// For deleted documents with respError path not found set to null
	var val value.AnnotatedValue

	// For non deleted documents
	if respError == gomemcached.SUBDOC_PATH_NOT_FOUND {
		// Final Doc value
		val = value.NewAnnotatedValue(nil)
	} else {
		// Sub-doc API does not use response header's data type.  It decompresses the document before it is sent/received.
		// (The xattr datatype array will show "snappy" but the data pointed to by respValue here is not compressed.)
		val = value.NewAnnotatedValue(value.NewParsedValue(respValue, false))
	}

	// type
	meta_type := "json"
	if val.Type() == value.BINARY {
		meta_type = "base64"
	}

	var flags, exptime uint32

	if subPaths[0] == "$document" {
		// Get flags and expiration from the $document virtual xattrs
		x, ok := xVal["$document"]
		if !ok {
			logging.Warnf("[%s] Invalid XATTRs: $document not found", k)
		} else {
			docMeta, ok := x.(map[string]interface{})
			if !ok {
				logging.Warnf("[%s] Invalid XATTR $document: %T (%v)", k, x, x)
			} else {
				// Convert unmarshalled int64 values to uint32
				v := value.NewValue(docMeta["flags"])
				if v.Type() != value.NUMBER {
					logging.Warnf("[%s] Invalid XATTR $document.flags: %v (%v)", k, v.Type(), v.String())
				} else {
					flags = uint32(value.AsNumberValue(v).Int64())
				}
				v = value.NewValue(docMeta["exptime"])
				if v.Type() != value.NUMBER {
					logging.Warnf("[%s] Invalid XATTR $document.exptime: %v (%v)", k, v.Type(), v.String())
				} else {
					exptime = uint32(value.AsNumberValue(v).Int64())
				}
			}
		}
	} else if subPaths[0] == "$document.exptime" {
		x, ok := xVal["$document.exptime"]
		if !ok {
			logging.Warnf("[%s] Invalid XATTRs: $document.exptime not found", k)
		} else {
			v := value.NewValue(x)
			if v.Type() != value.NUMBER {
				logging.Warnf("[%s] Invalid XATTR $document.exptime: %v (%v)", k, v.Type(), v.String())
			} else {
				exptime = uint32(value.AsNumberValue(v).Int64())
			}
		}
	}

	if noVirtualDocAttr {
		delete(xVal, "$document.exptime")
	}

	val.SetMetaField(value.META_KEYSPACE, fullName)
	val.SetMetaField(value.META_CAS, v.Cas)
	val.SetMetaField(value.META_TYPE, meta_type)
	val.SetMetaField(value.META_FLAGS, flags)
	val.SetMetaField(value.META_EXPIRATION, exptime)
	if len(xVal) > 0 {
		val.SetMetaField(value.META_XATTRS, xVal)
	}
	val.SetId(k)

	if tenant.IsServerless() {
		ru, _ := v.ComputeUnits()
		context.RecordKvRU(tenant.Unit(ru))
	}

	return val
}

func getSubDocFetchResults(k string, fullName string, v *gomemcached.MCResponse, projection []string,
	context datastore.QueryContext) value.AnnotatedValue {

	val := value.NewAnnotatedValue(value.NewNestedScopeValue(nil))
	index := 0
	for off := 0; off < len(v.Body)-6 && index < len(projection); index++ {
		status := gomemcached.Status(binary.BigEndian.Uint16(v.Body[off:]))
		off += 2
		rawLen := int(binary.BigEndian.Uint32(v.Body[off:]))
		off += 4
		raw := v.Body[off : off+rawLen]
		off += rawLen
		if status != gomemcached.SUBDOC_PATH_NOT_FOUND && index < len(projection) {
			val.SetField(projection[index], value.NewValue(raw))
		}
	}

	val.SetMetaField(value.META_KEYSPACE, fullName)
	val.SetMetaField(value.META_CAS, v.Cas)
	val.SetMetaField(value.META_TYPE, "json")
	val.SetMetaField(value.META_FLAGS, uint32(0))
	val.SetMetaField(value.META_EXPIRATION, uint32(0))
	val.SetId(k)

	if tenant.IsServerless() {
		ru, _ := v.ComputeUnits()
		context.RecordKvRU(tenant.Unit(ru))
	}

	return val
}

func (k *keyspace) checkRefresh(err error) {
	if cb.IsRefreshRequired(err) {
		k.Lock()
		k.flags |= _NEEDS_REFRESH
		k.Unlock()
		k.cbbucket.StopUpdater()
	} else if cb.IsUnknownCollection(err) {
		k.Lock()
		k.flags |= _NEEDS_MANIFEST
		k.Unlock()
	}
}

func (k *keyspace) setNeedsManifest() {
	k.Lock()
	k.flags |= _NEEDS_MANIFEST
	k.Unlock()
}

func isNotFoundError(err error) bool {
	if cb.IsKeyNoEntError(err) {
		return true
	}
	// it may have been wrapped in another error so check the text...
	if ee, ok := err.(errors.Error); ok {
		return ee.ContainsText("KEY_ENOENT")
	}
	return strings.Contains(err.Error(), "KEY_ENOENT")
}

func isEExistError(err error) bool {
	if cb.IsKeyEExistsError(err) {
		return true
	}
	// it may have been wrapped in another error so check the text...
	if ee, ok := err.(errors.Error); ok {
		return ee.ContainsText("KEY_EEXISTS")
	}
	return strings.Contains(err.Error(), "KEY_EEXISTS")
}

func getMeta(key string, val value.Value, must bool) (cas uint64, flags uint32, txnMeta interface{}, err error) {

	var av value.AnnotatedValue
	var ok bool
	var mv interface{}

	if av, ok = val.(value.AnnotatedValue); !ok || av == nil {
		if must {
			return 0, 0, nil, fmt.Errorf("Invalid value type (%T) or nil value for key %v", val, key)
		}
		return cas, flags, txnMeta, nil
	}

	if mv = av.GetMetaField(value.META_CAS); mv != nil {
		cas, _ = mv.(uint64)
	}

	if must && mv == nil {
		return 0, 0, nil, fmt.Errorf("Not valid Cas value for key %v", key)
	}

	if mv = av.GetMetaField(value.META_FLAGS); mv != nil {
		flags, _ = mv.(uint32)
	}

	if must && mv == nil {
		return 0, 0, nil, fmt.Errorf("Not valid Flags value for key %v", key)
	}

	if mv = av.GetMetaField(value.META_TXNMETA); mv != nil {
		txnMeta, _ = mv.(interface{})
	}

	return cas, flags, txnMeta, nil

}

func SetMetaCas(val value.Value, cas uint64) bool {
	if av, ok := val.(value.AnnotatedValue); ok && av != nil {
		av.SetMetaField(value.META_CAS, cas)
		return true
	}
	return false
}

func getExpiration(options value.Value) (exptime int, present bool) {
	if options != nil && options.Type() == value.OBJECT {
		if v, ok := options.Field("expiration"); ok && v.Type() == value.NUMBER {
			present = true
			exptime = int(value.AsNumberValue(v).Int64())
		}
	}
	return
}

func getMutatableXattrs(options value.Value) map[string]interface{} {
	if options != nil && options.Type() == value.OBJECT {
		if v, ok := options.Field("xattrs"); ok && v.Type() == value.OBJECT {
			// extract and return non-virtual xattrs only
			var rv map[string]interface{}
			for k, v := range v.Fields() {
				if k[0] != '$' {
					if rv == nil {
						rv = make(map[string]interface{})
					}
					rv[k] = v
				}
			}
			return rv
		}
	}
	return nil
}

func hasExpirationOrXattrs(options value.Value) bool {
	if options != nil && options.Type() == value.OBJECT {
		if v, ok := options.Field("expiration"); ok && v.Type() == value.NUMBER {
			return true
		}
		if v, ok := options.Field("xattrs"); ok && v.Type() == value.OBJECT {
			return true
		}
	}
	return false
}

// Struct with info passed to the mutation workers
type parallelInfo struct {
	sync.Mutex
	pairs  value.Pairs
	mPairs value.Pairs
	errs   errors.Errors
	wg     *sync.WaitGroup
	index  int
	mCount int // number of successfully mutated pairs
}

func (b *keyspace) performOp(op MutateOp, qualifiedName, scopeName, collectionName string, pairs value.Pairs,
	preserveMutations bool, context datastore.QueryContext, clientContext ...*memcached.ClientContext) (
	mCount int, mPairs value.Pairs, errs errors.Errors) {

	numPairs := len(pairs)

	if numPairs == 0 {
		return
	}

	if txContext, _ := context.GetTxContext().(*transactions.TranContext); txContext != nil {
		collId, user := getCollectionId(clientContext...)
		return b.txPerformOp(op, qualifiedName, scopeName, collectionName, user, collId,
			pairs, preserveMutations, context, txContext)
	}

	if err := setMutateClientContext(context, clientContext...); err != nil {
		errs = append(errs, err)
		return
	}

	numRoutines := util.MinInt(numPairs, _MAX_MUTATION_ROUTINES) // number of routines that will each perform mutations

	if numRoutines == 1 { // If numRoutines = 1, run mutations sequentially
		// number of successfully mutated pairs
		mCount = 0

		if preserveMutations {
			mPairs = make(value.Pairs, 0, numPairs)
		}

		for i := 0; i < numPairs; i++ {
			state, err := b.singleMutationOp(pairs[i], op, qualifiedName, context, clientContext...)

			if state == _MUTATED {
				mCount++
				if preserveMutations {
					mPairs = append(mPairs, pairs[i])
				}
			}

			if err != nil {
				if len(errs) == 0 {
					errs = make(errors.Errors, 0, numPairs)
				}

				errs = append(errs, err)
			}

			// if error limit was hit or a mutation resulted in a stopped state
			// stop subsequent mutations
			if state == _STOPPED || (context.ErrorLimit() > 0 && (len(errs)+context.ErrorCount()) > context.ErrorLimit()) {
				break
			}
		}

		return mCount, mPairs, errs

	} else {
		// if the number of keys to be modified is greater than 1 and the number of allowed routines > 1 , run modification
		// operations concurrently
		p := &parallelInfo{
			pairs:  pairs,
			wg:     &sync.WaitGroup{},
			index:  0,
			mCount: 0,
		}

		if preserveMutations {
			p.mPairs = make(value.Pairs, 0, numPairs)
		}

		p.wg.Add(numRoutines)

		// start the go routines that each perform mutations
		for i := 0; i < numRoutines; i++ {
			go b.parallelMutationOp(op, qualifiedName, p, preserveMutations, context, clientContext...)
		}

		p.wg.Wait()

		return p.mCount, p.mPairs, p.errs
	}
}

// Performs mutation for a single key
func (b *keyspace) singleMutationOp(kv value.Pair, op MutateOp, qualifiedName string, context datastore.QueryContext,
	clientContext ...*memcached.ClientContext) (mutationState, errors.Error) {

	retry := errors.NONE
	var err error
	var keyError errors.Error
	var wu uint64
	var val interface{}
	var exptime int
	var present bool
	var cas, newCas uint64
	var xattrs map[string]interface{}
	casMismatch := false

	// operator has been terminated
	if !context.IsActive() {
		return _STOPPED, nil
	}

	key := kv.Name
	if op != MOP_DELETE {
		if kv.Value.Type() == value.BINARY {
			return _STOPPED, errors.NewBinaryDocumentMutationError(MutateOpNames[op], key)
		}
		val = kv.Value.ActualForIndex()
		exptime, present = getExpiration(kv.Options)
		xattrs = getMutatableXattrs(kv.Options)
	}

	switch op {

	case MOP_INSERT:
		var added bool

		// add the key to the backend
		added, cas, wu, err = b.cbbucket.AddWithCAS(key, exptime, val, xattrs, clientContext...)

		context.RecordKvWU(tenant.Unit(wu))
		b.checkRefresh(err)
		if added == false {
			// false & err == nil => given key aready exists in the bucket
			if err != nil {
				retry, err = processIfMCError(retry, err, key, qualifiedName)
				err = errors.NewInsertError(err, key)
			} else {
				err = errors.NewDuplicateKeyError(key, "", nil)
				retry = errors.FALSE
			}
		} else { // if err != nil then added is false
			// refresh local meta CAS value
			logging.Debugf("After %s: key {<ud>%v</ud>} CAS %v for Keyspace <ud>%s</ud>.",
				MutateOpNames[op], key, cas, qualifiedName)
			SetMetaCas(kv.Value, cas)
		}
	case MOP_UPDATE:
		// check if the key exists and if so then use the cas value
		// to update the key
		var flags uint32

		cas, flags, _, err = getMeta(key, kv.Value, true)
		if err != nil { // Don't perform the update if the meta values are not found
			logging.Debugf("Failed to get meta value to perform %s on key <ud>%s<ud> for Keyspace <ud>%s</ud>. Error %s",
				MutateOpNames[op], key, qualifiedName, err)
			if retry == errors.NONE {
				retry = errors.TRUE
			}
		} else if err = setPreserveExpiry(present, context, clientContext...); err != nil {
			logging.Debugf("Failed to preserve the expiration to perform %s on key <ud>%s<ud> for Keyspace <ud>%s</ud>. Error %s",
				MutateOpNames[op], key, qualifiedName, err)
			if retry == errors.NONE {
				retry = errors.TRUE
			}
		} else {
			logging.Debugf("Before %s: key {<ud>%v</ud>} CAS %v flags <ud>%v</ud> value <ud>%v</ud> for Keyspace <ud>%s</ud>.",
				MutateOpNames[op], key, cas, flags, val, qualifiedName)
			newCas, wu, _, err = b.cbbucket.CasWithMeta(key, int(flags), exptime, cas, val, xattrs, clientContext...)

			context.RecordKvWU(tenant.Unit(wu))
			if err == nil {
				// refresh local meta CAS value
				logging.Debugf("After %s: key {<ud>%v</ud>} CAS %v for Keyspace <ud>%s</ud>.",
					MutateOpNames[op], key, cas, qualifiedName)
				SetMetaCas(kv.Value, newCas)
			}
			b.checkRefresh(err)
		}

	case MOP_UPSERT:
		if err = setPreserveExpiry(present, context, clientContext...); err != nil {
			logging.Debugf("Failed to preserve the expiration to perform %s on key <ud>%s<ud> for Keyspace <ud>%s</ud>. Error %s",
				MutateOpNames[op], key, qualifiedName, err)
			if retry == errors.NONE {
				retry = errors.TRUE
			}
		} else {
			newCas, wu, err = b.cbbucket.SetWithCAS(key, exptime, val, xattrs, clientContext...)

			context.RecordKvWU(tenant.Unit(wu))
			b.checkRefresh(err)
			if err == nil {
				logging.Debugf("After %s: key {<ud>%v</ud>} CAS %v for Keyspace <ud>%s</ud>.",
					MutateOpNames[op], key, cas, qualifiedName)
				SetMetaCas(kv.Value, newCas)
			}
		}
	case MOP_DELETE:
		wu, err = b.cbbucket.Delete(key, clientContext...)

		context.RecordKvWU(tenant.Unit(wu))
		b.checkRefresh(err)
	}

	if err != nil {
		msg := fmt.Sprintf("Failed to perform %s on key %s", MutateOpNames[op], key)
		if op == MOP_DELETE {
			if !isNotFoundError(err) {
				logging.Debugf("Failed to perform %s on key <ud>%s<ud> for Keyspace <ud>%s</ud>. Error %s",
					MutateOpNames[op], key, qualifiedName, err)
				retry, err = processIfMCError(retry, err, key, qualifiedName)
				keyError = errors.NewCbDeleteFailedError(err, key, msg)
			}
		} else if isEExistError(err) {
			if op != MOP_INSERT {
				casMismatch = true
				retry = errors.FALSE
			}
			if casMismatch {
				logging.Debugf("Failed to perform %s on key <ud>%s<ud> for Keyspace <ud>%s</ud>."+
					" CAS mismatch due to concurrent modifications. Error %s",
					MutateOpNames[op], key, qualifiedName, err)
			} else {
				logging.Debugf("Failed to perform %s on key <ud>%s<ud> for Keyspace <ud>%s</ud>."+
					" Concurrent modifications. Error %s",
					MutateOpNames[op], key, qualifiedName, err)
			}

			retry, err = processIfMCError(retry, err, key, qualifiedName)
			keyError = errors.NewCbDMLError(err, msg, casMismatch, retry, key, qualifiedName)
		} else if isNotFoundError(err) {
			err = errors.NewKeyNotFoundError(key, "", nil)
			retry = errors.FALSE
			keyError = errors.NewCbDMLError(err, msg, casMismatch, retry, key, qualifiedName)
		} else {
			// err contains key, redact
			logging.Debugf("Failed to perform %s on key <ud>%s<ud> for Keyspace <ud>%s</ud>. Error %s",
				MutateOpNames[op], key, qualifiedName, err)
			retry, err = processIfMCError(retry, err, key, qualifiedName)
			keyError = errors.NewCbDMLError(err, msg, casMismatch, retry, key, qualifiedName)
		}

		return _FAILED, keyError
	}

	return _MUTATED, nil
}

// Mutation worker
func (b *keyspace) parallelMutationOp(op MutateOp, qualifiedName string, p *parallelInfo, preserveMutations bool,
	context datastore.QueryContext, clientCtx ...*memcached.ClientContext) {

	defer func() {
		r := recover()
		if r != nil {
			logging.Stackf(logging.ERROR, "Mutation routine panicked. Panic: %v. Restarting mutation routine.", r)

			// restart mutation worker
			go b.parallelMutationOp(op, qualifiedName, p, preserveMutations, context, clientCtx...)
		}
	}()

	var clientContext []*memcached.ClientContext

	if len(clientCtx) > 0 {
		clientContext = append(clientContext, clientCtx[0].Copy())
	}

	state := _NONE
	var err errors.Error
	var kv value.Pair

	for {
		p.Lock()
		if err != nil {
			if p.errs == nil {
				p.errs = make(errors.Errors, 0, len(p.pairs))
			}

			p.errs = append(p.errs, err)
		}

		if state == _MUTATED {

			p.mCount++

			if preserveMutations {
				p.mPairs = append(p.mPairs, kv)
			}
		}

		// Check if subsequent mutations must be stopped - if context is inactive or when error limit is hit
		// Or if there are no more keys to mutate
		// If yes - end the mutation worker
		if !context.IsActive() || p.index >= len(p.pairs) ||
			(context.ErrorLimit() > 0 && (len(p.errs)+context.ErrorCount() > context.ErrorLimit())) {

			p.Unlock()
			break
		}

		kv = p.pairs[p.index]
		p.index++

		p.Unlock()

		state, err = b.singleMutationOp(kv, op, qualifiedName, context, clientContext...)
	}

	p.wg.Done()
}

func processIfMCError(retry errors.Tristate, err error, key string, keyspace string) (errors.Tristate, error) {
	if mcr, ok := err.(*gomemcached.MCResponse); ok {
		if gomemcached.IsFatal(mcr) {
			retry = errors.FALSE
		} else {
			retry = errors.TRUE
		}
		err = errors.NewCbDMLMCError(mcr.Status.String(), key, keyspace)
	}
	return retry, err
}

func (b *keyspace) Insert(inserts value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {

	return b.performOp(MOP_INSERT, b.QualifiedName(), "", "", inserts, preserveMutations, context)

}

func (b *keyspace) Update(updates value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {

	return b.performOp(MOP_UPDATE, b.QualifiedName(), "", "", updates, preserveMutations, context)
}

func (b *keyspace) Upsert(upserts value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {

	return b.performOp(MOP_UPSERT, b.QualifiedName(), "", "", upserts, preserveMutations, context)
}

func (b *keyspace) Delete(deletes value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {

	return b.performOp(MOP_DELETE, b.QualifiedName(), "", "", deletes, preserveMutations, context)
}

func (b *keyspace) SetSubDoc(key string, elems value.Pairs, context datastore.QueryContext) (
	value.Pairs, errors.Error) {

	cc := &memcached.ClientContext{User: getUser(context)}
	err := setMutateClientContext(context, cc)
	if err != nil {
		return nil, err
	}
	ops := make([]memcached.SubDocOp, len(elems))
	for i := range elems {
		ops[i].Path = elems[i].Name
		b, e := json.Marshal(elems[i].Value)
		if e != nil {
			return nil, errors.NewSubDocSetError(e)
		}
		ops[i].Value = b
		ops[i].Counter = (elems[i].Options != nil && elems[i].Options.Truth())
	}
	mcr, e := b.cbbucket.SetsSubDoc(key, ops, cc)
	if e != nil {
		if isNotFoundError(e) {
			return nil, errors.NewKeyNotFoundError(key, "", nil)
		}
		return nil, errors.NewSubDocSetError(e)
	}
	return processSubDocResults(ops, mcr), nil
}

func processSubDocResults(ops []memcached.SubDocOp, v *gomemcached.MCResponse) value.Pairs {
	res := make(value.Pairs, 0, len(ops))
	index := 0
	for i := 0; i < len(v.Body)-6 && index < len(ops); {
		index = int(v.Body[i])
		i++
		status := gomemcached.Status(binary.BigEndian.Uint16(v.Body[i:]))
		i += 2
		l := int(binary.BigEndian.Uint32(v.Body[i:]))
		i += 4
		val := v.Body[i : i+l]
		i += l
		if status != gomemcached.SUBDOC_PATH_NOT_FOUND && index < len(ops) {
			res = append(res, value.Pair{Name: ops[index].Path, Value: value.NewValue(val), Options: value.NewValue(index)})
		}
	}
	return res
}

func (b *keyspace) Release(bclose bool) {
	b.Lock()
	b.flags |= _DELETED
	agentProvider := b.agentProvider
	b.agentProvider = nil
	b.Unlock()
	b.cbbucket.SetDeleted()
	if bclose {
		b.cbbucket.StopUpdater()
		b.cbbucket.Close()
	}
	if agentProvider != nil {
		agentProvider.Close()
	}

	if gsiIndexer, ok := b.gsiIndexer.(interface{ Close() }); ok {
		gsiIndexer.Close()
	}
	// close an ftsIndexer that belongs to this keyspace
	if ftsIndexerCloser, ok := b.ftsIndexer.(io.Closer); ok {
		// FTSIndexer implements a Close() method
		ftsIndexerCloser.Close()
	}

	// no need to close anything for sequential scans
}

func (b *keyspace) refreshGSIIndexer(url string, poolName string) {
	var err error

	b.RLock()
	indexersLoaded := b.indexersLoaded
	b.RUnlock()
	if !indexersLoaded {
		return
	}
	b.gsiIndexer, err = gsi.NewGSIIndexer(url, poolName, b.Name(), b.namespace.store.connSecConfig)
	if err == nil {
		logging.Infof(" GSI Indexer loaded ")

		// We know the connSecConfig is present, because we checked when the keyspace was created.
		b.gsiIndexer.SetConnectionSecurityConfig(b.namespace.store.connSecConfig)
	} else {
		b.gsiIndexer = nil
		logging.Errorf(" Error while refreshing GSI indexer - %v", err)
	}
}

func (b *keyspace) refreshFTSIndexer(url string, poolName string) {
	var err error

	b.RLock()
	indexersLoaded := b.indexersLoaded
	b.RUnlock()
	if !indexersLoaded {
		return
	}
	b.ftsIndexer, err = ftsclient.NewFTSIndexer(url, poolName, b.Name())
	if err == nil {
		logging.Infof(" FTS Indexer loaded ")

		// We know the connSecConfig is present, because we checked when the keyspace was created.
		b.ftsIndexer.SetConnectionSecurityConfig(b.namespace.store.connSecConfig)
	} else {
		logging.Errorf(" Error while refreshing FTS indexer - %v", err)
	}
}

// we load indexers asynchronously because unless we are connecting to older KV's
// we're always going to use the collection indexers and we don't need the bucket
// indexes loaded
func (b *keyspace) loadIndexes() {
	var qerr errors.Error

	b.Lock()
	defer b.Unlock()

	// somebody's already done it
	if b.indexersLoaded {
		return
	}
	p := b.namespace
	store := p.store

	b.gsiIndexer, qerr = gsi.NewGSIIndexer(p.store.URL(), p.Name(), b.name, store.connSecConfig)
	if qerr != nil {
		b.gsiIndexer = nil
		logging.Warnf("Error loading GSI indexes for keyspace %s. Error %v", b.name, qerr)
	} else {
		b.gsiIndexer.SetConnectionSecurityConfig(store.connSecConfig)
	}

	b.ftsIndexer, qerr = ftsclient.NewFTSIndexer(store.URL(), p.Name(), b.name)
	if qerr != nil {
		logging.Warnf("Error loading FTS indexes for keyspace %s. Error %v", b.name, qerr)
	} else {
		b.ftsIndexer.SetConnectionSecurityConfig(store.connSecConfig)
	}

	if b.cbbucket.HasCapability(cb.RANGE_SCAN) {
		b.ssIndexer = newSeqScanIndexer(b)
		b.ssIndexer.SetConnectionSecurityConfig(store.connSecConfig)
	} else {
		b.ssIndexer = nil
	}

	b.indexersLoaded = true
}

func (b *keyspace) Scope() datastore.Scope {
	return nil
}

func (b *keyspace) ScopeId() string {
	return ""
}

func (b *keyspace) MaxTTL() int64 {
	return 0
}

func (ks *keyspace) DefaultKeyspace() (datastore.Keyspace, errors.Error) {
	switch d := ks.defaultCollection.(type) {
	case *collection:
		if d != nil {
			return ks.defaultCollection, nil
		}
	case *keyspace:

		// there are no scopes, operate in bucket mode
		return ks.defaultCollection, nil
	}
	return nil, errors.NewBucketNoDefaultCollectionError(fullName(ks.namespace.name, ks.name))
}

func (ks *keyspace) ScopeIds() ([]string, errors.Error) {
	ks.RLock()
	ids := make([]string, len(ks.scopes))
	ix := 0
	for k := range ks.scopes {
		ids[ix] = k
		ix++
	}
	ks.RUnlock()
	return ids, nil
}

func (ks *keyspace) ScopeNames() ([]string, errors.Error) {
	ks.RLock()
	ids := make([]string, len(ks.scopes))
	ix := 0
	for _, v := range ks.scopes {
		ids[ix] = v.Name()
		ix++
	}
	ks.RUnlock()
	return ids, nil
}

func (ks *keyspace) ScopeById(id string) (datastore.Scope, errors.Error) {
	ks.RLock()
	scope := ks.scopes[id]
	if scope == nil {
		ks.RUnlock()
		return nil, errors.NewCbScopeNotFoundError(nil, fullName(ks.namespace.name, ks.name, id))
	}
	ks.RUnlock()
	return scope, nil
}

func (ks *keyspace) ScopeByName(name string) (datastore.Scope, errors.Error) {
	ks.RLock()
	for _, v := range ks.scopes {
		if name == v.Name() {
			ks.RUnlock()
			return v, nil
		}
	}
	ks.RUnlock()
	return nil, errors.NewCbScopeNotFoundError(nil, fullName(ks.namespace.name, ks.name, name))
}

func (ks *keyspace) CreateScope(name string) errors.Error {
	err := ks.cbbucket.CreateScope(name)
	if err != nil {
		return errors.NewCbBucketCreateScopeError(fullName(ks.namespace.name, ks.name, name), err)
	}
	ks.setNeedsManifest()
	return nil
}

func (ks *keyspace) DropScope(name string) errors.Error {
	err := ks.cbbucket.DropScope(name)
	if err != nil {
		return errors.NewCbBucketDropScopeError(fullName(ks.namespace.name, ks.name, name), err)
	}
	ks.setNeedsManifest()

	// TODO remove
	// trigger scope refresh straight away to empty functions and dictionary caches
	time.AfterFunc(time.Second, func() { ks.namespace.keyspaceByName(ks.name) })
	return nil
}

func (ks *keyspace) Flush() errors.Error {
	return errors.NewNoFlushError(ks.name)
}

func (b *keyspace) IsBucket() bool {
	return true
}

func (b *keyspace) IsSystemCollection() bool {
	return false
}

func (ks *keyspace) getDefaultCid() (uint32, bool) {
	var cid uint32
	var ok bool
	ks.RLock()
	scope, ok := ks.scopes["_default"]
	if ok {
		coll, ok := scope.keyspaces["_default"]
		if ok {
			cid, ok = coll.uid, true
		}
	}
	ks.RUnlock()
	return cid, ok
}

func (ks *keyspace) StartKeyScan(context datastore.QueryContext, ranges []*datastore.SeqScanRange, offset int64,
	limit int64, ordered bool, timeout time.Duration, pipelineSize int, serverless bool, skipKey func(string) bool) (
	interface{}, errors.Error) {

	r := make([]*cb.SeqScanRange, len(ranges))
	for i := range ranges {
		r[i] = &cb.SeqScanRange{}
		r[i].Init(ranges[i].Start, ranges[i].ExcludeStart, ranges[i].End, ranges[i].ExcludeEnd)
	}

	if cid, ok := ks.getDefaultCid(); ok {
		return ks.cbbucket.StartKeyScan(context.RequestId(), context, cid, "", "", r, offset, limit, ordered, timeout,
			pipelineSize, serverless, context.UseReplica(), skipKey)
	}
	return ks.cbbucket.StartKeyScan(context.RequestId(), context, 0, "_default", "_default", r, offset, limit, ordered, timeout,
		pipelineSize, serverless, context.UseReplica(), skipKey)
}

func (ks *keyspace) StopScan(scan interface{}) (uint64, errors.Error) {
	return ks.cbbucket.StopScan(scan)
}

func (ks *keyspace) FetchKeys(scan interface{}, timeout time.Duration) ([]string, errors.Error, bool) {
	return ks.cbbucket.FetchKeys(scan, timeout)
}

func (ks *keyspace) FetchDocs(scan interface{}, timeout time.Duration) ([]value.AnnotatedValue, errors.Error, bool) {
	return ks.cbbucket.FetchDocs(scan, timeout)
}

func (ks *keyspace) StartRandomScan(context datastore.QueryContext, sampleSize int, timeout time.Duration,
	pipelineSize int, serverless bool, xattrs bool, withDocs bool) (interface{}, errors.Error) {

	if cid, ok := ks.getDefaultCid(); ok {
		return ks.cbbucket.StartRandomScan(context.RequestId(), context, cid, "", "", sampleSize, timeout, pipelineSize,
			serverless, context.UseReplica(), xattrs, withDocs)
	}
	return ks.cbbucket.StartRandomScan(context.RequestId(), context, 0, "_default", "_default", sampleSize, timeout, pipelineSize,
		serverless, context.UseReplica(), xattrs, withDocs)
}

func getCollectionId(clientContext ...*memcached.ClientContext) (collectionId uint32, user string) {
	if len(clientContext) > 0 {
		return clientContext[0].CollId, clientContext[0].User
	}
	return
}

func setPreserveExpiry(present bool, context datastore.QueryContext, clientContext ...*memcached.ClientContext) errors.Error {
	preserve := !present && context.PreserveExpiry()
	if len(clientContext) > 0 {
		clientContext[0].PreserveExpiry = preserve
	} else if preserve {
		return errors.NewPreserveExpiryNotSupported()
	}
	return nil
}

func setMutateClientContext(context datastore.QueryContext, clientContext ...*memcached.ClientContext) errors.Error {
	durability_level := context.DurabilityLevel()
	if durability_level >= datastore.DL_MAJORITY {
		if len(clientContext) > 0 {
			clientContext[0].DurabilityLevel = gomemcached.DurabilityLvl(durability_level - 1)
			clientContext[0].DurabilityTimeout = context.KvTimeout()
		} else {
			return errors.NewDurabilityNotSupported()
		}
	}
	return nil
}

// Cleanup entries in the system collection that have been orphaned by activity outside of the query service
// If a scope is dropped whilst the bucket is loaded we clean-up immediately, if not this aims to take care of it

const _BATCH_SIZE = 512

func CleanupSystemCollection(namespace string, bucket string) {

	processResult := func(key string) {
		if strings.HasPrefix(key, "seq::") {
			sequences.CleanupCacheEntry(namespace, bucket, key)
		} else if strings.HasPrefix(key, "udf::") {
			parts := strings.Split(key, "::")
			functions.FunctionClear(bucket+"."+parts[1], nil)
		}
	}

	processStaleCBOEntries := func(keyspaces []string) {
		for _, keyspace := range keyspaces {
			DropDictCacheEntry(keyspace, false)
		}
	}

	pairs := make([]value.Pair, 0, _BATCH_SIZE)
	errorCount := 0
	deletedCount := 0
	var staleCBOKeyspaces []string

	datastore.ScanSystemCollection(bucket, "", nil,
		func(key string, systemCollection datastore.Keyspace) errors.Error {
			logging.Debugf("Key: %v", key)
			parts := strings.Split(key, "::")
			toDelete := false
			isCBOKeyspaceDoc := false

			if len(parts) == 3 && (parts[0] == "seq" || parts[0] == "cbo" || parts[0] == "udf") {
				path := parts[len(parts)-1]
				if parts[0] == "cbo" {
					keyspace, keyspaceMayContainUUID, isKeyspaceDoc, err := GetCBOKeyspaceFromKey(path)
					if err != nil {
						errorCount++
						return nil
					}

					path = keyspace
					isCBOKeyspaceDoc = isKeyspaceDoc

					// Resolve UUIDs in the doc key to actual scope and collection names
					if keyspaceMayContainUUID {
						resolvedPath, found, err1 := GetCBOKeyspaceFromDoc(key, bucket, true)
						if err1 != nil {
							errorCount++
							return nil
						} else if !found {
							return nil
						}

						path = resolvedPath
					}
				}
				elements := strings.Split(path, ".")
				if len(elements) == 2 {
					s, err := systemCollection.Scope().Bucket().ScopeByName(elements[0])
					if err == nil && s.Uid() != parts[1] {
						err = errors.NewCbScopeNotFoundError(nil, s.Name()) // placeholder to trigger deletion
					}
					if err == nil && parts[0] == "cbo" {
						_, err = s.KeyspaceByName(elements[1])
					}
					if err != nil {
						toDelete = true
						if isCBOKeyspaceDoc {
							sb := strings.Builder{}
							sb.WriteString("default:")
							sb.WriteString(bucket)
							sb.WriteString(".")
							sb.WriteString(path)
							staleCBOKeyspaces = append(staleCBOKeyspaces, sb.String())
						}
					}
				}
			} else if len(parts) > 2 && parts[0] == "aus_setting" {
				// Refer to aus/aus_ee.go for AUS settings document key formats.
				path := parts[len(parts)-1]
				elements := strings.Split(path, ".")

				// Check if key is for a valid scope level AUS setting doc
				if len(parts) == 3 && len(elements) == 1 {
					s, err := systemCollection.Scope().Bucket().ScopeByName(elements[0])
					if err != nil || (s.Uid() != parts[1]) {
						toDelete = true
					}
				} else if len(parts) == 4 && len(elements) == 2 { // Check if key is for a valid collection level AUS setting doc
					s, err := systemCollection.Scope().Bucket().ScopeByName(elements[0])
					if err != nil || (s.Uid() != parts[1]) {
						toDelete = true
					}

					if !toDelete {
						c, err := s.KeyspaceByName(elements[1])
						if err != nil || (c.Uid() != parts[2]) {
							toDelete = true
						}
					}
				}
			} else if len(parts) == 4 && parts[0] == "aus_change_doc" {
				// Is an AUS change history document
				path := parts[len(parts)-1]
				elements := strings.Split(path, ".")

				if len(elements) == 3 {
					// Check is scope is still present
					s, err := systemCollection.Scope().Bucket().ScopeByName(elements[0])
					if err != nil || (s.Uid() != parts[1]) {
						toDelete = true
					}

					if !toDelete {
						// Check if collection is still present
						c, err := s.KeyspaceByName(elements[1])
						if err != nil || (c.Uid() != parts[2]) {
							toDelete = true
						}
					}
				}
			}

			if toDelete {
				logging.Infof("Deleting stale `%s` system collection key: %v", bucket, key)
				pairs = append(pairs, value.Pair{Name: key})
				if len(pairs) >= _BATCH_SIZE {
					_, results, errs := systemCollection.Delete(pairs, datastore.NULL_QUERY_CONTEXT, true)
					for i := range results {
						processResult(results[i].Name)
					}
					if errs != nil && len(errs) > 0 {
						errorCount += len(errs)
						logging.Debugf("%v:%v - %v", namespace, bucket, errs[0])
					}
					deletedCount += len(pairs) - len(errs)
					pairs = pairs[:0]

					processStaleCBOEntries(staleCBOKeyspaces)
					staleCBOKeyspaces = staleCBOKeyspaces[:0]
				}
			}
			return nil
		},
		func(systemCollection datastore.Keyspace) errors.Error {
			if len(pairs) > 0 {
				_, results, errs := systemCollection.Delete(pairs, datastore.NULL_QUERY_CONTEXT, true)
				for i := range results {
					processResult(results[i].Name)
				}
				if errs != nil && len(errs) > 0 {
					errorCount += len(errs)
					logging.Debugf("%v:%v - %v", namespace, bucket, errs[0])
				}
				deletedCount += len(pairs) - len(errs)
				processStaleCBOEntries(staleCBOKeyspaces)
				staleCBOKeyspaces = staleCBOKeyspaces[:0]
			}
			return nil
		})

	logging.Debugf("%v:%v deleted: %v errors: %v", namespace, bucket, deletedCount, errorCount)
}

const (
	_BUCKET_SYSTEM_SCOPE      = "_system"
	_BUCKET_SYSTEM_COLLECTION = "_query"
	_BUCKET_SYSTEM_PRIM_INDEX = "ix_system_query"
)

func (s *store) CreateSysPrimaryIndex(idxName, requestId string, indexer3 datastore.Indexer3) errors.Error {
	var er errors.Error

	// make sure there is an index service first
	numIndexNodes, errs := s.getNumIndexNodes()
	if len(errs) > 0 {
		return errs[0]
	}
	if numIndexNodes == 0 {
		return errors.NewNoIndexServiceError()
	}

	// next make sure index storage mode is set
	if gsiIndexer, ok := indexer3.(datastore.GsiIndexer); ok {
		cfgSet := false
		maxRetry := 8
		interval := 250 * time.Millisecond
		for i := 0; i < maxRetry; i++ {
			cfg := gsiIndexer.GetGsiClientConfig()["indexer.settings.storage_mode"]
			if cfgStr, ok := cfg.(string); ok && cfgStr != "" {
				cfgSet = true
				break
			}

			time.Sleep(interval)
			interval *= 2

			er = indexer3.Refresh()
			if er != nil {
				return er
			}
		}
		if !cfgSet {
			return errors.NewSystemCollectionError("Indexer storage mode not set", nil)
		}
	}

	// if not serverless, using the number of index nodes in the cluster create the primary index
	// with replicas in the following fashion:
	//    numIndexNodes >= 4    ==> num_replica = 2
	//    numIndexNodes >  1    ==> num_replica = 1
	//    numIndexNodes == 1    ==> no replicas
	// for serverless, the number of replicas is determined automatically
	var with value.Value
	var replica map[string]interface{}
	num_replica := 0
	if !tenant.IsServerless() {
		if numIndexNodes >= 4 {
			num_replica = 2
		} else if numIndexNodes > 1 {
			num_replica = 1
		}
		if num_replica > 0 {
			replica = make(map[string]interface{}, 1)
			replica["num_replica"] = num_replica
			with = value.NewValue(replica)
		}
	}

	var cont bool
	createPrimaryIndex := func(n_replica int) (bool, errors.Error) {
		_, err := indexer3.CreatePrimaryIndex3(requestId, idxName, nil, with)
		if err != nil && !errors.IsIndexExistsError(err) {
			if n_replica > 0 && err.HasCause(errors.E_ENTERPRISE_FEATURE) && err.ContainsText("Index Replica not supported") {
				n_replica = 1   // this will remove replicas from the repeat attempt below
				err = nil       // skip initial error check in the loop below
				num_replica = 0 // don't attempt to use replicas in the future
			}
			// if the create failed due to not enough indexer nodes, retry with fewer replicas
			for n_replica > 0 {
				// defined as ErrNotEnoughIndexers in indexing/secondary/common/const.go
				if err != nil && !err.ContainsText("not enough indexer nodes to create index with replica") {
					return false, err
				}

				n_replica--
				if n_replica == 0 {
					with = nil
				} else {
					replica["num_replica"] = n_replica
					with = value.NewValue(replica)
				}

				// retry with fewer replicas
				_, err = indexer3.CreatePrimaryIndex3(requestId, idxName, nil, with)
				if err == nil || errors.IsIndexExistsError(err) {
					break
				}
			}
			if err != nil && !errors.IsIndexExistsError(err) {
				return false, err
			}
		}
		return true, err
	}
	cont, er = createPrimaryIndex(num_replica)
	if !cont {
		return er
	}
	existing := er != nil && errors.IsIndexExistsError(er)

	var sysIndex datastore.Index
	maxRetry := 8
	if idxName == _BUCKET_SYSTEM_PRIM_INDEX {
		maxRetry = 10
	}
	interval := 250 * time.Millisecond
	for i := 0; i < maxRetry; i++ {
		time.Sleep(interval)
		interval *= 2

		er = indexer3.Refresh()
		if er != nil {
			return er
		}
		sysIndex, er = indexer3.IndexByName(idxName)
		if sysIndex != nil {
			state, _, err1 := sysIndex.State()
			if err1 != nil {
				return err1
			}
			if state == datastore.ONLINE {
				break
			}
		} else if er != nil {
			if !errors.IsIndexNotFoundError(er) {
				return er
			} else if existing {
				// if the initial creation attempted failed with "already exists" but
				// now the index is not found, retry the creation
				// it could still be waiting for that index to be reported by the
				// indexer, so keep on retrying if it still fails with "already exists"
				time.Sleep(time.Duration((rand.Int()%15)+1) * time.Millisecond) // try ensure no concurrent retries
				cont, er = createPrimaryIndex(num_replica)
				if !cont || (er != nil && !errors.IsIndexExistsError(er)) {
					return er
				} else if er == nil {
					existing = false
				}
			}
		}
	}

	return er
}

func (s *store) GetSystemCollection(bucketName string) (datastore.Keyspace, errors.Error) {
	return datastore.GetKeyspace("default", bucketName, _BUCKET_SYSTEM_SCOPE, _BUCKET_SYSTEM_COLLECTION)
}

func (s *store) getNumIndexNodes() (int, errors.Errors) {
	info := s.Info()
	nodes, errs := info.Topology()
	if len(errs) > 0 {
		return 0, errs
	}

	numIndexNodes := 0
	for _, node := range nodes {
		nodeServices, errs := info.Services(node)
		if len(errs) > 0 {
			return 0, errs
		}
		// the nodeServices should have an element named "services" which is
		// an array of service names on that node, e.g. ["n1ql", "kv", "index"]
		if services, ok := nodeServices["services"]; ok {
			if serviceArr, ok := services.([]interface{}); ok {
				for _, serv := range serviceArr {
					if name, ok := serv.(string); ok {
						if name == "index" {
							numIndexNodes++
						}
					}
				}
			}
		}
	}

	return numIndexNodes, nil
}

// check for existance of system collection, and create primary index if necessary
func (s *store) CheckSystemCollection(bucketName, requestId string, forceIndex bool, randomDelay int) (bool, errors.Error) {
	sysColl, err := s.GetSystemCollection(bucketName)
	if err != nil {
		// make sure the bucket exists before we wait (e.g. index advisor)
		switch err.Code() {
		case errors.E_CB_KEYSPACE_NOT_FOUND, errors.E_CB_BUCKET_NOT_FOUND:
			defaultPool, er := s.NamespaceByName("default")
			if er != nil {
				return false, er
			}

			_, er = defaultPool.BucketByName(bucketName)
			if er != nil {
				return false, er
			}
		case errors.E_CB_SCOPE_NOT_FOUND:
			// no-op, ignore
		default:
			return false, err
		}
		// wait for system collection to show up
		maxRetry := 8
		interval := 250 * time.Millisecond
		for i := 0; i < maxRetry; i++ {
			switch err.Code() {
			case errors.E_CB_KEYSPACE_NOT_FOUND, errors.E_CB_BUCKET_NOT_FOUND, errors.E_CB_SCOPE_NOT_FOUND:
				// no-op, ignore these errors
			default:
				return false, err
			}

			time.Sleep(interval)
			interval *= 2

			sysColl, err = s.GetSystemCollection(bucketName)
			if sysColl != nil || err == nil {
				break
			}
		}
		if err != nil {
			return false, err
		} else if sysColl == nil {
			return false, errors.NewSystemCollectionError("System collection not available for bucket "+bucketName, nil)
		}
	}

	if requestId == "" {
		return false, nil
	}

	empty := false

	cnt, err1 := sysColl.Count(datastore.NULL_QUERY_CONTEXT)
	if err1 != nil {
		return false, errors.NewSystemCollectionError("Count from system collection for bucket "+bucketName, err1)
	} else if cnt < 0 {
		return false, errors.NewSystemCollectionError(fmt.Sprintf("Invalid count (%d) from system collection for bucket %s", cnt, bucketName), nil)
	} else if cnt == 0 {
		empty = true
	}
	if !forceIndex {
		// if the system collection is empty, don't create the primary index yet
		if empty {
			return empty, nil
		}
		if randomDelay > 0 {
			// random delay requested, use 0-10 times the requested delay (in Milliseconds)
			delay := rand.Intn(11) * randomDelay
			if delay > 0 {
				time.Sleep(time.Duration(delay) * time.Millisecond)
			}
		}
	}

	indexer, er := sysColl.Indexer(datastore.GSI)
	if er != nil {
		return false, er
	}

	indexer3, ok := indexer.(datastore.Indexer3)
	if !ok {
		return false, errors.NewInvalidGSIIndexerError("Cannot get primary index on system collection")
	}

	sysIndex, er := indexer3.IndexByName(_BUCKET_SYSTEM_PRIM_INDEX)
	if er != nil {
		if !errors.IsIndexNotFoundError(er) {
			// only ignore index not found error
			return false, er
		}

		// create primary index on system collection if not already exists
		// the create function waits for ONLINE state before it returns
		er = s.CreateSysPrimaryIndex(_BUCKET_SYSTEM_PRIM_INDEX, requestId, indexer3)
		if er != nil && !errors.IsIndexExistsError(er) {
			// ignore index already exist error
			return false, er
		}
	} else {
		// make sure the primary index is ONLINE
		maxRetry := 10
		interval := 250 * time.Millisecond
		done := false
		for i := 0; i < maxRetry; i++ {
			state, _, er1 := sysIndex.State()
			if er1 != nil {
				return false, er1
			}
			if state == datastore.ONLINE {
				done = true
				break
			} else if state == datastore.DEFERRED {
				// build system index if it is deferred (e.g. just restored)
				er = indexer3.BuildIndexes(requestId, sysIndex.Name())
				if er != nil {
					return false, er
				}
			}

			time.Sleep(interval)
			interval *= 2

			er = indexer3.Refresh()
			if er != nil {
				return false, er
			}

			sysIndex, er = indexer3.IndexByName(_BUCKET_SYSTEM_PRIM_INDEX)
			if er != nil {
				return false, er
			}
		}
		if !done {
			return false, errors.NewSysCollectionPrimaryIndexError(bucketName)
		}
	}

	return empty, nil
}
