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
	"strings"
	"testing"
)

/*
   Test the common methods from common.go for push and pop.
*/

func pushval(args []string, pushv bool, t *testing.T) {
	errCode, errStr := PushOrSet(args, pushv)
	if errCode == 0 {
		t.Log("Push/ Set successful")
	} else {
		t.Error(HandleError(errCode, errStr))
	}
}

func TestPush_pushOrSet(t *testing.T) {
	// Case 1 : Push 2 values, then set.
	// Named Parameters.
	pushval(strings.Split("-$rate 5", " "), false, t)

	// Query Parameters
	pushval(strings.Split("-timeout \"10ms\"", " "), false, t)

	// User Defined parameters
	pushval(strings.Split("$tmpvar select * from `beer-sample`", " "), false, t)

	// Predefined parameters
	pushval(strings.Split("histfile .cbq_newhistory", " "), false, t)

	// Case 2 : \SET a value that exists.
	st, ok := NamedParam["rate"]
	if ok {
		v, errC, errS := st.Top()
		if errC == 0 {
			t.Log("Top value for rate :: ", v.Actual())
			pushval(strings.Split("-$rate 25", " "), true, t)
			v, errC, errS := st.Top()
			if errC == 0 {
				t.Log("New Top value for rate :: ", v.Actual())
			} else {
				t.Error(HandleError(errC, errS))
			}
		} else {
			t.Error(HandleError(errC, errS))
		}
	} else {
		t.Error("Named Parameter rate doesnt exist.")
	}

	// Case 3 : \SET a value that does not exist.
	pushval(strings.Split("-$newval 3", " "), true, t)
}

func pushparam(param map[string]*Stack, isrestp bool, isnamep bool, t *testing.T) {
	errCode, errStr := Pushparam_Helper(param, isrestp, isnamep)
	if errCode == 0 {
		t.Log("Push successful")
	} else {
		t.Error(HandleError(errCode, errStr))
	}
}

func TestPush_pushparamHelper(t *testing.T) {
	// Case 1 : Is a rest api parameter
	pushparam(QueryParam, true, false, t)
	//Case 2 : Is a named parameter
	pushparam(NamedParam, true, true, t)
	//Case 3 : Is neither a rest api parameter, not a named parameter
	pushparam(PreDefSV, false, false, t)
}

func TestPush(t *testing.T) {
	push := COMMAND_LIST["\\push"]

	//push a parameter without specifying a value.
	errCode, errStr := push.ExecCommand([]string{"-$rate"})
	if errCode != 0 {
		t.Log(HandleError(errCode, errStr))
	} else {
		t.Error("Push with one arg should return an error.")
	}

	//\PUSH without args should push the top value on the stack.
	errCode, errStr = push.ExecCommand([]string{})
	if errCode == 0 {
		t.Log("Top value has been pushed onto top of stack.")
	} else {
		t.Error(HandleError(errCode, errStr))
	}

	//pushing a parameter value
	errCode, errStr = push.ExecCommand(strings.Split("-$rate 5", " "))
	if errCode == 0 {
		t.Log("Value Pushed onto stack.")
	} else {
		t.Error(HandleError(errCode, errStr))
	}

}

func TestSet(t *testing.T) {
	set := COMMAND_LIST["\\set"]

	//\Set without any args
	var b bytes.Buffer
	writetmp := bufio.NewWriter(&b)
	SetOutput(writetmp, true)

	errCode, errStr := set.ExecCommand([]string{})
	writetmp.Flush()
	if errCode == 0 {
		t.Log(b.String())
	} else {
		t.Error(HandleError(errCode, errStr))
	}

	//setting a parameter value
	errCode, errStr = set.ExecCommand(strings.Split("-$rate 9.5", " "))
	if errCode == 0 {
		t.Log("Value Set.")
	} else {
		t.Error(HandleError(errCode, errStr))
	}

	//setting a parameter without specifying a value.
	errCode, errStr = set.ExecCommand([]string{"-$rate"})
	if errCode != 0 {
		t.Log(HandleError(errCode, errStr))
	} else {
		t.Error("Set with one arg should return an error.")
	}
}

func popparam(param map[string]*Stack, isrestp bool, isnamep bool, t *testing.T) {
	errCode, errStr := Popparam_Helper(param, isrestp, isnamep)
	if errCode == 0 {
		t.Log("pop successful")
	} else {
		t.Error(HandleError(errCode, errStr))
	}
}

func TestPop_popParamHelper(t *testing.T) {
	// Case 1 : Is a rest api parameter
	popparam(QueryParam, true, false, t)
	//Case 2 : Is a named parameter
	popparam(NamedParam, true, true, t)
	//Case 3 : Is neither a rest api parameter, not a named parameter
	popparam(PreDefSV, false, false, t)
}

func TestPop(t *testing.T) {
	pop := COMMAND_LIST["\\pop"]

	//\Pop without any args
	errCode, errStr := pop.ExecCommand([]string{})
	if errCode == 0 {
		t.Log("Popped values from every stack.")
	} else {
		t.Error(HandleError(errCode, errStr))
	}

	//popping a specific value
	errCode, errStr = pop.ExecCommand([]string{"-$rate"})
	if errCode == 0 {
		t.Log("Value Popped.")
	} else {
		t.Error(HandleError(errCode, errStr))
	}

	//Test with too many args
	errCode, errStr = pop.ExecCommand([]string{"-$rate $val"})
	if errCode != 0 {
		t.Log(HandleError(errCode, errStr))
	} else {
		t.Error("Pop can take only 1 argument")
	}

}

func TestUnset(t *testing.T) {
	unset := COMMAND_LIST["\\unset"]

	//Test with too many args
	errCode, errStr := unset.ExecCommand([]string{"$rate -$val"})
	if errCode != 0 {
		t.Log(HandleError(errCode, errStr))
	} else {
		t.Error("Max number of args should be 1.")
	}

	//Test with too few args
	errCode, errStr = unset.ExecCommand([]string{})
	if errCode != 0 {
		t.Log(HandleError(errCode, errStr))
	} else {
		t.Error("Min number of args should be 1.")
	}

	//Unset a sample parameter
	errCode, errStr = unset.ExecCommand([]string{"-$rate"})
	if errCode == 0 {
		t.Log("Unset and deleted -$rate")
	} else {
		t.Error(HandleError(errCode, errStr))
	}

	errCode, errStr = unset.ExecCommand([]string{"-timeout"})
	if errCode == 0 {
		t.Log("Unset and deleted -timeout")
	} else {
		t.Error(HandleError(errCode, errStr))
	}

	set := COMMAND_LIST["\\set"]

	//\Set without any args
	var b bytes.Buffer
	writetmp := bufio.NewWriter(&b)
	SetOutput(writetmp, true)

	errCode, errStr = set.ExecCommand([]string{})
	writetmp.Flush()
	if errCode == 0 {
		t.Log("New set of values, without -timeout and -$rate\n", b.String())
	} else {
		t.Error(HandleError(errCode, errStr))
	}
}
