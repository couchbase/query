//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package datastore

import (
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/timestamp"
	"github.com/couchbaselabs/query/value"
)

type IndexType string

const (
	DEFAULT IndexType = "default" // default may vary per backend
	VIEW    IndexType = "view"    // view index
	GSI     IndexType = "gsi"     // global secondary index
)

type Indexer interface {
	KeyspaceId() string                                                            // Id of the keyspace to which this indexer belongs
	Name() IndexType                                                               // Unique within a Keyspace.
	IndexIds() ([]string, errors.Error)                                            // Ids of the indexes defined on this keyspace
	IndexNames() ([]string, errors.Error)                                          // Names of the indexes defined on this keyspace
	IndexById(id string) (Index, errors.Error)                                     // Find an index on this keyspace using the index's id
	IndexByName(name string) (Index, errors.Error)                                 // Find an index on this keyspace using the index's name
	PrimaryIndexes() ([]PrimaryIndex, errors.Error)                                // Returns the server-recommended primary index
	Indexes() ([]Index, errors.Error)                                              // Returns all the indexes defined on this keyspace
	CreatePrimaryIndex(name string, with value.Value) (PrimaryIndex, errors.Error) // Create or return a primary index on this keyspace
	CreateIndex(name string, equalKey, rangeKey expression.Expressions,            // Create a secondary index on this keyspace
		where expression.Expression, with value.Value) (Index, errors.Error)
	BuildIndexes(name ...string) errors.Error // Build indexes that were deferred at creation
	Refresh() errors.Error                    // Refresh list of indexes from metadata
}

type IndexState string

const (
	PENDING IndexState = "pending" // The index is being built or rebuilt
	ONLINE  IndexState = "online"  // The index is available for use
	OFFLINE IndexState = "offline" // The index requires manual intervention
)

type ScanConsistency string

const (
	UNBOUNDED ScanConsistency = "unbounded"
	SCAN_PLUS ScanConsistency = "scan_plus"
	AT_PLUS   ScanConsistency = "at_plus"
)

type IndexKey expression.Expressions

type Indexes []Index

/*
Index is the base type for indexes, which may be distributed.
*/
type Index interface {
	KeyspaceId() string                                      // Id of the keyspace to which this index belongs
	Id() string                                              // Id of this index
	Name() string                                            // Name of this index
	Type() IndexType                                         // Type of this index
	SeekKey() expression.Expressions                         // Equality keys
	RangeKey() expression.Expressions                        // Range keys
	Condition() expression.Expression                        // Condition, if any
	State() (state IndexState, msg string, err errors.Error) // Obtain state of this index
	Statistics(span *Span) (Statistics, errors.Error)        // Obtain statistics for this index
	Drop() errors.Error                                      // Drop / delete this index
	Scan(span *Span, distinct bool, limit int64, cons ScanConsistency, vector timestamp.Vector,
		conn *IndexConnection) // Perform a scan on this index. Distinct and limit are hints.
}

/*
PrimaryIndex represents primary key indexes.
*/
type PrimaryIndex interface {
	Index

	ScanEntries(limit int64, cons ScanConsistency, vector timestamp.Vector,
		conn *IndexConnection) // Perform a scan of all the entries in this index
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
	HIGH              = 0x01 << 1
	BOTH              = LOW | HIGH
)

type Span struct {
	Seek  value.Values
	Range Range
}

type Spans []*Span

type IndexEntry struct {
	EntryKey   value.Values
	PrimaryKey string
}

type EntryChannel chan *IndexEntry
type StopChannel chan bool

// Statistics captures statistics for a range.
// - it may return heuristics and/or outdated values.
// - query shall not depend on the accuracy of this statistics.
// - primarily intended for optimizer's consumption.
type Statistics interface {
	Count() (int64, errors.Error)
	Min() (value.Values, errors.Error)
	Max() (value.Values, errors.Error)
	DistinctCount() (int64, errors.Error)
	Bins() ([]Statistics, errors.Error)
}

type Context interface {
	Fatal(errors.Error)
	Error(errors.Error)
	Warning(errors.Error)
}

type IndexConnection struct {
	entryChannel EntryChannel // Closed by the index when the scan is completed or aborted.
	stopChannel  StopChannel  // Notifies index to stop scanning. Never closed, just garbage-collected.
	context      Context
}

const _ENTRY_CAP = 1024

func NewIndexConnection(context Context) *IndexConnection {
	return &IndexConnection{
		entryChannel: make(EntryChannel, _ENTRY_CAP),
		stopChannel:  make(StopChannel, 1),
		context:      context,
	}
}

func (this *IndexConnection) EntryChannel() EntryChannel {
	return this.entryChannel
}

func (this *IndexConnection) StopChannel() StopChannel {
	return this.stopChannel
}

func (this *IndexConnection) Fatal(err errors.Error) {
	this.context.Fatal(err)
}

func (this *IndexConnection) Error(err errors.Error) {
	this.context.Error(err)
}

func (this *IndexConnection) Warning(wrn errors.Error) {
	this.context.Warning(wrn)
}
