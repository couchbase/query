//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

Package catalog provides a common catalog abstraction over all storage
engines, such as Couchbase server, cloud, mobile, file, 3rd-party
databases and storage engines, etc.

*/

package catalog

import (
	"github.com/couchbaselabs/query/ast"
	"github.com/couchbaselabs/query/err"
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
	CreateIndex(name string, match MatchKey, range_ RangeKey, using IndexType) (Index, err.Error)

	// Used by both SELECT and DML statements
	Fetch(keys []string) (map[string]value.Value, err.Error)
	FetchOne(key string) (value.Value, err.Error)

	// Used by DML statements
	Insert(keys []string, values []value.Value) ([]string, err.Error) // nil input keys are replaced with auto-generated keys
	Update(keys []string, values []value.Value) err.Error
	Upsert(keys []string, values []value.Value) ([]string, err.Error) // nil input keys are replaced with auto-generated keys
	Delete(keys []string) err.Error

	Release()
}

type IndexType string

const (
	UNSPECIFIED IndexType = "unspecified" // used by non-view primary_indexes
	VIEW        IndexType = "view"
)

type MatchKey []ast.Expression
type RangeKey []*RangePart

type RangePart struct {
	Expr ast.Expression
	Dir  Direction
}

// Index is the base type for all indexes.
type Index interface {
	BucketId() string
	Id() string
	Name() string
	Type() IndexType
	Match() MatchKey
	Range() RangeKey
	Drop() err.Error // PrimaryIndexes cannot be dropped
}

type IndexEntry struct {
	EntryKey   value.CompositeValue
	PrimaryKey string
}

type EntryChannel chan *IndexEntry

type IndexResponse struct {
	Chan     EntryChannel
	Warnchan err.ErrorChannel
	Errchan  err.ErrorChannel
}

// PrimaryIndex represents primary key indexes.
type PrimaryIndex interface {
	MatchIndex
	BucketScan(limit int64, response *IndexResponse)
}

// MatchIndexes support equality matching.
type MatchIndex interface {
	Index
	MatchScan(match value.CompositeValue, limit int64, response *IndexResponse)
	MatchCount(match value.CompositeValue, response *IndexResponse)
}

// Direction represents ASC and DESC
type Direction int

const (
	ASC  Direction = 1
	DESC           = 2
)

// Inclusion controls how the boundary values of a range are treated.
type RangeInclusion int

const (
	NEITHER RangeInclusion = iota
	LOW
	HIGH
	BOTH
)

type Range struct {
	Low       value.CompositeValue
	High      value.CompositeValue
	Inclusion RangeInclusion
}

// RangeIndexes support unrestricted range queries.
type RangeIndex interface {
	Index
	RangeStats(range_ *Range) (RangeStatistics, err.Error)
	RangeScan(range_ *Range, limit int64, response *IndexResponse)
	RangeCount(range_ *Range, response *IndexResponse)
	RangeCandidateMins(range_ *Range, response *IndexResponse)  // Anywhere from single Min value to RangeScan()
	RangeCandidateMaxes(range_ *Range, response *IndexResponse) // Anywhere from single Max value to RangeScan()
}

// DualIndexes support restricted range queries.
type DualIndex interface {
	Index
	DualStats(match value.CompositeValue, range_ *Range) (RangeStatistics, err.Error)
	DualScan(match value.CompositeValue, range_ *Range, limit int64, response *IndexResponse)
	DualCount(match value.CompositeValue, range_ *Range, response *IndexResponse)
	DualCandidateMins(match value.CompositeValue, range_ *Range, response *IndexResponse)  // Anywhere from single Min value to DualScan()
	DualCandidateMaxes(match value.CompositeValue, range_ *Range, response *IndexResponse) // Anywhere from single Max value to DualScan()
}

// RangeStatistics captures statistics for an index range.
type RangeStatistics interface {
	Count() (int64, err.Error)
	Min() (value.Value, err.Error)
	Max() (value.Value, err.Error)
	DistinctCount(int64, err.Error)
	Bins() ([]RangeStatistics, err.Error)
}
