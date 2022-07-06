//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package couchbase

import (
	"bufio"
	"bytes"
	"container/heap"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/couchbase/gomemcached"
	memcached "github.com/couchbase/gomemcached/client"
	qerrors "github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
)

const _SS_MAX_CONCURRENT_VBSCANS_PER_SERVER = 1

// try to size this to avoid blocking, particularly during initial creation when all are scheduled for first run
const _RSW_CHANNEL_SIZE = 100 /* max concurrent scans */ * _SS_MAX_CONCURRENT_VBSCANS_PER_SERVER * 2 /* avg ranges per scan */
const _SS_MAX_DURATION = time.Minute * 10
const _SS_MAX_KEYS_PER_REQUEST = uint32(10240) // try to avoid more than one scan per v-bucket
const _SS_INIT_KEYS = 256
const _SS_SPILL_BUFFER = 32768
const _SS_SPILL_FILE_PATTERN = "ss_spill-*"

/*
 * Bucket functions for driving and consuming a scan.
 */

func (b *Bucket) StartKeyScan(scope string, collection string, ranges []*SeqScanRange, offset int64, limit int64, ordered bool,
	timeout time.Duration, pipelineSize int, kvTimeout time.Duration) (interface{}, qerrors.Error) {

	collId, _, err := b.GetCollectionCID(scope, collection, time.Time{})
	if err != nil {
		return nil, qerrors.NewSSError(qerrors.E_SS_CID_GET, err)
	}
	if limit == 0 {
		limit = -1
	}

	scan := &seqScan{
		collId:       collId,
		ranges:       ranges,
		ordered:      ordered,
		limit:        limit,
		offset:       offset,
		pipelineSize: pipelineSize,
		kvTimeout:    kvTimeout,
	}
	scan.ch = make(chan interface{}, 1)

	go scan.seqScanCoordinator(b, timeout)

	return scan, nil
}

func (b *Bucket) StopKeyScan(scan interface{}) qerrors.Error {
	ss, ok := scan.(*seqScan)
	if !ok {
		return qerrors.NewSSError(qerrors.E_SS_INVALID, "stop")
	}
	ss.cancel()
	return nil
}

func (b *Bucket) FetchKeys(scan interface{}, timeout time.Duration) ([]string, qerrors.Error, bool) {
	var keys []string
	var err qerrors.Error
	var timedout bool

	ss, ok := scan.(*seqScan)
	if !ok {
		return nil, qerrors.NewSSError(qerrors.E_SS_INVALID, "fetch"), false
	}

	if err == nil {
		if ss.inactive {
			return nil, qerrors.NewSSError(qerrors.E_SS_INACTIVE, "fetch"), false
		}
		to := time.NewTimer(timeout)
		select {
		case <-to.C:
			timedout = true
		case i := <-ss.ch:
			switch i := i.(type) {
			case []string:
				keys = i
			case qerrors.Error:
				err = i
			default:
				panic(fmt.Sprintf("Invalid type on scan channel: %T", i))
			}
		}
		if !timedout && !to.Stop() {
			<-to.C
		}
		to = nil
	}

	return keys, err, timedout
}

/*
 * Ranges pushed down to the KV.
 */

type SeqScanRange struct {
	exls bool
	s    []byte
	exle bool
	e    []byte
}

func (this *SeqScanRange) Init(s []byte, exls bool, e []byte, exle bool) {
	this.s, this.exls, this.exle = s, exls, exle
	this.e = append(e, byte(0x0)) // force exact termination
}

func (this *SeqScanRange) String() string {
	var b strings.Builder
	b.WriteRune('[')
	if this.exls {
		b.WriteRune('-')
	} else {
		b.WriteRune('+')
	}
	for _, c := range this.s {
		if c != 0xff && unicode.IsPrint(rune(c)) {
			b.WriteRune(rune(c))
		} else {
			b.WriteString(fmt.Sprintf("<%02x>", byte(c)))
		}
	}
	b.WriteRune(':')
	if this.exle {
		b.WriteRune('-')
	} else {
		b.WriteRune('+')
	}
	for _, c := range this.e {
		if c != 0xff && unicode.IsPrint(rune(c)) {
			b.WriteRune(rune(c))
		} else {
			b.WriteString(fmt.Sprintf("<%02x>", byte(c)))
		}
	}
	b.WriteRune(']')
	return b.String()
}

func (this *SeqScanRange) startFilter() []byte {
	return this.s
}

func (this *SeqScanRange) endFilter() []byte {
	return this.e
}

