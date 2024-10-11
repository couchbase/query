//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package scheduler

import (
	"sync"
	"time"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/util"
)

const _LIMIT = 16384

type State string

const (
	SCHEDULED State = "scheduled" // Task is yet to execute
	DELETING  State = "deleting"  // Task in SCHEDULED state is in the process of being cancelled.
	ABORTING  State = "aborting"  // Task in RUNNING state is in the processing of being aborted.
	RUNNING   State = "running"   // Task is executing
	COMPLETED State = "completed" // Task has completed without any cancellations/ aborting
	CANCELLED State = "cancelled" // Task was cancelled before it executed/when in SCHEDULED state
	ABORTED   State = "aborted"   // Task was aborted while in execution/in RUNNING state
)

type CacheName int

const (
	ALL_TASKS_CACHE = CacheName(iota)
	SCHEDULED_TASKS_CACHE
	COMPLETED_TASKS_CACHE
)

type TaskFunc func(Context, interface{}) (interface{}, []errors.Error)

// A task function whose execution must be stopped when notified.
// When the function receives a signal on the channel, its execution must be stopped.
// When the task in the scheduler completes, the channel is closed.
type StoppableTaskFunc func(Context, interface{}, <-chan bool) (interface{}, []errors.Error)

type TaskEntry struct {
	sync.Mutex
	Class        string
	SubClass     string
	Name         string
	Id           string
	Exec         TaskFunc // Task execution function that cannot be stopped once its execution begins.
	Stop         TaskFunc // This function is called only when the task is in SCHEDULED state and deleted.
	PostTime     time.Time
	StartTime    time.Time
	EndTime      time.Time
	Delay        time.Duration
	State        State
	Results      interface{}
	Errors       []errors.Error
	QueryContext string
	Description  string

	timer      *time.Timer
	parameters interface{}
	context    Context

	isStoppable   bool              // If the execution function of the task can be stopped while in RUNNING state.
	stoppableExec StoppableTaskFunc // Task execution function that can be stopped even after its execution begins.
	stopChannel   chan bool         // Channel to signal to the stoppableExec function to stop its execution
}

type schedulerCache struct {
	scheduled *util.GenCache // Non LRU purging for scheduled tasks
	completed *util.GenCache // LRU purging for results
}

var scheduler = &schedulerCache{}

// init scheduler cache
func init() {
	scheduler.scheduled = util.NewGenCache(-1)
	scheduler.completed = util.NewGenCache(_LIMIT)
}

// configure scheduler cache

func SchedulerLimit() int {
	return scheduler.completed.Limit()
}

func SchedulerSetLimit(limit int) {
	scheduler.completed.SetLimit(limit)
}

// utilities for scheduler and system keyspaces
func CountTasks() int {
	return scheduler.scheduled.Size() + scheduler.completed.Size()
}

func NameTasks() []string {
	res := scheduler.scheduled.Names()
	comp := scheduler.completed.Names()
	return append(res, comp...)
}

func TasksForeach(nonBlocking func(string, *TaskEntry) bool, blocking func() bool, cacheName CacheName) {
	dummyF := func(id string, r interface{}) bool {
		return nonBlocking(id, r.(*TaskEntry))
	}

	all := cacheName == ALL_TASKS_CACHE

	if all || cacheName == SCHEDULED_TASKS_CACHE {
		scheduler.scheduled.ForEach(dummyF, blocking)
	}

	if all || cacheName == COMPLETED_TASKS_CACHE {
		scheduler.completed.ForEach(dummyF, blocking)
	}
}

func TaskDo(key string, f func(*TaskEntry)) {
	var process func(interface{}) = nil

	if f != nil {
		process = func(entry interface{}) {
			ce := entry.(*TaskEntry)
			f(ce)
		}
	}
	if scheduler.scheduled.Get(key, process) == nil {
		_ = scheduler.completed.Get(key, process)
	}

}

// scheduler primitives

// Schedule a task. This task's execution cannot be stopped once it begins
func ScheduleTask(name, class, subClass string, delay time.Duration, exec, stop TaskFunc,
	parms interface{}, description string, context Context) errors.Error {
	return scheduleTaskHelper(name, class, subClass, delay, false, exec, stop, nil, parms, description, context)
}

// Schedule a task. This task's execution can be stopped once it begins
// Its execution function will receive a signal via the channel when its execution is to be aborted.
func ScheduleStoppableTask(name, class, subClass string, delay time.Duration, stoppableExec StoppableTaskFunc, stop TaskFunc,
	parms interface{}, description string, context Context) errors.Error {
	return scheduleTaskHelper(name, class, subClass, delay, true, nil, stop, stoppableExec, parms, description, context)
}

