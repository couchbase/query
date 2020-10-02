//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

Package file provides a file-based implementation of the datastore
package.

*/
package file

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/virtual"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

// datastore is the root for the file-based Datastore.
type store struct {
	path           string
	namespaces     map[string]*namespace
	namespaceNames []string
	inferencer     datastore.Inferencer // what we use to infer schemas

	users map[string]*datastore.User
}

func (s *store) Id() string {
	return s.path
}

func (s *store) URL() string {
	return "file://" + s.path
}

func (s *store) Info() datastore.Info {
	return &infoImpl{}
}

type infoImpl struct {
}

func (i *infoImpl) Version() string {
	return util.VERSION
}

func (info *infoImpl) Topology() ([]string, []errors.Error) {
	return []string{}, nil
}

func (info *infoImpl) Services(node string) (map[string]interface{}, []errors.Error) {
	return map[string]interface{}{}, nil
}

func (s *store) NamespaceIds() ([]string, errors.Error) {
	return s.NamespaceNames()
}

func (s *store) NamespaceNames() ([]string, errors.Error) {
	return s.namespaceNames, nil
}

func (s *store) NamespaceById(id string) (p datastore.Namespace, e errors.Error) {
	return s.NamespaceByName(id)
}

func (s *store) NamespaceByName(name string) (p datastore.Namespace, e errors.Error) {
	p, ok := s.namespaces[strings.ToUpper(name)]
	if !ok {
		e = errors.NewFileNamespaceNotFoundError(nil, name)
	}

	return
}

func (s *store) Authorize(*auth.Privileges, *auth.Credentials) (auth.AuthenticatedUsers, errors.Error) {
	return nil, nil
}

func (s *store) PreAuthorize(*auth.Privileges) {
}

func (s *store) CredsString(req *http.Request) string {
	return ""
}

func (s *store) SetLogLevel(level logging.Level) {
	// No-op. Uses query engine logger.
}

// Ignore the name parameter for now
func (s *store) Inferencer(name datastore.InferenceType) (datastore.Inferencer, errors.Error) {
	return s.inferencer, nil
}

func (s *store) Inferencers() ([]datastore.Inferencer, errors.Error) {
	return []datastore.Inferencer{s.inferencer}, nil
}

func (s *store) StatUpdater() (datastore.StatUpdater, errors.Error) {
	return nil, errors.NewOtherNotImplementedError(nil, "UPDATE STATISTICS")
}

func (s *store) SetConnectionSecurityConfig(conSecConfig *datastore.ConnectionSecurityConfig) {
	// Do nothing.
}

func (s *store) AuditInfo() (*datastore.AuditInfo, errors.Error) {
	return nil, errors.NewOtherNotImplementedError(nil, "AuditInfo")
}

func (s *store) ProcessAuditUpdateStream(callb func(uid string) error) errors.Error {
	return errors.NewOtherNotImplementedError(nil, "ProcessAuditUpdateStream")
}

func (s *store) UserInfo() (value.Value, errors.Error) {
	// Return an array of no users.
	jsonData := make([]interface{}, 0)
	v := value.NewValue(jsonData)
	return v, nil
}

func (s *store) GetUserInfoAll() ([]datastore.User, errors.Error) {
	ret := make([]datastore.User, 0, len(s.users))
	for _, v := range s.users {
		ret = append(ret, *v)
	}
	return ret, nil
}

func (s *store) PutUserInfo(u *datastore.User) errors.Error {
	s.users[u.Id] = u
	return nil
}

func (s *store) GetRolesAll() ([]datastore.Role, errors.Error) {
	return []datastore.Role{
		datastore.Role{Name: "cluster_admin"},
		datastore.Role{Name: "replication_admin"},
		datastore.Role{Name: "bucket_admin", Target: "*"},
	}, nil
}

func (s *store) CreateSystemCBOStats(requestId string) errors.Error {
	return nil
}

func (s *store) GetSystemCBOStats() (datastore.Keyspace, errors.Error) {
	return nil, nil
}

func (s *store) HasSystemCBOStats() (bool, errors.Error) {
	return false, nil
}

func (s *store) StartTransaction(stmtAtomicity bool, context datastore.QueryContext) (map[string]bool, errors.Error) {
	return nil, errors.NewTranDatastoreNotSupportedError("file")
}

func (s *store) CommitTransaction(stmtAtomicity bool, context datastore.QueryContext) errors.Error {
	return errors.NewTranDatastoreNotSupportedError("file")
}

func (s *store) RollbackTransaction(stmtAtomicity bool, context datastore.QueryContext, sname string) errors.Error {
	return errors.NewTranDatastoreNotSupportedError("file")
}

