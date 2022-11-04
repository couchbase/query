//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"sync"
	"sync/atomic"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type BitFilterTerm struct {
	sync.RWMutex
	indexBFs map[string]*BitFilter
}

func newBitFilterTerm(nIndexes int) *BitFilterTerm {
	return &BitFilterTerm{
		indexBFs: make(map[string]*BitFilter, nIndexes),
	}
}

func (this *BitFilterTerm) getBitFilter(indexId string) (*BloomFilter, errors.Error) {
	if len(this.indexBFs) == 0 {
		return nil, nil
	}
	this.RLock()
	bf := this.indexBFs[indexId]
	this.RUnlock()
	if bf == nil {
		return nil, nil
	}
	return bf.getFilter()
}

func (this *BitFilterTerm) setBitFilter(indexId string, filter *BloomFilter) errors.Error {
	this.RLock()
	bf := this.indexBFs[indexId]
	this.RUnlock()
	if bf == nil {
		this.Lock()
		bf = this.indexBFs[indexId]
		if bf == nil {
			bf = newBitFilter()
			this.indexBFs[indexId] = bf
		}
		this.Unlock()
	}
	return bf.setFilter(filter)
}

func (this *BitFilterTerm) clearBitFilters(idxId string) (empty bool) {
	this.Lock()
	bf := this.indexBFs[idxId]
	if bf != nil {
		bf.clearFilter()
		delete(this.indexBFs, idxId)
		empty = len(this.indexBFs) == 0
	}
	this.Unlock()
	return
}

const (
	_BITFILTER_BUILD = int32(iota)
	_BITFILTER_PROBE
)

type BitFilter struct {
	sync.Mutex
	mode   int32
	count  int32
	filter *BloomFilter
}

func newBitFilter() *BitFilter {
	return &BitFilter{
		mode: _BITFILTER_BUILD,
	}
}

func (this *BitFilter) getFilter() (*BloomFilter, errors.Error) {
	if atomic.AddInt32(&(this.count), 1) == 1 {
		this.mode = _BITFILTER_PROBE
	} else if this.mode != _BITFILTER_PROBE {
		return nil, errors.NewExecutionInternalError("Getting bit filter when it's not in probe mode")
	}
	return this.filter, nil
}

func (this *BitFilter) setFilter(filter *BloomFilter) errors.Error {
	var err error
	this.Lock()
	if this.mode == _BITFILTER_PROBE {
		this.Unlock()
		return errors.NewExecutionInternalError("Adding bit filter after it's in probe mode")
	}
	if this.filter == nil {
		this.filter = filter
	} else {
		err = this.filter.Merge(filter)
	}
	this.Unlock()
	if err != nil {
		return errors.NewExecutionInternalError("BitFilter merge failed: " + err.Error())
	}
	return nil
}

func (this *BitFilter) clearFilter() {
	if atomic.AddInt32(&(this.count), -1) == 0 {
		this.Lock()
		this.filter = nil
		this.Unlock()
	}
}

type localBitFilter struct {
	exprs  expression.Expressions
	filter *BloomFilter
}

func newLocalBitFilter(exprs expression.Expressions, filter *BloomFilter) *localBitFilter {
	return &localBitFilter{
		exprs:  exprs,
		filter: filter,
	}
}

func (this *localBitFilter) BitFilter() *BloomFilter {
	return this.filter
}

func (this *localBitFilter) Expressions() expression.Expressions {
	return this.exprs
}

type buildBitFilterBase struct {
	localBuildFilters map[string]map[string]*localBitFilter
}

func (this *buildBitFilterBase) hasBuildBitFilter() bool {
	return len(this.localBuildFilters) > 0
}

