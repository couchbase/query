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

type pool struct {
	site    *site
	id      string
	name    string
	buckets map[string]catalog.Bucket
}

func (p *pool) SiteId() string {
	return p.site.Id()
}

func (p *pool) Id() string {
	return p.id
}

func (p *pool) Name() string {
	return p.name
}

func (p *pool) BucketIds() ([]string, err.Error) {
	return p.BucketNames()
}

func (p *pool) BucketNames() ([]string, err.Error) {
	rv := make([]string, len(p.buckets))
	i := 0
	for k, _ := range p.buckets {
		rv[i] = k
		i = i + 1
	}
	return rv, nil
}

func (p *pool) BucketById(id string) (catalog.Bucket, err.Error) {
	return p.BucketByName(id)
}

func (p *pool) BucketByName(name string) (catalog.Bucket, err.Error) {
	b, ok := p.buckets[name]
	if !ok {
		return nil, err.NewError(nil, "Bucket "+name+" not found.")
	}

	return b, nil
}

// newPool creates a new pool.
func newPool(s *site) (*pool, err.Error) {
	p := new(pool)
	p.site = s
	p.id = POOL_ID
	p.name = POOL_NAME
	p.buckets = make(map[string]catalog.Bucket)

	e := p.loadBuckets()
	if e != nil {
		return nil, e
	}
	return p, nil
}

func (p *pool) loadBuckets() (e err.Error) {

	sb, e := newSitesBucket(p)
	if e != nil {
		return e
	}
	p.buckets[sb.Name()] = sb

	pb, e := newPoolsBucket(p)
	if e != nil {
		return e
	}
	p.buckets[pb.Name()] = pb

	bb, e := newBucketsBucket(p)
	if e != nil {
		return e
	}
	p.buckets[bb.Name()] = bb

	db, e := newDualBucket(p)
	if e != nil {
		return e
	}
	p.buckets[db.Name()] = db

	ib, e := newIndexesBucket(p)
	if e != nil {
		return e
	}
	p.buckets[ib.Name()] = ib

	return nil
}
