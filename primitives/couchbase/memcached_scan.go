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
	"container/list"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/couchbase/gomemcached"
	memcached "github.com/couchbase/gomemcached/client"
	qerrors "github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/sort"
	"github.com/couchbase/query/util"
)

var _SS_MAX_CONCURRENT_VBSCANS_PER_SERVER = -1
var initL sync.Mutex

const _SS_SMALL_RESULT_SET = 100
const _SS_MAX_DURATION = time.Minute * 10
const _SS_MAX_KEYS_PER_REQUEST = uint32(10240) // try to avoid more than one scan per v-bucket
const _SS_INIT_KEYS = 256
const _SS_KEY_BUFFER = 10240
const _SS_SPILL_BUFFER = 10240
const _SS_SPILL_FILE_PATTERN = "ss_spill-*"
const _SS_MAX_WORKER_IDLE = time.Minute * 60
const _SS_MONITOR_INTERVAL = time.Minute * 15
const _SS_MIN_SAMPLE_SIZE = 1
const _SS_MIN_SCAN_SIZE = 256
const _SS_RETRIES = 35
const _SS_RETRY_DELAY = time.Millisecond * 100

/*
 * Bucket functions for driving and consuming a scan.
 */

func (b *Bucket) StartKeyScan(requestId string, log logging.Log, collId uint32, scope string, collection string,
	ranges []*SeqScanRange, offset int64, limit int64, ordered bool, timeout time.Duration, pipelineSize int,
	kvTimeout time.Duration, serverless bool) (interface{}, qerrors.Error) {

	if log == nil {
		log = logging.NULL_LOG
	}

	if limit == 0 {
		limit = -1
	}

	if scope != "" && collection != "" {
		var err error
		collId, _, err = b.GetCollectionCID(scope, collection, time.Time{})
		if err != nil {
			return nil, qerrors.NewSSError(qerrors.E_SS_CID_GET, err)
		}
	}

	scan := NewSeqScan(requestId, log, collId, ranges, offset, limit, ordered, pipelineSize, kvTimeout, serverless)

	logging.Debuga(func() string { return scan.String() }, log)
	go scan.coordinator(b, timeout)

	return scan, nil
}

func (b *Bucket) StartRandomScan(requestId string, log logging.Log, collId uint32, scope string, collection string,
	sampleSize int, timeout time.Duration, pipelineSize int, kvTimeout time.Duration, serverless bool) (
	interface{}, qerrors.Error) {

	if log == nil {
		log = logging.NULL_LOG
	}

	if scope != "" && collection != "" {
		var err error
		collId, _, err = b.GetCollectionCID(scope, collection, time.Time{})
		if err != nil {
			return nil, qerrors.NewSSError(qerrors.E_SS_CID_GET, err)
		}
	}

	scan := NewRandomScan(requestId, log, collId, sampleSize, pipelineSize, kvTimeout, serverless)

	logging.Debuga(func() string { return scan.String() }, log)
	go scan.coordinator(b, timeout)

	return scan, nil
}

