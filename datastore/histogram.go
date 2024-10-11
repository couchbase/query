//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package datastore

import (
	"time"

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

type DistBins []*DistBin
type OverflowBins []*OverflowBin

type DistBin struct {
	size     float64     // fraction of documents this bin represents
	distinct float64     // fraction of number of distinct values
	max      value.Value // maximum value in this bin
}

func NewDistBin(size, distinct float64, max value.Value) *DistBin {
	return &DistBin{
		size:     size,
		distinct: distinct,
		max:      max,
	}
}

func (this *DistBin) Size() float64 {
	return this.size
}

func (this *DistBin) SetSize(size float64) {
	this.size = size
}

func (this *DistBin) Distinct() float64 {
	return this.distinct
}

func (this *DistBin) SetDistinct(distinct float64) {
	this.distinct = distinct
}

func (this *DistBin) Max() value.Value {
	return this.max
}

type OverflowBin struct {
	size float64
	val  value.Value
}

func NewOverflowBin(size float64, val value.Value) *OverflowBin {
	return &OverflowBin{
		size: size,
		val:  val,
	}
}

func (this *OverflowBin) Size() float64 {
	return this.size
}

func (this *OverflowBin) SetSize(size float64) {
	this.size = size
}

func (this *OverflowBin) Val() value.Value {
	return this.val
}

type StatBins interface {
	GetValue(pos int) value.Value
	NumBins() int
}

func (this DistBins) GetValue(pos int) value.Value {
	if pos < len(this) {
		return this[pos].max
	}
	return nil
}

func (this DistBins) NumBins() int {
	return len(this)
}

func (this OverflowBins) GetValue(pos int) value.Value {
	if pos < len(this) {
		return this[pos].val
	}
	return nil
}

func (this OverflowBins) NumBins() int {
	return len(this)
}

const HISTOGRAM_VERSION = 2

type Histogram struct {
	version    int32
	keyspace   string
	key        expression.Expression
	docCount   int64
	sampleSize int64
	resolution float64
	fdistincts float64
	arrayInfo  *ArrayInfo
	distrib    DistBins
	ovrflow    OverflowBins
	internal   bool
	updated    time.Time

	// A list of index information for all indexes that the "key" depends on.
	indexInfo IndexInfo
}

type IndexInfo []IndexStats

type IndexStats struct {
	id        string
	name      string
	mutations float64
	items     float64
}

func NewIndexStats(indexId string, indexName string, mutations float64, items float64) IndexStats {
	return IndexStats{
		id:        indexId,
		name:      indexName,
		mutations: mutations,
		items:     items,
	}
}

func (this *IndexStats) IndexId() string {
	return this.id
}

func (this *IndexStats) IndexName() string {
	return this.name
}

func (this *IndexStats) Mutations() float64 {
	return this.mutations
}

func (this *IndexStats) Items() float64 {
	return this.items
}

type ArrayInfo struct {
	avgArrayLen float64
	missingArr  float64
	emptyArr    float64
}

func NewArrayInfo(avgArrayLen, missingArr, emptyArr float64) *ArrayInfo {
	return &ArrayInfo{
		avgArrayLen: avgArrayLen,
		missingArr:  missingArr,
		emptyArr:    emptyArr,
	}
}

func (this *Histogram) SetHistogram(version int32, keyspace string, key expression.Expression,
	docCount, sampleSize int64, resolution float64,
	fdistincts, avgArrayLen, missingArr, emptyArr float64,
	distrib DistBins, ovrflow OverflowBins, updated time.Time, indexInfo IndexInfo) {
	this.version = version
	this.keyspace = keyspace
	this.key = key
	this.docCount = docCount
	this.sampleSize = sampleSize
	this.resolution = resolution
	this.fdistincts = fdistincts
	this.distrib = distrib
	this.ovrflow = ovrflow
	this.updated = updated

	if avgArrayLen > 0.0 || missingArr > 0.0 || emptyArr > 0.0 {
		this.arrayInfo = NewArrayInfo(avgArrayLen, missingArr, emptyArr)
	}

	this.indexInfo = indexInfo

	return
}

func (this *Histogram) Version() int32 {
	return this.version
}

func (this *Histogram) Keyspace() string {
	return this.keyspace
}

func (this *Histogram) Key() expression.Expression {
	return this.key
}

func (this *Histogram) DocCount() int64 {
	return this.docCount
}

func (this *Histogram) SampleSize() int64 {
	return this.sampleSize
}

func (this *Histogram) Resolution() float64 {
	return this.resolution
}

func (this *Histogram) Fdistincts() float64 {
	return this.fdistincts
}

func (this *Histogram) Distrib() DistBins {
	return this.distrib
}

func (this *Histogram) Ovrflow() OverflowBins {
	return this.ovrflow
}

func (this *Histogram) Updated() time.Time {
	return this.updated
}

func (this *Histogram) SetInternal() {
	this.internal = true
}

func (this *Histogram) IsInternal() bool {
	return this.internal
}

func (this *Histogram) ArrayInfo() *ArrayInfo {
	return this.arrayInfo
}

func (this *Histogram) IndexInfo() IndexInfo {
	return this.indexInfo
}

func (this *ArrayInfo) AvgArrayLen() float64 {
	return this.avgArrayLen
}

func (this *ArrayInfo) MissingArray() float64 {
	return this.missingArr
}

func (this *ArrayInfo) EmptyArray() float64 {
	return this.emptyArr
}