func (this *SeqScanRange) start() ([]byte, bool) {
	return this.s, this.exls
}

func (this *SeqScanRange) end() ([]byte, bool) {
	return this.e, this.exle
}

func (this *SeqScanRange) singleKey() bool {
	if this.exls != this.exle || this.exls != false {
		return false
	}
	if len(this.s) != len(this.e) && len(this.s) != len(this.e)-1 {
		return false
	}
	if len(this.s) == len(this.e)-1 && this.e[len(this.e)-1] != 0x0 {
		return false
	}
	for i := range this.s {
		if this.s[i] != this.e[i] {
			return false
		}
	}
	return true
}

/*
 * ready queue for seqScan
 */

type rQueue struct {
	sync.Mutex
	ready     []*vbRangeScan
	head      int
	tail      int
	cancelled bool
	timedout  bool
	cond      sync.Cond
}

func (this *rQueue) cancel() {
	this.Lock()
	this.cancelled = true
	this.cond.Signal()
	this.Unlock()
}

func (this *rQueue) timeout() {
	this.Lock()
	this.timedout = true
	this.cond.Signal()
	this.Unlock()
}

func (this *rQueue) enqueue(scan *vbRangeScan) {
	this.Lock()
	// don't need to worry about wrapping over head since ready is sized to maximum possible up front
	this.ready[this.tail] = scan
	this.tail++
	if this.tail == cap(this.ready) {
		this.tail = 0
	}
	this.ready = append(this.ready, scan)
	this.cond.Signal()
	this.Unlock()
}

func (this *rQueue) pop() *vbRangeScan {
	var rv *vbRangeScan
	this.Lock()
	for !this.cancelled && !this.timedout {
		if this.tail != this.head {
			rv = this.ready[this.head]
			this.head++
			if this.head == cap(this.ready) {
				this.head = 0
			}
			break
		} else {
			this.cond.Wait()
		}
	}
	this.Unlock()
	return rv
}

/*
 * Sequential scan.  Coordinator/aggregator for the individual v-bucket scans needed.
 */

type seqScan struct {
	ch           chan interface{}
	inactive     bool
	timedout     bool
	collId       uint32
	ranges       []*SeqScanRange
	ordered      bool
	limit        int64
	offset       int64
	pipelineSize int
	kvTimeout    time.Duration
	readyQueue   rQueue
}

func (this *seqScan) fetchLimit() uint32 {
	l := _SS_MAX_KEYS_PER_REQUEST
	if this.limit > 0 {
		l = uint32(this.limit + this.offset)
	}
	if l > _SS_MAX_KEYS_PER_REQUEST {
		l = _SS_MAX_KEYS_PER_REQUEST
	}
	return l
}

func (this *seqScan) timeout() {
	this.timedout = true
	this.readyQueue.timeout()
}

func (this *seqScan) cancel() {
	this.inactive = true
	this.readyQueue.cancel()
}

func (this *seqScan) reportError(err qerrors.Error) bool {
	rv := false
	select {
	case this.ch <- err:
		rv = true
	}
	this.inactive = true
	return rv
}

func (this *seqScan) reportResults(data []string, remaining *time.Timer) bool {
	rv := false
	if !this.inactive && !this.timedout {
		select {
		case <-remaining.C:
		case this.ch <- data:
			rv = true
		}
	}
	return rv
}

func (this *seqScan) getRange(n int) *SeqScanRange {
	return this.ranges[n]
}

