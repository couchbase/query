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
	"github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type IndexType string

const (
	UNSPECIFIED IndexType = "unspecified" // used by non-view primary_indexes
	VIEW        IndexType = "view"
	LSM         IndexType = "lsm"
)

// Index is the base type for all indexes.
type Index interface {
	expression.Index
	BucketId() string
	Id() string
	Name() string
	Type() IndexType
	Drop() err.Error // PrimaryIndexes cannot be dropped
	Statistics(span *expression.Span) (Statistics, err.Error)
	Scan(span *expression.Span, conn *IndexConnection)
	CandidateMins(span *expression.Span, conn *IndexConnection)  // Anywhere from single min value to Scan()
	CandidateMaxes(span *expression.Span, conn *IndexConnection) // Anywhere from single max value to Scan()
}

type IndexEntry struct {
	EntryKey   value.CompositeValue
	PrimaryKey string
}

type EntryChannel chan *IndexEntry
type StopChannel chan bool

// PrimaryIndex represents primary key indexes.
type PrimaryIndex interface {
	Index
	PrimaryScan(conn *IndexConnection)
}

// Statistics captures statistics for an index span.
type Statistics interface {
	Count() (int64, err.Error)
	Min() (value.Value, err.Error)
	Max() (value.Value, err.Error)
	DistinctCount(int64, err.Error)
	Bins() ([]Statistics, err.Error)
}

type IndexConnection struct {
	entryChannel   EntryChannel     // Closed by index.
	stopChannel    StopChannel      // Stop notification to index. Never closed, just garbage-collected.
	warningChannel err.ErrorChannel // Written by index. Never closed, just garbage-collected.
	errorChannel   err.ErrorChannel // Written by index. Never closed, just garbage-collected.
}

const _ENTRY_CAP = 1024

func NewIndexConnection(warningChannel, errorChannel err.ErrorChannel) *IndexConnection {
	return &IndexConnection{
		entryChannel:   make(EntryChannel, _ENTRY_CAP),
		stopChannel:    make(StopChannel, 1),
		warningChannel: warningChannel,
		errorChannel:   errorChannel,
	}
}

func (this *IndexConnection) EntryChannel() EntryChannel {
	return this.entryChannel
}

func (this *IndexConnection) StopChannel() StopChannel {
	return this.stopChannel
}

func (this *IndexConnection) SendWarning(e err.Error) {
	this.warningChannel <- e
}

func (this *IndexConnection) SendError(e err.Error) {
	this.errorChannel <- e
}
