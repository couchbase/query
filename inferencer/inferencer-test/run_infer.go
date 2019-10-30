package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/couchbase/query/inferencer"
)

var KVSTORE = flag.String("kvstore", "127.0.0.1:12210", "KV store address (e.g., localhost:11210)")
var CLUSTER = flag.String("cluster", "http://127.0.0.1:9091", "Cluster address (e.g., http://localhost:8091)")
var BUCKET = flag.String("bucket", "beer-sample", "Bucket to test (e.g. beer-sample)")
var USER = flag.String("user", "Administrator", "User for authentication")
var PASSWORD = flag.String("password", "bluehorse", "Password for user")

func main() {

	//    // try the basic memcached client
	//
	//    	// connect to the kv store
	//
	//	mclient, err := memcached.Connect("tcp", *KVSTORE)
	//	if err != nil {
	//		fmt.Printf("Error connecting: %v\n", err)
	//		return
	//	}
	//
	//	// authenticate
	//	_, err = mclient.Auth(*USER, *PASSWORD)
	//	if err != nil {
	//		fmt.Printf("auth error: %v\n", err)
	//		return
	//	}
	//
	//	// get the bucket
	//	_, err = mclient.SelectBucket(*BUCKET)
	//	if err != nil {
	//		fmt.Printf("error selecting bucket: %v\n", err)
	//		return
	//	}
	//
	//    resp, err := mclient.GetRandomDoc()
	//
	//    fmt.Printf("Got random doc from mc: %v\n",resp)
	//
	//	// now try gocouchbase
	//
	//    var client couchbase.Client
	//
	//    //client, err = couchbase.ConnectWithAuthCreds(*CLUSTER, *USER, *PASSWORD)
	//    client, err = couchbase.Connect("http://Administrator:bluehorse@127.0.0.1:9091")
	//
	//	if err != nil { // check for errors
	//		error_msg := err.Error()
	//		fmt.Printf("Error connecting: %s\n", error_msg)
	//		return
	//	}
	//
	//	pool, err := client.GetPool("default")
	//	if err != nil { // check for errors
	//		error_msg := err.Error()
	//		fmt.Printf("Error getting pool %v: %s\n", pool,error_msg)
	//		return
	//	}
	//
	//
	//    bucket, err := pool.GetBucket(*BUCKET)
	//	if err != nil { // check for errors
	//		error_msg := fmt.Sprintf("Error getting bucket: %s - %s", bucket, err.Error())
	//		fmt.Printf("%s\n", error_msg)
	//		return
	//	}
	//
	////    newClient, _, err := bucket.GetRandomConnection()
	////	if err != nil { // check for errors
	////		error_msg := fmt.Sprintf("Error connection to bucket: %s - %s", bucket, err.Error())
	////		fmt.Printf("%s\n", error_msg)
	////		return
	////	}
	//
	////    resp, err = newClient.GetRandomDoc()
	//	resp, err = bucket.GetRandomDoc()
	//	if err != nil {
	//		fmt.Printf("Error getting random doc resp: %v, err: %s\n", resp,err)
	//		return
	//	}
	//
	//    fmt.Printf("Got resp: %v\n",resp)
	//    return
	//
	// connect to couchbase and do schema inferencing on travel-sample
	fmt.Printf("Connecting to %s as user %s\n", *CLUSTER, *USER)

	kvRetriever, errr := inferencer.MakeKVRandomDocumentRetriever(*CLUSTER, *USER, *PASSWORD, *BUCKET, "", 1000)

	if errr != nil {
		fmt.Printf("Error making retriever: %s\n", *errr)
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

	result, errr, warn := inferencer.DescribeKeyspace(nil, kvRetriever, 0.6, 5, 10, 60, 10)

	if errr != nil {
		fmt.Printf("Error result: %v err: %v warn %v\n", result, errr, warn)
		return
	}

	fmt.Printf("Finished INFER of %s after time: %dms\n", *BUCKET, int32(time.Now().Sub(start)/time.Millisecond))

}
