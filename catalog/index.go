//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package catalog

import (
	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/value"
)

type IndexType string

const (
	UNSPECIFIED IndexType = "unspecified" // used by non-view primary_indexes
	VIEW        IndexType = "view"
	LSM         IndexType = "lsm"
)

type EqualKey []algebra.Expression
type RangeKey []*RangePart

type RangePart struct {
	Expr algebra.Expression
	Dir  Direction
}

// Index is the base type for all indexes.
type Index interface {
	BucketId() string
	Id() string
	Name() string
	Type() IndexType
	Equal() EqualKey
	Range() RangeKey
	Drop() err.Error // PrimaryIndexes cannot be dropped
}

type IndexEntry struct {
	EntryKey   value.CompositeValue
	PrimaryKey string
}

type EntryChannel chan *IndexEntry
type StopChannel chan bool

// PrimaryIndex represents primary key indexes.
type PrimaryIndex interface {
	EqualIndex
	PrimaryScan(conn *IndexConnection)
}

// EqualIndexes support equality matching.
type EqualIndex interface {
	Index
	EqualScan(match value.CompositeValue, conn *IndexConnection)
	EqualCount(match value.CompositeValue, conn *IndexConnection)
}

// Direction represents ASC and DESC
type Direction int

const (
	ASC  Direction = 1
	DESC           = 2
)

// Inclusion controls how the boundary values of a range are treated.
type RangeInclusion int

const (
	NEITHER RangeInclusion = iota
	LOW
	HIGH
	BOTH
)

type Ranges []*Range

type Range struct {
	Low       value.CompositeValue
	High      value.CompositeValue
	Inclusion RangeInclusion
}

// RangeIndexes support unrestricted range queries.
type RangeIndex interface {
	Index
	RangeStats(r *Range) (RangeStatistics, err.Error)
	RangeScan(r *Range, conn *IndexConnection)
	RangeCount(r *Range, conn *IndexConnection)
	RangeCandidateMins(r *Range, conn *IndexConnection)  // Anywhere from single Min value to RangeScan()
	RangeCandidateMaxes(r *Range, conn *IndexConnection) // Anywhere from single Max value to RangeScan()
}

// DualIndexes support restricted range queries.
type DualIndex interface {
	Index
	DualStats(match value.CompositeValue, r *Range) (RangeStatistics, err.Error)
	DualScan(match value.CompositeValue, r *Range, conn *IndexConnection)
	DualCount(match value.CompositeValue, r *Range, conn *IndexConnection)
	DualCandidateMins(match value.CompositeValue, r *Range, conn *IndexConnection)  // Anywhere from single Min value to DualScan()
	DualCandidateMaxes(match value.CompositeValue, r *Range, conn *IndexConnection) // Anywhere from single Max value to DualScan()
}

// RangeStatistics captures statistics for an index range.
type RangeStatistics interface {
	Count() (int64, err.Error)
	Min() (value.Value, err.Error)
	Max() (value.Value, err.Error)
	DistinctCount(int64, err.Error)
	Bins() ([]RangeStatistics, err.Error)
}

type IndexConnection struct {
	entryChannel   EntryChannel     // Entries
	warningChannel err.ErrorChannel // Warnings
	errorChannel   err.ErrorChannel // Errors
	stopChannel    StopChannel      // Stop notification; replaces limit
}

const _ENTRY_CAP = 1024
const _ERROR_CAP = 64

func NewIndexConnection() *IndexConnection {
	return &IndexConnection{
		entryChannel:   make(EntryChannel, _ENTRY_CAP),
		warningChannel: make(err.ErrorChannel, _ERROR_CAP),
		errorChannel:   make(err.ErrorChannel, _ERROR_CAP),
		stopChannel:    make(StopChannel, 1),
	}
}

func (this *IndexConnection) EntryChannel() EntryChannel {
	return this.entryChannel
}

func (this *IndexConnection) WarningChannel() err.ErrorChannel {
	return this.warningChannel
}

func (this *IndexConnection) ErrorChannel() err.ErrorChannel {
	return this.errorChannel
}

func (this *IndexConnection) StopChannel() StopChannel {
	return this.stopChannel
}
