//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package auth

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/couchbase/cbauth"
)

type Privilege int

const (
	PRIV_READ                                   Privilege = 1
	PRIV_WRITE                                  Privilege = 2
	PRIV_SYSTEM_OPEN                            Privilege = 3  // Access to keyspaces in the system namespace, which may or may not be open.
	PRIV_SYSTEM_READ                            Privilege = 4  // Access to keyspaces in the system namespace, such as system:keyspaces.
	PRIV_SECURITY_READ                          Privilege = 5  // Reading user information.
	PRIV_SECURITY_WRITE                         Privilege = 6  // Updating user information.
	PRIV_QUERY_SELECT                           Privilege = 7  // Ability to run SELECT statements.
	PRIV_QUERY_UPDATE                           Privilege = 8  // Ability to run UPDATE statements.
	PRIV_QUERY_INSERT                           Privilege = 9  // Ability to run INSERT statements.
	PRIV_QUERY_DELETE                           Privilege = 10 // Ability to run DELETE statements.
	PRIV_QUERY_BUILD_INDEX                      Privilege = 11 // Ability to run BUILD INDEX statements.
	PRIV_QUERY_CREATE_INDEX                     Privilege = 12 // Ability to run CREATE INDEX statements.
	PRIV_QUERY_ALTER_INDEX                      Privilege = 13 // Ability to run ALTER INDEX statements.
	PRIV_QUERY_DROP_INDEX                       Privilege = 14 // Ability to run DROP INDEX statements.
	PRIV_QUERY_LIST_INDEX                       Privilege = 15 // Ability to list indexes of a keyspace.
	PRIV_QUERY_EXTERNAL_ACCESS                  Privilege = 16 // Ability to access the web from a N1QL query.
	PRIV_QUERY_MANAGE_FUNCTIONS                 Privilege = 17 // Ability to run CREATE / DROP  FUNCTION statements.
	PRIV_QUERY_EXECUTE_FUNCTIONS                Privilege = 18 // Ability to run EXECUTE FUNCTION statements.
	PRIV_QUERY_MANAGE_SCOPE_FUNCTIONS           Privilege = 19 // Ability to run CREATE / DROP  FUNCTION statements.
	PRIV_QUERY_EXECUTE_SCOPE_FUNCTIONS          Privilege = 20 // Ability to run EXECUTE FUNCTION statements.
	PRIV_QUERY_MANAGE_FUNCTIONS_EXTERNAL        Privilege = 21 // Ability to run CREATE / DROP  FUNCTION statements.
	PRIV_QUERY_EXECUTE_FUNCTIONS_EXTERNAL       Privilege = 22 // Ability to run EXECUTE FUNCTION statements.
	PRIV_QUERY_MANAGE_SCOPE_FUNCTIONS_EXTERNAL  Privilege = 23 // Ability to run CREATE / DROP  FUNCTION statements.
	PRIV_QUERY_EXECUTE_SCOPE_FUNCTIONS_EXTERNAL Privilege = 24 // Ability to run EXECUTE FUNCTION statements.
	PRIV_QUERY_BUCKET_ADMIN                     Privilege = 25 // Ability to manage buckets
	PRIV_QUERY_STATS                            Privilege = 26 // Ability to read query stats
	PRIV_QUERY_TRANSACTION_STMT                 Privilege = 27 // Ability to run Transaction statements.
	PRIV_UPSERT                                 Privilege = 28 // Ability to run docs UPSERT
	PRIV_BACKUP_CLUSTER                         Privilege = 29 // Ability to backup cluster level N1QL metadata
	PRIV_BACKUP_BUCKET                          Privilege = 30 // Ability to backup bucket level N1QL metadata
	PRIV_QUERY_SCOPE_ADMIN                      Privilege = 31 // Ability to add, drop, flush scopes and collections
	PRIV_XATTRS                                 Privilege = 32 // Ability to read system xattrs
	PRIV_ADMIN                                  Privilege = 33 // User is a full or read admin
	PRIV_CLUSTER_ADMIN                          Privilege = 34 // User has cluster_admin auth
	PRIV_QUERY_MANAGE_SEQUENCES                 Privilege = 35 // CREATE/ALTER/DROP sequences
	PRIV_QUERY_USE_SEQUENCES                    Privilege = 36 // get/advance sequence values
)

