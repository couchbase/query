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
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

// datastore is the root for the file-based Datastore.
type store struct {
	path           string
	namespaces     map[string]*namespace
	namespaceNames []string
}

func (s *store) Id() string {
	return s.path
}

func (s *store) URL() string {
	return "file://" + s.path
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
		e = errors.NewError(nil, "Namespace "+name+" not found.")
	}

	return
}

// NewStore creates a new file-based store for the given filepath.
func NewDatastore(path string) (s datastore.Datastore, e errors.Error) {
	path, er := filepath.Abs(path)
	if er != nil {
		return nil, errors.NewError(er, "")
	}

	fs := &store{path: path}

	e = fs.loadNamespaces()
	if e != nil {
		return
	}

	s = fs
	return
}

func (s *store) loadNamespaces() (e errors.Error) {
	dirEntries, er := ioutil.ReadDir(s.path)
	if er != nil {
		return errors.NewError(er, "")
	}

	s.namespaces = make(map[string]*namespace, len(dirEntries))
	s.namespaceNames = make([]string, 0, len(dirEntries))

	var p *namespace
	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			s.namespaceNames = append(s.namespaceNames, dirEntry.Name())
			diru := strings.ToUpper(dirEntry.Name())
			if _, ok := s.namespaces[diru]; ok {
				return errors.NewError(nil, "Duplicate namespace name "+dirEntry.Name())
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

func (p *namespace) KeyspaceById(id string) (b datastore.Keyspace, e errors.Error) {
	return p.KeyspaceByName(id)
}

func (p *namespace) KeyspaceByName(name string) (b datastore.Keyspace, e errors.Error) {
	b, ok := p.keyspaces[strings.ToUpper(name)]
	if !ok {
		e = errors.NewError(nil, "Keyspace "+name+" not found.")
	}

	return
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
		return errors.NewError(er, "")
	}

	p.keyspaces = make(map[string]*keyspace, len(dirEntries))
	p.keyspaceNames = make([]string, 0, len(dirEntries))

	var b *keyspace
	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			diru := strings.ToUpper(dirEntry.Name())
			if _, ok := p.keyspaces[diru]; ok {
				return errors.NewError(nil, "Duplicate keyspace name "+dirEntry.Name())
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

func (b *keyspace) Id() string {
	return b.Name()
}

func (b *keyspace) Name() string {
	return b.name
}

func (b *keyspace) Count() (int64, errors.Error) {
	dirEntries, er := ioutil.ReadDir(b.path())
	if er != nil {
		return 0, errors.NewError(er, "")
	}
	return int64(len(dirEntries)), nil
}

func (b *keyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.fi, nil
}

func (b *keyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	indexers := make([]datastore.Indexer, 0, 1)
	return append(indexers, b.fi), nil
}

func (b *keyspace) IndexIds() ([]string, errors.Error) {
	return b.fi.IndexIds()
}

func (b *keyspace) IndexNames() ([]string, errors.Error) {
	return b.fi.IndexNames()
}

func (b *keyspace) IndexById(id string) (datastore.Index, errors.Error) {
	return b.fi.IndexByName(id)
}

func (b *keyspace) IndexByName(name string) (datastore.Index, errors.Error) {
	return b.fi.IndexByName(name)
}

func (b *keyspace) IndexByPrimary() (datastore.PrimaryIndex, errors.Error) {
	return b.fi.IndexByPrimary()
}

func (b *keyspace) Indexes() ([]datastore.Index, errors.Error) {
	return b.fi.Indexes()
}

func (b *keyspace) Authenticate(credentials datastore.Credentials, requested datastore.Privileges) errors.Error {
	return nil
}

func (b *keyspace) CreatePrimaryIndex(using datastore.IndexType) (datastore.PrimaryIndex, errors.Error) {
	return b.fi.CreatePrimaryIndex()
}

func (b *keyspace) CreateIndex(name string, equalKey, rangeKey expression.Expressions,
	where expression.Expression, using datastore.IndexType) (datastore.Index, errors.Error) {
	return b.fi.CreateIndex(name, equalKey, rangeKey, where)
}

func (b *keyspace) Fetch(keys []string) ([]datastore.AnnotatedPair, errors.Error) {
	rv := make([]datastore.AnnotatedPair, len(keys))
	nils_count := 0
	for i, k := range keys {
		item, e := b.fetchOne(k)
		if item == nil {
			// Keep track of nils - they will be removed from slice
			nils_count++
		}

		if e != nil {
			return nil, e
		}

		rv[i].Key = k
		rv[i].Value = item
	}

	if nils_count > 0 {
		_rv := make([]datastore.AnnotatedPair, len(keys)-nils_count)
		i := 0
		for _, k := range rv {
			if k.Value != nil {
				_rv[i].Key = k.Key
				_rv[i].Value = k.Value
				i++
			}
		}
		rv = _rv
	}
	return rv, nil
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

func (b *keyspace) performOp(op int, kvPairs []datastore.Pair) ([]datastore.Pair, errors.Error) {

	if len(kvPairs) == 0 {
		return nil, errors.NewError(nil, "No keys to insert")
	}

	insertedKeys := make([]datastore.Pair, 0)
	var returnErr errors.Error

	// this lock can be mode more granular FIXME
	b.fileLock.Lock()
	defer b.fileLock.Unlock()

	for _, kv := range kvPairs {
		var file *os.File
		var err error

		key := kv.Key
		value, _ := json.Marshal(kv.Value.Actual())
		filename := filepath.Join(b.path(), key+".json")

		switch op {

		case INSERT:
			// add the key only if it doesn't exist
			if _, err = os.Stat(filename); err == nil {
				err = errors.NewError(nil, "File "+filename+" exists")
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
			returnErr = errors.NewError(returnErr, opToString(op)+" Failed "+err.Error())
		} else {
			insertedKeys = append(insertedKeys, kv)
		}
	}

	return insertedKeys, returnErr

}

func (b *keyspace) Insert(inserts []datastore.Pair) ([]datastore.Pair, errors.Error) {
	return b.performOp(INSERT, inserts)
}

func (b *keyspace) Update(updates []datastore.Pair) ([]datastore.Pair, errors.Error) {
	return b.performOp(UPDATE, updates)
}

func (b *keyspace) Upsert(upserts []datastore.Pair) ([]datastore.Pair, errors.Error) {
	return b.performOp(UPSERT, upserts)
}

func (b *keyspace) Delete(deletes []string) ([]string, errors.Error) {

	var fileError []string
	var deleted []string
	for _, key := range deletes {
		filename := filepath.Join(b.path(), key+".json")
		if err := os.Remove(filename); err != nil {
			if !os.IsNotExist(err) {
				fileError = append(fileError, err.Error())
			}
		} else {
			deleted = append(deleted, key)
		}
	}

	if len(fileError) > 0 {
		errLine := fmt.Sprintf("Delete failed on some keys %v", fileError)
		return deleted, errors.NewError(nil, errLine)
	}

	return deleted, nil
}

func (b *keyspace) Release() {
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
		return nil, errors.NewError(er, "")
	}

	if !fi.IsDir() {
		return nil, errors.NewError(nil, "Keyspace path must be a directory.")
	}

	b.fi = newFileIndexer(b)
	b.fi.CreatePrimaryIndex()

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

func (fi *fileIndexer) KeyspaceId() string {
	return fi.keyspace.Id()
}

func (fi *fileIndexer) Name() datastore.IndexType {
	return datastore.UNSPECIFIED
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
		return nil, errors.NewError(nil, fmt.Sprintf("Index %v not found.", name))
	}
	return index, nil
}

func (fi *fileIndexer) IndexByPrimary() (datastore.PrimaryIndex, errors.Error) {
	return fi.primary, nil
}

func (fi *fileIndexer) Indexes() ([]datastore.Index, errors.Error) {
	rv := make([]datastore.Index, 0, len(fi.indexes))
	for _, index := range fi.indexes {
		rv = append(rv, index)
	}
	return rv, nil
}

func (fi *fileIndexer) CreatePrimaryIndex() (datastore.PrimaryIndex, errors.Error) {
	if fi.primary == nil {
		pi := new(primaryIndex)
		fi.primary = pi
		pi.keyspace = fi.keyspace
		pi.name = "#primary"
		fi.indexes[pi.name] = pi
	}

	return fi.primary, nil
}

func (b *fileIndexer) CreateIndex(name string, equalKey, rangeKey expression.Expressions,
	where expression.Expression) (datastore.Index, errors.Error) {
	return nil, errors.NewError(nil, "Create index is not supported for file-based datastore.")
}

// primaryIndex performs full keyspace scans.
type primaryIndex struct {
	name     string
	keyspace *keyspace
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
	return datastore.UNSPECIFIED
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

func (pi *primaryIndex) State() (datastore.IndexState, errors.Error) {
	return datastore.ONLINE, nil
}

func (pi *primaryIndex) Statistics(span *datastore.Span) (datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *primaryIndex) Drop() errors.Error {
	return errors.NewError(nil, "This primary index cannot be dropped.")
}

func (pi *primaryIndex) Scan(span *datastore.Span, distinct bool, limit int64, conn *datastore.IndexConnection) {
	defer close(conn.EntryChannel())

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
			conn.Error(errors.NewError(nil, fmt.Sprintf("Invalid lower bound %v of type %T.", a, a)))
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
			conn.Error(errors.NewError(nil, fmt.Sprintf("Invalid upper bound %v of type %T.", a, a)))
			return
		}
	}

	dirEntries, er := ioutil.ReadDir(pi.keyspace.path())
	if er != nil {
		conn.Error(errors.NewError(er, ""))
		return
	}

	var n int64 = 0
	for _, dirEntry := range dirEntries {

		fmt.Printf("Dir entry being scanned %v", dirEntry.Name())
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
			conn.EntryChannel() <- &entry
			n++
		}
	}
}

func (pi *primaryIndex) ScanEntries(limit int64, conn *datastore.IndexConnection) {
	defer close(conn.EntryChannel())

	dirEntries, er := ioutil.ReadDir(pi.keyspace.path())
	if er != nil {
		conn.Error(errors.NewError(er, ""))
		return
	}

	for i, dirEntry := range dirEntries {
		if limit > 0 && int64(i) > limit {
			break
		}
		if !dirEntry.IsDir() {
			entry := datastore.IndexEntry{PrimaryKey: documentPathToId(dirEntry.Name())}
			conn.EntryChannel() <- &entry
		}
	}
}

func fetch(path string) (item value.AnnotatedValue, e errors.Error) {
	bytes, er := ioutil.ReadFile(path)
	if er != nil {
		if os.IsNotExist(er) {
			// file doesn't exist should simply return nil, nil
			return
		}
		return nil, errors.NewError(er, "")
	}

	doc := value.NewAnnotatedValue(value.NewValue(bytes))
	doc.SetAttachment("meta", map[string]interface{}{"id": documentPathToId(path)})
	item = doc

	return
}

func documentPathToId(p string) string {
	_, file := filepath.Split(p)
	ext := filepath.Ext(file)
	return file[0 : len(file)-len(ext)]
}
