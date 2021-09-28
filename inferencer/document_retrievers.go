// Copyright 2019-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
// file, in accordance with the Business Source License, use of this software will
// be governed by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package inferencer

import (
	"fmt"
	"sort"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/primitives/couchbase"
	"github.com/couchbase/query/value"
)

const _EMPTY_KEY = ""
const _KEYS_NOT_FOUND = 5
const _MAX_DUPLICATES = 100

//
// we need an interface describing access methods for getting a set
// of documents, since we might be getting them using random sampling via KV,
// and we also might get them using a primary index
//

type DocumentRetriever interface {
	GetNextDoc(context datastore.QueryContext) (string, value.Value, errors.Error) // returns nil for value when done
}

////////////////////////////////////////////////////////////////////////////////
// PrimaryIndexDocumentRetriever implementation
//
// Given a datastore with a primary index, use that index to retrieve
// documents.
////////////////////////////////////////////////////////////////////////////////

type PrimaryIndexDocumentRetriever struct {
	ks           datastore.Keyspace
	docIds       []string
	nextToReturn int
	sampleSize   int
}

func (pidr *PrimaryIndexDocumentRetriever) GetNextDoc(context datastore.QueryContext) (string, value.Value, errors.Error) {
	// have we reached the end of the set of primary keys?
	if pidr.nextToReturn >= len(pidr.docIds) {
		return _EMPTY_KEY, nil, nil
	}

	// retrieve the next key
	key := pidr.docIds[pidr.nextToReturn : pidr.nextToReturn+1]
	pidr.nextToReturn++
	docs := make(map[string]value.AnnotatedValue, 1)
	errs := pidr.ks.Fetch(key, docs, datastore.NULL_QUERY_CONTEXT, nil)

	if errs != nil {
		return _EMPTY_KEY, nil, errs[0]
	} else if len(docs) == 0 {
		return _EMPTY_KEY, nil, nil
	}

	return key[0], docs[key[0]], nil
}

func MakePrimaryIndexDocumentRetriever(ks datastore.Keyspace, optSampleSize int) (*PrimaryIndexDocumentRetriever, errors.Error) {
	pidr := new(PrimaryIndexDocumentRetriever)
	pidr.ks = ks
	pidr.docIds = make([]string, 0)
	pidr.nextToReturn = 0

	// how many docs in the keyspace?
	docCount, err := ks.Count(datastore.NULL_QUERY_CONTEXT)

	if err != nil {
		return nil, errors.NewInferKeyspaceError(ks.Name(), err)
	}

	// if they specified 0 for the sampleSize, set the limit to the number of documents
	if optSampleSize == 0 {
		pidr.sampleSize = int(docCount)
	} else {
		pidr.sampleSize = optSampleSize
	}

	indexer, err := ks.Indexer(datastore.GSI)
	if err != nil {
		return nil, errors.NewInferKeyspaceError(ks.Name(), err)
	}

	primaryIndexes, err := indexer.PrimaryIndexes()
	if err == nil {
		for _, index := range primaryIndexes {
			// make sure that the index is online
			state, _, err := index.State()
			if err != nil || state != datastore.ONLINE {
				continue
			}

			// if we get this far, the index should be good and online
			conn := datastore.NewIndexConnection(datastore.NULL_CONTEXT)
			go index.ScanEntries("retriever", int64(pidr.sampleSize), datastore.UNBOUNDED, nil, conn)

			for len(pidr.docIds) < pidr.sampleSize {
				entry, _ := conn.Sender().GetEntry()
				if entry == nil {
					break
				}
				pidr.docIds = append(pidr.docIds, entry.PrimaryKey)
			}
			return pidr, nil
		}
	}

	return nil, errors.NewInferNoSuitablePrimaryIndex(ks.Name())
}

////////////////////////////////////////////////////////////////////////////////
// AnyIndexDocumentRetriever implementation
//
// Rank indexes preferring GSI, unconditional, non-array with the shortest key.
// Then in order, fill sample by retrieving a unique set of keys from the indices
// stopping when sample size has been reached.
////////////////////////////////////////////////////////////////////////////////

type AnyIndexDocumentRetriever struct {
	ks         datastore.Keyspace
	docIds     map[string]bool
	sampleSize int
	Indexes    []string
}

func (aidr *AnyIndexDocumentRetriever) GetNextDoc(context datastore.QueryContext) (string, value.Value, errors.Error) {
	if len(aidr.docIds) == 0 {
		return _EMPTY_KEY, nil, nil
	}

	var key []string
	for k, _ := range aidr.docIds {
		key = []string{k}
		delete(aidr.docIds, k)
		break
	}
	docs := make(map[string]value.AnnotatedValue, 1)
	errs := aidr.ks.Fetch(key, docs, datastore.NULL_QUERY_CONTEXT, nil)

	if errs != nil {
		return _EMPTY_KEY, nil, errs[0]
	} else if len(docs) == 0 {
		return _EMPTY_KEY, nil, nil
	}

	return key[0], docs[key[0]], nil
}

