//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package sequences

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/couchbase/cbauth/metakv"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type sequence struct {
	sync.Mutex
	name      *algebra.Path
	key       string // storage key
	base      int64  // base value
	current   int64  // value
	cache     uint64 // size of cache (number of multiples)
	increment int64  // step size (+ve or -ve)
	min       int64  // minimum
	max       int64  // maximum
	cycle     bool   // cycle ?
	remaining uint64 // remaining before next cache load
	rev       int32  // cache revision when this was loaded
	version   int64  // tracks alterations
	accessed  bool   // if NEXT has been called for this sequence yet (since loading)
}

const _CACHE_LIMIT = 65536
const _SEQUENCE = "seq::"
const _CACHE_REVISION_PATH = "/query/sequences_cache/"
const _CACHE_REVISION = _CACHE_REVISION_PATH + "revision"
const _DEFAULT_SEQUENCE_CACHE = 50
const INTERNAL_OPT_INITIAL = "_initial"
const OPT_START = "start"
const OPT_RESTART = "restart"
const _SEQUENCE_RESTART_DEFAULT = "restart_default"
const OPT_CACHE = "cache"
const OPT_INCR = "increment"
const OPT_MIN = "min"
const OPT_MAX = "max"
const OPT_CYCLE = "cycle"
const _WARN_CACHE_SIZE = 10
const _SEQUENCES_ENABLED_CHECK_INTERVAL = time.Minute
const _BATCH_SIZE = 512

var _CREATE_SEQUENCE_OPTIONS = []string{OPT_START, OPT_CACHE, OPT_INCR, OPT_MIN, OPT_MAX, OPT_CYCLE, INTERNAL_OPT_INITIAL}
var _ALTER_SEQUENCE_OPTIONS = []string{OPT_RESTART, OPT_CACHE, OPT_INCR, OPT_MIN, OPT_MAX, OPT_CYCLE}
var sequences *util.GenCache
var cacheRevision int32

func init() {
	sequences = util.NewGenCache(_CACHE_LIMIT)
	err := metakv.Add(_CACHE_REVISION, fmtCacheRevision())
	if err != metakv.ErrRevMismatch {
		logging.Warnf("Unable to start sequences cache monitor: %v", errors.NewSequenceError(errors.E_SEQUENCE_METAKV, err))
	}
	go metakv.RunObserveChildren(_CACHE_REVISION_PATH, sequenceChangeMonitor, make(chan struct{}))
}

func getStorageKey(path *algebra.Path) string {
	uid, _ := datastore.GetScopeUid(path.Namespace(), path.Bucket(), path.Scope())
	return _SEQUENCE + uid + "::" + path.Scope() + "." + path.Keyspace()
}

func getCacheKey(namespace string, bucket string, key string) string {
	return namespace + ":" + bucket + "." + trimPrefixAndScopeUid(key)
}

func trimPrefixAndScopeUid(key string) string {
	if len(key) > len(_SEQUENCE)+10 && strings.HasPrefix(key, _SEQUENCE) {
		key = key[len(_SEQUENCE)+10:]
	}
	return key
}

func validateScope(path *algebra.Path) (errors.Error, string) {
	if path.Namespace() == datastore.SYSTEM_NAMESPACE {
		return errors.NewDatastoreInvalidPathError("system namespace not permitted"), ""
	}

	store := datastore.GetDatastore()
	if store == nil {
		return errors.NewNoDatastoreError(), ""
	}

	var ns datastore.Namespace
	var b datastore.Bucket
	var err errors.Error
	var uid string

	ns, err = store.NamespaceById(path.Namespace())
	if err == nil {
		b, err = ns.BucketByName(path.Bucket())
		if err == nil {
			var s datastore.Scope
			s, err = b.ScopeByName(path.Scope())
			if err == nil {
				uid = s.Uid()
			}
		}
	}
	return err, uid
}

func getSystemCollection(bucket string) (datastore.Keyspace, errors.Error) {
	var ks datastore.Keyspace
	var err errors.Error

	store := datastore.GetDatastore()
	if store == nil {
		return nil, errors.NewNoDatastoreError()
	}
	ks, err = store.GetSystemCollection(bucket)
	if err == nil && ks == nil {
		err = errors.NewSequenceError(errors.E_SEQUENCE_NOT_ENABLED, bucket, err)
	} else if err != nil && err.Code() == errors.E_CB_SCOPE_NOT_FOUND {
		err = errors.NewSequenceError(errors.E_SEQUENCE_NOT_ENABLED, bucket, nil)
	}
	return ks, err
}

func getSpan(prefix string) datastore.Spans2 {
	next := []byte(prefix)
	next[len(next)-1] = next[len(next)-1] + 1
	spans := make([]*datastore.Span2, 1)
	spans[0] = &datastore.Span2{}
	spans[0].Ranges = make([]*datastore.Range2, 1)
	spans[0].Ranges[0] = &datastore.Range2{}
	spans[0].Ranges[0].Low = value.NewValue(prefix)
	spans[0].Ranges[0].High = value.NewValue(string(next))
	spans[0].Ranges[0].Inclusion = datastore.LOW
	return spans
}

