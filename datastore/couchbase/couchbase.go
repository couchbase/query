//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/couchbase/cbauth"
	cb "github.com/couchbase/go-couchbase"
	"github.com/couchbase/gomemcached"
	"github.com/couchbase/gomemcached/client" // package name is memcached
	gsi "github.com/couchbase/indexing/secondary/queryport/n1ql"
	ftsclient "github.com/couchbase/n1fty"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/couchbase/gcagent"
	"github.com/couchbase/query/datastore/virtual"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/transactions"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

var REQUIRE_CBAUTH bool // Connection to authorization system must succeed.

// cbPoolMap and cbPoolServices implement a local cache of the datastore's topology
type cbPoolMap struct {
	sync.RWMutex
	poolServices map[string]cbPoolServices
}

type cbPoolServices struct {
	name         string
	rev          int
	nodeServices map[string]interface{}
}

var _POOLMAP cbPoolMap

const (
	PRIMARY_INDEX          = "#primary"
	_TRAN_CLEANUP_INTERVAL = 1 * time.Minute
)

func init() {

	// MB-27415 have a larger overflow pool and close overflow connections asynchronously
	cb.SetConnectionPoolParams(64, 64)
	cb.EnableAsynchronousCloser(true)

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
	_POOLMAP.poolServices = make(map[string]cbPoolServices, 1)

	cb.EnableCollections = true

	// transaction cache initalization
	transactions.TranContextCacheInit(_TRAN_CLEANUP_INTERVAL)
}

// store is the root for the couchbase datastore
type store struct {
	client         cb.Client // instance of go-couchbase client
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
	var errs []errors.Error

	for _, p := range info.client.Info.Pools {
		pool, err := info.client.GetPool(p.Name)

		if err == nil {
			for _, node := range pool.Nodes {
				nodes = append(nodes, fullhostName(node.Hostname))
			}
			pool.Close()
		} else {
			errs = append(errs, errors.NewDatastoreClusterError(err, p.Name))
		}
	}
	return nodes, errs
}

