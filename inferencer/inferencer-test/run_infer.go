/*
Copyright 2019-Present Couchbase, Inc.

Use of this software is governed by the Business Source License included in
the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
file, in accordance with the Business Source License, use of this software will
be governed by the Apache License, Version 2.0, included in the file
licenses/APL2.txt.
*/

package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/couchbase/query/inferencer"
)

var CLUSTER = flag.String("cluster", "http://Administrator:password@127.0.0.1:9091",
	"Cluster address (e.g., http://localhost:8091)")
var BUCKET = flag.String("bucket", "beer-sample", "Bucket to test (e.g. beer-sample)")

func main() {
	// connect to couchbase and do schema inferencing on travel-sample
	fmt.Printf("Connecting to %s\n", *CLUSTER)

	kvRetriever, err := inferencer.MakeKVRandomDocumentRetriever(*CLUSTER, *BUCKET, "", 1000)

	if err != nil {
		fmt.Printf("Error making retriever: %v\n", err)
		return
	}

	//    val, errr := kvRetriever.GetNextDoc()
	//    if errr != nil {
	//        fmt.Printf("Error getting doc: %s\n",*errr)
	//        return
	//    } else {
	//        fmt.Printf("Got doc: %v.\n",val.String())
	//    }

	start := time.Now() // remember when we started

	options := &inferencer.DescribeOptions{
		SimilarityMetric:    0.6,
		NumSampleValues:     5,
		DictionaryThreshold: 10,
		InferTimeout:        60,
		MaxSchemaMB:         10,
	}

	result, err := inferencer.DescribeKeyspace(nil, nil, kvRetriever, options)

	if err != nil {
		fmt.Printf("Error result: %v err: %v\n", result, err)
		return
	}

	fmt.Printf("Finished INFER of %s after time: %dms\n", *BUCKET, int32(time.Now().Sub(start)/time.Millisecond))

}
