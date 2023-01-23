//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package virtual

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

// Implement Interface Indexer{}
type VirtualIndexer struct {
	path    []string
	indexes map[string]datastore.Index
}

func NewVirtualIndexer(path []string) datastore.Indexer {
	return &VirtualIndexer{
		path:    path,
		indexes: make(map[string]datastore.Index, 1),
	}
}

func (this *VirtualIndexer) BucketId() string {
	if len(this.path) > 2 {
		return this.path[1]
	}
	return ""
}

func (this *VirtualIndexer) ScopeId() string {
	if len(this.path) > 2 {
		return this.path[2]
	}
	return ""
}

func (this *VirtualIndexer) KeyspaceId() string {
	return this.path[len(this.path)-1]
}

func (this *VirtualIndexer) Name() datastore.IndexType {
	return datastore.GSI
}

func (this *VirtualIndexer) IndexIds() ([]string, errors.Error) {
	rv := make([]string, 0, len(this.indexes))
	for _, v := range this.indexes {
		rv = append(rv, v.Name())
	}
	return rv, nil
}

func (this *VirtualIndexer) IndexNames() ([]string, errors.Error) {
	rv := make([]string, 0, len(this.indexes))
	for _, v := range this.indexes {
		rv = append(rv, v.Name())
	}
	return rv, nil
}

// Virtual Indexer can only have virtual indexes, virtual index has indexname as indexid
func (this *VirtualIndexer) IndexById(id string) (datastore.Index, errors.Error) {
	return this.IndexByName(id)
}

func (this *VirtualIndexer) IndexByName(name string) (datastore.Index, errors.Error) {
	index, ok := this.indexes[name]
	if ok {
		return index, nil
	}
	return nil, errors.NewVirtualIdxNotFoundError(nil, name)
}

func (this *VirtualIndexer) PrimaryIndexes() ([]datastore.PrimaryIndex, errors.Error) {
	return nil, errors.NewVirtualIdxerNotSupportedError(nil, "No primary indexes in virtual indexer")
}

func (this *VirtualIndexer) Indexes() ([]datastore.Index, errors.Error) {
	rv := make([]datastore.Index, 0, len(this.indexes))
	for _, idx := range this.indexes {
		rv = append(rv, idx)
	}
	return rv, nil
}

func (this *VirtualIndexer) CreatePrimaryIndex(requestId, name string, with value.Value) (datastore.PrimaryIndex, errors.Error) {
	return nil, errors.NewVirtualIdxerNotSupportedError(nil, "CREATE PRIMARY INDEX is not supported for virtual indexer")
}

func (this *VirtualIndexer) CreateIndex(
	requestId, name string, seekKey, rangeKey expression.Expressions,
	where expression.Expression, with value.Value) (datastore.Index, errors.Error) {
	return nil, errors.NewVirtualIdxerNotSupportedError(nil, "CREATE INDEX is not supported for virtual indexer")
}

func (this *VirtualIndexer) BuildIndexes(requestId string, names ...string) errors.Error {
	return errors.NewVirtualIdxerNotSupportedError(nil, "BUILD INDEXES is not supported for virtual indexer")
}

func (this *VirtualIndexer) Refresh() errors.Error {
	return nil
}

func (this *VirtualIndexer) MetadataVersion() uint64 {
	return 0
}

func (this *VirtualIndexer) SetLogLevel(level logging.Level) {
}

func (this *VirtualIndexer) SetConnectionSecurityConfig(conSecConfig *datastore.ConnectionSecurityConfig) {
}

// Implement Interface Indexer2{}
func (this *VirtualIndexer) CreateIndex2(
	requestId, name string, seekKey expression.Expressions, rangeKey datastore.IndexKeys,
	where expression.Expression, with value.Value) (datastore.Index, errors.Error) {
	return nil, errors.NewVirtualIdxerNotSupportedError(nil, "CREATE INDEX is not supported for virtual indexer2")
}

// Implement Interface Indexer3{}
func (this *VirtualIndexer) CreateIndex3(
	requestId, name string, rangeKey datastore.IndexKeys, indexPartition *datastore.IndexPartition,
	where expression.Expression, with value.Value) (datastore.Index, errors.Error) {
	return nil, errors.NewVirtualIdxerNotSupportedError(nil, "CREATE INDEX is not supported for virtual indexer3")
}

func (this *VirtualIndexer) CreatePrimaryIndex3(requestId, name string, indexPartition *datastore.IndexPartition, with value.Value) (datastore.PrimaryIndex, errors.Error) {
	return nil, errors.NewVirtualIdxerNotSupportedError(nil, "CREATE PRIMARY INDEX is not supported for virtual indexer3")
}
