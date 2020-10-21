package inferencer

import (
	"fmt"
	"math"

	"github.com/couchbase/go-couchbase"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/value"
)

//
// we need an interface describing access methods for getting a set
// of documents, since we might be getting them using random sampling via KV,
// and we also might get them using a primary index
//

const _EMPTY_KEY = ""

type DocumentRetriever interface {
	GetNextDoc() (string, value.Value, *string) // returns nil for value when done
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

func (pidr *PrimaryIndexDocumentRetriever) GetNextDoc() (string, value.Value, *string) {
	// have we reached the end of the set of primary keys?
	if pidr.nextToReturn >= len(pidr.docIds) {
		return _EMPTY_KEY, nil, nil
	}

	// retrieve the next key
	key := pidr.docIds[pidr.nextToReturn : pidr.nextToReturn+1]
	pidr.nextToReturn++
	docs := make(map[string]value.AnnotatedValue, 1)
	errs := pidr.ks.Fetch(key, docs, datastore.NULL_QUERY_CONTEXT, nil)

	if errs != nil || len(docs) == 0 {
		error_msg := fmt.Sprintf("Error fetching documents id %s: %s\n", key, errs)
		return _EMPTY_KEY, nil, &error_msg
	}

	return key[0], docs[key[0]], nil
}

func MakePrimaryIndexDocumentRetriever(ks datastore.Keyspace, optSampleSize int) (*PrimaryIndexDocumentRetriever, *string) {
	pidr := new(PrimaryIndexDocumentRetriever)
	pidr.ks = ks
	pidr.docIds = make([]string, 0)
	pidr.nextToReturn = 0

	// how many docs in the keyspace?

	docCount, err := ks.Count(datastore.NULL_QUERY_CONTEXT)

	if err != nil {
		error_msg := fmt.Sprintf(" Got error for keyspace %s: %s\n", ks.Name(), err)
		return nil, &error_msg
	}

	// if they specified 0 for the sampleSize, set the limit to the number of documents
	if optSampleSize == 0 {
		pidr.sampleSize = int(docCount)
	} else {
		pidr.sampleSize = optSampleSize
	}

	// get indexers for the keyspace

	indexers, err := ks.Indexers()
	if err != nil {
		error_msg := fmt.Sprintf(" Got error getting indexers for keyspace %s: %s\n", ks.Name(), err)
		return nil, &error_msg
	}

	// loop through the indexers for the keyspace to find a primary key
	foundPrimaryKey := false

	for _, iValue := range indexers {
		primaryIndexes, err := iValue.PrimaryIndexes()
		if err != nil {
			//fmt.Printf(" Got error getting primary indexes for keyspace %s: %s\n", ks.Name(), err)
			continue
		}

		//
		// find a primary key, get ~samepleSize keys
		//

		for _, pkValue := range primaryIndexes {
			// make sure that the index is online
			state, _, err := pkValue.State()
			if err != nil || state != datastore.ONLINE {
				continue
			}

			// if we get this far, the index should be good and online
			foundPrimaryKey = true
			conn := datastore.NewIndexConnection(datastore.NULL_CONTEXT)
			go pkValue.ScanEntries("retriever", math.MaxInt64, datastore.UNBOUNDED, nil, conn)

			var entry *datastore.IndexEntry

			ok := true
			for ok {
				if len(pidr.docIds) >= pidr.sampleSize {
					break
				}

				entry, _ = conn.Sender().GetEntry()
				if entry != nil {
					pidr.docIds = append(pidr.docIds, entry.PrimaryKey)
				} else {
					ok = false
				}
			} // end for ok

			break // if we found one primary key, we don't need to continue

		} // end for primaryIndexes
	}

	if !foundPrimaryKey {
		error_msg := fmt.Sprintf("No primary key found in bucket: %s!", ks.Name())
		return nil, &error_msg
	}

	//fmt.Printf("Made PrimaryIndexDocumentRetriever\n")

	return pidr, nil
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

func (krdr *KeyspaceRandomDocumentRetriever) GetNextDoc() (string, value.Value, *string) {
	// have we returned as many documents as were requested?
	if len(krdr.docIdsSeen) >= krdr.sampleSize {
		return _EMPTY_KEY, nil, nil
	}

	// try to retrieve the next document
	duplicatesSeen := 0
	for duplicatesSeen < 100 {
		key, value, err := krdr.rdr.GetRandomEntry() // get the doc
		if err != nil {                              // check for errors
			error_msg := err.Error()
			return _EMPTY_KEY, nil, &error_msg
		}

		// MB-42205 this may need improvement: a nil value only means that we run on of documents
		// on the last node we queried. There may be corner cases with many nodes and few documents
		// where we get a KEY_NOENT from one node, but there are more documents in other nodes
		if value == nil {
			break
		}
		if krdr.docIdsSeen[key] { // seen it before?
			duplicatesSeen++
			continue
		}

		krdr.docIdsSeen[key] = true
		return key, value, nil // new doc, return
	}

	// if we get here, we saw duplicate docs 100 times in a row, so we give
	// up on finding any more new docs
	return _EMPTY_KEY, nil, nil
}

func MakeKeyspaceRandomDocumentRetriever(ks datastore.Keyspace, sampleSize int) (*KeyspaceRandomDocumentRetriever, *string) {

	var ok bool
	krdr := new(KeyspaceRandomDocumentRetriever)
	krdr.rdr, ok = ks.(datastore.RandomEntryProvider)
	if !ok {
		error_msg := fmt.Sprintf("Keyspace does not implement RandomEntryProvider interface.")
		return nil, &error_msg
	}
	krdr.ks = ks
	krdr.docIdsSeen = make(map[string]bool)
	krdr.sampleSize = sampleSize

	//fmt.Printf("Made KeyspaceRandomDocumentRetriever\n")

	return krdr, nil
}

////////////////////////////////////////////////////////////////////////////////
// KVRandomDocumentRetriever implementation
//
// Given a server name, login & password, and bucket name and password,
// use the go-couchbase bucket GetRandomDoc() method to retrieve
// non-duplicate radom docs until we have sampleSize (or give up becasue we
// keep seeing duplicates).
////////////////////////////////////////////////////////////////////////////////

type KVRandomDocumentRetriever struct {
	docIdsSeen map[string]bool
	sampleSize int
	bucket     *couchbase.Bucket
}

func (kvrdr *KVRandomDocumentRetriever) GetNextDoc() (string, value.Value, *string) {
	// have we returned as many documents as were requested?
	if len(kvrdr.docIdsSeen) >= kvrdr.sampleSize {
		return _EMPTY_KEY, nil, nil
	}

	// try to retrieve the next document
	duplicatesSeen := 0
	for duplicatesSeen < 100 {
		resp, err := kvrdr.bucket.GetRandomDoc()

		if err != nil {
			error_msg := fmt.Sprintf("Error getting random doc, %v", err)
			return _EMPTY_KEY, nil, &error_msg
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

	// if we get here, we saw duplicate docs 100 times in a row, so we give
	// up on finding any more new docs
	return _EMPTY_KEY, nil, nil
}

func MakeKVRandomDocumentRetriever(serverURL, login, serverPass, bucket, bucketPass string, sampleSize int) (*KVRandomDocumentRetriever, *string) {

	kvrdr := new(KVRandomDocumentRetriever)
	kvrdr.docIdsSeen = make(map[string]bool)
	kvrdr.sampleSize = sampleSize

	var client couchbase.Client
	var err error

	// need to connect to the server...
	if len(login) == 0 {
		client, err = couchbase.Connect(serverURL)
	} else {
		client, err = couchbase.ConnectWithAuthCreds(serverURL, login, serverPass)
	}
	if err != nil { // check for errors
		error_msg := err.Error()
		return nil, &error_msg
	}

	pool, err := client.GetPool("default")
	if err != nil { // check for errors
		error_msg := err.Error()
		return nil, &error_msg
	}

	if len(bucketPass) > 0 {
		kvrdr.bucket, err = pool.GetBucketWithAuth(bucket, bucket, bucketPass)
	} else {
		kvrdr.bucket, err = pool.GetBucket(bucket)
	}
	if err != nil { // check for errors
		error_msg := fmt.Sprintf("Error getting bucket: %s - %s", bucket, err.Error())
		return nil, &error_msg
	}

	return kvrdr, nil
}
