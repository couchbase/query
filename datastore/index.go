//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package datastore

import (
	"strings"
	"time"

	"github.com/couchbase/cbauth"
	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type IndexType string

const (
	DEFAULT  IndexType = "default"        // default may vary per backend
	VIEW     IndexType = "view"           // view index
	GSI      IndexType = "gsi"            // global secondary index
	FTS      IndexType = "fts"            // full text index
	SYSTEM   IndexType = "system"         // system keyspace indexes
	VIRTUAL  IndexType = "virtual"        // The index is built as a virtual index
	SEQ_SCAN IndexType = "sequentialscan" // sequential scan
)

const (
	INDEX_API_1   = 1
	INDEX_API_2   = 2
	INDEX_API_3   = 3
	INDEX_API_4   = 4
	INDEX_API_MIN = INDEX_API_1
	INDEX_API_MAX = INDEX_API_4
)

type Indexer interface {
	BucketId() string
	ScopeId() string
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

	SetConnectionSecurityConfig(connSecConfig *ConnectionSecurityConfig) // Update TLS or node-to-node encryption settings.
}

type ConnectionSecurityConfig struct {
	TLSConfig               cbauth.TLSConfig
	ClusterEncryptionConfig cbauth.ClusterEncryptionConfig
	CAFile                  string
	CertFile                string
	KeyFile                 string
}

type IndexConfig interface {
	SetConfig(KVal map[string]interface{}) errors.Error
	SetParam(name string, val interface{}) errors.Error
}

type GsiIndexer interface {
	Indexer
	GetGsiClientConfig() map[string]interface{}
}

type IndexState string

const (
	DEFERRED  IndexState = "deferred"               // The index has not been built
	BUILDING  IndexState = "building"               // The index is being built or rebuilt
	PENDING   IndexState = "pending"                // The index is in progress but is not yet ready for use
	ONLINE    IndexState = "online"                 // The index is available for use
	OFFLINE   IndexState = "offline"                // The index requires manual intervention
	ABRIDGED  IndexState = "abridged"               // The index is missing some entries, e.g. due to size limits
	SCHEDULED IndexState = "scheduled for creation" // Index is scheduled for creation
)

func (indexState IndexState) String() string {
	return string(indexState)
}

type ScanConsistency string

const (
	NOT_SET   ScanConsistency = "not_set"
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
	//	BucketId() string
	//	ScopeId() string
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

type CollectionIndex interface {
	Index

	BucketId() string
	ScopeId() string
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
type IkAttributes int

type IndexKey struct {
	Expr       expression.Expression
	Attributes IkAttributes
}

const (
	IK_NONE    IkAttributes = 0x00
	IK_DESC                 = 0x01
	IK_MISSING              = 0x01 << 1
)

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
	AGG_MIN        AggregateType = "MIN"
	AGG_MAX        AggregateType = "MAX"
	AGG_SUM        AggregateType = "SUM"
	AGG_COUNT      AggregateType = "COUNT"
	AGG_COUNTN     AggregateType = "COUNTN" // Count only when argument is numeric. Required for AVG
	AGG_ARRAY      AggregateType = "ARRAY_AGG"
	AGG_AVG        AggregateType = "AVG"
	AGG_MEDIAN     AggregateType = "MEDIAN"
	AGG_STDDEV     AggregateType = "STDDEV"
	AGG_STDDEVPOP  AggregateType = "STDDEV_POP"
	AGG_STDDEVSAMP AggregateType = "STDDEV_SAMP"
	AGG_VARIANCE   AggregateType = "VARIANCE"
	AGG_VARSAMP    AggregateType = "VAR_SAMP"
	AGG_VARPOP     AggregateType = "VAR_POP"
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

////////////////////////////////////////////////////////////////////////
//
// Index API4 introduced in Mad-Hatter for Index Statistics for CBO
//
////////////////////////////////////////////////////////////////////////

type IndexStatType string

const (
	IX_STAT_NUM_PAGES     IndexStatType = "NUM_PAGES"
	IX_STAT_NUM_ITEMS     IndexStatType = "NUM_ITEMS"
	IX_STAT_RES_RATIO     IndexStatType = "RESIDENT_RATIO"
	IX_STAT_NUM_INSERT    IndexStatType = "NUM_INSERT"
	IX_STAT_NUM_DELETE    IndexStatType = "NUM_DELETE"
	IX_STAT_AVG_ITEM_SIZE IndexStatType = "AVG_ITEM_SIZE"
	IX_STAT_AVG_PAGE_SIZE IndexStatType = "AVG_PAGE_SIZE"
	IX_STAT_PARTITION_ID  IndexStatType = "PARTITION_ID"
)

func (indexStatType IndexStatType) String() string {
	return string(indexStatType)
}

type IndexStorageMode string

const (
	INDEX_MODE_MOI     IndexStorageMode = "MOI"
	INDEX_MODE_PLASMA  IndexStorageMode = "PLASMA"
	INDEX_MODE_FDB     IndexStorageMode = "FDB" // legacy forest db
	INDEX_MODE_VIRTUAL IndexStorageMode = "VIRTUAL"
)

type Index4 interface {
	Index3

	StorageMode() (IndexStorageMode, errors.Error)
	LeadKeyHistogram(requestId string) (*Histogram, errors.Error)
	StorageStatistics(requestid string) ([]map[IndexStatType]value.Value, errors.Error)
}

////////////////////////////////////////////////////////////////////////
//
// End of Index API4.
//
////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////
//
// Index API5 introduced in Elixir for Indexer accounting.
//
////////////////////////////////////////////////////////////////////////

type CountIndex5 interface {
	CountIndex2

	// Perform a count on index
	Count5(requestId string, spans Spans2, cons ScanConsistency, vector timestamp.Vector, conn *IndexConnection) (
		int64, errors.Error)

	// Perform a count distinct on index
	CountDistinct5(requestId string, spans Spans2, cons ScanConsistency, vector timestamp.Vector, conn *IndexConnection) (
		int64, errors.Error)
}

// Indexer5 interface introduced to handle restrictions on parameters in WITH clause in serverless mode
type Indexer5 interface {
	Indexer3

	CreateIndex5(requestId, name string, rangeKey IndexKeys, indexPartition *IndexPartition,
		where expression.Expression, with value.Value, conn *IndexConnection) (Index, errors.Error)

	CreatePrimaryIndex5(requestId, name string, indexPartition *IndexPartition,
		with value.Value, conn *IndexConnection) (PrimaryIndex, errors.Error)
}

////////////////////////////////////////////////////////////////////////
//
// End of Index API5.
//
////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////
//
// FTS Index API
//
////////////////////////////////////////////////////////////////////////

type FTSSearchInfo struct {
	Field   value.Value // Search Field
	Query   value.Value // Search query
	Options value.Value // Search options
	Order   []string    // "_score ASC/DESC"
	Offset  int64       // offset (0 in case of none)
	Limit   int64       // limit (MaxInt64 in case of none)
}

type FTSIndex interface {
	Index

	/* Search returns one IndexEntry for each document.
	   PrimaryKey -- document key
	   MetaData   -- by default contains "score", If options contains "meta":true then this
	                 object contains whole meta info except fields.
	   EntryKey   -- none
	*/
	Search(requestId string, searchInfo *FTSSearchInfo,
		cons ScanConsistency, vector timestamp.Vector, conn *IndexConnection)

	// For given field/query Index is qualified, exact=true when no false positives
	Sargable(field string, query, options expression.Expression, mappings interface{}) (nkeys int,
		size int64, exact bool, omappings interface{}, err errors.Error)

	// Pageable is allowed
	Pageable(order []string, offset, limit int64, query, options expression.Expression) bool

	// Transform N1QL predicate to Search() function request
	SargableFlex(requestId string, request *FTSFlexRequest) (resp *FTSFlexResponse, err errors.Error)
}

/*
 * This global package level function.
 *
 * NewVerify (collection, field string, query, options value.Value, parallelism int) (datastore.Verify, errors.Error)
 *
 * collection  -- bucketname or collection name (namespace:bucket.scope.collection)
 * filed       -- search filed name
 * query       -- search query
 * options     -- search options
 * parallelism -- max_parallelism
 *
 * NOTE: If FTSclient uses N1QL expression package, This must be put in separate package and should not use
 *       FTSclient package or N1QL expression package to avoid circular refrences.
 *
 */

type Verify interface {
	/* Given document verify the result based on serach parameters
	 * The verification must match the index search results. Should be able to verify without index definition.
	 * If not able to do should raise error
	 */
	// item  -- document
	Evaluate(item value.Value) (bool, errors.Error)
}

/*
Handle [NULLS FIRST|LAST] caluse
*/
const (
	ORDER_NULLS_NONE = 1 << iota
	ORDER_NULLS_FIRST
	ORDER_NULLS_LAST
)

type SortTerm struct {
	Expr       expression.Expression
	Descending bool
	NullsPos   uint32
}

type FTSFlexRequest struct {
	Keyspace      string                 // keyspace alias name
	Bindings      expression.Bindings    // Unnest bindings depends on this keyspace
	Pred          expression.Expression  // predicate depends on the keyspace
	Opaque        map[string]interface{} // opaque
	Cond          expression.Expression  // DNF index condition
	OrigCond      expression.Expression  // Original index condition
	CheckPageable bool                   // Do pageable check
	Order         []*SortTerm            // Order terms
	Offset        int64                  // offset (0 in case of none)
	Limit         int64                  // limit (MaxInt64 in case of none)
}

const (
	FTS_FLEXINDEX_EXACT  = 1 << iota // all the predicates used and transformed and no false positives
	FTS_FLEXINDEX_LIMIT              // Handle Limit
	FTS_FLEXINDEX_OFFSET             // Handle Offset
	FTS_FLEXINDEX_ORDER              // Handle Order
)

type FTSFlexResponse struct {
	SearchQuery     string                           // Search query/request
	SearchOptions   string                           // Search options
	SearchOrders    []string                         // results are ordered by
	StaticSargKeys  map[string]expression.Expression // static sargable key paths
	DynamicSargKeys map[string]expression.Expression // dynamic sargable key paths
	RespFlags       uint32                           // Response Flags
	NumIndexedKeys  uint32                           // number of indexed keys
}

////////////////////////////////////////////////////////////////////////
//
// End of FTS Index API
//
////////////////////////////////////////////////////////////////////////

type IndexEntry struct {
	EntryKey   value.Values
	MetaData   value.Value
	PrimaryKey string
}

func (this *IndexEntry) Copy() *IndexEntry {
	rv := &IndexEntry{
		PrimaryKey: this.PrimaryKey,
	}
	if this.MetaData != nil {
		rv.MetaData = this.MetaData.Copy()
	}
	rv.EntryKey = make(value.Values, len(this.EntryKey))
	for i, key := range this.EntryKey {
		if key != nil {
			rv.EntryKey[i] = key.Copy()
		}
	}
	return rv
}

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
	sender       EntryExchange
	context      Context
	timeout      bool
	primary      bool
	skipNewKeys  bool
	skipMetering bool
}

type Sender interface {
	SendEntry(entry *IndexEntry) bool
	GetEntry() (*IndexEntry, bool)
	Close()
	Capacity() int
	Length() int
	IsStopped() bool
}

const _ENTRY_CAP = 512 // Default index scan request size

// Context cannot be nil
func NewIndexConnection(context Context) *IndexConnection {
	size := context.GetScanCap()
	if size <= 0 {
		size = _ENTRY_CAP
	}

	rv := &IndexConnection{
		context: context,
	}
	newEntryExchange(&rv.sender, size)
	return rv
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

	rv := &IndexConnection{
		context: context,
	}
	newEntryExchange(&rv.sender, maxSize)
	return rv, nil
}

func NewSimpleIndexConnection(context Context) (*IndexConnection, errors.Error) {
	rv := &IndexConnection{
		context: context,
	}
	newEntryExchange(&rv.sender, 1)
	return rv, nil
}

func (this *IndexConnection) Dispose() {
	// Entry Exchange expects two closes, one from the sender and one from the receiver
	// the first marks the connection has having completed the data
	// the second marks all actors as gone, meaning the connection can be recycled
	this.sender.Close()
}

func (this *IndexConnection) SendStop() {
	this.sender.sendStop()
}

func (this *IndexConnection) SendTimeout() {
	this.sender.sendTimeout()
}

func (this *IndexConnection) Reset() {
	this.sender.reset()
}

func (this *IndexConnection) Sender() Sender {
	return &this.sender
}

func (this *IndexConnection) Fatal(err errors.Error) {
	if !this.sender.IsClosed() {
		this.context.Fatal(err)
	}
}

func (this *IndexConnection) MaxParallelism() int {
	return this.context.MaxParallelism()
}

func (this *IndexConnection) Error(err errors.Error) {
	if this.primary && (err.Code() == errors.E_CB_INDEX_SCAN_TIMEOUT || strings.Contains(err.Error(), "Index scan timed out")) {
		this.timeout = true
		return
	}
	if !this.sender.IsClosed() {
		this.context.Error(err)
	}
}

func (this *IndexConnection) Warning(wrn errors.Error) {
	if !this.sender.IsClosed() {
		this.context.Warning(wrn)
	}
}

func (this *IndexConnection) GetReqDeadline() time.Time {
	return this.context.GetReqDeadline()
}

func (this *IndexConnection) SetPrimary() {
	this.primary = true
}

func (this *IndexConnection) Timeout() bool {
	return this.timeout
}

func (this *IndexConnection) QueryContext() QueryContext {
	context, _ := this.context.(QueryContext)
	return context
}

func (this *IndexConnection) RecordGsiRU(ru tenant.Unit) {
	this.context.RecordGsiRU(ru)
}

func (this *IndexConnection) RecordFtsRU(ru tenant.Unit) {
	this.context.RecordFtsRU(ru)
}

func (this *IndexConnection) User() string {
	rv, _ := FirstCred(this.context.Credentials())
	return rv
}

func (this *IndexConnection) SetSkipNewKeys(on bool) {
	this.skipNewKeys = on
}

func (this *IndexConnection) SkipNewKeys() bool {
	return this.skipNewKeys
}

func (this *IndexConnection) SetSkipMetering(on bool) {
	this.skipMetering = on
}

func (this *IndexConnection) SkipMetering() bool {
	return this.skipMetering
}

func (this *IndexConnection) SkipKey(key string) bool {
	return this.context.SkipKey(key)
}

func (this *IndexConnection) Context() Context {
	return this.context
}

func (this *IndexKey) Expression() expression.Expression {
	return this.Expr
}

func (this *IndexKey) SetAttribute(attr IkAttributes, add bool) {
	if add {
		this.Attributes |= attr
	} else {
		this.Attributes = attr
	}
}

func (this *IndexKey) UnsetAttribute(attr IkAttributes) {
	this.Attributes &^= attr
}

func (this *IndexKey) HasAttribute(attr IkAttributes) bool {
	return (this.Attributes & attr) != 0
}