func (this *seqScan) seqScanCoordinator(b *Bucket, scanTimeout time.Duration) {

	returnCount := int64(0)
	returnLimit := this.limit

	if scanTimeout <= 0 {
		scanTimeout = _SS_MAX_DURATION
	}
	maxEndTime := util.Now() + util.Time(scanTimeout)

	// snapshot of vbmap to use to distribute vb scans
	smap := b.VBServerMap()
	vblist := smap.VBucketMap

	numServers := len(smap.ServerList)

	// initialise / resize worker pool if necessary
	_RSW.initWorkers(numServers)
	defer _RSW.releaseWorkers()

	this.readyQueue.ready = make([]*vbRangeScan, len(vblist)*len(this.ranges))
	this.readyQueue.cond.L = &this.readyQueue

	var vbScans vbRangeScanHeap
	vbScans = (vbRangeScanHeap)(make([]*vbRangeScan, 0, len(vblist)*len(this.ranges)))

	defer func() {
		for _, v := range vbScans {
			v.release()
		}
	}()

	servers := make([]int, len(smap.ServerList))

	timeout := time.AfterFunc(maxEndTime.Sub(util.Now()), func() {
		this.timeout()
	})
	defer func() {
		timeout.Stop()
	}()

	for rNum := range this.ranges {
		var min, max int
		if this.getRange(rNum).singleKey() {
			min = int(b.VBHash(string(this.getRange(rNum).startFilter())))
			max = min + 1
		} else {
			min = 0
			max = len(vblist)
		}
		for i := min; i < max; i++ {
			server := 0
			if len(vblist[i]) > 0 {
				server = vblist[i][0]
			}
			vbs := &vbRangeScan{scan: this, b: b, vb: uint16(i), rng: rNum, server: server}
			vbScans = append(vbScans, vbs)
			if servers[server] < _SS_MAX_CONCURRENT_VBSCANS_PER_SERVER {
				if !this.queueVBScan(vbs) {
					vbScans.cancelAll()
					return
				}
				servers[server]++
			}
		}
	}
	heapPrepared := false

	completed := false
processing:
	for !completed && !this.timedout && !this.inactive {
		vbscan := this.readyQueue.pop()
		if vbscan == nil {
			break
		} else {
			servers[vbscan.server]--
			if vbscan.state == _VBS_WORKED {
				vbscan.state = _VBS_PROCESSING
			} else if vbscan.state != _VBS_ERROR {
			}
			if vbscan.err != nil || vbscan.state == _VBS_ERROR {
				// a worker encountered an error; cancel all workers and we're done
				vbScans.cancelAll()
				this.reportError(vbscan.err)
				return
			}
			if !this.ordered {
				start := 0
				if len(vbscan.keys) > 0 {
					if returnCount < this.offset && returnCount+int64(len(vbscan.keys)) > this.offset {
						start = int(this.offset - returnCount)
						returnCount += int64(start)
					} else if returnCount < this.offset {
						start = len(vbscan.keys)
						returnCount += int64(start)
					}
				}
				if len(vbscan.keys) > start && this.limit > 0 && returnLimit > 0 {
					if returnLimit < int64(len(vbscan.keys)-start) {
						vbscan.keys = vbscan.keys[:start+int(returnLimit)]
					}
					returnLimit -= int64(len(vbscan.keys) - start)
				}
				if len(vbscan.keys) > start {
					// forward results
					// read the keys
					if !vbscan.seek(start) {
					}
					for start < len(vbscan.keys) {
						batch := len(vbscan.keys) - start
						if batch > this.pipelineSize {
							batch = this.pipelineSize
						}
						keys := make([]string, 0, batch)
						for i := 0; i < batch; i++ {
							keys = append(keys, string(vbscan.current()))
							if !vbscan.advance() {
								if i+1 < batch {
									logging.Debugf("BUG: fewer keys than thought: i: %v, #keys: %v", i+1, batch)
								}
								break
							}
						}
						if !this.reportResults(keys, timeout) {
							vbScans.cancelAll()
							this.reportError(qerrors.NewSSError(qerrors.E_SS_TIMEOUT))
							return
						}
						returnCount += int64(len(keys))
						start += batch
					}
					err := vbscan.truncate()
					if err != nil {
						vbScans.cancelAll()
						this.reportError(err.(qerrors.Error))
						return
					}
				}
				if this.limit > 0 && returnLimit == 0 {
					vbScans.cancelAll()
					break processing
				}
				if vbscan.kvOpsComplete {
					// we're done with it, remove from map to reduce further processing
					vbScans.Remove(vbscan)
					vbscan.release()
					vbscan = nil
				} else {
					vbscan.state = _VBS_READY
				}
			} else { // ordered
				if !vbscan.start() {
					panic("v-bucket scan start failed")
				}
				// check to see if we have results from all vbs that are still producing keys
				count := 0
				for i := 0; i < len(vbScans); i++ {
					if vbScans[i].state == _VBS_PROCESSING {
						if vbScans[i].kvOpsComplete && len(vbScans[i].keys) == 0 {
							if vbscan == vbScans[i] {
								vbscan = nil
							}
							vbScans[i].release()
							if heapPrepared {
								heap.Remove(&vbScans, i)
							} else {
								if i < len(vbScans)-1 {
									copy(vbScans[i:], vbScans[i+1:])
								}
								vbScans = vbScans[:len(vbScans)-1]
							}
							i--
						} else if len(vbScans[i].keys) > 0 {
							count++
						}
					}
				}
				if count == len(vbScans) && len(vbScans) > 0 {
					if !heapPrepared {
						heap.Init(&vbScans)
						heapPrepared = true
					}
					// stream results (merge sorting) until one vb is empty
					batch := make([]string, 0, this.pipelineSize)

					for (this.limit == 0 || returnLimit > 0) && len(vbScans) > 0 {
						smallest := vbScans[0] // no need to pop as we'll Fix() in place (cheaper than Pop+Push)
						returnCount++
						if returnCount > this.offset {
							batch = append(batch, string(smallest.current())) // forces a copy
							if this.limit > 0 {
								returnLimit--
							}
						}
						if !smallest.advance() {
							if !smallest.kvOpsComplete {
								smallest.state = _VBS_READY
								err := smallest.truncate()
								if err != nil {
									vbScans.cancelAll()
									this.reportError(err.(qerrors.Error))
									return
								}
								vbscan = nil // include all in next search
								break
							} else {
								if vbscan == smallest {
									vbscan = nil
								}
								heap.Pop(&vbScans)
								smallest.release()
							}
						} else {
							// since the value has changed
							heap.Fix(&vbScans, 0)
						}
						if len(batch) == cap(batch) {
							// forward results
							if !this.reportResults(batch, timeout) {
								vbScans.cancelAll()
								this.reportError(qerrors.NewSSError(qerrors.E_SS_TIMEOUT))
								return
							}
							batch = make([]string, 0, this.pipelineSize)
						}
					}
					if len(batch) > 0 {
						// forward results
						if !this.reportResults(batch, timeout) {
							vbScans.cancelAll()
							this.reportError(qerrors.NewSSError(qerrors.E_SS_TIMEOUT))
							return
						}
					}
					if this.limit > 0 && returnLimit == 0 {
						vbScans.cancelAll()
						break processing
					}
				}
			}
			// scan vb list to see if we have any vbs that have no results and are not complete and the server isn't busy
			// try to ensure we don't get stuck just fetching one vb
			completed = true
			t := 0
			for s := range servers {
				t += servers[s]
			}
			max := len(servers) * _SS_MAX_CONCURRENT_VBSCANS_PER_SERVER
			if t < max {
				for _, vbs := range vbScans {
					// skip the one that has just reported to give others a chance
					if !vbs.kvOpsComplete && vbs.state == _VBS_READY && vbs != vbscan {
						completed = false
						if len(vbs.keys) == 0 && servers[vbs.server] < _SS_MAX_CONCURRENT_VBSCANS_PER_SERVER {
							if !this.queueVBScan(vbs) {
								vbScans.cancelAll()
								return
							}
							servers[vbs.server]++
							t++
							if t >= max {
								break
							}
						}
					} else if vbs.state == _VBS_ERROR {
						// will only be in error state if aborted during processing (panic)
						vbScans.cancelAll()
						if vbs.err == nil {
							vbs.err = qerrors.NewSSError(qerrors.E_SS_WORKER_ABORT)
						}
						this.reportError(vbs.err)
						return
					} else if vbs.state != _VBS_READY || len(vbs.keys) > 0 {
						completed = false
					}
				}
				if t < max && vbscan != nil && !vbscan.kvOpsComplete { // check the one that has just reported
					completed = false
					if len(vbscan.keys) == 0 && servers[vbscan.server] < _SS_MAX_CONCURRENT_VBSCANS_PER_SERVER {
						if !this.queueVBScan(vbscan) {
							return
						}
						servers[vbscan.server]++
					}
				}
			}
		}
	}
	if this.timedout {
		this.reportError(qerrors.NewSSError(qerrors.E_SS_TIMEOUT))
	}

	// make sure we don't leave anything lingering
	vbScans.cancelAll()

	// send final end of data indicator
	if !this.reportResults([]string(nil), timeout) {
		this.reportError(qerrors.NewSSError(qerrors.E_SS_TIMEOUT))
	}
}

