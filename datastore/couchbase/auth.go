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
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
)

func opIsUnimplemented(namespace, bucket string, requested auth.Privilege) bool {
	if namespace == "#system" {
		// For system monitoring tables INSERT and UPDATE are not supported.
		if bucket == "prepareds" || bucket == "completed_requests" || bucket == "active_requests" {
			if requested == auth.PRIV_QUERY_UPDATE || requested == auth.PRIV_QUERY_INSERT {
				return true
			}
			return false
		}
		// For other system buckets, INSERT/UPDATE/DELETE are not supported.
		if requested == auth.PRIV_QUERY_UPDATE || requested == auth.PRIV_QUERY_INSERT || requested == auth.PRIV_QUERY_DELETE {
			return true
		}
		return false
	}
	return false
}

func privilegeString(namespace, bucket string, requested auth.Privilege) (string, error) {
	var permission string
	switch requested {
	case auth.PRIV_WRITE:
		permission = joinStrings("cluster.bucket[", bucket, "].data.docs!write")
	case auth.PRIV_READ:
		permission = joinStrings("cluster.bucket[", bucket, "].data.docs!read")
	case auth.PRIV_SYSTEM_READ:
		permission = "cluster.n1ql.meta!read"
	case auth.PRIV_SECURITY_READ:
		permission = "cluster.admin.security!read"
	case auth.PRIV_SECURITY_WRITE:
		permission = "cluster.admin.security!write"
	case auth.PRIV_QUERY_SELECT:
		permission = joinStrings("cluster.bucket[", bucket, "].n1ql.select!execute")
	case auth.PRIV_QUERY_UPDATE:
		permission = joinStrings("cluster.bucket[", bucket, "].n1ql.update!execute")
	case auth.PRIV_QUERY_INSERT:
		permission = joinStrings("cluster.bucket[", bucket, "].n1ql.insert!execute")
	case auth.PRIV_QUERY_DELETE:
		permission = joinStrings("cluster.bucket[", bucket, "].n1ql.delete!execute")
	case auth.PRIV_QUERY_BUILD_INDEX:
		permission = joinStrings("cluster.bucket[", bucket, "].n1ql.index!build")
	case auth.PRIV_QUERY_CREATE_INDEX:
		permission = joinStrings("cluster.bucket[", bucket, "].n1ql.index!create")
	case auth.PRIV_QUERY_ALTER_INDEX:
		permission = joinStrings("cluster.bucket[", bucket, "].n1ql.index!alter")
	case auth.PRIV_QUERY_DROP_INDEX:
		permission = joinStrings("cluster.bucket[", bucket, "].n1ql.index!drop")
	case auth.PRIV_QUERY_LIST_INDEX:
		permission = joinStrings("cluster.bucket[", bucket, "].n1ql.index!list")
	case auth.PRIV_QUERY_EXTERNAL_ACCESS:
		permission = "cluster.n1ql.curl!execute"
	default:
		return "", fmt.Errorf("Invalid Privileges")
	}
	return permission, nil
}

