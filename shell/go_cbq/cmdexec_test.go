//  Copyright (c) 2015-2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package main

import (
	"bufio"
	"bytes"
	"database/sql"
	//"os"
	"strings"
	"testing"

	go_n1ql "github.com/couchbase/go_n1ql"
	"github.com/couchbase/query/shell/go_cbq/command"
	"github.com/sbinet/liner"
)

var Server = "http://127.0.0.1:8091"

func execline(line string, t *testing.T) {
	go_n1ql.SetPassthroughMode(true)
	var liner = liner.NewLiner()
	defer liner.Close()

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	command.SetWriter(w)

	errC, errS := execute_input(line, w, true, liner)
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
// query. For this test, we will use the dummy bucket default.

func execn1ql(line string, t *testing.T) bool {
	go_n1ql.SetPassthroughMode(true)
	var liner = liner.NewLiner()
	defer liner.Close()

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	command.SetWriter(w)

	n1ql, err := sql.Open("n1ql", Server)
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

		err = n1ql.Ping()

		if err != nil {
			testconn := false
			t.Logf("Cannot connect to %v", Server)
			return testconn
		}

		errC, errS := ExecN1QLStmt(line, n1ql, w)
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
		ServerFlag = Server
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
	go_n1ql.SetPassthroughMode(true)
	var liner = liner.NewLiner()
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
