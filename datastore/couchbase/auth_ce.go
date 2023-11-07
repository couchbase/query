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

func messageForDeniedPrivilege(pair auth.PrivilegePair) (string, string) {
	keyspace := pair.Target

	privilege := ""
	role := ""
	base_role := ""
	switch pair.Priv {
	case auth.PRIV_READ:
		privilege = "run data read queries"
		base_role = "bucket_full_access"
		role = fmt.Sprintf("%s on %s", base_role, keyspace)
	case auth.PRIV_WRITE:
		privilege = "run data write queries"
		base_role = "bucket_full_access"
		role = fmt.Sprintf("%s on %s", base_role, keyspace)
	case auth.PRIV_UPSERT:
		privilege = "run data upsert queries"
		base_role = "bucket_full_access"
		role = fmt.Sprintf("%s on %s", base_role, keyspace)
	case auth.PRIV_SYSTEM_READ:
		privilege = "run queries accessing the system tables"
		base_role = "admin"
	case auth.PRIV_SECURITY_WRITE:
		privilege = "run queries updating user information"
		base_role = "admin"
	case auth.PRIV_SECURITY_READ:
		privilege = "run queries accessing user information"
		base_role = "admin"
	case auth.PRIV_QUERY_SELECT:
		privilege = fmt.Sprintf("run SELECT queries on %s", keyspace)
		base_role = "bucket_full_access"
		role = fmt.Sprintf("%s on %s", base_role, keyspace)
	case auth.PRIV_QUERY_UPDATE:
		privilege = fmt.Sprintf("run UPDATE queries on %s", keyspace)
		base_role = "bucket_full_access"
		role = fmt.Sprintf("%s on %s", base_role, keyspace)
	case auth.PRIV_QUERY_INSERT:
		privilege = fmt.Sprintf("run INSERT queries on %s", keyspace)
		base_role = "bucket_full_access"
		role = fmt.Sprintf("%s on %s", base_role, keyspace)
	case auth.PRIV_QUERY_DELETE:
		privilege = fmt.Sprintf("run DELETE queries on %s", keyspace)
		base_role = "bucket_full_access"
		role = fmt.Sprintf("%s on %s", base_role, keyspace)
	case auth.PRIV_QUERY_BUILD_INDEX, auth.PRIV_QUERY_CREATE_INDEX,
		auth.PRIV_QUERY_ALTER_INDEX, auth.PRIV_QUERY_DROP_INDEX, auth.PRIV_QUERY_LIST_INDEX:
		privilege = "run index operations"
		base_role = "bucket_full_access"
		role = fmt.Sprintf("%s on %s", base_role, keyspace)
	case auth.PRIV_QUERY_EXTERNAL_ACCESS:
		privilege = "run queries using the CURL() function"
		base_role = "admin"
	case auth.PRIV_BACKUP_CLUSTER:
		privilege = "backup cluster metadata"
		base_role = "backup_admin"
	case auth.PRIV_BACKUP_BUCKET:
		privilege = "backup bucket metadata"
		base_role = "data_backup"
		role = fmt.Sprintf("%s on %s", base_role, keyspace)
	case auth.PRIV_QUERY_MANAGE_FUNCTIONS:
		privilege = "manage global functions"
		base_role = "query_manage_global_functions"
	case auth.PRIV_QUERY_EXECUTE_FUNCTIONS:
		privilege = "execute global functions"
		base_role = "query_execute_global_functions"
	case auth.PRIV_QUERY_MANAGE_SCOPE_FUNCTIONS:
		privilege = "manage scope functions"
		base_role = "query_manage_functions"
		role = fmt.Sprintf("%s on %s", base_role, keyspace)
	case auth.PRIV_QUERY_EXECUTE_SCOPE_FUNCTIONS:
		privilege = "execute scope functions"
		base_role = "query_execute_functions"
		role = fmt.Sprintf("%s on %s", base_role, keyspace)
	case auth.PRIV_QUERY_MANAGE_FUNCTIONS_EXTERNAL:
		privilege = "manage global external functions"
		base_role = "query_manage_global_external_functions"
	case auth.PRIV_QUERY_EXECUTE_FUNCTIONS_EXTERNAL:
		privilege = "execute global external functions"
		base_role = "query_execute_global_external_functions"
	case auth.PRIV_QUERY_MANAGE_SCOPE_FUNCTIONS_EXTERNAL:
		privilege = "manage scope external functions"
		base_role = "query_manage_external_functions"
		role = fmt.Sprintf("%s on %s", base_role, keyspace)
	case auth.PRIV_QUERY_EXECUTE_SCOPE_FUNCTIONS_EXTERNAL:
		privilege = "execute scope external functions"
		base_role = "query_execute_external_functions"
		role = fmt.Sprintf("%s on %s", base_role, keyspace)
	case auth.PRIV_QUERY_MANAGE_SEQUENCES:
		privilege = "manage sequences"
		base_role = "query_manage_sequences"
		role = fmt.Sprintf("%s on %s", base_role, keyspace)
	case auth.PRIV_QUERY_USE_SEQUENCES:
		privilege = "use sequences"
		base_role = "query_use_sequences"
		role = fmt.Sprintf("%s on %s", base_role, keyspace)
	case auth.PRIV_SEARCH_CREATE_INDEX, auth.PRIV_SEARCH_DROP_INDEX:
		privilege = "manage fts indices"
		base_role = "fts_admin"
		role = fmt.Sprintf("%s on %s", base_role, keyspace)
	default:
		privilege = "run this type of query"
		base_role = "admin"
	}
	if role == "" && base_role != "" {
		role = base_role
	}

	return fmt.Sprintf("User does not have credentials to %s. Add role %s to allow the statement to run.", privilege, role),
		base_role
}
