//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package command

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

/* Helper function to create a stack. */
func Stack_Helper() *Stack {
	r := make(Stack, 0)
	return &r
}

/* Stack methods to be used for session parameters */
type Stack []value.Value

/* Push input value val onto the stack */
func (stack *Stack) Push(val value.Value) {
	*stack = append(*stack, val)
}

/*
Return the top element in the stack. If the stack

	is empty then return ZERO_VALUE.
*/
func (stack *Stack) Top() (val value.Value, err_code errors.ErrorCode, err_str string) {
	if stack.Len() == 0 {
		val = nil
		err_code = errors.E_SHELL_STACK_EMPTY
		err_str = ""
	} else {
		x := stack.Len() - 1
		val = (*stack)[x]
		err_code = 0
		err_str = ""
	}

	return
}

func (stack *Stack) SetTop(v value.Value) (err_code errors.ErrorCode, err_str string) {
	if stack.Len() == 0 {
		err_code = errors.E_SHELL_STACK_EMPTY
		err_str = ""
	} else {
		x := stack.Len() - 1
		(*stack)[x] = v
		err_code = 0
		err_str = ""
	}
	return
}

/*
Delete the top element in the stack. If the stack

	is empty then print err stack empty
*/
func (stack *Stack) Pop() (val value.Value, err_code errors.ErrorCode, err_str string) {
	if stack.Len() == 0 {
		val = nil
		err_code = errors.E_SHELL_STACK_EMPTY
		err_str = ""
	} else {
		x := stack.Len() - 1
		val = (*stack)[x]
		*stack = (*stack)[:x]
		err_code = 0
		err_str = ""
	}

	return
}

func (stack *Stack) Len() int {
	return len(*stack)
}
