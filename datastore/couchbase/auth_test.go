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

func (a authUser) IsAllowed(permission string) (bool, error) {
	return a.permissions[permission], nil
}

type testCase struct {
	purpose       string
	authSource    authSource
	privs         *auth.Privileges
	creds         auth.Credentials
	shouldSucceed bool
}

func TestGrantRole(t *testing.T) {
	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_SECURITY_WRITE)

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
		testCase{purpose: "Insufficient Credentials", authSource: as, privs: privs, creds: auth.Credentials{"nancy": "pwnancy"}},
		testCase{purpose: "Works", authSource: as, privs: privs, creds: auth.Credentials{"bob": "pwbob"}, shouldSucceed: true},
	}
	runCases(t, cases)
}

func TestSimpleSelect(t *testing.T) {
	privs := auth.NewPrivileges()
	privs.Add("testbucket", auth.PRIV_QUERY_SELECT)

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
		testCase{purpose: "No Credentials", authSource: as, privs: privs, creds: auth.Credentials{}},
		testCase{purpose: "Insufficient Credentials", authSource: as, privs: privs, creds: auth.Credentials{"nancy": "pwnancy"}},
		testCase{purpose: "Wrong password", authSource: as, privs: privs, creds: auth.Credentials{"bob": "badpassword"}},
		testCase{purpose: "Works", authSource: as, privs: privs, creds: auth.Credentials{"bob": "pwbob"}, shouldSucceed: true},
	}
	runCases(t, cases)
}

func runCases(t *testing.T, cases []testCase) {
	for _, c := range cases {
		_, err := cbAuthorize(c.authSource, c.privs, c.creds, nil)
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
	privs.Add("testbucket", auth.PRIV_QUERY_SELECT)

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

	loginCreds := auth.Credentials{"bob": "pwbob"}

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

func TestMessageForDeniedPrivilege(t *testing.T) {
	cases := []deniedCase{
		deniedCase{data: auth.PrivilegePair{Target: "testbucket", Priv: auth.PRIV_QUERY_SELECT},
			expected: "User does not have credentials to run SELECT queries on the testbucket bucket. Add role query_select on testbucket to allow the query to run."},
		deniedCase{data: auth.PrivilegePair{Target: ":testbucket", Priv: auth.PRIV_QUERY_DROP_INDEX},
			expected: "User does not have credentials to run index operations. Add role query_manage_index on testbucket to allow the query to run."},
		deniedCase{data: auth.PrivilegePair{Target: "", Priv: auth.PRIV_SYSTEM_READ},
			expected: "User does not have credentials to run queries accessing the system tables. Add role query_system_catalog to allow the query to run."},
	}

	for i, c := range cases {
		result := messageForDeniedPrivilege(c.data)
		if result != c.expected {
			t.Fatalf("Error in case %d. Expected %q, got %q.", i, c.expected, result)
		}
	}
}
