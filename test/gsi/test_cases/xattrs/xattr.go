//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package xattrs

import (
	"testing"
	"time"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/test/gsi"
	gocb "gopkg.in/couchbase/gocb.v1"
)

/*
Method to pass in parameters for site, pool and
namespace to Start method for Couchbase Server.
*/
func start_cs() *gsi.MockServer {
	return gsi.Start_cs(true)
}

func runStmt(mockServer *gsi.MockServer, q string) ([]interface{}, []errors.Error, errors.Error) {
	return gsi.RunStmt(mockServer, q)
}

func runMatch(filename string, prepared, explain bool, qc *gsi.MockServer, t *testing.T) {
	gsi.RunMatch(filename, prepared, explain, qc, t)
}

func gocb_SetupXattr() {
	cluster, _ := gocb.Connect(gsi.Pool_CBS)

	cluster.Authenticate(gocb.PasswordAuthenticator{
		Username: gsi.Username,
		Password: gsi.Password,
	})

	bucket, _ := cluster.OpenBucket("product", "")

	var _sync = map[string]interface{}{"name": "Information about products", "id": 23}
	var userAttr = [2]string{"Product 1", "Product 10"}

	// Add key-value pairs to hotel_10138, representing traveller-Ids and associated discount percentages
	bucket.MutateIn("product0_xattrs", 0, 0).
		UpsertEx("_sync", _sync, gocb.SubdocFlagXattr|gocb.SubdocFlagCreatePath).Execute()

	_sync["id"] = 231

	bucket.MutateIn("product10_xattrs", 0, 0).
		UpsertEx("_sync", _sync, gocb.SubdocFlagXattr|gocb.SubdocFlagCreatePath).Execute()

	bucket.MutateIn("product1_xattrs", 0, 0).
		UpsertEx("userAttr", userAttr, gocb.SubdocFlagXattr|gocb.SubdocFlagCreatePath).Execute()

	time.Sleep(time.Millisecond * 10)

}