func CreateSequence(path *algebra.Path, with value.Value) errors.Error {

	if path.Scope() == "" {
		return errors.NewSequenceError(errors.E_SEQUENCE_INVALID_NAME, path.SimpleString())
	}
	name := path.SimpleString()

	var w errors.Error
	start := int64(0)
	startSet := false
	initial := int64(0)
	initialSet := false
	cache := uint64(_DEFAULT_SEQUENCE_CACHE)
	min := int64(math.MinInt64)
	minSet := false
	max := int64(math.MaxInt64)
	maxSet := false
	increment := int64(1)
	cycle := false
	if with != nil {
		err := validateWithOptions(with, _CREATE_SEQUENCE_OPTIONS)
		if err != nil {
			return errors.NewSequenceError(errors.E_SEQUENCE_CREATE, name, err)
		}
		for _, opt := range _CREATE_SEQUENCE_OPTIONS {
			var tf bool
			var ok bool
			var num int64
			if opt == OPT_CYCLE {
				tf, ok, err = getWithBoolOption(with, opt, true)
			} else {
				num, ok, err = getWithOption(with, opt, true)
			}
			if err != nil {
				return errors.NewSequenceError(errors.E_SEQUENCE_CREATE, name, err)
			}
			if ok {
				switch opt {
				case OPT_START:
					start = num
					startSet = true
				case OPT_CACHE:
					if num <= 0 {
						return errors.NewSequenceError(errors.E_SEQUENCE_CREATE, name,
							errors.NewSequenceError(errors.E_SEQUENCE_INVALID_CACHE, fmt.Sprintf("%v", num)))
					}
					cache = uint64(num)
					if cache < _WARN_CACHE_SIZE {
						w = errors.NewSequenceError(errors.W_SEQUENCE_CACHE_SIZE, fmt.Sprintf("%v", num))
					}
				case OPT_MIN:
					min = num
					minSet = true
				case OPT_MAX:
					max = num
					maxSet = true
				case OPT_INCR:
					increment = num
				case OPT_CYCLE:
					cycle = tf
				case INTERNAL_OPT_INITIAL:
					initial = num
					initialSet = true
				}
			}
		}
	}
	if min >= max {
		return errors.NewSequenceError(errors.E_SEQUENCE_CREATE, name,
			errors.NewSequenceError(errors.E_SEQUENCE_INVALID_RANGE, fmt.Sprintf("%v to %v", min, max)))
	}

	if !startSet {
		if increment > 0 && minSet {
			start = min
		}
		if increment < 0 && maxSet {
			start = max
		}
	}

	if !initialSet {
		initial = start
	}

	seq, err := getLockedSequence(name)
	if err == nil {
		seq.Unlock()
		return errors.NewSequenceError(errors.E_SEQUENCE_ALREADY_EXISTS, name)
	}

	seq = &sequence{
		name:      path,
		key:       getStorageKey(path),
		current:   0,
		base:      start,
		cache:     cache,
		min:       min,
		max:       max,
		increment: increment,
		cycle:     cycle,
		remaining: 0,
		version:   0,
	}

	err, _ = validateScope(path)
	if err != nil {
		return errors.NewSequenceError(errors.E_SEQUENCE_CREATE, name, err)
	}
	b, err := getSystemCollection(path.Bucket())
	if err != nil {
		return err
	}
	if b.ScopeId() == path.Scope() {
		return errors.NewSequenceError(errors.E_SEQUENCE_INVALID_NAME, path.SimpleString())
	}

	pairs := make([]value.Pair, 1)
	pairs[0].Name = seq.key
	m := make(map[string]value.Value)
	m["initial"] = value.NewValue(fmt.Sprintf("%v", initial)) // only used for ALTER ... RESTART with no value specified
	m["base"] = value.NewValue(fmt.Sprintf("%v", start))
	m["cache"] = value.NewValue(fmt.Sprintf("%v", cache))
	m["min"] = value.NewValue(fmt.Sprintf("%v", min))
	m["max"] = value.NewValue(fmt.Sprintf("%v", max))
	m["increment"] = value.NewValue(fmt.Sprintf("%v", increment))
	m["cycle"] = value.NewValue(cycle)
	m["block"] = value.NewValue(0)
	m["version"] = value.NewValue(0)
	pairs[0].Value = value.NewAnnotatedValue(value.NewValue(m))

	_, _, errs := b.Insert(pairs, datastore.GetDurableQueryContextFor(b), true)
	if errs != nil && len(errs) > 0 {
		return errors.NewSequenceError(errors.E_SEQUENCE_CREATE, name, errs[0])
	}

	seq.rev = nextRevision()
	sequences.Add(seq, name, nil)

	return w
}

func DropSequence(path *algebra.Path, force bool) errors.Error {

	if path.Scope() == "" {
		return errors.NewSequenceError(errors.E_SEQUENCE_INVALID_NAME, path.SimpleString())
	}
	name := path.SimpleString()

	seq, err := getLockedSequence(name)
	if err != nil && !force {
		return err
	}

	b, err := getSystemCollection(path.Bucket())
	if err != nil {
		if seq != nil {
			seq.Unlock()
		}
		return err
	}

	pairs := make([]value.Pair, 1)
	pairs[0].Name = getStorageKey(path)
	_, _, errs := b.Delete(pairs, datastore.GetDurableQueryContextFor(b), true)
	if errs != nil && len(errs) > 0 {
		if seq != nil {
			seq.Unlock()
		}
		return errors.NewSequenceError(errors.E_SEQUENCE_DROP, name, errs[0])
	}
	if seq == nil {
		sequences.Delete(name, nil)
	} else {
		sequences.Delete(name, func(s interface{}) {
			seq := s.(*sequence)
			seq.Unlock()
		})
	}
	nextRevision()

	return nil
}

