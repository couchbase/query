//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

//go:build enterprise

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
		privilege = "data read queries"
		role = fmt.Sprintf("data_reader on %s", keyspace)
	case auth.PRIV_WRITE:
		privilege = "data write queries"
		role = fmt.Sprintf("data_reader_writer on %s", keyspace)
	case auth.PRIV_UPSERT:
		privilege = "data upsert queries"
		role = fmt.Sprintf("data_reader_writer on %s", keyspace)
	case auth.PRIV_SYSTEM_READ:
		privilege = "queries accessing the system tables"
		role = "query_system_catalog"
	case auth.PRIV_SECURITY_WRITE:
		privilege = "queries updating user information"
		role = "admin"
	case auth.PRIV_SECURITY_READ:
		privilege = "queries accessing user information"
		role = "admin"
	case auth.PRIV_QUERY_SELECT:
		privilege = fmt.Sprintf("SELECT queries on %s", keyspace)
		role = fmt.Sprintf("query_select on %s", keyspace)
	case auth.PRIV_QUERY_UPDATE:
		privilege = fmt.Sprintf("UPDATE queries on %s", keyspace)
		role = fmt.Sprintf("query_update on %s", keyspace)
	case auth.PRIV_QUERY_INSERT:
		privilege = fmt.Sprintf("INSERT queries on %s", keyspace)
		role = fmt.Sprintf("query_insert on %s", keyspace)
	case auth.PRIV_QUERY_DELETE:
		privilege = fmt.Sprintf("DELETE queries on %s", keyspace)
		role = fmt.Sprintf("query_delete on %s", keyspace)
	case auth.PRIV_QUERY_BUILD_INDEX, auth.PRIV_QUERY_CREATE_INDEX,
		auth.PRIV_QUERY_ALTER_INDEX, auth.PRIV_QUERY_DROP_INDEX, auth.PRIV_QUERY_LIST_INDEX:
		privilege = "index operations"
		role = fmt.Sprintf("query_manage_index on %s", keyspace)
	case auth.PRIV_QUERY_EXTERNAL_ACCESS:
		privilege = "queries using the CURL() function"
		role = "query_external_access"
	case auth.PRIV_BACKUP_CLUSTER:
		privilege = "backup cluster metadata"
		role = "backup_admin"
	case auth.PRIV_BACKUP_BUCKET:
		privilege = "backup bucket metadata"
		role = fmt.Sprintf("data_backup on %s", keyspace)
	default:
		privilege = "this type of query"
		role = "admin"
	}

	return fmt.Sprintf("User does not have credentials to run %s. Add role %s to allow the query to run.", privilege, role)
}