func (s *store) SetSavepoint(stmtAtomicity bool, context datastore.QueryContext, sname string) errors.Error {
	return errors.NewTranDatastoreNotSupportedError("file")
}

func (s *store) TransactionDeltaKeyScan(keyspace string, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()
}

// NewStore creates a new file-based store for the given filepath.
func NewDatastore(path string) (s datastore.Datastore, e errors.Error) {
	path, er := filepath.Abs(path)
	if er != nil {
		return nil, errors.NewFileDatastoreError(er, "")
	}

	fs := &store{path: path, users: make(map[string]*datastore.User, 4)}

	e = fs.loadNamespaces()
	if e != nil {
		return
	}

	// get the schema inferencer
	var err errors.Error
	fs.inferencer, err = GetDefaultInferencer(fs)
	if err != nil {
		return nil, err
	}

	s = fs
	return
}

func (s *store) loadNamespaces() (e errors.Error) {
	dirEntries, er := ioutil.ReadDir(s.path)
	if er != nil {
		return errors.NewFileDatastoreError(er, "")
	}

	s.namespaces = make(map[string]*namespace, len(dirEntries))
	s.namespaceNames = make([]string, 0, len(dirEntries))

	var p *namespace
	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			s.namespaceNames = append(s.namespaceNames, dirEntry.Name())
			diru := strings.ToUpper(dirEntry.Name())
			if _, ok := s.namespaces[diru]; ok {
				return errors.NewFileDuplicateNamespaceError(nil, dirEntry.Name())
			}

			p, e = newNamespace(s, dirEntry.Name())
			if e != nil {
				return
			}

			s.namespaces[diru] = p
		}
	}

	return
}

// namespace represents a file-based Namespace.
type namespace struct {
	store         *store
	name          string
	keyspaces     map[string]*keyspace
	keyspaceNames []string
}

func (p *namespace) DatastoreId() string {
	return p.store.Id()
}

func (p *namespace) Id() string {
	return p.Name()
}

func (p *namespace) Name() string {
	return p.name
}

func (p *namespace) KeyspaceIds() ([]string, errors.Error) {
	return p.KeyspaceNames()
}

func (p *namespace) KeyspaceNames() ([]string, errors.Error) {
	return p.keyspaceNames, nil
}

func (p *namespace) Objects() ([]datastore.Object, errors.Error) {
	rv := make([]datastore.Object, len(p.keyspaceNames))
	i := 0
	for _, k := range p.keyspaceNames {
		rv[i] = datastore.Object{Id: k, Name: k, IsKeyspace: true}
		i++
	}
	return rv, nil
}

func (p *namespace) KeyspaceById(id string) (b datastore.Keyspace, e errors.Error) {
	return p.KeyspaceByName(id)
}

func (p *namespace) KeyspaceByName(name string) (b datastore.Keyspace, e errors.Error) {
	b, ok := p.keyspaces[strings.ToUpper(name)]
	if !ok {
		e = errors.NewFileKeyspaceNotFoundError(nil, name)
	}

	return
}

func (p *namespace) VirtualKeyspaceByName(path []string) (datastore.Keyspace, errors.Error) {
	return virtual.NewVirtualKeyspace(p, path)
}

func (p *namespace) MetadataVersion() uint64 {
	return 0
}

func (p *namespace) MetadataId() string {
	return p.name
}

func (p *namespace) path() string {
	return filepath.Join(p.store.path, p.name)
}

// newNamespace creates a new namespace.
func newNamespace(s *store, dir string) (p *namespace, e errors.Error) {
	p = new(namespace)
	p.store = s
	p.name = dir

	e = p.loadKeyspaces()
	return
}

func (p *namespace) loadKeyspaces() (e errors.Error) {
	dirEntries, er := ioutil.ReadDir(p.path())
	if er != nil {
		return errors.NewFileDatastoreError(er, "")
	}

	p.keyspaces = make(map[string]*keyspace, len(dirEntries))
	p.keyspaceNames = make([]string, 0, len(dirEntries))

	var b *keyspace
	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			diru := strings.ToUpper(dirEntry.Name())
			if _, ok := p.keyspaces[diru]; ok {
				return errors.NewFileDuplicateKeyspaceError(nil, dirEntry.Name())
			}

			b, e = newKeyspace(p, dirEntry.Name())
			if e != nil {
				return
			}

			p.keyspaces[diru] = b
			p.keyspaceNames = append(p.keyspaceNames, b.Name())
		}
	}

	return
}