type PrivilegePair struct {
	Target string // For what resource is the privilege requested. Typically a string of
	// the form "namespace:keyspace". Could be blank, for system-wide
	// privileges
	Priv Privilege // The level of privilege requested. Note there could be multiple
	// privileges against the same target.
	Props int // propoerties of this privilage
	// Privileges that have been precompiled, if possible
	// this is store specific
	// Since it is specific to the store, it's never marshalled or unmarshalled
	Ready interface{} `json:"-"`
}

const (
	PRIV_PROPS_DYNAMIC_TARGET = 1 << iota
	PRIV_PROPS_NONE           = 0
)

// A set of permissions required, typically to run a specific query.
type Privileges struct {
	List []PrivilegePair
}

var NO_PRIVILEGES = NewPrivileges()

func NewPrivileges() *Privileges {
	return &Privileges{List: make([]PrivilegePair, 0, 16)}
}

func (this *Privileges) Num() int {
	return len(this.List)
}

func (this *Privileges) AddAll(other *Privileges) {
	if other == nil {
		return
	}
	for _, pair := range other.List {
		this.Add(pair.Target, pair.Priv, pair.Props)
	}
}

func (this *Privileges) ForEach(f func(PrivilegePair)) {
	for _, pair := range this.List {
		f(pair)
	}
}

func (this *Privileges) AddPair(pp PrivilegePair) {
	for _, pair := range this.List {
		if pair.Target == pp.Target && pair.Priv == pp.Priv && pair.Props == pp.Props {
			// already present
			return
		}
	}
	this.List = append(this.List, pp)
}

func (this *Privileges) Add(target string, priv Privilege, Props int) {
	for _, pair := range this.List {
		if pair.Target == target && pair.Priv == priv && pair.Props == Props {
			// already present
			return
		}
	}
	this.List = append(this.List, PrivilegePair{Target: target, Priv: priv, Props: Props})
}

/*
Type Credentials maps users to passwords.
*/

type Users map[string]string

type Credentials struct {
	Users                 Users
	HttpRequest           *http.Request
	AuthenticatedUsers    AuthenticatedUsers
	CbauthCredentialsList []cbauth.Creds
}

func NewCredentials() *Credentials {
	rv := &Credentials{}
	rv.Users = make(Users, 0)
	return rv
}

/*
Type AuthenticatedUsers is a list of users whose credentials checked out.
*/
type AuthenticatedUsers []string

// FIXME this should be provided by cbauth in order to support on behalf of
func GetWebAuth(req *http.Request) (string, string, error) {
	headers := req.Header["Authorization"]
	if len(headers) == 0 {
		return "", "", fmt.Errorf("no http request authorization found")
	} else if len(headers) > 1 {
		return "", "", fmt.Errorf("too many http request authorizations found")
	}
	if !strings.HasPrefix(headers[0], "Basic ") {
		return "", "", fmt.Errorf("no http request authorization found")
	}
	encoded_creds := strings.Split(headers[0], " ")[1]
	decoded_creds, err := base64.StdEncoding.DecodeString(encoded_creds)
	if err != nil {
		return "", "", err
	}

	// Authorization header is in format "user:pass"
	// per http://tools.ietf.org/html/rfc1945#section-10.2
	u_details := strings.Split(string(decoded_creds), ":")
	if len(u_details) == 2 {
		return u_details[0], u_details[1], nil
	}
	return "", "", fmt.Errorf("no valid user details found")
}
