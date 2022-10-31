// Copyright 2019-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
// file, in accordance with the Business Source License, use of this software
// will be governed by the Apache License, Version 2.0, included in the file
// licenses/APL2.txt.

package inferencer

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/primitives/couchbase"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/value"
)

const _EMPTY_KEY = ""
const _KEYS_NOT_FOUND = 5
const _MAX_DUPLICATES = 100
const _RANDOM_THRESHOLD = 0.75
const _MAX_INDEXES_TRIED_PER_DOC = 10
const _MAX_NUM_SCANS = 20000
const _MAX_SAMPLE_SIZE = 1000000
const _FETCH_TIMEOUT = time.Second * 10

type Flag int

const (
	NO_FLAGS     Flag = iota
	SINGLE_INDEX Flag = 1 << iota
	LIMIT_2_INDEXES
	LIMIT_5_INDEXES
	NO_RANDOM_ENTRY
	NO_PRIMARY_INDEX
	NO_SECONDARY_INDEX
	NO_RANDOM_INDEX_SAMPLE
	ALLOW_DUPLICATED_LEADING_KEY
	ALLOW_ARRAY_INDEXES
	NO_LIMIT_RANDOM
	ALLOW_CONDITIONAL
	ALLOW_SUPERSET_CONDITIONS
	CACHE_KEYS
	RANDOM_ENTRY_LAST
	NO_RANDOM_SCAN
	SAMPLE_ALL_DOCS
	SAMPLE_ALLOW_EXTRA
	FULL_SCAN
	ALLOW_DUPS
)

var flags_map = map[string]Flag{
	"no_flags":                     NO_FLAGS,
	"single_index":                 SINGLE_INDEX,
	"limit_2_indexes":              LIMIT_2_INDEXES,
	"limit_5_indexes":              LIMIT_5_INDEXES,
	"no_random_entry":              NO_RANDOM_ENTRY,
	"no_primary_index":             NO_PRIMARY_INDEX,
	"no_secondary_index":           NO_SECONDARY_INDEX,
	"no_random_index_sample":       NO_RANDOM_INDEX_SAMPLE,
	"allow_duplicated_leading_key": ALLOW_DUPLICATED_LEADING_KEY,
	"allow_array_indexes":          ALLOW_ARRAY_INDEXES,
	"no_limit_random":              NO_LIMIT_RANDOM,
	"allow_conditional":            ALLOW_CONDITIONAL,
	"allow_superset_conditions":    ALLOW_SUPERSET_CONDITIONS,
	"cache_keys":                   CACHE_KEYS,
	"random_entry_last":            RANDOM_ENTRY_LAST,
	"no_random_scan":               NO_RANDOM_SCAN,
	"sample_all_docs":              SAMPLE_ALL_DOCS,
	"sample_allow_extra":           SAMPLE_ALLOW_EXTRA,
	"full_scan":                    FULL_SCAN,
	"allow_dups":                   ALLOW_DUPS,
}

type DocumentRetriever interface {
	GetNextDoc(context datastore.QueryContext) (string, value.Value, errors.Error) // returns nil for value when done
	Reset()                                                                        // reset for reuse of cached results etc.
	Close()                                                                        // final clean-up to ensure any index connection is closed/cleaned-up too
}

type indexArray []datastore.Index

func (this indexArray) Len() int {
	return len(this)
}

func (this indexArray) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

func (this indexArray) Less(i, j int) bool {
	// prefer without condition
	if this[i].Condition() == nil && this[j].Condition() != nil {
		return true
	} else if this[i].Condition() != nil && this[j].Condition() == nil {
		return false
	} else if this[i].Condition() != nil && this[j].Condition() != nil {
		// prefer least selective condition
		if plannerbase.SubsetOf(this[j].Condition(), this[i].Condition()) {
			return true
		} else if plannerbase.SubsetOf(this[i].Condition(), this[j].Condition()) {
			return false
		}
	}
	// prefer without array keys
	rki := this[i].RangeKey()
	rkj := this[j].RangeKey()
	aki := false
	for _, k := range rki {
		ak, _, _ := k.IsArrayIndexKey()
		aki = aki || ak
	}
	akj := false
	for _, k := range rkj {
		ak, _, _ := k.IsArrayIndexKey()
		akj = akj || ak
	}
	if !aki && akj {
		return true
	} else if aki && !akj {
		return false
	}
	// prefer non-partitioned (for key-based restart)
	pk1, _ := this[i].(datastore.Index3).PartitionKeys()
	pk2, _ := this[j].(datastore.Index3).PartitionKeys()
	if pk1 == nil && pk2 != nil {
		return true
	} else if pk1 != nil && pk2 == nil {
		return false
	}
	// most documents
	cii := this[i].(datastore.CountIndex2)
	cij := this[j].(datastore.CountIndex2)
	nki, _ := cii.CountDistinct("retriever", nil, datastore.UNBOUNDED, nil)
	nkj, _ := cij.CountDistinct("retriever", nil, datastore.UNBOUNDED, nil)
	if nki > nkj {
		return true
	} else if nkj > nki {
		return false
	}
	// prefer fewest keys
	if len(rki) < len(rkj) {
		return true
	}
	// for consistency, lastly order by name
	if this[i].Name() < this[j].Name() {
		return true
	}
	return false
}

type UnifiedDocumentRetriever struct {
	name           string
	ks             datastore.Keyspace
	rnd            datastore.RandomEntryProvider
	lastRnd        datastore.RandomEntryProvider
	rs             datastore.RandomScanProvider
	rs_scan        interface{}
	ss             datastore.Index3
	iconn          *datastore.IndexConnection
	returned       int
	sampleSize     int
	currentIndex   int
	indexes        indexArray
	spans          datastore.Spans2
	flags          Flag
	cacheActive    bool
	cache          []string
	dedup          map[string]bool
	docs           map[string]value.AnnotatedValue
	keys           []string
	scanNum        int
	scanBlockSize  int
	scanSampleSize int
	offset         int64
	lastKeys       value.Values
}

