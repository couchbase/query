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
	"net/http"

	cb "github.com/couchbase/go-couchbase"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type viewIndexer struct {
	keyspace         *keyspace
	indexes          map[string]datastore.Index
	primary          map[string]datastore.PrimaryIndex
	nonUsableIndexes []string // indexes that cannot be used
}

func newViewIndexer(keyspace *keyspace) datastore.Indexer {
	rv := &viewIndexer{
		keyspace:         keyspace,
		indexes:          make(map[string]datastore.Index),
		primary:          make(map[string]datastore.PrimaryIndex),
		nonUsableIndexes: make([]string, 0, 10),
	}

	return rv
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
	view.Refresh()
	index, ok := view.indexes[name]
	if !ok {
		return nil, errors.NewCbViewNotFoundError(nil, name)
	}
	return index, nil
}

func (view *viewIndexer) IndexNames() ([]string, errors.Error) {
	view.Refresh()
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
	view.Refresh()
	logging.Infof(" Number of primary indexes on b0 %v", len(view.primary))
	rv := make([]datastore.PrimaryIndex, 0, len(view.primary))
	for _, index := range view.primary {
		rv = append(rv, index)
	}
	return rv, nil
}

func (view *viewIndexer) Indexes() ([]datastore.Index, errors.Error) {
	view.Refresh()
	rv := make([]datastore.Index, 0, len(view.indexes))
	for _, index := range view.indexes {
		rv = append(rv, index)
	}
	return rv, nil
}

func (view *viewIndexer) CreatePrimaryIndex(name string, with value.Value) (datastore.PrimaryIndex, errors.Error) {

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
			return nil, errors.NewCbViewExistsError(nil, "Non usuable index "+name)
		}
	}

	if with != nil {
		return nil, errors.NewCbViewsWithNotAllowedError(nil, "")
	}

	logging.Infof("Creating primary index %s", name)

	idx, err := newViewPrimaryIndex(view, name)
	if err != nil {
		return nil, errors.NewCbViewCreateError(err, name)
	}

	view.indexes[idx.Name()] = idx
	view.primary[idx.Name()] = idx
	return idx, nil
}

func (view *viewIndexer) CreateIndex(name string, equalKey, rangeKey expression.Expressions,
	where expression.Expression, with value.Value) (datastore.Index, errors.Error) {

	view.Refresh()
	if _, exists := view.indexes[name]; exists {
		return nil, errors.NewCbViewExistsError(nil, name)
	}

	// if the name matches any of the unusable indexes, return an error
	for _, iname := range view.nonUsableIndexes {
		if name == iname {
			return nil, errors.NewCbViewExistsError(nil, "Non usuable index "+name)
		}
	}

	if with != nil {
		return nil, errors.NewCbViewsWithNotAllowedError(nil, "")
	}

	logging.Infof("Creating index %s with equal key %v range key %v", name, equalKey, rangeKey)

	idx, err := newViewIndex(name, datastore.IndexKey(rangeKey), where, view)
	if err != nil {
		return nil, errors.NewCbViewCreateError(err, name)
	}
	view.indexes[idx.Name()] = idx
	return idx, nil
}

func (view *viewIndexer) BuildIndexes(names ...string) errors.Error {
	return errors.NewCbViewsNotSupportedError(nil, "BUILD INDEXES is not supported for VIEW.")
}

func (view *viewIndexer) loadViewIndexes() errors.Error {
	// #alldocs implicitly exists

	// and recreate remaining from ddocs
	indexes, err := loadViewIndexes(view)
	if err != nil {
		return errors.NewCbLoadIndexesError(err, "Keyspace "+view.KeyspaceId())
	}

	if len(indexes) == 0 {
		logging.Errorf("No view indexes found for bucket %s", view.keyspace.Name())
		return nil
	}

	for _, index := range indexes {
		logging.Infof("Found index on keyspace %s", (*index).KeyspaceId())
		name := (*index).Name()
		view.indexes[name] = *index
		if (*index).(*viewIndex).isPrimaryIndex() == true {
			view.primary[name] = (*index).(datastore.PrimaryIndex)
		}
	}

	return nil
}