func (p *namespace) BucketIds() ([]string, errors.Error) {
	return datastore.NO_STRINGS, nil
}

func (p *namespace) BucketNames() ([]string, errors.Error) {
	return datastore.NO_STRINGS, nil
}

func (p *namespace) BucketById(name string) (datastore.Bucket, errors.Error) {
	return nil, errors.NewOtherNoBuckets("file")
}

func (p *namespace) BucketByName(name string) (datastore.Bucket, errors.Error) {
	return nil, errors.NewOtherNoBuckets("file")
}

// keyspace is a file-based keyspace.
type keyspace struct {
	namespace *namespace
	name      string
	fi        datastore.Indexer
	fileLock  sync.Mutex
}

func (b *keyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *keyspace) Namespace() datastore.Namespace {
	return b.namespace
}

func (b *keyspace) Id() string {
	return b.Name()
}

func (b *keyspace) Name() string {
	return b.name
}

func (b *keyspace) QualifiedName() string {
	return b.namespace.name + ":" + b.name
}

func (b *keyspace) AuthKey() string {
	return b.name
}

func (b *keyspace) Scope() datastore.Scope {
	return nil
}

func (b *keyspace) ScopeId() string {
	return ""
}

func (b *keyspace) MetadataVersion() uint64 {
	return 0
}

func (b *keyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	dirEntries, er := ioutil.ReadDir(b.path())
	if er != nil {
		return 0, errors.NewFileDatastoreError(er, "")
	}
	return int64(len(dirEntries)), nil
}

func (b *keyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	dirEntries, er := ioutil.ReadDir(b.path())
	if er != nil {
		return 0, errors.NewFileDatastoreError(er, "")
	}
	var size int64
	for _, ent := range dirEntries {
		size += ent.Size()
	}
	return size, nil
}

func (b *keyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.fi, nil
}

func (b *keyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.fi}, nil
}

func (b *keyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string) []errors.Error {
	var errs []errors.Error

	for _, k := range keys {
		item, e := b.fetchOne(k)

		if e != nil {
			if os.IsNotExist(e.Cause()) {
				// file doesn't exist => key denotes non-existent doc => ignore it
				continue
			}
			if errs == nil {
				errs = make([]errors.Error, 0, 1)
			}
			errs = append(errs, e)
			continue
		}

		if item != nil {
			item.SetId(k)
		}

		keysMap[k] = item
	}

	return errs
}

func (b *keyspace) fetchOne(key string) (value.AnnotatedValue, errors.Error) {
	path := filepath.Join(b.path(), key+".json")
	item, e := fetch(path)
	if e != nil {
		item = nil
	}

	return item, e
}

const (
	INSERT = 0x01
	UPDATE = 0x02
	UPSERT = 0x04
)

func opToString(op int) string {

	switch op {
	case INSERT:
		return "insert"
	case UPDATE:
		return "update"
	case UPSERT:
		return "upsert"
	}

	return "unknown operation"
}

func (b *keyspace) performOp(op int, kvPairs []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {

	if len(kvPairs) == 0 {
		return nil, errors.NewFileNoKeysInsertError(nil, "keyspace "+b.Name())
	}

	insertedKeys := make([]value.Pair, 0)
	var returnErr errors.Error

	// this lock can be mode more granular FIXME
	b.fileLock.Lock()
	defer b.fileLock.Unlock()

	for _, kv := range kvPairs {
		var file *os.File
		var err error

		key := kv.Name
		value, _ := json.Marshal(kv.Value.Actual())
		filename := filepath.Join(b.path(), key+".json")

		switch op {

		case INSERT:
			// add the key only if it doesn't exist
			if _, err = os.Stat(filename); err == nil {
				err = errors.NewFileKeyExists(nil, "Key (File) "+filename)
			} else {
				// create and write the file
				if file, err = os.Create(filename); err == nil {
					_, err = file.Write(value)
					file.Close()
				}
			}
		case UPDATE:
			// add the key only if it doesn't exist
			if _, err = os.Stat(filename); err == nil {
				// open and write the file
				if file, err = os.OpenFile(filename, os.O_TRUNC|os.O_RDWR, 0666); err == nil {
					_, err = file.Write(value)
					file.Close()
				}
			}

		case UPSERT:
			// open the file for writing, if doesn't exist then create
			if file, err = os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666); err == nil {
				_, err = file.Write(value)
				file.Close()
			}
		}

		if err != nil {
			returnErr = errors.NewFileDMLError(returnErr, opToString(op)+" Failed "+err.Error())
		} else {
			insertedKeys = append(insertedKeys, kv)
		}
	}

	return insertedKeys, returnErr

}