func AlterSequence(path *algebra.Path, with value.Value) errors.Error {

	if path.Scope() == "" {
		return errors.NewSequenceError(errors.E_SEQUENCE_INVALID_NAME, path.SimpleString())
	}
	name := path.SimpleString()

	restart := int64(0)
	setRestart := false
	setRestartDefault := false
	cache := uint64(0)
	setCache := false
	min := int64(math.MinInt64)
	setMin := false
	max := int64(math.MaxInt64)
	setMax := false
	increment := int64(1)
	setIncrement := false
	cycle := false
	setCycle := false

	var w errors.Error
	if with != nil {
		err := validateWithOptions(with, _ALTER_SEQUENCE_OPTIONS)
		if err != nil {
			return errors.NewSequenceError(errors.E_SEQUENCE_ALTER, name, err)
		}
		for _, opt := range _ALTER_SEQUENCE_OPTIONS {
			var tf bool
			var ok bool
			var num int64
			if opt == OPT_CYCLE {
				tf, ok, err = getWithBoolOption(with, opt, true)
			} else if opt == OPT_RESTART {
				tf, ok, err = getWithBoolOption(with, opt, true)
				if err == nil {
					opt = _SEQUENCE_RESTART_DEFAULT
				} else {
					num, ok, err = getWithOption(with, opt, true)
				}

			} else {
				num, ok, err = getWithOption(with, opt, true)
			}
			if err != nil {
				return errors.NewSequenceError(errors.E_SEQUENCE_ALTER, name, err)
			}
			if ok {
				switch opt {
				case _SEQUENCE_RESTART_DEFAULT:
					if tf {
						setRestart = true
						setRestartDefault = true
					}
				case OPT_RESTART:
					restart = num
					setRestart = true
				case OPT_CACHE:
					if num <= 0 {
						return errors.NewSequenceError(errors.E_SEQUENCE_ALTER, name,
							errors.NewSequenceError(errors.E_SEQUENCE_INVALID_CACHE, fmt.Sprintf("%v", num)))
					}
					cache = uint64(num)
					setCache = true
					if cache < _WARN_CACHE_SIZE {
						w = errors.NewSequenceError(errors.W_SEQUENCE_CACHE_SIZE, fmt.Sprintf("%v", num))
					}
				case OPT_MIN:
					min = num
					setMin = true
				case OPT_MAX:
					max = num
					setMax = true
				case OPT_INCR:
					increment = num
					setIncrement = true
				case OPT_CYCLE:
					cycle = tf
					setCycle = true
				}
			}
		}
	}

	if setMin && setMax && min > max {
		return errors.NewSequenceError(errors.E_SEQUENCE_ALTER, name,
			errors.NewSequenceError(errors.E_SEQUENCE_INVALID_RANGE, fmt.Sprintf("%v to %v", min, max)))
	}

	seq, err := getLockedSequence(name)
	if err != nil {
		return err
	}
	defer seq.Unlock()

	if setCache && seq.cache == cache {
		setCache = false
	}
	if setMin && seq.min == min {
		setMin = false
	}
	if setMax && seq.max == max {
		setMax = false
	}
	if setIncrement && seq.increment == increment {
		setIncrement = false
	}
	if setCycle && seq.cycle == cycle {
		setCycle = false
	}

	if !setRestart && !setCache && !setMin && !setMax && !setIncrement && !setCycle {
		// nothing to alter
		return nil
	}

	if setMin && min > seq.max {
		max = seq.max
		return errors.NewSequenceError(errors.E_SEQUENCE_ALTER, name,
			errors.NewSequenceError(errors.E_SEQUENCE_INVALID_RANGE, fmt.Sprintf("%v to %v", min, max)))
	}

	if setMax && max < seq.min {
		min = seq.min
		return errors.NewSequenceError(errors.E_SEQUENCE_ALTER, name,
			errors.NewSequenceError(errors.E_SEQUENCE_INVALID_RANGE, fmt.Sprintf("%v to %v", min, max)))
	}

	err, _ = validateScope(path)
	if err != nil {
		return errors.NewSequenceError(errors.E_SEQUENCE_ALTER, name, err)
	}
	b, err := getSystemCollection(path.Bucket())
	if err != nil {
		return err
	}

	for retry := 0; ; retry++ {
		res := make(map[string]value.AnnotatedValue, 1)
		keys := make([]string, 1)
		keys[0] = seq.key
		errs := b.Fetch(keys, res, datastore.NULL_QUERY_CONTEXT, nil, nil, false)
		if errs != nil && len(errs) > 0 {
			return errors.NewSequenceError(errors.E_SEQUENCE_ALTER, name, errs[0])
		}

		av, ok := res[keys[0]]
		if !ok {
			return errors.NewSequenceError(errors.E_SEQUENCE_INVALID_DATA, fmt.Errorf("missing data"))
		}

		if setRestartDefault {
			v, ok := av.Field("initial")
			if !ok || v.Type() != value.STRING {
				return errors.NewSequenceError(errors.E_SEQUENCE_INVALID_DATA, fmt.Errorf("%v", res[keys[0]]))
			}
			var perr error
			restart, perr = strconv.ParseInt(v.ToString(), 10, 64)
			if perr != nil {
				return errors.NewSequenceError(errors.E_SEQUENCE_INVALID_DATA, fmt.Errorf("%v", res[keys[0]]))
			}
		}

		if setIncrement {
			if !setRestart {
				restart, ok = restartValue(av)
				if !ok {
					return errors.NewSequenceError(errors.E_SEQUENCE_INVALID_DATA, fmt.Errorf("%v", res[keys[0]]))
				}
				setRestart = true
			}
			av.SetField("increment", value.NewValue(fmt.Sprintf("%v", increment)))
		}
		if setCache {
			if !setRestart {
				restart, ok = restartValue(av)
				if !ok {
					return errors.NewSequenceError(errors.E_SEQUENCE_INVALID_DATA, fmt.Errorf("%v", res[keys[0]]))
				}
				setRestart = true
			}
			av.SetField("cache", value.NewValue(fmt.Sprintf("%v", cache)))
		}
		if setRestart {
			av.SetField("block", value.NewValue(0))
			av.SetField("base", value.NewValue(fmt.Sprintf("%v", restart)))
		}
		if setMin {
			av.SetField("min", value.NewValue(fmt.Sprintf("%v", min)))
		}
		if setMax {
			av.SetField("max", value.NewValue(fmt.Sprintf("%v", max)))
		}
		if setCycle {
			av.SetField("cycle", value.NewValue(cycle))
		}
		v, ok := av.Field("version")
		if !ok || v.Type() != value.NUMBER {
			return errors.NewSequenceError(errors.E_SEQUENCE_INVALID_DATA, fmt.Errorf("%v", res[keys[0]]))
		}
		av.SetField("version", value.NewValue(value.AsNumberValue(v).Int64()+1))

		pairs := make([]value.Pair, 1)
		pairs[0].Name = keys[0]
		pairs[0].Value = av
		_, _, errs = b.Update(pairs, datastore.GetDurableQueryContextFor(b), true)
		if errs != nil && len(errs) > 0 {
			if errs[0].HasCause(errors.E_CAS_MISMATCH) || errs[0].ContainsText("SYNC_WRITE_IN_PROGRESS") {
				continue
			}
			if errs[0].HasCause(errors.E_KEY_NOT_FOUND) {
				return errors.NewSequenceError(errors.E_SEQUENCE_NOT_FOUND, name)
			}
			return errors.NewSequenceError(errors.E_SEQUENCE_ALTER, name, errs[0])
		}
		break
	}

	if setRestart {
		seq.current = 0
		seq.base = restart
	}
	if setCache {
		seq.cache = cache
	}
	if setMin {
		seq.min = min
	}
	if setMax {
		seq.max = max
	}
	if setIncrement {
		seq.increment = increment
	}
	if setCycle {
		seq.cycle = cycle
	}
	seq.remaining = 0
	seq.rev = nextRevision()
	return w
}

