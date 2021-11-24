//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package command

import (
	"strconv"
	"testing"

	"github.com/couchbase/query/value"
)

/*
   Test the common methods
*/

// The Resolve tests test the Resolve, PushValue_Helper
// and PopValue_Helper methods

func TestResolve_alias(t *testing.T) {

	AliasCommand["tempcmd"] = "select 1"
	val, errCode, errStr := Resolve("\\\\tempcmd")
	if val.Actual() == AliasCommand["tempcmd"] {
		t.Logf("Value of Alias tempcmd is : %v", val.Actual())
	} else {
		t.Errorf("Error getting value : %v", HandleError(errCode, errStr))
	}

}

func TestResolve_queryp(t *testing.T) {

	//Test for PushValue_Helper and Resolve for QueryParam.
	errCode, errStr := PushValue_Helper(false, QueryParam, "timeout", "10ms")
	if errCode != 0 {
		t.Errorf("%s", HandleError(errCode, errStr))
	}

	val, errCode, errStr := Resolve("-timeout")
	v, _, _ := QueryParam["timeout"].Top()
	if val == v {
		t.Logf("Value of Query Parameter timeout is : %v", val.Actual())
	} else {
		t.Errorf("Error getting value : %v", HandleError(errCode, errStr))
	}

	errCode, errStr = PopValue_Helper(true, QueryParam, "timeout")
	if errCode != 0 {
		t.Errorf("Error unsetting parameter : %s", HandleError(errCode, errStr))
	}

}

func TestResolve_namedp(t *testing.T) {
	//Test for PushValue_Helper and Resolve for NamedParam.
	errCode, errStr := PushValue_Helper(false, NamedParam, "rate", "9.5")
	if errCode != 0 {
		t.Errorf("%s", HandleError(errCode, errStr))
	}

	val, errCode, errStr := Resolve("-$rate")
	v, _, _ := NamedParam["rate"].Top()
	if val == v {
		t.Logf("Value of Named Parameter rate is : %v", val.Actual())
	} else {
		t.Errorf("Error getting value : %v", HandleError(errCode, errStr))
	}

	errCode, errStr = PopValue_Helper(true, NamedParam, "rate")
	if errCode != 0 {
		t.Errorf("Error unsetting parameter : %s", HandleError(errCode, errStr))
	}
}

func TestResolve_userdefp(t *testing.T) {

	//Test for PushValue_Helper and Resolve for UserDefParam.
	errCode, errStr := PushValue_Helper(false, UserDefSV, "temp", "5")
	if errCode != 0 {
		t.Errorf("%s", HandleError(errCode, errStr))
	}

	val, errCode, errStr := Resolve("$temp")
	v, _, _ := UserDefSV["temp"].Top()

	if val == v {
		t.Logf("Value of User Defined Parameter temp is : %v", val.Actual())
	} else {
		t.Errorf("Error getting value : %v", HandleError(errCode, errStr))
	}

	errCode, errStr = PopValue_Helper(true, UserDefSV, "temp")
	if errCode != 0 {
		t.Errorf("Error unsetting parameter : %s", HandleError(errCode, errStr))
	}

}

func TestResolve_predefp(t *testing.T) {

	val, errCode, errStr := Resolve("histfile")
	if val.Actual() == ".cbq_history" {
		t.Logf("Value of histfile is : %v", val.Actual())
	} else {
		t.Errorf("Error getting value : %v", HandleError(errCode, errStr))
	}

	//Test Pushing values to the stack.
	errCode, errStr = PushValue_Helper(false, PreDefSV, "histfile", "newhistory")
	if errCode != 0 {
		t.Errorf("%s", HandleError(errCode, errStr))
	}

	errCode, errStr = PushValue_Helper(false, PreDefSV, "histfile", "history2")
	if errCode != 0 {
		t.Errorf("%s", HandleError(errCode, errStr))
	}

	v, _, _ := PreDefSV["histfile"].Top()
	if v.Actual() == "history2" {
		t.Logf("History file has been modified to : %v", v.Actual())
	} else {
		t.Error("Error pushing values onto histfile stack")
	}

	//Test popping the values one by one
	errCode, errStr = PopValue_Helper(false, PreDefSV, "histfile")
	if errCode != 0 {
		t.Errorf("Error popping parameter : %s", HandleError(errCode, errStr))
	} else {
		v, _, _ = PreDefSV["histfile"].Top()
		if v.Actual() == "newhistory" {
			t.Logf("History file has been modified to : %v", v.Actual())
		} else {
			t.Error("Error popping values from histfile stack")
		}
	}

	errCode, errStr = PopValue_Helper(false, PreDefSV, "histfile")
	if errCode != 0 {
		t.Errorf("Error popping parameter : %s", HandleError(errCode, errStr))
	} else {
		v, _, _ = PreDefSV["histfile"].Top()
		if v.Actual() == ".cbq_history" {
			t.Logf("History file has been modified to : %v", v.Actual())
		} else {
			t.Error("Error popping values from histfile stack")
		}
	}

}

func TestResolve_dummytest(t *testing.T) {
	val, errCode, errStr := Resolve("dummy val")
	if val.Actual() == "dummy val" {
		t.Logf("Value returned is : %v", val.Actual())
	} else {
		t.Errorf("Error getting value : %v", HandleError(errCode, errStr))
	}

}

func compareValue(ipval value.Value, valtype value.Type, t *testing.T) {
	if ipval.Type() == valtype {
		t.Logf("Input %v is a %v value.", ValToStr(ipval), ipval.Type())
	} else {
		t.Errorf("Input %v is a %v value, and not %v", ValToStr(ipval), ipval.Type(), valtype)
	}

}

func TestStrToVal_toStr(t *testing.T) {

	var val value.Value

	//Test types
	val = StrToVal("String")
	compareValue(val, value.STRING, t)

	val = StrToVal(strconv.Itoa(2))
	compareValue(val, value.NUMBER, t)

	val = StrToVal(string([]byte(`null`)))
	compareValue(val, value.NULL, t)

	val = StrToVal("[1, 2]")
	compareValue(val, value.ARRAY, t)

	val = StrToVal("{\"a\" : 1, \"b\":2}")
	compareValue(val, value.OBJECT, t)

}

func TestToCreds(t *testing.T) {
	//Case 1 : Administrator:password
	c, errCode, errStr := ToCreds("Administrator:password")
	if errCode == 0 {
		t.Log("Correct credentials : ", c)
	} else {
		t.Error(HandleError(errCode, errStr))
	}
	//Case 2 : :password
	c, errCode, errStr = ToCreds(":password")
	if errCode != 0 {
		t.Log("Error Case tested : InCorrect credentials : ", ":password")
		t.Log(HandleError(errCode, errStr))
	} else {
		t.Error("Username needed. Test should not pass.")
	}

	//Case 3 : User:
	c, errCode, errStr = ToCreds("User:")
	if errCode == 0 {
		t.Log("Empty password allowed : ", c)
	} else {
		t.Error(HandleError(errCode, errStr))
	}

	//Case 4 : U: (space)
	c, errCode, errStr = ToCreds("U: ")
	if errCode == 0 {
		t.Log("Correct credentials : ", c)
	} else {
		t.Error(HandleError(errCode, errStr))
	}
}
