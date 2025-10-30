//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package prepareds

import (
	"fmt"
	"math"
	"math/rand"
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
	"github.com/couchbase/query/tenant"
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
	Uses           atomic.AlignedInt64
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
		"__get": "PREPARE __get FROM SELECT META(d).id, META(d).cas, TO_STR(META(d).cas) AS scas, META(d).txnMeta, d AS doc " +
			"FROM $1 AS d USE KEYS $2;",
		"__insert": "PREPARE __insert FROM INSERT INTO $1 AS d VALUES ($2, $3, $4) RETURNING TO_STR(META(d).cas) AS scas;",
		"__upsert": "PREPARE __upsert FROM UPSERT INTO $1 AS d VALUES ($2, $3, $4) RETURNING TO_STR(META(d).cas) AS scas;",
		"__update": "PREPARE __update FROM UPDATE $1 AS d USE KEYS $2 SET d = $3, META(d).expiration = $4.expiration " +
			"RETURNING META(d).id, META(d).cas, TO_STR(META(d).cas) AS scas, META(d).txnMeta, d AS doc;",
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

	var preparedPrimeReport []*PrimeReport
	var getRemoteKeysFailed errors.Error
	// try each host until we get something
	for left > 0 {
		count := 0
		failed := 0
		reprepared := 0

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

		preparedPrimeReportEntry := &PrimeReport{
			StartTime: time.Now(),
		}
		decodeFailedReason := map[string]errors.Error{}
		decodeReprepReason := map[string]errors.Errors{}

		// get the keys
		distributed.RemoteAccess().GetRemoteKeys([]string{host}, "prepareds",
			func(id string) bool {
				_, name := distributed.RemoteAccess().SplitKey(id)

				// and for each key get the prepared and add it
				distributed.RemoteAccess().GetRemoteDoc(host, name, "prepareds", "GET",
					func(doc map[string]interface{}) {
						encoded_plan, ok := doc["encoded_plan"].(string)
						if ok {
							_, err, reprepareCause := DecodePrepared(name, encoded_plan, true, logging.NULL_LOG)
							if err == nil {
								count++
								if reprepareCause != nil {
									reprepared++
									decodeReprepReason[name] = reprepareCause
								}
							} else {
								failed++
								decodeFailedReason[name] = err
							}
						}
					},
					func(warn errors.Error) {
					}, distributed.NO_CREDS, "", nil)
				return true
			}, func(warn errors.Error) {
				getRemoteKeysFailed = warn
			}, distributed.NO_CREDS, "")

		preparedPrimeReportEntry.Success = count
		preparedPrimeReportEntry.Failed = failed
		preparedPrimeReportEntry.Reprepared = reprepared
		preparedPrimeReportEntry.Host = host

		if len(decodeFailedReason) > 0 {
			preparedPrimeReportEntry.Reason = decodeFailedReason
		} else if getRemoteKeysFailed != nil {
			preparedPrimeReportEntry.Reason = getRemoteKeysFailed.Error()
		}

		if len(decodeReprepReason) > 0 {
			preparedPrimeReportEntry.RepreparedReason = decodeReprepReason
		}

		preparedPrimeReportEntry.EndTime = time.Now()
		preparedPrimeReport = append(preparedPrimeReport, preparedPrimeReportEntry)
		// we found stuff, that's good enough
		if count > 0 {
			break
		}
	}

	if len(preparedPrimeReport) > 0 {
		if buf, err := json.Marshal(preparedPrimeReport); err == nil {
			logging.Infof("Prepared statement cache prime completed: %v", string(buf))
		}
	}
}

type PrimeReport struct {
	Host             string      `json:"host"`
	StartTime        time.Time   `json:"startTime"`
	EndTime          time.Time   `json:"endTime"`
	Reason           interface{} `json:"reason,omitempty"`
	Success          int         `json:"success"`
	Failed           int         `json:"failed"`
	Reprepared       int         `json:"reprepared"`
	RepreparedReason interface{} `json:"repreparedReason,omitempty"`
}

