//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package couchbase

import (
	"fmt"
	"sync"

	"github.com/couchbase/indexing/secondary/protobuf"
	"github.com/couchbase/indexing/secondary/queryport"
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

// ErrorNotImplemented is API not implemented.
var ErrorNotImplemented = errors.NewError(nil, "secondary.notImplemented")

// ErrorIndexEmpty is index not initialized.
var ErrorIndexEmpty = errors.NewError(nil, "secondaryIndex.empty")

// ErrorEmptyHost is no valid node hosting an index.
var ErrorEmptyHost = errors.NewError(nil, "secondaryIndex.emptyHost")

// ErrorEmptyStatistics is index-statistics not available
var ErrorEmptyStatistics = errors.NewError(nil, "secondaryIndex.emptyStatistics")

const QueryPortPageSize = 1

// secondaryIndex to hold meta data information, network-address for
// a single secondary-index.
type secondaryIndex struct {
	mu        sync.Mutex
	defnID    string
	name      string // name of the index
	keySpace  string // bucket
	isPrimary bool
	using     datastore.IndexType
	partnExpr string
	secExprs  string
	indxs     *secondaryIndexes
	stats     *statistics
	statBins  []*statistics
	// remote node hosting this index.
	host       string
	hostClient *queryport.Client
}

func (si *secondaryIndex) KeyspaceId() string {
	if si != nil {
		return si.keySpace // immutable field
	}
	return ""
}

func (si *secondaryIndex) Id() string {
	if si != nil {
		return si.defnID // immutable field
	}
	return ""
}

func (si *secondaryIndex) Name() string {
	if si != nil {
		return si.name // immutable field
	}
	return ""
}

func (si *secondaryIndex) Type() datastore.IndexType {
	if si != nil {
		return si.using // immutable field
	}
	return ""
}

func (si *secondaryIndex) Drop() errors.Error {
	if si == nil {
		return ErrorIndexEmpty
	}

	return si.indxs.deleteIndex(si)
}

func (si *secondaryIndex) EqualKey() (expr expression.Expressions) {
	if si != nil && si.partnExpr != "" {
		// TODO:
		// expr = expression.Parser(si.partnExpr)
		return expr
	}
	return
}

func (si *secondaryIndex) RangeKey() (expr expression.Expressions) {
	if si != nil && si.partnExpr != "" {
		// TODO:
		// expr = expression.Parser(si.secExprs)
		return
	}
	return
}

func (si *secondaryIndex) Condition() expression.Expression {
	panic(ErrorNotImplemented)
}

func (si *secondaryIndex) Rename(name string) errors.Error {
	panic(ErrorNotImplemented)
}

func (si *secondaryIndex) Statistics(
	span *datastore.Span) (datastore.Statistics, errors.Error) {

	if si.hostClient == nil {
		return nil, ErrorEmptyHost
	}

	var low, high []byte
	if len(span.Range.Low) > 0 {
		low = span.Range.Low[0].Bytes()
	}
	if len(span.Range.High) > 0 {
		high = span.Range.High[0].Bytes()
	}

	inclusion := uint32(span.Range.Inclusion)
	pstats, err := si.hostClient.Statistics(low, high, inclusion)
	if err != nil {
		return nil, errors.NewError(nil, err.Error())
	}

	si.mu.Lock()
	defer si.mu.Unlock()

	si.stats = (&statistics{}).updateStats(pstats)
	return si.stats, nil
}

func (si *secondaryIndex) Scan(
	span *datastore.Span, distinct bool, limit int64,
	conn *datastore.IndexConnection) errors.Error {

	if si.hostClient == nil {
		return ErrorEmptyHost
	}

	entryChannel := conn.EntryChannel()
	stopChannel := conn.StopChannel()

	defer close(entryChannel)

	var low, high []byte
	if len(span.Range.Low) > 0 {
		low = span.Range.Low[0].Bytes()
	}
	if len(span.Range.High) > 0 {
		high = span.Range.High[0].Bytes()
	}

	inclusion := uint32(span.Range.Inclusion)
	si.hostClient.Scan(
		low, high, inclusion, QueryPortPageSize, distinct, limit,
		func(data interface{}) bool {
			switch val := data.(type) {
			case *protobuf.ResponseStream:
				if err := val.GetErr().GetError(); err != "" {
					conn.Error(errors.NewError(nil, err))
					return false
				}
				for _, entry := range val.GetEntries() {
					e := &datastore.IndexEntry{
					// TODO: []bytes to implement value.Value{} interface.
					// EntryKey: entry.GetEntryKey(),
					// PrimaryKey: entry.GetPrimaryKey(),
					}
					fmt.Println(entry)
					select {
					case entryChannel <- e:
					case <-stopChannel:
						return false
					}
				}
				return true

			case error:
				conn.Error(errors.NewError(nil, val.Error()))
				return false
			}
			return false
		})
	return nil
}

// PrimaryIndex{} interface

func (si *secondaryIndex) ScanEntries(
	limit int64, conn *datastore.IndexConnection) errors.Error {

	if si.hostClient == nil {
		return ErrorEmptyHost
	}

	entryChannel := conn.EntryChannel()
	stopChannel := conn.StopChannel()

	defer close(entryChannel)

	si.hostClient.ScanAll(
		QueryPortPageSize, limit,
		func(data interface{}) bool {
			switch val := data.(type) {
			case *protobuf.ResponseStream:
				if err := val.GetErr().GetError(); err != "" {
					conn.Error(errors.NewError(nil, err))
					return false
				}
				for _, entry := range val.GetEntries() {
					e := &datastore.IndexEntry{
					// TODO: []bytes to implement value.Value{} interface.
					// EntryKey: entry.GetEntryKey(),
					// PrimaryKey: entry.GetPrimaryKey(),
					}
					fmt.Println(entry)
					select {
					case entryChannel <- e:
					case <-stopChannel:
						return false
					}
				}
				return true

			case error:
				conn.Error(errors.NewError(nil, val.Error()))
				return false
			}
			return false
		})
	return nil
}

type statistics struct {
	mu         sync.Mutex
	count      int64
	uniqueKeys int64
	min        []byte // JSON represented min value.Value{}
	max        []byte // JSON represented max value.Value{}
}

// Statistics{} interface

func (stats *statistics) Count() (int64, errors.Error) {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	if stats == nil {
		return 0, ErrorEmptyStatistics
	}
	return stats.count, nil
}

func (stats *statistics) DistinctCount() (int64, errors.Error) {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	if stats == nil {
		return 0, ErrorEmptyStatistics
	}
	return stats.uniqueKeys, nil
}

func (stats *statistics) Min() (value.Values, errors.Error) {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	if stats == nil {
		return nil, ErrorEmptyStatistics
	}
	// TODO: []bytes to implement value.Value{} interface.
	// return stats.min, nil
	return nil, nil
}

func (stats *statistics) Max() (value.Values, errors.Error) {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	if stats == nil {
		return nil, ErrorEmptyStatistics
	}
	// TODO: []bytes to implement value.Value{} interface.
	// return stats.max, nil
	return nil, nil
}

func (stats *statistics) Bins() ([]datastore.Statistics, errors.Error) {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	if stats == nil {
		return nil, ErrorEmptyStatistics
	}
	return nil, nil
}

// local function that can be used to asynchronously update
// meta-data information, host network-address from coordinator
// notifications.

func (si *secondaryIndex) setHost(host string) {
	si.mu.Lock()
	defer si.mu.Unlock()

	si.host = host
	// TODO: avoid magic numbers
	si.hostClient =
		queryport.NewClient(si.host, 5 /*poolSize*/, 2 /*poolOverflow*/)
}

func (stats *statistics) updateStats(pstats *protobuf.IndexStatistics) *statistics {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	stats.count = int64(pstats.GetCount())
	stats.uniqueKeys = int64(pstats.GetUniqueKeys())
	stats.min = pstats.GetMin()
	stats.max = pstats.GetMax()
	return stats
}