func scheduleTaskHelper(name, class, subClass string, delay time.Duration, isStoppable bool, exec, stop TaskFunc,
	stoppableExec StoppableTaskFunc, parms interface{}, description string, context Context) errors.Error {

	id, err := util.UUIDV5(class+subClass, name)
	if err != nil {
		return errors.NewSchedulerError("uuid", err)
	}

	task := &TaskEntry{
		Name:         name,
		Class:        class,
		SubClass:     subClass,
		Delay:        delay,
		PostTime:     time.Now(),
		Stop:         stop,
		State:        SCHEDULED,
		Id:           id,
		QueryContext: context.QueryContext(),
		Description:  description,
		parameters:   parms,
		context:      context,
	}

	if isStoppable {
		if stoppableExec == nil {
			return errors.NewTaskInvalidParameter("execution function")
		}

		task.isStoppable = true
		task.stoppableExec = stoppableExec
	} else {
		if exec == nil {
			return errors.NewTaskInvalidParameter("execution function")
		}
		task.Exec = exec
	}

	// lose any old completed run, so to preserve key uniqueness
	scheduler.completed.Delete(id, nil)

	// add it to the cache
	added := true
	scheduler.scheduled.Add(task, id, func(ce interface{}) util.Operation {

		// yawser - already there
		added = false
		return util.IGNORE
	})

	if !added {
		return errors.NewDuplicateTaskError(task.Id)
	}

	// and schedule execution
	task.timer = time.AfterFunc(delay, func() {

		// first, lock, check and mark as running
		bailOut := false
		scheduler.scheduled.Get(task.Id, func(ce interface{}) {
			// Check and modify state under lock to prevent races with the task's delete requests
			task.Lock()
			if task.State != SCHEDULED {
				bailOut = true
				task.Unlock()
				return
			}
			task.State = RUNNING

			if task.isStoppable {
				task.stopChannel = make(chan bool, 1)
			}

			task.StartTime = time.Now()
			task.Unlock()
		})
		if bailOut {
			return
		}

		var res interface{}
		var errs errors.Errors

		if task.isStoppable {
			// execute
			res, errs = task.stoppableExec(task.context, task.parameters, task.stopChannel)
		} else {
			// execute
			res, errs = task.Exec(task.context, task.parameters)
		}

		// mark complete and remove from scheduled
		scheduler.scheduled.Delete(task.Id, func(ce interface{}) {

			// Check state, modify state and cleanup under lock. This is to prevent races with the task's delete requests
			task.Lock()

			// If the task's stoppable execution completed as the task was deleted while it was running, change the state to ABORTED
			if task.State == ABORTING {
				task.State = ABORTED
			} else {
				task.State = COMPLETED
			}

			if task.isStoppable {
				// Once execution completes close the channel
				close(task.stopChannel)
			}

			task.Results = res
			task.Errors = errs
			task.EndTime = time.Now()

			// now that we are done, ditch everything we don't need
			task.Exec = nil
			task.Stop = nil
			task.context = nil
			task.parameters = nil
			task.timer = nil
			task.stoppableExec = nil
			task.stopChannel = nil
			task.Unlock()
		})
		scheduler.completed.Add(task, task.Id, func(ce interface{}) util.Operation {

			// can't happen, but for completeness, ditch any previous run
			return util.REPLACE
		})

	})

	return nil
}

func DeleteTask(id string) errors.Error {
	var task *TaskEntry
	bailOut := false
	deleted := false
	running := false

	_ = scheduler.scheduled.Get(id, func(ce interface{}) {
		task = ce.(*TaskEntry)

		// Check and modify state under lock to prevent races with other delete requests or the execution routine
		task.Lock()

		// Since the state change is done under lock, only one delete request for a SCHEDULED or RUNNING task will succeed
		if task.State == SCHEDULED {
			task.State = DELETING
			task.timer.Stop()
		} else if task.isStoppable && task.State == RUNNING {
			// No need to stop task.timer because the timer would not have already fired for the task to be in RUNNING state
			task.State = ABORTING

			// Safe to send to the stopChannel
			// Because the channel would have already been created if the task is in RUNNING state
			// And the channel would not be nil or closed, as the channel cleanup is only done
			// only once the task's execution completes
			select {
			case task.stopChannel <- false:
			default:
			}
			running = true
		} else {
			bailOut = true
		}
		task.Unlock()
	})

	// If the task was in RUNNING state there is no need for further processing.
	// The publishing of produced content and the cleanup of the task will be done once the task exits from its execution.
	if running {
		return nil
	}

	// Return an error if the task is not configured to be stopped while in RUNNING state
	// busy, can't delete
	if bailOut {
		return errors.NewTaskRunningError(id)
	}

	// The below actions are to delete a task that was in SCHEDULED state only
	// No need to perform the cleanup with a lock:
	// * Only one delete request for a task will succeed
	// * Since the task would have never entered RUNNING state, the execution routine cannot interfere
	// cleanup and remove
	if task != nil {
		var res interface{}
		var errs []errors.Error

		if task.Stop != nil {
			res, errs = task.Stop(task.context, task.parameters)
		}
		scheduler.scheduled.Delete(id, nil)

		task.Exec = nil
		task.Stop = nil
		task.context = nil
		task.parameters = nil
		task.timer = nil
		task.stoppableExec = nil

		// No need to close or cleanup the stopChannel
		// stopChannel would not have even been created for a task that never entered RUNNING state

		// cleanup produced content, publish it
		if res != nil || errs != nil {
			task.State = CANCELLED
			task.Results = res
			task.Errors = errs
			scheduler.completed.Add(task, task.Id, func(ce interface{}) util.Operation {

				// can't happen, but for completeness, ditch any previous run
				return util.REPLACE
			})
		}
		return nil
	}

	// not in scheduled - maybe completed?
	scheduler.completed.Delete(id, func(ce interface{}) {
		deleted = true
	})

	if deleted {
		return nil
	}

	return errors.NewTaskNotFoundError(id)
}