// preparedCache implements planner.PlanCache
func (this *preparedCache) IsPredefinedPrepareName(name string) bool {
	_, ok := predefinedPrepareStatements[name]
	return ok
}

func (this *preparedCache) GetText(text string, offset int) string {

	// in order to get the force/save option to not to mistake the
	// statement as different and refuse to replace the plan
	// we need to remove it from the statement
	// this we do for backwards compatibility - ideally we should just
	// store and compare the prepared text, since with the current
	// system, variations in the actual prepared statement (eg AS vs FROM, or
	// one extra space, specifying the name of an already prepared anonymous
	// statment, use of string vs identifier for the statement name...)s
	// makes the text verification fails, while it should't
	var i, length int
	prepare := text[:offset]
	uprepare := strings.ToUpper(prepare)
	i1 := strings.Index(uprepare, " FORCE")
	i2 := strings.Index(uprepare, " SAVE")
	if i1 < 0 && i2 < 0 {
		return text
	} else if i1 < 0 {
		// i2 >= 0
		i = i2
		length = 5
	} else if i2 < 0 {
		// i1 >= 0
		i = i1
		length = 6
	} else {
		// i1 >= 0 && i2 >= 0
		if i1 < i2 {
			i = i1
		} else {
			i = i2
		}
		length = 11
	}
	if i+length >= offset {
		return prepare[:i] + text[offset:]
	} else {
		return prepare[:i] + prepare[i+length:] + text[offset:]
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
	var sb strings.Builder
	sb.Grow(_REALM_SIZE) // Pre-allocate expected size
	fmt.Fprintf(&sb, "%x_%x_%t_%t_%s",
		context.IndexApiVersion(),
		context.FeatureControls(),
		context.UseFts(),
		context.UseCBO(),
		namespace)
	name, err := util.UUIDV5(sb.String(), text)
	if err != nil {
		return "", errors.NewPreparedNameError(err.Error())
	}
	return name, nil
}

const (
	EmptyQueryContext   = ""
	DefaultQueryContext = "default:"
	ColonQueryContext   = ":"
)

func encodeName(name string, queryContext string) string {
	if queryContext == EmptyQueryContext || queryContext == ColonQueryContext || queryContext == DefaultQueryContext {
		return name
	}
	var sb strings.Builder
	sb.Grow(len(name) + len(queryContext) + 2) // Pre-allocate capacity
	sb.WriteString(name)
	sb.WriteByte('(')
	sb.WriteString(queryContext)
	sb.WriteByte(')')
	return sb.String()
}

func (this *preparedCache) GetPlan(name, text, namespace string, context *planner.PrepareContext) (*plan.Prepared, errors.Error) {
	prep, err := getPrepared(name, context.QueryContext(), context.DeltaKeyspaces(), OPT_VERIFY, nil, context.Context())
	if err != nil {
		if err.Code() == errors.E_NO_SUCH_PREPARED {
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
			atomic.AddInt64(&rv.Uses, 1)

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
				atomic.AddInt64(&oldEntry.Uses, 1)

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

	var sb strings.Builder
	sb.Grow(_REALM_SIZE) // Pre-allocate expected size
	fmt.Fprintf(&sb, "%x_%x_%t_%t_%s",
		context.IndexApiVersion(),
		context.FeatureControls(),
		context.UseFts(),
		context.UseCBO(),
		context.QueryContext())
	name, err := util.UUIDV5(sb.String(), text)

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
	prep, err := getPrepared(name, "", context.DeltaKeyspaces(), OPT_TRACK|OPT_METACHECK, nil, context.Context())
	if err != nil {
		if err.Code() != errors.E_NO_SUCH_PREPARED {
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
			logging.Infof("Auto Prepare found mismatching name and statement %v %v %v", prepared.Name(), prepared.Text(),
				ce.Prepared.Text())
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

func DeletePreparedFunc(name string, f func(*CacheEntry) bool) errors.Error {
	var process func(interface{}) bool = nil

	if f != nil {
		process = func(entry interface{}) bool {
			ce := entry.(*CacheEntry)
			return f(ce)
		}
	}
	if prepareds.cache.DeleteWithCheck(name, process) {
		return nil
	}
	return errors.NewNoSuchPreparedError(name)
}

func GetPrepared(fullName string, deltaKeyspaces map[string]bool, args ...logging.Log) (
	prepared *plan.Prepared, err errors.Error) {

	var l logging.Log
	if len(args) > 0 {
		l = args[0]
	}
	return getPrepared(fullName, "", deltaKeyspaces, 0, nil, l)
}

func GetPreparedWithContext(preparedName string, queryContext string, deltaKeyspaces map[string]bool,
	options uint32, phaseTime *time.Duration, args ...logging.Log) (*plan.Prepared, errors.Error) {

	var l logging.Log
	if len(args) > 0 {
		l = args[0]
	}
	return getPrepared(preparedName, queryContext, deltaKeyspaces, options, phaseTime, l)
}

func getPrepared(preparedName string, queryContext string, deltaKeyspaces map[string]bool, options uint32,
	phaseTime *time.Duration, log logging.Log) (prepared *plan.Prepared, err errors.Error) {

	prepared, err = prepareds.getPrepared(preparedName, queryContext, options, phaseTime, log)
	if err == nil {
		if len(deltaKeyspaces) > 0 || (deltaKeyspaces != nil && prepared.Type() == "DELETE") {
			prepared, err = getTxPrepared(prepared, deltaKeyspaces, phaseTime, log)
		}
	}
	return prepared, err
}

func (prepareds *preparedCache) getPrepared(preparedName string, queryContext string, options uint32, phaseTime *time.Duration,
	log logging.Log) (*plan.Prepared, errors.Error) {

	var err errors.Error
	var prepared *plan.Prepared

	track := (options & OPT_TRACK) != 0
	remote := (options & OPT_REMOTE) != 0
	verify := (options & (OPT_VERIFY | OPT_METACHECK)) != 0
	metaCheck := (options & OPT_METACHECK) != 0

	host, name := distributed.RemoteAccess().SplitKey(preparedName)
	if host != "" {
		host = tenant.DecodeNodeName(host)
	}
	statement, ok := predefinedPrepareStatements[name]
	if ok {
		queryContext = ""
	}

	encodedName := encodeName(name, queryContext)
	ce := prepareds.get(encodedName, track)
	if ce != nil {
		prepared = ce.Prepared
	} else if ok {
		_, err = predefinedPrepareStatement(name, statement, "", "default", log)
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
					prepared, err, _ = DecodePreparedWithContext(name, queryContext, encoded_plan, track, phaseTime, true, log)
				}
			},
			func(warn errors.Error) {
			}, distributed.NO_CREDS, "", nil)
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
				good = prepared.Verify() == nil
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
				good = prepared.Verify() == nil
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
		if (!good || prepared.PreparedTime().IsZero()) && !metaCheck {
			prepared, err = reprepare(prepared, nil, phaseTime, log)
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

func DecodePrepared(prepared_name string, prepared_stmt string, reprep bool, log logging.Log) (*plan.Prepared, errors.Error, errors.Errors) {
	return DecodePreparedWithContext(prepared_name, "", prepared_stmt, false, nil, reprep, log)
}

func DecodePreparedWithContext(prepared_name string, queryContext string, prepared_stmt string, track bool,
	phaseTime *time.Duration, reprep bool, log logging.Log) (*plan.Prepared, errors.Error, errors.Errors) {

	added := true

	prepared, err, unmarshallErr := unmarshalPrepared(prepared_stmt, phaseTime, reprep, log)
	if err != nil {
		return nil, err, nil
	}

	// MB-19509 we now have to check that the encoded plan matches
	// the prepared statement named in the rest API
	_, prepared_key := distributed.RemoteAccess().SplitKey(prepared_name)

	// if a query context is specified, name and query context have to match
	// if it isn't, encoded name and query context before comparing to key
	if queryContext != "" {
		if prepared_key != prepared.Name() {
			return nil, errors.NewEncodingNameMismatchError(prepared_name, prepared.Name()), nil
		}
		if queryContext != prepared.QueryContext() {
			return nil, errors.NewEncodingContextMismatchError(prepared_name, queryContext, prepared.QueryContext()), nil
		}
	} else {
		name := encodeName(prepared.Name(), prepared.QueryContext())
		if prepared_key != name {
			return nil, errors.NewEncodingNameMismatchError(prepared_name, name), nil
		}
	}

	// we don't trust anything strangers give us.
	// check the plan and populate metadata counters
	// reprepare if no good
	verifyErr := prepared.Verify()
	if verifyErr != nil {
		newPrepared, prepErr := reprepare(prepared, nil, phaseTime, log)
		if prepErr == nil {
			prepared = newPrepared
		} else {
			return nil, prepErr, nil
		}
	}

	prepareds.add(prepared, verifyErr == nil, track,
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
		var reprepReason errors.Errors
		if unmarshallErr != nil {
			reprepReason = append(reprepReason, unmarshallErr)
		}
		if verifyErr != nil {
			reprepReason = append(reprepReason, verifyErr)
		}
		return prepared, nil, reprepReason
	} else {
		return nil, errors.NewPreparedEncodingMismatchError(prepared_name), nil
	}
}

func unmarshalPrepared(encoded string, phaseTime *time.Duration, reprep bool, log logging.Log) (*plan.Prepared, errors.Error, errors.Error) {
	prepared, bytes, err := plan.NewPreparedFromEncodedPlan(encoded)
	if err != nil {
		if reprep && len(bytes) > 0 {

			// if we failed to unmarshall, we find  the statement
			// and try preparing from scratch
			text, err1 := json.FindKey(bytes, "text")
			if text != nil && err1 == nil {
				var stmt string

				err1 = json.Unmarshal(text, &stmt)
				if err1 == nil {
					prepared.SetText(stmt)
					pl, _ := reprepare(prepared, nil, phaseTime, log)
					if pl != nil {
						return pl, nil, err
					}
				}
			} else {
				err = errors.NewUnrecognizedPreparedError(fmt.Errorf("Couldn't find the \"text\" field in the encoded plan"))
			}
		}
		return nil, err, nil
	} else if reprep && (prepared.PlanVersion() > util.PLAN_VERSION) {

		// we got the statement, but it was prepared by a newer engine, reprepare to produce a plan we understand
		pl, err := reprepare(prepared, nil, phaseTime, log)
		if err != nil {
			return nil, err, nil
		}
		return pl, nil, errors.NewPlanVersionChange()
	} else {
		prepared.SetEncodedPlan(encoded)
	}
	return prepared, nil, nil
}

func distributePrepared(name, plan string) {
	go distributed.RemoteAccess().DoRemoteOps([]string{}, "prepareds", "PUT", name, plan,
		func(warn errors.Error) {
			if warn != nil {
				logging.Infof("failed to distribute statement <ud>%v</ud>: %v", name, warn)
			}
		}, distributed.NO_CREDS, "")
}

func reprepare(prepared *plan.Prepared, deltaKeyspaces map[string]bool, phaseTime *time.Duration, log logging.Log) (
	*plan.Prepared, errors.Error) {

	parse := util.Now()

	stmt, err := n1ql.ParseStatement2(prepared.Text(), prepared.Namespace(), prepared.QueryContext(), log)
	if phaseTime != nil {
		*phaseTime += util.Now().Sub(parse)
	}

	if err != nil {
		// this should never happen: the statement parsed to start with
		return nil, errors.NewReprepareError(err)
	}

	if _, err = stmt.Accept(rewrite.NewRewrite(rewrite.REWRITE_PHASE1)); err != nil {
		return nil, errors.NewRewriteError(err, "")
	}

	// since this is a reprepare, no need to check semantics again after parsing.
	prep := util.Now()
	requestId, err := util.UUIDV4()
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
		optimizer, deltaKeyspaces, nil, true)

	pl, err, _ := planner.BuildPrepared(stmt.(*algebra.Prepare).Statement(), store, systemstore, prepared.Namespace(),
		false, true, &prepContext)
	if phaseTime != nil {
		*phaseTime += util.Now().Sub(prep)
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
	pl.SetPreparedTime(prep.ToTime()) // reset the time the plan was re-prepared as the time the plan was generated
	pl.SetReprepared(true)
	pl.SetUserAgent(prepared.UserAgent())
	pl.SetRemoteAddr(prepared.RemoteAddr())
	pl.SetUsers(prepared.Users())

	_, err = pl.BuildEncodedPlan()
	if err != nil {
		return nil, errors.NewReprepareError(err)
	}

	if !util.IsFeatureEnabled(util.GetN1qlFeatureControl(), util.N1QL_IGNORE_IDXR_META) {
		pl.Verify() // this adds the indexers so we can immediately detect future changes
	}
	return pl, nil
}

func predefinedPrepareStatement(name, statement, queryContext, namespace string, log logging.Log) (
	*plan.Prepared, errors.Error) {

	var optimizer planner.Optimizer

	useCBO := util.GetUseCBO()
	if useCBO {
		optimizer = getNewOptimizer()
	}

	requestId, err := util.UUIDV4()
	if err != nil {
		return nil, errors.NewPlanError(nil, "request id is nil")
	}

	// don't pass datastore context for prepared statements
	var prepContext planner.PrepareContext
	planner.NewPrepareContext(&prepContext, requestId, queryContext, nil, nil,
		util.GetMaxIndexAPI(), util.GetN1qlFeatureControl(), false, useCBO, optimizer, nil, nil, true)

	stmt, err := n1ql.ParseStatement2(statement, namespace, queryContext, log)
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

	prepared, err, _ := planner.BuildPrepared(stmt.(*algebra.Prepare).Statement(), store, systemstore, namespace, false, true,
		&prepContext)
	if err != nil {
		return nil, errors.NewPlanError(err, "BuildPrepared")
	}

	if prepared == nil {
		return nil, errors.NewNoSuchPreparedWithContextError(name, "")
	}

	prepared.SetPreparedTime(time.Now()) // set the time the plan was generated
	prepared.SetName(name)
	prepared.SetText(statement)
	prepared.SetIndexApiVersion(prepContext.IndexApiVersion())
	prepared.SetFeatureControls(prepContext.FeatureControls())
	prepared.SetNamespace(namespace)
	prepared.SetQueryContext(prepContext.QueryContext())
	prepared.SetUseFts(prepContext.UseFts())
	prepared.SetUseCBO(prepContext.UseCBO())
	prepared.SetType(stmt.Type())
	_, err = prepared.BuildEncodedPlan()
	if err != nil {
		return nil, errors.NewPlanError(err, "")
	}

	return prepared, AddPrepared(prepared)
}

const (
	_PREPARE_TX_KEYSPACES = 1
)

func getTxPrepared(prepared *plan.Prepared, deltaKeyspaces map[string]bool, phaseTime *time.Duration, log logging.Log) (
	txPrepared *plan.Prepared, err errors.Error) {
	var hashCode string
	txPrepared, hashCode = prepared.GetTxPrepared(deltaKeyspaces)
	if txPrepared != nil {
		return
	}
	txPrepared, err = reprepare(prepared, deltaKeyspaces, phaseTime, log)
	if err == nil {
		prepared.SetTxPrepared(txPrepared, hashCode)
	}
	return

}
