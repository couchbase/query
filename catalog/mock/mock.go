//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

Package mock provides a fake, mock 100%-in-memory implementation of
the catalog package, which can be useful for testing.  Because it is
memory-oriented, performance testing of higher layers may be easier
with this mock catalog.

*/
package mock

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/couchbaselabs/query/catalog"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

const (
	DEFAULT_NUM_NAMESPACES = 1
	DEFAULT_NUM_KEYSPACES  = 1
	DEFAULT_NUM_ITEMS      = 100000
)

// site is the root for the mock-based Site.
type site struct {
	path           string
	namespaces     map[string]*namespace
	namespaceNames []string
	params         map[string]int
}

func (s *site) Id() string {
	return s.URL()
}

func (s *site) URL() string {
	return "mock:" + s.path
}

func (s *site) NamespaceIds() ([]string, errors.Error) {
	return s.NamespaceNames()
}

func (s *site) NamespaceNames() ([]string, errors.Error) {
	return s.namespaceNames, nil
}

func (s *site) NamespaceById(id string) (p catalog.Namespace, e errors.Error) {
	return s.NamespaceByName(id)
}

func (s *site) NamespaceByName(name string) (p catalog.Namespace, e errors.Error) {
	p, ok := s.namespaces[name]
	if !ok {
		p, e = nil, errors.NewError(nil, "Namespace "+name+" not found.")
	}

	return
}

// namespace represents a mock-based Namespace.
type namespace struct {
	site          *site
	name          string
	keyspaces     map[string]*keyspace
	keyspaceNames []string
}

