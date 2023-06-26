//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package couchbase

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/tenant"
)

func privilegeString(namespace, target, obj string, requested auth.Privilege) (string, error) {
	var permission string
	switch requested {
	case auth.PRIV_WRITE:
		permission = join5Strings("cluster.", obj, "[", target, "].data.docs!write")
	case auth.PRIV_READ:
		permission = join5Strings("cluster.", obj, "[", target, "].data.docs!read")
	case auth.PRIV_UPSERT:
		permission = join5Strings("cluster.", obj, "[", target, "].data.docs!upsert")
	case auth.PRIV_SYSTEM_OPEN:
		fallthrough
	case auth.PRIV_SYSTEM_READ:
		permission = "cluster.n1ql.meta!read"
	case auth.PRIV_SECURITY_READ:
		permission = "cluster.admin.security!read"
	case auth.PRIV_SECURITY_WRITE:
		permission = "cluster.admin.security!write"
	case auth.PRIV_QUERY_SELECT:
		permission = join5Strings("cluster.", obj, "[", target, "].n1ql.select!execute")
	case auth.PRIV_QUERY_UPDATE:
		permission = join5Strings("cluster.", obj, "[", target, "].n1ql.update!execute")
	case auth.PRIV_QUERY_INSERT:
		permission = join5Strings("cluster.", obj, "[", target, "].n1ql.insert!execute")
	case auth.PRIV_QUERY_DELETE:
		permission = join5Strings("cluster.", obj, "[", target, "].n1ql.delete!execute")
	case auth.PRIV_QUERY_BUILD_INDEX:
		permission = join5Strings("cluster.", obj, "[", target, "].n1ql.index!build")
	case auth.PRIV_QUERY_CREATE_INDEX:
		// In serverless mode, check for privileges to create regular indexes
		if tenant.IsServerless() {
			permission = join5Strings("cluster.", obj, "[", target, "].n1ql.index!create")
		} else { // In on-prem mode, check for privileges to create indexes with no restrictions on parameters in WITH clause
			permission = join5Strings("cluster.", obj, "[", target, "].n1ql.index.parameterized!create")
		}
	case auth.PRIV_QUERY_ALTER_INDEX:
		permission = join5Strings("cluster.", obj, "[", target, "].n1ql.index!alter")
	case auth.PRIV_QUERY_DROP_INDEX:
		permission = join5Strings("cluster.", obj, "[", target, "].n1ql.index!drop")
	case auth.PRIV_QUERY_LIST_INDEX:
		permission = join5Strings("cluster.", obj, "[", target, "].n1ql.index!list")
	case auth.PRIV_QUERY_EXTERNAL_ACCESS:
		permission = "cluster.n1ql.curl!execute"
	case auth.PRIV_QUERY_MANAGE_FUNCTIONS:
		permission = "cluster.n1ql.udf!manage"
	case auth.PRIV_QUERY_EXECUTE_FUNCTIONS:
		permission = "cluster.n1ql.udf!execute"
	case auth.PRIV_QUERY_MANAGE_SCOPE_FUNCTIONS:
		permission = join5Strings("cluster.", obj, "[", target, "].n1ql.udf!manage")
	case auth.PRIV_QUERY_EXECUTE_SCOPE_FUNCTIONS:
		permission = join5Strings("cluster.", obj, "[", target, "].n1ql.udf!execute")
	case auth.PRIV_QUERY_MANAGE_FUNCTIONS_EXTERNAL:
		permission = "cluster.n1ql.udf_external!manage"
	case auth.PRIV_QUERY_EXECUTE_FUNCTIONS_EXTERNAL:
		permission = "cluster.n1ql.udf_external!execute"
	case auth.PRIV_QUERY_MANAGE_SCOPE_FUNCTIONS_EXTERNAL:
		permission = join5Strings("cluster.", obj, "[", target, "].n1ql.udf_external!manage")
	case auth.PRIV_QUERY_EXECUTE_SCOPE_FUNCTIONS_EXTERNAL:
		permission = join5Strings("cluster.", obj, "[", target, "].n1ql.udf_external!execute")
	case auth.PRIV_QUERY_BUCKET_ADMIN:
		permission = join3Strings("cluster.bucket[", target, "]!manage")
	case auth.PRIV_QUERY_SCOPE_ADMIN:
		permission = join5Strings("cluster.", obj, "[", target, "].collections!write")
	case auth.PRIV_QUERY_STATS:
		permission = "cluster.admin.internal.stats!read"
	case auth.PRIV_BACKUP_CLUSTER:
		permission = "cluster.n1ql.meta!backup"
	case auth.PRIV_BACKUP_BUCKET:
		permission = join3Strings("cluster.bucket[", target, "].n1ql.meta!backup")
	case auth.PRIV_XATTRS:
		permission = join5Strings("cluster.", obj, "[", target, "].data.sxattr!read")
	case auth.PRIV_ADMIN:

		// this is a special case - to check that the user is an admin, we check an impossible privilege
		// only administrators pass checks on undefined privileges
		permission = "cluster.admin.internal.nothrottle!read"
	case auth.PRIV_CLUSTER_ADMIN:
		permission = "cluster.admin!write"
	default:
		return "", fmt.Errorf("Invalid Privileges")
	}
	return permission, nil
}

