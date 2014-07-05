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
	DEFAULT     IndexType = "default"     // default may vary per backend
	UNSPECIFIED IndexType = "unspecified" // used by non-view primary_indexes
	VIEW        IndexType = "view"
	LSM         IndexType = "lsm"
)

// Index is the base type for indexes.
type Index interface {
	BucketId() string                                                   // Id of the bucket to which this index belongs
	Id() string                                                         // Id of this index
	Name() string                                                       // Name of this index
	Type() IndexType                                                    // Type of this index
	Drop() err.Error                                                    // Drop / delete this index
	EqualKey() expression.Expressions                                   // Equality keys
	RangeKey() expression.Expressions                                   // Range keys
	Condition() expression.Expression                                   // Condition, if any
	Statistics(span *Span) (Statistics, err.Error)                      // Obtain statistics for this index
	Scan(span *Span, distinct bool, limit int64, conn *IndexConnection) // Perform a scan on this index. Distinct and limit are hints.
}

// PrimaryIndex represents primary key indexes.
type PrimaryIndex interface {
	ScanEntries(limit int64, conn *IndexConnection) // Perform a scan of all the entries in this index
}

type Range struct {
	Low       value.Values
	High      value.Values
	Inclusion Inclusion
}

type Ranges []*Range

// Inclusion controls how the boundary values of a range are treated.
type Inclusion int

const (
	NEITHER Inclusion = 0x00
	LOW               = 0x01
	HIGH              = 0x10
	BOTH              = LOW | HIGH
)

type Span struct {
	Equal value.Values
	Range *Range
}

type Spans []*Span

type IndexEntry struct {
	EntryKey   value.Values
	PrimaryKey string
}

type EntryChannel chan *IndexEntry
type StopChannel chan bool

// Statistics captures statistics for a range.
type Statistics interface {
	Count() (int64, err.Error)
	Min() (value.Values, err.Error)
	Max() (value.Values, err.Error)
	DistinctCount(int64, err.Error)
	Bins() ([]Statistics, err.Error)
}

type IndexConnection struct {
	entryChannel   EntryChannel     // Closed by the index when the scan is completed or aborted.
	stopChannel    StopChannel      // Notifies index to stop scanning. Never closed, just garbage-collected.
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