func (this *buildBitFilterBase) createLocalBuildFilters(buildBitFilters plan.BitFilters) {
	this.localBuildFilters = make(map[string]map[string]*localBitFilter, len(buildBitFilters))

	for _, bfs := range buildBitFilters {
		aliasBuildFilters := make(map[string]*localBitFilter, len(bfs.IndexBitFilters()))
		this.localBuildFilters[bfs.Alias()] = aliasBuildFilters
		for _, indexBF := range bfs.IndexBitFilters() {
			bfilter := newBloomFilter(indexBF.Size())
			if bfilter != nil {
				indexFilter := newLocalBitFilter(indexBF.Expressions(), bfilter)
				aliasBuildFilters[indexBF.IndexId()] = indexFilter
			}
		}
	}
}

func (this *buildBitFilterBase) buildBitFilters(item value.AnnotatedValue, context *opContext) bool {
	for _, bfs := range this.localBuildFilters {
		for _, bf := range bfs {
			_, err := processBitFilters(item, bf.BitFilter(), bf.Expressions(), true, context)
			if err != nil {
				context.Error(err)
				return false
			}
		}
	}
	return true
}

func (this *buildBitFilterBase) setBuildBitFilters(alias string, context *Context) {
	for _, bfs := range this.localBuildFilters {
		for idx, bf := range bfs {
			err := context.setBitFilter(alias, idx, len(this.localBuildFilters), len(bfs), bf.BitFilter())
			if err != nil {
				context.Error(err)
			}
		}
	}
	this.localBuildFilters = nil
}

type probeBitFilterBase struct {
	localProbeFilters map[string]map[string]*localBitFilter
}

func (this *probeBitFilterBase) hasProbeBitFilter() bool {
	return len(this.localProbeFilters) > 0
}

func (this *probeBitFilterBase) getLocalProbeFilters(probeBitFilters plan.BitFilters, context *Context) errors.Error {

	this.localProbeFilters = make(map[string]map[string]*localBitFilter, len(probeBitFilters))

	for _, bfs := range probeBitFilters {
		buildAlias := bfs.Alias()
		aliasProbeFilters := make(map[string]*localBitFilter, len(bfs.IndexBitFilters()))
		this.localProbeFilters[buildAlias] = aliasProbeFilters
		for _, indexBF := range bfs.IndexBitFilters() {
			idx := indexBF.IndexId()
			bfilter, err := context.getBitFilter(buildAlias, idx)
			if err != nil {
				return err
			}
			if bfilter != nil {
				aliasProbeFilters[idx] = newLocalBitFilter(indexBF.Expressions(), bfilter)
			}
		}
	}

	return nil
}

func (this *probeBitFilterBase) probeBitFilters(item value.AnnotatedValue, context *opContext) (bool, bool) {
	for _, bfs := range this.localProbeFilters {
		for _, bf := range bfs {
			pass, err := processBitFilters(item, bf.BitFilter(), bf.Expressions(), false, context)
			if err != nil {
				context.Error(err)
				return false, false
			}
			if !pass {
				return true, false
			}
		}
	}

	return true, true
}

func (this *probeBitFilterBase) clearProbeBitFilters(context *Context) {
	for alias, bfs := range this.localProbeFilters {
		for idx, _ := range bfs {
			context.clearBitFilter(alias, idx)
		}
	}
	this.localProbeFilters = nil
}

func processBitFilters(av value.AnnotatedValue, bitFilter *BloomFilter,
	bitFilterExprs expression.Expressions, build bool, context *opContext) (bool, errors.Error) {

	vals := make(value.Values, 0, len(bitFilterExprs))
	for _, exp := range bitFilterExprs {
		val, err := exp.Evaluate(av, context)
		if err != nil {
			return false, errors.NewEvaluationError(err, "bit filter")
		}
		vals = append(vals, val)
	}
	var bitVal value.Value
	if len(vals) == 1 {
		bitVal = vals[0]
	} else {
		bitVal = value.NewValue(vals)
	}

	data, err := bitVal.MarshalJSON()
	if err != nil {
		return false, errors.NewExecutionInternalError("marshal value")
	}

	var ok bool
	if build {
		ok = true
		bitFilter.Add(data)
	} else {
		ok = bitFilter.Test(data)
	}

	return ok, nil
}
