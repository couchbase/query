//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

Package catalog provides a common catalog abstraction over storage
engines, such as Couchbase server, cloud, mobile, file, 3rd-party
databases and storage engines, etc.

*/
package catalog

import (
	"github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

// log channel for the catalog lifecycle
const CHANNEL = "CATALOG"

// Site represents a cluster or single-node server.
type Site interface {
	Id() string
	Url() string
	PoolIds() ([]string, err.Error)
	PoolNames() ([]string, err.Error)
	PoolById(id string) (Pool, err.Error)
	PoolByName(name string) (Pool, err.Error)
}

// Pool represents a logical authentication, query, and resource
// allocation boundary, as well as a grouping of buckets.
type Pool interface {
	SiteId() string
	Id() string
	Name() string
	BucketIds() ([]string, err.Error)
	BucketNames() ([]string, err.Error)
	BucketById(name string) (Bucket, err.Error)
	BucketByName(name string) (Bucket, err.Error)
}

// Bucket is a collection of key-value entries (typically
// key-document, but not always).
type Bucket interface {
	PoolId() string
	Id() string
	Name() string
	Count() (int64, err.Error)
	IndexIds() ([]string, err.Error)
	IndexNames() ([]string, err.Error)
	IndexById(id string) (Index, err.Error)
	IndexByName(name string) (Index, err.Error)
	IndexByPrimary() (PrimaryIndex, err.Error) // Returns the server-recommended primary index
	Indexes() ([]Index, err.Error)
	CreatePrimaryIndex() (PrimaryIndex, err.Error)
	CreateIndex(name string, equal, ranje expression.CompositeExpression, using IndexType) (Index, err.Error)

	// Used by both SELECT and DML statements
	Fetch(keys []string) ([]value.Value, err.Error)
	FetchOne(key string) (value.Value, err.Error)

	// Used by DML statements
	// For all these methods, nil input keys are replaced with auto-generated keys
	Insert(inserts []Pair) ([]string, err.Error)
	Update(updates []Pair) err.Error
	Upsert(upserts []Pair) ([]string, err.Error)
	Delete(deletes []string) err.Error

	Release()
}

type Pair struct {
	Key   string
	Value value.Value
}
