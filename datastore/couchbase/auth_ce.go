//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build !enterprise

package couchbase

import (
	"fmt"

	"github.com/couchbase/query/auth"
)

func messageForDeniedPrivilege(pair auth.PrivilegePair) string {
	keyspace := pair.Target

	privilege := ""
	role := ""
	switch pair.Priv {
	case auth.PRIV_READ:
		privilege = "run data read queries"
		role = fmt.Sprintf("bucket_full_access on %s", keyspace)
	case auth.PRIV_WRITE:
		privilege = "run data write queries"
		role = fmt.Sprintf("bucket_full_access on %s", keyspace)
	case auth.PRIV_UPSERT:
		privilege = "run data upsert queries"
		role = fmt.Sprintf("bucket_full_access on %s", keyspace)
	case auth.PRIV_SYSTEM_READ:
		privilege = "run queries accessing the system tables"
		role = "admin"
	case auth.PRIV_SECURITY_WRITE:
		privilege = "run queries updating user information"
		role = "admin"
	case auth.PRIV_SECURITY_READ:
		privilege = "run queries accessing user information"
		role = "admin"
	case auth.PRIV_QUERY_SELECT:
		privilege = fmt.Sprintf("run SELECT queries on %s", keyspace)
		role = fmt.Sprintf("bucket_full_access on %s", keyspace)
	case auth.PRIV_QUERY_UPDATE:
		privilege = fmt.Sprintf("run UPDATE queries on %s", keyspace)
		role = fmt.Sprintf("bucket_full_access on %s", keyspace)
	case auth.PRIV_QUERY_INSERT:
		privilege = fmt.Sprintf("run INSERT queries on %s", keyspace)
		role = fmt.Sprintf("bucket_full_access on %s", keyspace)
	case auth.PRIV_QUERY_DELETE:
		privilege = fmt.Sprintf("run DELETE queries on %s", keyspace)
		role = fmt.Sprintf("bucket_full_access on %s", keyspace)
	case auth.PRIV_QUERY_BUILD_INDEX, auth.PRIV_QUERY_CREATE_INDEX,
		auth.PRIV_QUERY_ALTER_INDEX, auth.PRIV_QUERY_DROP_INDEX, auth.PRIV_QUERY_LIST_INDEX:
		privilege = "run index operations"
		role = fmt.Sprintf("bucket_full_access on %s", keyspace)
	case auth.PRIV_QUERY_EXTERNAL_ACCESS:
		privilege = "run queries using the CURL() function"
		role = "admin"
	case auth.PRIV_BACKUP_CLUSTER:
		privilege = "backup cluster metadata"
		role = "backup_admin"
	case auth.PRIV_BACKUP_BUCKET:
		privilege = "backup bucket metadata"
		role = fmt.Sprintf("data_backup on %s", keyspace)
	case auth.PRIV_QUERY_MANAGE_FUNCTIONS:
		privilege = "manage global functions"
		role = "query_manage_global_functions"
	case auth.PRIV_QUERY_EXECUTE_FUNCTIONS:
		privilege = "execute global functions"
		role = "query_execute_global_functions"
	case auth.PRIV_QUERY_MANAGE_SCOPE_FUNCTIONS:
		privilege = "manage scope functions"
		role = fmt.Sprintf("query_manage_functions on %s", keyspace)
	case auth.PRIV_QUERY_EXECUTE_SCOPE_FUNCTIONS:
		privilege = "execute scope functions"
		role = fmt.Sprintf("query_execute_functions on %s", keyspace)
	case auth.PRIV_QUERY_MANAGE_FUNCTIONS_EXTERNAL:
		privilege = "manage global external functions"
		role = "query_manage_global_external_functions"
	case auth.PRIV_QUERY_EXECUTE_FUNCTIONS_EXTERNAL:
		privilege = "execute global external functions"
		role = "query_execute_global_external_functions"
	case auth.PRIV_QUERY_MANAGE_SCOPE_FUNCTIONS_EXTERNAL:
		privilege = "manage scope external functions"
		role = fmt.Sprintf("query_manage_external_functions on %s", keyspace)
	case auth.PRIV_QUERY_EXECUTE_SCOPE_FUNCTIONS_EXTERNAL:
		privilege = "execute scope external functions"
		role = fmt.Sprintf("query_execute_external_functions on %s", keyspace)
	case auth.PRIV_QUERY_MANAGE_SEQUENCES:
		privilege = "manage sequences"
		role = fmt.Sprintf("query_manage_sequences on %s", keyspace)
	case auth.PRIV_QUERY_USE_SEQUENCES:
		privilege = "use sequences"
		role = fmt.Sprintf("query_use_sequences on %s", keyspace)
	default:
		privilege = "run this type of query"
		role = "admin"
	}

	return fmt.Sprintf("User does not have credentials to %s. Add role %s to allow the statement to run.", privilege, role)
}
