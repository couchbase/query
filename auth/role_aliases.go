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

var _SHORT_TO_LONG = map[string]string{
	"select": "query_select",
	"insert": "query_insert",
	"update": "query_update",
	"delete": "query_delete",
}

var _LONG_TO_SHORT = map[string]string{
	"query_select": "select",
	"query_insert": "insert",
	"query_update": "update",
	"query_delete": "delete",
}

func NormalizeRoleNames(names []string) []string {
	ret := make([]string, len(names))
	for i, v := range names {
		lc := strings.ToLower(v)
		role, found := _SHORT_TO_LONG[lc]
		if !found {
			role = lc
		}
		ret[i] = role
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

func AliasToRole(alias string) string {
	lc := strings.ToLower(alias)
	role, found := _SHORT_TO_LONG[lc]
	if !found {
		return lc
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
