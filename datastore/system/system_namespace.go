//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
)

type namespace struct {
	store     *store
	id        string
	name      string
	keyspaces map[string]datastore.Keyspace
}

func (p *namespace) Datastore() datastore.Datastore {
	return p.store
}

func (p *namespace) Id() string {
	return p.id
}

func (p *namespace) Name() string {
	return p.name
}

func (p *namespace) KeyspaceIds() ([]string, errors.Error) {
	return p.KeyspaceNames()
}

func (p *namespace) KeyspaceNames() ([]string, errors.Error) {
	rv := make([]string, len(p.keyspaces))
	i := 0
	for k, _ := range p.keyspaces {
		rv[i] = k
		i = i + 1
	}
	return rv, nil
}

func (p *namespace) Objects(credentials *auth.Credentials, filter func(string) bool, preload bool) (
	[]datastore.Object, errors.Error) {

	rv := make([]datastore.Object, len(p.keyspaces))
	i := 0
	for k, _ := range p.keyspaces {
		rv[i] = datastore.Object{Id: k, Name: k, IsKeyspace: true}
		i++
	}
	return rv, nil
}

func (p *namespace) KeyspaceById(id string) (datastore.Keyspace, errors.Error) {
	return p.KeyspaceByName(id)
}

func (p *namespace) KeyspaceByName(name string) (datastore.Keyspace, errors.Error) {

	b, ok := p.keyspaces[name]
	if !ok {
		return nil, errors.NewSystemKeyspaceNotFoundError(nil, name)
	}

	return b, nil
}

func (p *namespace) MetadataVersion() uint64 {
	return 0
}

func (p *namespace) MetadataId() string {
	return p.name
}

// newNamespace creates a new namespace.
func newNamespace(s *store) (*namespace, errors.Error) {
	p := new(namespace)
	p.store = s
	p.id = NAMESPACE_ID
	p.name = NAMESPACE_NAME
	p.keyspaces = make(map[string]datastore.Keyspace)

	e := p.loadKeyspaces()
	if e != nil {
		return nil, e
	}
	return p, nil
}

func (p *namespace) loadKeyspaces() (e errors.Error) {

	sb, e := newStoresKeyspace(p)
	if e != nil {
		return e
	}
	registerKeyspace(p, sb)

	pb, e := newNamespacesKeyspace(p)
	if e != nil {
		return e
	}
	registerKeyspace(p, pb)

	bk, e := newBucketsKeyspace(p, p.store.actualStore, KEYSPACE_NAME_BUCKETS)
	if e != nil {
		return e
	}
	registerKeyspace(p, bk)

	sk, e := newScopesKeyspace(p, p.store.actualStore, KEYSPACE_NAME_SCOPES, true)
	if e != nil {
		return e
	}
	registerKeyspace(p, sk)

	ask, e := newScopesKeyspace(p, p.store, KEYSPACE_NAME_ALL_SCOPES, false)
	if e != nil {
		return e
	}
	registerKeyspace(p, ask)

	kk, e := newKeyspacesKeyspace(p, p.store.actualStore, KEYSPACE_NAME_KEYSPACES, true, false)
	if e != nil {
		return e
	}
	registerKeyspace(p, kk)

	akk, e := newKeyspacesKeyspace(p, p.store, KEYSPACE_NAME_ALL_KEYSPACES, false, false)
	if e != nil {
		return e
	}
	registerKeyspace(p, akk)

	kki, e := newKeyspacesKeyspace(p, p.store.actualStore, KEYSPACE_NAME_KEYSPACES_INFO, true, true)
	if e != nil {
		return e
	}
	registerKeyspace(p, kki)

	akki, e := newKeyspacesKeyspace(p, p.store, KEYSPACE_NAME_ALL_KEYSPACES_INFO, false, true)
	if e != nil {
		return e
	}
	registerKeyspace(p, akki)

	db, e := newDualKeyspace(p)
	if e != nil {
		return e
	}
	registerKeyspace(p, db)

	ib, e := newIndexesKeyspace(p, p.store.actualStore, KEYSPACE_NAME_INDEXES, true)
	if e != nil {
		return e
	}
	registerKeyspace(p, ib)

	aib, e := newIndexesKeyspace(p, p.store, KEYSPACE_NAME_ALL_INDEXES, false)
	if e != nil {
		return e
	}
	registerKeyspace(p, aib)

	preps, e := newPreparedsKeyspace(p)
	if e != nil {
		return e
	}
	registerKeyspace(p, preps)

	funcsCache, e := newFunctionsCacheKeyspace(p)
	if e != nil {
		return e
	}
	registerKeyspace(p, funcsCache)

	funcs, e := newFunctionsKeyspace(p)
	if e != nil {
		return e
	}
	registerKeyspace(p, funcs)

	dictCache, e := newDictionaryCacheKeyspace(p, KEYSPACE_NAME_DICTIONARY_CACHE)
	if e != nil {
		return e
	}
	registerKeyspace(p, dictCache)

	dict, e := newDictionaryKeyspace(p, KEYSPACE_NAME_DICTIONARY)
	if e != nil {
		return e
	}
	registerKeyspace(p, dict)

	tasksCache, e := newTasksCacheKeyspace(p)
	if e != nil {
		return e
	}

	registerKeyspace(p, tasksCache)

	reqs, e := newRequestsKeyspace(p)
	if e != nil {
		return e
	}
	registerKeyspace(p, reqs)

	actives, e := newActiveRequestsKeyspace(p)
	if e != nil {
		return e
	}
	registerKeyspace(p, actives)

	userInfo, e := newUserInfoKeyspace(p)
	if e != nil {
		return e
	}
	registerKeyspace(p, userInfo)

	myUserInfo, e := newMyUserInfoKeyspace(p)
	if e != nil {
		return e
	}
	registerKeyspace(p, myUserInfo)

	groupInfo, e := newGroupInfoKeyspace(p)
	if e != nil {
		return e
	}
	registerKeyspace(p, groupInfo)

	bucketInfo, e := newBucketInfoKeyspace(p)
	if e != nil {
		return e
	}
	registerKeyspace(p, bucketInfo)

	nodes, e := newNodesKeyspace(p)
	if e != nil {
		return e
	}
	registerKeyspace(p, nodes)

	applicableRoles, e := newApplicableRolesKeyspace(p)
	if e != nil {
		return e
	}
	registerKeyspace(p, applicableRoles)

	transactions, e := newTransactionsKeyspace(p)
	if e != nil {
		return e
	}
	registerKeyspace(p, transactions)

	vitals, e := newVitalsKeyspace(p)
	if e != nil {
		return e
	}
	registerKeyspace(p, vitals)

	qk, e := newSequencesKeyspace(p, p.store.actualStore, KEYSPACE_NAME_SEQUENCES, true)
	if e != nil {
		return e
	}
	registerKeyspace(p, qk)

	aqk, e := newSequencesKeyspace(p, p.store.actualStore, KEYSPACE_NAME_ALL_SEQUENCES, true)
	if e != nil {
		return e
	}
	registerKeyspace(p, aqk)

	return nil
}

func (p *namespace) BucketIds() ([]string, errors.Error) {
	return datastore.NO_STRINGS, nil
}

func (p *namespace) BucketNames() ([]string, errors.Error) {
	return datastore.NO_STRINGS, nil
}

func (p *namespace) BucketById(name string) (datastore.Bucket, errors.Error) {
	return nil, errors.NewSystemNoBuckets()
}

func (p *namespace) BucketByName(name string) (datastore.Bucket, errors.Error) {
	return nil, errors.NewSystemNoBuckets()
}
