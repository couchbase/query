//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package prepareds

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"math"
	"sync"
	"time"

	atomic "github.com/couchbase/go-couchbase/platform"
	json "github.com/couchbase/go_json"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/parser/n1ql"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/planner"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

// prepared statements cache retrieval options
const (
	OPT_TRACK  = 1 << iota // track statement in cache
	OPT_REMOTE             // check with remote node, if available
	OPT_VERIFY             // verify that the plan is still valid
)

type preparedCache struct {
	cache *util.GenCache
}

type CacheEntry struct {
	Prepared       *plan.Prepared
	LastUse        time.Time
	Uses           int32
	ServiceTime    atomic.AlignedUint64
	RequestTime    atomic.AlignedUint64
	MinServiceTime atomic.AlignedUint64
	MinRequestTime atomic.AlignedUint64
	MaxServiceTime atomic.AlignedUint64
	MaxRequestTime atomic.AlignedUint64
	// FIXME add moving averages, latency
	// This requires the use of metrics

	sync.Mutex // for concurrent checking
	populated  bool
}

var prepareds = &preparedCache{}
var store datastore.Datastore
var systemstore datastore.Datastore
var namespace string

// init prepareds cache

func PreparedsInit(limit int) {
	prepareds.cache = util.NewGenCache(limit)
}

func PreparedsReprepareInit(ds, sy datastore.Datastore, ns string) {
	store = ds
	systemstore = sy
	namespace = ns
}

// configure prepareds cache

func PreparedsLimit() int {
	return prepareds.cache.Limit()
}

func PreparedsSetLimit(limit int) {
	prepareds.cache.SetLimit(limit)
}

func (this *preparedCache) get(name value.Value, track bool) *CacheEntry {
	var cv interface{}

	if name.Type() != value.STRING || !name.Truth() {
		return nil
	}

	n := name.Actual().(string)
	if track {
		cv = prepareds.cache.Use(n, nil)
	} else {
		cv = prepareds.cache.Get(n, nil)
	}
	rv, ok := cv.(*CacheEntry)
	if ok {
		if track {
			atomic.AddInt32(&rv.Uses, 1)

			// this is not exactly accurate, but since the MRU queue is
			// managed properly, we'd rather be inaccurate and make the
			// change outside of the lock than take a performance hit
			rv.LastUse = time.Now()
		}
		return rv
	}
	return nil
}

func (this *preparedCache) add(prepared *plan.Prepared, populated bool, process func(*CacheEntry) bool) {

	// prepare a new entry, if statement does not exist
	ce := &CacheEntry{
		Prepared:       prepared,
		MinServiceTime: math.MaxUint64,
		MinRequestTime: math.MaxUint64,
		populated:      populated,
	}
	prepareds.cache.Add(ce, prepared.Name(), func(entry interface{}) util.Operation {
		var op util.Operation = util.AMEND
		var cont bool = true

		// check existing entry, amend if all good, ignore otherwise
		oldEntry := entry.(*CacheEntry)
		if process != nil {
			cont = process(oldEntry)
		}
		if cont {
			oldEntry.Prepared = prepared
			oldEntry.populated = false
		} else {
			op = util.IGNORE
		}
		return op
	})
}

func CountPrepareds() int {
	return prepareds.cache.Size()
}

func NamePrepareds() []string {
	return prepareds.cache.Names()
}

func PreparedsForeach(nonBlocking func(string, *CacheEntry) bool,
	blocking func() bool) {
	dummyF := func(id string, r interface{}) bool {
		return nonBlocking(id, r.(*CacheEntry))
	}
	prepareds.cache.ForEach(dummyF, blocking)
}

func PreparedDo(name string, f func(*CacheEntry)) {
	var process func(interface{}) = nil

	if f != nil {
		process = func(entry interface{}) {
			ce := entry.(*CacheEntry)
			f(ce)
		}
	}
	_ = prepareds.cache.Get(name, process)
}

func AddPrepared(prepared *plan.Prepared) errors.Error {
	added := true

	prepareds.add(prepared, false, func(ce *CacheEntry) bool {
		if ce.Prepared.Text() != prepared.Text() {
			added = false
		}
		return added
	})
	if !added {
		return errors.NewPreparedNameError(
			fmt.Sprintf("duplicate name: %s", prepared.Name()))
	} else {
		distributePrepared(prepared.Name(), prepared.EncodedPlan())
		return nil
	}
}

func DeletePrepared(name string) errors.Error {
	if prepareds.cache.Delete(name, nil) {
		return nil
	}
	return errors.NewNoSuchPreparedError(name)
}

