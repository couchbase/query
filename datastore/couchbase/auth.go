//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package couchbase

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
)

func doAuthByCreds(creds cbauth.Creds, bucket string, requested auth.Privilege) (bool, error) {
	var permission string
	switch requested {
	case auth.PRIV_WRITE:
		permission = fmt.Sprintf("cluster.bucket[%s].data.docs!write", bucket)
	case auth.PRIV_READ:
		permission = fmt.Sprintf("cluster.bucket[%s].data.docs!read", bucket)
	case auth.PRIV_SYSTEM_READ:
		permission = "cluster.n1ql.meta!read"
	case auth.PRIV_SECURITY_READ:
		permission = "cluster.security!read"
	case auth.PRIV_SECURITY_WRITE:
		permission = "cluster.security!write"
	case auth.PRIV_QUERY_SELECT:
		permission = fmt.Sprintf("cluster.bucket[%s].n1ql.select!execute", bucket)
	case auth.PRIV_QUERY_UPDATE:
		permission = fmt.Sprintf("cluster.bucket[%s].n1ql.update!execute", bucket)
	case auth.PRIV_QUERY_INSERT:
		permission = fmt.Sprintf("cluster.bucket[%s].n1ql.insert!execute", bucket)
	case auth.PRIV_QUERY_DELETE:
		permission = fmt.Sprintf("cluster.bucket[%s].n1ql.delete!execute", bucket)
	case auth.PRIV_QUERY_BUILD_INDEX:
		permission = fmt.Sprintf("cluster.bucket[%s].n1ql.index!build", bucket)
	case auth.PRIV_QUERY_CREATE_INDEX:
		permission = fmt.Sprintf("cluster.bucket[%s].n1ql.index!create", bucket)
	case auth.PRIV_QUERY_ALTER_INDEX:
		permission = fmt.Sprintf("cluster.bucket[%s].n1ql.index!alter", bucket)
	case auth.PRIV_QUERY_DROP_INDEX:
		permission = fmt.Sprintf("cluster.bucket[%s].n1ql.index!drop", bucket)
	case auth.PRIV_QUERY_LIST_INDEX:
		permission = fmt.Sprintf("cluster.bucket[%s].n1ql.index!list", bucket)
	case auth.PRIV_QUERY_EXTERNAL_ACCESS:
		permission = "cluster.admin.security.external_access!execute"
	default:
		return false, fmt.Errorf("Invalid Privileges")
	}

	authResult, err := creds.IsAllowed(permission)
	if err != nil || authResult == false {
		return false, err
	}

	return true, nil

}

type authSource interface {
	adminIsOpen() bool
	auth(user, pwd string) (cbauth.Creds, error)
	isAuthTokenPresent(req *http.Request) bool
	authWebCreds(req *http.Request) (cbauth.Creds, error)
}

// Try to get privsSought privileges from the availableCredentials credentials.
// Return the privileges that were not granted.
func authAgainstCreds(as authSource, privsSought []auth.PrivilegePair, availableCredentials []cbauth.Creds) ([]auth.PrivilegePair, error) {
	deniedPrivs := make([]auth.PrivilegePair, 0, len(privsSought))
	for _, pair := range privsSought {
		keyspace := pair.Target
		privilege := pair.Priv
		if strings.Contains(keyspace, ":") {
			q := strings.Split(keyspace, ":")
			keyspace = q[1]
		}

		thisPrivGranted := false

		if keyspace == "nodes" && privilege == auth.PRIV_SYSTEM_READ && as.adminIsOpen() {
			// The system:nodes table follows the underlying ns_server API.
			// If all tables have passwords, the API requires credentials.
			// But if any don't, the API is open to read.
			continue
		}

		// Check requested privilege against the list of credentials.
		for _, creds := range availableCredentials {
			authResult, err := doAuthByCreds(creds, keyspace, privilege)

			if err != nil {
				return nil, err
			}

			// Auth succeeded
			if authResult == true {
				thisPrivGranted = true
				break
			}
		}

		// This privilege can not be granted by these credentials.
		if !thisPrivGranted {
			deniedPrivs = append(deniedPrivs, pair)
		}
	}
	return deniedPrivs, nil
}

