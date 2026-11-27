//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package datastore

import (
	"fmt"
	"strings"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
)

func CredsArray(creds *auth.Credentials) []string {
	if creds != nil {
		if len(creds.CbauthCredentialsList) == 0 {
			GetDatastore().Authorize(nil, creds)
		}
		return []string(creds.AuthenticatedUsers)
	}
	return []string{}
}

func CredsString(creds *auth.Credentials) string {
	return strings.Join(CredsArray(creds), ",")
}

func FirstCred(creds *auth.Credentials) (string, string) {
	ds := GetDatastore()
	if ds != nil && creds != nil {
		return ds.CredsString(creds)
	}
	return "", ""
}

func EncodeName(user, domain string) string {
	if domain == "" {
		return "local:" + user
	}
	return domain + ":" + user
}

func DecodeName(encodedName string) (string, string) {
	fields := strings.Split(encodedName, ":")
	switch len(fields) {
	case 1:
		return fields[0], "local"
	case 2:
		return fields[1], fields[0]
	}
	return "", ""
}

// Checks if the user is an Administrator but does NOT audit the authorization check.
func IsAdmin(creds *auth.Credentials) bool {
	ds := GetDatastore()
	if ds != nil && creds != nil {

		privs := auth.NewPrivileges()
		privs.Add("", auth.PRIV_ADMIN, 0)

		// avoid logging an audit for an internal authorization action
		return ds.AuthorizeInternal(privs, creds) == nil
	}
	return false
}

// NonAdminUsers returns nil for an Administrator (callers that treat a nil user list as an
// access-check bypass rely on this), otherwise a non-nil (possibly empty) slice of
// authenticated users. CredsArray itself may return a nil slice for non-admin credentials
// with no authenticated users, which callers must not conflate with the admin-bypass case -
// this normalizes that so nil unambiguously means "admin".
func NonAdminUsers(creds *auth.Credentials) []string {
	if IsAdmin(creds) {
		return nil
	}
	users := CredsArray(creds)
	if users == nil {
		users = []string{}
	}
	return users
}

// SystemReadUser returns "" if any of the credentials has PRIV_SYSTEM_READ, meaning the caller is
// not restricted to their own system keyspace entries, otherwise the first authenticated user
// (domain:name), built as the users that entries such as active requests are owned by are.
// Like NonAdminUsers, but with PRIV_SYSTEM_READ rather than PRIV_ADMIN granting unrestricted access.
// If the privilege can't be checked, it returns the error together with the restricted user, so that
// callers that go on regardless don't grant unrestricted access.
func SystemReadUser(creds *auth.Credentials) (string, errors.Error) {
	ds := GetDatastore()
	if ds != nil && creds != nil {
		privs := auth.NewPrivileges()
		privs.Add("", auth.PRIV_SYSTEM_READ, auth.PRIV_PROPS_NONE)

		// avoid logging an audit for an internal authorization action
		err := ds.AuthorizeInternal(privs, creds)
		if err == nil {
			return "", nil
		}
		if err.Code() != errors.E_DATASTORE_INSUFFICIENT_CREDENTIALS {
			return restrictedUser(creds), err
		}
	}
	return restrictedUser(creds), nil
}

// a user with no authenticated users owns no requests: return a name that matches none
func restrictedUser(creds *auth.Credentials) string {
	if users := CredsArray(creds); len(users) > 0 {
		return users[0]
	}
	return EncodeName("", "")
}

func AdminCreds(node string) (*auth.Credentials, error) {
	if _DATASTORE == nil {
		return nil, fmt.Errorf("datastore not initialized")
	}
	u, p, err := _DATASTORE.AdminUser(node)
	if err != nil {
		return nil, err
	}
	return auth.NewCredentials(u, p), nil
}

func GetUserUUID(creds *auth.Credentials) string {
	if _DATASTORE == nil || creds == nil {
		return ""
	}
	return _DATASTORE.GetUserUUID(creds)
}

func GetUserBuckets(creds *auth.Credentials) []string {
	if _DATASTORE == nil || creds == nil {
		return []string{}
	}
	return _DATASTORE.GetUserBuckets(creds)
}

func GetImpersonateBuckets(user, domain string) []string {
	if _DATASTORE == nil || len(user) == 0 {
		return []string{}
	}
	return _DATASTORE.GetImpersonateBuckets(user, domain)
}
