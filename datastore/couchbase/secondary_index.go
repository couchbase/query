//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package couchbase

import "encoding/json"
import "sync"

import "github.com/couchbase/indexing/secondary/collatejson"
import "github.com/couchbase/indexing/secondary/protobuf"
import "github.com/couchbase/indexing/secondary/queryport"
import "github.com/couchbase/indexing/secondary/indexer"
import "github.com/couchbaselabs/query/datastore"
import "github.com/couchbaselabs/query/errors"
import "github.com/couchbaselabs/query/expression"
import "github.com/couchbaselabs/query/expression/parser"
import "github.com/couchbaselabs/query/value"
import "github.com/couchbaselabs/query/logging"
import c "github.com/couchbase/indexing/secondary/common"

// ErrorIndexEmpty is index not initialized.
var ErrorIndexEmpty = errors.NewError(nil, "secondaryIndex.empty")

// ErrorEmptyHost is no valid node hosting an index.
var ErrorEmptyHost = errors.NewError(nil, "secondaryIndex.emptyHost")

// ErrorEmptyStatistics is index-statistics not available.
var ErrorEmptyStatistics = errors.NewError(nil, "secondaryIndex.emptyStatistics")

const QueryPortPageSize = 1

// secondaryIndex to hold meta data information, network-address for
// a single secondary-index.
type secondaryIndex struct {
	mu        sync.Mutex
	name      string // name of the index
	defnID    string
	keySpace  *keyspace
	isPrimary bool
	using     datastore.IndexType
	partnExpr string
	secExprs  []string
	whereExpr string
	stats     *statistics
	statBins  []*statistics
	// remote node hosting this index.
	hosts       []string
	hostClients []*queryport.Client
}

var twoiInclusion = map[datastore.Inclusion]indexer.Inclusion{
	datastore.NEITHER: indexer.Neither,
	datastore.LOW:     indexer.Low,
	datastore.HIGH:    indexer.High,
	datastore.BOTH:    indexer.Both,
}

func (si *secondaryIndex) getHostClient() (*queryport.Client, errors.Error) {
	if si.hostClients == nil || len(si.hostClients) == 0 {
		return nil, ErrorEmptyHost
	}
	// TODO: use round-robin or other statistical heuristics to load balance.
	client := si.hostClients[0]
	return client, nil
}

// KeyspaceId implement Index{} interface.
func (si *secondaryIndex) KeyspaceId() string {
	return si.keySpace.Id()
}

// Id implement Index{} interface.
func (si *secondaryIndex) Id() string {
	return si.Name()
}

// Name implement Index{} interface.
func (si *secondaryIndex) Name() string {
	return si.name
}

// Type implement Index{} interface.
func (si *secondaryIndex) Type() datastore.IndexType {
	return si.using
}

// IsPrimary implement Index{} interface.
func (si *secondaryIndex) IsPrimary() bool {
	return false
}

// EqualKey implement Index{} interface.
func (si *secondaryIndex) EqualKey() expression.Expressions {
	if si != nil && si.partnExpr != "" {
		expr, _ := parser.Parse(si.partnExpr)
		return expression.Expressions{expr}
	}
	return nil
}

// RangeKey implement Index{} interface.
func (si *secondaryIndex) RangeKey() expression.Expressions {
	if si != nil && si.secExprs != nil {
		exprs := make(expression.Expressions, 0, len(si.secExprs))
		for _, exprS := range si.secExprs {
			expr, _ := parser.Parse(exprS)
			exprs = append(exprs, expr)
		}
		return exprs
	}
	return nil
}

// Condition implement Index{} interface.
func (si *secondaryIndex) Condition() expression.Expression {
	if si != nil && si.whereExpr != "" {
		expr, _ := parser.Parse(si.whereExpr)
		return expr
	}
	return nil
}

// State implement Index{} interface.
func (si *secondaryIndex) State() (datastore.IndexState, errors.Error) {
	return datastore.ONLINE, nil
}

// Statistics implement Index{} interface.
func (si *secondaryIndex) Statistics(
	span *datastore.Span) (datastore.Statistics, errors.Error) {

	client, err := si.getHostClient()
	if err != nil {
		return nil, err
	}

	low, high := keys2JSON(span.Range.Low), keys2JSON(span.Range.High)
	equal := [][]byte{keys2JSON(span.Equal)}
	incl := uint32(twoiInclusion[span.Range.Inclusion])
	indexn, bucketn := si.name, si.keySpace.Name()
	pstats, e := client.Statistics(indexn, bucketn, low, high, equal, incl)
	if e != nil {
		return nil, errors.NewError(nil, e.Error())
	}

	si.mu.Lock()
	defer si.mu.Unlock()
	si.stats = (&statistics{}).updateStats(pstats)
	return si.stats, nil
}

// Drop implement Index{} interface.
func (si *secondaryIndex) Drop() errors.Error {
	if si == nil {
		return ErrorIndexEmpty
	}
	client := queryport.NewClusterClient(ClusterManagerAddr)
	err := client.DropIndex(si.defnID)
	if err != nil {
		return errors.NewError(nil, err.Error())
	}
	delete(si.keySpace.indexes, si.Name())
	logging.Infof("Dropped index %v", si.Name())
	return nil
}