func (this *seqScan) queueVBScan(vbscan *vbRangeScan) bool {
	vbscan.state = _VBS_SCHEDULED

	if this.timedout {
		vbscan.scan.reportError(qerrors.NewSSError(qerrors.E_SS_TIMEOUT))
		return false
	}
	if this.inactive {
		return false
	}
	_RSW.queueScan(vbscan.server, vbscan)
	return true
}

/*
 * Individual v-bucket scan.
 */

type vbRsState int

const (
	_VBS_READY vbRsState = iota
	_VBS_SCHEDULED
	_VBS_WORKING
	_VBS_WORKED
	_VBS_PROCESSING
	_VBS_ERROR
	_VBS_CANCELLED
)

func (s vbRsState) String() string {
	switch s {
	case _VBS_READY:
		return "READY"
	case _VBS_SCHEDULED:
		return "SCHEDULED"
	case _VBS_WORKING:
		return "WORKING"
	case _VBS_WORKED:
		return "WORKED"
	case _VBS_PROCESSING:
		return "PROCESSING"
	case _VBS_ERROR:
		return "ERROR"
	case _VBS_CANCELLED:
		return "CANCELLED"
	}
	return "?"
}

type vbRangeScan struct {
	scan          *seqScan
	b             *Bucket
	vb            uint16
	rng           int
	server        int
	state         vbRsState
	kvOpsComplete bool
	continueFrom  []byte

	spill      *os.File
	keys       []uint32
	currentKey int
	reader     *bufio.Reader
	head       []byte

	err qerrors.Error
}

