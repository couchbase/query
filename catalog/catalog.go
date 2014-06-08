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

The logical hierarchy for the query language is site -> pool -> bucket
-> document. Indexes are also attached to buckets.

TODO: This hierarchy should be revisited and aligned with long-term
plans before query Beta / GA.

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
	Id() string                               // Id of this site
	URL() string                              // URL to this site
	PoolIds() ([]string, err.Error)           // Ids of the pools contained in this site
	PoolNames() ([]string, err.Error)         // Names of the pools contained in this site
	PoolById(id string) (Pool, err.Error)     // Find a pool in this site using the pool's Id
	PoolByName(name string) (Pool, err.Error) // Find a pool in this site using the pool's name
}

// Pool represents a logical boundary that is within a site and above
// a bucket. In the query language, a pool is only used as a namespace
// to qualify bucket names. No assumptions are made about pools and
// isolation, resource management, or any other concerns.
type Pool interface {
	SiteId() string                               // Id of the site that contains this pool
	Id() string                                   // Id of this pool
	Name() string                                 // Name of this pool
	BucketIds() ([]string, err.Error)             // Ids of the buckets contained in this pool
	BucketNames() ([]string, err.Error)           // Names of the buckets contained in this pool
	BucketById(name string) (Bucket, err.Error)   // Find a bucket in this pool using the bucket's id
	BucketByName(name string) (Bucket, err.Error) // Find a bucket in this pool using the bucket's name
}

// Bucket is a collection of key-value entries (typically
// key-document, but also key-counter, key-blob, etc.).
type Bucket interface {
	PoolId() string                                // Id of the pool that contains this bucket
	Id() string                                    // Id of this bucket
	Name() string                                  // Name of this bucket
	Count() (int64, err.Error)                     // Number of key-value entries in this bucket
	IndexIds() ([]string, err.Error)               // Ids of the indexes defined on this bucket
	IndexNames() ([]string, err.Error)             // Names of the indexes defined on this bucket
	IndexById(id string) (Index, err.Error)        // Find an index on this bucket using the index's id
	IndexByName(name string) (Index, err.Error)    // Find an index on this bucket using the index's name
	IndexByPrimary() (PrimaryIndex, err.Error)     // Returns the server-recommended primary index
	Indexes() ([]Index, err.Error)                 // Returns all the indexes defined on this bucket
	CreatePrimaryIndex() (PrimaryIndex, err.Error) // Create or return a primary index on this bucket

	CreateIndex(name string, equalKey, rangeKey expression.Expressions, using IndexType) (Index, err.Error) // Create a secondary index on this bucket

	// Used by both SELECT and DML statements
	Fetch(keys []string) ([]Pair, err.Error)      // Bulk key-value fetch from this bucket
	FetchOne(key string) (value.Value, err.Error) // Single key-value fetch from this bucket

	// Used by DML statements
	// For insert and upsert, nil input keys are replaced with auto-generated keys
	Insert(inserts []Pair) ([]Pair, err.Error) // Bulk key-value insert into this bucket
	Update(updates []Pair) ([]Pair, err.Error) // Bulk key-value updates into this bucket
	Upsert(upserts []Pair) ([]Pair, err.Error) // Bulk key-value upserts into this bucket
	Delete(deletes []string) err.Error         // Bulk key-value deletes from this bucket

	Release() // Release any query engine resources held by this object
}

// Key-value pair
type Pair struct {
	Key   string
	Value value.Value
}