func (udr *UnifiedDocumentRetriever) Name() string {
	return udr.name + "_retriever"
}

func (udr *UnifiedDocumentRetriever) Reset() {
	if udr.iconn != nil {
		udr.iconn.Sender().Close()
		udr.iconn = nil
	}
	if udr.rs != nil && udr.rs_scan != nil {
		udr.rs.StopKeyScan(udr.rs_scan)
		udr.rs_scan = nil
		udr.rs = nil
	}
	udr.returned = 0
	udr.currentIndex = -1
	if udr.dedup != nil {
		udr.dedup = make(map[string]bool)
	}
	udr.docs = make(map[string]value.AnnotatedValue, 1)
	udr.keys = nil
	udr.scanNum = 0
	udr.lastKeys = nil
	udr.cacheActive = udr.isFlagOn(CACHE_KEYS)
	logging.Debuga(func() string {
		if udr.cache == nil {
			return fmt.Sprintf("UDR: reset without cache (active:%v)", udr.cacheActive)
		} else {
			return fmt.Sprintf("UDR: reset with cache (active:%v) of %v keys", udr.cacheActive, len(udr.cache))
		}
	})
}

func (udr *UnifiedDocumentRetriever) isFlagOn(what Flag) bool {
	return (udr.flags & what) != 0
}

func (udr *UnifiedDocumentRetriever) isFlagOff(what Flag) bool {
	return (udr.flags & what) == 0
}

// safety net to ensure we don't leak index connections or random scans
func udrFinalizer(udr *UnifiedDocumentRetriever) {
	if udr.iconn != nil {
		logging.Warnf("UDR: Finalizer closing index connection.")
		udr.iconn.Sender().Close()
		udr.iconn = nil
	}
	if udr.rs != nil && udr.rs_scan != nil {
		logging.Warnf("UDR: Finalizer closing random scan.")
		udr.rs.StopKeyScan(udr.rs_scan)
		udr.rs_scan = nil
		udr.rs = nil
	}
}

func (udr *UnifiedDocumentRetriever) Close() {
	if udr.iconn != nil {
		udr.iconn.Sender().Close()
		udr.iconn = nil
	}
	if udr.rs != nil && udr.rs_scan != nil {
		udr.rs.StopKeyScan(udr.rs_scan)
		udr.rs_scan = nil
		udr.rs = nil
	}
	// hints for GC
	udr.dedup = nil
	udr.docs = nil
	udr.cache = nil
	udr.cacheActive = false
	udr.keys = nil
	udr.indexes = nil
	udr.lastKeys = nil
	runtime.SetFinalizer(udr, nil)
}

