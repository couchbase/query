//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package migration

// TODO much like functions/metakv this package is ns_server specific

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/couchbase/cbauth/metakv"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/logging"
)

const _MIGRATION_PATH = "/query/migration/"
const _MIGRATION_STATE = "/state"
const (
	_MIGRATED = "migrated"
	_ABORTED  = "aborted"
)

type waiters struct {
	lock     sync.Mutex
	cond     *sync.Cond
	released bool
	success  bool
}

type migrationDescriptor struct {
	Node  string    `json:"node"`
	State string    `json:"state"`
	When  time.Time `json:"when"`
}

var mapLock sync.Mutex
var waitersMap map[string]*waiters

func init() {
	waitersMap = make(map[string]*waiters)
	go metakv.RunObserveChildren(_MIGRATION_PATH, callback, make(chan struct{}))
}

func setState(state string) []byte {
	desc := &migrationDescriptor{distributed.RemoteAccess().WhoAmI(), state, time.Now()}
	out, _ := json.Marshal(desc)
	return out
}

func getState(val []byte) string {
	var in migrationDescriptor
	json.Unmarshal(val, &in)
	return in.State
}

func Complete(what string, success bool) {
	state := _MIGRATED
	if !success {
		state = _ABORTED
	}
	err := metakv.Set(_MIGRATION_PATH+what+_MIGRATION_STATE, setState(state), nil)
	if err != nil {
		logging.Warnf("%v migration: Cannot switch to completed (err %v) - please restart node",
			what, err)
	}

	// We rely on the metakv callback to wake up the waiters
}

// determine if migration ca be skipped
func IsComplete(what string) (bool, bool) {
	var state string
	val, _, err := metakv.Get(_MIGRATION_PATH + what + _MIGRATION_STATE)
	if err == nil {
		state = getState(val)
		if state == _MIGRATED {
			return true, true
		} else if state == _ABORTED {
			return true, false
		}
	}
	return false, false
}

// checking for migration to complete and waiting is it hasn't
func Await(what string) bool {
	val, _, err := metakv.Get(_MIGRATION_PATH + what + _MIGRATION_STATE)
	if err == nil {
		state := getState(val)
		if state == _MIGRATED {
			return true
		} else if state == _ABORTED {
			return false
		}
	}

	// no dice
	mapLock.Lock()
	w := waitersMap[what]
	if w != nil {
		mapLock.Unlock()
		w.cond.L.Lock()
		if w.released {
			w.cond.L.Unlock()
			return w.success
		}
		w.cond.Wait()

		w.cond.L.Unlock()
		return w.success
	}

	// add migration
	w = &waiters{}
	w.cond = sync.NewCond(&w.lock)
	w.cond.L.Lock()
	waitersMap[what] = w
	mapLock.Unlock()
	w.cond.Wait()

	// wait leaves the lock locked on exit
	w.cond.L.Unlock()
	return w.success
}

// migration callback
func callback(kve metakv.KVEntry) error {
	path := string(kve.Path)
	if !strings.HasPrefix(path, _MIGRATION_PATH) ||
		!strings.HasSuffix(path, _MIGRATION_STATE) {
		return nil
	}

	what := path[len(_MIGRATION_PATH):]
	what = what[:len(what)-len(_MIGRATION_STATE)]
	newState := getState(kve.Value)

	logging.Infof("%v migration: Metakv callback - migration state changed to %v", what, newState)

	// this is a good place to hook in a migration callback if we want
	// to offer the option of reacting to changing migration states

	success := false
	if newState == _MIGRATED {
		success = true
	} else if newState != _ABORTED {
		return nil
	}

	// call a separate go routine in case something happens in the call (wait on mutex/condition)
	go callback1(what, success)
	return nil
}

func callback1(what string, success bool) {
	mapLock.Lock()
	w := waitersMap[what]
	if w != nil {
		mapLock.Unlock()
		logging.Infof("%v migration: Releasing waiters", what)
		w.cond.L.Lock()
		w.released = true
		w.success = success
		w.cond.L.Unlock()
		w.cond.Broadcast()
	} else {

		// no waiters found, but record state for posterity
		// just in case somebody tries to wait on a migrated topic
		w = &waiters{}
		w.cond = sync.NewCond(&w.lock)
		w.cond.L.Lock()
		waitersMap[what] = w
		mapLock.Unlock()
		logging.Infof("%v migration: Complete with no waiters", what)
	}
}
