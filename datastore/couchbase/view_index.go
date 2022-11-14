//  Copyright (c) 2013 Couchbase, Inc.
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
	"net/http"
	"sort"
	"sync"
	"time"

	cb "github.com/couchbase/go-couchbase"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type viewIndexer struct {
	keyspace         *keyspace
	indexes          map[string]datastore.Index
	primary          map[string]datastore.PrimaryIndex
	nonUsableIndexes []string // indexes that cannot be used
	sync.RWMutex
}

const _BATCH_SIZE = 64 * 1024

func newViewIndexer(keyspace *keyspace) datastore.Indexer {
	rv := &viewIndexer{
		keyspace:         keyspace,
		indexes:          make(map[string]datastore.Index),
		primary:          make(map[string]datastore.PrimaryIndex),
		nonUsableIndexes: make([]string, 0, 10),
	}

	go rv.keepIndexesFresh()
	return rv
}

func (view *viewIndexer) keepIndexesFresh() {

	tickChan := time.Tick(500 * time.Millisecond)

	for _ = range tickChan {
		if view.keyspace.flags != 0 {
			return
		}
		view.Refresh()
	}
}

func (view *viewIndexer) KeyspaceId() string {
	return view.keyspace.Name()
}

func (view *viewIndexer) Name() datastore.IndexType {
	return datastore.VIEW
}

func (view *viewIndexer) IndexById(id string) (datastore.Index, errors.Error) {
	return view.IndexByName(id)
}

func (view *viewIndexer) IndexByName(name string) (datastore.Index, errors.Error) {
	view.RLock()
	index, ok := view.indexes[name]
	view.RUnlock()

	if !ok {
		return nil, errors.NewCbViewNotFoundError(nil, name)
	}
	return index, nil
}

