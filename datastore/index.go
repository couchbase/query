//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package datastore

import (
	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type IndexType string

const (
	DEFAULT IndexType = "default" // default may vary per backend
	VIEW    IndexType = "view"    // view index
	GSI     IndexType = "gsi"     // global secondary index
	FTS     IndexType = "fts"     // full text index
	SYSTEM  IndexType = "system"  // system keyspace indexes
)

const (
	INDEX_API_1   = 1
	INDEX_API_2   = 2
	INDEX_API_3   = 3
	INDEX_API_MIN = INDEX_API_1
	INDEX_API_MAX = INDEX_API_3
)

type Indexer interface {
	KeyspaceId() string                                                                       // Id of the keyspace to which this indexer belongs
	Name() IndexType                                                                          // Unique within a Keyspace.
	IndexIds() ([]string, errors.Error)                                                       // Ids of the indexes defined on this keyspace
	IndexNames() ([]string, errors.Error)                                                     // Names of the indexes defined on this keyspace
	IndexById(id string) (Index, errors.Error)                                                // Find an index on this keyspace using the index's id
	IndexByName(name string) (Index, errors.Error)                                            // Find an index on this keyspace using the index's name
	PrimaryIndexes() ([]PrimaryIndex, errors.Error)                                           // Returns the server-recommended primary index
	Indexes() ([]Index, errors.Error)                                                         // Returns all the indexes defined on this keyspace
	CreatePrimaryIndex(requestId, name string, with value.Value) (PrimaryIndex, errors.Error) // Create or return a primary index on this keyspace
	CreateIndex(requestId, name string, seekKey, rangeKey expression.Expressions,             // Create a secondary index on this keyspace
		where expression.Expression, with value.Value) (Index, errors.Error)
	BuildIndexes(requestId string, name ...string) errors.Error // Build indexes that were deferred at creation
	Refresh() errors.Error                                      // Refresh list of indexes from metadata
	MetadataVersion() uint64                                    // Meta data change counter
	SetLogLevel(level logging.Level)                            // Set log level for in-process logging
}

type IndexConfig interface {
	SetConfig(KVal map[string]interface{}) errors.Error
	SetParam(name string, val interface{}) errors.Error
}

type IndexState string

const (
	DEFERRED IndexState = "deferred" // The index has not been built
	BUILDING IndexState = "building" // The index is being built or rebuilt
	PENDING  IndexState = "pending"  // The index is in progress but is not yet ready for use
	ONLINE   IndexState = "online"   // The index is available for use
	OFFLINE  IndexState = "offline"  // The index requires manual intervention
	ABRIDGED IndexState = "abridged" // The index is missing some entries, e.g. due to size limits
)

func (indexState IndexState) String() string {
	return string(indexState)
}

type ScanConsistency string

const (
	UNBOUNDED ScanConsistency = "unbounded"
	SCAN_PLUS ScanConsistency = "scan_plus"
	AT_PLUS   ScanConsistency = "at_plus"
)

type Spans []*Span

type Span struct {
	Seek  value.Values
	Range Range
}

type Ranges []*Range

type Range struct {
	Low       value.Values
	High      value.Values
	Inclusion Inclusion
}

// Inclusion controls how the boundary values of a range are treated.
type Inclusion int

const (
	NEITHER Inclusion = 0x00
	LOW               = 0x01
	HIGH              = 0x01 << 1
	BOTH              = LOW | HIGH
)

type Indexes []Index

/*
Index is the base type for indexes, which may be distributed.
*/
type Index interface {
	KeyspaceId() string                                                 // Id of the keyspace to which this index belongs
	Id() string                                                         // Id of this index
	Name() string                                                       // Name of this index
	Type() IndexType                                                    // Type of this index
	Indexer() Indexer                                                   // Indexer this index hangs from
	SeekKey() expression.Expressions                                    // Seek keys
	RangeKey() expression.Expressions                                   // Range keys
	Condition() expression.Expression                                   // Condition, if any
	IsPrimary() bool                                                    // Is this a primary index
	State() (state IndexState, msg string, err errors.Error)            // Obtain state of this index
	Statistics(requestId string, span *Span) (Statistics, errors.Error) // Obtain statistics for this index
	Drop(requestId string) errors.Error                                 // Drop / delete this index

	// Perform a scan on this index. Distinct and limit are hints.
	Scan(requestId string, span *Span, distinct bool, limit int64, cons ScanConsistency,
		vector timestamp.Vector, conn *IndexConnection)
}

type CountIndex interface {
	Index

	// Perform a count on index
	Count(span *Span, cons ScanConsistency, vector timestamp.Vector) (int64, errors.Error)
}

/*
PrimaryIndex represents primary key indexes.
*/
type PrimaryIndex interface {
	Index

	// Perform a scan of all the entries in this index
	ScanEntries(requestId string, limit int64, cons ScanConsistency,
		vector timestamp.Vector, conn *IndexConnection)
}

type SizedIndex interface {
	Index

	SizeFromStatistics(requestId string) (int64, errors.Error)
}

////////////////////////////////////////////////////////////////////////
//
// Index API2 introduced in Spock for more efficient index pushdowns.
//
////////////////////////////////////////////////////////////////////////

type IndexKeys []*IndexKey

type IndexKey struct {
	Expr expression.Expression
	Desc bool
}

type Indexer2 interface {
	Indexer

	// Create a secondary index on this keyspace
	CreateIndex2(requestId, name string, seekKey expression.Expressions,
		rangeKey IndexKeys, where expression.Expression, with value.Value) (
		Index, errors.Error)
}

type Spans2 []*Span2

type Span2 struct {
	Seek   value.Values
	Ranges Ranges2
}

type Ranges2 []*Range2

type Range2 struct {
	Low       value.Value
	High      value.Value
	Inclusion Inclusion
}

type IndexProjection struct {
	EntryKeys []int // >= 0 and < length(indexKeys) project indexKey at that position
	// >= len(indexKeys)  Project matching EntryKeyId in  Groups or Aggregates
	PrimaryKey bool
}

type Index2 interface {
	Index

	RangeKey2() IndexKeys // Range keys

	// Perform a scan on this index. distinctAfterProjection and limit are hints.
	Scan2(requestId string, spans Spans2, reverse, distinctAfterProjection,
		ordered bool, projection *IndexProjection, offset, limit int64,
		cons ScanConsistency, vector timestamp.Vector, conn *IndexConnection)
}

type CountIndex2 interface {
	CountIndex

	// Perform a count on index
	Count2(requestId string, spans Spans2, cons ScanConsistency, vector timestamp.Vector) (
		int64, errors.Error)

	// Can perform count distinct
	CanCountDistinct() bool

	// Perform a count distinct on index
	CountDistinct(requestId string, spans Spans2, cons ScanConsistency, vector timestamp.Vector) (
		int64, errors.Error)
}

type StreamingDistinctIndex interface {
	Index2

	// Perform a streaming distinct scan on this index.  The
	// results must be distinct across all the returned
	// keys. secondaryKeys specifies the projection. secondaryKeys
	// is a leading subset of the index keys.
	ScanStreamingDistinct(requestId string, spans Spans2, reverse, ordered bool,
		secondaryKeys int, offset, limit int64, cons ScanConsistency,
		vector timestamp.Vector, conn *IndexConnection)
}

////////////////////////////////////////////////////////////////////////
//
// End of Index API2.
//
////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////
//
// Index API3 introduced in Vulcan for Index GROUP and Aggregates
//
////////////////////////////////////////////////////////////////////////

type AggregateType string

const (
	AGG_MIN    AggregateType = "MIN"
	AGG_MAX    AggregateType = "MAX"
	AGG_SUM    AggregateType = "SUM"
	AGG_COUNT  AggregateType = "COUNT"
	AGG_COUNTN AggregateType = "COUNTN" // Count only when argument is numeric. Required for AVG
	AGG_ARRAY  AggregateType = "ARRAY_AGG"
	AGG_AVG    AggregateType = "AVG"
)

type IndexGroupKeys []*IndexGroupKey
type IndexAggregates []*IndexAggregate

type IndexGroupKey struct {
	EntryKeyId int                   // Id that can be used in IndexProjection
	KeyPos     int                   // >=0 means use expr at index key position otherwise use Expr
	Expr       expression.Expression // group expression
}

type IndexAggregate struct {
	Operation  AggregateType         // Aggregate operation
	EntryKeyId int                   // Id that can be used in IndexProjection
	KeyPos     int                   // >=0 means use expr at index key position otherwise use Expr
	Expr       expression.Expression // Aggregate expression
	Distinct   bool                  // Distinct on aggregate expression.
	// Aggregate only on Distinct values with in the group
}

type IndexGroupAggregates struct {
	Name               string          // name of the index aggregate
	Group              IndexGroupKeys  // group keys, nil means no group by
	Aggregates         IndexAggregates // aggregates with in the group, nil means no aggregates
	DependsOnIndexKeys []int           // GROUP and Aggregates Depends on List of index keys positions
	IndexKeyNames      []string        // Index key names used in expressions
	OneForPrimaryKey   bool            // Leading Key is ALL ARRAY index key and equality span conside one per META().id
	AllowPartialAggr   bool            // Partial aggregation are allowed
}

type IndexKeyOrders []*IndexKeyOrder

type IndexKeyOrder struct {
	KeyPos int
	Desc   bool
}

type PartitionType string

const (
	NO_PARTITION   PartitionType = ""
	HASH_PARTITION PartitionType = "HASH"
)

type IndexPartition struct {
	Strategy PartitionType
	Exprs    expression.Expressions
}

type Index3 interface {
	Index2

	CreateAggregate(requestId string, groupAggs *IndexGroupAggregates, with value.Value) errors.Error
	DropAggregate(requestId, name string) errors.Error
	Aggregates() ([]IndexGroupAggregates, errors.Error)
	PartitionKeys() (*IndexPartition, errors.Error) // Partition Info

	Scan3(requestId string, spans Spans2, reverse, distinctAfterProjection bool,
		projection *IndexProjection, offset, limit int64,
		groupAggs *IndexGroupAggregates, indexOrders IndexKeyOrders,
		cons ScanConsistency, vector timestamp.Vector, conn *IndexConnection)

	Alter(requestId string, with value.Value) (Index, errors.Error)
}

type PrimaryIndex3 interface {
	Index3

	// Perform a scan of all the entries in this index
	ScanEntries3(requestId string, projection *IndexProjection, offset, limit int64,
		groupAggs *IndexGroupAggregates, indexOrders IndexKeyOrders, cons ScanConsistency,
		vector timestamp.Vector, conn *IndexConnection)
}

type Indexer3 interface {
	Indexer2

	// Create a secondary index on this keyspace
	CreateIndex3(requestId, name string, rangeKey IndexKeys, indexPartition *IndexPartition,
		where expression.Expression, with value.Value) (Index, errors.Error)

	// Create a primary index on this keyspace
	CreatePrimaryIndex3(requestId, name string, indexPartition *IndexPartition,
		with value.Value) (PrimaryIndex, errors.Error)
}

////////////////////////////////////////////////////////////////////////
//
// End of Index API3.
//
////////////////////////////////////////////////////////////////////////

type IndexEntry struct {
	EntryKey   value.Values
	PrimaryKey string
}

type EntryChannel chan *IndexEntry

// Statistics captures statistics for a range.
// - it may return heuristics and/or outdated values.
// - query shall not depend on the accuracy of this statistics.
// - primarily intended for optimizer's consumption.
type Statistics interface {
	Count() (int64, errors.Error)
	Min() (value.Values, errors.Error)
	Max() (value.Values, errors.Error)
	DistinctCount() (int64, errors.Error)
	Bins() ([]Statistics, errors.Error)
}

type IndexConnection struct {
	entryChannel EntryChannel // Closed by the index when the scan is completed or aborted.
	stopChannel  StopChannel  // Notifies index to stop scanning. Never closed, just garbage-collected.
	context      Context
	timeout      bool
	primary      bool
}

const _ENTRY_CAP = 512 // Default index scan request size

// Context cannot be nil
func NewIndexConnection(context Context) *IndexConnection {
	size := context.GetScanCap()
	if size <= 0 {
		size = _ENTRY_CAP
	}

	return &IndexConnection{
		entryChannel: make(EntryChannel, size),
		stopChannel:  make(StopChannel, 1),
		context:      context,
	}
}

var scanCap atomic.AlignedInt64

func SetScanCap(scap int64) {
	if scap < 1 {
		scap = _ENTRY_CAP
	}
	atomic.StoreInt64(&scanCap, scap)
}

func GetScanCap() int64 {
	scap := atomic.LoadInt64(&scanCap)
	if scap > 0 {
		return scap
	} else {
		return _ENTRY_CAP
	}
}

func NewSizedIndexConnection(size int64, context Context) (*IndexConnection, errors.Error) {
	if size <= 0 {
		return nil, errors.NewIndexScanSizeError(size)
	}

	maxSize := int64(GetScanCap())
	if (maxSize > 0) && (size > maxSize) {
		size = maxSize
	}

	return &IndexConnection{
		entryChannel: make(EntryChannel, size),
		stopChannel:  make(StopChannel, 1),
		context:      context,
	}, nil
}

func (this *IndexConnection) EntryChannel() EntryChannel {
	return this.entryChannel
}

func (this *IndexConnection) StopChannel() StopChannel {
	return this.stopChannel
}

func (this *IndexConnection) Fatal(err errors.Error) {
	this.context.Fatal(err)
}

func (this *IndexConnection) Error(err errors.Error) {
	if this.primary && err.Code() == errors.INDEX_SCAN_TIMEOUT {
		this.timeout = true
		return
	}
	this.context.Error(err)
}

func (this *IndexConnection) Warning(wrn errors.Error) {
	this.context.Warning(wrn)
}

func (this *IndexConnection) SetPrimary() {
	this.primary = true
}

func (this *IndexConnection) Timeout() bool {
	return this.timeout
}
