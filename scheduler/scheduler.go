//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package scheduler

import (
	"time"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/util"
)

const _LIMIT = 16384

type State string

const (
	SCHEDULED State = "scheduled"
	DELETING  State = "deleting"
	RUNNING   State = "running"
	COMPLETED State = "completed"
	CANCELLED State = "cancelled"
)

type TaskFunc func(Context, interface{}) (interface{}, []errors.Error)

type TaskEntry struct {
	Class        string
	SubClass     string
	Name         string
	Id           string
	Exec         TaskFunc
	Stop         TaskFunc
	PostTime     time.Time
	StartTime    time.Time
	EndTime      time.Time
	Delay        time.Duration
	State        State
	Results      interface{}
	Errors       []errors.Error
	QueryContext string

	timer      *time.Timer
	parameters interface{}
	context    Context
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

func TasksForeach(nonBlocking func(string, *TaskEntry) bool,
	blocking func() bool) {
	dummyF := func(id string, r interface{}) bool {
		return nonBlocking(id, r.(*TaskEntry))
	}
	scheduler.scheduled.ForEach(dummyF, blocking)
	scheduler.completed.ForEach(dummyF, blocking)
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
func ScheduleTask(name, class, subClass string, delay time.Duration, exec, stop TaskFunc, parms interface{}, context Context) errors.Error {

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
		Exec:         exec,
		Stop:         stop,
		State:        SCHEDULED,
		Id:           id,
		QueryContext: context.QueryContext(),
		parameters:   parms,
		context:      context,
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
			if task.State != SCHEDULED {
				bailOut = true
				return
			}
			task.State = RUNNING
			task.StartTime = time.Now()
		})
		if bailOut {
			return
		}

		// execute
		res, errs := task.Exec(task.context, task.parameters)

		// mark complete and remove from scheduled
		scheduler.scheduled.Delete(task.Id, func(ce interface{}) {
			task.State = COMPLETED
			task.Results = res
			task.Errors = errs
			task.EndTime = time.Now()

			// now that we are done, ditch everything we don't need
			task.Exec = nil
			task.Stop = nil
			task.context = nil
			task.parameters = nil
			task.timer = nil
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

	_ = scheduler.scheduled.Get(id, func(ce interface{}) {
		task = ce.(*TaskEntry)
		if task.State == SCHEDULED {
			task.State = DELETING
			task.timer.Stop()
		} else {
			bailOut = true
		}
	})

	// busy, can't delete
	if bailOut {
		return errors.NewTaskRunningError(id)
	}

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
