//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package auth

import "strings"

// On other database systems, permissions to run queries follow the names of the princial
// statement names, such as SELECT, INSERT, and UPDATE. In Couchbase, these names are a
// little different; our equivalent roles are named query_select, query_insert, and query_update.
// But in the N1QL context, we want to present the familiar names, as aliases.
// Accordingly, we allow users to use the short forms of the names in GRANT and REVOKE
// statements, and we show the roles using the short forms in the system:user_info,
// system:my_user_info, and system:applicable_roles tables.
//
// This package contains the code for supporting this functionality.

// Source type constants
const (
	SOURCE_KEYSPACE        = ""
	SOURCE_CATALOG         = "catalog"
	SOURCE_CREDENTIALSTORE = "credentialstore"
)

var _SHORT_TO_LONG = map[string]string{
	"select":  "query_select",
	"insert":  "query_insert",
	"update":  "query_update",
	"delete":  "query_delete",
	"consume": "credential_consumer",
}

var _LONG_TO_SHORT = map[string]string{
	"query_select":                  "select",
	"query_insert":                  "insert",
	"query_update":                  "update",
	"query_delete":                  "delete",
	"query_select_external_catalog": "select",
	"query_insert_external_catalog": "insert",
	"query_update_external_catalog": "update",
	"query_delete_external_catalog": "delete",
	"external_catalog_admin":        "external_catalog_admin",
	"external_catalog_reader":       "external_catalog_reader",
	"credential_consumer":           "consume",
}

func NormalizeRoleNames(names []string, sourceType string) []string {
	ret := make([]string, len(names))
	for i, v := range names {
		ret[i] = AliasToRole(v, sourceType)
	}
	return ret
}

func RoleToAlias(role string) string {
	alias, found := _LONG_TO_SHORT[role]
	if !found {
		return role
	}
	return alias
}

// RoleToAliasSource splits the role name and returns the alias and source type
// Returns: (alias, sourceType)
func RoleToAliasSource(role string) (string, string) {
	alias, found := _LONG_TO_SHORT[role]
	if !found {
		return role, SOURCE_KEYSPACE
	}

	// Split the role name to get source type
	parts := strings.SplitN(role, "_", 4)
	switch len(parts) {
	case 2:
		if parts[0] == "credential" {
			return alias, SOURCE_CREDENTIALSTORE
		}
	case 3:
		if parts[1] == "catalog" {
			return alias, SOURCE_CATALOG
		}
	case 4:
		return alias, SOURCE_CATALOG
	}
	return alias, SOURCE_KEYSPACE
}

func AliasToRole(alias, sourceType string) string {
	lc := strings.ToLower(alias)
	role, found := _SHORT_TO_LONG[lc]
	if !found {
		return lc
	} else if sourceType == SOURCE_CATALOG {
		return role + "_external_" + sourceType
	}
	return role
}

// Expecting parsed JSON, in a format like this:
// { "domain":"local",
//
//	"id":"reviewowner",
//	"name":"OwnerOfreview",
//	"roles":[{"bucket_name":"customer","role":"query_select"},
//	         {"bucket_name":"customer","role":"query_insert"},
//	         {"bucket_name":"review","role":"bucket_full_access"}]}
//
// Change the roles to their alias forms, like this:
// { "domain":"local",
//
//	"id":"reviewowner",
//	"name":"OwnerOfreview",
//	"roles":[{"bucket_name":"customer","role":"select"},
//	         {"bucket_name":"customer","role":"insert"},
//	         {"bucket_name":"review","role":"bucket_full_access"}]}
//
// If the data is in an unexpected format, we leave it as we found it.
func ConvertRolesToAliases(user map[string]interface{}) {
	roles, present := user["roles"]
	if !present || roles == nil {
		return
	}
	rolesArr, ok := roles.([]interface{})
	if !ok {
		return
	}
	for _, role := range rolesArr {
		roleMap, ok := role.(map[string]interface{})
		if !ok {
			continue
		}
		roleVal, present := roleMap["role"]
		if !present || roleVal == nil {
			continue
		}
		roleValStr, ok := roleVal.(string)
		if !ok {
			continue
		}
		shortForm, found := _LONG_TO_SHORT[roleValStr]
		if found {
			roleMap["role"] = shortForm
		}
	}
}