func doAuthByCreds(creds cbauth.Creds, permission string) (bool, error) {
	var err error

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

// splits the target into namespace and couchbase target (bucket, or collection path separated by :)
// determines the type of object to check (bucket, scope, collection)
func namespaceKeyspaceTypeFromPrivPair(pair auth.PrivilegePair) (string, string, string) {
	var target string
	var obj string

	elems := algebra.ParsePath(pair.Target)
	namespace := elems[0]
	if namespace == "" {
		namespace = "default"
	}

	switch len(elems) {
	case 2:
		obj = "bucket"
		target = elems[1]
	case 3:
		obj = "scope"
		target = join3Strings(elems[1], ":", elems[2])
	case 4:
		obj = "collection"
		target = join5Strings(elems[1], ":", elems[2], ":", elems[3])
	}
	return namespace, target, obj
}

type cbPrecompiled string

// Try to get the privileges sought from the availableCredentials credentials.
// Return the privileges that were not granted.
func authAgainstCreds(as authSource, privsSought []auth.PrivilegePair, availableCredentials []cbauth.Creds) ([]auth.PrivilegePair, error) {
	var deniedPrivs []auth.PrivilegePair
	for p, _ := range privsSought {
		var res bool
		var err error

		privilege := privsSought[p].Priv
		if privilege == auth.PRIV_QUERY_TRANSACTION_STMT {
			// ignore transaction statements
			continue
		}
		if privilege == auth.PRIV_SYSTEM_OPEN && as.adminIsOpen() {
			// If all buckets have passwords, the API requires credentials.
			// But if any don't, the API is open to read.
			continue
		}

		precompiled, ok := privsSought[p].Ready.(cbPrecompiled)
		if ok {
			res, err = authAgainstCred(string(precompiled), availableCredentials)
		} else {
			var target string

			namespace, keyspace, obj := namespaceKeyspaceTypeFromPrivPair(privsSought[p])
			target, err = privilegeString(namespace, keyspace, obj, privilege)
			if err == nil {
				res, err = authAgainstCred(target, availableCredentials)
			}
		}

		if err != nil {
			return nil, err
		}

		// This privilege can not be granted by these credentials.
		if !res {
			deniedPrivs = append(deniedPrivs, privsSought[p])
		}
	}
	return deniedPrivs, nil
}

func authAgainstCred(cbTarget string, availableCredentials []cbauth.Creds) (bool, error) {
	thisPrivGranted := false

	// Check requested privilege against the list of credentials.
	for _, creds := range availableCredentials {
		authResult, err := doAuthByCreds(creds, cbTarget)

		if err != nil {
			return false, err
		}

		// Auth succeeded
		if authResult == true {
			thisPrivGranted = true
			break
		}
	}
	return thisPrivGranted, nil
}

func userKeyString(c cbauth.Creds) string {
	return join3Strings(c.Domain(), ":", c.Name())
}

// could be variadic functions, but we are looking for speed
func join3Strings(s1, s2, s3 string) string {
	var buff bytes.Buffer

	buff.WriteString(s1)
	buff.WriteString(s2)
	buff.WriteString(s3)
	return buff.String()
}

func join5Strings(s1, s2, s3, s4, s5 string) string {
	var buff bytes.Buffer

	buff.WriteString(s1)
	buff.WriteString(s2)
	buff.WriteString(s3)
	buff.WriteString(s4)
	buff.WriteString(s5)
	return buff.String()
}

func isClientCertPresent(req *http.Request) bool {
	return req != nil && req.TLS != nil && req.TLS.PeerCertificates != nil
}

func cbAuthorize(s authSource, privileges *auth.Privileges, credentials *auth.Credentials) errors.Error {

	var reason error

	if credentials.AuthenticatedUsers == nil {

		var authenticatedUsers auth.AuthenticatedUsers
		var credentialsList []cbauth.Creds

		// Create credentials - list and authenticated users to use for auth calls
		if credentials.Users == nil {
			credentials.Users = make(map[string]string, 0)
		}
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

		req := credentials.HttpRequest
		if req != nil {
			creds, err := s.authWebCreds(req)
			if err == nil {
				credentialsList = append(credentialsList, creds)
				authenticatedUsers = append(authenticatedUsers, userKeyString(creds))
			} else {
				logging.Debuga(func() string {
					u, _, _ := cbauth.ExtractCredsGeneric(req.Header)
					return fmt.Sprintf("authWebCreds: <ud>%v</ud> - %v", u, err)
				})

				clientAuthType, err1 := cbauth.GetClientCertAuthType()
				if err1 != nil {
					return errors.NewDatastoreAuthorizationError(err1)
				}

				// If enable or mandatory and client cert is present, and you see an error
				// Then return the error.
				if clientAuthType != tls.NoClientCert && isClientCertPresent(req) {
					return errors.NewDatastoreAuthorizationError(err)
				} else if clientAuthType == tls.NoClientCert {
					impersonation, _, _ := cbauth.ExtractOnBehalfIdentityGeneric(req.Header)
					if impersonation != "" {
						reason = errors.NewDatastoreAuthorizationError(err)
					}
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
			for username, password := range credentials.Users {
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
					return errors.NewDatastoreAuthorizationError(err)
				} else {
					reason = nil
					credentialsList = append(credentialsList, creds)
					if un != "" {
						un = userKeyString(creds)
						found := false
						for _, u := range authenticatedUsers {
							if un == u {
								found = true
								break
							}
						}
						if !found {
							authenticatedUsers = append(authenticatedUsers, un)
						}
					}
				}
			}
		}

		credentials.AuthenticatedUsers = authenticatedUsers
		credentials.CbauthCredentialsList = credentialsList
	}

	// No privileges to check? Done.
	if privileges == nil || len(privileges.List) == 0 {
		return nil
	}

	// Check every requested privilege against the credentials list.
	// if the authentication fails for any of the requested privileges return an error
	deniedPrivileges, err := authAgainstCreds(s, privileges.List, credentials.CbauthCredentialsList)

	if err != nil {
		return errors.NewDatastoreAuthorizationError(err)
	}

	if len(deniedPrivileges) == 0 {
		// Everything is authorized. Success!
		return nil
	}

	msg := messageForDeniedPrivilege(deniedPrivileges[0])

	var path []string
	if deniedPrivileges[0].Target != "" {
		path = algebra.ParsePath(deniedPrivileges[0].Target)
	}

	return errors.NewDatastoreInsufficientCredentials(msg, reason, path)
}

func cbPreAuthorize(privileges *auth.Privileges) {
	if privileges == nil {
		return
	}
	for i, _ := range privileges.List {
		if privileges.List[i].Priv == auth.PRIV_QUERY_TRANSACTION_STMT {
			// ignore transaction statements
			continue
		}

		namespace, keyspace, obj := namespaceKeyspaceTypeFromPrivPair(privileges.List[i])
		p, err := privilegeString(namespace, keyspace, obj, privileges.List[i].Priv)
		if err == nil && p != "" {
			privileges.List[i].Ready = cbPrecompiled(p)
		}
	}
}
