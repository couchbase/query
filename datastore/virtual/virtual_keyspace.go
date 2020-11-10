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

// for completeness with collections
type virtualBucket struct {
	id        string
	namespace datastore.Namespace
	scope     datastore.Scope
}

func (b *virtualBucket) Id() string {
	return b.id
}

func (b *virtualBucket) Name() string {
	return b.id
}

func (b *virtualBucket) Uid() string {
	return b.id
}

func (b *virtualBucket) AuthKey() string {
	return b.id
}

func (b *virtualBucket) NamespaceId() string {
	return b.namespace.Id()
}

func (b *virtualBucket) Namespace() datastore.Namespace {
	return b.namespace
}

func (b *virtualBucket) ScopeIds() ([]string, errors.Error) {
	return []string{b.scope.Id()}, nil
}

func (b *virtualBucket) ScopeNames() ([]string, errors.Error) {
	return []string{b.scope.Name()}, nil
}

func (b *virtualBucket) ScopeById(id string) (datastore.Scope, errors.Error) {
	if b.scope.Id() != id {
		return nil, errors.NewVirtualScopeNotFoundError(nil, id)
	}
	return b.scope, nil
}

func (b *virtualBucket) ScopeByName(name string) (datastore.Scope, errors.Error) {
	if b.scope.Name() != name {
		return nil, errors.NewVirtualScopeNotFoundError(nil, name)
	}
	return b.scope, nil
}

func (b *virtualBucket) DefaultKeyspace() (datastore.Keyspace, errors.Error) {
	return nil, errors.NewBucketNoDefaultCollectionError(b.id)
}

func (b *virtualBucket) CreateScope(name string) errors.Error {
	return errors.NewVirtualBucketCreateScopeError(name, fmt.Errorf("not supported by virtual buckets"))
}

func (b *virtualBucket) DropScope(name string) errors.Error {
	return errors.NewVirtualBucketDropScopeError(name, fmt.Errorf("not supported by virtual buckets"))
}

// for completeness with collections
type virtualScope struct {
	id       string
	bucket   *virtualBucket
	keyspace *virtualKeyspace
}

func (sc *virtualScope) Id() string {
	return sc.id
}

func (sc *virtualScope) Name() string {
	return sc.id
}

func (sc *virtualScope) Uid() string {
	return sc.id
}

func (sc *virtualScope) AuthKey() string {
	return sc.id
}

func (sc *virtualScope) BucketId() string {
	return sc.bucket.Id()
}

func (sc *virtualScope) Bucket() datastore.Bucket {
	return sc.bucket
}

func (sc *virtualScope) KeyspaceIds() ([]string, errors.Error) {
	return []string{sc.keyspace.Id()}, nil
}

func (sc *virtualScope) KeyspaceNames() ([]string, errors.Error) {
	return []string{sc.keyspace.Name()}, nil
}

func (sc *virtualScope) KeyspaceById(id string) (datastore.Keyspace, errors.Error) {
	if sc.keyspace.Id() != id {
		return nil, errors.NewVirtualKeyspaceNotFoundError(nil, id)
	}
	return sc.keyspace, nil
}

func (sc *virtualScope) KeyspaceByName(name string) (datastore.Keyspace, errors.Error) {
	if sc.keyspace.Name() != name {
		return nil, errors.NewVirtualKeyspaceNotFoundError(nil, name)
	}
	return sc.keyspace, nil
}

func (sc *virtualScope) CreateCollection(name string) errors.Error {
	return errors.NewCbBucketCreateCollectionError(name, fmt.Errorf("not supported by virtual scopes"))
}

func (sc *virtualScope) DropCollection(name string) errors.Error {
	return errors.NewCbBucketDropCollectionError(name, fmt.Errorf("not supported by virtual scopes"))
}

type virtualKeyspace struct {
	path      []string
	namespace datastore.Namespace
	indexer   datastore.Indexer
	scope     datastore.Scope
}

func NewVirtualKeyspace(namespace datastore.Namespace, path []string) (datastore.Keyspace, errors.Error) {
	if len(path) != 2 && len(path) != 4 {
		return nil, errors.NewDatastoreInvalidPathError("")
	}

	rv := &virtualKeyspace{
		path:      path,
		namespace: namespace,
		indexer:   NewVirtualIndexer(path),
	}
	if len(path) == 4 {
		scope := &virtualScope{id: path[2], keyspace: rv}
		bucket := &virtualBucket{id: path[1], namespace: namespace, scope: scope}
		scope.bucket = bucket
		rv.scope = scope
	}
	return rv, nil
}

func (this *virtualKeyspace) Id() string {
	return this.path[len(this.path)-1]
}

func (this *virtualKeyspace) Name() string {
	return this.path[len(this.path)-1]
}

func (this *virtualKeyspace) Uid() string {
	return this.path[len(this.path)-1]
}

func (this *virtualKeyspace) QualifiedName() string {
	if len(this.path) == 2 {
		return this.path[0] + ":" + this.path[1]
	}
	return this.path[0] + ":" + this.path[1] + "." + this.path[2] + "." + this.path[3]
}

func (this *virtualKeyspace) AuthKey() string {
	return this.path[len(this.path)-1]
}

// Virtual keyspace will be directly under a namespace.
func (this *virtualKeyspace) NamespaceId() string {
	return this.path[0]
}

func (this *virtualKeyspace) Namespace() datastore.Namespace {
	return this.namespace
}

func (this *virtualKeyspace) ScopeId() string {
	if len(this.path) == 4 {
		return this.path[2]
	}
	return ""
}

func (this *virtualKeyspace) Scope() datastore.Scope {
	return this.scope
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

func (this *virtualKeyspace) Insert(inserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return nil, errors.NewVirtualKSNotSupportedError(nil, "Insert for virtual keyspace.")
}

func (this *virtualKeyspace) Update(updates []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return nil, errors.NewVirtualKSNotSupportedError(nil, "Update for virtual keyspace.")
}

func (this *virtualKeyspace) Upsert(upserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return nil, errors.NewVirtualKSNotSupportedError(nil, "Upsert for virtual keyspace.")
}

func (this *virtualKeyspace) Delete(deletes []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return nil, errors.NewVirtualKSNotSupportedError(nil, "Delete for virtual keyspace.")
}

func (this *virtualKeyspace) Release(close bool) {}

func (this *virtualKeyspace) Flush() errors.Error {
	return errors.NewVirtualKSNotSupportedError(nil, "Flush for virtual keyspace.")
}

func (this *virtualKeyspace) IsBucket() bool {
	return len(this.path) == 2
}
