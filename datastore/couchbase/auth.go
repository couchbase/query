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
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
)

func doAuthByCreds(creds cbauth.Creds, bucket string, requested datastore.Privilege) (bool, error) {
	var permission string
	switch requested {
	case datastore.PRIV_WRITE:
		permission = fmt.Sprintf("cluster.bucket[%s].data.docs!write", bucket)
	case datastore.PRIV_READ:
		permission = fmt.Sprintf("cluster.bucket[%s].data.docs!read", bucket)
	case datastore.PRIV_SYSTEM_READ:
		permission = "cluster.n1ql.meta!read"
	case datastore.PRIV_SECURITY_READ:
		permission = "cluster.security!read"
	case datastore.PRIV_SECURITY_WRITE:
		permission = "cluster.security!write"
	case datastore.PRIV_QUERY_SELECT:
		permission = fmt.Sprintf("cluster.bucket[%s].n1ql.select!execute", bucket)
	case datastore.PRIV_QUERY_UPDATE:
		permission = fmt.Sprintf("cluster.bucket[%s].n1ql.update!execute", bucket)
	case datastore.PRIV_QUERY_INSERT:
		permission = fmt.Sprintf("cluster.bucket[%s].n1ql.insert!execute", bucket)
	case datastore.PRIV_QUERY_DELETE:
		permission = fmt.Sprintf("cluster.bucket[%s].n1ql.delete!execute", bucket)
	case datastore.PRIV_QUERY_BUILD_INDEX:
		permission = fmt.Sprintf("cluster.bucket[%s].n1ql.index!build", bucket)
	case datastore.PRIV_QUERY_CREATE_INDEX:
		permission = fmt.Sprintf("cluster.bucket[%s].n1ql.index!create", bucket)
	case datastore.PRIV_QUERY_ALTER_INDEX:
		permission = fmt.Sprintf("cluster.bucket[%s].n1ql.index!alter", bucket)
	case datastore.PRIV_QUERY_DROP_INDEX:
		permission = fmt.Sprintf("cluster.bucket[%s].n1ql.index!drop", bucket)
	case datastore.PRIV_QUERY_LIST_INDEX:
		permission = fmt.Sprintf("cluster.bucket[%s].n1ql.index!list", bucket)
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

func cbAuthorize(s authSource, privileges *datastore.Privileges, credentials datastore.Credentials,
	req *http.Request) (datastore.AuthenticatedUsers, errors.Error) {
	if credentials == nil {
		credentials = make(datastore.Credentials)
	}

	authenticatedUsers := make(datastore.AuthenticatedUsers, 0, len(credentials))

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

	// No authenticated user, but credentials to check?
	if len(credentialsList) == 0 {
		if len(credentials) == 0 {
			return nil, errors.NewDatastoreNoUserSupplied()
		} else {
			return nil, errors.NewDatastoreInvalidUsernamePassword()
		}
	}

	// Check every requested privilege against the credentials list.
	// if the authentication fails for any of the requested privileges return an error
	for _, pair := range privileges.List {
		keyspace := pair.Target
		privilege := pair.Priv
		if strings.Contains(keyspace, ":") {
			q := strings.Split(keyspace, ":")
			keyspace = q[1]
		}

		logging.Debugf("Authenticating for keyspace %s", keyspace)

		thisBucketAuthorized := false
		var rememberedError error

		if keyspace == "nodes" && privilege == datastore.PRIV_SYSTEM_READ && s.adminIsOpen() {
			// The system:nodes table follows the underlying ns_server API.
			// If all tables have passwords, the API requires credentials.
			// But if any don't, the API is open to read.
			continue
		}

		// Check requested privilege against the list of credentials.
		for _, creds := range credentialsList {
			authResult, err := doAuthByCreds(creds, keyspace, privilege)

			// Auth succeeded
			if authResult == true {
				thisBucketAuthorized = true
				break
			} else if err != nil {
				rememberedError = err
			}
		}

		if !thisBucketAuthorized {
			msg := ""
			if keyspace != "" {
				msg = fmt.Sprintf(" Keyspace %s.", keyspace)
			}
			return nil, errors.NewDatastoreAuthorizationError(rememberedError, msg)
		}
	}

	// If we got this far, every bucket is authorized. Success!
	return authenticatedUsers, nil
}
