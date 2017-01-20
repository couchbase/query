//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package system

import (
	"github.com/couchbase/query/clustering"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type nodeKeyspace struct {
	namespace *namespace
	name      string
	si        datastore.Indexer
}

func (b *nodeKeyspace) Release() {
}

func (b *nodeKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *nodeKeyspace) Id() string {
	return b.Name()
}

func (b *nodeKeyspace) Name() string {
	return b.name
}

// TODO scan all node types
func (b *nodeKeyspace) Count() (int64, errors.Error) {
	var count int64 = 0
	cm := _CONFIGSTORE.ConfigurationManager()
	clusters, err := cm.GetClusters()
	if err != nil {
		return count, err
	}

	for _, c := range clusters {
		queryNodes, err := c.QueryNodeNames()
		if err != nil {
			return count, err
		}

		count += int64(len(queryNodes))

	}
	return count, nil
}

func (b *nodeKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.si, nil
}

func (b *nodeKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.si}, nil
}

// TODO do all node types
// TODO go-couchbase should also have a proper node map
func (b *nodeKeyspace) Fetch(keys []string) ([]value.AnnotatedPair, []errors.Error) {
	var errs []errors.Error
	rv := make([]value.AnnotatedPair, 0, len(keys))

	cm := _CONFIGSTORE.ConfigurationManager()
	clusters, err := cm.GetClusters()
	if err != nil {
		return rv, appendError(errs, err)
	}

	for _, c := range clusters {
		clm := c.ClusterManager()
		queryNodes, err := clm.GetQueryNodes()
		if err != nil {
			errs = appendError(errs, err)
			continue
		}

	loop:
		for _, queryNode := range queryNodes {
			var k string

			for _, k = range keys {

				if makeKey(c, queryNode.Name()) == k {
					item := value.NewAnnotatedValue(map[string]interface{}{

						// TODO fields by type
						"name":     k,
						"endpoint": queryNode.QueryEndpoint(),
					})
					item.SetAttachment("meta", map[string]interface{}{
						"id": k,
					})

					rv = append(rv, value.AnnotatedPair{
						Name:  k,
						Value: item,
					})
					continue loop
				}
			}
			errs = appendError(errs, errors.NewSystemDatastoreError(nil, "Key Not Found "+k))
		}
	}

	return rv, errs
}

func makeKey(c clustering.Cluster, n string) string {
	return c.Name() + "." + n
}

func appendError(errs []errors.Error, err errors.Error) []errors.Error {
	if errs == nil {
		errs = make([]errors.Error, 0, 1)
	}
	errs = append(errs, err)
	return errs
}

func (b *nodeKeyspace) Insert(inserts []value.Pair) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *nodeKeyspace) Update(updates []value.Pair) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *nodeKeyspace) Upsert(upserts []value.Pair) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *nodeKeyspace) Delete(deletes []string) ([]string, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func newNodesKeyspace(p *namespace) (*nodeKeyspace, errors.Error) {
	b := new(nodeKeyspace)
	b.namespace = p
	b.name = KEYSPACE_NAME_NODES

	primary := &nodeIndex{name: "#primary", keyspace: b}
	b.si = newSystemIndexer(b, primary)

	return b, nil
}

type nodeIndex struct {
	name     string
	keyspace *nodeKeyspace
}

func (pi *nodeIndex) KeyspaceId() string {
	return pi.name
}

func (pi *nodeIndex) Id() string {
	return pi.Name()
}

func (pi *nodeIndex) Name() string {
	return pi.name
}

func (pi *nodeIndex) Type() datastore.IndexType {
	return datastore.DEFAULT
}

func (pi *nodeIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *nodeIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *nodeIndex) Condition() expression.Expression {
	return nil
}

func (pi *nodeIndex) IsPrimary() bool {
	return true
}

func (pi *nodeIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *nodeIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *nodeIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, pi.Name())
}

func (pi *nodeIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	pi.ScanEntries(requestId, limit, cons, vector, conn)
}

func (pi *nodeIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer close(conn.EntryChannel())

	// TODO - stub configstore should return no entries
	cm := _CONFIGSTORE.ConfigurationManager()
	clusters, err := cm.GetClusters()

	// TODO no error management in scans?
	if err != nil {
		return
	}

	for _, c := range clusters {
		queryNodes, err := c.QueryNodeNames()

		// TODO ditto
		if err != nil {
			continue
		}

		// TODO all node types
		for _, queryNode := range queryNodes {
			entry := datastore.IndexEntry{PrimaryKey: makeKey(c, queryNode)}
			conn.EntryChannel() <- &entry
		}

	}

}
