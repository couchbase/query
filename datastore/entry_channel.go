//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package datastore

type idxEntryChannel struct {
	conn *IndexConnection
}

func newEntryChannel(entryChannel *idxEntryChannel, conn *IndexConnection) {
	entryChannel.conn = conn
}

func (this *idxEntryChannel) dispose() {
}

// capacity
func (this *idxEntryChannel) Capacity() int {
	return cap(this.conn.EntryChannel())
}

// length
func (this *idxEntryChannel) Length() int {
	return len(this.conn.EntryChannel())
}

// send
func (this *idxEntryChannel) SendEntry(e *IndexEntry) bool {
	select {
	case this.conn.EntryChannel() <- e:
		return true
	case <-this.conn.StopChannel():
		return false
	}
}

// no need for receive

// last orders
func (this *idxEntryChannel) Close() {
	close(this.conn.EntryChannel())
}

// signal stop
func (this *idxEntryChannel) sendStop() {
	select {
	case this.conn.StopChannel() <- true:
	default:
	}
}

// did we get a stop?
func (this *idxEntryChannel) IsStopped() bool {
	return len(this.conn.StopChannel()) > 0
}
