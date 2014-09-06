//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

// +build ignore

package couchbase

import (
	"sync"

	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
)

// TODO:
// 1. Access to metadata repository.
// 2. How to get notified when ever index metadata / topology
//    get changes.

// ErrorSecondaryKeyspaceEmpty if parent keyspace is found nil.
var ErrorSecondaryKeyspaceEmpty = errors.NewError(nil, "couchbase.secondary.empty")

// ErrorSecondaryIndexNotFound if index is not found by `id` or `name`.
var ErrorSecondaryIndexNotFound = errors.NewError(nil, "couchbase.secondary.indexNotFound")

// ErrorPrimaryIndexNotFound if primary index is not created.
var ErrorPrimaryIndexNotFound = errors.NewError(nil, "couchbase.secondary.primaryNotFound")

// secondaryIndexes provide a list of indexes that are
// created and available via secondary indexing system.
// Provide method that can be used by couchbase Keyspace{}
// interface.
type secondaryIndexes struct {
	mu           sync.Mutex
	nameSpace    string
	keySpace     string                     // parent keyspace that will hold this structure.
	indexes      map[string]*secondaryIndex // index-id -> *secondaryIndex
	primaryIndex *secondaryIndex            // will also be present in `indexes` field.
	repoaddrs    []string
}

func newSecondaryIndexes(nameSpace, keySpace string, repoaddrs []string) *secondaryIndexes {
	indxs := &secondaryIndexes{
		nameSpace:    nameSpace,
		keySpace:     keySpace,
		indexes:      make(map[string]*secondaryIndex),
		primaryIndex: nil,
		repoaddrs:    repoaddrs,
	}
	return indxs
}

// CreateIndex create a new index using secondary-indexing system.
func (indxs *secondaryIndexes) CreateIndex(
	name string, // index name
	equalKey, rangeKey expression.Expressions, // evaluators
	using datastore.IndexType, // index backend
) (*secondaryIndex, errors.Error) {

	if indxs == nil {
		return nil, ErrorSecondaryKeyspaceEmpty
	}

	// TODO: expression.Expression must be marshalable.
	// partnExpr, err := expression.NewStringer().Visit(equalKey)
	partnExpr, err := "", errors.Error(nil)
	if err != nil {
		return nil, err
	}
	// TODO: expression.Expression must be marshalable.
	// secExprs, err := expression.NewStringer().Visit(rangeKey.Marshal())
	secExprs, err := "", errors.Error(nil)
	if err != nil {
		return nil, err
	}

	index := &secondaryIndex{
		name:      name,
		keySpace:  indxs.keySpace,
		isPrimary: false,
		using:     using,
		partnExpr: partnExpr,
		secExprs:  secExprs,
		indxs:     indxs,
	}

	// TODO: post this index to repository and get back a valid DefnId.
	// and index location address.
	// index = repoaddr.CreateIndex(index)

	// TODO:
	// defID, host fields will be updated by secondary-index coordinator
	// and returned back.

	indxs.mu.Lock()
	defer indxs.mu.Unlock()

	indxs.indexes[index.Name()] = index
	return index, nil
}

func (indxs *secondaryIndexes) CreatePrimaryIndex() (*secondaryIndex, errors.Error) {
	if indxs == nil {
		return nil, ErrorSecondaryKeyspaceEmpty
	}
	index := &secondaryIndex{
		name:      "PRIMARY_SEC", // TODO: move this to couchbase.go
		keySpace:  indxs.keySpace,
		isPrimary: true,
		using:     "FORESTDB", // TODO: move this to couchbase.go
		indxs:     indxs,
	}
	// TODO: post this index to repository and get back a valid DefnId.
	// and index location address.
	// index = repos.CreateIndex(index)

	// TODO:
	// defID, host fields will be updated by secondary-index coordinator
	// and returned back.

	indxs.mu.Lock()
	defer indxs.mu.Unlock()

	indxs.indexes[index.Name()] = index
	indxs.primaryIndex = index
	return index, nil
}

func (indxs *secondaryIndexes) IndexIds() ([]string, errors.Error) {
	indxs.mu.Lock()
	defer indxs.mu.Unlock()

	if indxs == nil {
		return nil, ErrorSecondaryKeyspaceEmpty
	}

	ids := make([]string, 0, len(indxs.indexes))
	for _, index := range indxs.indexes {
		ids = append(ids, index.Id())
	}
	return ids, nil
}

func (indxs *secondaryIndexes) IndexNames() ([]string, errors.Error) {
	indxs.mu.Lock()
	defer indxs.mu.Unlock()

	if indxs == nil {
		return nil, ErrorSecondaryKeyspaceEmpty
	}

	names := make([]string, 0, len(indxs.indexes))
	for name := range indxs.indexes {
		names = append(names, name)
	}
	return names, nil
}

func (indxs *secondaryIndexes) IndexById(id string) (*secondaryIndex, errors.Error) {
	return indxs.IndexByName(id)
}

func (indxs *secondaryIndexes) IndexByName(name string) (*secondaryIndex, errors.Error) {
	indxs.mu.Lock()
	defer indxs.mu.Unlock()

	if indxs == nil {
		return nil, ErrorSecondaryKeyspaceEmpty
	} else if index, ok := indxs.indexes[name]; ok {
		return index, nil
	}
	return nil, ErrorSecondaryIndexNotFound
}

func (indxs *secondaryIndexes) IndexByPrimary() (*secondaryIndex, errors.Error) {
	indxs.mu.Lock()
	defer indxs.mu.Unlock()

	if indxs == nil {
		return nil, ErrorSecondaryKeyspaceEmpty

	} else if indxs.primaryIndex == nil {
		return nil, ErrorPrimaryIndexNotFound
	}
	return indxs.primaryIndex, nil
}

func (indxs *secondaryIndexes) Indexes() ([]*secondaryIndex, errors.Error) {
	indxs.mu.Lock()
	defer indxs.mu.Unlock()

	if indxs.primaryIndex == nil {
		return nil, ErrorPrimaryIndexNotFound
	}
	indexes := make([]*secondaryIndex, 0, len(indxs.indexes))
	for _, index := range indxs.indexes {
		indexes = append(indexes, index)
	}
	return indexes, nil
}

// local function that can be used to asynchronously update
// meta-data information from coordinator notifications.

func (indxs *secondaryIndexes) updateRepoaddrs(repoaddrs []string) {
	indxs.mu.Lock()
	defer indxs.mu.Unlock()

	indxs.repoaddrs = repoaddrs
}

func (indxs *secondaryIndexes) updateIndex(index *secondaryIndex, isPrimary bool) {
	indxs.mu.Lock()
	defer indxs.mu.Unlock()

	indxs.indexes[index.Name()] = index
	if isPrimary {
		indxs.primaryIndex = index
	}
}

func (indxs *secondaryIndexes) deleteIndex(index *secondaryIndex) (err errors.Error) {
	indxs.mu.Lock()
	defer indxs.mu.Unlock()

	// TODO: connect with meta-data repository and delete this index.
	// err = repoaddr.DeleteIndex(index)
	if err != nil {
		delete(indxs.indexes, index.Name())
	}
	return
}
