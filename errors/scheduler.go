//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package errors

import (
	"fmt"
)

func NewSchedulerError(what string, e error) Error {
	return &err{level: EXCEPTION, ICode: E_SCHEDULER, IKey: "scheduler.generic.error", ICause: e,
		InternalMsg:    fmt.Sprintf("The scheduler encountered an error in %v", what),
		InternalCaller: CallerN(1)}
}

func NewDuplicateTaskError(t string) Error {
	return &err{level: EXCEPTION, ICode: E_DUPLICATE_TASK, IKey: "scheduler.duplicate.error", ICause: fmt.Errorf("%v", t),
		InternalMsg:    fmt.Sprintf("Task already exists %v", t),
		InternalCaller: CallerN(1)}
}

func NewTaskRunningError(t string) Error {
	return &err{level: EXCEPTION, ICode: E_TASK_RUNNING, IKey: "scheduler.running.error", ICause: fmt.Errorf("%v", t),
		InternalMsg:    fmt.Sprintf("Task %v is currently executing and cannot be deleted", t),
		InternalCaller: CallerN(1)}
}

func NewTaskNotFoundError(t string) Error {
	return &err{level: EXCEPTION, ICode: E_TASK_NOT_FOUND, IKey: "scheduler.notfound.error", ICause: fmt.Errorf("%v", t),
		InternalMsg:    fmt.Sprintf("the task %v was not found", t),
		InternalCaller: CallerN(1)}
}

func NewTaskInvalidParameter(param string) Error {
	return &err{level: EXCEPTION, ICode: E_TASK_INVALID_PARAMETER, IKey: "scheduler.task.invalid_parameter",
		InternalMsg: fmt.Sprintf("Task parameter '%v' not provided.", param), InternalCaller: CallerN(1)}
}
