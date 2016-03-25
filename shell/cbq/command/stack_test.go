//  Copyright (c) 2015-2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