func MakeUnifiedDocumentRetriever(name string, context datastore.QueryContext, ks datastore.Keyspace, sampleSize int, flags Flag) (
	*UnifiedDocumentRetriever, errors.Error) {

	var errs []errors.Error

	udr := new(UnifiedDocumentRetriever)
	runtime.SetFinalizer(udr, udrFinalizer)
	udr.name = name
	udr.ks = ks
	udr.currentIndex = -1
	udr.flags = flags
	udr.scanNum = 0

	docCount, err := ks.Count(context)
	if err != nil {
		return nil, errors.NewInferKeyspaceError(ks.Name(), err)
	}

	if udr.isFlagOff(NO_LIMIT_RANDOM) {
		if float64(sampleSize) >= float64(docCount)*_RANDOM_THRESHOLD {
			udr.flags |= RANDOM_ENTRY_LAST | NO_RANDOM_INDEX_SAMPLE | SAMPLE_ALL_DOCS
			sampleSize = int(docCount)
		}
	}
	logging.Debuga(func() string {
		s := make([]rune, 0, 128)
		s = append(s, []rune("UDR: flags:")...)
		if udr.flags == 0 {
			s = append(s, []rune(" no_flags")...)
		} else {
			for k, v := range flags_map {
				if udr.flags&v != 0 {
					s = append(s, ' ')
					s = append(s, []rune(k)...)
				}
			}
		}
		return string(s)
	})

	if sampleSize <= 0 || sampleSize > int(docCount) {
		udr.sampleSize = int(docCount)
		udr.flags |= SAMPLE_ALL_DOCS
	} else {
		udr.sampleSize = sampleSize
	}
	if udr.sampleSize > _MAX_SAMPLE_SIZE {
		udr.sampleSize = _MAX_SAMPLE_SIZE
	}

	logging.Debuga(func() string { return fmt.Sprintf("UDR: sampleSize: %v", udr.sampleSize) })

	var ok bool

	methods := 0

	if udr.isFlagOff(NO_RANDOM_SCAN) {
		udr.rs, ok = ks.(datastore.RandomScanProvider)
		if ok {
			var err errors.Error
			if udr.isFlagOn(SAMPLE_ALL_DOCS) {
				udr.rs_scan, err = udr.rs.StartRandomScan(math.MaxInt, 0, int(datastore.GetScanCap()), context.KvTimeout(),
					tenant.IsServerless())
			} else {
				udr.rs_scan, err = udr.rs.StartRandomScan(udr.sampleSize, 0, int(datastore.GetScanCap()), context.KvTimeout(),
					tenant.IsServerless())
			}
			if err != nil {
				logging.Debuga(func() string { return fmt.Sprintf("UDR: random scan start failed: %v", err) })
				errs = append(errs, err)
				udr.rs = nil
			} else {
				if udr.docs == nil {
					udr.docs = make(map[string]value.AnnotatedValue, 1)
				}
			}
			methods++
		} else {
			logging.Debugf("UDR: RandomScanProvider not supported")
			errs = append(errs, errors.NewInferNoRandomScanProvider(ks.Name()))
		}
	} else {
		logging.Debugf("UDR: flags exclude random scan")
	}

	if udr.isFlagOff(NO_RANDOM_ENTRY) {
		udr.rnd, ok = ks.(datastore.RandomEntryProvider)
		if ok {
			i := 0
			for i = 0; i < _KEYS_NOT_FOUND; i++ {
				_, val, _ := udr.rnd.GetRandomEntry(context)
				if val != nil {
					break
				}
			}
			if i == _KEYS_NOT_FOUND {
				logging.Debugf("UDR: not returning random documents")
				errs = append(errs, errors.NewInferNoRandomDocuments(ks.Name()))
				udr.rnd = nil
			}
			methods++
		} else {
			logging.Debugf("UDR: RandomEntryProvider not supported")
			errs = append(errs, errors.NewInferNoRandomEntryProvider(ks.Name()))
		}
	} else {
		logging.Debugf("UDR: flags exclude random entry")
	}

	if udr.isFlagOff(NO_PRIMARY_INDEX) || udr.isFlagOff(NO_SECONDARY_INDEX) {
		indexer, err := ks.Indexer(datastore.GSI)
		if err == nil {
			udr.indexes = make(indexArray, 0, 32)

			udr.spans = append(datastore.Spans2(nil), &datastore.Span2{
				Seek: nil,
				Ranges: append(datastore.Ranges2(nil), &datastore.Range2{
					Low:       nil,
					High:      nil,
					Inclusion: datastore.BOTH,
				}),
			})

			if udr.isFlagOff(NO_PRIMARY_INDEX) {
				primaryIndexes, err := indexer.PrimaryIndexes()
				found := true
				if err == nil {
					found = false
					for _, index := range primaryIndexes {
						// make sure that the index is online
						state, _, err := index.State()
						if err != nil || state != datastore.ONLINE {
							continue
						}

						// if the Index does not implement the PrimaryIndex3 interface - like system keyspace indexes - do not consider the index
						if _, ok := index.(datastore.PrimaryIndex3); !ok {
							continue
						}
						udr.indexes = append(udr.indexes, index.(datastore.Index))
						found = true
						logging.Debuga(func() string { return fmt.Sprintf("UDR: primary index (%v) found", index.Name()) })
						// once a primary index has been picked, we won't bother with secondary indexes
						udr.flags |= NO_SECONDARY_INDEX
						methods++
						break
					}
				}
				if err != nil || !found {
					logging.Debugf("UDR: no primary index")
					errs = append(errs, errors.NewInferNoSuitablePrimaryIndex(ks.Name()))
				} else if udr.docs == nil {
					udr.docs = make(map[string]value.AnnotatedValue, 1)
				}
			} else {
				logging.Debugf("UDR: flags exclude primary")
			}

			if udr.isFlagOff(NO_SECONDARY_INDEX) {
				ilist, err := indexer.Indexes()
				found := true
				if err == nil {
					found = false
				secondary_indexes:
					for _, idx := range ilist {
						if state, _, err := idx.State(); err == nil && state == datastore.ONLINE && !idx.IsPrimary() {
							if i3, ok := idx.(datastore.Index3); ok {
								if udr.isFlagOff(ALLOW_ARRAY_INDEXES) {
									keys := i3.RangeKey()
									for _, key := range keys {
										if is, _, _ := key.IsArrayIndexKey(); is {
											continue secondary_indexes
										}
									}
								}
								if udr.isFlagOff(ALLOW_CONDITIONAL) && i3.Condition() != nil {
									continue secondary_indexes
								}
								if udr.isFlagOff(ALLOW_SUPERSET_CONDITIONS) && i3.Condition() != nil {
									for n, other := range udr.indexes {
										if other.Condition() == nil {
											continue
										}
										if plannerbase.SubsetOf(i3.Condition(), other.Condition()) {
											logging.Debuga(func() string {
												return fmt.Sprintf("UDR: excluding %v - subset of %v",
													i3.Name(), other.Name())
											})
											continue secondary_indexes
										} else if plannerbase.SubsetOf(other.Condition(), i3.Condition()) {
											logging.Debuga(func() string {
												return fmt.Sprintf("UDR: swapping secondary index %v for %v",
													i3.Name(), other.Name())
											})
											udr.indexes[n] = i3
											continue secondary_indexes
										}
									}
								}
								udr.indexes = append(udr.indexes, idx)
								logging.Debuga(func() string {
									return fmt.Sprintf("UDR: secondary index %v included", idx.Name())
								})
								methods++
								found = true
								if udr.isFlagOn(SINGLE_INDEX) {
									break
								}
							}
						}
					}
				}
				if err != nil || !found {
					logging.Debugf("UDR: no secondary index")
					errs = append(errs, errors.NewInferNoSuitableSecondaryIndex(ks.Name()))
				} else if udr.docs == nil {
					udr.docs = make(map[string]value.AnnotatedValue, 1)
				}

				if udr.isFlagOff(SINGLE_INDEX) && len(udr.indexes) > 1 {
					sort.Sort(udr.indexes)
					if udr.isFlagOff(ALLOW_DUPLICATED_LEADING_KEY) {
						leading := make(map[string]bool)
						prune := false
						for i := range udr.indexes {
							if udr.indexes[i].IsPrimary() {
								continue
							}
							lkey := udr.indexes[i].RangeKey()[0].String()
							if leading[lkey] {
								logging.Debuga(func() string {
									return fmt.Sprintf("UDR: %v excluded - duplicate leading key",
										udr.indexes[i].Name())
								})
								udr.indexes[i] = nil
								prune = true
							} else {
								leading[lkey] = true
							}
						}
						if prune {
							temp := make(indexArray, 0, len(udr.indexes))
							for _, index := range udr.indexes {
								if index != nil {
									temp = append(temp, index)
								}
							}
							udr.indexes = temp
						}
					}
					if udr.isFlagOn(LIMIT_5_INDEXES) && len(udr.indexes) > 5 {
						udr.indexes = udr.indexes[:5]
					}

					if udr.isFlagOn(LIMIT_2_INDEXES) && len(udr.indexes) > 2 {
						udr.indexes = udr.indexes[:2]
					}
					logging.Debuga(func() string {
						s := "Ranked index list: "
						for i, idx := range udr.indexes {
							s += fmt.Sprintf("[%v] %v,", i, idx.Name())
						}
						s = s[:len(s)-1]
						return s
					})
				}
			} else {
				logging.Debugf("UDR: flags or primary exclude secondary")
			}

			logging.Debuga(func() string {
				return fmt.Sprintf("UDR: rs: %v rnd: %v idxs: %v", udr.rs != nil, udr.rnd != nil,
					len(udr.indexes))
			})
		} else {
			errs = append(errs, errors.NewInferKeyspaceError(ks.Name(), err))
		}
	}

	if udr.isFlagOn(FULL_SCAN) {
		indexer, err := ks.Indexer(datastore.SEQ_SCAN)
		if err == nil {
			if udr.spans == nil {
				udr.spans = append(datastore.Spans2(nil), &datastore.Span2{
					Seek: nil,
					Ranges: append(datastore.Ranges2(nil), &datastore.Range2{
						Low:       nil,
						High:      nil,
						Inclusion: datastore.BOTH,
					}),
				})
			}

			seqScan, err := indexer.PrimaryIndexes()
			found := true
			if err == nil {
				found = false
				for _, index := range seqScan {
					// make sure that scanning it is online
					state, _, err := index.State()
					if err != nil || state != datastore.ONLINE {
						continue
					}

					if _, ok := index.(datastore.PrimaryIndex3); !ok {
						continue
					}
					udr.ss = index.(datastore.Index3)
					methods++
					found = true
					logging.Debuga(func() string { return fmt.Sprintf("UDR: sequential scan found for %v", ks.Name()) })
					break
				}
			}
			if err != nil || !found {
				logging.Debugf("UDR: no sequential scan")
				errs = append(errs, errors.NewInferNoSequentialScan(ks.Name()))
			} else if udr.docs == nil {
				udr.docs = make(map[string]value.AnnotatedValue, 1)
			}
		} else {
			logging.Debugf("UDR: no sequential scan indexer")
		}
	}

	if udr.rs == nil && udr.rnd == nil && len(udr.indexes) == 0 && udr.ss == nil {
		if len(errs) == 0 {
			errs = append(errs, errors.NewInferNoRetrievers(ks.Name()))
		}
		return nil, errors.NewInferCreateRetrieverFailed(errs...)
	}

	logging.Debuga(func() string {
		s := ""
		if udr.rs != nil {
			s += ",random scan"
		}
		if udr.rnd != nil {
			s += ",random entry"
		}
		if len(udr.indexes) > 0 {
			s += ",index"
		}
		if udr.ss != nil {
			s += ",sequential scan"
		}
		return "UDR: retriever(s): " + s[1:]
	})

	if udr.isFlagOff(ALLOW_DUPS) && (methods > 1 || udr.rnd != nil) {
		udr.dedup = make(map[string]bool)
	}

	return udr, nil
}

