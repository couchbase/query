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
)

type waiters struct {
	lock     sync.Mutex
	cond     *sync.Cond
	released bool
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

func state(val []byte) string {
	var in migrationDescriptor
	json.Unmarshal(val, &in)
	return in.State
}

func Complete(what string) {
	err := metakv.Set(_MIGRATION_PATH+what+_MIGRATION_STATE, setState(_MIGRATED), nil)
	if err != nil {
		logging.Warnf("%v migration: Cannot switch to completed (err %v) - please restart node",
			what, err)
	}

	// We rely on the metakv callback to wake up the waiters
}

// determine if migration ca be skipped
func IsComplete(what string) bool {
	val, _, err := metakv.Get(_MIGRATION_PATH + what + _MIGRATION_STATE)
	return err == nil && state(val) == _MIGRATED
}

// checking for migration to complete and waiting is it hasn't
func Await(what string) {
	val, _, err := metakv.Get(_MIGRATION_PATH + what + _MIGRATION_STATE)
	if err == nil && state(val) == _MIGRATED {
		return
	}

	// no dice
	mapLock.Lock()
	w := waitersMap[what]
	if w != nil {
		mapLock.Unlock()
		w.cond.L.Lock()
		if w.released {
			w.cond.L.Unlock()
			return
		}
		w.cond.Wait()

		w.cond.L.Unlock()
		return
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
	newState := state(kve.Value)

	logging.Infof("%v migration: Metakv callback - migration state changed to %v", what, newState)

	// this is a good place to hook in a migration callback if we want
	// to offer the option of reacting to changing migration states

	if state(kve.Value) != _MIGRATED {
		return nil
	}

	// call a separate go routine in case something happens in the call (wait on mutex/condition)
	go callback1(what)
	return nil
}

func callback1(what string) {
	mapLock.Lock()
	w := waitersMap[what]
	if w != nil {
		mapLock.Unlock()
		logging.Infof("%v migration: Releasing waiters", what)
		w.cond.L.Lock()
		w.released = true
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
