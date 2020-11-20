//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package system

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
)

type namespace struct {
	store     *store
	id        string
	name      string
	keyspaces map[string]datastore.Keyspace
}

func (p *namespace) DatastoreId() string {
	return p.store.Id()
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

func (p *namespace) Objects(preload bool) ([]datastore.Object, errors.Error) {
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
	p.keyspaces[sb.Name()] = sb

	pb, e := newNamespacesKeyspace(p)
	if e != nil {
		return e
	}
	p.keyspaces[pb.Name()] = pb

	bk, e := newBucketsKeyspace(p, p.store.actualStore, KEYSPACE_NAME_BUCKETS)
	if e != nil {
		return e
	}
	p.keyspaces[bk.Name()] = bk

	sk, e := newScopesKeyspace(p, p.store.actualStore, KEYSPACE_NAME_SCOPES, true)
	if e != nil {
		return e
	}
	p.keyspaces[sk.Name()] = sk

	ask, e := newScopesKeyspace(p, p.store, KEYSPACE_NAME_ALL_SCOPES, false)
	if e != nil {
		return e
	}
	p.keyspaces[ask.Name()] = ask

	kk, e := newKeyspacesKeyspace(p, p.store.actualStore, KEYSPACE_NAME_KEYSPACES, true)
	if e != nil {
		return e
	}
	p.keyspaces[kk.Name()] = kk

	akk, e := newKeyspacesKeyspace(p, p.store, KEYSPACE_NAME_ALL_KEYSPACES, false)
	if e != nil {
		return e
	}
	p.keyspaces[akk.Name()] = akk

	db, e := newDualKeyspace(p)
	if e != nil {
		return e
	}
	p.keyspaces[db.Name()] = db

	ib, e := newIndexesKeyspace(p, p.store.actualStore, KEYSPACE_NAME_INDEXES, true)
	if e != nil {
		return e
	}
	p.keyspaces[ib.Name()] = ib

	aib, e := newIndexesKeyspace(p, p.store, KEYSPACE_NAME_ALL_INDEXES, false)
	if e != nil {
		return e
	}
	p.keyspaces[aib.Name()] = aib

	preps, e := newPreparedsKeyspace(p)
	if e != nil {
		return e
	}
	p.keyspaces[preps.Name()] = preps

	funcsCache, e := newFunctionsCacheKeyspace(p)
	if e != nil {
		return e
	}
	p.keyspaces[funcsCache.Name()] = funcsCache

	funcs, e := newFunctionsKeyspace(p)
	if e != nil {
		return e
	}
	p.keyspaces[funcs.Name()] = funcs

	dictCache, e := newDictionaryCacheKeyspace(p, KEYSPACE_NAME_DICTIONARY_CACHE)
	if e != nil {
		return e
	}
	p.keyspaces[dictCache.Name()] = dictCache

	dict, e := newDictionaryKeyspace(p, KEYSPACE_NAME_DICTIONARY)
	if e != nil {
		return e
	}
	p.keyspaces[dict.Name()] = dict

	tasksCache, e := newTasksCacheKeyspace(p)
	if e != nil {
		return e
	}

	p.keyspaces[tasksCache.Name()] = tasksCache

	reqs, e := newRequestsKeyspace(p)
	if e != nil {
		return e
	}
	p.keyspaces[reqs.Name()] = reqs

	actives, e := newActiveRequestsKeyspace(p)
	if e != nil {
		return e
	}
	p.keyspaces[actives.Name()] = actives

	userInfo, e := newUserInfoKeyspace(p)
	if e != nil {
		return e
	}
	p.keyspaces[userInfo.Name()] = userInfo

	myUserInfo, e := newMyUserInfoKeyspace(p)
	if e != nil {
		return e
	}
	p.keyspaces[myUserInfo.Name()] = myUserInfo

	nodes, e := newNodesKeyspace(p)
	if e != nil {
		return e
	}
	p.keyspaces[nodes.Name()] = nodes

	applicableRoles, e := newApplicableRolesKeyspace(p)
	if e != nil {
		return e
	}
	p.keyspaces[applicableRoles.Name()] = applicableRoles

	transactions, e := newTransactionsKeyspace(p)
	if e != nil {
		return e
	}

	p.keyspaces[transactions.Name()] = transactions
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
