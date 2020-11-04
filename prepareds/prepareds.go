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
	"math/rand"
	"strconv"
	"strings"
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
	"github.com/couchbase/query/rewrite"
	"github.com/couchbase/query/semantics"
	"github.com/couchbase/query/util"
)

// empty plan for backwards compatibility with older SDKs, engines
// decodes to ""
const EmptyPlan = "H4sIAAAAAAAA/wEAAP//AAAAAAAAAAA="

// prepared statements cache retrieval options
const (
	OPT_TRACK     = 1 << iota // track statement in cache
	OPT_REMOTE                // check with remote node, if available
	OPT_VERIFY                // verify that the plan is still valid
	OPT_METACHECK             // metadata check only
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
var predefinedPrepareStatements map[string]string

// init prepareds cache
func PreparedsInit(limit int) {
	prepareds.cache = util.NewGenCache(limit)
	planner.SetPlanCache(prepareds)
	predefinedPrepareStatements = map[string]string{
		"__get":    "PREPARE __get FROM SELECT META(d).id, META(d).cas, TO_STR(META(d).cas) AS scas, META(d).txnMeta, d AS doc FROM $1 AS d USE KEYS $2;",
		"__insert": "PREPARE __insert FROM INSERT INTO $1 AS d VALUES ($2, $3, $4) RETURNING TO_STR(META(d).cas) AS scas;",
		"__upsert": "PREPARE __upsert FROM UPSERT INTO $1 AS d VALUES ($2, $3, $4) RETURNING TO_STR(META(d).cas) AS scas;",
		"__update": "PREPARE __update FROM UPDATE $1 AS d USE KEYS $2 SET d = $3, META(d).expiration = $4.expiration RETURNING META(d).id, META(d).cas, TO_STR(META(d).cas) AS scas, META(d).txnMeta, d AS doc;",
		"__delete": "PREPARE __delete FROM DELETE FROM $1 AS d USE KEYS $2;",
	}
}

// initialize the cache from a different node
func PreparedsRemotePrime() {

	// wait for the node to be part of a cluster
	thisHost := distributed.RemoteAccess().WhoAmI()
	for distributed.RemoteAccess().Starting() && thisHost == "" {
		time.Sleep(time.Second)
		thisHost = distributed.RemoteAccess().WhoAmI()
	}

	if distributed.RemoteAccess().StandAlone() {
		return
	}

	nodeNames := distributed.RemoteAccess().GetNodeNames()
	left := len(nodeNames)
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	// try each host until we get something
	for left > 0 {
		count := 0

		// choose a random host
		n := r1.Intn(left)
		host := nodeNames[n]
		if n == (left - 1) {
			nodeNames = nodeNames[:n]
		} else {
			nodeNames = append(nodeNames[:n], nodeNames[n+1:]...)
		}
		left--

		// but not us
		if host == thisHost {
			continue
		}

		// get the keys
		distributed.RemoteAccess().GetRemoteKeys([]string{host}, "prepareds",
			func(id string) bool {
				_, name := distributed.RemoteAccess().SplitKey(id)

				// and for each key get the prepared and add it
				distributed.RemoteAccess().GetRemoteDoc(host, name, "prepareds", "GET",
					func(doc map[string]interface{}) {
						encoded_plan, ok := doc["encoded_plan"].(string)
						if ok {
							_, err := DecodePrepared(name, encoded_plan)
							if err == nil {
								count++
							}
						}
					},
					func(warn errors.Error) {
					}, distributed.NO_CREDS, "")
				return true
			}, nil)

		// we found stuff, that's good enough
		if count > 0 {
			break
		}
	}
}

// preparedCache implements planner.PlanCache
func (this *preparedCache) IsPredefinedPrepareName(name string) bool {
	_, ok := predefinedPrepareStatements[name]
	return ok
}

func (this *preparedCache) GetText(text string, offset int) string {

	// in order to get the force option to not to mistake the
	// statement as different and refuse to replace the plan
	// we need to remove it from the statement
	// this we do for backwards compatibility - ideally we should just
	// store and compare the prepared text, since with the current
	// system, variations in the actual prepared statement (eg AS vs FROM, or
	// one extra space, specifying the name of an already prepared anonymous
	// statment, use of string vs identifier for the statement name...)s
	// makes the text verification fails, while it should't
	prepare := text[:offset]
	i := strings.Index(strings.ToUpper(prepare), "FORCE")
	if i < 0 {
		return text
	}
	if i+6 >= offset {
		return prepare[:i] + text[offset:]
	} else {
		return prepare[:i] + prepare[i+6:] + text[offset:]
	}
}

const _REALM_SIZE = 256

func (this *preparedCache) GetName(text, namespace string, context *planner.PrepareContext) (string, errors.Error) {

	// different feature controls and index API version generate different names
	// so that the same statement prepared differently can coexist
	// prepare options are skipped so that prepare and prepare force yield the same
	// name
	// name is independent of query context as well, so as to make anonynoums and
	// named prepared statements behaviour consistent

	var buf [_REALM_SIZE]byte
	realm := buf[0:0:_REALM_SIZE]
	realm = strconv.AppendInt(realm, int64(context.IndexApiVersion()), 16)
	realm = append(realm, '_')
	realm = strconv.AppendInt(realm, int64(context.FeatureControls()), 16)
	realm = append(realm, '_')
	realm = strconv.AppendBool(realm, context.UseFts())
	realm = append(realm, '_')
	realm = strconv.AppendBool(realm, context.UseCBO())
	realm = append(realm, '_')
	realm = append(realm, namespace...)
	name, err := util.UUIDV5(string(realm), text)
	if err != nil {
		return "", errors.NewPreparedNameError(err.Error())
	}
	return name, nil
}

func encodeName(name string, queryContext string) string {
	if queryContext == "" {
		return name
	}
	return name + "(" + queryContext + ")"
}

func (this *preparedCache) GetPlan(name, text, namespace string, context *planner.PrepareContext) (*plan.Prepared, errors.Error) {
	prep, err := getPrepared(name, context.QueryContext(), context.DeltaKeyspaces(), OPT_VERIFY, nil)
	if err != nil {
		if err.Code() == errors.NO_SUCH_PREPARED {
			return nil, nil
		}
		return nil, err
	}
	if prep.IndexApiVersion() != context.IndexApiVersion() || prep.FeatureControls() != context.FeatureControls() ||
		prep.Namespace() != namespace || prep.QueryContext() != context.QueryContext() || prep.Text() != text ||
		prep.UseFts() != context.UseFts() || prep.UseCBO() != context.UseCBO() {
		return nil, nil
	}
	return prep, nil
}

func PreparedsReprepareInit(ds, sy datastore.Datastore) {
	store = ds
	systemstore = sy
}

// configure prepareds cache

func PreparedsLimit() int {
	return prepareds.cache.Limit()
}

func PreparedsSetLimit(limit int) {
	prepareds.cache.SetLimit(limit)
}

func (this *preparedCache) get(fullName string, track bool) *CacheEntry {
	var cv interface{}

	if track {
		cv = prepareds.cache.Use(fullName, nil)
	} else {
		cv = prepareds.cache.Get(fullName, nil)
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

func (this *preparedCache) add(prepared *plan.Prepared, populated bool, track bool, process func(*CacheEntry) bool) {

	// prepare a new entry, if statement does not exist
	ce := &CacheEntry{
		Prepared:       prepared,
		MinServiceTime: math.MaxUint64,
		MinRequestTime: math.MaxUint64,
		populated:      populated,
	}
	when := time.Now()
	if track {
		ce.Uses = 1
		ce.LastUse = when
	}
	prepareds.cache.Add(ce, encodeName(prepared.Name(), prepared.QueryContext()), func(entry interface{}) util.Operation {
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
			if track {
				atomic.AddInt32(&oldEntry.Uses, 1)

				// as before
				oldEntry.LastUse = when
			}
		} else {
			op = util.IGNORE
		}
		return op
	})
}

// Auto Prepare
func GetAutoPrepareName(text string, context *planner.PrepareContext) string {

	// different feature controls and index API version generate different names
	// so that the same statement prepared differently can coexist

	var buf [_REALM_SIZE]byte
	realm := buf[0:0:_REALM_SIZE]
	realm = strconv.AppendInt(realm, int64(context.IndexApiVersion()), 16)
	realm = append(realm, '_')
	realm = strconv.AppendInt(realm, int64(context.FeatureControls()), 16)
	realm = append(realm, '_')
	realm = strconv.AppendBool(realm, context.UseFts())
	realm = append(realm, '_')
	realm = strconv.AppendBool(realm, context.UseCBO())
	realm = append(realm, '_')
	realm = append(realm, context.QueryContext()...)
	name, err := util.UUIDV5(string(realm), text)

	// this never happens
	if err != nil {
		return ""
	}
	return name
}

func GetAutoPreparePlan(name, text, namespace string, context *planner.PrepareContext) *plan.Prepared {

	// for auto prepare, we don't verify or reprepare because that would mean
	// accepting valid but possibly suboptimal statements
	// instead, we only check the meta data change counters.
	// either they match, and we have the latest possible plan, or they don't
	// in which case we should plan again, so as to match the non AutoPrepare
	// behaviour
	// we'll let the caller handle the planning.
	// The new statement will have the latest change counters, so until we
	// have a new index no other planning will be necessary
	prep, err := getPrepared(name, "", context.DeltaKeyspaces(), OPT_TRACK|OPT_METACHECK, nil)
	if err != nil {
		if err.Code() != errors.NO_SUCH_PREPARED {
			logging.Infof("Auto Prepare plan fetching failed with %v", err)
		}
		return nil
	}

	// this should never happen
	if text != prep.Text() {
		logging.Infof("Auto Prepare found mismatching name and statement %v %v", name, text)
		return nil
	}
	if prep.IndexApiVersion() != context.IndexApiVersion() || prep.FeatureControls() != context.FeatureControls() ||
		prep.Namespace() != namespace || prep.UseFts() != context.UseFts() || prep.UseCBO() != context.UseCBO() {
		return nil
	}
	return prep
}

func AddAutoPreparePlan(stmt algebra.Statement, prepared *plan.Prepared) bool {

	// certain statements we don't cache anyway
	switch stmt.Type() {
	case "EXPLAIN":
		return false
	case "EXECUTE":
		return false
	case "PREPARE":
		return false
	case "":
		return false
	}

	// we also don't cache anything that might depend on placeholders
	// (you should be using prepared statements for that anyway!)
	if stmt.Params() > 0 {
		return false
	}

	added := true
	prepareds.add(prepared, false, true, func(ce *CacheEntry) bool {
		added = ce.Prepared.Text() == prepared.Text()
		if !added {
			logging.Infof("Auto Prepare found mismatching name and statement %v %v %v", prepared.Name(), prepared.Text(), ce.Prepared.Text())
		}
		return added
	})
	return added
}

// Prepareds and system keyspaces
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

	prepareds.add(prepared, false, false, func(ce *CacheEntry) bool {
		if ce.Prepared.Text() != prepared.Text() {
			added = false
		}
		return added
	})
	fullName := encodeName(prepared.Name(), prepared.QueryContext())
	if !added {
		return errors.NewPreparedNameError(
			fmt.Sprintf("duplicate name: %s", fullName))
	} else {
		distributePrepared(fullName, prepared.EncodedPlan())
		return nil
	}
}

func DeletePrepared(name string) errors.Error {
	if prepareds.cache.Delete(name, nil) {
		return nil
	}
	return errors.NewNoSuchPreparedError(name)
}

func GetPrepared(fullName string, deltaKeyspaces map[string]bool) (prepared *plan.Prepared, err errors.Error) {
	return getPrepared(fullName, "", deltaKeyspaces, 0, nil)
}

func GetPreparedWithContext(preparedName string, queryContext string, deltaKeyspaces map[string]bool,
	options uint32, phaseTime *time.Duration) (*plan.Prepared, errors.Error) {
	return getPrepared(preparedName, queryContext, deltaKeyspaces, options, phaseTime)
}

func getPrepared(preparedName string, queryContext string, deltaKeyspaces map[string]bool, options uint32,
	phaseTime *time.Duration) (prepared *plan.Prepared, err errors.Error) {

	prepared, err = prepareds.getPrepared(preparedName, queryContext, options, phaseTime)
	if err == nil {
		if len(deltaKeyspaces) > 0 || (deltaKeyspaces != nil && prepared.Type() == "DELETE") {
			prepared, err = getTxPrepared(prepared, deltaKeyspaces, phaseTime)
		}
	}
	return prepared, err
}

func (prepareds *preparedCache) getPrepared(preparedName string, queryContext string, options uint32,
	phaseTime *time.Duration) (*plan.Prepared, errors.Error) {
	var err errors.Error
	var prepared *plan.Prepared

	track := (options & OPT_TRACK) != 0
	remote := (options & OPT_REMOTE) != 0
	verify := (options & (OPT_VERIFY | OPT_METACHECK)) != 0
	metaCheck := (options & OPT_METACHECK) != 0

	host, name := distributed.RemoteAccess().SplitKey(preparedName)
	statement, ok := predefinedPrepareStatements[name]
	if ok {
		queryContext = ""
	}

	encodedName := encodeName(name, queryContext)
	ce := prepareds.get(encodedName, track)
	if ce != nil {
		prepared = ce.Prepared
	} else if ok {
		_, err = predefinedPrepareStatement(name, statement, "", "default")
		if err == nil {
			ce = prepareds.get(encodedName, track)
			if ce != nil {
				prepared = ce.Prepared
			}
		}
	}

	if prepared == nil && remote && host != "" && host != distributed.RemoteAccess().WhoAmI() {
		distributed.RemoteAccess().GetRemoteDoc(host, encodedName, "prepareds", "GET",
			func(doc map[string]interface{}) {
				encoded_plan, ok := doc["encoded_plan"].(string)
				if ok {
					prepared, err = DecodePreparedWithContext(name, queryContext, encoded_plan, track, phaseTime)
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
			if !good && !metaCheck {
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
		if !good && !metaCheck {
			prepared, err = reprepare(prepared, nil, phaseTime)
			if err == nil {
				err = AddPrepared(prepared)
			}
		}
	}
	if err != nil {
		return nil, err
	}
	if prepared == nil {
		return nil, errors.NewNoSuchPreparedWithContextError(name, queryContext)
	}
	return prepared, nil
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

func DecodePrepared(prepared_name string, prepared_stmt string) (*plan.Prepared, errors.Error) {
	return DecodePreparedWithContext(prepared_name, "", prepared_stmt, false, nil)
}

func DecodePreparedWithContext(prepared_name string, queryContext string, prepared_stmt string, track bool, phaseTime *time.Duration) (*plan.Prepared, errors.Error) {
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

	// if a query context is specified, name and query context have to match
	// if it isn't, encoded name and query context before comapring to key
	if queryContext != "" {
		if prepared_key != prepared.Name() {
			return nil, errors.NewEncodingNameMismatchError(prepared_name, prepared.Name())
		}
		if queryContext != prepared.QueryContext() {
			return nil, errors.NewEncodingContextMismatchError(prepared_name, queryContext, prepared.QueryContext())
		}
	} else {
		name := encodeName(prepared.Name(), prepared.QueryContext())
		if prepared_key != name {
			return nil, errors.NewEncodingNameMismatchError(prepared_name, name)
		}
	}

	// we don't trust anything strangers give us.
	// check the plan and populate metadata counters
	// reprepare if no good
	good := prepared.Verify()
	if !good {
		newPrepared, prepErr := reprepare(prepared, nil, phaseTime)
		if prepErr == nil {
			prepared = newPrepared
		} else {
			return nil, prepErr
		}
	}

	prepareds.add(prepared, good, track,
		func(oldEntry *CacheEntry) bool {

			// MB-19509: if the entry exists already, the new plan must
			// also be for the same statement as we have in the cache
			if oldEntry.Prepared != prepared &&
				oldEntry.Prepared.Text() != prepared.Text() {
				added = false
				return added
			}

			// MB-19659: this is where we decide plan conflict.
			// the current behaviour is to always use the new plan
			// and amend the cache
			// This is still to be finalized
			return added
		})

	if added {
		return prepared, nil
	} else {
		return nil, errors.NewPreparedEncodingMismatchError(prepared_name)
	}
}

func unmarshalPrepared(bytes []byte, phaseTime *time.Duration) (*plan.Prepared, errors.Error) {
	prepared := plan.NewPrepared(nil, nil, nil)
	err := prepared.UnmarshalJSON(bytes)
	if err != nil {

		// if we failed to unmarshall, we find  the statement
		// and try preparing from scratch
		text, err1 := json.FindKey(bytes, "text")
		if text != nil && err1 == nil {
			var stmt string

			err1 = json.Unmarshal(text, &stmt)
			if err1 == nil {
				prepared.SetText(stmt)
				pl, _ := reprepare(prepared, nil, phaseTime)
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

func reprepare(prepared *plan.Prepared, deltaKeyspaces map[string]bool, phaseTime *time.Duration) (*plan.Prepared, errors.Error) {
	parse := time.Now()

	stmt, err := n1ql.ParseStatement2(prepared.Text(), prepared.Namespace(), prepared.QueryContext())
	if phaseTime != nil {
		*phaseTime += time.Since(parse)
	}

	if err != nil {
		// this should never happen: the statement parsed to start with
		return nil, errors.NewReprepareError(err)
	}

	if _, err = stmt.Accept(rewrite.NewRewrite(rewrite.REWRITE_PHASE1)); err != nil {
		return nil, errors.NewRewriteError(err, "")
	}

	// since this is a reprepare, no need to check semantics again after parsing.
	prep := time.Now()
	requestId, err := util.UUIDV3()
	if err != nil {
		return nil, errors.NewReprepareError(fmt.Errorf("Context is nil"))
	}

	var optimizer planner.Optimizer
	if util.IsFeatureEnabled(prepared.FeatureControls(), util.N1QL_CBO) {
		optimizer = getNewOptimizer()
	}
	// building prepared statements should not depend on args
	var prepContext planner.PrepareContext
	planner.NewPrepareContext(&prepContext, requestId, prepared.QueryContext(), nil, nil,
		prepared.IndexApiVersion(), prepared.FeatureControls(), prepared.UseFts(), prepared.UseCBO(),
		optimizer, deltaKeyspaces)

	pl, err := planner.BuildPrepared(stmt.(*algebra.Prepare).Statement(), store, systemstore, prepared.Namespace(),
		false, true, &prepContext)
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
	pl.SetNamespace(prepared.Namespace())
	pl.SetQueryContext(prepared.QueryContext())
	pl.SetUseFts(prepared.UseFts())
	pl.SetUseCBO(prepared.UseCBO())

	json_bytes, err := pl.MarshalJSON()
	if err != nil {
		return nil, errors.NewReprepareError(err)
	}
	pl.BuildEncodedPlan(json_bytes)
	return pl, nil
}

func predefinedPrepareStatement(name, statement, queryContext, namespace string) (*plan.Prepared, errors.Error) {
	var optimizer planner.Optimizer

	useCBO := util.GetUseCBO()
	if useCBO {
		optimizer = getNewOptimizer()
	}

	requestId, err := util.UUIDV3()
	if err != nil {
		return nil, errors.NewPlanError(nil, "request id is nil")
	}

	var prepContext planner.PrepareContext
	planner.NewPrepareContext(&prepContext, requestId, queryContext, nil, nil,
		util.GetMaxIndexAPI(), util.GetN1qlFeatureControl(), false, useCBO, optimizer, nil)

	stmt, err := n1ql.ParseStatement2(statement, namespace, queryContext)
	if err != nil {
		return nil, errors.NewParseSyntaxError(err, "")
	}

	if _, err = stmt.Accept(rewrite.NewRewrite(rewrite.REWRITE_PHASE1)); err != nil {
		return nil, errors.NewRewriteError(err, "")
	}

	semChecker := semantics.NewSemChecker(true, stmt.Type(), false)
	_, err = stmt.Accept(semChecker)
	if err != nil {
		return nil, errors.NewSemanticsError(err, "")
	}

	prepared, err := planner.BuildPrepared(stmt.(*algebra.Prepare).Statement(), store, systemstore, namespace, false, true, &prepContext)
	if err != nil {
		return nil, errors.NewPlanError(err, "BuildPrepared")
	}

	if prepared == nil {
		return nil, errors.NewNoSuchPreparedWithContextError(name, "")
	}

	prepared.SetName(name)
	prepared.SetText(statement)
	prepared.SetIndexApiVersion(prepContext.IndexApiVersion())
	prepared.SetFeatureControls(prepContext.FeatureControls())
	prepared.SetNamespace(namespace)
	prepared.SetQueryContext(prepContext.QueryContext())
	prepared.SetUseFts(prepContext.UseFts())
	prepared.SetUseCBO(prepContext.UseCBO())
	prepared.SetType(stmt.Type())

	json_bytes, err := prepared.MarshalJSON()
	if err != nil {
		return nil, errors.NewPlanError(err, "")
	}

	prepared.BuildEncodedPlan(json_bytes)

	return prepared, AddPrepared(prepared)
}

const (
	_PREPARE_TX_KEYSPACES = 1
)

func getTxPrepared(prepared *plan.Prepared, deltaKeyspaces map[string]bool, phaseTime *time.Duration) (
	txPrepared *plan.Prepared, err errors.Error) {
	var hashCode string
	txPrepared, hashCode = prepared.GetTxPrepared(deltaKeyspaces)
	if txPrepared != nil {
		return
	}
	txPrepared, err = reprepare(prepared, deltaKeyspaces, phaseTime)
	if err == nil {
		prepared.SetTxPrepared(txPrepared, hashCode)
	}
	return

}