func (udr *UnifiedDocumentRetriever) GetNextDoc(context datastore.QueryContext) (string, value.Value, errors.Error) {
	if udr.returned >= udr.sampleSize && (udr.rs == nil || udr.cacheActive ||
		(udr.rs != nil && udr.isFlagOff(SAMPLE_ALLOW_EXTRA))) {

		if udr.iconn != nil {
			udr.iconn.Sender().Close()
			udr.iconn = nil
		}
		if udr.rs != nil && udr.rs_scan != nil {
			ru, err := udr.rs.StopKeyScan(udr.rs_scan)
			if ru > 0 {
				context.RecordKvRU(tenant.Unit(ru))
			}
			udr.rs_scan = nil
			return _EMPTY_KEY, nil, err
		}
		return _EMPTY_KEY, nil, nil
	}

	// if we have cached keys (and this has been reset) just use them
	if udr.cacheActive {
		if udr.returned >= len(udr.cache) {
			return _EMPTY_KEY, nil, nil
		}

		for udr.returned < len(udr.cache) {
			errs := udr.ks.Fetch(udr.cache[udr.returned:udr.returned+1], udr.docs, context, nil)
			if errs != nil {
				return _EMPTY_KEY, nil, errs[0]
			} else if len(udr.docs) != 0 {
				break
			}
			udr.returned++
		}

		if udr.returned >= len(udr.cache) {
			return _EMPTY_KEY, nil, nil
		}

		k := udr.cache[udr.returned]
		udr.returned++
		defer func() { delete(udr.docs, k) }()
		return k, udr.docs[k], nil
	}

	if udr.rs != nil {
		if udr.returned == 0 {
			logging.Debugf("UDR: retrieving using random scan")
		}
		k, v, e, cont := udr.getNextRandomScan(context)
		if v != nil || e != nil || !cont {
			return k, v, e
		}
		logging.Debuga(func() string {
			return "UDR: random scan exhausted"
		})
		udr.rs = nil
	}

	if udr.rnd != nil && udr.isFlagOff(RANDOM_ENTRY_LAST) {
		if udr.returned == 0 {
			logging.Debugf("UDR: retrieving using random entry")
		}
		k, v, e := udr.getRandom(context)
		if v != nil || e != nil {
			return k, v, e
		}
		logging.Debuga(func() string {
			return "UDR: random retriever exhausted"
		})
		udr.rnd = nil
	}

	if len(udr.indexes) > 0 {
		k, v, e, cont := udr.getNextIndexScan(context)
		if v != nil || e != nil || !cont {
			return k, v, e
		}
		logging.Debuga(func() string {
			return "UDR: indexes exhausted"
		})
	}

	if udr.ss != nil {
		if udr.returned == 0 {
			logging.Debugf("UDR: retrieving using full scan")
		}
		k, v, e, cont := udr.getNextFullScan(context)
		if v != nil || e != nil || !cont {
			return k, v, e
		}
		logging.Debuga(func() string {
			return "UDR: full scan exhausted"
		})
	}

	if udr.iconn != nil {
		udr.iconn.Sender().Close()
		udr.iconn = nil
	}
	if udr.isFlagOn(RANDOM_ENTRY_LAST) && udr.rnd != nil {
		udr.flags &^= RANDOM_ENTRY_LAST
		return udr.getRandom(context)
	}
	return _EMPTY_KEY, nil, nil
}