func GetPrepared(prepared_stmt value.Value, options uint32, phaseTime *time.Duration) (*plan.Prepared, errors.Error) {
	var err errors.Error

	track := (options & OPT_TRACK) != 0
	remote := (options & OPT_REMOTE) != 0
	verify := (options & OPT_VERIFY) != 0
	switch prepared_stmt.Type() {
	case value.STRING:
		var prepared *plan.Prepared

		host, name := distributed.RemoteAccess().SplitKey(prepared_stmt.Actual().(string))
		ce := prepareds.get(value.NewValue(name), track)
		if ce != nil {
			prepared = ce.Prepared
		}
		if prepared == nil && remote && host != "" && host != distributed.RemoteAccess().WhoAmI() {
			distributed.RemoteAccess().GetRemoteDoc(host, name, "prepareds", "GET",
				func(doc map[string]interface{}) {
					encoded_plan, ok := doc["encoded_plan"].(string)
					if ok {
						prepared, err = DecodePrepared(name, encoded_plan, false, false, phaseTime)
					}
				},
				func(warn errors.Error) {
				}, distributed.NO_CREDS, "")
		} else if prepared != nil && verify {
			var good bool

			// things have already been set up
			// take the short way home
			if ce.populated {

				// note that it's fine to check and repopulate without a lock
				// since the structure of the plan tree won't change, nor the
				// keyspaces and indexers, the worse that is going to happen is
				// two requests amending the same counter
				good = prepared.MetadataCheck()

				// counters have changed. fetch new values
				if !good {
					good = prepared.Verify()
				}
			} else {

				// we have to proceed under a lock to avoid multiple
				// requests populating metadata counters at the same time
				ce.Lock()

				// check again, somebody might have done it in the interim
				if ce.populated {
					good = true
				} else {

					// nada - have to go the long way
					good = prepared.Verify()
					if good {
						ce.populated = true
					}
				}
				ce.Unlock()
			}

			// after all this, it did not work out!
			// here we are going to accept multiple requests creating a new
			// plan concurrently as we don't have a good way to serialize
			// without blocking the whole prepared cacheline
			// locking will occur at adding time: both requests will insert,
			// the last wins
			if !good {
				prepared, err = reprepare(prepared, phaseTime)
				if err == nil {
					err = AddPrepared(prepared)
				}
			}
		}
		if err != nil {
			return nil, err
		}
		if prepared == nil {
			return nil, errors.NewNoSuchPreparedError(name)
		}
		return prepared, nil
	case value.OBJECT:
		name_value, has_name := prepared_stmt.Field("name")
		if has_name {
			if ce := prepareds.get(name_value, track); ce != nil {
				return ce.Prepared, nil
			}
		}
		prepared_bytes, err := prepared_stmt.MarshalJSON()
		if err != nil {
			return nil, errors.NewUnrecognizedPreparedError(err)
		}
		return unmarshalPrepared(prepared_bytes, phaseTime)
	default:
		return nil, errors.NewUnrecognizedPreparedError(fmt.Errorf("Invalid prepared stmt %v", prepared_stmt))
	}
}

func RecordPreparedMetrics(prepared *plan.Prepared, requestTime, serviceTime time.Duration) {
	if prepared == nil {
		return
	}
	name := prepared.Name()
	if name == "" {
		return
	}

	// cache get had already moved this entry to the top of the LRU
	// no need to do it again
	_ = prepareds.cache.Get(name, func(entry interface{}) {
		ce := entry.(*CacheEntry)
		atomic.AddUint64(&ce.ServiceTime, uint64(serviceTime))
		util.TestAndSetUint64(&ce.MinServiceTime, uint64(serviceTime),
			func(old, new uint64) bool { return old > new }, 0)
		util.TestAndSetUint64(&ce.MaxServiceTime, uint64(serviceTime),
			func(old, new uint64) bool { return old < new }, 0)
		atomic.AddUint64(&ce.RequestTime, uint64(requestTime))
		util.TestAndSetUint64(&ce.MinRequestTime, uint64(requestTime),
			func(old, new uint64) bool { return old > new }, 0)
		util.TestAndSetUint64(&ce.MaxRequestTime, uint64(requestTime),
			func(old, new uint64) bool { return old < new }, 0)
	})
}