func (view *viewIndexer) IndexNames() ([]string, errors.Error) {
	view.RLock()
	defer view.RUnlock()

	rv := make([]string, 0, len(view.indexes))
	for name, _ := range view.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (view *viewIndexer) IndexIds() ([]string, errors.Error) {
	return view.IndexNames()
}

func (view *viewIndexer) PrimaryIndexes() ([]datastore.PrimaryIndex, errors.Error) {
	view.RLock()
	defer view.RUnlock()

	logging.Debugf(" Number of primary indexes on b0 %v", len(view.primary))
	rv := make([]datastore.PrimaryIndex, 0, len(view.primary))
	for _, index := range view.primary {
		rv = append(rv, index)
	}
	return rv, nil
}

func (view *viewIndexer) Indexes() ([]datastore.Index, errors.Error) {
	view.RLock()
	defer view.RUnlock()

	rv := make([]datastore.Index, 0, len(view.indexes))
	for _, index := range view.indexes {
		rv = append(rv, index)
	}
	return rv, nil
}

func (view *viewIndexer) CreatePrimaryIndex(requestId, name string, with value.Value) (
	datastore.PrimaryIndex, errors.Error) {

	// if name is not provided then use default name #primary
	if name == "" {
		name = PRIMARY_INDEX
	}

	view.Refresh()
	if _, exists := view.indexes[name]; exists {
		return nil, errors.NewCbViewExistsError(nil, name)
	}

	// if the name matches any of the unusable indexes, return an error
	for _, iname := range view.nonUsableIndexes {
		if name == iname {
			return nil, errors.NewCbViewExistsError(nil, "Non usable index "+name)
		}
	}

	if with != nil {
		return nil, errors.NewCbViewsWithNotAllowedError(nil, "")
	}

	logging.Debugf("Creating primary index <ud>%s</ud>", name)

	idx, err := newViewPrimaryIndex(view, name)
	if err != nil {
		return nil, errors.NewCbViewCreateError(err, name)
	}

	view.Lock()
	defer view.Unlock()

	view.indexes[idx.Name()] = idx
	view.primary[idx.Name()] = idx
	return idx, nil
}

func (view *viewIndexer) CreateIndex(requestId, name string, seekKey, rangeKey expression.Expressions,
	where expression.Expression, with value.Value) (datastore.Index, errors.Error) {

	view.Refresh()
	if _, exists := view.indexes[name]; exists {
		return nil, errors.NewCbViewExistsError(nil, name)
	}

	// if the name matches any of the unusable indexes, return an error
	for _, iname := range view.nonUsableIndexes {
		if name == iname {
			return nil, errors.NewCbViewExistsError(nil, "Non usable index "+name)
		}
	}

	logging.Debugf("Creating index <ud>%s</ud> with equal key <ud>%v</ud> range key <ud>%v</ud>", name, seekKey, rangeKey)

	var idx datastore.Index
	var err error

	if with != nil {
		s, ok := with.Actual().(string)
		if !ok || s == "" {
			return nil, errors.NewCbViewCreateError(nil, "WITH value must be a string naming a design doc.")
		}

		idx, err = newViewIndexFromExistingMap(name, s, rangeKey, view)
	} else {
		idx, err = newViewIndex(name, rangeKey, where, view)
	}

	if err != nil {
		return nil, errors.NewCbViewCreateError(err, name)
	}

	view.Lock()
	defer view.Unlock()

	view.indexes[idx.Name()] = idx
	return idx, nil
}

func (view *viewIndexer) BuildIndexes(requestId string, names ...string) errors.Error {
	return errors.NewCbViewsNotSupportedError(nil, "BUILD INDEXES is not supported for VIEW.")
}

func (view *viewIndexer) indexesUpdated(a, b map[string]datastore.Index) bool {

	if len(a) != len(b) {
		return true
	}

	view.RLock()
	defer view.RUnlock()

	defer func() {
		if err := recover(); err != nil {
			logging.Errorf("Panic in compare", err)
		}
	}()

	// if the checksum of each index is the same
	for name, idx_a := range a {
		idx_b, ok := b[name]
		if !ok {
			return true
		}

		switch idx_a.(type) {
		case *primaryIndex:
			if idx_a.(*primaryIndex).signature() != idx_b.(*primaryIndex).signature() {
				return true
			}
		default:
			if idx_a.(*viewIndex).signature() != idx_b.(*viewIndex).signature() {
				return true
			}
		}
	}

	return false
}

func (view *viewIndexer) loadViewIndexes() errors.Error {

	indexList, nonUsableIndexes, err := loadViewIndexes(view)
	if err != nil {
		return errors.NewCbLoadIndexesError(err, "Keyspace "+view.KeyspaceId())
	}

	// recreate indexes from ddocs
	indexes := make(map[string]datastore.Index, len(indexList))
	primary := make(map[string]datastore.PrimaryIndex, len(indexList))

	for _, index := range indexList {
		name := index.Name()
		indexes[name] = index
		switch index.(type) {
		case *primaryIndex:
			primary[name] = index.(datastore.PrimaryIndex)
		}
	}

	sort.Strings(nonUsableIndexes)
	if !util.SortedStringsEqual(view.nonUsableIndexes, nonUsableIndexes) {
		view.Lock()
		view.nonUsableIndexes = nonUsableIndexes
		view.Unlock()
	}

	// only if the indexes have changed then update
	if view.indexesUpdated(view.indexes, indexes) {
		logging.Infof("View indexes updated.")

		view.Lock()
		view.indexes = indexes
		view.primary = primary
		view.Unlock()
	}

	return nil
}

func (view *viewIndexer) Refresh() errors.Error {
	// trigger refresh of this indexer
	err := view.loadViewIndexes()
	if err != nil {
		return errors.NewCbViewIndexesLoadingError(err, view.keyspace.Name())
	}

	return nil
}

func (view *viewIndexer) MetadataVersion() uint64 {
	return plan.REPREPARE_CHECK
}

func (view *viewIndexer) SetLogLevel(level logging.Level) {
	// No-op, uses query engine logger
}

func (view *viewIndexer) SetConnectionSecurityConfig(conSecConfig *datastore.ConnectionSecurityConfig) {
	// Do nothing.
}

type viewIndex struct {
	name      string
	using     datastore.IndexType
	on        expression.Expressions
	where     expression.Expression
	ddoc      *designdoc
	keyspace  *keyspace
	view      *viewIndexer
	isPrimary bool
}

type designdoc struct {
	name     string
	viewname string
	mapfn    string
	reducefn string
	cksum    int
}

func (vi *viewIndex) KeyspaceId() string {
	return vi.keyspace.Id()
}

func (vi *viewIndex) Id() string {
	return vi.name
}

func (vi *viewIndex) Name() string {
	return vi.name
}

func (vi *viewIndex) Type() datastore.IndexType {
	return vi.using
}

func (vi *viewIndex) Indexer() datastore.Indexer {
	return vi.view
}

func (vi *viewIndex) Key() expression.Expressions {
	return vi.on
}

func (idx *viewIndex) DDocName() string {
	return idx.ddoc.name
}

func (idx *viewIndex) ViewName() string {
	return idx.ddoc.viewname
}

func (vi *viewIndex) SeekKey() expression.Expressions {
	return nil
}

func (vi *viewIndex) RangeKey() expression.Expressions {
	return vi.on
}

func (vi *viewIndex) Condition() expression.Expression {
	return vi.where
}

func (vi *viewIndex) IsPrimary() bool {
	return vi.isPrimary
}

func (vi *viewIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (vi *viewIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (vi *viewIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	vi.Scan(requestId, nil, false, limit, cons, vector, conn)
}

func (vi *viewIndex) Drop(requestId string) errors.Error {

	err := vi.DropViewIndex()
	if err != nil {
		return errors.NewCbViewsDropIndexError(err, vi.Name())
	}
	// TODO need mutex

	vi.view.Lock()
	vi.view.Unlock()

	delete(vi.view.indexes, vi.name)
	if vi.IsPrimary() == true {
		logging.Infof(" Primary index being dropped ")
		delete(vi.view.primary, vi.name)
	}
	return nil
}

func (vi *viewIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()

	// For primary indexes, bounds must always be strings, so we
	// can just enforce that directly

	viewOptions := map[string]interface{}{}
	viewOptions = generateViewOptions(cons, span, vi.isPrimary) /*span.Range.Low, span.Range.High, span.Range.Inclusion) */
	viewRowChannel := make(chan cb.ViewRow)
	viewErrChannel := make(chan errors.Error)
	doneChannel := make(chan bool)
	defer close(doneChannel)

	go WalkViewInBatches(viewRowChannel, viewErrChannel, doneChannel, vi.keyspace.cbbucket,
		vi.DDocName(), vi.ViewName(), vi.IsPrimary(), viewOptions, _BATCH_SIZE, limit)

	var viewRow cb.ViewRow
	var err errors.Error
	sentRows := false
	ok := true
	numRows := 0
	for ok {
		select {
		case viewRow, ok = <-viewRowChannel:
			if ok {
				entry := datastore.IndexEntry{PrimaryKey: viewRow.ID}

				// try to add the view row key as the entry key (unless this is _all_docs)
				if vi.IsPrimary() == false {
					lookupValue, err := convertCouchbaseViewKeyToLookupValue(viewRow.Key)
					if err == nil {
						entry.EntryKey = lookupValue
					} else {
						conn.Error(errors.NewError(err, "View Row "+fmt.Sprintf("%v", viewRow.Key)))
					}
				}
				if conn.Sender().SendEntry(&entry) {
					sentRows = true
					numRows++
				} else {
					logging.Debugf(" Asked to stop after sending %v rows", numRows)
					ok = false
				}
			}
		case err, ok = <-viewErrChannel:
			if err != nil {
				logging.Errorf("%v", err)
				// check to possibly detect a bucket that was already deleted
				if !sentRows {
					logging.Debugf("Checking bucket URI: %v", vi.keyspace.cbbucket.URI)
					_, err := http.Get(vi.keyspace.cbbucket.URI)
					if err != nil {
						logging.Errorf("%v", err)

						// remove this specific bucket from the pool cache
						vi.keyspace.namespace.lock.Lock()
						delete(vi.keyspace.namespace.keyspaceCache, vi.keyspace.Name())
						vi.keyspace.namespace.lock.Unlock()
						// close this bucket
						vi.keyspace.Release(true)
						// ask the pool to refresh
						vi.keyspace.namespace.reload()
						// bucket doesnt exist any more
						conn.Error(errors.NewCbViewsAccessError(nil, "keyspace "+vi.keyspace.Name()+" or view index missing"))
						return
					}

				}

				conn.Error(err)
				return
			}
		}
	}

	logging.Debugf("Number of entries fetched from the index %d", numRows)

}

func (vi *viewIndex) signature() int {
	return vi.ddoc.cksum
}