func doAuthByCreds(creds cbauth.Creds, namespace string, bucket string, requested auth.Privilege) (bool, error) {
	permission, err := privilegeString(namespace, bucket, requested)
	if err != nil {
		return false, err
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
	authWebCreds(req *http.Request) (cbauth.Creds, error)
}

func namespaceKeyspaceFromPrivPair(pair auth.PrivilegePair) (namespace, keyspace string) {
	if strings.Contains(pair.Target, ":") {
		q := strings.Split(pair.Target, ":")
		namespace = q[0]
		keyspace = q[1]
	} else {
		namespace = "default"
		keyspace = pair.Target
	}
	return
}

// Try to get privsSought privileges from the availableCredentials credentials.
// Return the privileges that were not granted.
func authAgainstCreds(as authSource, privsSought []auth.PrivilegePair, availableCredentials []cbauth.Creds) ([]auth.PrivilegePair, error) {
	deniedPrivs := make([]auth.PrivilegePair, 0, len(privsSought))
	for _, pair := range privsSought {
		namespace, keyspace := namespaceKeyspaceFromPrivPair(pair)
		privilege := pair.Priv

		thisPrivGranted := false

		if keyspace == "nodes" && privilege == auth.PRIV_SYSTEM_READ && as.adminIsOpen() {
			// The system:nodes table follows the underlying ns_server API.
			// If all tables have passwords, the API requires credentials.
			// But if any don't, the API is open to read.
			continue
		}

		if opIsUnimplemented(namespace, keyspace, privilege) {
			// Trivially grant permission for unimplemented operations.
			// Error reporting will be handled by the execution layer.
			continue
		}

		// Check requested privilege against the list of credentials.
		for _, creds := range availableCredentials {
			authResult, err := doAuthByCreds(creds, namespace, keyspace, privilege)

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

func userKeyString(c cbauth.Creds) string {
	return joinStrings(c.Domain(), ":", c.Name())
}

// could be a variadic function, but we are looking for speed and
// as it happens we always require joining 3 strings...
func joinStrings(s1, s2, s3 string) string {
	var buff bytes.Buffer

	buff.WriteString(s1)
	buff.WriteString(s2)
	buff.WriteString(s3)
	return buff.String()
}

func isClientCertPresent(req *http.Request) bool {
	return req != nil && req.TLS != nil && req.TLS.PeerCertificates != nil
}

func cbAuthorize(s authSource, privileges *auth.Privileges, credentials auth.Credentials,
	req *http.Request) (auth.AuthenticatedUsers, errors.Error) {

	// Create credentials - list and authenticated users to use for auth calls
	if credentials == nil {
		credentials = make(auth.Credentials)
	}
	authenticatedUsers := make(auth.AuthenticatedUsers, 0, len(credentials))
	credentialsList := make([]cbauth.Creds, 0, 2)

	// Query allows 4 kinds of authorization methods -
	// 		 1. Basic Auth
	//		 2. Auth Header/Token
	//		 3. Certificates
	//		 4. Creds query parameter

	// The call to AuthWebCreds takes care of the first 3 checks.
	// This needs to be performed first. The following is taken care of by authwebcreds -

	// X509 - mandatory
	// Only certs can be used to authorize. No other method.

	// X509 - disable
	// 		1. Certificates,Basic Auth or Auth header can be used to authorize

	// X509 - enable
	// 		1. Cert needs to be used to authorize if present
	//		2. If cert not present then we need to use other methods
	//		   (partially done by auth web creds)

	if req != nil {
		creds, err := s.authWebCreds(req)
		if err == nil {
			credentialsList = append(credentialsList, creds)
			authenticatedUsers = append(authenticatedUsers, userKeyString(creds))
		} else if err.Error() == "No web credentials found in request." {
			// Do nothing.
		} else {

			clientAuthType, err1 := cbauth.GetClientCertAuthType()
			if err1 != nil {
				return nil, errors.NewDatastoreAuthorizationError(err1)
			}

			// If enable or mandatory and client cert is present, and you see an error
			// Then return the error.
			if clientAuthType != tls.NoClientCert && isClientCertPresent(req) {
				return nil, errors.NewDatastoreAuthorizationError(err)
			}
		}
	}

	// Either error isn't nil and mode is disable and error is nil

	// If we could not successfully authorize above, and
	//  - if x509 is disable or
	//  - if x509 is enable and cert is not present
	// we need to check if creds can be used to authorize

	// request could be nil - in which case we handle credentials only

	if len(credentialsList) == 0 {
		for username, password := range credentials {
			var un string
			userCreds := strings.Split(username, ":")
			if len(userCreds) == 1 {
				un = userCreds[0]
			} else {
				un = userCreds[1]
			}

			logging.Debugf(" Credentials for user <ud>%v</ud>", un)
			creds, err := s.auth(un, password)
			if err != nil {
				logging.Debugf("Unable to authorize <ud>%s</ud>. Error - %v", username, err)
			} else {
				credentialsList = append(credentialsList, creds)
				if un != "" {
					authenticatedUsers = append(authenticatedUsers, userKeyString(creds))
				}
			}
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
		return nil, errors.NewDatastoreAuthorizationError(err)
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
		return nil, errors.NewDatastoreAuthorizationError(err)
	}

	if len(deniedPrivileges) == 0 {
		// Authorized using defaults.
		return authenticatedUsers, nil
	}

	msg := messageForDeniedPrivilege(deniedPrivileges[0])
	return nil, errors.NewDatastoreInsufficientCredentials(msg)
}

func messageForDeniedPrivilege(pair auth.PrivilegePair) string {
	_, keyspace := namespaceKeyspaceFromPrivPair(pair)

	privilege := ""
	role := ""
	switch pair.Priv {
	case auth.PRIV_READ:
		privilege = "data read queries"
		role = fmt.Sprintf("data_reader on bucket %s", keyspace)
	case auth.PRIV_WRITE:
		privilege = "data write queries"
		role = fmt.Sprintf("data_reader_writer on bucket %s", keyspace)
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
		privilege = fmt.Sprintf("SELECT queries on the %s bucket", keyspace)
		role = fmt.Sprintf("query_select on %s", keyspace)
	case auth.PRIV_QUERY_UPDATE:
		privilege = fmt.Sprintf("UPDATE queries on the %s bucket", keyspace)
		role = fmt.Sprintf("query_update on %s", keyspace)
	case auth.PRIV_QUERY_INSERT:
		privilege = fmt.Sprintf("INSERT queries on the %s bucket", keyspace)
		role = fmt.Sprintf("query_insert on %s", keyspace)
	case auth.PRIV_QUERY_DELETE:
		privilege = fmt.Sprintf("DELETE queries on the %s bucket", keyspace)
		role = fmt.Sprintf("query_delete on %s", keyspace)
	case auth.PRIV_QUERY_BUILD_INDEX, auth.PRIV_QUERY_CREATE_INDEX,
		auth.PRIV_QUERY_ALTER_INDEX, auth.PRIV_QUERY_DROP_INDEX, auth.PRIV_QUERY_LIST_INDEX:
		privilege = "index operations"
		role = fmt.Sprintf("query_manage_index on %s", keyspace)
	case auth.PRIV_QUERY_EXTERNAL_ACCESS:
		privilege = "queries using the CURL() function"
		role = "query_external_access"
	default:
		privilege = "this type of query"
		role = "admin"
	}

	return fmt.Sprintf("User does not have credentials to run %s. Add role %s to allow the query to run.", privilege, role)
}