func restartValue(av value.AnnotatedValue) (int64, bool) {
	v, ok := av.Field("base")
	if !ok || v.Type() != value.STRING {
		return 0, false
	}
	bv, perr := strconv.ParseInt(v.ToString(), 10, 64)
	if perr != nil {
		return 0, false
	}
	v, ok = av.Field("increment")
	if !ok || v.Type() != value.STRING {
		return 0, false
	}
	incr, perr := strconv.ParseInt(v.ToString(), 10, 64)
	if perr != nil {
		return 0, false
	}
	v, ok = av.Field("cache")
	if !ok || v.Type() != value.STRING {
		return 0, false
	}
	cache, perr := strconv.ParseInt(v.ToString(), 10, 64)
	if perr != nil {
		return 0, false
	}
	v, ok = av.Field("block")
	if !ok || v.Type() != value.NUMBER {
		return 0, false
	}
	block := value.AsNumberValue(v).Int64()
	rv := block*cache*incr + bv
	return rv, true
}

func loadSequence(name string) errors.Error {
	var seq *sequence
	s := sequences.Get(name, nil)
	if s == nil {
		elements := algebra.ParsePath(name)
		if len(elements) < 3 {
			return errors.NewSequenceError(errors.E_SEQUENCE, fmt.Errorf("%v", name))
		}
		path := algebra.NewPathFromElements(elements)
		seq = &sequence{name: path, key: getStorageKey(path), current: 0}
		sequences.Add(seq, name, nil)
	} else {
		seq = s.(*sequence)
	}
	seq.Lock()
	err := seq.load()
	if err != nil {
		sequences.Delete(name, nil)
	}
	seq.Unlock()
	return err
}

func DropAllSequences(namespace string, bucket string, scope string, uid string) errors.Error {

	var del string
	keyPrefix := namespace + ":" + bucket
	if scope != "" {
		keyPrefix += "." + scope + "."
	}

	// clear the cache entries
	listWalkMutex.Lock()
	sequences.ForEach(
		func(k string, s interface{}) bool {
			if strings.HasPrefix(k, keyPrefix) {
				del = k
			} else {
				del = ""
			}
			return true
		},
		func() bool {
			if del != "" {
				sequences.Delete(del, nil)
			}
			return true
		})
	listWalkMutex.Unlock()

	var lastError errors.Error
	pairs := make([]value.Pair, 0, _BATCH_SIZE)
	errorCount := 0

	prefix := _SEQUENCE
	if scope != "" {
		prefix += uid + "::" + scope + "."
	}
	var qcontext datastore.QueryContext
	err := datastore.ScanSystemCollection(bucket, prefix,
		func(systemCollection datastore.Keyspace) errors.Error {
			qcontext = datastore.GetDurableQueryContextFor(systemCollection)
			return nil
		},
		func(key string, systemCollection datastore.Keyspace) errors.Error {
			pairs = append(pairs, value.Pair{Name: key})
			if len(pairs) >= _BATCH_SIZE {
				_, results, errs := systemCollection.Delete(pairs, qcontext, true)
				for i := range results {
					sequences.Delete(getCacheKey(namespace, bucket, results[i].Name), nil)
				}
				if errs != nil && len(errs) > 0 {
					errorCount += len(errs)
					lastError = errors.NewSequenceError(errors.E_SEQUENCE_DROP_ALL, bucket+"."+scope+".*", errs[0])
				}
				pairs = pairs[:0]
			}
			return nil
		},
		func(systemCollection datastore.Keyspace) errors.Error {
			if len(pairs) > 0 {
				_, results, errs := systemCollection.Delete(pairs, qcontext, true)
				for i := range results {
					sequences.Delete(getCacheKey(namespace, bucket, results[i].Name), nil)
				}
				if errs != nil && len(errs) > 0 {
					errorCount += len(errs)
					lastError = errors.NewSequenceError(errors.E_SEQUENCE_DROP_ALL, bucket+"."+scope+".*", errs[0])
				}
			}
			return nil
		})
	if err != nil && err.Code() == errors.E_CB_KEYSPACE_NOT_FOUND {
		logging.Debugf("%v:%v.%v %v", namespace, bucket, scope, err)
		return nil
	}
	if err != nil && lastError == nil {
		lastError = err
	}
	logging.Debugf("%v:%v.%v %v - %v", namespace, bucket, scope, errorCount, lastError)
	return lastError
}

