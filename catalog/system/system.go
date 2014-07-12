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
const KEYSPACE_NAME_SITES = "sites"
const KEYSPACE_NAME_NAMESPACES = "namespaces"
const KEYSPACE_NAME_KEYSPACES = "keyspaces"
const KEYSPACE_NAME_INDEXES = "indexes"
const KEYSPACE_NAME_DUAL = "dual"

type site struct {
	actualSite             catalog.Site
	systemCatalogNamespace *namespace
}

func (s *site) Id() string {
	return s.actualSite.Id()
}

func (s *site) URL() string {
	return s.actualSite.URL()
}

func (s *site) NamespaceIds() ([]string, errors.Error) {
	namespaceIds, err := s.actualSite.NamespaceIds()
	if err != nil {
		return nil, err
	}
	namespaceIds = append(namespaceIds, s.systemCatalogNamespace.Id())
	return namespaceIds, err
}

func (s *site) NamespaceNames() ([]string, errors.Error) {
	namespaceNames, err := s.actualSite.NamespaceNames()
	if err != nil {
		return nil, err
	}
	namespaceNames = append(namespaceNames, s.systemCatalogNamespace.Name())
	return namespaceNames, err
}

func (s *site) NamespaceById(id string) (catalog.Namespace, errors.Error) {
	if id == NAMESPACE_ID {
		return s.systemCatalogNamespace, nil
	}
	return s.actualSite.NamespaceById(id)
}

func (s *site) NamespaceByName(name string) (catalog.Namespace, errors.Error) {
	if name == NAMESPACE_NAME {
		return s.systemCatalogNamespace, nil
	}
	return s.actualSite.NamespaceByName(name)
}

func NewSite(actualSite catalog.Site) (catalog.Site, errors.Error) {
	s := &site{actualSite: actualSite}

	e := s.loadNamespace()
	if e != nil {
		return nil, e
	}

	return s, e
}

func (s *site) loadNamespace() errors.Error {
	p, e := newNamespace(s)
	if e != nil {
		return e
	}

	s.systemCatalogNamespace = p
	return nil
}