func (this *vbRangeScan) sameScan(other *vbRangeScan) bool {
	if this.vb != other.vb ||
		this.server != other.server ||
		this.b != other.b {

		return false
	}
	ts, tsi := this.startFrom()
	os, osi := other.startFrom()
	if tsi != osi || bytes.Compare(ts, os) != 0 {
		return false
	}
	te, tei := this.endWith()
	oe, oei := other.endWith()
	if tei != oei || bytes.Compare(te, oe) != 0 {
		return false
	}
	return true
}

func (this *vbRangeScan) startFrom() ([]byte, bool) {
	if this.continueFrom != nil {
		return this.continueFrom, true
	}
	return this.scan.ranges[this.rng].start()
}

func (this *vbRangeScan) endWith() ([]byte, bool) {
	return this.scan.ranges[this.rng].end()
}

func (this *vbRangeScan) release() {
	this.truncate() // release space
	this.spill = nil
	this.reader = nil
	this.keys = nil
	this.currentKey = 0
}

func (this *vbRangeScan) truncate() error {
	var err error
	if this.keys != nil {
		this.keys = this.keys[:0]
	}
	if this.spill != nil {
		var size int64
		size, err = this.spill.Seek(0, os.SEEK_END)
		if err == nil {
			_, err = this.spill.Seek(0, os.SEEK_SET)
			if err == nil {
				err = this.spill.Truncate(0)
				if err == nil {
					util.ReleaseTemp(this.spill.Name(), size)
				}
			}
		}
	}
	if err == nil {
		return nil
	}
	return qerrors.NewSSError(qerrors.E_SS_SPILL, err)
}

func (this *vbRangeScan) start() bool {
	rv := this.seek(0)
	if len(this.keys) == 0 {
		rv = true
	}
	return rv
}

func (this *vbRangeScan) seek(n int) bool {
	if n < 0 || n >= len(this.keys) {
		return false
	}
	off := int64(0)
	if n > 0 {
		off = int64(this.keys[n-1])
	}
	_, err := this.spill.Seek(off, os.SEEK_SET)
	if err != nil {
		return false
	}
	if this.reader == nil {
		this.reader = bufio.NewReaderSize(this.spill, _SS_SPILL_BUFFER)
	} else {
		this.reader.Reset(this.spill)
	}
	this.currentKey = n
	return this.readCurrent()
}

func (this *vbRangeScan) readCurrent() bool {
	l := this.keyLen(this.currentKey)
	if l == -1 {
		return false
	}
	if len(this.head) < l {
		this.head = make([]byte, 0, l)
	}
	this.head = this.head[:l]
	_, err := io.ReadFull(this.reader, this.head)
	return err == nil
}

func (this *vbRangeScan) current() []byte {
	return this.head
}

func (this *vbRangeScan) keyLen(n int) int {
	if n < 0 || n >= len(this.keys) {
		return -1
	}
	if n == 0 {
		return int(this.keys[0])
	}
	return int(this.keys[n] - this.keys[n-1])
}

func (this *vbRangeScan) advance() bool {
	if this.currentKey >= len(this.keys) {
		return false
	}
	this.currentKey++
	return this.readCurrent()
}

func (this *vbRangeScan) vbucket() uint16 {
	return this.vb
}

func (this *vbRangeScan) rangeNum() int {
	return this.rng
}

func (this *vbRangeScan) reportError(err qerrors.Error) {
	if this.state == _VBS_WORKING {
		this.state = _VBS_WORKED
	}
	this.err = err
	this.scan.readyQueue.enqueue(this)
}

func (this *vbRangeScan) sendData() {
	this.scan.readyQueue.enqueue(this)
}

/*
 * Heap implementation to support ordered results.
 */

type vbRangeScanHeap []*vbRangeScan