// Scan implement Index{} interface.
func (si *secondaryIndex) Scan(
	span *datastore.Span, distinct bool, limit int64,
	conn *datastore.IndexConnection) {

	entryChannel := conn.EntryChannel()
	stopChannel := conn.StopChannel()
	defer close(entryChannel)

	client, err := si.getHostClient()
	if err != nil {
		return
	}

	low, high := keys2JSON(span.Range.Low), keys2JSON(span.Range.High)
	equal := [][]byte{keys2JSON(span.Equal)}
	incl := uint32(twoiInclusion[span.Range.Inclusion])
	indexn, bucketn := si.name, si.keySpace.Name()
	client.Scan(
		indexn, bucketn, low, high, equal, incl,
		QueryPortPageSize, distinct, limit,
		func(data interface{}) bool {
			switch val := data.(type) {
			case *protobuf.ResponseStream:
				if err := val.GetErr().GetError(); err != "" {
					conn.Error(errors.NewError(nil, err))
					return false
				}
				for _, entry := range val.GetEntries() {
					key, err := json2Entry(entry.GetEntryKey())
					if err != nil {
						conn.Error(errors.NewError(nil, err.Error()))
						return false
					}
					e := &datastore.IndexEntry{
						EntryKey:   value.Values(key),
						PrimaryKey: string(entry.GetPrimaryKey()),
					}
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
}

// Scan implement PrimaryIndex{} interface.
func (si *secondaryIndex) ScanEntries(
	limit int64, conn *datastore.IndexConnection) {

	entryChannel := conn.EntryChannel()
	stopChannel := conn.StopChannel()
	defer close(entryChannel)

	client, err := si.getHostClient()
	if err != nil {
		return
	}

	indexn, bucketn := si.name, si.keySpace.Name()
	client.ScanAll(
		indexn, bucketn, QueryPortPageSize, limit,
		func(data interface{}) bool {
			switch val := data.(type) {
			case *protobuf.ResponseStream:
				if err := val.GetErr().GetError(); err != "" {
					conn.Error(errors.NewError(nil, err))
					return false
				}
				for _, entry := range val.GetEntries() {
					key, err := json2Entry(entry.GetEntryKey())
					if err != nil {
						conn.Error(errors.NewError(nil, err.Error()))
						return false
					}
					e := &datastore.IndexEntry{
						EntryKey:   value.Values(key),
						PrimaryKey: string(entry.GetPrimaryKey()),
					}
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
}

type statistics struct {
	mu         sync.Mutex
	count      int64
	uniqueKeys int64
	min        []byte // JSON represented min value.Value{}
	max        []byte // JSON represented max value.Value{}
}

// Count implement Statistics{} interface.
func (stats *statistics) Count() (int64, errors.Error) {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	if stats == nil {
		return 0, ErrorEmptyStatistics
	}
	return stats.count, nil
}

// DistinctCount implement Statistics{} interface.
func (stats *statistics) DistinctCount() (int64, errors.Error) {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	if stats == nil {
		return 0, ErrorEmptyStatistics
	}
	return stats.uniqueKeys, nil
}

// Min implement Statistics{} interface.
func (stats *statistics) Min() (value.Values, errors.Error) {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	if stats == nil {
		return nil, ErrorEmptyStatistics
	}
	vals := value.NewValue(stats.min).Actual().([]interface{})
	values := make(value.Values, 0, len(vals))
	for _, val := range vals {
		values = append(values, value.NewValue(val))
	}
	return values, nil
}

// Max implement Statistics{} interface.
func (stats *statistics) Max() (value.Values, errors.Error) {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	if stats == nil {
		return nil, ErrorEmptyStatistics
	}
	vals := value.NewValue(stats.max).Actual().([]interface{})
	values := make(value.Values, 0, len(vals))
	for _, val := range vals {
		values = append(values, value.NewValue(val))
	}
	return values, nil
}

// Bins implement Statistics{} interface.
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

// create a queryport client connected to `host`.
func (si *secondaryIndex) setHost(hosts []string) {
	si.mu.Lock()
	defer si.mu.Unlock()

	si.hosts = hosts
	config := c.SystemConfig.Clone()
	if len(hosts) > 0 {
		si.hostClients = make([]*queryport.Client, 0, len(hosts))
		for _, host := range hosts {
			c := queryport.NewClient(host, config)
			si.hostClients = append(si.hostClients, c)
		}
	}
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

// shape of key passed to scan-coordinator (indexer node) is,
//      [key1, key2, ... keyN]
// where N expressions supplied in CREATE INDEX
// to evaluate secondary-key.
func keys2JSON(arg value.Values) []byte {
	if arg == nil {
		return nil
	}
	values := []value.Value(arg)
	arr := value.NewValue(make([]interface{}, len(values)))
	for i, val := range values {
		arr.SetIndex(i, val)
	}
	bin, err := arr.MarshalJSON()
	if err != nil {
		logging.Errorf("unable to marshal %v: %v", arg, err)
	}
	return bin
}

// shape of return key from scan-coordinator is,
//      [key1, key2, ... keyN]
// where N keys where evaluated using N expressions supplied in
// CREATE INDEX.
//
// * Each key will be unmarshalled using json and composed into
//   value.Value{}.
// * Missing key will be composed using NewMissingValue(), btw,
//   `key1` will never be missing.
func json2Entry(data []byte) ([]value.Value, error) {
	arr := []interface{}{}
	err := json.Unmarshal(data, &arr)
	if err != nil {
		return nil, err
	}

	// [key1, key2, ... keyN]
	key := make([]value.Value, len(arr))
	for i := 0; i < len(arr); i++ {
		if s, ok := arr[i].(string); ok && collatejson.MissingLiteral.Equal(s) {
			key[i] = value.NewMissingValue()
		} else {
			key[i] = value.NewValue(arr[i])
		}
	}
	return key, nil
}
