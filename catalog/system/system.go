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
	"github.com/couchbaselabs/query/catalog"
	"github.com/couchbaselabs/query/errors"
)

const NAMESPACE_ID = "system"
const NAMESPACE_NAME = "system"
const KEYSPACE_NAME_DATASTORES = "datastores"
const KEYSPACE_NAME_NAMESPACES = "namespaces"
const KEYSPACE_NAME_KEYSPACES = "keyspaces"
const KEYSPACE_NAME_INDEXES = "indexes"
const KEYSPACE_NAME_DUAL = "dual"

type datastore struct {
	actualDatastore        catalog.Datastore
	systemCatalogNamespace *namespace
}

func (s *datastore) Id() string {
	return s.actualDatastore.Id()
}

func (s *datastore) URL() string {
	return s.actualDatastore.URL()
}

func (s *datastore) NamespaceIds() ([]string, errors.Error) {
	namespaceIds, err := s.actualDatastore.NamespaceIds()
	if err != nil {
		return nil, err
	}
	namespaceIds = append(namespaceIds, s.systemCatalogNamespace.Id())
	return namespaceIds, err
}

func (s *datastore) NamespaceNames() ([]string, errors.Error) {
	namespaceNames, err := s.actualDatastore.NamespaceNames()
	if err != nil {
		return nil, err
	}
	namespaceNames = append(namespaceNames, s.systemCatalogNamespace.Name())
	return namespaceNames, err
}

func (s *datastore) NamespaceById(id string) (catalog.Namespace, errors.Error) {
	if id == NAMESPACE_ID {
		return s.systemCatalogNamespace, nil
	}
	return s.actualDatastore.NamespaceById(id)
}

func (s *datastore) NamespaceByName(name string) (catalog.Namespace, errors.Error) {
	if name == NAMESPACE_NAME {
		return s.systemCatalogNamespace, nil
	}
	return s.actualDatastore.NamespaceByName(name)
}

func NewDatastore(actualDatastore catalog.Datastore) (catalog.Datastore, errors.Error) {
	s := &datastore{actualDatastore: actualDatastore}

	e := s.loadNamespace()
	if e != nil {
		return nil, e
	}

	return s, e
}

func (s *datastore) loadNamespace() errors.Error {
	p, e := newNamespace(s)
	if e != nil {
		return e
	}

	s.systemCatalogNamespace = p
	return nil
}