// Authority to access the sequence must be validated by the caller
func NextSequenceValue(name string) (int64, errors.Error) {
	seq, err := getLockedSequence(name)
	if err != nil {
		return 0, err
	}
	seq.accessed = true

	if seq.remaining > 0 {
		seq.remaining--
		prev := seq.current
		seq.current += seq.increment
		return seq.checkCycle(prev)
	}

	return seq.cacheBlock()
}

// Authority to access the sequence must be validated by the caller
func PrevSequenceValue(name string) (int64, errors.Error) {
	seq, err := getLockedSequence(name)
	if err != nil {
		return 0, err
	}
	if !seq.accessed {
		seq.Unlock()
		return 0, errors.NewSequenceError(errors.W_SEQUENCE_NO_PREV_VALUE)
	}
	if seq.remaining > 0 {
		rv := seq.current
		seq.Unlock()
		return rv, nil
	}

	return seq.cacheBlock()
}

func getLockedSequence(name string) (*sequence, errors.Error) {
	loaded := false
	for {
		s := sequences.Get(name, nil)
		if s == nil {
			if !loaded {
				err := loadSequence(name)
				if err != nil {
					if err.Code() == errors.E_CB_KEYSPACE_NOT_FOUND {
						return nil, errors.NewSequenceError(errors.E_SEQUENCE_NOT_FOUND, name)
					}
					return nil, err
				}
				loaded = true
				continue
			}
			return nil, errors.NewSequenceError(errors.E_SEQUENCE_NOT_FOUND, name)
		}
		seq := s.(*sequence)

		seq.Lock()

		// trigger refresh if needed
		_, err := getSystemCollection(seq.name.Bucket())
		if err != nil {
			seq.Unlock()
			return nil, err
		}

		s = sequences.Get(name, nil)
		if s == nil {
			// seq doesn't exist so unlocking not necessary
			return nil, errors.NewSequenceError(errors.E_SEQUENCE_NOT_FOUND, name)
		}

		if seq.rev != cacheRevision {
			if seq.validateVersion() {
				seq.rev = cacheRevision
				return seq, nil
			}
			seq.Unlock()
			err := loadSequence(name)
			if err != nil {
				if err.Code() == errors.E_CB_KEYSPACE_NOT_FOUND {
					return nil, errors.NewSequenceError(errors.E_SEQUENCE_NOT_FOUND, name)
				}
				return nil, err
			}
			continue
		}
		return seq, nil
	}
}

// expect locked on entry
func (seq *sequence) load() errors.Error {
	err, _ := validateScope(seq.name)
	if err != nil {
		return err // bubble scope-not-found up
	}
	b, err := getSystemCollection(seq.name.Bucket())
	if err != nil {
		return err
	}

	res := make(map[string]value.AnnotatedValue, 1)
	keys := make([]string, 1)
	keys[0] = seq.key
	errs := b.Fetch(keys, res, datastore.NULL_QUERY_CONTEXT, nil, nil, false)
	if errs != nil && len(errs) > 0 {
		if !errors.IsNotFoundError("", errs[0]) && !errs[0].HasCause(errors.E_CB_BULK_GET) {
			return errs[0]
		}
		return errors.NewSequenceError(errors.E_SEQUENCE_NOT_FOUND, seq.name.SimpleString())
	}

	av, ok := res[keys[0]]
	if !ok {
		return errors.NewSequenceError(errors.E_SEQUENCE_NOT_FOUND, seq.name.SimpleString())
	}

	v, ok := av.Field("version")
	if !ok || v.Type() != value.NUMBER {
		return errors.NewSequenceError(errors.E_SEQUENCE_INVALID_DATA, fmt.Errorf("%v", av))
	}
	version := value.AsNumberValue(v).Int64()

	v, ok = av.Field("cache")
	if !ok || v.Type() != value.STRING {
		return errors.NewSequenceError(errors.E_SEQUENCE_INVALID_DATA, fmt.Errorf("%v", av))
	}
	cache, perr := strconv.ParseUint(v.ToString(), 10, 64)
	if perr != nil {
		return errors.NewSequenceError(errors.E_SEQUENCE,
			errors.NewSequenceError(errors.E_SEQUENCE_INVALID_DATA, fmt.Errorf("%v", av), perr))
	}

	v, ok = av.Field("base")
	if !ok || v.Type() != value.STRING {
		return errors.NewSequenceError(errors.E_SEQUENCE_INVALID_DATA, fmt.Errorf("%v", av))
	}
	base, perr := strconv.ParseInt(v.ToString(), 10, 64)
	if perr != nil {
		return errors.NewSequenceError(errors.E_SEQUENCE_INVALID_DATA, fmt.Errorf("%v", av))
	}

	v, ok = av.Field("min")
	if !ok || v.Type() != value.STRING {
		return errors.NewSequenceError(errors.E_SEQUENCE_INVALID_DATA, fmt.Errorf("%v", av))
	}
	min, perr := strconv.ParseInt(v.ToString(), 10, 64)
	if perr != nil {
		return errors.NewSequenceError(errors.E_SEQUENCE_INVALID_DATA, fmt.Errorf("%v", av))
	}

	v, ok = av.Field("max")
	if !ok || v.Type() != value.STRING {
		return errors.NewSequenceError(errors.E_SEQUENCE_INVALID_DATA, fmt.Errorf("%v", av))
	}
	max, perr := strconv.ParseInt(v.ToString(), 10, 64)
	if perr != nil {
		return errors.NewSequenceError(errors.E_SEQUENCE,
			errors.NewSequenceError(errors.E_SEQUENCE_INVALID_DATA, fmt.Errorf("%v", av), perr))
	}

	v, ok = av.Field("increment")
	if !ok || v.Type() != value.STRING {
		return errors.NewSequenceError(errors.E_SEQUENCE_INVALID_DATA, fmt.Errorf("%v", av))
	}
	increment, perr := strconv.ParseInt(v.ToString(), 10, 64)
	if perr != nil {
		return errors.NewSequenceError(errors.E_SEQUENCE_INVALID_DATA, fmt.Errorf("%v", av))
	}

	v, ok = av.Field("cycle")
	if !ok || v.Type() != value.BOOLEAN {
		return errors.NewSequenceError(errors.E_SEQUENCE_INVALID_DATA, fmt.Errorf("%v", av))
	}
	cycle := v.Truth()

	seq.base = base
	seq.cache = cache
	seq.min = min
	seq.max = max
	seq.increment = increment
	seq.cycle = cycle
	seq.rev = cacheRevision
	seq.version = version
	seq.remaining = 0

	return nil
}