func (b *Bucket) StopKeyScan(scan interface{}) (uint64, qerrors.Error) {
	ss, ok := scan.(*seqScan)
	if !ok {
		return 0, qerrors.NewSSError(qerrors.E_SS_INVALID, "stop")
	}
	ss.cancel()
	return ss.runits, nil
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
			select {
			case i := <-ss.ch:
				if e, ok := i.(qerrors.Error); ok {
					return nil, e, false
				}
			default:
			}
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

func (this *SeqScanRange) String() string {
	var b strings.Builder
	if this.exls {
		b.WriteString("{-start:")
	} else {
		b.WriteString("{+start:")
	}
	b.WriteString(fmt.Sprintf("%v", this.s))
	if this.exle {
		b.WriteString(",-end:")
	} else {
		b.WriteString(",+end:")
	}
	b.WriteString(fmt.Sprintf("%v", this.e))
	b.WriteString("}")
	return b.String()
}

func (this *SeqScanRange) Init(s []byte, exls bool, e []byte, exle bool) {
	this.s, this.exls, this.exle = s, exls, exle
	this.e = append(e, byte(0x0)) // force exact termination
}

func (this *SeqScanRange) start() ([]byte, bool) {
	return this.s, this.exls
}

func (this *SeqScanRange) end() ([]byte, bool) {
	return this.e, this.exle
}

func (this *SeqScanRange) isSingleKey() bool {
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
 * V-bucket scan result ready queue.
 */

type rQueue struct {
	sync.Mutex
	ready     *list.List
	cancelled bool
	timedout  bool
	cond      sync.Cond
}

func (this *rQueue) init() {
	this.ready = list.New()
	this.cond.L = this
}

func (this *rQueue) cancel() {
	this.Lock()
	this.cancelled = true
	this.cond.Broadcast()
	this.Unlock()
}

func (this *rQueue) timeout() {
	this.Lock()
	this.timedout = true
	this.cond.Broadcast()
	this.Unlock()
}

func (this *rQueue) enqueue(scan *vbRangeScan) {
	this.Lock()
	this.ready.PushBack(scan)
	this.cond.Broadcast()
	this.Unlock()
}

func (this *rQueue) pop() *vbRangeScan {
	var rv *vbRangeScan
	this.Lock()
	for !this.cancelled && !this.timedout {
		e := this.ready.Front()
		if e != nil {
			rv = e.Value.(*vbRangeScan)
			this.ready.Remove(e)
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
	requestId    string
	log          logging.Log
	ch           chan interface{}
	abortch      chan bool
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
	fetchLimit   uint32
	serverless   bool
	runits       uint64
	sampleSize   int
}

func NewSeqScan(requestId string, log logging.Log, collId uint32, ranges []*SeqScanRange, offset int64, limit int64, ordered bool,
	pipelineSize int, kvTimeout time.Duration, serverless bool) *seqScan {

	scan := &seqScan{
		requestId:    requestId,
		log:          log,
		collId:       collId,
		ranges:       ranges,
		ordered:      ordered,
		limit:        limit,
		offset:       offset,
		pipelineSize: pipelineSize,
		kvTimeout:    kvTimeout,
		serverless:   serverless,
	}
	scan.ch = make(chan interface{}, 1)
	scan.abortch = make(chan bool, 1)
	scan.fetchLimit = _SS_MAX_KEYS_PER_REQUEST
	if scan.limit > 0 {
		scan.fetchLimit = uint32(scan.limit + scan.offset)
	}
	if scan.fetchLimit > _SS_MAX_KEYS_PER_REQUEST {
		scan.fetchLimit = _SS_MAX_KEYS_PER_REQUEST
	}
	scan.readyQueue.init()
	return scan
}

func NewRandomScan(requestId string, log logging.Log, collId uint32, sampleSize int, pipelineSize int, kvTimeout time.Duration,
	serverless bool) *seqScan {

	if sampleSize <= _SS_MIN_SAMPLE_SIZE {
		sampleSize = _SS_MIN_SAMPLE_SIZE
	}
	scan := &seqScan{
		requestId:    requestId,
		log:          log,
		collId:       collId,
		sampleSize:   sampleSize,
		pipelineSize: pipelineSize,
		kvTimeout:    kvTimeout,
		serverless:   serverless,
	}
	scan.ch = make(chan interface{}, 1)
	scan.abortch = make(chan bool, 1)
	scan.readyQueue.init()
	return scan
}

func (this *seqScan) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("{%p", this))
	b.WriteString(this.requestId)
	b.WriteString(",collId:")
	b.WriteString(fmt.Sprintf("%d", this.collId))
	b.WriteString(",sampleSize:")
	b.WriteString(fmt.Sprintf("%v", this.sampleSize))
	b.WriteString(",ordered:")
	b.WriteString(fmt.Sprintf("%v", this.ordered))
	b.WriteString(",limit:")
	b.WriteString(fmt.Sprintf("%v", this.limit))
	b.WriteString(",offset:")
	b.WriteString(fmt.Sprintf("%v", this.offset))
	b.WriteString(",pipelineSize:")
	b.WriteString(fmt.Sprintf("%v", this.pipelineSize))
	b.WriteString(",fetchLimit:")
	b.WriteString(fmt.Sprintf("%v", this.fetchLimit))
	b.WriteString(",ranges:[")
	for i, r := range this.ranges {
		if i != 0 {
			b.WriteString(",")
		}
		b.WriteString(r.String())
	}
	b.WriteString("]}")
	return b.String()
}

func (this *seqScan) timeout() {
	this.timedout = true
	this.readyQueue.timeout()
	this.abortch <- true
}

func (this *seqScan) cancel() {
	this.inactive = true
	this.readyQueue.cancel()
	this.abortch <- true
}

func (this *seqScan) addRU(ru uint64) {
	if ru != 0 {
		atomic.AddUint64(&this.runits, ru)
	}
}

func (this *seqScan) reportError(err qerrors.Error) bool {
	rv := false
	if !this.inactive {
		select {
		case <-this.abortch:
		case this.ch <- err:
			rv = true
		}
		this.inactive = true
	}
	return rv
}

func (this *seqScan) reportResults(data []string) bool {
	rv := false
	if !this.inactive && !this.timedout {
		select {
		case <-this.abortch:
		case this.ch <- data:
			rv = true
		}
	}
	return rv
}

func (this *seqScan) getRange(n int) *SeqScanRange {
	return this.ranges[n]
}

func (this *seqScan) coordinator(b *Bucket, scanTimeout time.Duration) {
	if _SS_MAX_CONCURRENT_VBSCANS_PER_SERVER == -1 {
		initL.Lock()
		if _SS_MAX_CONCURRENT_VBSCANS_PER_SERVER == -1 {
			minCC := 1024
			nodes := b.Nodes()
			for _, n := range nodes {
				found := false
				for _, s := range n.Services {
					if s == "kv" {
						found = true
						break
					}
				}
				if !found {
					continue
				}
				var cc int
				switch ccnt := n.CpuCount.(type) {
				case float64:
					cc = int(ccnt / float64(len(n.Services)) * 0.75)
				default:
					logging.Infof("Unable to determine CPU count for %v[%v]", n.NodeUUID, n.Hostname)
					cc = 1
				}
				if cc < minCC {
					minCC = cc
				}
			}
			if minCC < 1 {
				minCC = 1
			}
			_SS_MAX_CONCURRENT_VBSCANS_PER_SERVER = minCC
			logging.Infof("Max concurrent v-bucket range scans per server set to: %v", _SS_MAX_CONCURRENT_VBSCANS_PER_SERVER)
		}
		initL.Unlock()
	}

	returnCount := int64(0)
	returnLimit := this.limit

	if scanTimeout <= 0 {
		scanTimeout = _SS_MAX_DURATION
	}

	smap := b.VBServerMap()
	vblist := smap.VBucketMap

	numServers := len(smap.ServerList)
	if numServers < 1 {
		logging.Severef("Sequential scan coordinator: [%p] invalid VB map for bucket %v - no server list", this, b.Name)
		this.reportError(qerrors.NewSSError(qerrors.E_SS_FAILED))
		return
	}

	// initialise / resize worker pool if necessary
	_RSW.initWorkers(numServers)

	var vbScans vbRangeScanHeap
	vbScans = (vbRangeScanHeap)(make([]*vbRangeScan, 0, len(vblist)*len(this.ranges)))

	cancelAll := func() {
		for i := range vbScans {
			vbScans[i].deferRelease = (vbScans[i].state == _VBS_WORKING)
			atomic.StoreInt32((*int32)(&vbScans[i].state), int32(_VBS_CANCELLED))
		}
		_RSW.cancelQueuedScans(this)
	}

	defer func() {
		for _, v := range vbScans {
			if !v.deferRelease {
				v.release()
			}
		}
	}()

	queues := make([]int, numServers)
	timeout := time.AfterFunc(scanTimeout, this.timeout)
	defer func() {
		timeout.Stop()
	}()

	if this.sampleSize != 0 {
		returnLimit = int64(this.sampleSize)
		var sampleSize int
		if this.sampleSize == math.MaxInt {
			sampleSize = this.sampleSize
		} else {
			sampleSize = (this.sampleSize + len(vblist) - 1) / len(vblist)
			// adjust to compensate for probability based sampling in the range scan meaning we're likely to get
			// fewer samples than requested
			sampleSize += int(math.Ceil(float64(sampleSize) * 0.2153 * math.Pow(float64(sampleSize), -0.398)))
			returnLimit = int64(sampleSize * len(vblist))
		}
		for vb := 0; vb < len(vblist); vb++ {
			server := 0
			if len(vblist[vb]) > 0 {
				// first server that's in the list
				for n := 0; n < len(vblist[vb]); n++ {
					server = vblist[vb][n]
					if server < numServers {
						break
					}
				}
				if server >= numServers {
					logging.Severef("Sequential scan coordinator: [%p] Invalid server: %d (max valid: %d)",
						this, server, numServers-1)
					this.reportError(qerrors.NewSSError(qerrors.E_SS_FAILED))
					cancelAll()
					return
				}
			}
			vbs := &vbRangeScan{scan: this, b: b, vb: uint16(vb), queue: server, sampleSize: sampleSize, retries: _SS_RETRIES}
			vbScans = append(vbScans, vbs)
		}
		// pick a random scan to start from so there is a greater chance of spreading load
		n := rand.Int() % len(vbScans)
		queued := 0
		for i := 0; i < len(vbScans) && queued < len(queues)*_SS_MAX_CONCURRENT_VBSCANS_PER_SERVER; i++ {
			if queues[vbScans[n].queue] < _SS_MAX_CONCURRENT_VBSCANS_PER_SERVER {
				if !this.queueVBScan(vbScans[n]) {
					cancelAll()
					return
				}
				queues[vbScans[n].queue]++
				queued++
			}
			n++
			if n == len(vbScans) {
				n = 0
			}
		}
	} else {
		queued := 0
		if !this.ordered && this.fetchLimit < _SS_SMALL_RESULT_SET {
			// queue only one request initially as we're likely to serviced by just one
			queued = len(queues)*_SS_MAX_CONCURRENT_VBSCANS_PER_SERVER - 1
		}
		for rNum := range this.ranges {
			var min, max int
			var singleKey bool
			if this.getRange(rNum).isSingleKey() {
				f, _ := this.getRange(rNum).start()
				min = int(b.VBHash(string(f)))
				max = min + 1
				singleKey = true
			} else {
				min = 0
				max = len(vblist)
				singleKey = false
			}
			for vb := min; vb < max; vb++ {
				server := 0
				if len(vblist[vb]) > 0 {
					// first server that's in the list
					for n := 0; n < len(vblist[vb]); n++ {
						server = vblist[vb][n]
						if server < numServers {
							break
						}
					}
					if server >= numServers {
						logging.Severef("Sequential scan coordinator: [%p] Invalid server: %d (max valid: %d)",
							this, server, numServers-1)
						this.reportError(qerrors.NewSSError(qerrors.E_SS_FAILED))
						cancelAll()
						return
					}
				}
				vbs := &vbRangeScan{scan: this,
					b:         b,
					vb:        uint16(vb),
					rng:       rNum,
					queue:     server,
					singleKey: singleKey,
					retries:   _SS_RETRIES,
				}
				vbScans = append(vbScans, vbs)
			}
			if queued < len(queues)*_SS_MAX_CONCURRENT_VBSCANS_PER_SERVER {
				// pick a random scan to start from so there is a greater chance of spreading load
				n := rand.Int() % len(vbScans)
				for i := 0; i < len(vbScans); i++ {
					if queues[vbScans[n].queue] < _SS_MAX_CONCURRENT_VBSCANS_PER_SERVER {
						if !this.queueVBScan(vbScans[n]) {
							cancelAll()
							return
						}
						queues[vbScans[n].queue]++
						queued++
					}
					n++
					if n == len(vbScans) {
						n = 0
					}
				}
			}
		}
	}
	heapPrepared := false

	completed := false
processing:
	for !completed && !this.timedout && !this.inactive && len(vbScans) > 0 {
		vbscan := this.readyQueue.pop()
		if vbscan == nil {
			break
		} else {
			queues[vbscan.queue]--
			if vbscan.state == _VBS_WORKED {
				vbscan.state = _VBS_PROCESSING
			} else if vbscan.state == _VBS_CANCELLED {
				vbScans.Remove(vbscan)
				vbscan.release()
				continue
			}
			if vbscan.err != nil || vbscan.state == _VBS_ERROR {
				// a worker encountered an error; cancel all workers and we're done
				cancelAll()
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
						logging.Debugf("BUG: scan seek failed", this.log)
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
									logging.Debugf("BUG: fewer keys than thought: i: %v, #keys: %v", i+1, batch, this.log)
								}
								break
							}
						}
						if !this.reportResults(keys) {
							cancelAll()
							this.reportError(qerrors.NewSSError(qerrors.E_SS_TIMEOUT))
							return
						}
						returnCount += int64(len(keys))
						start += batch
					}
					err := vbscan.truncate()
					if err != nil {
						cancelAll()
						this.reportError(err.(qerrors.Error))
						return
					}
				}
				if this.limit > 0 && returnLimit == 0 {
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
									cancelAll()
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
							if !this.reportResults(batch) {
								cancelAll()
								this.reportError(qerrors.NewSSError(qerrors.E_SS_TIMEOUT))
								return
							}
							batch = make([]string, 0, this.pipelineSize)
						}
					}
					if len(batch) > 0 {
						// forward results
						if !this.reportResults(batch) {
							cancelAll()
							this.reportError(qerrors.NewSSError(qerrors.E_SS_TIMEOUT))
							return
						}
					}
					if this.limit > 0 && returnLimit == 0 {
						break processing
					}
				}
			}
			// scan vb list to see if we have any vbs that have no results and are not complete and the server isn't busy
			// try to ensure we don't get stuck just fetching one vb
			completed = true
			t := 0
			for s := range queues {
				t += queues[s]
			}
			if t < len(queues)*_SS_MAX_CONCURRENT_VBSCANS_PER_SERVER && len(vbScans) > 0 {
				n := rand.Int() % len(vbScans)
				for c := 0; c < len(vbScans); c++ {
					vbs := vbScans[n]
					n++
					if n == len(vbScans) {
						n = 0
					}
					// skip the one that has just reported to give others a chance
					if !vbs.kvOpsComplete && vbs.state == _VBS_READY && vbs != vbscan {
						completed = false
						if len(vbs.keys) == 0 && queues[vbs.queue] < _SS_MAX_CONCURRENT_VBSCANS_PER_SERVER {
							if !this.queueVBScan(vbs) {
								cancelAll()
								return
							}
							queues[vbs.queue]++
							t++
							if t >= len(queues)*_SS_MAX_CONCURRENT_VBSCANS_PER_SERVER {
								break
							}
						}
					} else if vbs.state == _VBS_ERROR {
						// will only be in error state if aborted during processing (panic)
						cancelAll()
						if vbs.err == nil {
							vbs.err = qerrors.NewSSError(qerrors.E_SS_WORKER_ABORT)
						}
						this.reportError(vbs.err)
						return
					} else if vbs.state != _VBS_READY || len(vbs.keys) > 0 {
						completed = false
					}
				}
				// check the one that has just reported
				if t < len(queues)*_SS_MAX_CONCURRENT_VBSCANS_PER_SERVER && vbscan != nil && !vbscan.kvOpsComplete {
					completed = false
					if len(vbscan.keys) == 0 && queues[vbscan.queue] < _SS_MAX_CONCURRENT_VBSCANS_PER_SERVER {
						if !this.queueVBScan(vbscan) {
							return
						}
						queues[vbscan.queue]++
					}
				}
			}
		}
	}
	if this.timedout {
		this.reportError(qerrors.NewSSError(qerrors.E_SS_TIMEOUT))
	}

	// make sure we don't leave anything lingering
	cancelAll()

	if !this.inactive && !this.timedout {
		// send final end of data indicator
		if !this.reportResults([]string(nil)) {
			this.reportError(qerrors.NewSSError(qerrors.E_SS_TIMEOUT))
		}
	}
}