func (udr *UnifiedDocumentRetriever) getRandom(context datastore.QueryContext) (string, value.Value, errors.Error) {
	duplicates := 0
	for duplicates < _MAX_DUPLICATES {
		key, value, err := udr.rnd.GetRandomEntry(context)
		if err != nil {
			logging.Debuga(func() string { return fmt.Sprintf("UDR: random retriever error: %v", err) })
			return _EMPTY_KEY, nil, errors.NewInferRandomError(err)
		}

		if value == nil || (udr.dedup != nil && udr.dedup[key]) {
			duplicates++
			continue
		}

		if udr.dedup != nil {
			udr.dedup[key] = true
		}
		udr.returned++
		udr.cacheKey(key)
		return key, value, nil
	}
	logging.Debugf("UDR: maximum random duplicates reached")
	return _EMPTY_KEY, nil, nil
}

func (udr *UnifiedDocumentRetriever) getNextRandomScan(context datastore.QueryContext) (string, value.Value, errors.Error, bool) {
	for {
		if len(udr.keys) == 0 {
			var err errors.Error
			udr.keys, err, _ = udr.rs.FetchKeys(udr.rs_scan, _FETCH_TIMEOUT)
			if err != nil {
				if udr.returned == 0 && (udr.rnd != nil || len(udr.indexes) > 0 || udr.ss != nil) {
					logging.Debuga(func() string {
						return fmt.Sprintf("UDR: random scan fetch failed (%v) trying other methods", err)
					})
					udr.rs.StopKeyScan(udr.rs_scan)
					udr.rs = nil
					udr.rs_scan = nil
					return _EMPTY_KEY, nil, err, true
				}
				return _EMPTY_KEY, nil, err, false
			}
			logging.Debuga(func() string { return fmt.Sprintf("UDR: fetched %d keys from random scan", len(udr.keys)) })
		}
		if len(udr.keys) == 0 {
			ru, err := udr.rs.StopKeyScan(udr.rs_scan)
			if ru > 0 {
				context.RecordKvRU(tenant.Unit(ru))
			}
			udr.rs_scan = nil
			if udr.returned < udr.sampleSize {
				logging.Debuga(func() string {
					return fmt.Sprintf("UDR: random scan stopped short (%v) of sample size (%v)", udr.returned, udr.sampleSize)
				})
				udr.rs = nil
				return _EMPTY_KEY, nil, err, true
			}
			return _EMPTY_KEY, nil, err, false
		}
		for len(udr.keys) > 0 {
			if udr.dedup == nil || !udr.dedup[udr.keys[0]] {
				if udr.dedup != nil {
					udr.dedup[udr.keys[0]] = true
				}
				errs := udr.ks.Fetch(udr.keys[:1], udr.docs, context, nil)
				if errs != nil {
					return _EMPTY_KEY, nil, errs[0], false
				} else if len(udr.docs) != 0 {
					break
				}
			}
			udr.keys = udr.keys[1:]
		}
		if len(udr.keys) == 0 {
			continue
		}
		k := udr.keys[0]
		udr.cacheKey(k)
		udr.returned++
		defer func() { delete(udr.docs, k); udr.keys = udr.keys[1:] }()
		return k, udr.docs[k], nil, false
	}
}

