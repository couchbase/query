//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package couchbase

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/auth"
)

type authSourceImpl struct {
	users []authUser
}

func (asi *authSourceImpl) adminIsOpen() bool {
	return false
}

func (asi *authSourceImpl) auth(id, pwd string) (cbauth.Creds, error) {
	for _, user := range asi.users {
		if user.id == id {
			if user.password == pwd {
				return user, nil
			}
			return nil, fmt.Errorf("Invalid password %s supplied for user %s.", pwd, id)
		}
	}
	return nil, fmt.Errorf("Could not find user %s.", id)
}

func (asi *authSourceImpl) authWebCreds(req *http.Request) (cbauth.Creds, error) {
	return nil, fmt.Errorf("authWebCreds is not implemented")
}

// authUser implements cbauth.Creds
type authUser struct {
	id          string
	password    string
	permissions map[string]bool
}

func (a authUser) Name() string {
	return a.id
}

func (a authUser) Source() string {
	return a.Domain()
}

func (a authUser) Domain() string {
	return "internal"
}

func (a authUser) User() (string, string) {
	return a.id, a.Domain()
}

func (a authUser) Uuid() (string, error) {
	return "internal", nil
}

func (a authUser) IsAllowed(permission string) (bool, error) {
	return a.permissions[permission], nil
}

type testCase struct {
	purpose       string
	authSource    authSource
	privs         *auth.Privileges
	creds         *auth.Credentials
	shouldSucceed bool
}

func TestGrantRole(t *testing.T) {
	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_SECURITY_WRITE, auth.PRIV_PROPS_NONE)

	as := &authSourceImpl{
		users: []authUser{
			authUser{id: "bob", password: "pwbob",
				permissions: map[string]bool{
					"cluster.admin.security!write":                   true,
					"cluster.bucket[testbucket].n1ql.select!execute": true,
				},
			},
			authUser{id: "nancy", password: "pwnancy",
				permissions: map[string]bool{
					"cluster.bucket[testbucket].data.docs!read": true,
				},
			},
		},
	}

	cases := []testCase{
		testCase{purpose: "Insufficient Credentials", authSource: as, privs: privs, creds: &auth.Credentials{map[string]string{"nancy": "pwnancy"}, nil, nil, nil}},
		testCase{purpose: "Works", authSource: as, privs: privs, creds: &auth.Credentials{map[string]string{"bob": "pwbob"}, nil, nil, nil}, shouldSucceed: true},
	}
	runCases(t, cases)
}

func TestSimpleSelect(t *testing.T) {
	privs := auth.NewPrivileges()
	privs.Add("testbucket", auth.PRIV_QUERY_SELECT, auth.PRIV_PROPS_NONE)

	as := &authSourceImpl{
		users: []authUser{
			authUser{id: "bob", password: "pwbob",
				permissions: map[string]bool{
					"cluster.bucket[testbucket].n1ql.select!execute": true,
				},
			},
			authUser{id: "nancy", password: "pwnancy",
				permissions: map[string]bool{
					"cluster.bucket[testbucket].data.docs!read": true,
				},
			},
		},
	}

	cases := []testCase{
		testCase{purpose: "No Credentials", authSource: as, privs: privs, creds: &auth.Credentials{}},
		testCase{purpose: "Insufficient Credentials", authSource: as, privs: privs, creds: &auth.Credentials{map[string]string{"nancy": "pwnancy"}, nil, nil, nil}},
		testCase{purpose: "Wrong password", authSource: as, privs: privs, creds: &auth.Credentials{map[string]string{"bob": "badpassword"}, nil, nil, nil}},
		testCase{purpose: "Works", authSource: as, privs: privs, creds: &auth.Credentials{map[string]string{"bob": "pwbob"}, nil, nil, nil}, shouldSucceed: true},
	}
	runCases(t, cases)
}

func runCases(t *testing.T, cases []testCase) {
	for _, c := range cases {
		err := cbAuthorize(c.authSource, c.privs, c.creds)
		if c.shouldSucceed {
			if err != nil {
				t.Fatalf("Case %s should succeed, but it failed with error %v.", c.purpose, err)
			}
		} else {
			if err == nil {
				t.Fatalf("Case %s should fail, but it passed.", c.purpose)
			}
		}
	}
}

func TestDefaultCredentials(t *testing.T) {
	privs := auth.NewPrivileges()
	privs.Add("testbucket", auth.PRIV_QUERY_SELECT, auth.PRIV_PROPS_NONE)

	asNoDefault := &authSourceImpl{
		users: []authUser{
			authUser{id: "bob", password: "pwbob", permissions: map[string]bool{}},
		},
	}

	asWrongPerms := &authSourceImpl{
		users: []authUser{
			authUser{id: "bob", password: "pwbob", permissions: map[string]bool{}},
			authUser{id: "testbucket", password: "",
				permissions: map[string]bool{
					"cluster.bucket[wrong].data.docs!read":      true,
					"cluster.bucket[wrong].n1ql.select!execute": true,
				},
			},
		},
	}

	asWrongPassword := &authSourceImpl{
		users: []authUser{
			authUser{id: "bob", password: "pwbob", permissions: map[string]bool{}},
			authUser{id: "testbucket", password: "wrong",
				permissions: map[string]bool{
					"cluster.bucket[testbucket].data.docs!read":      true,
					"cluster.bucket[testbucket].n1ql.select!execute": true,
				},
			},
		},
	}

	asWorks := &authSourceImpl{
		users: []authUser{
			authUser{id: "bob", password: "pwbob", permissions: map[string]bool{}},
			authUser{id: "testbucket", password: "",
				permissions: map[string]bool{
					"cluster.bucket[testbucket].data.docs!read":      true,
					"cluster.bucket[testbucket].n1ql.select!execute": true,
				},
			},
		},
	}

	loginCreds := &auth.Credentials{map[string]string{"bob": "pwbob"}, nil, nil, nil}

	cases := []testCase{
		testCase{purpose: "No Default User", authSource: asNoDefault, privs: privs, creds: loginCreds},
		testCase{purpose: "Default User Has Wrong Permissions", authSource: asWrongPerms, privs: privs, creds: loginCreds},
		testCase{purpose: "Default User Has Unexpected Password", authSource: asWrongPassword, privs: privs, creds: loginCreds},
		testCase{purpose: "Works", authSource: asWorks, privs: privs, creds: loginCreds, shouldSucceed: true},
	}
	runCases(t, cases)
}

type deniedCase struct {
	data     auth.PrivilegePair
	expected string
}
