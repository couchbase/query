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
	stdatomic "sync/atomic"
	"time"
	"unicode"

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
	"github.com/couchbase/query/settings"
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
	// MB-75506: what a cache miss is for, which decides whether it looks for a saved plan (see
	// checkSavedPlan()); neither of them: a prepared statement, or an inline UDF entry
	OPT_ADHOC      // an ad hoc (auto prepared) statement
	OPT_CACHE_ONLY // the caller has the plan (e.g. distributed by another node)
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

	// MB-75507: under the MODERATE error policy, the plan re-planned for executions while the cached plan can
	// not be used (see moderateStandIn())
	standIn stdatomic.Pointer[standIn]

	// MB-75516: the plan was re-planned, and not saved, because an index of the saved plan was not found while the
	// GSI metadata was settling (see isIndexerNotReady()): the saved plan is loaded again once it has settled,
	// and from this time (Unix nanoseconds; 0: not provisional)
	provisional stdatomic.Int64
}

// MB-75516: a provisional plan whose saved plan could not be loaded again is loaded again after this time
const _PROVISIONAL_RETRY = time.Minute

// MB-75507: under the MODERATE error policy, the plan kept with the cache entry that stands in for the cached
// plan (see standIn), if it stands in for that plan (the cached plan may have been replaced since it was
// re-planned), and is usable; otherwise it is dropped, and the cached plan is checked again
func (this *CacheEntry) moderateStandIn(cached *plan.Prepared) *plan.Prepared {
	standIn := this.standIn.Load()
	if standIn == nil {
		return nil
	} else if standIn.standsFor == cached && standIn.usable() {
		return standIn.prepared
	}
	this.standIn.CompareAndSwap(standIn, nil)
	return nil
}

var prepareds = &preparedCache{}
var store datastore.Datastore
var systemstore datastore.Datastore
var predefinedPrepareStatements map[string]string