func (udr *UnifiedDocumentRetriever) getNextFullScan(context datastore.QueryContext) (string, value.Value, errors.Error, bool) {
	if len(udr.keys) == 0 {
		if udr.iconn == nil {
			udr.iconn = datastore.NewIndexConnection(datastore.NULL_CONTEXT)
			udr.iconn.SetSkipMetering(true)
			proj := &datastore.IndexProjection{PrimaryKey: true}
			go udr.ss.Scan3(udr.Name(), nil, false, false, proj, 0, int64(math.MaxInt64), nil, nil,
				datastore.UNBOUNDED, nil, udr.iconn)

			logging.Debuga(func() string {
				return fmt.Sprintf("UDR: scanning %v (scan: %v)", udr.ss.Name(), udr.scanNum)
			})
		}
		docCount, err := udr.ks.Count(context)
		if err != nil {
			udr.iconn.Sender().Close()
			udr.iconn = nil
			udr.ss = nil
			return _EMPTY_KEY, nil, err, false
		}
		block := docCount / int64(udr.sampleSize)
		berr := docCount % int64(udr.sampleSize)
		if block < 1 {
			block = 1
		}
		n := block
		ec := int64(0)
		sel := int64(rand.Int()) % block
		for len(udr.keys) < udr.sampleSize-udr.returned {
			entry, _ := udr.iconn.Sender().GetEntry()
			if entry == nil {
				break
			}
			n--
			if n == sel {
				if udr.dedup == nil || !udr.dedup[entry.PrimaryKey] {
					if udr.dedup != nil {
						udr.dedup[udr.keys[0]] = true
					}
					udr.keys = append(udr.keys, entry.PrimaryKey)
				} else if n > 0 {
					// try pick another from the remaining keys in the block
					sel = int64(rand.Int()) % n
				}
			}
			if n <= 0 {
				ec += berr
				if ec >= int64(udr.sampleSize) {
					n = block + 1
					ec -= int64(udr.sampleSize)
				} else {
					n = block
				}
				sel = int64(rand.Int()) % n
			}
		}
		udr.iconn.Sender().Close()
		udr.iconn = nil
		logging.Debuga(func() string {
			return fmt.Sprintf("UDR: sequential scan returned %v keys", len(udr.keys))
		})
	}

	for len(udr.keys) > 0 {
		errs := udr.ks.Fetch(udr.keys[0:1], udr.docs, context, nil)

		if errs != nil {
			return _EMPTY_KEY, nil, errs[0], false
		} else if len(udr.docs) > 0 {
			break
		}
		udr.keys = udr.keys[1:]
	}

	if len(udr.keys) == 0 {
		if udr.returned < udr.sampleSize {
			return _EMPTY_KEY, nil, nil, true
		} else {
			return _EMPTY_KEY, nil, nil, false
		}
	}

	udr.returned++
	defer func() { delete(udr.docs, udr.keys[0]); udr.keys = udr.keys[1:] }()
	udr.cacheKey(udr.keys[0])
	if len(udr.keys) == 1 && udr.returned < udr.sampleSize {
		udr.ss = nil
	}
	return udr.keys[0], udr.docs[udr.keys[0]], nil, false
}

func (udr *UnifiedDocumentRetriever) getNextIndexScan(context datastore.QueryContext) (string, value.Value, errors.Error, bool) {
next_index:
	for indexesTried := 0; indexesTried < _MAX_INDEXES_TRIED_PER_DOC; {
		duplicates := 0
		if udr.iconn == nil && len(udr.keys) == 0 {
			udr.currentIndex++
			if udr.currentIndex >= len(udr.indexes) || (udr.currentIndex > 0 && udr.isFlagOn(SINGLE_INDEX)) {
				if udr.iconn != nil {
					udr.iconn.Sender().Close()
					udr.iconn = nil
				}
				logging.Debuga(func() string {
					ci := udr.currentIndex
					if ci > len(udr.indexes) {
						ci = len(udr.indexes)
					}
					return fmt.Sprintf("UDR: ending index scanning after %v index(es), %v docs returned",
						ci, udr.returned)
				})
				return _EMPTY_KEY, nil, nil, true
			}

			start := int64(0)
			if udr.scanNum > 0 {
				start = int64(udr.scanBlockSize) - (udr.offset % int64(udr.scanBlockSize))
				start %= int64(udr.scanBlockSize)
			} else {
				udr.lastKeys = nil
				udr.spans[0].Ranges[0].Low = nil
				udr.spans[0].Ranges[0].Inclusion = datastore.BOTH
				udr.spans[0].Ranges = udr.spans[0].Ranges[:1]

				// set-up for index-based options
				remainingSampleSize := udr.sampleSize - udr.returned
				udr.scanSampleSize = 1
				numScans := _MAX_NUM_SCANS
				if numScans > remainingSampleSize || !udr.indexes[udr.currentIndex].IsPrimary() {
					numScans = remainingSampleSize
					if numScans <= 0 {
						numScans = 1
					}
				} else {
					udr.scanSampleSize = (remainingSampleSize + (numScans - 1)) / numScans
				}
				// break the number of keys down into blocks within which the samples can be randomly picked
				// this is to try ensure more even distribution of sampling across the key range
				ci := udr.indexes[udr.currentIndex].(datastore.CountIndex2)
				nk, err := ci.CountDistinct(udr.Name(), nil, datastore.UNBOUNDED, nil)
				if err != nil {
					docCount, err := udr.ks.Count(context)
					if err != nil {
						return _EMPTY_KEY, nil, err, false
					}
					udr.scanBlockSize = int(docCount / int64(numScans))
				} else {
					udr.scanBlockSize = int(nk / int64(numScans))
				}
				if udr.scanBlockSize < udr.scanSampleSize {
					udr.scanBlockSize = udr.scanSampleSize
				}
				logging.Debuga(func() string {
					return fmt.Sprintf("UDR: %v: ss-size: %v, sb-size: %v, remaining samples: %v",
						udr.indexes[udr.currentIndex].Name(), udr.scanSampleSize, udr.scanBlockSize, remainingSampleSize)
				})
			}

			udr.offset = 0
			if udr.isFlagOff(NO_RANDOM_INDEX_SAMPLE) && udr.scanBlockSize > udr.scanSampleSize {
				// if the scan block size is greater than the index count, use the index count so we get at least 1 sample from it
				ci := udr.indexes[udr.currentIndex].(datastore.CountIndex2)
				count, err := ci.CountDistinct(udr.Name(), nil, datastore.UNBOUNDED, nil)
				if err == nil && int64(udr.scanBlockSize) > count {
					if int(count) > udr.scanSampleSize {
						udr.offset = int64(rand.Int() % (int(count) - udr.scanSampleSize))
					}
				} else {
					udr.offset = int64(rand.Int() % (udr.scanBlockSize - udr.scanSampleSize))
				}
			}
			start += udr.offset
			udr.restartScan(start)
		}

		if len(udr.keys) == 0 {
			for len(udr.keys) < udr.scanSampleSize {
				entry, _ := udr.iconn.Sender().GetEntry()
				if entry == nil {
					timeout := udr.iconn.Timeout()
					udr.iconn.Sender().Close()
					udr.iconn = nil
					if timeout {
						if len(udr.keys) > 0 && udr.indexes[udr.currentIndex].IsPrimary() {
							udr.spans[0].Ranges[0].Low = value.NewValue(udr.keys[len(udr.keys)-1])
							udr.spans[0].Ranges[0].Inclusion = datastore.HIGH
							udr.restartScan(0)
							continue
						} else if !udr.indexes[udr.currentIndex].IsPrimary() && udr.lastKeys != nil {
							udr.restartAfterLastKey()
							continue
						}
					}
					if len(udr.keys) == 0 {
						udr.scanNum = 0
						indexesTried++
						continue next_index
					} else {
						break
					}
				}
				udr.lastKeys = entry.EntryKey
				udr.offset++
				if udr.dedup == nil || !udr.dedup[entry.PrimaryKey] {
					if udr.dedup != nil {
						udr.dedup[entry.PrimaryKey] = true
					}
					udr.keys = append(udr.keys, entry.PrimaryKey)
				} else {
					duplicates++
					if duplicates > _MAX_DUPLICATES {
						udr.iconn.Sender().Close()
						udr.iconn = nil
						if len(udr.keys) == 0 {
							udr.scanNum = 0
							indexesTried++
							continue next_index
						} else {
							break
						}
					}
				}
			}
			if len(udr.keys) == 0 {
				indexesTried++
				udr.scanNum = 0
				continue next_index
			}
			if udr.indexes[udr.currentIndex].IsPrimary() {
				if udr.iconn != nil {
					// repeat this index with a different offset
					udr.iconn.Sender().Close()
					udr.iconn = nil
					udr.scanNum++
					if len(udr.keys) > 0 {
						udr.spans[0].Ranges[0].Low = value.NewValue(udr.keys[len(udr.keys)-1])
						udr.spans[0].Ranges[0].Inclusion = datastore.HIGH
					}
					udr.currentIndex--
				} else {
					udr.scanNum = 0
				}
			} else {
				if udr.iconn != nil {
					// read and discard keys to provide random sampling
					skip := int64(udr.scanBlockSize) - (udr.offset % int64(udr.scanBlockSize))
					skip %= int64(udr.scanBlockSize)
					if udr.isFlagOff(NO_RANDOM_INDEX_SAMPLE) && udr.scanBlockSize > udr.scanSampleSize {
						skip += int64(rand.Int() % (udr.scanBlockSize - udr.scanSampleSize))
					}
					for ; skip > 0; skip-- {
						entry, _ := udr.iconn.Sender().GetEntry()
						if entry == nil {
							timeout := udr.iconn.Timeout()
							udr.iconn.Sender().Close()
							udr.iconn = nil
							if timeout && udr.lastKeys != nil {
								udr.restartAfterLastKey()
							}
							break
						}
						udr.lastKeys = entry.EntryKey
						udr.offset++
					}
				}
			}
		}

		for len(udr.keys) > 0 {
			errs := udr.ks.Fetch(udr.keys[0:1], udr.docs, context, nil)
			if errs != nil {
				return _EMPTY_KEY, nil, errs[0], false
			} else if len(udr.docs) != 0 {
				break
			}
			udr.keys = udr.keys[1:]
		}
		if len(udr.keys) == 0 {
			continue
		}

		udr.returned++
		defer func() { delete(udr.docs, udr.keys[0]); udr.keys = udr.keys[1:] }()
		udr.cacheKey(udr.keys[0])
		return udr.keys[0], udr.docs[udr.keys[0]], nil, false
	}
	return _EMPTY_KEY, nil, nil, true
}