func DecodePrepared(prepared_name string, prepared_stmt string, track bool, distribute bool, phaseTime *time.Duration) (*plan.Prepared, errors.Error) {
	added := true

	decoded, err := base64.StdEncoding.DecodeString(prepared_stmt)
	if err != nil {
		return nil, errors.NewPreparedDecodingError(err)
	}
	var buf bytes.Buffer
	buf.Write(decoded)
	reader, err := gzip.NewReader(&buf)
	if err != nil {
		return nil, errors.NewPreparedDecodingError(err)
	}
	prepared_bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, errors.NewPreparedDecodingError(err)
	}
	prepared, err := unmarshalPrepared(prepared_bytes, phaseTime)
	if err != nil {
		return nil, errors.NewPreparedDecodingError(err)
	}

	prepared.SetEncodedPlan(prepared_stmt)

	// MB-19509 we now have to check that the encoded plan matches
	// the prepared statement named in the rest API
	_, prepared_key := distributed.RemoteAccess().SplitKey(prepared_name)
	if prepared.Name() != "" && prepared_name != "" &&
		prepared_key != prepared.Name() {
		return nil, errors.NewEncodingNameMismatchError(prepared_name)
	}

	if prepared.Name() == "" {
		return prepared, nil
	}

	// we don't trust anything strangers give us.
	// check the plan and populate metadata counters
	// reprepare if no good
	good := prepared.Verify()
	if !good {
		newPrepared, prepErr := reprepare(prepared, phaseTime)
		if prepErr == nil {
			prepared = newPrepared
		} else {
			return nil, prepErr
		}
	}

	when := time.Now()
	prepareds.add(prepared, good,
		func(oldEntry *CacheEntry) bool {

			// MB-19509: if the entry exists already, the new plan must
			// also be for the same statement as we have in the cache
			if oldEntry.Prepared != prepared &&
				oldEntry.Prepared.Text() != prepared.Text() {
				added = false
				return added
			}

			// track the entry if required, whether we amend the plan or
			// not, as at the end of the statement we will record the
			// metrics anyway
			if track {
				atomic.AddInt32(&oldEntry.Uses, 1)
				oldEntry.LastUse = when
			}

			// MB-19659: this is where we decide plan conflict.
			// the current behaviour is to always use the new plan
			// and amend the cache
			// This is still to be finalized
			return added
		})

	if added {
		if distribute {
			distributePrepared(prepared.Name(), prepared_stmt)
		}
		return prepared, nil
	} else {
		return nil, errors.NewPreparedEncodingMismatchError(prepared_name)
	}
}

func unmarshalPrepared(bytes []byte, phaseTime *time.Duration) (*plan.Prepared, errors.Error) {
	prepared := plan.NewPrepared(nil, nil)
	err := prepared.UnmarshalJSON(bytes)
	if err != nil {

		// if we failed to unmarshall, we find  the statement
		// and try preparing from scratch
		text, err1 := json.FirstFind(bytes, "text")
		if text != nil && err1 == nil {
			var stmt string

			err1 = json.Unmarshal(text, &stmt)
			if err1 == nil {
				prepared.SetText(stmt)
				pl, _ := reprepare(prepared, phaseTime)
				if pl != nil {
					return pl, nil
				}
			}
		}
		return nil, errors.NewUnrecognizedPreparedError(fmt.Errorf("JSON unmarshalling error: %v", err))
	}
	return prepared, nil
}

func distributePrepared(name, plan string) {
	go distributed.RemoteAccess().DoRemoteOps([]string{}, "prepareds", "PUT", name, plan,
		func(warn errors.Error) {
			if warn != nil {
				logging.Infof("failed to distribute statement <ud>%v</ud>: %v", name, warn)
			}
		}, distributed.NO_CREDS, "")
}

func reprepare(prepared *plan.Prepared, phaseTime *time.Duration) (*plan.Prepared, errors.Error) {
	parse := time.Now()
	stmt, err := n1ql.ParseStatement(prepared.Text())
	if phaseTime != nil {
		*phaseTime += time.Since(parse)
	}
	if err != nil {

		// this should never happen: the statement parsed to start with
		return nil, errors.NewReprepareError(err)
	}
	prep := time.Now()
	pl, err := planner.BuildPrepared(stmt.(*algebra.Prepare).Statement(), store, systemstore, namespace, false,

		// building prepared statements should not depend on args
		nil, nil, prepared.IndexApiVersion(), prepared.FeatureControls())
	if phaseTime != nil {
		*phaseTime += time.Since(prep)
	}
	if err != nil {
		return nil, errors.NewReprepareError(err)
	}

	pl.SetName(prepared.Name())
	pl.SetText(prepared.Text())
	pl.SetType(prepared.Type())
	pl.SetIndexApiVersion(prepared.IndexApiVersion())
	pl.SetFeatureControls(prepared.FeatureControls())

	json_bytes, err := pl.MarshalJSON()
	if err != nil {
		return nil, errors.NewReprepareError(err)
	}
	pl.BuildEncodedPlan(json_bytes)
	return pl, nil
}