type indexArray []datastore.Index3

func (this indexArray) Len() int {
	return len(this)
}

func (this indexArray) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

func (this indexArray) Less(i, j int) bool {
	// prefer without condition
	if this[i].Condition() == nil && this[j].Condition() != nil {
		return true
	} else if this[i].Condition() != nil && this[j].Condition() == nil {
		return false
	}
	// prefer without array keys
	rki := this[i].RangeKey2()
	rkj := this[j].RangeKey2()
	aki := false
	for _, k := range rki {
		ak, _, _ := k.Expr.IsArrayIndexKey()
		aki = aki || ak
	}
	akj := false
	for _, k := range rkj {
		ak, _, _ := k.Expr.IsArrayIndexKey()
		akj = akj || ak
	}
	if !aki && akj {
		return true
	} else if aki && !akj {
		return false
	}
	// prefer fewest keys
	if len(rki) < len(rkj) {
		return true
	}
	// for consistency, lastly order by name
	if this[i].Name() < this[j].Name() {
		return true
	}
	return false
}

func MakeAnyIndexDocumentRetriever(ks datastore.Keyspace, optSampleSize int) (*AnyIndexDocumentRetriever, errors.Error) {
	aidr := new(AnyIndexDocumentRetriever)
	aidr.ks = ks
	aidr.docIds = make(map[string]bool, 0)

	docCount, err := ks.Count(datastore.NULL_QUERY_CONTEXT)

	if err != nil {
		return nil, errors.NewInferKeyspaceError(ks.Name(), err)
	}

	// if they specified 0 for the sampleSize, set the limit to the number of documents
	if optSampleSize == 0 {
		aidr.sampleSize = int(docCount)
	} else {
		aidr.sampleSize = optSampleSize
	}

	indexer, err := ks.Indexer(datastore.GSI)
	if err != nil {
		return nil, errors.NewInferKeyspaceError(ks.Name(), err)
	}

	ranges := append(datastore.Ranges2(nil), &datastore.Range2{
		Low:       nil,
		High:      nil,
		Inclusion: datastore.BOTH,
	})
	spans := append(datastore.Spans2(nil), &datastore.Span2{
		Seek:   nil,
		Ranges: ranges,
	})
	ilist, err := indexer.Indexes()
	if err == nil {
		indexes := make(indexArray, 0, len(ilist))
		for i, _ := range ilist {
			if state, _, err := ilist[i].State(); err == nil && state == datastore.ONLINE {
				if i3, ok := ilist[i].(datastore.Index3); ok {
					indexes = append(indexes, i3)
				}
			}
		}
		sort.Sort(indexes)

		for _, index := range indexes {
			if index == nil {
				continue
			}

			aidr.Indexes = append(aidr.Indexes, index.Name())
			// if we get this far, the index should be good and online
			conn := datastore.NewIndexConnection(datastore.NULL_CONTEXT)
			go index.Scan3("retriever", spans, false, false, &datastore.IndexProjection{PrimaryKey: true}, 0,
				int64(aidr.sampleSize), nil, nil, datastore.UNBOUNDED, nil, conn)

			for len(aidr.docIds) < aidr.sampleSize {
				entry, _ := conn.Sender().GetEntry()
				if entry == nil {
					break
				}
				aidr.docIds[entry.PrimaryKey] = true
			}
			if len(aidr.docIds) >= aidr.sampleSize {
				break
			}
		}
	}

	if len(aidr.docIds) == 0 {
		return nil, errors.NewInferNoSuitableSecondaryIndex(ks.Name())
	}

	return aidr, nil
}

////////////////////////////////////////////////////////////////////////////////
// KeyspaceRandomDocumentRetriever implementation
//
// Given a query/datastore/keyspace, use the GetRandomDoc() method to retrieve
// non-duplicate docs until we have retrieved sampleSize (or give up because
// we keep seeing duplicates.
////////////////////////////////////////////////////////////////////////////////

type KeyspaceRandomDocumentRetriever struct {
	ks         datastore.Keyspace
	rdr        datastore.RandomEntryProvider
	docIdsSeen map[string]bool
	sampleSize int
}