func (udr *UnifiedDocumentRetriever) restartScan(offset int64) {
	udr.iconn = datastore.NewIndexConnection(datastore.NULL_CONTEXT)
	udr.iconn.SetSkipMetering(true)
	index := udr.indexes[udr.currentIndex].(datastore.Index3)
	proj := &datastore.IndexProjection{PrimaryKey: true}
	ss := int64(math.MaxInt64)
	udr.iconn.SetPrimary() // always set as primary so we can trap timeouts
	if index.IsPrimary() {
		ss = int64(udr.scanSampleSize)
	} else {
		proj.EntryKeys = make([]int, len(index.RangeKey()))
		for i := range proj.EntryKeys {
			proj.EntryKeys[i] = i
		}
	}
	go index.Scan3(udr.Name(), udr.spans, false, false, proj, offset, ss, nil, nil, datastore.UNBOUNDED, nil, udr.iconn)

	logging.Debuga(func() string {
		return fmt.Sprintf("UDR: scanning index %v (scan: %v, offset: %v, low: %v (first of %d))",
			udr.indexes[udr.currentIndex].Name(), udr.scanNum, offset, udr.spans[0].Ranges[0].Low, len(udr.spans[0].Ranges))
	})
}

func (udr *UnifiedDocumentRetriever) restartAfterLastKey() {
	// reset the block calculation
	udr.offset = 0

	if len(udr.spans[0].Ranges) != len(udr.lastKeys) {
		udr.spans[0].Ranges = make(datastore.Ranges2, len(udr.lastKeys))
	}
	for i := range udr.spans[0].Ranges {
		udr.spans[0].Ranges[i] = &datastore.Range2{Low: udr.lastKeys[i], Inclusion: datastore.HIGH}
	}
	udr.lastKeys = nil
	udr.restartScan(0)
}

func (udr *UnifiedDocumentRetriever) cacheKey(key string) {
	if udr.isFlagOff(CACHE_KEYS) {
		return
	}
	if udr.cache == nil {
		udr.cache = make([]string, 0, udr.sampleSize)
	}
	udr.cache = append(udr.cache, key)
}