func (view *viewIndexer) Refresh() errors.Error {
	// trigger refresh of this indexer
	logging.Infof("Refreshing Indexes in keyspace %s", view.keyspace.Name())

	indexes, err := loadViewIndexes(view)
	if err != nil {
		logging.Errorf(" Error loading indexes for bucket %s", view.keyspace.Name())
		return errors.NewCbViewIndexesLoadingError(err, view.keyspace.Name())
	}

	if len(indexes) == 0 {
		logging.Infof("No indexes found for bucket %s", view.keyspace.Name())
		return nil
	}

	return nil
}

type viewIndex struct {
	name      string
	using     datastore.IndexType
	on        datastore.IndexKey
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

func (vi *viewIndex) Key() datastore.IndexKey {
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
	return expression.Expressions(vi.on)
}

func (vi *viewIndex) Condition() expression.Expression {
	return expression.Expression(vi.where)
}

func (vi *viewIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (vi *viewIndex) Statistics(span *datastore.Span) (datastore.Statistics, errors.Error) {
	return nil, nil
}

func (vi *viewIndex) ScanEntries(limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	vi.Scan(nil, false, limit, cons, vector, conn)
}

func (vi *viewIndex) isPrimaryIndex() bool {
	return vi.isPrimary
}

func (vi *viewIndex) Drop() errors.Error {

	err := vi.DropViewIndex()
	if err != nil {
		return errors.NewCbViewsDropIndexError(err, vi.Name())
	}
	// TODO need mutex
	delete(vi.view.indexes, vi.name)
	if vi.isPrimaryIndex() == true {
		logging.Infof(" Primary index being dropped ")
		delete(vi.view.primary, vi.name)
	}
	return nil
}

func (vi *viewIndex) Scan(span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer close(conn.EntryChannel())
	// For primary indexes, bounds must always be strings, so we
	// can just enforce that directly

	viewOptions := map[string]interface{}{}
	viewOptions = generateViewOptions(cons, span) /*span.Range.Low, span.Range.High, span.Range.Inclusion) */
	viewRowChannel := make(chan cb.ViewRow)
	viewErrChannel := make(chan errors.Error)
	go WalkViewInBatches(viewRowChannel, viewErrChannel, vi.keyspace.cbbucket, vi.DDocName(), vi.ViewName(), viewOptions, 1000, limit)

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
				if vi.DDocName() != "" /* FIXME && vi.IsPrimary() == false */ {
					lookupValue, err := convertCouchbaseViewKeyToLookupValue(viewRow.Key)
					if err == nil {
						entry.EntryKey = lookupValue
					} else {
						logging.Debugf("unable to convert index key to lookup value err:%v key %v", err, viewRow.Key)
					}
				}

				conn.EntryChannel() <- &entry
				sentRows = true
				numRows++
			}
		case err, ok = <-viewErrChannel:
			if err != nil {
				logging.Errorf("%v", err)
				// check to possibly detect a bucket that was already deleted
				if !sentRows {
					logging.Infof("Checking bucket URI: %v", vi.keyspace.cbbucket.URI)
					_, err := http.Get(vi.keyspace.cbbucket.URI)
					if err != nil {
						logging.Errorf("%v", err)

						// remove this specific bucket from the pool cache
						vi.keyspace.namespace.lock.Lock()
						delete(vi.keyspace.namespace.keyspaceCache, vi.keyspace.Name())
						vi.keyspace.namespace.lock.Unlock()
						// close this bucket
						vi.keyspace.Release()
						// ask the pool to refresh
						vi.keyspace.namespace.refresh(true)
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

	logging.Infof("Number of entries fetched from the index %d", numRows)

}