func (this *seqScan) queueVBScan(vbscan *vbRangeScan) bool {
	vbscan.state = _VBS_SCHEDULED

	if this.timedout {
		this.reportError(qerrors.NewSSError(qerrors.E_SS_TIMEOUT))
		return false
	}
	if this.inactive {
		return false
	}
	err := vbscan.truncate()
	if err != nil {
		this.reportError(err.(qerrors.Error))
		return false
	}
	return _RSW.queueScan(vbscan)
}

/*
 * Individual v-bucket scan.
 */

type vbRsState int32

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
	sampleSize    int
	singleKey     bool
	queue         int
	state         vbRsState
	deferRelease  bool
	kvOpsComplete bool

	continueFrom      []byte
	continueExcluding bool

	uuid       []byte
	spill      *os.File
	buffer     []byte
	offset     uint32
	keys       []uint32
	currentKey int
	reader     *bufio.Reader
	head       []byte

	err qerrors.Error

	retries    int
	delayUntil util.Time
}

func (this *vbRangeScan) String() string {
	return fmt.Sprintf("[%p,%p,%d]", this.scan, this, this.vbucket())
}

func (this *vbRangeScan) startFrom() ([]byte, bool) {
	if this.continueFrom != nil {
		rv := this.continueFrom
		this.continueFrom = nil
		return rv, this.continueExcluding
	}
	return this.scan.getRange(this.rng).start()
}

