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

	cb "github.com/couchbaselabs/go-couchbase"
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/logging"
	"github.com/couchbaselabs/query/value"
)

type viewIndex struct {
	name     string
	using    datastore.IndexType
	on       datastore.IndexKey
	where    expression.Expression
	ddoc     *designdoc
	keyspace *keyspace
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

func (vi *viewIndex) IsPrimary() bool {
	return false
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

func (vi *viewIndex) EqualKey() expression.Expressions {
	return nil
}

func (vi *viewIndex) RangeKey() expression.Expressions {
	// FIXME
	return nil
}

func (vi *viewIndex) Condition() expression.Expression {
	return nil
}

func (vi *viewIndex) Drop() errors.Error {
	bucket := vi.keyspace
	if vi.IsPrimary() {
		return errors.NewError(nil, "Primary index cannot be dropped.")
	}
	err := vi.DropViewIndex()
	if err != nil {
		return errors.NewError(err, fmt.Sprintf("Cannot drop index %s", vi.Name()))
	}
	delete(bucket.indexes, vi.name)
	return nil
}

func (b *keyspace) loadViewIndexes() errors.Error {
	// #alldocs implicitly exists
	/*
	   pi := newAllDocsIndex(b)
	   b.indexes[pi.name] = pi
	*/

	// and recreate remaining from ddocs
	indexes, err := loadViewIndexes(b)
	if err != nil {
		return errors.NewError(err, "Error loading indexes")
	}

	if len(indexes) == 0 {
		logging.Errorf("No indexes found for bucket %s", b.Name())
		return errors.NewError(nil, "No primary index found for bucket "+b.Name()+". Create a primary index ")
	}

	for _, index := range indexes {
		logging.Infof("Found index on keyspace %s", (*index).Name())
		name := (*index).Name()
		b.indexes[name] = *index
	}

	return nil
}

func (vi *viewIndex) Statistics(span *datastore.Span) (datastore.Statistics, errors.Error) {
	return nil, nil
}

func (vi *viewIndex) ScanEntries(limit int64, conn *datastore.IndexConnection) {
	vi.Scan(nil, false, limit, conn)
}

func (vi *viewIndex) Scan(span *datastore.Span, distinct bool, limit int64, conn *datastore.IndexConnection) {
	defer close(conn.EntryChannel())
	// For primary indexes, bounds must always be strings, so we
	// can just enforce that directly

	var low, high value.Value
	var inclusion datastore.Inclusion

	if span != nil && span.Range != nil {
		if len(span.Range.Low) == 0 {
			low = value.NewValue("")
		} else {
			low = span.Range.Low[0]
		}

		if len(span.Range.High) == 0 {
			high = value.NewValue("")
		} else {
			high = span.Range.High[0]
		}

		inclusion = span.Range.Inclusion
	}

	viewOptions := generateViewOptions(low, high, inclusion)
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
				if vi.DDocName() != "" {
					lookupValue, err := convertCouchbaseViewKeyEntryToValue(viewRow.Key)
					if err == nil {
						entry.EntryKey = value.Values{lookupValue}
					} else {
						logging.Errorf("unable to convert index key to lookup value:%v", err)
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
						delete(vi.keyspace.namespace.keyspaceCache, vi.keyspace.Name())
						// close this bucket
						vi.keyspace.Release()
						// ask the pool to refresh
						vi.keyspace.namespace.refresh()
						// bucket doesnt exist any more
						conn.Error(errors.NewError(nil, "bucket "+vi.keyspace.Name()+" not found"))
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

/*

func (vi *viewIndex) ValueCount() (int64, errors.Error) {
    indexItemChannel := make(catalog.EntryChannel)
    indexWarnChannel := make(query.ErrorChannel)
    indexErrorChannel := make(query.ErrorChannel)

    go vi.ScanRange(catalog.LookupValue{dparval.NewValue(nil)}, catalog.LookupValue{dparval.NewValue(nil)}, catalog.Both, 0, indexItemChannel, indexWarnChannel, indexErrorChannel)

    var err query.Error
    nullCount := int64(0)
    ok := true
    for ok {
        select {
        case _, ok = <-indexItemChannel:
            if ok {
                nullCount += 1
            }
        case _, ok = <-indexWarnChannel:
            // ignore warnings here
        case err, ok = <-indexErrorChannel:
            if err != nil {
                return 0, err
            }
        }
    }

    totalCount, err := ViewTotalRows(vi.bucket.cbbucket, vi.DDocName(), vi.ViewName(), map[string]interface{}{})
    if err != nil {
        return 0, err
    }

    return totalCount - nullCount, nil

}

*/
