//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

// +build !enterprise

package couchbase

import (
	"fmt"

	"github.com/couchbase/query/auth"
)

func messageForDeniedPrivilege(pair auth.PrivilegePair) string {
	_, keyspace := namespaceKeyspaceFromPrivPair(pair)

	privilege := ""
	role := ""
	switch pair.Priv {
	case auth.PRIV_READ:
		privilege = "data read queries"
		role = fmt.Sprintf("bucket_full_access on bucket %s", keyspace)
	case auth.PRIV_WRITE:
		privilege = "data write queries"
		role = fmt.Sprintf("bucket_full_access on bucket %s", keyspace)
	case auth.PRIV_SYSTEM_READ:
		privilege = "queries accessing the system tables"
		role = "admin"
	case auth.PRIV_SECURITY_WRITE:
		privilege = "queries updating user information"
		role = "admin"
	case auth.PRIV_SECURITY_READ:
		privilege = "queries accessing user information"
		role = "admin"
	case auth.PRIV_QUERY_SELECT:
		privilege = fmt.Sprintf("SELECT queries on the %s bucket", keyspace)
		role = fmt.Sprintf("bucket_full_access on %s", keyspace)
	case auth.PRIV_QUERY_UPDATE:
		privilege = fmt.Sprintf("UPDATE queries on the %s bucket", keyspace)
		role = fmt.Sprintf("bucket_full_access on %s", keyspace)
	case auth.PRIV_QUERY_INSERT:
		privilege = fmt.Sprintf("INSERT queries on the %s bucket", keyspace)
		role = fmt.Sprintf("bucket_full_access on %s", keyspace)
	case auth.PRIV_QUERY_DELETE:
		privilege = fmt.Sprintf("DELETE queries on the %s bucket", keyspace)
		role = fmt.Sprintf("bucket_full_access on %s", keyspace)
	case auth.PRIV_QUERY_BUILD_INDEX, auth.PRIV_QUERY_CREATE_INDEX,
		auth.PRIV_QUERY_ALTER_INDEX, auth.PRIV_QUERY_DROP_INDEX, auth.PRIV_QUERY_LIST_INDEX:
		privilege = "index operations"
		role = fmt.Sprintf("bucket_full_access on %s", keyspace)
	case auth.PRIV_QUERY_EXTERNAL_ACCESS:
		privilege = "queries using the CURL() function"
		role = "admin"
	default:
		privilege = "this type of query"
		role = "admin"
	}

	return fmt.Sprintf("User does not have credentials to run %s. Add role %s to allow the query to run.", privilege, role)
}
