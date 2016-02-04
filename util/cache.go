//  Copyright (c) 2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*
GenCache provides a highly concurrent, resizable set of structures, implemented as an
an array of doubly linked lists for sequential access complemented by maps for direct
access.

The ForEach method is not meant to provide a snapshot of the current state of affairs
but rather an almost accurate picture: deletes and inserts are allowed as the scan
takes place.
*/
package util

import (
	"sync"

	atomic "github.com/couchbase/go-couchbase/platform"
)

type genSubList struct {
	ID       string
	next     *genSubList
	prev     *genSubList
	refcount int
	deleted  bool
	contents interface{}
}

const _CACHE_SIZE = 1 << 10
const _CACHES = 4

type GenCache struct {

	// one lock per cache bucket to aid concurrency
	locks [_CACHES]sync.RWMutex

	// doubly linked lists for scans
	listHead [_CACHES]*genSubList
	listTail [_CACHES]*genSubList

	// maps for direct access
	maps [_CACHES]map[string]*genSubList

	// max size, for MRU lists
	limit   int
	curSize int32
}

func NewGenCache(l int) *GenCache {
	rv := &GenCache{
		limit: l,
	}

	for b := 0; b < _CACHES; b++ {
		rv.maps[b] = make(map[string]*genSubList, _CACHE_SIZE)
	}
	return rv
}

// Add (or update, if ID found) entry, and size control if MRU list
func (this *GenCache) Add(entry interface{}, id string) {
	cacheNum := HashString(id, _CACHES)
	this.locks[cacheNum].Lock()
	defer this.locks[cacheNum].Unlock()

	elem, ok := this.maps[cacheNum][id]
	if ok {
		elem.contents = entry

		// Move to the front
		this.ditch(elem, cacheNum)
		this.insert(elem, cacheNum)
	} else {

		// In order not to have to acquire a different lock
		// we ditch the LRU entry from this hash node:
		// it makes the list a bit lopsided at the lower end
		// but it buys us performance
		elem = this.listTail[cacheNum]
		if this.limit > 0 && int(this.curSize) >= this.limit && elem != nil {

			delete(this.maps[cacheNum], elem.ID)
			this.ditch(elem, cacheNum)
		} else {
			atomic.AddInt32(&this.curSize, 1)
		}
		elem = &genSubList{
			contents: entry,
			ID:       id,
		}
		this.insert(elem, cacheNum)
		this.maps[cacheNum][id] = elem
		atomic.AddInt32(&this.curSize, 1)
	}
}

// Remove entry
func (this *GenCache) Delete(id string, cleanup func(interface{})) bool {
	cacheNum := HashString(id, _CACHES)
	this.locks[cacheNum].Lock()
	defer this.locks[cacheNum].Unlock()

	elem, ok := this.maps[cacheNum][id]
	if ok {
		if cleanup != nil {
			cleanup(elem.contents)
		}
		delete(this.maps[cacheNum], id)
		this.ditch(elem, cacheNum)
		atomic.AddInt32(&this.curSize, -1)
		return true
	}
	return false
}

// Returns an element's contents by id
func (this *GenCache) Get(id string, process func(interface{})) interface{} {
	cacheNum := HashString(id, _CACHES)
	this.locks[cacheNum].RLock()
	defer this.locks[cacheNum].RUnlock()
	elem, ok := this.maps[cacheNum][id]
	if !ok {
		return nil
	} else {
		if process != nil {
			process(elem.contents)
		}
		return elem.contents
	}
}

// List Size
func (this *GenCache) Size() int {
	return int(this.curSize)
}

// LRU cleanup limit
func (this *GenCache) Limit() int {
	return this.limit
}

// Set the list limit
func (this *GenCache) SetLimit(limit int) {

	// this we ought to do with a lock, however
	// we only envisage one request to change the limit
	// every blue moon and it's only Add that's using it
	// to keep the list compact: in the worse case we
	// skip ditching entries, which is done here anyhow...
	this.limit = limit

	// reign in entries a bit
	c := 0
	for this.limit > 0 && int(this.curSize) > this.limit {
		this.locks[c].Lock()
		elem := this.listTail[c]
		if elem != nil {
			delete(this.maps[c], elem.ID)
			this.ditch(elem, c)
			atomic.AddInt32(&this.curSize, -1)
		}
		this.locks[c].Unlock()
		c = (c + 1) % _CACHES
	}
}

// Return a slice with all the entry id's
func (this *GenCache) Names() []string {
	i := 0

	// we have emergency extra space not to have to append
	// if we can avoid it
	sz := _CACHES + int(this.curSize)
	n := make([]string, sz)
	this.ForEach(func(id string, entry interface{}) {
		if i < sz {
			n[i] = id
		} else {
			n = append(n, id)
		}
		i++
	})
	return n
}

// Scan the list
// As noted in the starting comments, this is not a consistent snapshot
// but rather a a low cost, almost accurate view
func (this *GenCache) ForEach(f func(string, interface{})) {
	for b := 0; b < _CACHES; b++ {
		needDecr := false
		this.locks[b].RLock()
		elem := this.listTail[b]
		if elem == nil {
			this.locks[b].RUnlock()
			continue
		}
		for {

			// if the current element had been marked as in use
			// release it
			if needDecr {
				elem.refcount--
				needDecr = false
			}

			// and if somebody had deleted it in the interim
			// skip it
			if elem.deleted {
				oldElem := elem
				elem = elem.next

				// and if no longer referenced, get rid of it for real
				if oldElem.refcount == 0 {

					// a bit naughty, but since we own the lock
					// and refcount is 0, it will be deleted
					this.ditch(oldElem, b)
				}

				// if not, do what's required
			} else {
				f(elem.ID, elem.contents)
				elem = elem.next
			}

			// now mark that this element will be scanned so
			// that it doesn't get removed from the list and
			// we get lost mid scan...
			if elem != nil {
				elem.refcount++
				needDecr = true
			} else {

				// No need to go through the unlock / relock
				// if we are done
				this.locks[b].RUnlock()
				break
			}

			// it would be nice golang locks showed waiters
			// (by catergory, even), so that we could avoid
			// releasing the lock if no writer is waiting
			this.locks[b].RUnlock()
			this.locks[b].RLock()
		}
	}
}

func (this *GenCache) insert(elem *genSubList, cacheNum int) {
	elem.next = nil
	if this.listHead[cacheNum] == nil {
		this.listHead[cacheNum] = elem
		this.listTail[cacheNum] = elem
		elem.prev = nil
	} else {
		elem.prev = this.listHead[cacheNum]
		elem.prev.next = elem
		this.listHead[cacheNum] = elem
	}

}

func (this *GenCache) ditch(elem *genSubList, cacheNum int) {

	// corner cases: head
	if elem == this.listHead[cacheNum] {
		this.listHead[cacheNum] = elem.prev

		// ...and tail
		if elem == this.listTail[cacheNum] {
			this.listTail[cacheNum] = elem.next
		} else {
			elem.prev.next = nil
		}

		// tail
	} else if elem == this.listTail[cacheNum] {
		this.listTail[cacheNum] = elem.next
		elem.next.prev = nil

		// middle
	} else {
		elem.prev.next = elem
		elem.next.prev = elem
	}

	// help the GC
	elem.next = nil
	elem.prev = nil

	// pity we can't nil contents...
}
