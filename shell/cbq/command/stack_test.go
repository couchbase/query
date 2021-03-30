//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package command

import (
	"testing"
)

/*
   Test the stack methods.
*/

func TestStackMethods(t *testing.T) {
	newStack := Stack_Helper()

	topval, errC, errS := newStack.Top()
	if errC != 0 {
		t.Log("Stack is empty")
	}

	newStack.Push(StrToVal("Value"))
	errC, errS = newStack.SetTop(StrToVal("9.5"))
	if errC == 0 {
		topval, errC, errS = newStack.Top()
		if errC == 0 {
			if topval.Actual() == 9.5 {
				t.Log("Top value is ", topval.Actual())
				t.Log("Length of stack is ", newStack.Len())
			} else {
				t.Errorf("Incorrect top value %v", topval.Actual())
			}
		} else {
			t.Error(HandleError(errC, errS))
		}
	} else {
		t.Error(HandleError(errC, errS))
	}

	v, errC, errS := newStack.Pop()
	if errC == 0 {
		if v.Actual() == 9.5 {
			t.Log("Popped value ", v.Actual())
		} else {
			t.Errorf("Incorrect value popped %v", v.Actual())
		}
	} else {
		t.Error(HandleError(errC, errS))
	}
}