func (this *vbRangeScanHeap) Len() int { return len(*this) }
func (this *vbRangeScanHeap) Less(i, j int) bool {
	return bytes.Compare((*this)[i].current(), (*this)[j].current()) < 0
}
func (this *vbRangeScanHeap) Swap(i, j int)      { (*this)[i], (*this)[j] = (*this)[j], (*this)[i] }
func (this *vbRangeScanHeap) Push(x interface{}) { *this = append(*this, x.(*vbRangeScan)) }
func (this *vbRangeScanHeap) Pop() interface{} {
	i := len(*this) - 1
	last := (*this)[i]
	*this = (*this)[:i]
	return last
}

func (this *vbRangeScanHeap) Remove(vbscan *vbRangeScan) {
	for i := range *this {
		if (*this)[i] == vbscan {
			if i < len(*this) {
				copy((*this)[i:], (*this)[i+1:])
			}
			*this = (*this)[:len(*this)-1]
			return
		}
	}
}

func (this *vbRangeScanHeap) cancelAll() {
	for i := range *this {
		(*this)[i].state = _VBS_CANCELLED
	}
}

/*
 * Grouped list container for scans.  Handles actual KV interaction.
 */

type vbScanShare struct {
	next  *vbScanShare
	scans []*vbRangeScan
}

