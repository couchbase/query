//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package virtual

import (
	"fmt"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

type virtualKeyspace struct {
	name      string
	namespace datastore.Namespace
	indexer   datastore.Indexer
}

func NewVirtualKeyspace(name string, namespace datastore.Namespace) datastore.Keyspace {
	return &virtualKeyspace{
		name:      name,
		namespace: namespace,
		indexer:   NewVirtualIndexer(namespace.Name(), name),
	}
}

func (this *virtualKeyspace) Id() string {
	return this.name
}

func (this *virtualKeyspace) Name() string {
	return this.name
}

// Virtual keyspace will be directly under a namespace.
func (this *virtualKeyspace) NamespaceId() string {
	return this.namespace.Id()
}

func (this *virtualKeyspace) Namespace() datastore.Namespace {
	return this.namespace
}

func (this *virtualKeyspace) ScopeId() string {
	return ""
}

func (this *virtualKeyspace) Scope() datastore.Scope {
	return nil
}

func (this *virtualKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	return 0, nil
}

func (this *virtualKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return 0, nil
}

func (this *virtualKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	switch name {
	case datastore.GSI, datastore.DEFAULT:
		if this.indexer == nil {
			return nil, errors.NewVirtualKSIdxerNotFoundError(nil, "GSI indxer for virtual keyspace.")
		}
	default:
		return nil, errors.NewVirtualKSNotImplementedError(nil, fmt.Sprintf("Type %s indexer for virtual keyspace.", name))
	}
	return this.indexer, nil
}

func (this *virtualKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	if this.indexer == nil {
		return nil, errors.NewVirtualKSIdxerNotFoundError(nil, "for virtual keyspace.")
	}
	return []datastore.Indexer{this.indexer}, nil
}

func (this *virtualKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue, context datastore.QueryContext, subPath []string) []errors.Error {
	return []errors.Error{errors.NewVirtualKSNotSupportedError(nil, "Fetch for virtual keyspace.")}
}

func (this *virtualKeyspace) Insert(inserts []value.Pair) ([]value.Pair, errors.Error) {
	return nil, errors.NewVirtualKSNotSupportedError(nil, "Insert for virtual keyspace.")
}

func (this *virtualKeyspace) Update(updates []value.Pair) ([]value.Pair, errors.Error) {
	return nil, errors.NewVirtualKSNotSupportedError(nil, "Update for virtual keyspace.")
}

func (this *virtualKeyspace) Upsert(upserts []value.Pair) ([]value.Pair, errors.Error) {
	return nil, errors.NewVirtualKSNotSupportedError(nil, "Upsert for virtual keyspace.")
}

func (this *virtualKeyspace) Delete(deletes []string, context datastore.QueryContext) ([]string, errors.Error) {
	return nil, errors.NewVirtualKSNotSupportedError(nil, "Delete for virtual keyspace.")
}

func (this *virtualKeyspace) Release(close bool) {}