func (info *infoImpl) Services(node string) (map[string]interface{}, []errors.Error) {
	var errs []errors.Error

	isReadLock := true
	_POOLMAP.RLock()
	defer func() {
		if isReadLock {
			_POOLMAP.RUnlock()
		} else {
			_POOLMAP.Unlock()
		}
	}()

	// scan the pools
	for _, p := range info.client.Info.Pools {
		pool, err := info.client.GetPool(p.Name)
		poolServices, pErr := info.client.GetPoolServices(p.Name)

		if err == nil && pErr == nil {
			var found bool = false
			var services cbPoolServices

			services, ok := _POOLMAP.poolServices[p.Name]
			found = ok && (services.rev == poolServices.Rev)

			// missing the information, rebuild
			if !found {

				// promote the lock
				if isReadLock {
					_POOLMAP.RUnlock()
					_POOLMAP.Lock()
					isReadLock = false

					// now that we have promoted the lock, did we get beaten by somebody else to it?
					services, ok = _POOLMAP.poolServices[p.Name]
					found = ok && (services.rev == poolServices.Rev)
					if found {
						continue
					}
				}

				newPoolServices := cbPoolServices{name: p.Name, rev: poolServices.Rev}
				nodeServices := make(map[string]interface{}, len(pool.Nodes))

				// go through all the nodes in the pool
				for _, n := range pool.Nodes {
					var servicesCopy []interface{}

					newServices := make(map[string]interface{}, 3)
					newServices["name"] = fullhostName(n.Hostname)
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
							msg := fmt.Sprintf("NodeServices does not report mgmt endpoint for "+
								"this node: %v", node)
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
				_POOLMAP.poolServices[p.Name] = newPoolServices
				services = newPoolServices
			}
			nodeServices, ok := services.nodeServices[node]
			if ok {
				return nodeServices.(map[string]interface{}), errs
			}
		} else {
			if err != nil {
				errs = append(errs, errors.NewDatastoreClusterError(err, p.Name))
			}
			if pErr != nil {
				errs = append(errs, errors.NewDatastoreClusterError(pErr, p.Name))
			}
		}
		if err == nil {
			pool.Close()
		}
	}
	return map[string]interface{}{}, errs
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

func (s *store) Authorize(privileges *auth.Privileges, credentials *auth.Credentials) (auth.AuthenticatedUsers, errors.Error) {
	if s.CbAuthInit == false {
		// cbauth is not initialized. Access to SASL protected buckets will be
		// denied by the couchbase server
		logging.Warnf("CbAuth not intialized")
		return nil, nil
	}
	return cbAuthorize(s, privileges, credentials)
}

func (s *store) PreAuthorize(privileges *auth.Privileges) {
	cbPreAuthorize(privileges)
}

func (s *store) CredsString(req *http.Request) string {
	if req != nil {
		creds, err := cbauth.AuthWebCreds(req)
		if err == nil {
			return creds.Name()
		}
	}
	return ""
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
		return nil, errors.NewSystemUnableToRetrieveError(err)
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
		UserWhitelisted: users,
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

type DefaultObject struct {
	Uid string `json:"auditUid"`
}

func (s *store) UserInfo() (value.Value, errors.Error) {
	data, err := s.client.GetUserRoles()
	if err != nil {
		return nil, errors.NewSystemUnableToRetrieveError(err)
	}
	return value.NewValue(data), nil
}

func (s *store) GetUserInfoAll() ([]datastore.User, errors.Error) {
	sourceUsers, err := s.client.GetUserInfoAll()
	if err != nil {
		return nil, errors.NewSystemUnableToRetrieveError(err)
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
	err := s.client.PutUserInfo(&outputUser)
	if err != nil {
		return errors.NewSystemUnableToUpdateError(err)
	}
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
	}
	return roles, nil
}

func (s *store) SetClientConnectionSecurityConfig() (err error) {
	if s.connSecConfig != nil && s.connSecConfig.ClusterEncryptionConfig.EncryptData {
		err = s.client.InitTLS(s.connSecConfig.CertFile)
		if err == nil && s.gcClient != nil {
			err = s.gcClient.InitTLS(s.connSecConfig.CertFile)
		}
		if err != nil {
			err = fmt.Errorf("Unable to initialize TLS using cert file %s. Aborting security update. Error:%v",
				s.connSecConfig.CertFile, err)
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

	// Implementation based on SetLogLevel(), above.
	for _, n := range s.namespaceCache {
		n.lock.Lock()
		for _, k := range n.keyspaceCache {
			if k.cbKeyspace == nil {
				continue
			}

			// Make new TLS settings take effect in the buckets.
			k.cbKeyspace.cbbucket.RefreshFully()
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

	client, err := cb.ConnectWithAuth(url, cbauth.NewAuthHandler(nil))
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
		client, err = cb.Connect(u)
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

	return store, nil
}

func loadNamespace(s *store, name string) (*namespace, errors.Error) {
	cbpool, err := s.client.GetPool(name)
	if err != nil {
		if name == "default" {
			// if default pool is not available, try reconnecting to the server
			url := s.URL()

			var client cb.Client

			if s.CbAuthInit == true {
				client, err = cb.ConnectWithAuth(url, cbauth.NewAuthHandler(nil))
			} else {
				client, err = cb.Connect(url)
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

const (
	_MIN_ERR_INTERVAL            time.Duration = 5 * time.Second
	_THROTTLING_TIMEOUT          time.Duration = 10 * time.Millisecond
	_CLEANUP_INTERVAL            time.Duration = time.Hour
	_NAMESPACE_REFRESH_THRESHOLD time.Duration = 100 * time.Millisecond
	_STATS_REFRESH_THRESHOLD     time.Duration = 1 * time.Second
)

func (p *namespace) DatastoreId() string {
	return p.store.Id()
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

func (p *namespace) Objects(preload bool) ([]datastore.Object, errors.Error) {
	p.refresh()
	p.nslock.RLock()
	rv := make([]datastore.Object, len(p.cbNamespace.BucketMap))
	i := 0

	for name, _ := range p.cbNamespace.BucketMap {
		var defaultCollection datastore.Keyspace

		o := datastore.Object{name, name, false, false}
		p.lock.RLock()
		entry := p.keyspaceCache[name]
		if entry != nil && entry.cbKeyspace != nil {
			defaultCollection = entry.cbKeyspace.defaultCollection
		}
		p.lock.RUnlock()

		if preload && defaultCollection == nil {
			ks, _ := p.KeyspaceByName(name)
			if ks != nil {
				defaultCollection = ks.(*keyspace).defaultCollection
			}
		}

		// if we have loaded the bucket, check if the bucket has a default collection
		// if we haven't loaded the bucket, see if you can get the default collection id
		// the bucket is a keyspace if the default collection exists
		if defaultCollection != nil {
			switch k := defaultCollection.(type) {
			case *collection:
				o.IsKeyspace = (k != nil)
				o.IsBucket = true
			case *keyspace:
				o.IsKeyspace = (k != nil)
				o.IsBucket = false
			}
		} else if !preload {
			bucket, _ := p.cbNamespace.GetBucket(name)
			if bucket != nil {
				_, _, err := bucket.GetCollectionCID("_default", "_default", time.Time{})
				if err == nil {
					o.IsKeyspace = true
				}
			}
			o.IsBucket = true
		}
		rv[i] = o
		i++
	}
	p.nslock.RUnlock()
	return rv, nil
}

func (p *namespace) KeyspaceByName(name string) (datastore.Keyspace, errors.Error) {
	return p.keyspaceByName(name)
}

func (p *namespace) VirtualKeyspaceByName(path []string) (datastore.Keyspace, errors.Error) {
	return virtual.NewVirtualKeyspace(p, path)
}

func (p *namespace) keyspaceByName(name string) (*keyspace, errors.Error) {
	var err errors.Error
	var keyspace *keyspace

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
						if mani.Uid > keyspace.collectionsManifestUid {
							keyspace.collectionsManifestUid = mani.Uid
							keyspace.scopes = scopes

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
	entry = p.keyspaceCache[name]
	if entry == nil {

		// adding a new keyspace does not force the namespace version to change
		// all previously prepared statements are still good
		entry = &keyspaceEntry{}
		p.keyspaceCache[name] = entry
	} else if entry.cbKeyspace != nil && entry.cbKeyspace.flags != 0 {

		// a keyspace that has been deleted or needs refreshing causes a
		// version change
		entry.cbKeyspace = nil
	}
	entry.lastUse = util.Now()
	p.lock.Unlock()

	// 2) serialize the loading by locking the entry
	entry.Lock()
	defer entry.Unlock()

	// 3) check if somebody has done the job for us in the interim
	if entry.cbKeyspace != nil {
		return entry.cbKeyspace, nil
	}

	// 4) if previous loads resulted in errors, throttle requests
	if entry.errCount > 0 && util.Since(entry.lastUse) < _THROTTLING_TIMEOUT {
		time.Sleep(_THROTTLING_TIMEOUT)
	}

	// 5) try the loading
	k, err := newKeyspace(p, name)
	if err != nil {

		// We try not to flood the log with errors
		if entry.errCount == 0 {
			entry.errTime = util.Now()
		} else if util.Since(entry.errTime) > _MIN_ERR_INTERVAL {
			entry.errTime = util.Now()
		}
		entry.errCount++
		return nil, err
	}
	entry.errCount = 0

	// this is the only place where entry.cbKeyspace is set
	// it is never unset - so it's safe to test cbKeyspace != nil
	entry.cbKeyspace = k
	return k, nil
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
	return p.keyspaceByName(id)

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
	return p.keyspaceByName(name)
}

func (p *namespace) BucketByName(name string) (datastore.Bucket, errors.Error) {
	return p.keyspaceByName(name)
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

	// trigger refresh of this pool
	logging.Debugf("Refreshing pool %s", p.name)

	newpool, err := p.store.client.GetPool(p.name)
	if err != nil {
		newpool, err = p.reload1(err)
		if err == nil {
			p.reload2(&newpool)
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
		p.reload2(&newpool)
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

func (p *namespace) reload() {
	logging.Debugf("Reload %s", p.name)

	newpool, err := p.store.client.GetPool(p.name)
	if err != nil {
		newpool, err = p.reload1(err)
		if err != nil {
			return
		}
	}
	p.reload2(&newpool)
}

func (p *namespace) reload1(err error) (cb.Pool, error) {
	var client cb.Client

	logging.Errorf("Error updating pool name %s: Error %v", p.name, err)
	url := p.store.URL()

	/*
		transport := cbauth.WrapHTTPTransport(cb.HTTPTransport, nil)
		cb.HTTPClient.Transport = transport
	*/

	if p.store.CbAuthInit == true {
		client, err = cb.ConnectWithAuth(url, cbauth.NewAuthHandler(nil))
	} else {
		client, err = cb.Connect(url)
	}
	if err != nil {
		logging.Errorf("Error connecting to URL %s - %v", url, err)
		return cb.Pool{}, err
	}
	// check if the default pool exists
	newpool, err := client.GetPool(p.name)
	if err != nil {
		logging.Errorf("Retry Failed Error updating pool name <ud>%s</ud>: Error %v", p.name, err)
		return newpool, err
	}
	p.store.client = client

	err = p.store.SetClientConnectionSecurityConfig()
	if err != nil {
		return newpool, err
	}

	return newpool, nil
}

func (p *namespace) reload2(newpool *cb.Pool) {
	p.lock.Lock()
	for name, ks := range p.keyspaceCache {
		logging.Debugf(" Checking keyspace %s", name)
		if ks.cbKeyspace == nil {
			if util.Since(ks.lastUse) > _CLEANUP_INTERVAL {
				delete(p.keyspaceCache, name)
			}
			continue
		}
		newbucket, err := newpool.GetBucket(name)
		if err != nil {
			ks.cbKeyspace.Release(true)
			logging.Errorf(" Error retrieving bucket %s - %v", name, err)
			delete(p.keyspaceCache, name)

		} else if ks.cbKeyspace.cbbucket.UUID != newbucket.UUID {
			logging.Debugf(" UUid of keyspace %v uuid now %v", ks.cbKeyspace.cbbucket.UUID, newbucket.UUID)
			// UUID has changed. Update the keyspace struct with the newbucket
			// and release old one
			ks.cbKeyspace.cbbucket.Close()
			ks.cbKeyspace.cbbucket = newbucket
		} else {

			// we are reloading, so close old and set new bucket
			ks.cbKeyspace.cbbucket.Close()
			ks.cbKeyspace.cbbucket = newbucket
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

	// MB-36458 switch pool and...
	p.nslock.Lock()
	oldPool := p.cbNamespace
	p.cbNamespace = newpool
	p.nslock.Unlock()

	// ...MB-33185 let go of old pool when noone is accessing it
	oldPool.Close()

	// keyspaces have been reloaded, force full auto reprepare check
	p.version++
}

const (
	_DELETED        = 1 << iota // this bucket no longer exists
	_NEEDS_REFRESH              // received error that indicates the bucket needs refreshing
	_NEEDS_MANIFEST             // scopes or collections changed
)

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
	chkIndex       chkIndexDict
	indexersLoaded bool

	collectionsManifestUid    uint64            // current manifest id
	newCollectionsManifestUid uint64            // announced manifest id
	scopes                    map[string]*scope // scopes by id
	defaultCollection         datastore.Keyspace
	last                      util.Time // last refresh
}

var _NO_SCOPES map[string]*scope = map[string]*scope{}

func newKeyspace(p *namespace, name string) (*keyspace, errors.Error) {

	cbNamespace := p.getPool()
	cbbucket, err := cbNamespace.GetBucket(name)

	if err != nil {
		logging.Infof(" keyspace %s not found %v", name, err)
		// go-couchbase caches the buckets
		// to be sure no such bucket exists right now
		// we trigger a refresh
		p.reload()
		cbNamespace = p.getPool()

		// and then check one more time
		logging.Infof(" Retrying bucket %s", name)
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
	} else {
		logging.Infof("Unable to retrieve collections info for bucket %s: %v", name, err)
	}

	// if we don't have any scope (not even default) revert to old style keyspace
	if len(rv.scopes) == 0 {
		rv.defaultCollection = rv
	}

	logging.Infof("Created New Bucket %s", name)

	// Create a bucket updater that will keep the couchbase bucket fresh.
	cbbucket.RunBucketUpdater2(p.KeyspaceUpdateCallback, p.KeyspaceDeleteCallback)

	return rv, nil
}

// Called by go-couchbase if a configured keyspace is deleted
func (p *namespace) KeyspaceDeleteCallback(name string, err error) {

	var cbKeyspace *keyspace

	p.lock.Lock()

	ks, ok := p.keyspaceCache[name]
	if ok && ks.cbKeyspace != nil {
		logging.Infof("Keyspace %v being deleted", name)
		cbKeyspace = ks.cbKeyspace
		ks.cbKeyspace.Release(false)
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
			// N1QL_SYSTEM_BUCKET.N1QL_SYSTEM_SCOPE.N1QL_CBO_STATS
			dropDictCacheEntries(cbKeyspace)
		}
	}
}

// Called by go-couchbase if a configured keyspace is updated
func (p *namespace) KeyspaceUpdateCallback(bucket *cb.Bucket) {

	checkSysBucket := false

	p.lock.Lock()

	ks, ok := p.keyspaceCache[bucket.Name]
	if ok && ks.cbKeyspace != nil {
		ks.cbKeyspace.Lock()
		uid, _ := strconv.ParseUint(bucket.CollectionsManifestUid, 16, 64)
		if ks.cbKeyspace.collectionsManifestUid != uid {
			ks.cbKeyspace.flags |= _NEEDS_MANIFEST
			ks.cbKeyspace.newCollectionsManifestUid = uid
			if isSysBucket(ks.cbKeyspace.name) {
				checkSysBucket = true
			}
		}
		ks.cbKeyspace.Unlock()
	} else {
		logging.Warnf("Keyspace %v not configured on this server", bucket.Name)
	}

	p.lock.Unlock()

	if checkSysBucket {
		chkSysBucket()
	}
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
}

func (b *keyspace) Stats(context datastore.QueryContext, which []datastore.KeyspaceStats) ([]int64, errors.Error) {
	return b.stats(context, which)
}

func (b *keyspace) stats(context datastore.QueryContext, which []datastore.KeyspaceStats, clientContext ...*memcached.ClientContext) ([]int64, errors.Error) {
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
	count, err := b.cbbucket.GetCount(b.needsTimeRefresh(_STATS_REFRESH_THRESHOLD), clientContext...)
	if err != nil {
		b.checkRefresh(err)
		return 0, errors.NewCbKeyspaceCountError(err, b.fullName)
	}
	return count, nil
}

func (b *keyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return b.size(context)
}

func (b *keyspace) size(context datastore.QueryContext, clientContext ...*memcached.ClientContext) (int64, errors.Error) {
	size, err := b.cbbucket.GetSize(b.needsTimeRefresh(_STATS_REFRESH_THRESHOLD), clientContext...)
	if err != nil {
		b.checkRefresh(err)
		return 0, errors.NewCbKeyspaceSizeError(err, b.fullName)
	}
	return size, nil
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

	return indexers, err
}

// return a document key free from collection ids
func key(k []byte, clientContext ...*memcached.ClientContext) []byte {
	if len(clientContext) == 0 {
		return k
	}

	i := 1
	collId := clientContext[0].CollId
	for collId >= 0x80 {
		collId >>= 7
		i++
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

func (k *keyspace) GetRandomEntry() (string, value.Value, errors.Error) {
	return k.getRandomEntry("", "")
}

func (k *keyspace) getRandomEntry(scopeName, collectionName string,
	clientContext ...*memcached.ClientContext) (string, value.Value, errors.Error) {
	resp, err := k.cbbucket.GetRandomDoc(clientContext...)

	if err != nil {
		k.checkRefresh(err)

		// Ignore "Not found" errors
		if isNotFoundError(err) {
			return "", nil, nil
		}
		return "", nil, errors.NewCbGetRandomEntryError(err)
	}

	return string(key(resp.Key, clientContext...)), value.NewValue(resp.Body), nil
}

func (b *keyspace) Fetch(keys []string, fetchMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string) []errors.Error {
	return b.fetch(b.fullName, b.QualifiedName(), "", "", keys, fetchMap, context, subPaths)
}

func (b *keyspace) fetch(fullName, qualifiedName, scopeName, collectionName string, keys []string,
	fetchMap map[string]value.AnnotatedValue, context datastore.QueryContext, subPaths []string,
	clientContext ...*memcached.ClientContext) []errors.Error {

	if txContext, _ := context.GetTxContext().(*transactions.TranContext); txContext != nil {
		return b.txFetch(fullName, qualifiedName, scopeName, collectionName, getCollectionId(clientContext...),
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
		mcr, err = b.cbbucket.GetsMC(keys[0], context.GetReqDeadline(), clientContext...)
	} else {
		if ls > 0 && (subPaths[0] != "$document" && subPaths[0] != "$document.exptime") {
			subPaths = append([]string{"$document"}, subPaths...)
			noVirtualDocAttr = true
		}

		if l == 1 {
			mcr, err = b.cbbucket.GetsSubDoc(keys[0], context.GetReqDeadline(), subPaths, clientContext...)
		} else {
			bulkResponse, err = b.cbbucket.GetBulk(keys, context.GetReqDeadline(), subPaths, clientContext...)
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
			return []errors.Error{errors.NewCbBulkGetError(err, "")}
		}
	}

	if fast {
		if mcr != nil && err == nil {
			fetchMap[keys[0]] = doFetch(keys[0], fullName, mcr)
		}

	} else if l == 1 {
		if mcr != nil && err == nil {
			fetchMap[keys[0]] = getSubDocFetchResults(keys[0], mcr, subPaths, noVirtualDocAttr)
		}
	} else {
		i := 0
		if ls > 0 {
			for k, v := range bulkResponse {
				fetchMap[k] = getSubDocFetchResults(k, v, subPaths, noVirtualDocAttr)
				i++
			}
		} else {
			for k, v := range bulkResponse {
				fetchMap[k] = doFetch(k, fullName, v)
				i++
			}
			logging.Debugf("Requested keys %d Fetched %d keys ", l, i)
		}
	}

	return nil
}

func doFetch(k string, fullName string, v *gomemcached.MCResponse) value.AnnotatedValue {
	val := value.NewAnnotatedValue(value.NewParsedValue(v.Body, (v.DataType&byte(0x01) != 0)))

	var flags, expiration uint32

	if len(v.Extras) >= 4 {
		flags = binary.BigEndian.Uint32(v.Extras[0:4])
	}

	if len(v.Extras) >= 8 {
		expiration = binary.BigEndian.Uint32(v.Extras[4:8])
	}

	meta_type := "json"
	if val.Type() == value.BINARY {
		meta_type = "base64"
	}

	meta := val.NewMeta()
	meta["keyspace"] = fullName
	meta["cas"] = v.Cas
	meta["type"] = meta_type
	meta["flags"] = flags
	meta["expiration"] = expiration
	val.SetId(k)

	// Uncomment when needed
	//logging.Debugf("CAS Value for key %v is %v flags %v", k, uint64(v.Cas), meta_flags)

	return val
}

func getSubDocFetchResults(k string, v *gomemcached.MCResponse, subPaths []string, noVirtualDocAttr bool) value.AnnotatedValue {
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
		docMeta := xVal["$document"].(map[string]interface{})

		// Convert unmarshalled int64 values to uint32
		flags = uint32(value.NewValue(docMeta["flags"]).(value.NumberValue).Int64())
		exptime = uint32(value.NewValue(docMeta["exptime"]).(value.NumberValue).Int64())
	} else if subPaths[0] == "$document.exptime" {
		exptime = uint32(value.NewValue(xVal["$document.exptime"]).(value.NumberValue).Int64())

	}

	if noVirtualDocAttr {
		delete(xVal, "$document")
	}

	meta := val.NewMeta()
	meta["cas"] = v.Cas
	meta["type"] = meta_type
	meta["flags"] = flags
	meta["expiration"] = exptime
	if len(xVal) > 0 {
		meta["xattrs"] = xVal
	}

	val.SetId(k)

	return val
}

func (k *keyspace) checkRefresh(err error) {
	if cb.IsRefreshRequired(err) {
		k.Lock()
		k.flags |= _NEEDS_REFRESH
		k.Unlock()
	} else if cb.IsUnknownCollection(err) {
		k.Lock()
		k.flags |= _NEEDS_REFRESH
		k.Unlock()
	}
}

func (k *keyspace) setNeedsManifest() {
	k.Lock()
	k.flags |= _NEEDS_MANIFEST
	k.Unlock()
}

func isNotFoundError(err error) bool {
	return cb.IsKeyNoEntError(err)
}

func isEExistError(err error) bool {
	return cb.IsKeyEExistsError(err)
}

func getMeta(key string, val value.Value, must bool) (cas uint64, flags uint32, txnMeta interface{}, err error) {

	var meta map[string]interface{}
	var av value.AnnotatedValue
	var ok bool

	if av, ok = val.(value.AnnotatedValue); ok && av != nil {
		meta = av.GetMeta()
	}

	if _, ok = meta["cas"]; ok {
		cas, ok = meta["cas"].(uint64)
	}

	if must && !ok {
		return 0, 0, nil, fmt.Errorf("Not valid Cas value for key %v", key)
	}

	if _, ok = meta["flags"]; ok {
		flags, ok = meta["flags"].(uint32)
	}

	if must && !ok {
		return 0, 0, nil, fmt.Errorf("Not valid Flags value for key %v", key)
	}

	if _, ok = meta["txnMeta"]; ok {
		txnMeta, _ = meta["txnMeta"].(interface{})
	}

	return cas, flags, txnMeta, nil

}

func SetMetaCas(val value.Value, cas uint64) bool {
	if av, ok := val.(value.AnnotatedValue); ok && av != nil {
		av.NewMeta()["cas"] = cas
		return true
	}
	return false
}

func getExpiration(options value.Value) (exptime uint32) {
	if options != nil && options.Type() == value.OBJECT {
		if v, ok := options.Field("expiration"); ok && v.Type() == value.NUMBER {
			exptime = uint32(value.AsNumberValue(v).Int64())
		}
	}
	return
}

func (b *keyspace) performOp(op MutateOp, qualifiedName, scopeName, collectionName string, pairs []value.Pair,
	context datastore.QueryContext, clientContext ...*memcached.ClientContext) ([]value.Pair, errors.Error) {

	if len(pairs) == 0 {
		return nil, nil
	}

	if txContext, _ := context.GetTxContext().(*transactions.TranContext); txContext != nil {
		return b.txPerformOp(op, qualifiedName, scopeName, collectionName, getCollectionId(clientContext...),
			pairs, context, txContext)
	}

	var failedDeletes []string
	var err error
	mPairs := make(value.Pairs, 0, len(pairs))

	for _, kv := range pairs {
		var val interface{}
		var exptime int

		key := kv.Name
		if op != MOP_DELETE {
			if kv.Value.Type() == value.BINARY {
				return nil, errors.NewBinaryDocumentMutationError(_MutateOpNames[op], key)
			}
			val = kv.Value.ActualForIndex()
			exptime = int(getExpiration(kv.Options))
		}

		//mv := kv.Value.GetMeta()

		// TODO Need to also set meta

		switch op {

		case MOP_INSERT:
			var added bool

			// add the key to the backend
			added, err = b.cbbucket.Add(key, exptime, val, clientContext...)
			b.checkRefresh(err)
			if added == false {
				// false & err == nil => given key aready exists in the bucket
				if err != nil {
					err = errors.NewError(err, "Key "+key)
				} else {
					err = errors.NewError(nil, "Duplicate Key "+key)
				}
			}
		case MOP_UPDATE:
			// check if the key exists and if so then use the cas value
			// to update the key
			var cas uint64
			var flags uint32

			cas, flags, _, err = getMeta(key, kv.Value, true)
			if err != nil {
				// Don't perform the update if the meta values are not found
				logging.Errorf("Failed to get meta values for key <ud>%v</ud>, error %v", key, err)
			} else {

				logging.Debugf("CAS Value (Update) for key <ud>%v</ud> is %v flags <ud>%v</ud> value <ud>%v</ud>",
					key, uint64(cas), flags, val)
				_, _, err = b.cbbucket.CasWithMeta(key, int(flags), exptime, uint64(cas), val, clientContext...)
				b.checkRefresh(err)
			}

		case MOP_UPSERT:
			err = b.cbbucket.Set(key, exptime, val, clientContext...)
			b.checkRefresh(err)
		case MOP_DELETE:
			err = b.cbbucket.Delete(key, clientContext...)
			b.checkRefresh(err)
		}

		if err != nil {
			if op == MOP_DELETE {
				if !isNotFoundError(err) {
					logging.Infof("Failed to delete key <ud>%s</ud> Error %s", key, err)
					failedDeletes = append(failedDeletes, key)
				}
			} else if isEExistError(err) {
				logging.Errorf("Failed to perform update on key <ud>%s</ud>. CAS mismatch due to concurrent modifications. Error - %v", key, err)
			} else {
				logging.Errorf("Failed to perform <ud>%s</ud> on key <ud>%s</ud> for Keyspace %s. Error - %v",
					MutateOpToName(op), key, b.Name(), err)
			}
		} else {
			mPairs = append(mPairs, kv)
		}
	}

	if op == MOP_DELETE {
		if len(failedDeletes) > 0 {
			return mPairs, errors.NewCbDeleteFailedError(err, "Some keys were not deleted "+fmt.Sprintf("%v", failedDeletes))
		}
	} else if len(mPairs) == 0 {
		return nil, errors.NewCbDMLError(err, "Failed to perform "+MutateOpToName(op))
	}

	return mPairs, nil
}

func (b *keyspace) Insert(inserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return b.performOp(MOP_INSERT, b.QualifiedName(), "", "", inserts, context)

}

func (b *keyspace) Update(updates []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return b.performOp(MOP_UPDATE, b.QualifiedName(), "", "", updates, context)
}

func (b *keyspace) Upsert(upserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return b.performOp(MOP_UPSERT, b.QualifiedName(), "", "", upserts, context)
}

func (b *keyspace) Delete(deletes []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return b.performOp(MOP_DELETE, b.QualifiedName(), "", "", deletes, context)
}

func (b *keyspace) Release(bclose bool) {
	b.Lock()
	b.flags |= _DELETED
	agentProvider := b.agentProvider
	b.agentProvider = nil
	b.Unlock()
	if bclose {
		b.cbbucket.Close()
	}
	if agentProvider != nil {
		agentProvider.Close()
	}

	// close an ftsIndexer that belongs to this keyspace
	if ftsIndexerCloser, ok := b.ftsIndexer.(io.Closer); ok {
		// FTSIndexer implements a Close() method
		ftsIndexerCloser.Close()
	}
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
	b.indexersLoaded = true
}

func (b *keyspace) Scope() datastore.Scope {
	return nil
}

func (b *keyspace) ScopeId() string {
	return ""
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
	ids := make([]string, len(ks.scopes))
	ix := 0
	for k := range ks.scopes {
		ids[ix] = k
		ix++
	}
	return ids, nil
}

func (ks *keyspace) ScopeNames() ([]string, errors.Error) {
	ids := make([]string, len(ks.scopes))
	ix := 0
	for _, v := range ks.scopes {
		ids[ix] = v.Name()
		ix++
	}
	return ids, nil
}

func (ks *keyspace) ScopeById(id string) (datastore.Scope, errors.Error) {
	scope := ks.scopes[id]
	if scope == nil {
		return nil, errors.NewCbScopeNotFoundError(nil, fullName(ks.namespace.name, ks.name, id))
	}
	return scope, nil
}

func (ks *keyspace) ScopeByName(name string) (datastore.Scope, errors.Error) {
	for _, v := range ks.scopes {
		if name == v.Name() {
			return v, nil
		}
	}
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

func getCollectionId(clientContext ...*memcached.ClientContext) uint32 {
	collectionId := uint32(0)
	if len(clientContext) > 0 {
		collectionId = clientContext[0].CollId
	}
	return collectionId
}
