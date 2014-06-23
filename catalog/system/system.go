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
	"github.com/couchbaselabs/query/err"
)

const POOL_ID = "system"
const POOL_NAME = "system"
const BUCKET_NAME_SITES = "sites"
const BUCKET_NAME_POOLS = "pools"
const BUCKET_NAME_BUCKETS = "buckets"
const BUCKET_NAME_INDEXES = "indexes"
const BUCKET_NAME_DUAL = "dual"

type site struct {
	actualSite        catalog.Site
	systemCatalogPool *pool
}

func (s *site) Id() string {
	return s.actualSite.Id()
}

func (s *site) URL() string {
	return s.actualSite.URL()
}

func (s *site) PoolIds() ([]string, err.Error) {
	poolIds, err := s.actualSite.PoolIds()
	if err != nil {
		return nil, err
	}
	poolIds = append(poolIds, s.systemCatalogPool.Id())
	return poolIds, err
}

func (s *site) PoolNames() ([]string, err.Error) {
	poolNames, err := s.actualSite.PoolNames()
	if err != nil {
		return nil, err
	}
	poolNames = append(poolNames, s.systemCatalogPool.Name())
	return poolNames, err
}

func (s *site) PoolById(id string) (catalog.Pool, err.Error) {
	if id == POOL_ID {
		return s.systemCatalogPool, nil
	}
	return s.actualSite.PoolById(id)
}

func (s *site) PoolByName(name string) (catalog.Pool, err.Error) {
	if name == POOL_NAME {
		return s.systemCatalogPool, nil
	}
	return s.actualSite.PoolByName(name)
}

func NewSite(actualSite catalog.Site) (catalog.Site, err.Error) {
	s := &site{actualSite: actualSite}

	e := s.loadPool()
	if e != nil {
		return nil, e
	}

	return s, e
}

func (s *site) loadPool() err.Error {
	p, e := newPool(s)
	if e != nil {
		return e
	}

	s.systemCatalogPool = p
	return nil
}