func (krdr *KeyspaceRandomDocumentRetriever) GetNextDoc(context datastore.QueryContext) (string, value.Value, errors.Error) {
	// have we returned as many documents as were requested?
	if len(krdr.docIdsSeen) >= krdr.sampleSize {
		return _EMPTY_KEY, nil, nil
	}

	// try to retrieve the next document
	duplicatesSeen := 0
	for duplicatesSeen < _MAX_DUPLICATES {
		key, value, err := krdr.rdr.GetRandomEntry(context) // get the doc
		if err != nil {                                     // check for errors
			return _EMPTY_KEY, nil, errors.NewInferRandomError(err)
		}

		// MB-42205 this may need improvement: a nil value only means that we run on of documents
		// on the last node we queried. There may be corner cases with many nodes and few documents
		// where we get a KEY_NOENT from one node, but there are more documents in other nodes.
		// Try at up to _MAX_DUPLICATES times if value is nil.
		// https://github.com/couchbase/kv_engine/blob/master/docs/BinaryProtocol.md#0xb6-get-random-key
		// resident items only using a randomised vbucket as a start point and then randomised hash-tables
		// buckets for searching within a vbucket.

		if value == nil || krdr.docIdsSeen[key] { // seen it before?
			duplicatesSeen++
			continue
		}

		krdr.docIdsSeen[key] = true
		return key, value, nil // new doc, return
	}

	// if we get here, we saw duplicate docs or nil keys _MAX_DUPLICATES times so we give up on finding any more new docs
	return _EMPTY_KEY, nil, nil
}

func MakeKeyspaceRandomDocumentRetriever(ks datastore.Keyspace, sampleSize int) (*KeyspaceRandomDocumentRetriever, errors.Error) {

	var ok bool
	krdr := new(KeyspaceRandomDocumentRetriever)
	krdr.rdr, ok = ks.(datastore.RandomEntryProvider)
	if !ok {
		return nil, errors.NewInferNoRandomEntryProvider(ks.Name())
	}

	i := 0
	for i = 0; i < _KEYS_NOT_FOUND; i++ {
		_, val, _ := krdr.rdr.GetRandomEntry(nil)
		if val != nil {
			break
		}
	}
	if i == _KEYS_NOT_FOUND {
		return nil, errors.NewInferNoRandomDocuments(ks.Name())
	}

	krdr.ks = ks
	krdr.docIdsSeen = make(map[string]bool)
	krdr.sampleSize = sampleSize

	return krdr, nil
}

////////////////////////////////////////////////////////////////////////////////
// KVRandomDocumentRetriever implementation
//
// Given a server name, login & password, and bucket name and password,
// use the couchbase bucket GetRandomDoc() method to retrieve
// non-duplicate radom docs until we have sampleSize (or give up because we
// keep seeing duplicates).
////////////////////////////////////////////////////////////////////////////////

type KVRandomDocumentRetriever struct {
	docIdsSeen map[string]bool
	sampleSize int
	bucket     *couchbase.Bucket
}

func (kvrdr *KVRandomDocumentRetriever) GetNextDoc(context datastore.QueryContext) (string, value.Value, errors.Error) {
	// have we returned as many documents as were requested?
	if len(kvrdr.docIdsSeen) >= kvrdr.sampleSize {
		return _EMPTY_KEY, nil, nil
	}

	// try to retrieve the next document
	duplicatesSeen := 0
	for duplicatesSeen < 100 {
		resp, err := kvrdr.bucket.GetRandomDoc()

		if err != nil {
			return _EMPTY_KEY, nil, errors.NewInferRandomError(err)
		}

		key := fmt.Sprintf("%s", resp.Key)
		val := value.NewValue(resp.Body)

		if kvrdr.docIdsSeen[key] { // seen it before?
			duplicatesSeen++
			continue
		}

		kvrdr.docIdsSeen[key] = true
		return key, val, nil // new doc, return
	}

	// if we get here, we saw duplicate docs 100 times in a row, so we give up on finding any more new docs
	return _EMPTY_KEY, nil, nil
}

func MakeKVRandomDocumentRetriever(serverURL, bucket, bucketPass string, sampleSize int) (*KVRandomDocumentRetriever, errors.Error) {

	kvrdr := new(KVRandomDocumentRetriever)
	kvrdr.docIdsSeen = make(map[string]bool)
	kvrdr.sampleSize = sampleSize

	var client couchbase.Client
	var err error

	client, err = couchbase.Connect(serverURL)
	if err != nil {
		return nil, errors.NewInferConnectFailed(serverURL, err)
	}

	pool, err := client.GetPool("default")
	if err != nil {
		return nil, errors.NewInferGetPoolFailed(err)
	}

	kvrdr.bucket, err = pool.GetBucket(bucket)
	if err != nil {
		return nil, errors.NewInferGetBucketFailed(bucket, err)
	}

	return kvrdr, nil
}