func (this *vbRangeScan) endWith() ([]byte, bool) {
	return this.scan.getRange(this.rng).end()
}

func (this *vbRangeScan) setContinueFrom(val []byte, excluding bool) {
	this.continueFrom = make([]byte, len(val))
	copy(this.continueFrom, val)
	this.continueExcluding = excluding
}

func (this *vbRangeScan) setContinueFromLastKey() {
	if len(this.keys) < 1 {
		this.continueFrom = nil
		return
	}
	this.seek(len(this.keys) - 1)
	this.setContinueFrom(this.current(), true)
}

func (this *vbRangeScan) release() {
	this.truncate() // release space
	if this.spill != nil {
		this.spill.Close()
	}
	this.spill = nil
	this.reader = nil
	this.keys = nil
	this.currentKey = 0
}

func (this *vbRangeScan) truncate() error {
	var err error
	if this.buffer != nil {
		this.buffer = this.buffer[:0]
	}
	if this.keys != nil {
		this.keys = this.keys[:0]
	}
	if this.spill != nil {
		var size int64
		size, err = this.spill.Seek(0, os.SEEK_END)
		if err == nil && size > 0 {
			_, err = this.spill.Seek(0, os.SEEK_SET)
			if err == nil {
				err = this.spill.Truncate(0)
				if err == nil {
					util.ReleaseTemp(this.spill.Name(), size)
				}
			}
		}
	}
	this.offset = 0
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
	if this.offset >= uint32(cap(this.buffer)) {
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
	var err error
	if this.offset >= uint32(cap(this.buffer)) {
		_, err = io.ReadFull(this.reader, this.head)
	} else {
		copy(this.head, this.buffer[this.keyStart(this.currentKey):])
	}
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

func (this *vbRangeScan) keyStart(n int) int {
	if n < 0 || n >= len(this.keys) {
		return -1
	}
	if n == 0 {
		return 0
	}
	return int(this.keys[n-1])
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
	logging.Debuga(func() string { return fmt.Sprintf("%s sending %d keys", this, len(this.keys)) }, this.scan.log)
	this.scan.readyQueue.enqueue(this)
}

func (this *vbRangeScan) addKey(key []byte) bool {
	var err error
	if bytes.HasPrefix(key, []byte("_txn:")) {
		// exclude transaction binary documents
		return true
	}
	if this.buffer == nil {
		this.buffer = make([]byte, 0, _SS_KEY_BUFFER)
	}
	if this.offset+uint32(len(key)) >= uint32(cap(this.buffer)) && this.spill == nil {
		this.spill, err = util.CreateTemp(_SS_SPILL_FILE_PATTERN, true)
		if err != nil {
			this.reportError(qerrors.NewSSError(qerrors.E_SS_SPILL, err))
			return false
		}
	}
	if this.keys == nil {
		this.keys = make([]uint32, 0, _SS_INIT_KEYS)
	}
	if this.offset < uint32(cap(this.buffer)) && this.offset+uint32(len(key)) >= uint32(cap(this.buffer)) {
		// flush the buffer to spill file
		_, err = this.spill.Write(this.buffer)
		if err != nil {
			this.reportError(qerrors.NewSSError(qerrors.E_SS_SPILL, err))
			return false
		}
		this.buffer = this.buffer[:0]
	}
	this.offset += uint32(len(key))
	if this.offset >= uint32(cap(this.buffer)) {
		if !util.UseTemp(this.spill.Name(), int64(len(key))) {
			this.reportError(qerrors.NewTempFileQuotaExceededError())
			return false
		}
		_, err = this.spill.Write(key)
		if err != nil {
			this.reportError(qerrors.NewSSError(qerrors.E_SS_SPILL, err))
			return false
		}
	} else {
		this.buffer = append(this.buffer, key...)
	}
	if len(this.keys) == cap(this.keys) {
		nw := make([]uint32, len(this.keys), cap(this.keys)*2)
		copy(nw, this.keys)
		this.keys = nw
	}
	this.keys = append(this.keys, this.offset)
	return true
}

func (this *vbRangeScan) setupRetry() {
	this.delayUntil = util.Now().Add(_SS_RETRY_DELAY + (time.Duration(_SS_RETRIES-this.retries) * _SS_RETRY_DELAY))
	this.retries--
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

/*
 * Grouped list container for scans.  Handles actual KV interaction.
 */

func (this *vbRangeScan) validateSingleKey(conn *memcached.Client) bool {
	key, _ := this.startFrom()
	logging.Debuga(func() string { return fmt.Sprintf("%s %v \"%v\"", this, key, string(key)) }, this.scan.log)
	ok, err := conn.ValidateKey(this.vbucket(), string(key), &memcached.ClientContext{CollId: this.scan.collId})
	if err != nil {
		this.reportError(qerrors.NewSSError(qerrors.E_SS_VALIDATE, err))
		return true
	}
	this.kvOpsComplete = true
	if !ok {
		// success but no data
		if this.keys != nil {
			this.keys = this.keys[:0]
		}
		err = this.truncate()
		if err != nil {
			this.reportError(qerrors.NewSSError(qerrors.E_SS_SPILL, err))
		}
	} else {
		if !this.addKey(key) {
			this.reportError(qerrors.NewError(nil, "validateSingleKey: failed to add key to scan results"))
		}
	}
	if this.state != _VBS_CANCELLED {
		this.state = _VBS_WORKED
	}
	this.sendData()
	return true
}

func (this *vbRangeScan) runScan(conn *memcached.Client, node string) bool {
	if this.state != _VBS_WORKING {
		// cancelled whilst connection was being obtained
		return true
	}
	defer func() {
		if this.state == _VBS_WORKING {
			this.state = _VBS_ERROR
		}
		if this.deferRelease {
			this.release()
		}
	}()

	if this.singleKey {
		return this.validateSingleKey(conn)
	}

	var err error
	var response *gomemcached.MCResponse
	var uuid []byte
	fetchLimit := this.scan.fetchLimit
	var start []byte
	var exclStart bool

	opaque := uint32(0)

	cancelScan := func(keepConn bool) bool {
		_RSW.queueCancel(this, uuid)
		return keepConn
	}

	createScan := func() (bool, bool) {
		if this.sampleSize != 0 {
			fetchLimit = uint32(this.sampleSize)
			logging.Debuga(func() string {
				if this.sampleSize == math.MaxInt {
					return fmt.Sprintf("%s Creating random scan to sample all keys", this)
				} else {
					return fmt.Sprintf("%s Creating random scan with sample size %d and limit %d",
						this, this.sampleSize, fetchLimit)
				}
			}, this.scan.log)
			response, err = conn.CreateRandomScan(this.vbucket(), this.scan.collId, this.sampleSize)
		} else {
			start, exclStart = this.startFrom()
			end, exclEnd := this.endWith()
			logging.Debuga(func() string {
				return fmt.Sprintf("%s Creating scan from: %v (excl:%v)", this, start, exclStart)
			}, this.scan.log)
			response, err = conn.CreateRangeScan(this.vbucket(), this.scan.collId, start, exclStart, end, exclEnd)
		}
		if err != nil || len(response.Body) < 16 {
			resp, ok := err.(*gomemcached.MCResponse)
			if ok && resp.Status == gomemcached.KEY_ENOENT {
				// success but no data
				this.kvOpsComplete = true
				if this.keys != nil {
					this.keys = this.keys[:0]
				}
				err = this.truncate()
				if err != nil {
					this.reportError(qerrors.NewSSError(qerrors.E_SS_SPILL, err))
				} else {
					this.state = _VBS_WORKED
					this.sendData()
				}
				return false, true
			} else if ok && resp.Status == gomemcached.WOULD_THROTTLE {
				logging.Debuga(func() string {
					return fmt.Sprintf("%s throttling %v on scan creation", this, this.b.Name)
				}, this.scan.log)
				// scan hasn't started so re-queue it so we can return and handle something else
				Suspend(this.b.Name, getDelay(resp), node)
				if this.sampleSize == 0 {
					this.setContinueFrom(start, exclStart)
				}
				if !_RSW.reQueueScan(this) {
					this.reportError(qerrors.NewSSError(qerrors.E_SS_CREATE, err))
				}
				return false, true
			}
			if err == nil {
				logging.Debuga(func() string {
					return fmt.Sprintf("%s create failed, response: %v", this, response)
				}, this.scan.log)
				this.reportError(qerrors.NewSSError(qerrors.E_SS_CREATE))
			} else {
				logging.Debuga(func() string {
					return fmt.Sprintf("%s create failed, error: %v", this, err)
				}, this.scan.log)
				if this.retries > 0 {
					this.setupRetry()
					logging.Errora(func() string {
						return fmt.Sprintf("Range scan %s creation failed with: %v. Remaining retries: %v",
							this, err, this.retries)
					})
					if !_RSW.reQueueScan(this) {
						this.reportError(qerrors.NewSSError(qerrors.E_SS_CREATE, err))
					}
					return false, false // do not retain the connection on a retry
				}
				this.reportError(qerrors.NewSSError(qerrors.E_SS_CREATE, err))
			}
			return false, ok // only retain the connection if a valid KV error response
		}
		if this.scan.serverless {
			ru, _ := response.ComputeUnits()
			this.scan.addRU(ru)
		}
		opaque = response.Opaque
		copy(uuid, response.Body[0:16])
		if this.state == _VBS_CANCELLED {
			return false, cancelScan(true)
		}
		return true, false
	}

	cancelWorking := func() bool {
		if response.Status == gomemcached.RANGE_SCAN_COMPLETE {
			return true
		} else if response.Status == gomemcached.RANGE_SCAN_MORE {
			_, err := conn.CancelRangeScan(this.vbucket(), uuid, 0)
			if err != nil {
				resp, ok := err.(*gomemcached.MCResponse)
				if ok && resp.Status == gomemcached.KEY_ENOENT {
					err = nil
				}
			}
			return err == nil
		}
		return cancelScan(false)
	}

	retryable := false
	if this.uuid != nil {
		// try continuing with a previous scan
		uuid = this.uuid
		this.uuid = nil
		retryable = true
	} else {
		// create a new scan
		uuid = make([]byte, 16)
		cont, rv := createScan()
		if !cont {
			return rv
		}
	}

	// loop issuing continue scan commands
	for {
		// allow retry with new scan when continue of previously created scan fails
		for {
			logging.Debuga(func() string {
				return fmt.Sprintf("%s continuing %s with scan limit %d", this, uuidAsString(uuid), fetchLimit)
			}, this.scan.log)
			err = conn.ContinueRangeScan(this.vbucket(), uuid, opaque, fetchLimit, 0, 0)
			if resp, ok := err.(*gomemcached.MCResponse); ok {
				if resp.Status == gomemcached.WOULD_THROTTLE {
					logging.Debuga(func() string {
						return fmt.Sprintf("%s throttling %v on continue of %s",
							this, this.b.Name, uuidAsString(uuid))
					}, this.scan.log)
					Suspend(this.b.Name, getDelay(resp), node)
					this.uuid = uuid // scan is open; we'll try to continue with it when re-run
					if this.sampleSize == 0 {
						this.setContinueFrom(start, exclStart) // in case we have to create a new scan
					}
					if !_RSW.reQueueScan(this) {
						this.reportError(qerrors.NewSSError(qerrors.E_SS_CONTINUE, err))
					}
					return true
				} else if resp.Status == gomemcached.KEY_EEXISTS && retryable {
					cont, rv := createScan()
					if !cont {
						return rv
					}
					retryable = false
					continue
				} else if (resp.Status == gomemcached.NOT_MY_VBUCKET || resp.Status == gomemcached.KEY_ENOENT) && this.retries > 0 {
					if this.sampleSize == 0 && len(this.keys) > 0 {
						this.setContinueFromLastKey()
					} else if this.sampleSize == 0 {
						this.setContinueFrom(start, exclStart)
					}
					this.setupRetry()
					logging.Errora(func() string {
						return fmt.Sprintf("Range scan %s %v continue failed with: %v. Remaining retries: %v",
							this, uuidAsString(uuid), resp.Status, this.retries)
					}, this.scan.log)
					if !_RSW.reQueueScan(this) {
						this.reportError(qerrors.NewSSError(qerrors.E_SS_CONTINUE, err))
					}
					return false
				}
			}
			break
		}
		if err != nil {
			logging.Debuga(func() string {
				return fmt.Sprintf("%s %v - continue for %v failed: %v",
					this, this.b.Name, uuidAsString(uuid), err)
			}, this.scan.log)
			this.reportError(qerrors.NewSSError(qerrors.E_SS_CONTINUE, err))
			return cancelScan(false)
		}
		// loop receiving and accumulating results into all the vbRangeScans
		for {
			if this.state == _VBS_CANCELLED {
				return cancelScan(false)
			}

			response, err = conn.ReceiveWithDeadline(time.Now().Add(this.scan.kvTimeout))
			if err != nil {
				resp, ok := err.(*gomemcached.MCResponse)
				if ok && resp.Status != gomemcached.SUCCESS &&
					resp.Status != gomemcached.RANGE_SCAN_MORE &&
					resp.Status != gomemcached.RANGE_SCAN_COMPLETE {

					if resp.Status == gomemcached.WOULD_THROTTLE {
						logging.Debuga(func() string {
							return fmt.Sprintf("%s throttling %v on %v receive after %d keys",
								this, this.b.Name, uuidAsString(uuid), len(this.keys))
						}, this.scan.log)
						Suspend(this.b.Name, getDelay(resp), node)
						if this.sampleSize == 0 && len(this.keys) > 0 {
							this.setContinueFromLastKey()
						} else if this.sampleSize == 0 {
							this.setContinueFrom(start, exclStart)
						}
						this.uuid = uuid // scan is open; we'll try to continue with it when re-run
						if !_RSW.reQueueScan(this) {
							this.reportError(qerrors.NewSSError(qerrors.E_SS_CONTINUE, err))
						}
						return true
					} else if (resp.Status == gomemcached.NOT_MY_VBUCKET || resp.Status == gomemcached.KEY_ENOENT) &&
						this.retries > 0 {

						if this.sampleSize == 0 && len(this.keys) > 0 {
							this.setContinueFromLastKey()
						} else if this.sampleSize == 0 {
							this.setContinueFrom(start, exclStart)
						}
						this.setupRetry()
						logging.Errora(func() string {
							return fmt.Sprintf("Range scan %s %v continue failed with: %v. Remaining retries: %v",
								this, uuidAsString(uuid), resp.Status, this.retries)
						}, this.scan.log)
						if !_RSW.reQueueScan(this) {
							this.reportError(qerrors.NewSSError(qerrors.E_SS_CONTINUE, err))
						}
						return cancelScan(false)
					} else {
						logging.Debuga(func() string {
							return fmt.Sprintf("%s %v receive on %v failed receive after %d keys: %v",
								this, this.b.Name, uuidAsString(uuid), len(this.keys), err)
						}, this.scan.log)
						this.reportError(qerrors.NewSSError(qerrors.E_SS_CONTINUE, err))
						return cancelScan(false)
					}
				}
			}
			if this.scan.serverless {
				ru, _ := response.ComputeUnits()
				this.scan.addRU(ru)
			}

			if this.state == _VBS_CANCELLED {
				return cancelWorking()
			}

			lastStart := 0
			lastEnd := 0
			if len(response.Body) > 0 {
				num_keys := 0
				var l, p uint32
				for i := 0; i < len(response.Body) && this.state == _VBS_WORKING && len(this.keys) < int(fetchLimit); {
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
					lastStart = i
					lastEnd = i + int(l)
					if !this.addKey(response.Body[lastStart:lastEnd]) {
						// addKey will have reported the error already
						return cancelWorking()
					}
					num_keys++
					i += int(l)
				}
				logging.Debuga(func() string {
					return fmt.Sprintf("%s processed %v keys from response of %v bytes",
						this, num_keys, len(response.Body))
				}, this.scan.log)
			}
			if this.state == _VBS_CANCELLED {
				return cancelWorking()
			}

			if response.Status == gomemcached.RANGE_SCAN_MORE &&
				(this.sampleSize != 0 || (int(fetchLimit) > len(this.keys) && len(this.keys) < _SS_MIN_SCAN_SIZE)) {
				break // issue another continue
			}
			if len(response.Body) == 0 ||
				response.Status == gomemcached.RANGE_SCAN_MORE ||
				response.Status == gomemcached.RANGE_SCAN_COMPLETE {

				keepConn := true
				if response.Status != gomemcached.RANGE_SCAN_MORE || this.sampleSize != 0 {
					// end of scan
					this.kvOpsComplete = true
				} else {
					this.setContinueFromLastKey()
					_, err := conn.CancelRangeScan(this.vbucket(), uuid, 0)
					if err != nil {
						resp, ok := err.(*gomemcached.MCResponse)
						if ok && resp.Status == gomemcached.KEY_ENOENT {
							err = nil
						}
					}
					keepConn = (err == nil)
				}
				this.state = _VBS_WORKED
				this.sendData()
				return keepConn
			}
		}
	}
}

/*
 * Range scan workers & queues.
 */

type scanCancel struct {
	vbucket uint16
	uuid    []byte
	b       *Bucket
}

type scanCancelSlice []*scanCancel

func (this scanCancelSlice) Len() int           { return len(this) }
func (this scanCancelSlice) Less(i, j int) bool { return this[i].b.Name < this[i].b.Name }
func (this scanCancelSlice) Swap(i, j int)      { this[i], this[j] = this[j], this[i] }

type rswCancelQueue struct {
	sync.RWMutex
	scans scanCancelSlice
	abort bool
	cond  sync.Cond
	queue *rswQueue
}

func (cqueue *rswCancelQueue) runWorker() {
	var cr scanCancelSlice
	cqueueLocked := false
	cqueueQLocked := false
	defer func() {
		r := recover()
		if r != nil {
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, false)
			s := string(buf[0:n])
			logging.Severef("Range scan cancel worker {%p} panic: %v\n%v", cqueue, r, s)
			// cannot panic and die
			if cqueueLocked {
				cqueue.Unlock()
			}
			if !cqueueQLocked {
				cqueue.queue.Lock()
			}
			if cr != nil && cqueue.queue != nil {
				// clean-up to ensure we don't end up with stuck v-buckets
				for i := range cr {
					if len(cr[i].uuid) > 0 {
						if _, ok := cqueue.queue.active[cr[i].vbucket]; ok {
							cqueue.queue.active[cr[i].vbucket]--
						}
					}
				}
			}
			cqueue.queue.Unlock()
			go cqueue.runWorker()
		}
	}()

	var err error
	var conn *memcached.Client
	var pool *connectionPool
	var b *Bucket
	cqueue.Lock()
	cqueueLocked = true
	for {
		if cqueue.abort {
			cqueue.Unlock()
			cqueueLocked = false
			if conn != nil {
				conn.SetReplica(false)
				pool.Return(conn)
				conn = nil
			}
			// worker pool is shrinking so exit gracefully
			return
		} else if cqueue.scans == nil {
			// relinquish any held connection before waiting for more work
			if conn != nil {
				conn.SetReplica(false)
				pool.Return(conn)
				conn = nil
			}
			cqueue.cond.Wait()
		} else {
			cr := cqueue.scans
			cqueue.scans = nil
			cqueue.Unlock()
			cqueueLocked = false
			if cr != nil {
				sort.Sort(cr)
				var replica bool
				var vbucket uint16
			cancel:
				for i := range cr {
					if conn != nil && (b != cr[i].b || (replica && cr[i].vbucket != vbucket)) {
						conn.SetReplica(false)
						pool.Return(conn)
						conn = nil
					}
					if conn == nil {
						b = cr[i].b
						replica = false
						vbucket = cr[i].vbucket
						desc := &doDescriptor{useReplicas: true, version: b.Version, maxTries: b.backOffRetries(), retry: true}
						for desc.attempts = 0; desc.attempts < desc.maxTries; {
							conn, pool, err = b.getVbConnection(uint32(vbucket), desc)
							if err != nil {
								if desc.retry {
									desc.attempts++
									continue
								}
								conn = nil
							}
							break
						}
						if conn == nil {
							break cancel
						}
						if desc.replica > 0 {
							conn.SetReplica(true)
							replica = true
						}
					}
					// always reset the deadline for each new piece of work
					dl, _ := getDeadline(noDeadline, DefaultTimeout)
					conn.SetDeadline(dl)
					// TODO: batch by bucket, if/when protcol changes to support batch cancellations
					_, err := conn.CancelRangeScan(cr[i].vbucket, cr[i].uuid, 0)
					if err != nil {
						resp, ok := err.(*gomemcached.MCResponse)
						if !ok || resp.Status != gomemcached.KEY_ENOENT {
							logging.Debuga(func() string {
								return fmt.Sprintf("%s: vb %d cancel failed: %v", uuidAsString(cr[i].uuid), cr[i].vbucket, err)
							})
							pool.Discard(conn)
							conn = nil
						}
					}
					cqueue.queue.Lock()
					cqueueQLocked = true
					cqueue.queue.active[cr[i].vbucket]--
					cr[i].uuid = cr[i].uuid[:0]
					cqueueQLocked = false
					cqueue.queue.Unlock()
				}
			}
			cqueue.Lock()
			cqueueLocked = true
		}
	}
}

type rswQueue struct {
	sync.RWMutex
	scans     *list.List
	active    map[uint16]int
	abort     bool
	waitStart util.Time
	cond      sync.Cond
	cqueue    *rswCancelQueue
}

// queue must be locked on entry
func (queue *rswQueue) nextScanLocked() *vbRangeScan {
	for e := queue.scans.Front(); e != nil; e = e.Next() {
		vbscan := e.Value.(*vbRangeScan)
		cnt, found := queue.active[vbscan.vbucket()]
		if !found || cnt <= 0 {
			// If suspended don't schedule the scan. Since we'll spin checking as the list isn't empty, we'll pick up suspended
			// items as soon as the suspension is lifted.
			if !IsSuspended(vbscan.b.Name) && vbscan.delayUntil <= util.Now() {
				queue.scans.Remove(e)
				queue.active[vbscan.vbucket()] = 1
				return vbscan
			}
		}
	}
	return nil
}

func (queue *rswQueue) addScan(vbscan *vbRangeScan) {
	queue.Lock()
	queue.scans.PushBack(vbscan)
	queue.cond.Broadcast()
	queue.Unlock()
}

func (queue *rswQueue) finishScan(vbscan *vbRangeScan) {
	queue.Lock()
	queue.finishScanLocked(vbscan)
	queue.Unlock()
}

func (queue *rswQueue) finishScanLocked(vbscan *vbRangeScan) {
	queue.active[vbscan.vbucket()]--
}

func (queue *rswQueue) cancelSeqScan(ss *seqScan) {
	queue.Lock()
	for e := queue.scans.Front(); e != nil; {
		vbscan := e.Value.(*vbRangeScan)
		en := e.Next()
		if vbscan.scan == ss {
			queue.scans.Remove(e)
		}
		e = en
	}
	queue.Unlock()
}

func (queue *rswQueue) runWorker() {
	var vbscan *vbRangeScan
	queueLocked := false
	defer func() {
		r := recover()
		if r != nil {
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, false)
			s := string(buf[0:n])
			logging.Severef("Range scan worker [%p] panic: %v\n%v", queue, r, s)
			// cannot panic and die
			if queueLocked {
				queue.Unlock()
			}
			if vbscan != nil && vbscan.state == _VBS_WORKING {
				// clean-up to ensure we don't end up with stuck v-buckets
				queue.finishScan(vbscan)
			}
			go queue.runWorker()
		}
	}()

	var err error
	var conn *memcached.Client
	var pool *connectionPool
	var b *Bucket
	var vb uint16
	var replica bool

	queue.Lock()
	queueLocked = true
	for {
		if queue.abort {
			queue.Unlock()
			queueLocked = false
			if conn != nil {
				conn.SetReplica(false)
				pool.Return(conn)
				conn = nil
			}
			// worker pool is shrinking so exit gracefully
			return
		} else if queue.scans.Front() == nil {
			// relinquish any held connection before waiting for more work
			if conn != nil {
				conn.SetReplica(false)
				pool.Return(conn)
				conn = nil
			}
			queue.waitStart = util.Now()
			queue.cond.Wait()
			queue.waitStart = 0
		} else {
			vbscan := queue.nextScanLocked()
			if vbscan != nil && !atomic.CompareAndSwapInt32((*int32)(&vbscan.state), int32(_VBS_SCHEDULED), int32(_VBS_WORKING)) {
				queue.finishScanLocked(vbscan)
				vbscan = nil
				continue
			}
			queue.Unlock()
			queueLocked = false
			if vbscan != nil {
				if conn != nil && (b != vbscan.b || (replica && vbscan.vbucket() != vb)) {
					conn.SetReplica(false)
					pool.Return(conn)
					conn = nil
				}
				run := true
				if conn == nil {
					// the connection here can be reused for any future scans handled by this worker since each scan queue is
					// bound to a single server
					b = vbscan.b
					vb = vbscan.vbucket()
					replica = false
					desc := &doDescriptor{useReplicas: false, version: b.Version, maxTries: b.backOffRetries(), retry: true}
					for desc.attempts = 0; desc.attempts < desc.maxTries; {
						conn, pool, err = b.getVbConnection(uint32(vb), desc)
						if err != nil {
							if desc.retry {
								desc.attempts++
								continue
							}
							if vbscan.retries > 0 {
								vbscan.setupRetry()
							} else {
								vbscan.reportError(qerrors.NewSSError(qerrors.E_SS_CONN, err))
							}
							conn = nil
							run = false
						}
						break
					}
					if conn == nil && run == true {
						vbscan.reportError(qerrors.NewSSError(qerrors.E_SS_CONN, err))
						run = false
					}
					if conn != nil {
						if desc.replica > 0 {
							conn.SetReplica(true)
							replica = true
						}
					}
				}
				if run == true {
					// always reset the deadline for each new piece of work
					dl, _ := getDeadline(noDeadline, _SS_MAX_DURATION)
					conn.SetDeadline(dl)
					if !vbscan.runScan(conn, pool.Node()) {
						pool.Discard(conn)
						conn = nil
					}
				} else if conn == nil && vbscan != nil && vbscan.delayUntil > util.Now() {
					logging.Errora(func() string {
						return fmt.Sprintf("Range scan %s connection failed with: %v. Remaining retries: %v",
							vbscan, err, vbscan.retries)
					}, vbscan.scan.log)
					if !_RSW.reQueueScan(vbscan) {
						vbscan.reportError(qerrors.NewSSError(qerrors.E_SS_CONN, err))
					}
				}
			}
			queue.Lock()
			queueLocked = true
			if vbscan != nil {
				queue.finishScanLocked(vbscan)
			}
		}
	}
}

/*
 * Queues by server, access and controllers.
 */

type rswControl struct {
	sync.RWMutex
	queues []*rswQueue
}

var _RSW = &rswControl{}

func (this *rswControl) reQueueScan(vbscan *vbRangeScan) bool {
	if !atomic.CompareAndSwapInt32((*int32)(&vbscan.state), int32(_VBS_WORKING), int32(_VBS_SCHEDULED)) {
		return false
	}
	return this.queueScan(vbscan)
}

func (this *rswControl) queueScan(vbscan *vbRangeScan) bool {
	this.RLock()
	if vbscan.queue >= len(this.queues) {
		l := len(this.queues)
		this.RUnlock()
		logging.Severef("Sequential scan: Invalid queue %v (# of queues: %v)", vbscan.queue, l)
		return false
	}
	queue := this.queues[vbscan.queue]
	this.RUnlock()
	queue.addScan(vbscan)
	return true
}

func (this *rswControl) cancelQueuedScans(ss *seqScan) {
	this.RLock()
	for _, q := range this.queues {
		q.cancelSeqScan(ss)
	}
	this.RUnlock()
}

func (this *rswControl) queueCancel(vbscan *vbRangeScan, uuid []byte) {
	this.RLock()
	if vbscan.queue >= len(this.queues) {
		l := len(this.queues)
		this.RUnlock()
		logging.Severef("Sequential scan: Invalid queue %v (# of queues: %v) for cancel", vbscan.queue, l)
		return
	}
	queue := this.queues[vbscan.queue]
	this.RUnlock()
	if queue.cqueue == nil {
		return
	}
	queue.cqueue.Lock()
	if queue.cqueue.scans == nil {
		queue.cqueue.scans = make([]*scanCancel, 0, 256)
	} else if len(queue.cqueue.scans) == cap(queue.cqueue.scans) {
		n := make([]*scanCancel, len(queue.cqueue.scans), cap(queue.cqueue.scans)*2)
		copy(n, queue.cqueue.scans)
		queue.cqueue.scans = n
		logging.Debuga(func() string {
			return fmt.Sprintf("[%p] queueCancel: new cap: %v", queue.cqueue, cap(queue.cqueue.scans))
		})
	}
	queue.cqueue.scans = append(queue.cqueue.scans, &scanCancel{vbucket: vbscan.vbucket(), uuid: uuid, b: vbscan.b})
	queue.Lock()
	queue.active[vbscan.vbucket()]++
	queue.Unlock()
	queue.cqueue.cond.Broadcast()
	queue.cqueue.Unlock()
}

func (this *rswControl) initWorkers(servers int) {
	if servers < 1 {
		servers = 1
	}
	this.Lock()
	if this.queues == nil {
		// first time init
		this.queues = make([]*rswQueue, 0, 32)
		go this.monitorWorkers()
	}
	// add workers if necessary
	for len(this.queues) < servers {
		cqueue := &rswCancelQueue{}
		cqueue.cond.L = cqueue
		nq := &rswQueue{scans: list.New(), active: make(map[uint16]int, 1024), cqueue: cqueue}
		nq.cond.L = nq
		cqueue.queue = nq
		this.queues = append(this.queues, nq)
		go cqueue.runWorker()
		for i := 0; i < _SS_MAX_CONCURRENT_VBSCANS_PER_SERVER; i++ {
			go nq.runWorker()
		}
	}
	this.Unlock()
}

func (this *rswControl) monitorWorkers() {
	for {
		time.Sleep(_SS_MONITOR_INTERVAL)
		mark := util.Now()
		this.Lock()
		var n int
		for n = len(this.queues) - 1; n > 0; n-- {
			this.queues[n].Lock()
			if this.queues[n].waitStart == 0 || mark.Sub(this.queues[n].waitStart) < _SS_MAX_WORKER_IDLE {
				this.queues[n].Unlock()
				break
			}
			this.queues[n].cqueue.Lock()
			this.queues[n].cqueue.abort = true
			this.queues[n].cqueue.cond.Broadcast()
			this.queues[n].cqueue.Unlock()
			this.queues[n].cqueue = nil
			this.queues[n].abort = true
			this.queues[n].cond.Broadcast()
			this.queues[n].Unlock()
			this.queues[n] = nil
		}
		if n < len(this.queues)-1 {
			this.queues = this.queues[:n+1]
		}
		this.Unlock()
	}
}

func uuidAsString(uuid []byte) string {
	var sb strings.Builder
	for i, b := range uuid {
		sb.WriteString(fmt.Sprintf("%02x", b))
		if i == 3 || i == 5 || i == 7 || i == 9 {
			sb.WriteRune('-')
		}
	}
	return sb.String()
}
