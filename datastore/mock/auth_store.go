//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package mock

import (
	"slices"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
)

// AuthStore wraps a datastore for tests of per-user access: it authenticates any local user whose password
// is "pw", from the credentials or from HTTP basic authentication, and grants every privilege, PRIV_SYSTEM_READ
// included, to "admin" and to internal ("@...") users alone
type AuthStore struct {
	datastore.Datastore
}

func (s *AuthStore) Authorize(privs *auth.Privileges, creds *auth.Credentials) errors.Error {
	if creds.AuthenticatedUsers == nil {
		users := auth.AuthenticatedUsers{}
		if creds.HttpRequest != nil {
			if u, p, ok := creds.HttpRequest.BasicAuth(); ok && p == "pw" {
				users = append(users, datastore.EncodeName(u, "local"))
			}
		}
		up := creds.UsersAndPasswords()
		for i := 0; i+1 < len(up); i += 2 {
			if up[i+1] == "pw" {
				users = append(users, datastore.EncodeName(up[i], "local"))
			}
		}
		creds.AuthenticatedUsers = users
	}
	if privs == nil || len(privs.List) == 0 || slices.ContainsFunc(creds.AuthenticatedUsers, func(u string) bool {
		return u == "local:admin" || auth.IsUsernameInternal(u)
	}) {
		return nil
	}
	return errors.NewDatastoreInsufficientCredentials("denied", nil, "", "", "")
}

func (s *AuthStore) AuthorizeInternal(privs *auth.Privileges, creds *auth.Credentials) errors.Error {
	return s.Authorize(privs, creds)
}

func (s *AuthStore) CredsString(creds *auth.Credentials) (string, string) {
	if len(creds.AuthenticatedUsers) == 0 {
		return "", ""
	}
	return datastore.DecodeName(creds.AuthenticatedUsers[0])
}
