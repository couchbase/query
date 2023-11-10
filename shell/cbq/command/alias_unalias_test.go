//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package command

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/couchbase/query/errors"
)

/*
   Test the \ALIAS and \UNALIAS commands.
*/

func TestAlias(t *testing.T) {
	//Create Alias
	alias := COMMAND_LIST["\\alias"]
	s := make([]string, 2)

	s[0] = "command1"
	s[1] = "\\ECHO This \"Is A\" histfile"
	errCode, errStr := alias.ExecCommand(s)

	if AliasCommand[s[0]] != s[1] {
		t.Errorf("Alias %s not created properly.\n", s[0])
		t.Errorf("Error : %s", HandleError(errCode, errStr))
	} else {
		t.Logf(" The value of Alias %s is :: %s ", s[0], AliasCommand[s[0]])
	}

	//Unset Alias
	unalias := COMMAND_LIST["\\unalias"]
	tmp := []string{s[0]}
	errCode, errStr = unalias.ExecCommand(tmp)
	if errCode == errors.E_SHELL_NO_SUCH_ALIAS {
		t.Errorf("Error using \\UNALIAS %s", HandleError(errCode, errStr))
	} else {
		t.Logf("%s deleted using \\UNALIAS", s[0])
	}
}

func TestListAlias(t *testing.T) {
	//Without Args, to list all aliases.
	alias := COMMAND_LIST["\\alias"]
	tmp := make([]string, 0)

	var b bytes.Buffer
	writetmp := bufio.NewWriter(&b)
	SetOutput(writetmp, true)

	errCode, errStr := alias.ExecCommand(tmp)
	writetmp.Flush()

	if errCode == 0 {
		t.Logf("\n%s", b.String())
	} else {
		t.Error("Error with displaying ALIAS")
		t.Errorf("Error : %s ", HandleError(errCode, errStr))
	}
}

func TestAliasErrors(t *testing.T) {
	//Error Case 1 : Too few args
	alias := COMMAND_LIST["\\alias"]
	tmp := make([]string, 1)

	errCode, errStr := alias.ExecCommand(tmp)

	if errCode == errors.E_SHELL_TOO_FEW_ARGS {
		t.Log("Too Few arguments to \\ALIAS command.")
	} else {
		t.Error("Minimum number of args for \\ALIAS has changed.")
	}

	//Error Case 2 : Alias does not exist
	_, ok := AliasCommand["newcommand"]
	if !ok {
		t.Logf("ALIAS newcommand doesn't exist.")
	}

	//Error Case 3 : There are no aliases
	unalias := COMMAND_LIST["\\unalias"]

	//Delete the existing aliases.
	for key, _ := range AliasCommand {
		tmp[0] = key
		errCode, errStr := unalias.ExecCommand(tmp)
		if errCode == errors.E_SHELL_NO_SUCH_ALIAS {
			t.Errorf("Error using \\UNALIAS %s", HandleError(errCode, errStr))
		} else {
			t.Log("All ALIAS deleted using \\UNALIAS")
		}
	}

	tmp = make([]string, 0)

	//test case where too few args for \unalias
	errCode, errStr = unalias.ExecCommand(tmp)
	if errCode == errors.E_SHELL_TOO_FEW_ARGS {
		t.Logf("%s", HandleError(errCode, errStr))
	} else {
		t.Errorf("Minimum number of args for \\UNALIAS has changed.")
	}

	var b bytes.Buffer
	writetmp := bufio.NewWriter(&b)
	SetOutput(writetmp, true)

	errCode, errStr = alias.ExecCommand(tmp)
	writetmp.Flush()

	if errCode == errors.E_SHELL_NO_SUCH_ALIAS {
		t.Logf("%s", HandleError(errCode, errStr))
	} else {
		t.Errorf("Unknown Error %s", HandleError(errCode, errStr))
	}

}