func (seq *sequence) validateVersion() bool {
	b, err := getSystemCollection(seq.name.Bucket())
	if err != nil {
		return false
	}
	res, err := b.SetSubDoc(seq.key, blockCmd[:2], datastore.GetDurableQueryContextFor(b))
	if err != nil {
		return false
	}
	iv := value.AsNumberValue(res[1].Value).Int64()
	return seq.version == iv
}

// expected to be locked on entry
func (seq *sequence) cacheBlock() (int64, errors.Error) {
	v, err := seq.nextAvailableBlock(seq.cache == 0)
	if err != nil {
		seq.Unlock()
		return 0, err
	}
	seq.current = v
	seq.remaining = seq.cache - 1
	// infinite recursion is prevented by min having to be smaller than max and cycling switching to the opposite end of the range
	return seq.checkCycle(v)
}

var blockCmd = value.Pairs{
	{Name: "version", Value: value.ONE_VALUE, Options: value.TRUE_VALUE},
	{Name: "version", Value: value.NEG_ONE_VALUE, Options: value.TRUE_VALUE},
	{Name: "block", Value: value.ONE_VALUE, Options: value.TRUE_VALUE},
	{Name: "block", Value: value.NEG_ONE_VALUE, Options: value.TRUE_VALUE},
}

func (seq *sequence) nextAvailableBlock(getOnly bool) (int64, errors.Error) {
	b, err := getSystemCollection(seq.name.Bucket())
	if err != nil {
		return 0, err
	}

	var res value.Pairs
	n := 3
	if getOnly {
		n++
	}
	for {
		res, err = b.SetSubDoc(seq.key, blockCmd[:n], datastore.GetDurableQueryContextFor(b))
		if err != nil {
			if err.Code() == errors.E_KEY_NOT_FOUND {
				return 0, errors.NewSequenceError(errors.E_SEQUENCE_NOT_FOUND, seq.name.SimpleString())
			}
			return 0, err
		}
		if len(res) != n || res[1].Name != blockCmd[1].Name || res[n-1].Name != blockCmd[n-1].Name {
			return 0, errors.NewSequenceError(errors.E_SEQUENCE_INVALID_DATA, fmt.Errorf("%v", seq.name.SimpleString()))
		}
		iv := value.AsNumberValue(res[1].Value).Int64()
		if seq.version != iv {
			err = seq.load()
			if err != nil {
				return 0, err
			}
			if iv != seq.version {
				continue
			}
		}
		break
	}
	block := value.AsNumberValue(res[n-1].Value).Int64()
	if !getOnly {
		block--
	}
	rv := (block * int64(seq.cache) * seq.increment) + seq.base
	if !getOnly && !seq.cycle {
		prev := ((block - 1) * int64(seq.cache) * seq.increment) + seq.base
		if (seq.increment < 0 && prev < rv) || (seq.increment > 0 && prev > rv) {
			return 0, errors.NewSequenceError(errors.E_SEQUENCE_EXHAUSTED, seq.name.SimpleString())
		}
	}
	return rv, nil
}

func (seq *sequence) checkCycle(prev int64) (int64, errors.Error) {
	if seq.cycle {
		if seq.increment < 0 && seq.current < seq.min {
			return seq.cycleAndCache(seq.max)
		} else if seq.increment > 0 && seq.current > seq.max {
			return seq.cycleAndCache(seq.min)
		}
	} else {
		if (seq.increment < 0 && (seq.current < seq.min || prev < seq.current)) ||
			(seq.increment > 0 && (seq.current > seq.max || prev > seq.current)) {

			seq.current = prev // undo the change that failed
			seq.Unlock()
			return 0, errors.NewSequenceError(errors.E_SEQUENCE_EXHAUSTED, seq.name.SimpleString())
		}
	}
	rv := seq.current
	seq.Unlock()
	return rv, nil
}

var cycleCmd = value.Pairs{
	{Name: "version", Value: value.ONE_VALUE, Options: value.TRUE_VALUE},
	{Name: "block", Value: value.ZERO_VALUE},
	{Name: "base", Value: nil},
}

func (seq *sequence) cycleAndCache(newValue int64) (int64, errors.Error) {
	seq.current = 0
	seq.remaining = 0
	seq.base = newValue

	b, err := getSystemCollection(seq.name.Bucket())
	if err != nil {
		seq.Unlock()
		return 0, err
	}

	var res value.Pairs
	vals := make(value.Pairs, 3)
	copy(vals, cycleCmd)
	vals[2].Value = value.NewValue(fmt.Sprintf("%v", seq.base))
	res, err = b.SetSubDoc(seq.key, vals, datastore.GetDurableQueryContextFor(b))
	if err != nil {
		seq.Unlock()
		if err.Code() == errors.E_KEY_NOT_FOUND {
			return 0, errors.NewSequenceError(errors.E_SEQUENCE_NOT_FOUND, seq.name.SimpleString())
		}
		return 0, err
	}
	if len(res) < 1 || res[0].Name != vals[0].Name {
		seq.Unlock()
		return 0, errors.NewSequenceError(errors.E_SEQUENCE_INVALID_DATA, fmt.Errorf("%v", seq.name.SimpleString()))
	}
	seq.version = value.AsNumberValue(res[0].Value).Int64()

	seq.rev = nextRevision()
	return seq.cacheBlock()
}

