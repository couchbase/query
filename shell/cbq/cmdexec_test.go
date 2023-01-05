//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package main

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/couchbase/godbc/n1ql"
	"github.com/couchbase/query/shell/cbq/command"
	"github.com/couchbase/query/shell/liner"
)

var Server = "http://localhost:8091"

func execline(line string, t *testing.T) {
	var liner, _ = liner.NewLiner(false)
	defer liner.Close()

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	command.SetWriter(w)

	errC, errS := dispatch_command(line, w, true, liner)
	w.Flush()
	if errC != 0 {
		t.Errorf("Error :: %v, %s", line, command.HandleError(errC, errS))
	} else {
		t.Log("Ran command ", line, " successfully.")
		t.Logf("%s", b.String())
	}
	b.Reset()
}

func TestExecuteInput(t *testing.T) {
	//Test comments
	line := "--This is a comment"
	execline(line, t)

	line = "#This is a comment"
	execline(line, t)

	//Test shell command execution. Also test if command is case insensitive.
	line = "\\Echo \\\\serverversion This is select ()"
	execline(line, t)

}

// This function tests both the n1ql execution function
// and the write helper method to write the output of the
// query. For this test, we will use the dummy bucket shellTest.

func execn1ql(line string, t *testing.T) bool {

	var liner, _ = liner.NewLiner(false)
	defer liner.Close()

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	command.SetWriter(w)

	dBn1ql, err := n1ql.OpenExtended(Server, "test")
	n1ql.SetUsernamePassword("Administrator", "password")
	if err != nil {
		// If the test cannot connect to a server
		// don't execute the TestExecN1QLStmt method.
		testconn := false
		t.Logf("Cannot connect to %v", Server)
		return testconn
	} else {
		//Successfully logged into the server
		//For the case where server url is valid
		//sql.Open will not throw an error. Hence ping
		//the server and see if it returns an error.

		err = dBn1ql.Ping()

		if err != nil {
			testconn := false
			t.Logf("Cannot connect to %v", Server)
			return testconn
		}

		errC, errS := ExecN1QLStmt(line, dBn1ql, w)
		w.Flush()
		if errC != 0 {
			t.Errorf("Error executing statement : %v", line)
			t.Error(command.HandleError(errC, errS))
		} else {
			t.Log("Ran command ", line, " successfully.")
			t.Logf("%s", b.String())
		}
	}
	b.Reset()
	return true
}

func TestExecN1QLStmt(t *testing.T) {
	//Run the tests against couchbase only if CBSOURCE
	//environment variable is set to true.

	line := "\\SET -scan_consistency REQUEST_PLUS"
	execshell(line, t)

	line = "create primary index on shellTest"
	testconn := execn1ql(line, t)
	if testconn == true {
		t.Log("Testing N1QL statements")

		//test to check output where all values are missing.
		//test for MB-17509
		line = "SELECT feature_name, IFNAN(story_point[2],story_point[1]) as point FROM shellTest ORDER BY feature_name"
		execn1ql(line, t)

		line = "SELECT RAW (feature_name) as point FROM shellTest ORDER BY feature_name"
		execn1ql(line, t)

		line = "SELECT element (feature_name) as point FROM shellTest ORDER BY feature_name"
		execn1ql(line, t)

		//Insert data into shellTest
		line = "insert into shellTest values (\"1\", {\"name\" : \"Mission Peak\" , \"distance\" : 6}), (\"2\", {\"name\" : \"Black Mountain\", \"Location\" : \"Santa Clara County\", \"distance\" : 17})"
		execn1ql(line, t)

		//select from bucket where
		line = "Select name, distance from shellTest"
		execn1ql(line, t)

		//upsert data
		line = "upsert into shellTest values (\"1\", {\"name\" : \"Mission Peak\" , \"Location\" : \"Fremont\", \"distance\" : 6.5})"
		execn1ql(line, t)

		//query using named and positional parameters
		line = "\\SET -args [6]"
		execshell(line, t)

		line = "\\SET -$max_distance 15"
		execshell(line, t)

		line = "\\echo -$max_distance -args"
		execshell(line, t)

		line = "select * from shellTest where distance > $1 and distance < $max_distance"
		execn1ql(line, t)

		//prepared statement
		line = "prepare test from select * from shellTest where distance > $1 and distance < $max_distance"
		execn1ql(line, t)

		line = "execute test"
		execn1ql(line, t)

		//update
		line = "update shellTest set distance = 10 where name = \"Black Mountain\""
		execn1ql(line, t)

		//See the change in the results
		t.Log("Results change to incorporate update :: ")
		line = "execute test"
		execn1ql(line, t)

		line = "delete from shellTest"
		execn1ql(line, t)

		line = "drop primary index on shellTest"
		execn1ql(line, t)

		//Since the sample.txt file also contain n1ql statements,
		//run this test only if n1ql connection is possible.
		testFileCmd(t)

		//Test executing alias
		serverFlag = Server
		line = "\\\\serverversion"
		execline(line, t)

		//Test a sample N1QL command execution
		line = "select version()"
		execline(line, t)

	}
}

func testFileCmd(t *testing.T) {
	// \SOURCE
	//Also test if command is case insensitive.
	line := "\\SOurCE sample.txt"
	execshell(line, t)

}

func execshell(line string, t *testing.T) {

	var liner, _ = liner.NewLiner(false)
	defer liner.Close()

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	command.SetWriter(w)

	errC, errS := ExecShellCmd(line, liner)
	w.Flush()
	if errC != 0 {
		if strings.HasPrefix(line, "\\Sample") {
			t.Logf("Expected Error :: %v, %s", line, command.HandleError(errC, errS))
		} else {
			t.Errorf("Error :: %v, %s", line, command.HandleError(errC, errS))
		}
	} else {
		if strings.HasPrefix(line, "\\Sample") {
			t.Errorf("Expected error for command %v. It doesnt exist.", line)
		} else {
			t.Log("Ran command ", line, " successfully.")
			t.Logf("%s", b.String())
		}

	}
	b.Reset()
}

func TestExecShellCmd(t *testing.T) {
	//Test shell commands
	// Command doesnt exist.
	line := "\\Sample"
	execshell(line, t)

	// \ALIAS
	line = "\\alias tempcommand select * from `beer-sample`"
	execshell(line, t)

	// \ECHO
	line = "\\ECHO \\\\tempcommand histfile histsize \\\"test\\\""
	execshell(line, t)

}