func (p *namespace) SiteId() string {
	return p.site.Id()
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

func (p *namespace) KeyspaceById(id string) (b catalog.Keyspace, e errors.Error) {
	return p.KeyspaceByName(id)
}

func (p *namespace) KeyspaceByName(name string) (b catalog.Keyspace, e errors.Error) {
	b, ok := p.keyspaces[name]
	if !ok {
		b, e = nil, errors.NewError(nil, "Keyspace "+name+" not found.")
	}

	return
}

// keyspace is a mock-based keyspace.
type keyspace struct {
	namespace *namespace
	name      string
	nitems    int
	indexes   map[string]catalog.Index
	primary   catalog.PrimaryIndex
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
	return int64(b.nitems), nil
}

func (b *keyspace) IndexIds() ([]string, errors.Error) {
	return b.IndexNames()
}

func (b *keyspace) IndexNames() ([]string, errors.Error) {
	rv := make([]string, 0, len(b.indexes))
	for name, _ := range b.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (b *keyspace) IndexById(id string) (catalog.Index, errors.Error) {
	return b.IndexByName(id)
}

func (b *keyspace) IndexByName(name string) (catalog.Index, errors.Error) {
	index, ok := b.indexes[name]
	if !ok {
		return nil, errors.NewError(nil, fmt.Sprintf("Index %v not found.", name))
	}
	return index, nil
}

func (b *keyspace) IndexByPrimary() (catalog.PrimaryIndex, errors.Error) {
	return b.primary, nil
}

func (b *keyspace) Indexes() ([]catalog.Index, errors.Error) {
	rv := make([]catalog.Index, 0, len(b.indexes))
	for _, index := range b.indexes {
		rv = append(rv, index)
	}
	return rv, nil
}

func (b *keyspace) CreatePrimaryIndex() (catalog.PrimaryIndex, errors.Error) {
	if b.primary != nil {
		return b.primary, nil
	}

	return nil, errors.NewError(nil, "Not supported.")
}

func (b *keyspace) CreateIndex(name string, equalKey, rangeKey expression.Expressions, using catalog.IndexType) (catalog.Index, errors.Error) {
	return nil, errors.NewError(nil, "Not supported.")
}

func (b *keyspace) Fetch(keys []string) ([]catalog.Pair, errors.Error) {
	rv := make([]catalog.Pair, len(keys))
	for i, k := range keys {
		item, e := b.FetchOne(k)
		if e != nil {
			return nil, e
		}

		rv[i].Key = k
		rv[i].Value = item
	}

	return rv, nil
}

func (b *keyspace) FetchOne(key string) (value.Value, errors.Error) {
	i, e := strconv.Atoi(key)
	if e != nil {
		return nil, errors.NewError(e, fmt.Sprintf("no mock item: %v", key))
	} else {
		return genItem(i, b.nitems)
	}
}

// generate a mock document - used by FetchOne to mock a document in the keyspace
func genItem(i int, nitems int) (value.Value, errors.Error) {
	if i < 0 || i >= nitems {
		return nil, errors.NewError(nil,
			fmt.Sprintf("item out of mock range: %v [0,%v)", i, nitems))
	}
	id := strconv.Itoa(i)
	doc := value.NewAnnotatedValue(map[string]interface{}{"id": id, "i": float64(i)})
	doc.SetAttachment("meta", map[string]interface{}{"id": id})
	return doc, nil
}

func (b *keyspace) Insert(inserts []catalog.Pair) ([]catalog.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *keyspace) Update(updates []catalog.Pair) ([]catalog.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *keyspace) Upsert(upserts []catalog.Pair) ([]catalog.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewError(nil, "Not yet implemented.")
}

func (b *keyspace) Delete(deletes []string) errors.Error {
	// FIXME
	return errors.NewError(nil, "Not yet implemented.")
}

func (b *keyspace) Release() {
}

// NewSite creates a new mock site for the given "path".  The path has
// prefix "mock:", with the rest of the path treated as a
// comma-separated key=value params.  For example:
//     mock:namespaces=2,keyspaces=5,items=50000
// The above means 2 namespaces.
// And, each namespace has 5 keyspaces.
// And, each keyspace with 50000 items.
// By default, you get...
//     mock:namespaces=1,keyspaces=1,items=100000
// Which is what you'd get by specifying a path of just...
//     mock:
func NewSite(path string) (catalog.Site, errors.Error) {
	if strings.HasPrefix(path, "mock:") {
		path = path[5:]
	}
	params := map[string]int{}
	for _, kv := range strings.Split(path, ",") {
		if kv == "" {
			continue
		}
		pair := strings.Split(kv, "=")
		v, e := strconv.Atoi(pair[1])
		if e != nil {
			return nil, errors.NewError(e,
				fmt.Sprintf("could not parse mock param key: %s, val: %s",
					pair[0], pair[1]))
		}
		params[pair[0]] = v
	}
	nnamespaces := paramVal(params, "namespaces", DEFAULT_NUM_NAMESPACES)
	nkeyspaces := paramVal(params, "keyspaces", DEFAULT_NUM_KEYSPACES)
	nitems := paramVal(params, "items", DEFAULT_NUM_ITEMS)
	s := &site{path: path, params: params, namespaces: map[string]*namespace{}, namespaceNames: []string{}}
	for i := 0; i < nnamespaces; i++ {
		p := &namespace{site: s, name: "p" + strconv.Itoa(i), keyspaces: map[string]*keyspace{}, keyspaceNames: []string{}}
		for j := 0; j < nkeyspaces; j++ {
			b := &keyspace{namespace: p, name: "b" + strconv.Itoa(j), nitems: nitems,
				indexes: map[string]catalog.Index{}}
			pi := &primaryIndex{name: "all_docs", keyspace: b}
			b.primary = pi
			b.indexes["all_docs"] = pi
			p.keyspaces[b.name] = b
			p.keyspaceNames = append(p.keyspaceNames, b.name)
		}
		s.namespaces[p.name] = p
		s.namespaceNames = append(s.namespaceNames, p.name)
	}
	return s, nil
}

func paramVal(params map[string]int, key string, defaultVal int) int {
	v, ok := params[key]
	if ok {
		return v
	}
	return defaultVal
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

func (pi *primaryIndex) Type() catalog.IndexType {
	return catalog.UNSPECIFIED
}

func (pi *primaryIndex) Drop() errors.Error {
	return errors.NewError(nil, "This primary index cannot be dropped.")
}

func (pi *primaryIndex) EqualKey() expression.Expressions {
	return nil
}

func (pi *primaryIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *primaryIndex) Condition() expression.Expression {
	return nil
}

func (pi *primaryIndex) Statistics(span *catalog.Span) (catalog.Statistics, errors.Error) {
	return nil, nil
}

func (pi *primaryIndex) Scan(span *catalog.Span, distinct bool, limit int64, conn *catalog.IndexConnection) {
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
			conn.SendError(errors.NewError(nil, fmt.Sprintf("Invalid lower bound %v of type %T.", a, a)))
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
			conn.SendError(errors.NewError(nil, fmt.Sprintf("Invalid upper bound %v of type %T.", a, a)))
			return
		}
	}

	if limit == 0 {
		limit = int64(pi.keyspace.nitems)
	}
	for i := 0; i < pi.keyspace.nitems && int64(i) < limit; i++ {
		id := strconv.Itoa(i)

		if low != "" &&
			(id < low ||
				(id == low && (span.Range.Inclusion&catalog.LOW == 0))) {
			continue
		}

		low = ""

		if high != "" &&
			(id > high ||
				(id == high && (span.Range.Inclusion&catalog.HIGH == 0))) {
			break
		}

		entry := catalog.IndexEntry{PrimaryKey: id}
		conn.EntryChannel() <- &entry
	}
}

func (pi *primaryIndex) ScanEntries(limit int64, conn *catalog.IndexConnection) {
	defer close(conn.EntryChannel())

	if limit == 0 {
		limit = int64(pi.keyspace.nitems)
	}

	for i := 0; i < pi.keyspace.nitems && int64(i) < limit; i++ {
		entry := catalog.IndexEntry{PrimaryKey: strconv.Itoa(i)}
		conn.EntryChannel() <- &entry
	}
}