func (this *vbScanShare) runScan() {
	var scanReq int64
	var scanRes int64

	// walk scans list and remove any cancelled scans
	for i := 0; i < len(this.scans); i++ {
		if this.scans[i].scan.inactive {
			this.removeScan(i)
			i--
		}
	}
	if len(this.scans) == 0 {
		return
	}

	for _, s := range this.scans {
		s.state = _VBS_WORKING
	}

	defer func() {
		for _, s := range this.scans {
			if s.state == _VBS_WORKING {
				s.state = _VBS_ERROR
			}
		}
	}()

	// since shared (same) with all, just use the first one's
	vbucket := this.scans[0].vbucket()
	b := this.scans[0].b

	var err error
	var response *gomemcached.MCResponse
	var conn *memcached.Client
	var pool *connectionPool

	uuid := make([]byte, 16)
	opaque := uint32(0)

	cancelScanFunc := func(mc *memcached.Client, vb uint16) error {
		mc.CancelRangeScan(vbucket, uuid, opaque)
		return nil
	}

	desc := &doDescriptor{useReplicas: false, version: b.Version, maxTries: b.backOffRetries(), retry: true}
	for desc.attempts = 0; desc.attempts < desc.maxTries; {
		conn, pool, err = b.getVbConnection(uint32(vbucket), desc)
		if err != nil {
			if desc.retry {
				desc.attempts++
				continue
			}
			this.reportErrorToAll(qerrors.NewSSError(qerrors.E_SS_CONN, err))
			return
		}
		break
	}
	if conn == nil {
		b.do3(vbucket, cancelScanFunc, false, false, 1) // unlikely to succeed, but try anyway
		this.reportErrorToAll(qerrors.NewSSError(qerrors.E_SS_CONN, err))
		return
	}
	if DefaultTimeout > 0 {
		conn.SetDeadline(getDeadline(noDeadline, DefaultTimeout))
	} else {
		conn.SetDeadline(noDeadline)
	}
	if desc.replica > 0 {
		conn.SetReplica(true)
	}

	defer func() {
		if desc.discard {
			pool.Discard(conn)
		} else {
			conn.SetReplica(false)
			pool.Return(conn)
		}
	}()

	start, exclStart := this.scans[0].startFrom()
	end, exclEnd := this.scans[0].endWith()

	response, err = conn.CreateRangeScan(vbucket, this.scans[0].scan.collId, start, exclStart, end, exclEnd)
	if err != nil {
		resp, ok := err.(*gomemcached.MCResponse)
		if ok && resp.Status == gomemcached.KEY_ENOENT {
			// success but no data
			for _, s := range this.scans {
				s.kvOpsComplete = true
				if s.keys != nil {
					s.keys = s.keys[:0]
				}
				if s.spill != nil {
					err = s.truncate()
					if err != nil {
						s.reportError(qerrors.NewSSError(qerrors.E_SS_SPILL, err))
					}
				}
			}
			if !this.sendDataToAll() {
				logging.Debugf("DATA SEND FAILED")
			}
			return
		}
		this.reportErrorToAll(qerrors.NewSSError(qerrors.E_SS_CREATE, err))
		return
	}
	opaque = response.Opaque
	copy(uuid, response.Body[0:16])

	for n := 0; n < len(this.scans); {
		if this.scans[n].state == _VBS_CANCELLED {
			this.removeScan(n)
		} else {
			n++
		}
	}
	if len(this.scans) == 0 {
		b.do3(vbucket, cancelScanFunc, false, false, 1)
		return
	}

	err = conn.ContinueRangeScan(vbucket, uuid, opaque, this.scans[0].scan.fetchLimit(), 0)
	if err != nil {
		this.reportErrorToAll(qerrors.NewSSError(qerrors.E_SS_CONTINUE, err))
		return
	}
	scanReq++
	for n := 0; n < len(this.scans); {
		s := this.scans[n]
		if s.spill == nil {
			s.spill, err = util.CreateTemp(_SS_SPILL_FILE_PATTERN, true)
			if err != nil {
				s.reportError(qerrors.NewSSError(qerrors.E_SS_SPILL, err))
				if !this.removeScan(n) {
					desc.discard = true // as it will need to drain an unknown number of responses otherwise
					b.do3(vbucket, cancelScanFunc, false, false, 1)
					return
				}
				continue
			}
		} else {
			err = s.truncate()
			if err != nil {
				s.reportError(err.(qerrors.Error))
				if !this.removeScan(n) {
					desc.discard = true // as it will need to drain an unknown number of responses otherwise
					b.do3(vbucket, cancelScanFunc, false, false, 1)
					return
				}
				continue
			}
		}
		if s.keys == nil {
			s.keys = make([]uint32, 0, _SS_INIT_KEYS)
		} else {
			s.keys = s.keys[:0]
		}
		n++
	}
	offset := uint32(0)
	// loop receiving and accumulating results into all the vbRangeScans
processing:
	for {
		for n := 0; n < len(this.scans); {
			if this.scans[n].state == _VBS_CANCELLED {
				this.removeScan(n)
			} else {
				n++
			}
		}
		if len(this.scans) == 0 {
			desc.discard = true // as it will need to drain an unknown number of responses otherwise
			b.do3(vbucket, cancelScanFunc, false, false, 1)
			break processing
		}

		// Receive a CONTINUE SCAN response
		response, err = conn.ReceiveWithDeadline(time.Now().Add(this.scans[0].scan.kvTimeout))
		if err != nil {
			resp, ok := err.(*gomemcached.MCResponse)
			if ok && resp.Status != gomemcached.SUCCESS &&
				resp.Status != gomemcached.RANGE_SCAN_MORE &&
				resp.Status != gomemcached.RANGE_SCAN_COMPLETE {

				this.reportErrorToAll(qerrors.NewSSError(qerrors.E_SS_CONTINUE, err))
				break processing
			}
		}
		scanRes++

		if len(response.Body) > 0 {
			var l, p uint32
			for i := 0; i < len(response.Body); {
				// read a length... leb128 format (use 32-bits even though length will likely never be this large)
				l = uint32(0)
				for shift := 0; i < len(response.Body); {
					p = uint32(response.Body[i])
					i++
					l |= (p & uint32(0x7f)) << shift
					if p&uint32(0x80) == 0 {
						break
					}
					shift += 7
				}
				if i+int(l) > len(response.Body) {
					l = uint32(len(response.Body) - int(i))
				}
				offset += l
				for n := 0; n < len(this.scans); {
					if !util.UseTemp(this.scans[n].spill.Name(), int64(l)) {
						this.scans[n].reportError(qerrors.NewTempFileQuotaExceededError())
						if !this.removeScan(n) {
							desc.discard = true // as it will need to drain an unknown number of responses otherwise
							b.do3(vbucket, cancelScanFunc, false, false, 1)
							break processing
						}
						continue
					}
					_, err = this.scans[n].spill.Write(response.Body[i : i+int(l)])
					if err != nil {
						this.scans[n].reportError(qerrors.NewSSError(qerrors.E_SS_SPILL, err))
						if !this.removeScan(n) {
							desc.discard = true // as it will need to drain an unknown number of responses otherwise
							b.do3(vbucket, cancelScanFunc, false, false, 1)
							break processing
						}
						continue
					} else {
						if len(this.scans[n].keys) == cap(this.scans[n].keys) {
							nw := make([]uint32, len(this.scans[n].keys), cap(this.scans[n].keys)<<2)
							copy(nw, this.scans[n].keys)
							this.scans[n].keys = nw
						}
						this.scans[n].keys = append(this.scans[n].keys, offset)
					}
					n++
				}
				i += int(l)
			}
		}

		if len(response.Body) == 0 ||
			response.Status == gomemcached.RANGE_SCAN_MORE ||
			response.Status == gomemcached.RANGE_SCAN_COMPLETE {

			if response.Status != gomemcached.RANGE_SCAN_MORE {
				// end of scan
				for _, s := range this.scans {
					s.kvOpsComplete = true
				}
			}
			if !this.sendDataToAll() {
				logging.Debugf("DATA SEND FAILED")
			}
			break processing
		}
	}
}