// init prepareds cache
func PreparedsInit(limit int) {
	prepareds.cache = util.NewGenCache(limit)
	planner.SetPlanCache(prepareds)
	settings.SetPlanCache(prepareds)
	planStabilityInit()
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

const _DEF_MAP_SIZE = 8

// MB-73680: delay before retrying a prime decode that failed because the GSI metadata watcher hasn't
// synced yet (see waitForIndexerSync in datastore/couchbase), rather than treating it as a genuine failure.
const _PRIME_RETRY_DELAY = 10 * time.Second

// isIndexerNotReady reports whether a DecodePrepared failure's cause is an index-not-found error - see
// errors.NewUnrecognizedPreparedError, which is what sets Cause() to it - while the GSI metadata has not synced
// yet: the index may exist (MB-73680).
// MB-75507: once the GSI metadata has synced, an index that is not found does not exist (e.g. it was dropped): a
// saved plan using it is re-planned right away, and under FLEXIBLE the new plan replaces the saved plan; it is not
// kept, and re-planned on every node after every restart.
// MB-75516: the GSI metadata first syncs with one indexer node; the indexes of other indexer nodes (e.g. slower to
// restart) may not be found for a while after: an index not found is only taken as not existing once the GSI
// metadata has been synced for _GSI_SETTLE (see SetGsiSettle()).
func isIndexerNotReady(err errors.Error) bool {
	if err == nil || gsiSettledFn() {
		return false
	}
	cause, ok := err.Cause().(errors.Error)
	return ok && errors.IsIndexNotFoundError(cause)
}

// MB-75516: how long after the GSI metadata first synced an index not found is taken as not existing
const _GSI_SETTLE = 2 * time.Minute

var gsiSettle = func() *stdatomic.Int64 {
	rv := &stdatomic.Int64{}
	rv.Store(int64(_GSI_SETTLE))
	return rv
}()

// for tests (e.g. an in-process query engine started just before): the time after the GSI metadata first synced
// for an index not found to be taken as not existing; returns the previous time
func SetGsiSettle(settle time.Duration) time.Duration {
	return time.Duration(gsiSettle.Swap(int64(settle)))
}

func gsiSettled() bool {
	since, synced := datastore.GsiSyncedSince()
	return synced && since >= time.Duration(gsiSettle.Load())
}

// test hook
var gsiSettledFn = gsiSettled

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

	// the plan stability error policy applies to the plans from other nodes as well (they are decoded
	// after the saved plans are loaded, see PreparedsFromPersisted())
	planStabilityMode := settings.GetPlanStabilityMode()
	planStabilityErrorPolicy := settings.GetPlanStabilityErrorPolicy()
	planStability := planStabilityMode != settings.PS_MODE_OFF

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
		decodeFailedReason := make(map[string]errors.Error, _DEF_MAP_SIZE)
		decodeReprepReason := make(map[string]errors.Errors, _DEF_MAP_SIZE)
		var pendingRetry []struct{ name, encoded_plan string }
		retried := 0

		// get the keys
		distributed.RemoteAccess().GetRemoteKeys([]string{host}, "prepareds",
			func(id string) bool {
				_, name := distributed.RemoteAccess().SplitKey(id)

				// and for each key get the prepared and add it
				distributed.RemoteAccess().GetRemoteDoc(host, name, "prepareds", "GET",
					func(doc map[string]any) {
						encoded_plan, ok := doc["encoded_plan"].(string)
						if ok {
							// deferIndexerRace=true: an indexer-not-ready failure is deferred (below)
							// instead of repreparing against the same not-yet-synced state; anything
							// else reprepares immediately, as this loop always has.
							_, err, reprepareCause := DecodePreparedWithContext(name, "", encoded_plan, false, nil,
								true, false, planStability, true, planStabilityMode, planStabilityErrorPolicy,
								logging.NULL_LOG)
							if err == nil {
								count++
								if reprepareCause != nil {
									reprepared++
									decodeReprepReason[name] = reprepareCause
								}
							} else if isIndexerNotReady(err) {
								retried++
								pendingRetry = append(pendingRetry,
									struct{ name, encoded_plan string }{name, encoded_plan})
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

		// one delayed retry for the entries deferred above before giving up on them; reprep=true as a
		// last-resort fallback if the watcher still hasn't synced.
		retrySuccess := 0
		if len(pendingRetry) > 0 {
			time.Sleep(_PRIME_RETRY_DELAY)
			for _, p := range pendingRetry {
				_, err, reprepareCause := DecodePrepared(p.name, p.encoded_plan, true, false, planStability,
					planStabilityMode, planStabilityErrorPolicy, logging.NULL_LOG)
				if err == nil {
					retrySuccess++
					if reprepareCause != nil {
						reprepared++
						decodeReprepReason[p.name] = reprepareCause
					}
				} else {
					failed++
					decodeFailedReason[p.name] = err
				}
			}
		}

		// count and retrySuccess don't overlap: the former is first-try successes only.
		preparedPrimeReportEntry.Success = count
		preparedPrimeReportEntry.Failed = failed
		preparedPrimeReportEntry.Reprepared = reprepared
		preparedPrimeReportEntry.Retried = retried
		preparedPrimeReportEntry.RetrySuccess = retrySuccess
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
		if count > 0 || retrySuccess > 0 {
			break
		}
	}

	if len(preparedPrimeReport) > 0 {
		if buf, err := json.Marshal(preparedPrimeReport); err == nil {
			logging.Infof("Prepared statement cache prime from remote nodes completed: %v", string(buf))
		}
	}
}

type PrimeReport struct {
	Host             string    `json:"host,omitempty"`
	StartTime        time.Time `json:"startTime"`
	EndTime          time.Time `json:"endTime"`
	Reason           any       `json:"reason,omitempty"`
	Success          int       `json:"success"`
	Failed           int       `json:"failed"`
	Reprepared       int       `json:"reprepared"`
	RepreparedReason any       `json:"repreparedReason,omitempty"`

	// Skipped: saved plans not loaded, the plan stability mode does not use them (MB-75517)
	Skipped int `json:"skipped,omitempty"`

	// Retried: decodes that hit an index-not-found error and got a delayed retry. RetrySuccess: how
	// many of those the retry fixed (not included in Success).
	Retried      int `json:"retried,omitempty"`
	RetrySuccess int `json:"retrySuccess,omitempty"`
}

// preparedCache implements planner.PlanCache
// MB-75515: the names of inline UDF entries (plan stability) are reserved too: a prepared statement with such a
// name would take the place of the entry of that UDF
func (this *preparedCache) IsPredefinedPrepareName(name string) bool {
	_, ok := predefinedPrepareStatements[name]
	return ok || strings.HasPrefix(name, _INLINE_UDF_NAME_PREFIX)
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
	return removePrepareOptions(text, offset)
}

// text[:offset] ends with the PREPARE options, if any (FORCE, SAVE, in either order): remove them, and the
// whitespace before them, whatever the case and the whitespace around them (e.g. two spaces or a tab between
// the options); single spaced options are removed as before
func removePrepareOptions(text string, offset int) string {
	if offset <= 0 || offset > len(text) {
		return text
	}
	prepare := text[:offset]
	for {
		trimmed := strings.TrimRightFunc(prepare, unicode.IsSpace)
		removed := false
		for _, option := range []string{"FORCE", "SAVE"} {
			i := len(trimmed) - len(option) - 1
			if i >= 0 && strings.EqualFold(trimmed[i+1:], option) && unicode.IsSpace(rune(trimmed[i])) {
				prepare = trimmed[:i]
				removed = true
				break
			}
		}
		if !removed {
			break
		}
	}
	if len(prepare) == offset {
		return text
	}
	return strings.TrimRightFunc(prepare, unicode.IsSpace) + text[offset:]
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

func (this *preparedCache) EncodeName(name, queryContext string) string {
	return encodeName(name, queryContext)
}

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
	prep, err := getPrepared(name, context.QueryContext(), context.DeltaKeyspaces(), OPT_VERIFY, nil,
		context.GetPlanStabilityMode(), context.GetPlanStabilityErrorPolicy(), context.Context(),
		context.ScanConsistency())
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
	// MB-75507: the PREPARE returns the encoded plan of the plan; under MODERATE, a plan re-planned for a plan that
	// can not be used (for this execution, or standing in for it) is built without it (see reprepare())
	if prep.EncodedPlan() == "" {
		if _, err1 := prep.BuildEncodedPlan(); err1 != nil {
			return nil, errors.NewPreparedEncodedPlanError(name, err1)
		}
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
	var cv any

	if track {
		cv = prepareds.cache.Use(fullName, nil)
	} else {
		cv = prepareds.cache.Get(fullName, nil)
	}
	rv, ok := cv.(*CacheEntry)
	if ok {
		if track && !rv.Prepared.HasVerificationError() {
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

	// cached plans live long: don't let them keep the request that created them alive
	prepared.OwnStrings()

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
	fullName := encodeName(prepared.Name(), prepared.QueryContext())
	// whether added or amended, or ignored because a different statement holds the name, the
	// name is now in the cache
	defer clearUnloaded(fullName)
	if prepared.Persist() {
		markSaved(fullName)
	}
	prepareds.cache.Add(ce, fullName, func(entry any) util.Operation {
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
			oldEntry.standIn.Store(nil)   // MB-75507: for the replaced plan
			oldEntry.provisional.Store(0) // MB-75516: for the replaced plan
			if track && !prepared.HasVerificationError() {
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

// With plan stability AD_HOC or AD_HOC_READ_ONLY, an error (e.g. a saved plan failed verification
// under error policy STRICT) is returned, and the caller must fail the request: planning the
// statement afresh would replace the saved plan. Otherwise errors are logged and a nil plan is
// returned, for the caller to plan the statement.
func GetAutoPreparePlan(name, text, namespace string, context *planner.PrepareContext) (*plan.Prepared, errors.Error) {

	// for auto prepare, we don't verify or reprepare because that would mean
	// accepting valid but possibly suboptimal statements
	// instead, we only check the meta data change counters.
	// either they match, and we have the latest possible plan, or they don't
	// in which case we should plan again, so as to match the non AutoPrepare
	// behaviour
	// we'll let the caller handle the planning.
	// The new statement will have the latest change counters, so until we
	// have a new index no other planning will be necessary
	options := uint32(OPT_TRACK | OPT_ADHOC)
	planStabilityAdHoc := context.IsPlanStabilityAdHoc() || context.IsPlanStabilityAdHocReadOnly()
	if planStabilityAdHoc {
		options |= OPT_VERIFY
	} else {
		options |= OPT_METACHECK
	}
	// MB-75514: the plan is cached, and saved, under the name and the query context (see AddAutoPreparePlan())
	prep, err := getPrepared(name, context.QueryContext(), context.DeltaKeyspaces(), options, nil,
		context.GetPlanStabilityMode(), context.GetPlanStabilityErrorPolicy(), context.Context(),
		context.ScanConsistency())
	if err != nil {
		if err.Code() != errors.E_NO_SUCH_PREPARED {
			if planStabilityAdHoc {
				return nil, err
			}
			logging.Infof("Auto Prepare plan fetching failed with %v", err)
		}
		return nil, nil
	}

	// this should never happen
	if text != prep.Text() {
		logging.Infof("Auto Prepare found mismatching name and statement %v %v", name, text)
		return nil, nil
	}
	if prep.IndexApiVersion() != context.IndexApiVersion() || prep.FeatureControls() != context.FeatureControls() ||
		prep.Namespace() != namespace || prep.UseFts() != context.UseFts() || prep.UseCBO() != context.UseCBO() {
		return nil, nil
	}
	return prep, nil
}

// deltaKeyspaces are those of the transaction the statement runs in, if any (the plan was built with them)
func AddAutoPreparePlan(stmt algebra.Statement, prepared *plan.Prepared, planStabilityMode settings.PlanStabilityMode,
	deltaKeyspaces map[string]bool) (bool, errors.Error) {

	// certain statements we don't cache anyway
	switch stmt.Type() {
	case "EXPLAIN", "EXECUTE", "PREPARE", "":
		return false, nil
	}

	// we also don't cache anything that might depend on placeholders
	// (you should be using prepared statements for that anyway!)
	if stmt.ParamsCount() > 0 {
		return false, nil
	}

	// MB-75514: a plan built in a transaction (with its delta keyspaces, e.g. without covering) is for this
	// request only: it is not the plan of the statement, for requests outside the transaction (in a transaction,
	// the plan of the statement is adapted to it, see getTxPrepared())
	if deltaKeyspaces != nil {
		return false, nil
	}

	// MB-75514: transaction statements (e.g. of implicit transactions, see ExecuteTranStatement()) are cached, but
	// not saved as ad hoc statements
	switch stmt.(type) {
	case *algebra.StartTransaction, *algebra.CommitTransaction, *algebra.RollbackTransaction, *algebra.Savepoint,
		*algebra.TransactionIsolation:
		prepared.SetAdHoc(false)
	}

	fullName := encodeName(prepared.Name(), prepared.QueryContext())
	planStabilityAdHoc := prepared.AdHoc() && (planStabilityMode == settings.PS_MODE_AD_HOC)
	featureName := "Auto Prepare"
	if planStabilityAdHoc {
		featureName = "Plan Stability (AD_HOC)"
	}

	// MB-75479: the mode changed to AD_HOC while this request ran under another mode: the plan is not marked as
	// ad hoc (and would not be saved), and the cached ad hoc statements have been removed (see
	// removeAutoPreparedEntries()); do not cache it, the next execution plans and saves it under AD_HOC
	if planStabilityMode != settings.PS_MODE_AD_HOC && !prepared.AdHoc() && !prepared.Persist() &&
		planStabilityModeFn() == settings.PS_MODE_AD_HOC {
		return false, nil
	}

	// MB-75435: later executions of the statement reuse the cached plan and take the statement type
	// from it (transaction statement check, request accounting)
	prepared.SetType(stmt.Type())
	prepared.SetAutoPrepared(true)
	added := true
	prepareds.add(prepared, false, true, func(ce *CacheEntry) bool {
		added = ce.Prepared.Text() == prepared.Text()
		if !added {
			logging.Errorf("%s found mismatching name and statement %v %v %v", featureName,
				fullName, prepared.Text(), ce.Prepared.Text())
		}
		return added
	})
	if !added {
		return false, nil
	}

	if planStabilityAdHoc {
		err := persistPreparedFn(prepared)
		if err != nil {
			// MB-75464: the statement runs with this plan, but the plan is not kept in the cache: the next
			// execution plans the statement again and retries the save
			logging.Errorf("Plan Stability (AD_HOC) encounters error while trying to persist plan (%v) for statement %v, error %v",
				fullName, prepared.Text(), err)
			removeUnsaved(prepared)
			return false, nil
		}

		distributePrepared(fullName, prepared.EncodedPlan())
	}

	return true, nil
}

const _INLINE_UDF_NAME_PREFIX = "__INLINE_UDF__"

func GetUdfPreparedName(udfName string) string {
	return _INLINE_UDF_NAME_PREFIX + udfName
}

// what a cache miss is for, from the getPrepared() options
func lookupKindOf(options uint32) lookupKind {
	switch {
	case (options & OPT_CACHE_ONLY) != 0:
		return _LOOKUP_CACHE_ONLY
	case (options & OPT_ADHOC) != 0:
		return _LOOKUP_AD_HOC
	}
	return _LOOKUP_PREPARED
}

// test hooks
var loadPreparedFn = loadPrepared
var newPreparedFromEncodedFn = plan.NewPreparedFromEncodedPlan
var persistPreparedFn = persistPrepared
var deletePreparedFn = deletePrepared

// a plan saved in QUERY_METADATA (see savedPlanInfo())
type savedPlan struct {
	text     string
	encoded  string // MB-75506: the encoded plan saved
	persist  bool   // saved with PREPARE SAVE
	found    bool
	readable bool
}

// test hook
var savedPlanInfoFn = savedPlanInfo

// test hooks for decodePrepared() and unmarshalPrepared(): a plan failing verification, and re-planning a
// plan, need a datastore
var verifyPreparedFn = func(prepared *plan.Prepared) errors.Error { return prepared.Verify() }
var reprepareFn = reprepare

// MB-75516: the cached plan was re-planned while the GSI metadata was settling, as an index of the saved plan
// was not found (see decodePrepared()): now that it has settled, load the saved plan again. If the index is
// found, the saved plan is used; if not, it does not exist, and the new plan replaces the saved plan. Either
// way, the plan loaded replaces the cached plan (prepareds.add()), and is returned, to go through the checks
// of a cached plan (see getPrepared()).
// The cached plan is kept (nil is returned) if:
//   - the saved plan would not be replaced (the error policy changed, or plan stability was turned off for
//     a plan saved without PREPARE SAVE): it is not loaded
//   - nothing is saved under the name any more (e.g. plan stability turned off, the saved plan deleted)
//   - the saved plan could not be loaded (e.g. a KV error, re-planning failed): the entry stays provisional,
//     a lookup after _PROVISIONAL_RETRY loads it again
func reloadProvisional(name string, ce *CacheEntry, planStabilityMode settings.PlanStabilityMode,
	planStabilityErrorPolicy settings.PlanStabilityErrorPolicy) *plan.Prepared {

	cached := ce.Prepared
	if planStabilityMode != settings.PS_MODE_OFF {
		if planStabilityErrorPolicy != settings.PS_ERROR_FLEXIBLE {
			return nil
		}
	} else if !cached.Persist() {
		return nil
	}
	prepared, err := loadPreparedFn(name, planStabilityMode, planStabilityErrorPolicy)
	if err != nil {
		logging.Infof("Loading the saved plan of prepared statement %s again failed, the cached plan is used: %v",
			name, err)
		if ce.Prepared == cached {
			ce.provisional.CompareAndSwap(0, time.Now().Add(_PROVISIONAL_RETRY).UnixNano())
		}
		return nil
	}
	return prepared
}

// MB-75464: a plan that could not be saved is not kept in the cache, which would otherwise hold a plan that
// is not saved, and never try to save it again; it is removed only if the cache entry still holds it. The
// entry may have replaced the entry for a saved plan (e.g. PREPARE FORCE, a re-planned plan), so a cache
// miss looks for the saved plan again (see plan_load.go)
func removeUnsaved(prepared *plan.Prepared) {
	fullName := encodeName(prepared.Name(), prepared.QueryContext())
	prepareds.cache.DeleteWithCheck(fullName, func(entry any) bool {
		return entry.(*CacheEntry).Prepared == prepared
	})
	markUnloaded(fullName)
}

func AddUdfPrepared(prepared *plan.Prepared, lock bool) errors.Error {

	added := true
	prepareds.add(prepared, false, false, func(ce *CacheEntry) bool {
		added = ce.Prepared.IsInlineUdf() && ce.Prepared.GetSubqueryPlans(false).IsEquivalent(prepared.GetSubqueryPlans(false), lock)
		if !added {
			logging.Infof("Prepared for Inline UDF found mismatching name and subqueries %v", prepared.Name())
		}
		return added
	})
	if !added {
		return errors.NewInlineUdfPreparedError(prepared.Name(), "adding to prepared cache failed", nil)
	}

	err := persistPreparedFn(prepared)
	if err != nil {
		// MB-75464: the UDF runs with its subquery plans, but the entry is not kept in the cache: the next
		// execution adds it again and retries the save
		logging.Errorf("Plan Stability encounters error while trying to persist plan (%v) for inline UDF, error %v",
			prepared.Name(), err)
		removeUnsaved(prepared)
	}

	return nil
}

func GetUdfPrepared(udfName string, track bool, planStabilityMode settings.PlanStabilityMode,
	planStabilityErrorPolicy settings.PlanStabilityErrorPolicy, args ...logging.Log) (prepared *plan.Prepared, err errors.Error) {

	var l logging.Log
	if len(args) > 0 {
		l = args[0]
	}
	var options uint32
	if track {
		options = OPT_TRACK
	}
	prepName := GetUdfPreparedName(udfName)
	// the inline UDF entry only holds subquery plans (no statement text), so it can not be reprepared
	// for transaction delta keyspaces; subquery plans for transactions are handled by the caller
	// MB-75506: a cache miss looks for the saved entry (see checkSavedPlan()), once
	prepared, err = getPrepared(prepName, "", nil, options, nil, planStabilityMode, planStabilityErrorPolicy, l,
		datastore.UNBOUNDED)

	if prepared != nil {
		return prepared, nil
	} else if err != nil && err.Code() != errors.E_NO_SUCH_PREPARED {
		return nil, err
	}
	return nil, nil
}

func (this *preparedCache) DeleteUdfPrepared(udfName string, planStabilityMode settings.PlanStabilityMode,
	planStabilityErrorPolicy settings.PlanStabilityErrorPolicy) errors.Error {
	prepared, err := GetUdfPrepared(udfName, false, planStabilityMode, planStabilityErrorPolicy, nil)
	if err != nil && err.Code() != errors.E_NO_SUCH_PREPARED {
		return err
	}
	if prepared != nil && prepared.IsInlineUdf() {
		return DeletePreparedFunc(prepared.Name(), nil)
	}
	return nil
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
	dummyF := func(id string, r any) bool {
		return nonBlocking(id, r.(*CacheEntry))
	}
	prepareds.cache.ForEach(dummyF, blocking)
}

func PreparedDo(name string, f func(*CacheEntry)) {
	var process func(any) = nil

	if f != nil {
		process = func(entry any) {
			ce := entry.(*CacheEntry)
			f(ce)
		}
	}
	_ = prepareds.cache.Get(name, process)
}

func AddPrepared(prepared *plan.Prepared, planStability bool) errors.Error {
	return addPrepared(prepared, planStability, true)
}

// a new plan for a PREPARE statement (e.g. PREPARE FORCE): without SAVE and with plan stability off, it is not
// saved, and replaces a plan saved with PREPARE SAVE, so the saved plan is removed as well (MB-75477);
// otherwise it would be loaded again at the next startup
func AddPreparedStatement(prepared *plan.Prepared, planStability bool) errors.Error {
	// MB-75517: plan stability turned on since the request started: saved as well (persistPrepared() decides by the
	// current mode); otherwise the mode change, which saves the cached plans, may have passed the entry already
	planStability = planStability || planStabilityModeFn() != settings.PS_MODE_OFF
	if !prepared.Persist() && !planStability {
		fullName := encodeName(prepared.Name(), prepared.QueryContext())
		remove := false
		if ce := prepareds.get(fullName, false); ce != nil {
			// a different statement holding the name is not replaced (addPrepared() fails)
			remove = ce.Prepared.Persist() && ce.Prepared.Text() == prepared.Text()
		} else if checkSavedPlan(fullName, _LOOKUP_PREPARED, settings.PS_MODE_OFF) {
			// a saved plan may not be in the cache (MB-75506: e.g. evicted; plan stability is off here, so only
			// a name known on this node to be saved with SAVE is looked for): same as a cached one, a different
			// statement holding the name is not replaced (no saved plan if QUERY_METADATA does not exist)
			// a saved plan that can not be read is replaced (and removed): it can not be used
			saved, err := savedPlanInfoFn(fullName)
			if err != nil {
				return err
			}
			if saved.found && saved.readable && saved.text != prepared.Text() {
				return errors.NewPreparedNameError(fmt.Sprintf("duplicate name: %s", fullName))
			}
			remove = saved.found && (saved.persist || !saved.readable)
			if !saved.found {
				clearSaved(fullName)
			}
		}
		if remove {
			// deleting a plan that is not saved succeeds; if it fails, nothing is replaced
			if err := deletePreparedFn(fullName); err != nil {
				return err
			}
			clearSaved(fullName)
		}
	}
	return addPrepared(prepared, planStability, true)
}

// MB-75514: a re-planned plan replacing the cached plan: an auto prepared statement that is not ad hoc (cached, but
// neither saved nor distributed, e.g. with auto_prepare under AD_HOC_READ_ONLY, see AddAutoPreparePlan()) stays
// that way; otherwise as addPrepared(), e.g. a saved ad hoc statement under AD_HOC_READ_ONLY is saved again (see
// getPrepared())
func addReplanned(prepared *plan.Prepared, planStability, distribute bool) errors.Error {
	if prepared.AutoPrepared() && !prepared.AdHoc() && !prepared.Persist() {
		added := true
		prepareds.add(prepared, false, false, func(ce *CacheEntry) bool {
			added = ce.Prepared.Text() == prepared.Text()
			return added
		})
		if !added {
			return errors.NewPreparedNameError(fmt.Sprintf("duplicate name: %s",
				encodeName(prepared.Name(), prepared.QueryContext())))
		}
		return nil
	}
	return addPrepared(prepared, planStability, distribute)
}

func addPrepared(prepared *plan.Prepared, planStability, distribute bool) errors.Error {
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
	}

	// a sequential scan plan stays local: persisting it would also spread it to the other nodes and
	// survive a restart, same as distributing it
	if !distribute {
		return nil
	}

	if prepared.Persist() || planStability {
		err := persistPreparedFn(prepared)
		if err != nil {
			// MB-75464: a plan that is not saved is not kept in the cache
			removeUnsaved(prepared)
			return err
		}
	}

	distributePrepared(fullName, prepared.EncodedPlan())
	return nil
}

// serializes rebuild and save of encoded_plan for newly generated subquery plans, such that
// an older encoded_plan does not overwrite a newer one
var subqPersistLock sync.Mutex

// if one or more subquery plans are newly generated, persist the new subquery plan
// should not be called while holding the mutex of the prepared statement's subquery plans
func PersistSubqueryPlan(prepared *plan.Prepared, planStabilityMode settings.PlanStabilityMode) errors.Error {
	// a sequential scan plan is kept local, same as in addPrepared(); this also covers a newly generated
	// subquery plan with a sequential scan, since its indexers have been added to the prepared statement
	if !shouldPersist(prepared, planStabilityMode) || seqScanKeptLocal(prepared) {
		return nil
	}

	fullName := encodeName(prepared.Name(), prepared.QueryContext())

	subqPersistLock.Lock()
	defer subqPersistLock.Unlock()

	// do not save a prepared statement that has been removed from (or replaced in) the cache
	current := false
	prepareds.cache.Get(fullName, func(entry any) {
		current = entry.(*CacheEntry).Prepared == prepared
	})
	if !current {
		return nil
	}

	// rebuild the encoded_plan (to include subquery)
	_, err := prepared.BuildEncodedPlan()
	if err != nil {
		return errors.NewPreparedEncodedPlanError(fullName, err)
	}
	return persistPrepared(prepared)
}

// same criteria as when a prepared statement is first added to the cache (AddPrepared(),
// AddAutoPreparePlan()): auto prepared statements are only saved under AD_HOC mode
func shouldPersist(prepared *plan.Prepared, planStabilityMode settings.PlanStabilityMode) bool {
	if prepared.Persist() {
		return true
	} else if prepared.IsInlineUdf() {
		return false
	} else if prepared.AutoPrepared() {
		return prepared.AdHoc() && planStabilityMode == settings.PS_MODE_AD_HOC
	}
	return planStabilityMode == settings.PS_MODE_PREPARED_ONLY ||
		planStabilityMode == settings.PS_MODE_AD_HOC ||
		planStabilityMode == settings.PS_MODE_AD_HOC_READ_ONLY
}

// with N1QL_IGNORE_IDXR_META bit set a plan with a sequential scan is not persisted or distributed: each node
// makes its own reprepare decision
// the feature bit check is needed: HasSeqScan() is accurate on any verified plan, but reprepare() only verifies
// the new plan with the bit set
func seqScanKeptLocal(prepared *plan.Prepared) bool {
	return !util.IsFeatureEnabled(util.GetN1qlFeatureControl(), util.N1QL_IGNORE_IDXR_META) && hasSeqScanFn(prepared)
}

// test hook: the current plan stability mode
var planStabilityModeFn = settings.GetPlanStabilityMode

// test hook: HasSeqScan() needs the indexers of a verified plan
var hasSeqScanFn = func(prepared *plan.Prepared) bool { return prepared.HasSeqScan() }

// this function does not check whether the prepared has any saved query plans on disk
func deletePreparedFromCache(name string) errors.Error {
	if prepareds.cache.Delete(name, nil) {
		return nil
	}
	return errors.NewNoSuchPreparedError(name)
}

func DeletePreparedFunc(name string, f func(*CacheEntry) bool) errors.Error {
	ce := prepareds.get(name, false)
	if ce == nil || (f != nil && !f(ce)) {
		return errors.NewNoSuchPreparedError(name)
	}
	// the cache amends an entry in place when it is replaced (e.g. PREPARE FORCE), so check the plan
	prepared := ce.Prepared

	// MB-75464: delete the saved plan before the cache entry: if deleting the saved plan fails, the entry
	// stays in the cache (and the saved plan in QUERY_METADATA), and the delete can be retried; deleting
	// a plan that is not saved succeeds
	if prepared.Persist() || settings.IsPlanStabilityEnabled() {
		if err := deletePreparedFn(name); err != nil {
			return err
		}
		if prepared.Persist() {
			clearSaved(name)
		}
	}

	// not if the entry has been replaced in the meantime: the new plan is kept
	prepareds.cache.DeleteWithCheck(name, func(entry any) bool {
		e := entry.(*CacheEntry)
		return e.Prepared == prepared && (f == nil || f(e))
	})
	return nil
}

func GetPrepared(fullName string, deltaKeyspaces map[string]bool, planStabilityMode settings.PlanStabilityMode,
	planStabilityErrorPolicy settings.PlanStabilityErrorPolicy, args ...logging.Log) (prepared *plan.Prepared, err errors.Error) {

	var l logging.Log
	if len(args) > 0 {
		l = args[0]
	}
	// MB-75506: used for a plan distributed by another node, which the caller has: a cache miss does not look
	// for the saved plan, other than while priming or for an unloaded name
	return getPrepared(fullName, "", deltaKeyspaces, OPT_CACHE_ONLY, nil, planStabilityMode, planStabilityErrorPolicy,
		l, datastore.UNBOUNDED)
}

func GetPreparedWithContext(preparedName string, queryContext string, deltaKeyspaces map[string]bool,
	options uint32, phaseTime *time.Duration, planStabilityMode settings.PlanStabilityMode,
	planStabilityErrorPolicy settings.PlanStabilityErrorPolicy, scanConsistency datastore.ScanConsistency,
	args ...logging.Log) (*plan.Prepared, errors.Error) {

	var l logging.Log
	if len(args) > 0 {
		l = args[0]
	}
	prepared, err := getPrepared(preparedName, queryContext, deltaKeyspaces, options, phaseTime, planStabilityMode,
		planStabilityErrorPolicy, l, scanConsistency)
	// MB-75515: an inline UDF entry (plan stability) only holds the subquery plans of the UDF: it is not a prepared
	// statement, and can not be executed
	if err == nil && prepared != nil && prepared.IsInlineUdf() {
		_, name := distributed.RemoteAccess().SplitKey(preparedName)
		return nil, errors.NewNoSuchPreparedWithContextError(name, queryContext)
	}
	return prepared, err
}

func getPrepared(preparedName string, queryContext string, deltaKeyspaces map[string]bool, options uint32,
	phaseTime *time.Duration, planStabilityMode settings.PlanStabilityMode, planStabilityErrorPolicy settings.PlanStabilityErrorPolicy,
	log logging.Log, scanConsistency datastore.ScanConsistency) (prepared *plan.Prepared, err errors.Error) {

	prepared, err = prepareds.getPrepared(preparedName, queryContext, options, phaseTime,
		planStabilityMode, planStabilityErrorPolicy, log, scanConsistency)
	if err == nil {
		if len(deltaKeyspaces) > 0 || (deltaKeyspaces != nil && prepared.Type() == "DELETE") {
			prepared, err = getTxPrepared(prepared, deltaKeyspaces, phaseTime,
				planStabilityMode, planStabilityErrorPolicy, log)
		}
	}
	return prepared, err
}

func (prepareds *preparedCache) getPrepared(preparedName string, queryContext string, options uint32, phaseTime *time.Duration,
	planStabilityMode settings.PlanStabilityMode, planStabilityErrorPolicy settings.PlanStabilityErrorPolicy,
	log logging.Log, scanConsistency datastore.ScanConsistency) (*plan.Prepared, errors.Error) {

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
	var loaded *plan.Prepared
	if ce != nil {
		if at := ce.provisional.Load(); at != 0 && gsiSettledFn() && time.Now().UnixNano() >= at &&
			ce.provisional.CompareAndSwap(at, 0) {
			loaded = reloadProvisional(encodedName, ce, planStabilityMode, planStabilityErrorPolicy)
		}
	}
	if loaded != nil {
		// MB-75516: the plan loaded again has replaced the cached plan, and is checked as the cached plan below;
		// if it is not in the cache (e.g. it could not be saved), it has just been verified
		prepared = loaded
		if ce = prepareds.get(encodedName, false); ce != nil && ce.Prepared != loaded {
			ce = nil
		}
	} else if ce != nil {
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

	// with plan stability off, the error policy does not apply, also not to a plan saved with PREPARE SAVE:
	// it is a prepared statement that is also saved, re-planned (and saved again) when it can not be used
	planStability := planStabilityMode != settings.PS_MODE_OFF

	if prepared == nil && remote && host != "" && host != distributed.RemoteAccess().WhoAmI() {
		distributed.RemoteAccess().GetRemoteDoc(host, encodedName, "prepareds", "GET",
			func(doc map[string]any) {
				encoded_plan, ok := doc["encoded_plan"].(string)
				if ok {
					prepared, err, _ = DecodePreparedWithContext(name, queryContext, encoded_plan, track, phaseTime,
						true, false, planStability, false, planStabilityMode, planStabilityErrorPolicy, log)
				}
			},
			func(warn errors.Error) {
			}, distributed.NO_CREDS, "", nil)
	} else if prepared != nil {
		if scanConsistency != datastore.UNBOUNDED && scanConsistency != datastore.NOT_SET &&
			scanConsistency != "" && prepared.HasCountScan() {
			// the plan for this scan consistency is for this execution only, and not saved: the plan
			// stability error policy does not apply
			var newPrepared *plan.Prepared
			newPrepared, err = reprepare(prepared, nil, phaseTime, planStability,
				planStabilityMode, planStabilityErrorPolicy, log, scanConsistency, true)
			if err == nil {
				prepared = newPrepared
			}
		} else if verify && ce != nil {
			var good bool

			// MB-75507: under MODERATE, a plan re-planned for the executions while the cached plan can not be
			// used, which is used while it is usable (see standIn)
			moderate := planStability && planStabilityErrorPolicy == settings.PS_ERROR_MODERATE && !metaCheck
			var standInPlan *plan.Prepared
			cached := prepared // the cached plan, before it may be re-planned

			// things have already been set up
			// take the short way home
			populated := ce.populated
			if populated {

				// note that it's fine to check and repopulate without a lock
				// since the structure of the plan tree won't change, nor the
				// keyspaces and indexers, the worse that is going to happen is
				// two requests amending the same counter
				good = metadataCheckFn(prepared)
			}
			var indexers []indexerVersion
			if !good && moderate {
				// the cached plan failed verification, and nothing changed since that could make it usable
				standInPlan = ce.moderateStandIn(cached)
				if standInPlan == nil {
					// the cached plan is verified, and may be re-planned: the GSI indexers of its keyspaces,
					// before, for a plan that may stand in for it (see newStandIn())
					indexers = keyspaceIndexersFn(cached.GetKeyspaceReferences())
				}
			}

			if standInPlan != nil {
				// MB-75507: neither verify the cached plan, nor re-plan
			} else if populated {

				// counters have changed. fetch new values
				if !good && !metaCheck {
					good = verifyPreparedFn(prepared) == nil
					if !good {
						// MB-75516: a verification that fails part way has recorded the current metadata
						// versions of the indexers verified before (see Prepared.addIndexer()): the next
						// MetadataCheck() would pass for the plan that can not be used; verify it in full
						// until it verifies
						ce.Lock()
						ce.populated = false
						ce.Unlock()
					}
				}
			} else {
				var persist bool

				// we have to proceed under a lock to avoid multiple
				// requests populating metadata counters at the same time
				ce.Lock()

				// check again, somebody might have done it in the interim
				if ce.populated {
					good = true
				} else {

					// nada - have to go the long way
					good = verifyPreparedFn(prepared) == nil
					if good {
						ce.populated = true
						if prepared.IsRestored() {
							err1 := prepared.RebuildRestoredPlan()
							if err1 == nil && (prepared.Persist() || planStability) {
								persist = true
							}
						}
					}
				}
				ce.Unlock()

				if persist {
					// MB-75464: the restored plan is saved already, this only refreshes its encoded_plan; the
					// plan is good, so a failure does not fail the request
					if err1 := persistPreparedFn(prepared); err1 != nil {
						logging.Errorf("Error saving rebuilt restored plan for prepared statement %s: %v",
							encodedName, err1)
					}
				}
			}

			if good && ce.standIn.Load() != nil {
				// MB-75507: the cached plan can be used again
				ce.standIn.Store(nil)
			}

			// after all this, it did not work out!
			// here we are going to accept multiple requests creating a new
			// plan concurrently as we don't have a good way to serialize
			// without blocking the whole prepared cacheline
			// locking will occur at adding time: both requests will insert,
			// the last wins
			if standInPlan != nil {
				prepared = standInPlan
			} else if metaCheck {
				if !good {
					prepared = nil
				}
			} else if !good || prepared.PreparedTime().IsZero() {
				// with N1QL_IGNORE_IDXR_META bit set reprepare() calls Verify() on the new plan, recording the indexer
				// metadata versions so that a later index change is detected
				// MB-75479: under AD_HOC_READ_ONLY, an ad hoc statement in the cache already (e.g. its saved plan)
				// is handled as a prepared statement, i.e. follows the error policy (FLEXIBLE replaces the saved
				// plan); the mode only keeps new ad hoc statements from being saved
				// MB-75516: a plan that verified, whose prepared time was unset (UPDATE system:prepareds UNSET
				// planPreparedTime, or the admin endpoint) is re-planned on request: the new plan replaces it,
				// whatever the error policy, as for PREPARE FORCE
				var replace bool
				if good {
					replace = true
				} else {
					replace, err = reprepareAllowed(planStability, planStabilityErrorPolicy)
				}
				if err == nil {
					prepared, err = reprepareFn(prepared, nil, phaseTime, planStability,
						planStabilityMode, planStabilityErrorPolicy, log, scanConsistency, replace)
				}
				if err == nil && replace {
					// if the reprepared plan has a sequential scan don't persist or distribute it: each node
					// makes its own reprepare decision
					// MB-75464: if the new plan can not be saved, it is not kept in the cache (the next
					// execution loads the saved plan again, and re-plans it), but this execution uses it
					// (addPrepared() also fails if a different statement holds the name by now)
					if err1 := addReplanned(prepared, planStability, !seqScanKeptLocal(prepared)); err1 != nil {
						logging.Infof("Re-prepared plan for prepared statement %s used for this execution only, "+
							"not kept in the cache: %v", encodedName, err1)
					}
				} else if err == nil && !good && moderate {
					// MB-75507: MODERATE, used for the next executions too, while it is usable, and the cache
					// entry still holds the plan it stands in for (it may have been replaced meanwhile)
					if standIn := newStandIn(prepared, cached, "", indexers); standIn != nil {
						ce.standIn.Store(standIn)
					}
				}
			}
		}
	}
	if err != nil {
		return nil, err
	}
	if prepared == nil {
		// look for a saved plan if one can exist for the statement (MB-75506), the cache is still
		// being primed, or the saved plan could not be put in the cache (see plan_load.go);
		// loading applies the plan stability error policy to the saved plan
		if checkSavedPlan(encodedName, lookupKindOf(options), planStabilityMode) {
			prepared, err = loadPreparedFn(encodedName, planStabilityMode, planStabilityErrorPolicy)
			if err != nil {
				return nil, err
			}
			if prepared == nil {
				// nothing saved under this name
				clearUnloaded(encodedName)
				clearSaved(encodedName)
			}
		}
		if prepared == nil {
			return nil, errors.NewNoSuchPreparedWithContextError(name, queryContext)
		}
	}
	return prepared, nil
}

func RecordPreparedMetrics(prepared *plan.Prepared, requestTime, serviceTime time.Duration) {
	if prepared == nil {
		return
	}
	if prepared.Name() == "" {
		return
	}
	// MB-75514: the cache entry is named after the name and the query context
	name := encodeName(prepared.Name(), prepared.QueryContext())

	// cache get had already moved this entry to the top of the LRU
	// no need to do it again
	_ = prepareds.cache.Get(name, func(entry any) {
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

func DecodePrepared(prepared_name string, prepared_stmt string, reprep, remap, planStability bool,
	planStabilityMode settings.PlanStabilityMode, planStabilityErrorPolicy settings.PlanStabilityErrorPolicy,
	log logging.Log) (*plan.Prepared, errors.Error, errors.Errors) {
	return DecodePreparedWithContext(prepared_name, "", prepared_stmt, false, nil, reprep, remap,
		planStability, false, planStabilityMode, planStabilityErrorPolicy, log)
}

// deferIndexerRace, with reprep true, makes an indexer-not-ready decode failure (see isIndexerNotReady)
// come back as-is instead of repreparing - only PreparedsRemotePrime sets it.
func DecodePreparedWithContext(prepared_name string, queryContext string, prepared_stmt string, track bool,
	phaseTime *time.Duration, reprep, remap, planStability bool, deferIndexerRace bool,
	planStabilityMode settings.PlanStabilityMode, planStabilityErrorPolicy settings.PlanStabilityErrorPolicy,
	log logging.Log) (*plan.Prepared, errors.Error, errors.Errors) {

	return decodePrepared(prepared_name, queryContext, prepared_stmt, track, phaseTime, reprep, remap,
		planStability, deferIndexerRace, false, planStabilityMode, planStabilityErrorPolicy, log)
}

// persisted is set for a plan loaded from the persisted plans
func decodePrepared(prepared_name string, queryContext string, prepared_stmt string, track bool,
	phaseTime *time.Duration, reprep, remap, planStability, deferIndexerRace, persisted bool,
	planStabilityMode settings.PlanStabilityMode, planStabilityErrorPolicy settings.PlanStabilityErrorPolicy,
	log logging.Log) (*plan.Prepared, errors.Error, errors.Errors) {

	added := true

	// MB-19509 we now have to check that the encoded plan matches
	// the prepared statement named in the rest API
	_, prepared_key := distributed.RemoteAccess().SplitKey(prepared_name)

	// the name of the cache entry
	cacheName := prepared_key
	if queryContext != "" {
		cacheName = encodeName(prepared_key, queryContext)
	}

	// a plan loaded from the saved plans, or under plan stability (e.g. a plan distributed by another
	// node, or from another node at startup, which saved it), that is not put in the cache:
	// a cache miss needs to load the saved plan and apply the error policy (e.g. STRICT returns an
	// error), instead of re-planning or returning 4040
	unloaded := persisted || planStability

	prepared, add, replanned, err, unmarshallErr, indexers := unmarshalPrepared(prepared_stmt, phaseTime, reprep,
		remap, planStability, deferIndexerRace, planStabilityMode, planStabilityErrorPolicy, log)
	if err != nil {
		if unloaded {
			markUnloaded(cacheName)
		}
		return nil, err, nil
	}

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
	// entries for inline UDF only has UDF subquery plans and thus no need to verify (verification
	// for UDF subquery plans performed elsewhere, see VerifySubqueryPlans())
	// a plan re-planned for this execution only (add == false, plan stability MODERATE) is not
	// verified again (reprepare() takes care of that)
	var verifyErr errors.Error
	forceReprepare := false
	if add && !prepared.IsInlineUdf() {
		verifyErr = verifyPreparedFn(prepared)

		// a sequential scan may have been chosen by a node that could not see the indexes at the time:
		// replan against the indexes this node sees (kept local, not persisted or distributed)
		// plan stability MODERATE and STRICT don't allow replacing the plan, so leave it be
		// a plan loaded from the persisted plans is left be: it is the saved plan, not one distributed by another node
		forceReprepare = verifyErr == nil && reprep && !persisted && seqScanKeptLocal(prepared) &&
			(planStabilityMode == settings.PS_MODE_OFF || planStabilityErrorPolicy == settings.PS_ERROR_FLEXIBLE)
		if verifyErr != nil || forceReprepare {
			var newPrepared *plan.Prepared
			replace, prepErr := reprepareAllowed(planStability, planStabilityErrorPolicy)
			if prepErr == nil && !replace && persisted {
				// MB-75507: the GSI indexers of the keyspaces, before re-planning, for a plan that may stand in
				// for the saved plan (see newStandIn())
				indexers = keyspaceIndexersFn(prepared.GetKeyspaceReferences())
			}
			if prepErr == nil {
				newPrepared, prepErr = reprepareFn(prepared, nil, phaseTime, planStability,
					planStabilityMode, planStabilityErrorPolicy, log, datastore.UNBOUNDED, replace)
			}
			if prepErr == nil {
				prepared = newPrepared
				add = replace
				replanned = verifyErr != nil
			} else {
				if unloaded {
					markUnloaded(cacheName)
				}
				return nil, prepErr, nil
			}
		}
	}

	if unloaded && !add {
		// MODERATE: the new plan is for this execution only, the saved plan stays as is and is
		// not in the cache
		markUnloaded(cacheName)
		if persisted {
			// MB-75507: the new plan is used for the next executions too, while the saved plan and the
			// metadata do not change (see loadPrepared())
			setUnloadedStandIn(cacheName, prepared_stmt, prepared, indexers)
		}
	}

	if add {
		prepareds.add(prepared, verifyErr == nil && !forceReprepare, track,
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
	}

	// if add == false, added remains true even though it is not added to the cache
	if added {
		// error policy FLEXIBLE (MODERATE does not add, STRICT does not re-plan): the saved plan
		// could not be used and has been replaced in the cache, replace the saved plan as well,
		// same as when a cached plan is re-planned (see getPrepared()), except:
		//   - a plan with a sequential scan stays local, same as in addPrepared()
		//   - the saved plan could not be decoded because an index was not found while the GSI
		//     metadata has not synced yet: this node may not see the index yet (MB-73680), so the
		//     new plan may lack it (e.g. a primary scan); keep the saved plan, only cache the new
		//     plan (MB-75507: once synced, the index does not exist, the saved plan is replaced)
		// MB-75516: with plan stability off, a plan saved with PREPARE SAVE is saved again too (the error
		// policy does not apply, see getPrepared(); as addPrepared())
		saveReplanned := persisted && add && replanned && (planStability || prepared.Persist()) &&
			!seqScanKeptLocal(prepared)
		if saveReplanned && isIndexerNotReady(unmarshallErr) {
			// MB-75516: the saved plan is loaded again once the GSI metadata has settled (see getPrepared())
			if ce := prepareds.get(cacheName, false); ce != nil && ce.Prepared == prepared {
				ce.provisional.Store(1)
			}
		} else if saveReplanned {
			if err1 := persistPreparedFn(prepared); err1 != nil {
				// MB-75464: this execution uses the new plan, but it is not kept in the cache: the next
				// execution loads the saved plan again, and re-plans it
				logging.Errorf("Error saving re-prepared plan for prepared statement %s: %v", prepared_name, err1)
				removeUnsaved(prepared)
			}
		}

		var reprepReason errors.Errors
		if unmarshallErr != nil {
			reprepReason = append(reprepReason, unmarshallErr)
		}
		if verifyErr != nil {
			reprepReason = append(reprepReason, verifyErr)
		} else if forceReprepare {
			// the plan verified good, report why it was reprepared anyway
			reprepReason = append(reprepReason, errors.NewPlanVerificationError(
				"sequential scan plan reprepared against the indexes available on this node", nil))
		}
		return prepared, nil, reprepReason
	} else {
		return nil, errors.NewPreparedEncodingMismatchError(prepared_name), nil
	}
}

// Returns:
//   - add: whether the plan returned can be added to the cache (false for a plan re-planned for
//     this execution only, plan stability error policy MODERATE)
//   - replanned: whether the plan returned was re-planned because the encoded plan could not be
//     used (not set when re-planned because of a newer plan version)
//   - the error, and the unmarshalling error that caused a re-plan
//   - MB-75507: for a plan re-planned for this execution only, the GSI indexers of the keyspaces of the statement,
//     before re-planning (see newStandIn())
func unmarshalPrepared(encoded string, phaseTime *time.Duration, reprep, remap, planStability bool,
	deferIndexerRace bool, planStabilityMode settings.PlanStabilityMode,
	planStabilityErrorPolicy settings.PlanStabilityErrorPolicy, log logging.Log) (
	*plan.Prepared, bool, bool, errors.Error, errors.Error, []indexerVersion) {

	prepared, bytes, err := newPreparedFromEncodedFn(encoded, remap, nil)
	if err != nil {
		if reprep && len(bytes) > 0 && !(deferIndexerRace && isIndexerNotReady(err)) {

			// if we failed to unmarshall, we find  the statement
			// and try preparing from scratch
			text, err1 := json.FindKey(bytes, "text")
			if text != nil && err1 == nil {
				var stmt string

				err1 = json.Unmarshal(text, &stmt)
				if err1 == nil {
					prepared.SetText(stmt)
					replace, prepErr := reprepareAllowed(planStability, planStabilityErrorPolicy)
					if prepErr != nil {
						// error policy STRICT does not allow re-planning: return that, rather than
						// the unmarshalling error
						return nil, false, false, prepErr, nil, nil
					}
					var indexers []indexerVersion
					if !replace {
						// the plan can not be decoded: its keyspaces from the encoded plan
						var refs []string
						if val, er := json.FindKey(bytes, "keyspaceReferences"); er == nil && val != nil {
							json.Unmarshal(val, &refs)
						}
						indexers = keyspaceIndexersFn(refs)
					}
					pl, _ := reprepareFn(prepared, nil, phaseTime, planStability,
						planStabilityMode, planStabilityErrorPolicy, log, datastore.UNBOUNDED, replace)
					if pl != nil {
						return pl, replace, true, nil, err, indexers
					}
				}
			} else {
				udf, err1 := json.FindKey(bytes, "inlineUdf")
				var inlineUdf bool
				if udf != nil && err1 == nil {
					err1 = json.Unmarshal(udf, &inlineUdf)
				}
				if !inlineUdf {
					err = errors.NewUnrecognizedPreparedError(fmt.Errorf("Couldn't find the \"text\" field in the encoded plan"))
				}
			}
		}
		return nil, false, false, err, nil, nil
	} else if reprep && (prepared.PlanVersion() > util.PLAN_VERSION) {

		// we got the statement, but it was prepared by a newer engine, reprepare to produce a plan we understand
		pl, err := reprepare(prepared, nil, phaseTime, false, planStabilityMode, planStabilityErrorPolicy, log,
			datastore.UNBOUNDED, true)
		if err != nil {
			return nil, false, false, err, nil, nil
		}
		return pl, true, false, nil, errors.NewPlanVersionChange(), nil
	} else {
		prepared.SetEncodedPlan(encoded)
	}
	return prepared, true, false, nil, nil, nil
}

// a plan distributed by another node (PUT /admin/prepareds/<name>), under the (encoded) name
func AddDistributedPrepared(name, encoded string, planStabilityMode settings.PlanStabilityMode,
	planStabilityErrorPolicy settings.PlanStabilityErrorPolicy, log logging.Log) errors.Error {

	cached, _ := GetPrepared(name, nil, planStabilityMode, planStabilityErrorPolicy, log)

	// nothing to do if the prepared is there and the plan matches
	if cached != nil && !cached.MismatchingEncodedPlan(encoded) {
		return nil
	}

	// the plan stability error policy applies to a plan distributed by another node
	prepared, err, _ := DecodePrepared(name, encoded, true, false, planStabilityMode != settings.PS_MODE_OFF,
		planStabilityMode, planStabilityErrorPolicy, log)
	if err != nil {
		return err
	}

	// MB-75506: with plan stability off, a new plan without SAVE for a statement saved with PREPARE SAVE replaces
	// the saved plan (see AddPreparedStatement()): the node that made it may not have removed the saved plan
	// (it was not known there to be saved, e.g. its distribution did not reach that node), which would be loaded
	// again at the next startup. The saved plan is removed once the new plan is in the cache: if decoding the new
	// plan fails, the cached plan, and its saved plan, stay in use. If removing it fails, the new plan stays in the
	// cache, as on the node that made it.
	if planStabilityMode == settings.PS_MODE_OFF && cached != nil && cached.Persist() && !prepared.Persist() &&
		cached.Text() == prepared.Text() {
		removeReplacedSavedPlan(name, cached)
	}
	return nil
}

// MB-75506: remove the plan saved with PREPARE SAVE for the cached plan, replaced by a plan without SAVE from another
// node, only if the saved plan is still the cached plan: a newer plan saved with PREPARE SAVE (e.g. distributed
// before the plan without SAVE, or prepared on this node in the meantime) is kept
// The cached plan is compared exactly with the saved plan: if this node re-planned the plan saved with PREPARE SAVE
// that it received (e.g. a sequential scan plan re-planned locally), the saved plan is kept too, and is only removed
// by the node that made the plan without SAVE. A looser comparison (e.g. the text) would remove a newer plan saved
// with PREPARE SAVE for the same statement.
func removeReplacedSavedPlan(name string, cached *plan.Prepared) {
	saved, err := savedPlanInfoFn(name)
	if err != nil {
		logging.Errorf("Error reading the saved plan for prepared statement %s, replaced by a plan without SAVE "+
			"from another node, not removed: %v", name, err)
		return
	} else if !saved.found {
		clearSaved(name)
		return
	} else if !saved.persist || saved.encoded != cached.EncodedPlan() {
		logging.Infof("Saved plan for prepared statement %s, replaced by a plan without SAVE from another node, "+
			"is not the cached plan: not removed", name)
		return
	}
	if err = deletePreparedFn(name); err != nil {
		logging.Errorf("Error removing the saved plan for prepared statement %s, replaced by a plan without SAVE "+
			"from another node: %v", name, err)
	} else {
		clearSaved(name)
	}
}

func distributePrepared(name, plan string) {
	go distributed.RemoteAccess().DoRemoteOps([]string{}, "prepareds", "PUT", name, plan,
		func(warn errors.Error, node string) {
			if warn != nil {
				logging.Infof("failed to distribute statement <ud>%v</ud>: %v", name, warn)
			}
		}, distributed.NO_CREDS, "")
}

// Whether a plan is re-prepared because the existing plan can not be used, under the plan stability
// error policy: the caller checks that first, see reprepareAllowed(). A plan built for a single
// execution context (transaction delta keyspaces, scan consistency for a count scan) is not cached
// as the statement's plan nor saved, and the error policy does not apply.
// planStability is passed on to BuildPrepared() (generate subquery plans).
// MB-75507: encode is false for a plan that does not replace the existing plan (error policy MODERATE, see
// reprepareAllowed()): it is never saved, distributed, or shown as the statement's plan (it is used for this
// execution, and may stand in for the existing plan, see standIn), so its encoded plan is not built.
func reprepare(prepared *plan.Prepared, deltaKeyspaces map[string]bool, phaseTime *time.Duration,
	planStability bool, planStabilityMode settings.PlanStabilityMode,
	planStabilityErrorPolicy settings.PlanStabilityErrorPolicy, log logging.Log,
	scanConsistency datastore.ScanConsistency, encode bool) (*plan.Prepared, errors.Error) {

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
		optimizer, deltaKeyspaces, nil, true, planStabilityMode, planStabilityErrorPolicy,
		scanConsistency)

	// the text of an auto-prepared statement is the statement itself, not a PREPARE (MB-65809)
	statement := stmt
	if prepStmt, ok := stmt.(*algebra.Prepare); ok {
		statement = prepStmt.Statement()
	}
	pl, err, _ := planner.BuildPrepared(statement, store, systemstore, prepared.Namespace(),
		false, true, prepared.Persist() || planStability, &prepContext)
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
	pl.SetAutoPrepared(prepared.AutoPrepared())
	pl.SetPreparedTime(prep.ToTime()) // reset the time the plan was re-prepared as the time the plan was generated
	pl.SetReprepared(true)
	pl.SetUserAgent(prepared.UserAgent())
	pl.SetRemoteAddr(prepared.RemoteAddr())
	pl.SetUsers(prepared.Users())
	pl.SetPersist(prepared.Persist())
	pl.SetAdHoc(prepared.AdHoc())
	pl.KeyspaceReferences()

	if encode {
		_, err = pl.BuildEncodedPlan()
		if err != nil {
			return nil, errors.NewReprepareError(err)
		}
	}

	if !util.IsFeatureEnabled(util.GetN1qlFeatureControl(), util.N1QL_IGNORE_IDXR_META) {
		pl.Verify() // this adds the indexers so we can immediately detect future changes
	}
	return pl, nil
}

// When the existing (possibly saved) plan can not be used, whether it can be re-prepared under the
// plan stability error policy, and if so, whether the new plan replaces the existing plan (in the
// cache, and the saved plan):
//   - FLEXIBLE (or plan stability off): re-prepare and replace
//   - MODERATE: re-prepare, use the new plan for this execution only
//   - STRICT: do not re-prepare, return an error
func reprepareAllowed(planStability bool, planStabilityErrorPolicy settings.PlanStabilityErrorPolicy) (
	bool, errors.Error) {

	if planStability {
		switch planStabilityErrorPolicy {
		case settings.PS_ERROR_STRICT:
			return false, errors.NewReprepareError(fmt.Errorf("Plan Stability error policy STRICT prevents reprepare"))
		case settings.PS_ERROR_MODERATE:
			return false, nil
		}
	}
	return true, nil
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
		util.GetMaxIndexAPI(), util.GetN1qlFeatureControl(), false, useCBO, optimizer, nil, nil, true,
		settings.GetPlanStabilityMode(), settings.GetPlanStabilityErrorPolicy(),
		datastore.UNBOUNDED)

	stmt, err := n1ql.ParseStatement2(statement, namespace, queryContext, log)
	if err != nil {
		return nil, errors.NewParseSyntaxError(err, "")
	}

	if _, err = stmt.Accept(rewrite.NewRewrite(rewrite.REWRITE_PHASE1)); err != nil {
		return nil, errors.NewRewriteError(err, "")
	}

	semChecker := semantics.GetSemChecker(stmt.Type(), false)
	_, err = stmt.Accept(semChecker)
	if err != nil {
		return nil, errors.NewSemanticsError(err, "")
	}

	prepStmt := stmt.(*algebra.Prepare)
	prepared, err, _ := planner.BuildPrepared(prepStmt.Statement(), store, systemstore, namespace,
		false, true, false, &prepContext)
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

	return prepared, AddPrepared(prepared, settings.IsPlanStabilityEnabled())
}

const (
	_PREPARE_TX_KEYSPACES = 1
)

func getTxPrepared(prepared *plan.Prepared, deltaKeyspaces map[string]bool, phaseTime *time.Duration,
	planStabilityMode settings.PlanStabilityMode, planStabilityErrorPolicy settings.PlanStabilityErrorPolicy,
	log logging.Log) (txPrepared *plan.Prepared, err errors.Error) {
	var hashCode string
	txPrepared, hashCode = prepared.GetTxPrepared(deltaKeyspaces)
	if txPrepared != nil {
		return
	}
	// the transaction plan is kept with the prepared statement for these delta keyspaces only, and not
	// saved: the plan stability error policy does not apply
	txPrepared, err = reprepare(prepared, deltaKeyspaces, phaseTime, (planStabilityMode != settings.PS_MODE_OFF),
		planStabilityMode, planStabilityErrorPolicy, log, datastore.UNBOUNDED, true)
	if err == nil {
		prepared.SetTxPrepared(txPrepared, hashCode)
	}
	return

}