func sequenceChangeMonitor(kve metakv.KVEntry) error {
	if kve.Path != _CACHE_REVISION {
		return nil
	}
	node, _ := distributed.RemoteAccess().SplitKey(string(kve.Value))
	logging.Debugf("%v (I am: %v)", string(node), distributed.RemoteAccess().WhoAmI())
	if node == "" || node != distributed.RemoteAccess().WhoAmI() {
		atomic.AddInt32(&cacheRevision, 1)
	}
	return nil
}

func nextRevision() int32 {
	atomic.AddInt32(&cacheRevision, 1)
	err := metakv.Set(_CACHE_REVISION, fmtCacheRevision(), nil)
	if err != nil && err.Error() == "Not found" {
		err = metakv.Add(_CACHE_REVISION, fmtCacheRevision())
	}
	if err != nil {
		logging.Infof("Unable to update sequences cache monitor %v", errors.NewMetaKVChangeCounterError(err))
	}
	return cacheRevision
}

func fmtCacheRevision() []byte {
	return []byte(distributed.RemoteAccess().MakeKey(distributed.RemoteAccess().WhoAmI(), strconv.Itoa(int(cacheRevision))))
}

func validateWithOptions(with value.Value, valid []string) error {
	for k, _ := range with.Fields() {
		found := false
		for _, v := range valid {
			if k == v {
				found = true
				break
			}
		}
		if !found {
			return errors.NewWithInvalidOptionError(k)
		}
	}
	return nil
}

func getWithOption(with value.Value, opt string, optional bool) (int64, bool, error) {
	v, found := with.Field(opt)
	if !found || v.Type() != value.NUMBER {
		if !found && optional {
			return 0, false, nil
		}
		return 0, true, errors.NewWithInvalidValueError(opt)
	}
	i, ok := value.IsIntValue(v)
	if !ok {
		return 0, true, errors.NewWithInvalidValueError(opt)
	}
	return int64(i), true, nil
}

func getWithBoolOption(with value.Value, opt string, optional bool) (bool, bool, error) {
	v, found := with.Field(opt)
	if !found || v.Type() != value.BOOLEAN {
		if !found && optional {
			return false, false, nil
		}
		return false, true, errors.NewWithInvalidValueError(opt)
	}
	return v.Truth(), true, nil
}

// As we latch items whilst walking the lists for clean-up, we need to do the walking serially
var listWalkMutex sync.Mutex

// to support system keyspace
func ListSequenceKeys(namespace string, bucket string, scope string, cachedOnly bool, limit int64) ([]string, errors.Error) {

	if limit <= 0 {
		return nil, nil
	}

	filter := getCacheKey(namespace, bucket, scope) + "."

	res := make([]string, 0, 32)

	if cachedOnly {
		listWalkMutex.Lock()
		sequences.ForEach(func(k string, s interface{}) bool {
			elements := algebra.ParsePath(k)
			if strings.HasPrefix(k, filter) && len(elements) == 4 {
				res = append(res, k)
				limit--
			}
			return limit > 0
		}, nil)
		listWalkMutex.Unlock()
		return res, nil
	}

	prefix := _SEQUENCE
	if scope != "" {
		path := algebra.NewPathFromElements([]string{namespace, bucket, scope})
		err, uid := validateScope(path)
		if err == nil {
			prefix += uid + "::" + scope + "."
		}
	}

	datastore.ScanSystemCollection(bucket, prefix,
		func(systemCollection datastore.Keyspace) errors.Error {
			if systemCollection.ScopeId() == scope {
				// don't look for _system scope sequences
				return errors.NewSequenceError(errors.E_SEQUENCE) // will just stop the scan
			}
			return nil
		},
		func(key string, systemCollection datastore.Keyspace) errors.Error {
			if limit > 0 {
				res = append(res, getCacheKey(namespace, bucket, key))
				limit--
				return nil
			}
			return errors.NewSequenceError(errors.E_SEQUENCE) // will just stop the scan
		}, nil)

	return res, nil
}