func (b *keyspace) Insert(inserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return b.performOp(INSERT, inserts, context)
}

func (b *keyspace) Update(updates []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return b.performOp(UPDATE, updates, context)
}

func (b *keyspace) Upsert(upserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return b.performOp(UPSERT, upserts, context)
}

func (b *keyspace) Delete(deletes []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {

	var fileError []string
	var deleted []value.Pair
	for _, pair := range deletes {
		key := pair.Name
		filename := filepath.Join(b.path(), key+".json")
		if err := os.Remove(filename); err != nil {
			if !os.IsNotExist(err) {
				fileError = append(fileError, err.Error())
			}
		} else {
			deleted = append(deleted, pair)
		}
	}

	if len(fileError) > 0 {
		errLine := fmt.Sprintf("Delete failed on some keys %v", fileError)
		return deleted, errors.NewFileDatastoreError(nil, errLine)
	}

	return deleted, nil
}

func (b *keyspace) Release(close bool) {
}

func (b *keyspace) CreateScope(name string) errors.Error {
	return errors.NewScopesNotSupportedError(b.name)
}

func (b *keyspace) DropScope(name string) errors.Error {
	return errors.NewScopesNotSupportedError(b.name)
}

func (b *keyspace) Flush() errors.Error {
	return errors.NewNoFlushError(b.name)
}

func (b *keyspace) IsBucket() bool {
	return true
}

func (b *keyspace) path() string {
	return filepath.Join(b.namespace.path(), b.name)
}

// newKeyspace creates a new keyspace.
func newKeyspace(p *namespace, dir string) (b *keyspace, e errors.Error) {
	b = new(keyspace)
	b.namespace = p
	b.name = dir

	fi, er := os.Stat(b.path())
	if er != nil {
		return nil, errors.NewFileDatastoreError(er, "")
	}

	if !fi.IsDir() {
		return nil, errors.NewFileKeyspaceNotDirError(nil, "Keyspace path "+dir)
	}

	b.fi = newFileIndexer(b)
	b.fi.CreatePrimaryIndex("", "#primary", nil)

	return
}

type fileIndexer struct {
	keyspace *keyspace
	indexes  map[string]datastore.Index
	primary  datastore.PrimaryIndex
}

func newFileIndexer(keyspace *keyspace) datastore.Indexer {

	return &fileIndexer{
		keyspace: keyspace,
		indexes:  make(map[string]datastore.Index),
	}
}

func (fi *fileIndexer) BucketId() string {
	return ""
}

func (fi *fileIndexer) ScopeId() string {
	return ""
}

func (fi *fileIndexer) KeyspaceId() string {
	return fi.keyspace.Id()
}

func (fi *fileIndexer) Name() datastore.IndexType {
	return datastore.DEFAULT
}

func (fi *fileIndexer) IndexIds() ([]string, errors.Error) {
	rv := make([]string, 0, len(fi.indexes))
	for name, _ := range fi.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (fi *fileIndexer) IndexNames() ([]string, errors.Error) {
	rv := make([]string, 0, len(fi.indexes))
	for name, _ := range fi.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (fi *fileIndexer) IndexById(id string) (datastore.Index, errors.Error) {
	return fi.IndexByName(id)
}

func (fi *fileIndexer) IndexByName(name string) (datastore.Index, errors.Error) {
	index, ok := fi.indexes[name]
	if !ok {
		return nil, errors.NewFileIdxNotFound(nil, name)
	}
	return index, nil
}

func (fi *fileIndexer) PrimaryIndexes() ([]datastore.PrimaryIndex, errors.Error) {
	return []datastore.PrimaryIndex{fi.primary}, nil
}

func (fi *fileIndexer) Indexes() ([]datastore.Index, errors.Error) {
	return []datastore.Index{fi.primary}, nil
}

func (fi *fileIndexer) CreatePrimaryIndex(requestId, name string, with value.Value) (
	datastore.PrimaryIndex, errors.Error) {
	if fi.primary == nil {
		pi := new(primaryIndex)
		fi.primary = pi
		pi.keyspace = fi.keyspace
		pi.name = name
		pi.indexer = fi
		fi.indexes[pi.name] = pi
	}

	return fi.primary, nil
}

func (b *fileIndexer) CreateIndex(requestId, name string, seekKey, rangeKey expression.Expressions,
	where expression.Expression, with value.Value) (datastore.Index, errors.Error) {
	return nil, errors.NewFileNotSupported(nil, "CREATE INDEX is not supported for file-based datastore.")
}

func (b *fileIndexer) BuildIndexes(requestId string, names ...string) errors.Error {
	return errors.NewFileNotSupported(nil, "BUILD INDEXES is not supported for file-based datastore.")
}

func (b *fileIndexer) Refresh() errors.Error {
	return nil
}

func (b *fileIndexer) MetadataVersion() uint64 {
	return 0
}

func (b *fileIndexer) SetLogLevel(level logging.Level) {
	// No-op, uses query engine logger
}

func (b *fileIndexer) SetConnectionSecurityConfig(conSecConfig *datastore.ConnectionSecurityConfig) {
	// Do nothing.
}

// primaryIndex performs full keyspace scans.
type primaryIndex struct {
	name     string
	keyspace *keyspace
	indexer  *fileIndexer
}

func (pi *primaryIndex) BucketId() string {
	return ""
}

func (pi *primaryIndex) ScopeId() string {
	return ""
}

func (pi *primaryIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *primaryIndex) Id() string {
	return pi.Name()
}

func (pi *primaryIndex) Name() string {
	return pi.name
}

func (pi *primaryIndex) Type() datastore.IndexType {
	return datastore.DEFAULT
}

func (pi *primaryIndex) Indexer() datastore.Indexer {
	return pi.indexer
}

func (pi *primaryIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *primaryIndex) RangeKey() expression.Expressions {
	// FIXME
	return nil
}

func (pi *primaryIndex) Condition() expression.Expression {
	return nil
}

func (pi *primaryIndex) IsPrimary() bool {
	return true
}

func (pi *primaryIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *primaryIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *primaryIndex) Drop(requestId string) errors.Error {
	return errors.NewFilePrimaryIdxNoDropError(nil, pi.Name())
}

func (pi *primaryIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()

	// For primary indexes, bounds must always be strings, so we
	// can just enforce that directly
	low, high := "", ""

	// Ensure that lower bound is a string, if any
	if len(span.Range.Low) > 0 {
		a := span.Range.Low[0].Actual()
		switch a := a.(type) {
		case string:
			low = a
		default:
			conn.Error(errors.NewFileDatastoreError(nil, fmt.Sprintf("Invalid lower bound %v of type %T.", a, a)))
			return
		}
	}

	// Ensure that upper bound is a string, if any
	if len(span.Range.High) > 0 {
		a := span.Range.High[0].Actual()
		switch a := a.(type) {
		case string:
			high = a
		default:
			conn.Error(errors.NewFileDatastoreError(nil, fmt.Sprintf("Invalid upper bound %v of type %T.", a, a)))
			return
		}
	}

	dirEntries, er := ioutil.ReadDir(pi.keyspace.path())
	if er != nil {
		conn.Error(errors.NewFileDatastoreError(er, ""))
		return
	}

	var n int64 = 0
	for _, dirEntry := range dirEntries {

		logging.Debugf("Dir entry being scanned <ud>%v</ud>", dirEntry.Name())
		if limit > 0 && n > limit {
			break
		}

		id := documentPathToId(dirEntry.Name())

		if low != "" &&
			(id < low ||
				(id == low && (span.Range.Inclusion&datastore.LOW == 0))) {
			continue
		}

		low = ""

		if high != "" &&
			(id > high ||
				(id == high && (span.Range.Inclusion&datastore.HIGH == 0))) {
			break
		}

		if !dirEntry.IsDir() {
			entry := datastore.IndexEntry{PrimaryKey: id}
			conn.Sender().SendEntry(&entry)
			n++
		}
	}
}

func (pi *primaryIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()

	dirEntries, er := ioutil.ReadDir(pi.keyspace.path())
	if er != nil {
		conn.Error(errors.NewFileDatastoreError(er, ""))
		return
	}

	for i, dirEntry := range dirEntries {
		if limit > 0 && int64(i) > limit {
			break
		}
		if !dirEntry.IsDir() {
			entry := datastore.IndexEntry{PrimaryKey: documentPathToId(dirEntry.Name())}
			conn.Sender().SendEntry(&entry)
		}
	}
}

func fetch(path string) (item value.AnnotatedValue, e errors.Error) {
	bytes, er := ioutil.ReadFile(path)
	if er != nil {
		return nil, errors.NewFileDatastoreError(er, "")
	}

	doc := value.NewAnnotatedValue(value.NewValue(bytes))
	doc.SetId(documentPathToId(path))
	item = doc

	return
}

func documentPathToId(p string) string {
	_, file := filepath.Split(p)
	ext := filepath.Ext(file)
	return file[0 : len(file)-len(ext)]
}