////////////////////////////////////////////////////////////////////////////////
// KVRandomDocumentRetriever implementation
//
// Given a server name, login & password, and bucket name and password,
// use the couchbase bucket GetRandomDoc() method to retrieve
// non-duplicate radom docs until we have sampleSize (or give up because we
// keep seeing duplicates).
////////////////////////////////////////////////////////////////////////////////

type KVRandomDocumentRetriever struct {
	docIdsSeen map[string]bool
	sampleSize int
	bucket     *couchbase.Bucket
}

func (kvrdr *KVRandomDocumentRetriever) Reset() {
}

func (kvrdr *KVRandomDocumentRetriever) Close() {
}

func (kvrdr *KVRandomDocumentRetriever) GetNextDoc(context datastore.QueryContext) (string, value.Value, errors.Error) {
	// have we returned as many documents as were requested?
	if len(kvrdr.docIdsSeen) >= kvrdr.sampleSize {
		return _EMPTY_KEY, nil, nil
	}

	// try to retrieve the next document
	duplicatesSeen := 0
	for duplicatesSeen < _MAX_DUPLICATES {
		resp, err := kvrdr.bucket.GetRandomDoc()

		if err != nil {
			return _EMPTY_KEY, nil, errors.NewInferRandomError(err)
		}

		key := fmt.Sprintf("%s", resp.Key)
		val := value.NewValue(resp.Body)

		if kvrdr.docIdsSeen[key] { // seen it before?
			duplicatesSeen++
			continue
		}

		kvrdr.docIdsSeen[key] = true
		return key, val, nil // new doc, return
	}

	// if we get here, we saw duplicate docs 100 times in a row, so we give up on finding any more new docs
	return _EMPTY_KEY, nil, nil
}

func MakeKVRandomDocumentRetriever(serverURL, bucket, bucketPass string, sampleSize int) (*KVRandomDocumentRetriever, errors.Error) {

	kvrdr := new(KVRandomDocumentRetriever)
	kvrdr.docIdsSeen = make(map[string]bool)
	kvrdr.sampleSize = sampleSize

	var client couchbase.Client
	var err error

	client, err = couchbase.Connect(serverURL)
	if err != nil {
		return nil, errors.NewInferConnectFailed(serverURL, err)
	}

	pool, err := client.GetPool("default")
	if err != nil {
		return nil, errors.NewInferGetPoolFailed(err)
	}

	kvrdr.bucket, err = pool.GetBucket(bucket)
	if err != nil {
		return nil, errors.NewInferGetBucketFailed(bucket, err)
	}

	return kvrdr, nil
}

////////////////////////////////////////////////////////////////////////////////
// ExpressionDocumentRetriever implementation
//
// Given an expression, evaluate to produce the document.
////////////////////////////////////////////////////////////////////////////////

type subqueryResults interface {
	Results() (interface{}, uint64, error)
	NextDocument() (value.Value, error)
	Cancel()
}

type ExpressionDocumentRetriever struct {
	doc         value.Value
	returnIndex int
	sampleSize  int
	subquery    subqueryResults
}

func MakeExpressionDocumentRetriever(context datastore.QueryContext, expr expression.Expression, sampleSize int) (
	*ExpressionDocumentRetriever, errors.Error) {

	if sampleSize < 1 {
		sampleSize = 1
	}
	edr := new(ExpressionDocumentRetriever)
	edr.returnIndex = 0
	edr.sampleSize = sampleSize

	ectx, ok := context.(expression.Context)
	if !ok {
		return nil, errors.NewInferMissingContext(fmt.Sprintf("%T", context))
	}

	var err error
	if sq, ok := expr.(*algebra.Subquery); ok {
		// since we will not want any more than sampleSize results, we might as well limit the subquery to this to save processing
		if sq.Select() != nil && sq.Select().Limit() == nil {
			sq.Select().SetLimit(expression.NewConstant(sampleSize))
		}
		// stream subqueries to save on caching a potentially large result-set all at once, even though it means processing this
		// statement a second time
		logging.Debuga(func() string { return sq.Select().String() })
		edr.subquery, err = ectx.OpenStatement(sq.Select().String(), nil, nil, false, true)
		if err != nil {
			return nil, errors.NewInferExpressionEvalFailed(err)
		}
	} else {
		edr.doc, err = expr.Evaluate(nil, ectx)
		if err != nil {
			return nil, errors.NewInferExpressionEvalFailed(err)
		}
		if edr.doc.Type() == value.ARRAY {
			a := edr.doc.Actual().([]interface{})
			if len(a) > sampleSize {
				a = a[:sampleSize]
				edr.doc = value.NewValue(a)
			}
		}
	}

	return edr, nil
}

func (this *ExpressionDocumentRetriever) Reset() {
}

func (this *ExpressionDocumentRetriever) Close() {
	this.subquery = nil
}

func (this *ExpressionDocumentRetriever) GetNextDoc(context datastore.QueryContext) (string, value.Value, errors.Error) {
	if this.returnIndex >= this.sampleSize {
		if this.subquery != nil {
			this.subquery.Cancel()
		}
		return _EMPTY_KEY, nil, nil
	}

	if this.subquery != nil {
		doc, err := this.subquery.NextDocument()
		if err != nil {
			if e, ok := err.(errors.Error); ok {
				return _EMPTY_KEY, nil, e
			} else {
				return _EMPTY_KEY, nil, errors.NewError(err, "NextDocument failed")
			}
		}
		this.returnIndex++
		return fmt.Sprintf("_%d", this.returnIndex), doc, nil
	} else {
		this.returnIndex = this.sampleSize
		return "_1", this.doc, nil
	}
}