// Determine the set of keyspaces referenced in the list of privileges, and derive
// credentials for them with empty passwords. This corresponds to the case of access users that were
// created for passwordless buckets at upgrate time.
func deriveDefaultCredentials(as authSource, privs []auth.PrivilegePair) ([]cbauth.Creds, auth.AuthenticatedUsers) {
	keyspaces := make(map[string]bool, len(privs))
	for _, pair := range privs {
		keyspace := pair.Target
		if keyspace == "" {
			continue
		}
		if strings.Contains(keyspace, ":") {
			q := strings.Split(keyspace, ":")
			keyspace = q[1]
		}
		keyspaces[keyspace] = true
	}

	creds := make([]cbauth.Creds, 0, len(keyspaces))
	authUsers := make(auth.AuthenticatedUsers, 0, len(keyspaces))
	password := ""
	for username := range keyspaces {
		user, err := as.auth(username, password)
		if err == nil {
			creds = append(creds, user)
			authUsers = append(authUsers, username)
		}
	}
	return creds, authUsers
}

func cbAuthorize(s authSource, privileges *auth.Privileges, credentials auth.Credentials,
	req *http.Request) (auth.AuthenticatedUsers, errors.Error) {
	if credentials == nil {
		credentials = make(auth.Credentials)
	}

	authenticatedUsers := make(auth.AuthenticatedUsers, 0, len(credentials))

	// Build the credentials list.
	credentialsList := make([]cbauth.Creds, 0, 2)
	for username, password := range credentials {
		var un string
		userCreds := strings.Split(username, ":")
		if len(userCreds) == 1 {
			un = userCreds[0]
		} else {
			un = userCreds[1]
		}

		logging.Debugf(" Credentials for user %v", un)
		creds, err := s.auth(un, password)
		if err != nil {
			logging.Debugf("Unable to authorize %s:%s.", username, password)
		} else {
			credentialsList = append(credentialsList, creds)
			if un != "" {
				authenticatedUsers = append(authenticatedUsers, un)
			}
		}
	}

	// Check for credentials from auth token in request
	if req != nil && s.isAuthTokenPresent(req) {
		creds, err := s.authWebCreds(req)
		if err != nil {
			logging.Debugf("Token auth error: %v", err)
		} else {
			credentialsList = append(credentialsList, creds)
		}
	}

	// No privileges to check? Done.
	if privileges == nil || len(privileges.List) == 0 {
		return authenticatedUsers, nil
	}

	// Check every requested privilege against the credentials list.
	// if the authentication fails for any of the requested privileges return an error
	remainingPrivileges, err := authAgainstCreds(s, privileges.List, credentialsList)

	if err != nil {
		return nil, errors.NewDatastoreAuthorizationError(err, "")
	}

	if len(remainingPrivileges) == 0 {
		// Everything is authorized. Success!
		return authenticatedUsers, nil
	}

	// Derive possible default credentials from remaining privileges.
	defaultCredentials, defaultUsers := deriveDefaultCredentials(s, remainingPrivileges)
	authenticatedUsers = append(authenticatedUsers, defaultUsers...)
	deniedPrivileges, err := authAgainstCreds(s, remainingPrivileges, defaultCredentials)

	if err != nil {
		return nil, errors.NewDatastoreAuthorizationError(err, "")
	}

	if len(deniedPrivileges) == 0 {
		// Authorized using defaults.
		return authenticatedUsers, nil
	}

	deniedKeyspace := deniedPrivileges[0].Target
	msg := ""
	if deniedKeyspace != "" {
		msg = fmt.Sprintf(" Keyspace %s.", deniedKeyspace)
	}
	return nil, errors.NewDatastoreAuthorizationError(nil, msg)
}
