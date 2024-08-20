//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package couchbase

import (
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"github.com/couchbase/query/datastore"
	qe "github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

var _SCAN_RANGE_MIN = []byte{0x0}
var _SCAN_RANGE_MAX = []byte{0xff}

const _SCAN_POLL_TIMOUT = time.Second * 10
const _DEFAULT_REQUEST_TIMEOUT = time.Minute * 10
const _RS_ID = "#sequentialscan"

type seqScanIndexer struct {
	keyspace datastore.Keyspace
	primary  []datastore.PrimaryIndex
}

func newSeqScanIndexer(keyspace datastore.Keyspace) datastore.Indexer {
	rv := &seqScanIndexer{
		keyspace: keyspace,
	}
	rv.primary = make([]datastore.PrimaryIndex, 1, 1)
	rv.primary[0] = &seqScan{keyspace: keyspace, indexer: rv}
	return rv
}

func (this *seqScanIndexer) MetadataVersion() uint64 {
	if util.IsFeatureEnabled(util.GetN1qlFeatureControl(), util.N1QL_IGNORE_IDXR_META) {
		return 0
	}
	// return the associated GSI indexer's (if there is one) metadata version so that if an index is created prepared
	// statements using this indexer will be reprepared and may then take advantage of indices
	gsi, err := this.keyspace.Indexer(datastore.GSI)
	if err != nil {
		return 0
	}
	return gsi.MetadataVersion()
}

func (this *seqScanIndexer) SetConnectionSecurityConfig(conSecConfig *datastore.ConnectionSecurityConfig) {
}

func (this *seqScanIndexer) BucketId() string {
	scope := this.keyspace.Scope()
	if scope != nil {
		return scope.BucketId()
	}
	return this.keyspace.Id()
}

func (this *seqScanIndexer) ScopeId() string {
	return this.keyspace.ScopeId()
}

func (this *seqScanIndexer) KeyspaceId() string {
	return this.keyspace.Id()
}

func (this *seqScanIndexer) Name() datastore.IndexType {
	return datastore.SEQ_SCAN
}

func (this *seqScanIndexer) IndexById(id string) (datastore.Index, qe.Error) {
	if this.enabled() && id == this.primary[0].Id() {
		return this.primary[0].(datastore.Index), nil
	}
	return nil, qe.NewSSError(qe.E_SS_IDX_NOT_FOUND)
}

func (this *seqScanIndexer) enabled() bool {
	return this.keyspace.IsSystemCollection() || util.IsFeatureEnabled(util.GetN1qlFeatureControl(), util.N1QL_SEQ_SCAN)
}

func (this *seqScanIndexer) IndexByName(name string) (datastore.Index, qe.Error) {
	if this.enabled() && name == _RS_ID {
		return this.primary[0].(datastore.Index), nil
	}
	return nil, qe.NewSSError(qe.E_SS_IDX_NOT_FOUND)
}

func (this *seqScanIndexer) IndexNames() ([]string, qe.Error) {
	if !this.enabled() {
		return nil, nil
	}
	return []string{_RS_ID}, nil
}

func (this *seqScanIndexer) IndexIds() ([]string, qe.Error) {
	if !this.enabled() {
		return nil, nil
	}
	return []string{this.primary[0].Id()}, nil
}

func (this *seqScanIndexer) PrimaryIndexes() ([]datastore.PrimaryIndex, qe.Error) {
	if !this.enabled() {
		return nil, nil
	}
	return this.primary, nil
}

func (this *seqScanIndexer) Indexes() ([]datastore.Index, qe.Error) {
	if !this.enabled() {
		return nil, nil
	}
	rv := make([]datastore.Index, 0, 1)
	rv = append(rv, this.primary[0].(datastore.Index))
	return rv, nil
}

func (this *seqScanIndexer) CreatePrimaryIndex(requestId, name string, with value.Value) (
	datastore.PrimaryIndex, qe.Error) {

	return nil, qe.NewSSError(qe.E_SS_NOT_SUPPORTED, "CREATE PRIMARY INDEX is")
}

func (this *seqScanIndexer) CreateIndex(requestId, name string, seekKey, rangeKey expression.Expressions,
	where expression.Expression, with value.Value) (datastore.Index, qe.Error) {

	return nil, qe.NewSSError(qe.E_SS_NOT_SUPPORTED, "CREATE INDEX is")
}

func (this *seqScanIndexer) BuildIndexes(requestId string, names ...string) qe.Error {
	return qe.NewSSError(qe.E_SS_NOT_SUPPORTED, "BUILD INDEXES is")
}

func (this *seqScanIndexer) CreateIndex2(requestId, name string, seekKey expression.Expressions,
	rangeKey datastore.IndexKeys, where expression.Expression, with value.Value) (datastore.Index, qe.Error) {

	return nil, qe.NewSSError(qe.E_SS_NOT_SUPPORTED, "CREATE INDEX is")
}

func (this *seqScanIndexer) CreateIndex3(requestId, name string, rangeKey datastore.IndexKeys,
	indexPartition *datastore.IndexPartition, where expression.Expression, with value.Value) (datastore.Index, qe.Error) {

	return nil, qe.NewSSError(qe.E_SS_NOT_SUPPORTED, "CREATE INDEX is")
}

func (this *seqScanIndexer) CreatePrimaryIndex3(requestId, name string, indexPartition *datastore.IndexPartition,
	with value.Value) (datastore.PrimaryIndex, qe.Error) {

	return nil, qe.NewSSError(qe.E_SS_NOT_SUPPORTED, "CREATE PRIMARY INDEX is")
}

func (this *seqScanIndexer) Refresh() qe.Error {
	return nil
}

func (this *seqScanIndexer) SetLogLevel(level logging.Level) {
}

type seqScan struct {
	indexer          *seqScanIndexer
	keyspace         datastore.Keyspace
	totalScans       uint64
	totalReturnCount uint64
	lastScanAt       int64
	lastScanCount    uint64
}

func (this *seqScan) KeyspaceId() string {
	return this.indexer.KeyspaceId()
}

func (this *seqScan) Id() string {
	v := this.indexer.MetadataVersion()
	if v == 0 {
		return _RS_ID
	}
	return fmt.Sprintf("%v_%x", _RS_ID, v)
}

func (this *seqScan) Name() string {
	return _RS_ID
}

func (this *seqScan) Type() datastore.IndexType {
	return datastore.SEQ_SCAN
}

func (this *seqScan) Indexer() datastore.Indexer {
	return this.indexer
}

func (this *seqScan) SeekKey() expression.Expressions {
	return nil
}

func (this *seqScan) RangeKey() expression.Expressions {
	return nil
}

func (this *seqScan) Condition() expression.Expression {
	return nil
}

func (this *seqScan) IsPrimary() bool {
	return true
}

func (this *seqScan) State() (state datastore.IndexState, msg string, err qe.Error) {
	return datastore.ONLINE, "", nil
}

func (this *seqScan) Statistics(requestId string, span *datastore.Span) (datastore.Statistics, qe.Error) {
	return nil, nil
}

func (this *seqScan) Drop(requestId string) qe.Error {
	return qe.NewSSError(qe.E_SS_NOT_SUPPORTED, "DROP INDEX is")
}

func (this *seqScan) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {

	defer conn.Sender().Close()
	var i interface{}
	i = this
	logging.Stackf(logging.DEBUG, "Scan should never see this: %T", i.(datastore.Index3))
}

func (this *seqScan) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {

	defer conn.Sender().Close()
	var i interface{}
	i = this
	logging.Stackf(logging.DEBUG, "ScanEntries should never see this: %T", i.(datastore.PrimaryIndex3))
}

func (this *seqScan) Count(span *datastore.Span, cons datastore.ScanConsistency, vector timestamp.Vector) (int64, qe.Error) {
	return 0, nil
}

func (this *seqScan) RangeKey2() datastore.IndexKeys {
	return nil
}

func (this *seqScan) Scan2(requestId string, spans datastore.Spans2, reverse, distinctAfterProjection,
	ordered bool, projection *datastore.IndexProjection, offset, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {

	defer conn.Sender().Close()
	logging.Stackf(logging.DEBUG, "Scan2 should never see this: ordered: %v", ordered)
}

func (this *seqScan) Count2(requestId string, spans datastore.Spans2, cons datastore.ScanConsistency,
	vector timestamp.Vector) (int64, qe.Error) {

	return 0, nil
}

func (this *seqScan) CanCountDistinct() bool {
	return false
}

func (this *seqScan) CountDistinct(requestId string, spans datastore.Spans2, cons datastore.ScanConsistency,
	vector timestamp.Vector) (int64, qe.Error) {

	return 0, nil
}

func (this *seqScan) CreateAggregate(requestId string, groupAggs *datastore.IndexGroupAggregates,
	with value.Value) qe.Error {
	return qe.NewSSError(qe.E_SS_NOT_SUPPORTED, "CREATE AGGREGATE is")
}

func (this *seqScan) DropAggregate(requestId, name string) qe.Error {
	return qe.NewSSError(qe.E_SS_NOT_SUPPORTED, "DROP AGGREGATE is")
}

func (this *seqScan) Aggregates() ([]datastore.IndexGroupAggregates, qe.Error) {
	return nil, qe.NewSSError(qe.E_SS_NOT_SUPPORTED, "Precomputed Aggregates are")
}

func (this *seqScan) PartitionKeys() (*datastore.IndexPartition, qe.Error) {
	return nil, nil
}

func (this *seqScan) Scan3(requestId string, spans datastore.Spans2, reverse, distinctAfterProjection bool,
	projection *datastore.IndexProjection, offset, limit int64,
	groupAggs *datastore.IndexGroupAggregates, indexOrders datastore.IndexKeyOrders,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {

	var ranges []*datastore.SeqScanRange
	fullScan := &datastore.SeqScanRange{Start: _SCAN_RANGE_MIN, ExcludeStart: true, End: _SCAN_RANGE_MAX, ExcludeEnd: true}

	if len(spans) > 0 {
		ranges = make([]*datastore.SeqScanRange, 0, len(spans))
	spans:
		for _, span := range spans {
			for _, rng := range span.Ranges {
				r := &datastore.SeqScanRange{
					Start:        filterFromValue(rng.Low, false),
					ExcludeStart: rng.Inclusion&datastore.LOW == 0,
					End:          filterFromValue(rng.High, true),
					ExcludeEnd:   rng.Inclusion&datastore.HIGH == 0,
				}
				if r.Equals(fullScan) {
					ranges = nil
					break spans
				}
				ranges = mergeInto(ranges, r)
			}
		}
	}

	if len(ranges) == 0 {
		ranges = append(ranges, fullScan)
	}

	this.doScanEntries(requestId, indexOrders != nil, offset, limit, cons, vector, conn, ranges)
}

func filterFromValue(val value.Value, max bool) []byte {
	if val == nil || val.Type() != value.STRING {
		if max {
			return _SCAN_RANGE_MAX
		} else {
			return _SCAN_RANGE_MIN
		}
	}
	return []byte(val.ToString())
}

func mergeInto(ranges []*datastore.SeqScanRange, r *datastore.SeqScanRange) []*datastore.SeqScanRange {
	for i := range ranges {
		if r.OverlapsWith(ranges[i]) {
			if r.MergeWith(ranges[i]) {
				return ranges
			}
			if i != len(ranges)-1 {
				copy(ranges[i:], ranges[i+1:])
			}
			return mergeInto(ranges[:len(ranges)-1], r)
		}
	}
	return append(ranges, r)
}

func (this *seqScan) Alter(requestId string, with value.Value) (datastore.Index, qe.Error) {
	return nil, qe.NewSSError(qe.E_SS_NOT_SUPPORTED, "ALTER is")
}

func (this *seqScan) ScanEntries3(requestId string, projection *datastore.IndexProjection, offset, limit int64,
	groupAggs *datastore.IndexGroupAggregates, indexOrders datastore.IndexKeyOrders, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {

	fullScan := &datastore.SeqScanRange{Start: _SCAN_RANGE_MIN, ExcludeStart: true, End: _SCAN_RANGE_MAX, ExcludeEnd: true}
	ranges := append([]*datastore.SeqScanRange(nil), fullScan)

	this.doScanEntries(requestId, indexOrders != nil, offset, limit, cons, vector, conn, ranges)
}

func (this *seqScan) doScanEntries(requestId string, ordered bool, offset, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection, ranges []*datastore.SeqScanRange) {

	defer conn.Sender().Close()

	qctx := conn.QueryContext()
	if qctx == nil {
		qctx = datastore.NULL_QUERY_CONTEXT
	}
	qctx.Infof("Running sequential scan on %v", this.KeyspaceId())

	atomic.AddUint64(&this.totalScans, 1)
	atomic.StoreInt64(&this.lastScanAt, int64(time.Now().UnixNano()))

	if limit < 0 {
		return
	} else if limit == 0 {
		limit = math.MaxInt64
	}

	// deadline will be set whenever there is a request timeout set
	rqd := conn.GetReqDeadline()
	var tout time.Duration
	if !rqd.IsZero() {
		tout = rqd.Sub(time.Now())
	} else {
		// if not, use a default to ensure we never hang entirely here
		tout = _DEFAULT_REQUEST_TIMEOUT
	}
	if tout <= 0 {
		conn.Error(qe.NewSSError(qe.E_SS_FAILED, qe.NewSSError(qe.E_SS_TIMEOUT)))
		return
	}
	deadline := util.Now().Add(tout)
	scanPollTimeout := _SCAN_POLL_TIMOUT
	returned := int64(0)

	scanner, ok := this.keyspace.(datastore.SeqScanner)
	if !ok {
		conn.Error(qe.NewSSError(qe.E_SS_NOT_SUPPORTED, fmt.Sprintf("%T", this.keyspace)))
		return
	}

	if offset < 0 {
		offset = 0
	}
	var ss interface{}
	var err qe.Error
	var timeout bool
	var sk func(string) bool

	if conn.SkipNewKeys() {
		sk = conn.SkipKey
	}

	ss, err = scanner.StartKeyScan(qctx, ranges, offset, limit, ordered, tout, conn.Sender().Capacity(), tenant.IsServerless(), sk)
	if err != nil {
		conn.Error(qe.NewSSError(qe.E_SS_FAILED, err))
		return
	}

	scanTimeout := time.NewTimer(time.Second)
	if !scanTimeout.Stop() {
		<-scanTimeout.C
	}
	defer scanTimeout.Stop()

	for cont := true; cont && !conn.Sender().IsStopped() && (limit <= 0 || returned < limit); {
		if util.Now() >= deadline {
			conn.SendTimeout()
			conn.Error(qe.NewSSError(qe.E_SS_FAILED, qe.NewSSError(qe.E_SS_TIMEOUT)))
			break
		}

		var keys []string
		remaining := deadline.Sub(util.Now())
		if remaining < scanPollTimeout {
			scanPollTimeout = remaining
		}

		keys, err, timeout = scanner.FetchKeys(ss, scanPollTimeout)
		if err != nil {
			conn.Error(qe.NewSSError(qe.E_SS_FAILED, err))
			break
		}
		if timeout {
			// allow for checking if a stop has been sent or dealine has been reached
			continue
		}
		if len(keys) == 0 {
			break // EOF
		}

		// send the keys on
		for i := range keys {
			entry := &datastore.IndexEntry{PrimaryKey: keys[i]}

			scanTimeout.Reset(deadline.Sub(util.Now()))
			select {
			case <-scanTimeout.C:
				conn.SendTimeout()
				conn.Error(qe.NewSSError(qe.E_SS_FAILED, qe.NewSSError(qe.E_SS_TIMEOUT)))
			default:
				cont = conn.Sender().SendEntry(entry)
				if !scanTimeout.Stop() {
					<-scanTimeout.C
				}
			}
			if cont && !conn.Timeout() {
				returned++
				if limit > 0 && returned >= limit {
					break
				}
			} else {
				break
			}
		}
	}

	if ss != nil {
		var ru uint64
		ru, err = scanner.StopScan(ss)
		if err != nil {
			conn.Error(qe.NewSSError(qe.E_SS_FAILED, err))
		}
		if ru > 0 {
			conn.Context().RecordKvRU(tenant.Unit(ru))
		}
	}
	if returned > 0 {
		n := atomic.AddUint64(&this.totalReturnCount, uint64(returned))
		if n < uint64(returned) {
			atomic.StoreUint64(&this.totalReturnCount, uint64(returned))
			atomic.StoreUint64(&this.totalScans, uint64(1))
		}
	}
	atomic.StoreUint64(&this.lastScanCount, uint64(returned))

	qctx.Infof("Sequential scan on `%v`.`%v`.`%v` returned %v keys",
		this.Indexer().BucketId(), this.Indexer().ScopeId(), this.KeyspaceId(), returned)
}

func (this *seqScan) IndexMetadata() map[string]interface{} {
	rv := make(map[string]interface{})
	ts := atomic.LoadUint64(&this.totalScans)
	tk := atomic.LoadUint64(&this.totalReturnCount)
	rv["total_scans"] = ts
	rv["total_keys_returned"] = tk
	avg := uint64(0)
	if ts > 0 {
		avg = uint64(math.Round(float64(tk) / float64(ts)))
	}
	rv["average_keys_per_scan"] = avg
	ls := atomic.LoadInt64(&this.lastScanAt)
	if ls != 0 {
		rv["last_scan_time"] = time.UnixMilli(ls / 1000000).Format(expression.DEFAULT_FORMAT)
		lsc := atomic.LoadUint64(&this.lastScanCount)
		rv["last_scan_keys"] = lsc
	}
	return rv
}