func FetchSequence(name string, cacheOnly bool) (value.AnnotatedValue, errors.Error) {

	elements := algebra.ParsePath(name)
	if len(elements) < 3 {
		return nil, errors.NewSequenceError(errors.E_SEQUENCE_INVALID_NAME, name)
	}
	_, err := datastore.GetScope(elements[0:3]...)
	if err != nil {
		return nil, err
	}
	path := algebra.NewPathFromElements(elements)

	val := make(map[string]interface{})
	cache := int64(0)
	base := int64(0)
	min := int64(math.MinInt64)
	max := int64(math.MaxInt64)
	increment := int64(1)
	cycle := false

	s := sequences.Get(name, nil)
	if s != nil {
		seq := s.(*sequence)
		base = seq.base
		cache = int64(seq.cache)
		min = seq.min
		max = seq.max
		increment = seq.increment
		cycle = seq.cycle
		if seq.remaining > 0 || (cache == 1 && seq.current > base) {
			val[distributed.RemoteAccess().NodeUUID(distributed.RemoteAccess().WhoAmI())] = seq.current
		} else {
			val[distributed.RemoteAccess().NodeUUID(distributed.RemoteAccess().WhoAmI())] = nil
		}
	} else if cacheOnly {
		return nil, nil
	}

	m := make(map[string]interface{})
	m["namespace"] = elements[0]
	m["namespace_id"] = elements[0]
	m["bucket"] = elements[1]
	m["scope_id"] = elements[2]
	m["name"] = elements[3]
	m["path"] = path.ProtectedString()

	if !cacheOnly {
		b, err := getSystemCollection(elements[1])
		if err != nil {
			return nil, err
		}

		res := make(map[string]value.AnnotatedValue, 1)
		keys := make([]string, 1)
		keys[0] = getStorageKey(path)
		errs := b.Fetch(keys, res, datastore.NULL_QUERY_CONTEXT, nil, nil, false)
		if errs != nil && len(errs) > 0 {
			if !errors.IsNotFoundError("", errs[0]) && !errs[0].HasCause(errors.E_CB_BULK_GET) {
				return nil, errs[0]
			}
			return nil, nil
		}

		av, ok := res[keys[0]]
		if !ok {
			return nil, errors.NewSequenceError(errors.E_SEQUENCE_NOT_FOUND, name)
		}
		v, _ := av.Field("cache")
		cache, _ = strconv.ParseInt(v.ToString(), 10, 64)
		v, _ = av.Field("base")
		base, _ = strconv.ParseInt(v.ToString(), 10, 64)
		v, _ = av.Field("min")
		min, _ = strconv.ParseInt(v.ToString(), 10, 64)
		v, _ = av.Field("max")
		max, _ = strconv.ParseInt(v.ToString(), 10, 64)
		v, _ = av.Field("increment")
		increment, _ = strconv.ParseInt(v.ToString(), 10, 64)
		v, _ = av.Field("cycle")
		cycle = v.Truth()
		v, _ = av.Field("block")
		val["~next_block"] = value.AsNumberValue(v).Int64()*cache*increment + base
	}

	m["value"] = val
	m["cache"] = cache
	m["min"] = min
	m["max"] = max
	m["increment"] = increment
	m["cycle"] = cycle

	return value.NewAnnotatedValue(value.NewValue(m)), nil
}

func ListCachedSequences() []string {
	res := make([]string, 0, 32)
	listWalkMutex.Lock()
	sequences.ForEach(func(k string, s interface{}) bool {
		res = append(res, k)
		return true
	}, nil)
	listWalkMutex.Unlock()
	return res
}

func BackupSequences(namespace string, bucket string, filter func(string) bool) ([]interface{}, errors.Error) {

	target := make([]interface{}, 0, 16)
	res := make(map[string]value.AnnotatedValue, 1)
	keys := make([]string, 1)

	namePrefix := namespace + ":" + bucket + "."

	err := datastore.ScanSystemCollection(bucket, _SEQUENCE, nil,
		func(key string, systemCollection datastore.Keyspace) errors.Error {
			name := namePrefix + trimPrefixAndScopeUid(key)
			if filter == nil || filter(name) {
				keys[0] = key
				errs := systemCollection.Fetch(keys, res, datastore.NULL_QUERY_CONTEXT, nil, nil, false)
				if errs != nil && len(errs) > 0 {
					return errs[0]
				}
				av, ok := res[keys[0]]
				if ok {
					m := make(map[string]interface{})
					m["identity"] = name
					v, _ := av.Field("cache")
					m["cache"] = v.ToString()
					v, _ = av.Field("min")
					m["min"] = v.ToString()
					v, _ = av.Field("max")
					m["max"] = v.ToString()
					v, _ = av.Field("increment")
					m["increment"] = v.ToString()
					v, _ = av.Field("cycle")
					m["cycle"] = v.Truth()
					v, _ = av.Field("initial")
					m["initial"] = v.ToString()

					start, _ := restartValue(av)
					m["start"] = start

					target = append(target, m)
				}
			}
			return nil
		}, nil)

	return target, err
}

func CleanupCacheEntry(namespace string, bucket string, key string) {
	sequences.Delete(getCacheKey(namespace, bucket, key), nil)
}

func ValidateCreateSequenceOption(m map[string]interface{}, optionName string, optionValue value.Value) string {

	if optionValue == nil {
		return "invalid option value"
	}

	if optionName == "with" {
		if len(m) != 0 {
			return "WITH may not be used with other options"
		} else if optionValue.Type() == value.OBJECT {
			m["with"] = optionValue
			return ""
		} else {
			return "invalid option value"
		}
	} else {
		if _, ok := m["with"]; ok {
			return "options may not be used with WITH clause"
		}
	}
	if _, ok := m[optionName]; ok {
		return "duplicate option"
	}

	if optionName == OPT_CYCLE {
		if optionValue.Type() == value.BOOLEAN {
			m[optionName] = optionValue.Truth()
			return ""
		} else {
			return "invalid option value"
		}
	} else if optionValue.Type() == value.NUMBER {
		if i, ok := value.IsIntValue(optionValue); ok {
			m[optionName] = i
			return ""
		}
	}
	return "invalid option value"
}

func ValidateAlterSequenceOption(m map[string]interface{}, optionName string, optionValue value.Value) string {

	if optionValue == nil {
		return "invalid option value"
	}
	if _, ok := m[optionName]; ok {
		return "duplicate option"
	}
	if optionName == OPT_CYCLE {
		if optionValue.Type() == value.BOOLEAN {
			m[optionName] = optionValue.Truth()
			return ""
		} else {
			return "invalid option value"
		}
	} else if optionName == OPT_RESTART && optionValue.Type() == value.NULL {
		m[optionName] = true
		return ""
	} else if optionValue.Type() == value.NUMBER {
		if i, ok := value.IsIntValue(optionValue); ok {
			m[optionName] = i
			return ""
		}
	}
	return "invalid option value"
}
