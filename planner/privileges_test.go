package planner

import (
	"testing"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/parser/n1ql"
)

func verifyPrivs(t *testing.T, id string, expectedPrivs *auth.Privileges, gotPrivs *auth.Privileges) {
	if expectedPrivs == nil {
		t.Fatalf("Case %s: Unexpected nil in expectedPrivs", id)
	}
	if gotPrivs == nil {
		t.Fatalf("Case %s: Unexpected nil in gotPrivs", id)
	}
	if expectedPrivs.Num() != gotPrivs.Num() {
		t.Fatalf("Case %s: privileges are wrong length. Expected %v, got %v.", id, *expectedPrivs, *gotPrivs)
	}
outer:
	for _, pair := range expectedPrivs.List {
		for _, gPair := range gotPrivs.List {
			if pair == gPair {
				continue outer
			}
		}
		t.Fatalf("Case %s: Expected pair %v does not appear in received value %v", id, pair, *gotPrivs)
	}

}

type testCase struct {
	id            string
	text          string
	expectedPrivs *auth.Privileges
}

func runCase(t *testing.T, c *testCase) {
	stmt, err := n1ql.ParseStatement(c.text)
	if err != nil {
		t.Fatalf("Case %s: Unable to parse text: %v", c.id, err)
	}

	privs, err := stmt.Privileges()
	if err != nil {
		t.Fatalf("Case %s: Unable to get privileges of statement: %v", c.id, err)
	}

	verifyPrivs(t, c.id, c.expectedPrivs, privs)
}

func TestStatementPrivileges(t *testing.T) {
	testCases := []testCase{
		testCase{id: "Simple Select", text: "select CURL('http://ip.jsontest.com') as res",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{auth.PrivilegePair{Target: "", Priv: auth.PRIV_QUERY_EXTERNAL_ACCESS}}}},
		testCase{id: "Select in FROM", text: "select * from CURL('http://ip.jsontest.com') res",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{auth.PrivilegePair{Target: "", Priv: auth.PRIV_QUERY_EXTERNAL_ACCESS}}}},
		testCase{id: "Select in LIMIT", text: "select * from testbucket limit CURL('http://ip.jsontest.com')",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "", Priv: auth.PRIV_QUERY_EXTERNAL_ACCESS},
				auth.PrivilegePair{Target: ":testbucket", Priv: auth.PRIV_READ},
				auth.PrivilegePair{Target: ":testbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		testCase{id: "Select in WHERE", text: "select * from testbucket where foo = CURL('http://ip.jsontest.com')",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "", Priv: auth.PRIV_QUERY_EXTERNAL_ACCESS},
				auth.PrivilegePair{Target: ":testbucket", Priv: auth.PRIV_READ},
				auth.PrivilegePair{Target: ":testbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		testCase{id: "Insert from VALUES", text: "insert into testbucket values ('foo', CURL('http://ip.jsontest.com'))",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "", Priv: auth.PRIV_QUERY_EXTERNAL_ACCESS},
				auth.PrivilegePair{Target: ":testbucket", Priv: auth.PRIV_WRITE},
				auth.PrivilegePair{Target: ":testbucket", Priv: auth.PRIV_QUERY_INSERT},
			}}},
		testCase{id: "Select with USE KEYS", text: "select * from testbucket use keys CURL('http://ip.jsontest.com')",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "", Priv: auth.PRIV_QUERY_EXTERNAL_ACCESS},
				auth.PrivilegePair{Target: ":testbucket", Priv: auth.PRIV_READ},
				auth.PrivilegePair{Target: ":testbucket", Priv: auth.PRIV_QUERY_SELECT},
			}}},
		testCase{id: "Update with Subselect",
			text: "update default use keys 'mykey' SET myWebServiceEndpoint = (SELECT raw result FROM CURL('http://ip.jsontest.com') result )",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "", Priv: auth.PRIV_QUERY_EXTERNAL_ACCESS},
				auth.PrivilegePair{Target: ":default", Priv: auth.PRIV_WRITE},
				auth.PrivilegePair{Target: ":default", Priv: auth.PRIV_QUERY_UPDATE},
			}}},
		testCase{id: "Grant Role",
			text: "grant role data_reader(foo) to don",
			expectedPrivs: &auth.Privileges{List: []auth.PrivilegePair{
				auth.PrivilegePair{Target: "", Priv: auth.PRIV_SECURITY_WRITE},
			}}},
	}

	for _, testCase := range testCases {
		runCase(t, &testCase)
	}
}
