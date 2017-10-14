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
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

const NAMESPACE_ID = "#system"
const NAMESPACE_NAME = "#system"
const KEYSPACE_NAME_DATASTORES = "datastores"
const KEYSPACE_NAME_NAMESPACES = "namespaces"
const KEYSPACE_NAME_KEYSPACES = "keyspaces"
const KEYSPACE_NAME_INDEXES = "indexes"
const KEYSPACE_NAME_DUAL = "dual"
const KEYSPACE_NAME_PREPAREDS = "prepareds"
const KEYSPACE_NAME_REQUESTS = "completed_requests"
const KEYSPACE_NAME_ACTIVE = "active_requests"

type store struct {
	actualStore              datastore.Datastore
	systemDatastoreNamespace *namespace
}

func (s *store) Id() string {
	return s.actualStore.Id()
}

func (s *store) URL() string {
	return s.actualStore.URL()
}

func (s *store) NamespaceIds() ([]string, errors.Error) {
	namespaceIds, err := s.actualStore.NamespaceIds()
	if err != nil {
		return nil, err
	}
	namespaceIds = append(namespaceIds, s.systemDatastoreNamespace.Id())
	return namespaceIds, err
}

func (s *store) NamespaceNames() ([]string, errors.Error) {
	namespaceNames, err := s.actualStore.NamespaceNames()
	if err != nil {
		return nil, err
	}
	namespaceNames = append(namespaceNames, s.systemDatastoreNamespace.Name())
	return namespaceNames, err
}

func (s *store) NamespaceById(id string) (datastore.Namespace, errors.Error) {
	if id == NAMESPACE_ID {
		return s.systemDatastoreNamespace, nil
	}
	return s.actualStore.NamespaceById(id)
}

func (s *store) NamespaceByName(name string) (datastore.Namespace, errors.Error) {
	if name == NAMESPACE_NAME {
		return s.systemDatastoreNamespace, nil
	}
	return s.actualStore.NamespaceByName(name)
}

func (s *store) Authorize(datastore.Privileges, datastore.Credentials) errors.Error {
	logging.Logf(logging.INFO, "System authorize")
	return nil
}

func (s *store) SetLogLevel(level logging.Level) {
	// No-op. Uses query engine logger.
}

func (s *store) Inferencer(name datastore.InferenceType) (datastore.Inferencer, errors.Error) {
	return nil, errors.NewOtherNotImplementedError(nil, "INFER")
}

func (s *store) Inferencers() ([]datastore.Inferencer, errors.Error) {
	return nil, errors.NewOtherNotImplementedError(nil, "INFER")
}

func NewDatastore(actualStore datastore.Datastore) (datastore.Datastore, errors.Error) {
	s := &store{actualStore: actualStore}

	e := s.loadNamespace()
	if e != nil {
		return nil, e
	}

	return s, e
}

func (s *store) loadNamespace() errors.Error {
	p, e := newNamespace(s)
	if e != nil {
		return e
	}

	s.systemDatastoreNamespace = p
	return nil
}

// Interface to access remote system data over various protocols
type SystemRemoteAccess interface {
	MakeKey(node string, key string) string // given a node and a local key, produce a cluster key
	SplitKey(key string) (string, string)   // given a cluster key, return node and local key

	GetRemoteKeys(nodes []string, endpoint string, keyFn func(id string),
		warnFn func(warn errors.Error)) // collect cluster keys for a keyspace from a set of remote nodes
	GetRemoteDoc(node string, key string, endpoint string, command string,
		docFn func(doc map[string]interface{}),
		warnFn func(warn errors.Error)) // collect a document for a keyspace from a remote node
	WhoAmI() string // local node name, if known
}

var _REMOTEACCESS SystemRemoteAccess = NewSystemRemoteStub()

func SetRemoteAccess(remoteAccess SystemRemoteAccess) {
}

type systemIndexer struct {
	keyspace datastore.Keyspace
	primary  datastore.PrimaryIndex
	indexes  map[string]datastore.Index
}

func newSystemIndexer(keyspace datastore.Keyspace, primary datastore.PrimaryIndex) datastore.Indexer {
	return &systemIndexer{
		keyspace: keyspace,
		primary:  primary,
		indexes:  map[string]datastore.Index{"#primary": primary},
	}
}

func (si *systemIndexer) KeyspaceId() string {
	return si.keyspace.Id()
}

func (si *systemIndexer) Name() datastore.IndexType {
	return datastore.DEFAULT
}

func (si *systemIndexer) IndexIds() ([]string, errors.Error) {
	rv := make([]string, 0, len(si.indexes))
	for name, _ := range si.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (si *systemIndexer) IndexNames() ([]string, errors.Error) {
	rv := make([]string, 0, len(si.indexes))
	for name, _ := range si.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (si *systemIndexer) IndexById(id string) (datastore.Index, errors.Error) {
	return si.IndexByName(id)
}

func (si *systemIndexer) IndexByName(name string) (datastore.Index, errors.Error) {
	index, ok := si.indexes[name]
	if !ok {
		return nil, errors.NewSystemIdxNotFoundError(nil, name)
	}
	return index, nil
}

func (si *systemIndexer) PrimaryIndexes() ([]datastore.PrimaryIndex, errors.Error) {
	return []datastore.PrimaryIndex{si.primary}, nil
}

func (si *systemIndexer) Indexes() ([]datastore.Index, errors.Error) {
	return []datastore.Index{si.primary}, nil
}

func (si *systemIndexer) CreatePrimaryIndex(requestId, name string, with value.Value) (
	datastore.PrimaryIndex, errors.Error) {
	return nil, errors.NewSystemNotSupportedError(nil, "CREATE PRIMARY INDEX is not supported for system datastore.")
}

func (mi *systemIndexer) CreateIndex(requestId, name string, seekKey, rangeKey expression.Expressions,
	where expression.Expression, with value.Value) (datastore.Index, errors.Error) {
	return nil, errors.NewSystemNotSupportedError(nil, "CREATE INDEX is not supported for system datastore.")
}

func (mi *systemIndexer) BuildIndexes(requestId string, names ...string) errors.Error {
	return errors.NewSystemNotSupportedError(nil, "BUILD INDEXES is not supported for system datastore.")
}

func (mi *systemIndexer) Refresh() errors.Error {
	return nil
}

func (mi *systemIndexer) SetLogLevel(level logging.Level) {
	// No-op, uses query engine logger
}