func (this *vbScanShare) removeScan(n int) bool {
	if n < 0 {
		n = 0
	}
	if n < len(this.scans)-1 {
		copy(this.scans[n:], this.scans[n+1:])
	}
	this.scans = this.scans[:len(this.scans)-1]
	return len(this.scans) != 0
}

func (this *vbScanShare) reportErrorToAll(err qerrors.Error) {
	for _, s := range this.scans {
		s.reportError(err)
	}
}

func (this *vbScanShare) sendDataToAll() bool {
	succeeded := false
	for _, s := range this.scans {
		if s.state == _VBS_CANCELLED {
			continue
		}
		s.state = _VBS_WORKED
		s.sendData()
		succeeded = true
	}
	return succeeded
}

/*
 * Pooled workers and work lists organised by data server.
 */

type rswQueue struct {
	sync.RWMutex
	scans *vbScanShare
	abort bool
	cond  sync.Cond
}

type rswControl struct {
	sync.RWMutex
	queues      []*rswQueue
	nextRSWID   int
	activeScans int
	shrink      bool
}

var _RSW = &rswControl{}

func (this *rswControl) queueScan(qNum int, vbscan *vbRangeScan) {
	this.RLock()
	queue := this.queues[qNum]
	this.RUnlock()
	queue.Lock()
	if queue.scans == nil {
		scan := &vbScanShare{}
		scan.scans = append(scan.scans, vbscan)
		queue.scans = scan
	} else {
		// walk list looking for same scans
		for e := queue.scans; e != nil; e = e.next {
			if e.scans[0].sameScan(vbscan) {
				// share if the same
				e.scans = append(e.scans, vbscan)
				break
			} else if e.next == nil {
				// new at end of list
				scan := &vbScanShare{}
				scan.scans = append(scan.scans, vbscan)
				e.next = scan
				break
			}
		}
	}
	queue.cond.Signal()
	queue.Unlock()
}

func (this *rswControl) nextScanShare(qNum int) *vbScanShare {
	this.RLock()
	queue := this.queues[qNum]
	this.RUnlock()
	queue.Lock()
	rv := queue.scans
	if queue.scans != nil {
		queue.scans = rv.next
		rv.next = nil
	}
	queue.Unlock()
	return rv
}

func (this *rswControl) initWorkers(servers int) {

	if servers < 1 {
		servers = 1
	}

	this.Lock()
	if this.queues == nil {
		this.queues = make([]*rswQueue, 0, 128)
	}
	for len(this.queues) < servers {
		nq := &rswQueue{}
		nq.cond.L = nq
		this.queues = append(this.queues, nq)
		for i := 0; i < _SS_MAX_CONCURRENT_VBSCANS_PER_SERVER; i++ {
			go this.runWorker(len(this.queues)-1, this.nextRSWID)
			this.nextRSWID++
		}
	}
	if len(this.queues) > servers {
		this.shrink = true
	}
	this.activeScans++
	this.Unlock()
}

func (this *rswControl) releaseWorkers() {
	this.Lock()
	this.activeScans--
	if this.activeScans <= 0 {
		if this.activeScans != 0 {
			this.activeScans = 0
		}
		if this.shrink {
			for _, q := range this.queues {
				q.Lock()
				q.abort = true
				q.cond.Signal()
				q.Unlock()
			}
			this.queues = nil
			this.shrink = false
		}
	}
	this.Unlock()
}

func (this *rswControl) runWorker(qNum int, id int) {
	defer func() {
		r := recover()
		if r != nil {
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, false)
			s := string(buf[0:n])
			logging.Severef("Range scan worker [%v] {%v} panic: %v\n%v", id, qNum, r, s)
		}
		// cannot panic and die
		go this.runWorker(qNum, id)
	}()

	queue := this.queues[qNum]
	queue.Lock()
	for {
		if queue.abort {
			queue.Unlock()
			// pool is shrinking so exit gracefully
			return
		} else if queue.scans == nil {
			queue.cond.Wait()
		} else {
			queue.Unlock()
			ss := this.nextScanShare(qNum)
			if ss != nil {
				ss.runScan()
			}
			queue.Lock()
		}
	}
}
