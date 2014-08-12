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
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/errors"
)

const NAMESPACE_ID = "#system"
const NAMESPACE_NAME = "#system"
const KEYSPACE_NAME_DATASTORES = "datastores"
const KEYSPACE_NAME_NAMESPACES = "namespaces"
const KEYSPACE_NAME_KEYSPACES = "keyspaces"
const KEYSPACE_NAME_INDEXES = "indexes"
const KEYSPACE_NAME_DUAL = "dual"

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
