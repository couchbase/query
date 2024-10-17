//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package server

import (
	"bytes"
	"compress/zlib"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/couchbase/cbauth/metakv"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/logging"
	planshape "github.com/couchbase/query/planshape/encode"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

const (
	_AWR_PATH               = "/query/awr/"
	_AWR_MKV_CONFIG         = _AWR_PATH + "config"
	_AWR_MAX_TEXT           = 10240
	_AWR_COMPRESS_THRESHOLD = 256
	_AWR_DEF_NUM_STMTS      = 10000
	_AWR_MIN_INTERVAL       = time.Minute
	_AWR_DEF_INTERVAL       = time.Minute * 10
	_AWR_MSG                = "AWR: "
	_AWR_CPU_MULTIPLIER     = 20
	_AWR_DEF_THRESHOLD      = time.Second
	_AWR_QUIESCENT          = uint32(0)
	_AWR_ACTIVE             = uint32(1)
	_AWR_MAX_PLAN           = 512
	_AWR_READY_WAIT_INTRVL  = time.Second
)

func InitAWR() {
	process := func(val []byte, thisNode string) {
		var m map[string]interface{}
		if err := json.Unmarshal(val, &m); err == nil {
			if n, ok := m["node"]; ok && n != thisNode {
				if err = AwrCB.SetConfig(m, false); err != nil {
					logging.Errorf(_AWR_MSG+"Failed to update configuration: %v", err)
				}
			}
		} else {
			logging.Errorf(_AWR_MSG+"Failed to update configuration: %v", err)
		}
	}
	val, _, err := metakv.Get(_AWR_MKV_CONFIG)
	if err != nil || len(val) == 0 {
		// save initial config if not already there
		err := metakv.Add(_AWR_MKV_CONFIG, AwrCB.distribConfig())
		if err != nil && err != metakv.ErrRevMismatch {
			logging.Errorf(_AWR_MSG+"Unable to initialize common configuration: %v", err)
		}
	} else {
		// make sure we process initial settings immediately
		process(val, "")
	}
	// monitor entry
	go metakv.RunObserveChildren(_AWR_PATH, func(kve metakv.KVEntry) error {
		if kve.Path == _AWR_MKV_CONFIG {
			process(kve.Value, distributed.RemoteAccess().WhoAmI())
		}
		return nil
	}, make(chan struct{}))
}

var AwrCB awrCB

type awrConfig struct {
	location  string
	queueLen  int
	numStmts  int
	threshold time.Duration
	interval  time.Duration
}

func (this *awrConfig) sameAs(other *awrConfig) bool {
	return this.Location() == other.Location() &&
		this.QueueLen() == other.QueueLen() &&
		this.NumStmts() == other.NumStmts() &&
		this.Threshold() == other.Threshold() &&
		this.Interval() == other.Interval()
}

func (this *awrConfig) Location() string {
	return this.location
}

func (this *awrConfig) QueueLen() int {
	if this.queueLen <= 0 {
		return util.NumCPU() * _AWR_CPU_MULTIPLIER
	}
	return this.queueLen
}

func (this *awrConfig) NumStmts() int {
	if this.numStmts <= 0 {
		return _AWR_DEF_NUM_STMTS
	}
	return this.numStmts
}

func (this *awrConfig) Threshold() time.Duration {
	return this.threshold
}

func (this *awrConfig) Interval() time.Duration {
	if this.interval < _AWR_MIN_INTERVAL {
		return _AWR_DEF_INTERVAL
	}
	return this.interval
}

func (this *awrConfig) setLocation(ks string) {
	this.location = ks
}

func (this *awrConfig) setQueueLen(v int) {
	if v <= 0 {
		v = util.NumCPU() * _AWR_CPU_MULTIPLIER
	}
	this.queueLen = v
}

func (this *awrConfig) setNumStmts(v int) {
	this.numStmts = v
}

func (this *awrConfig) setThreshold(v time.Duration) {
	if v < time.Duration(0) {
		v = _AWR_DEF_THRESHOLD
	}
	this.threshold = v
}

func (this *awrConfig) setInterval(v time.Duration) {
	if v <= time.Duration(0) {
		v = _AWR_DEF_INTERVAL
	} else if v < _AWR_MIN_INTERVAL {
		v = _AWR_MIN_INTERVAL
	}
	this.interval = v
}

func (this *awrConfig) getStorageCollection() (datastore.Keyspace, error) {
	if this.location == "" {
		return nil, errors.NewAWRError(errors.E_AWR_SETTING, "", "location")
	}
	parts := algebra.ParsePath(this.location)
	var ks datastore.Keyspace
	var err error
	if parts[0] != "default" && parts[0] != "" {
		return nil, errors.NewAWRError(errors.E_AWR_SETTING, this.location, "location", fmt.Errorf("Invalid namespace."))
	}
	for {
		ks, err = datastore.GetKeyspace(parts...)
		if err == nil {
			return ks, nil
		}
		if e, ok := err.(errors.Error); !ok {
			break
		} else if e.Code() == errors.E_CB_SCOPE_NOT_FOUND {
			// get the system collection for the bucket; once it shows up we know the test for the user's collection is valid
			// (this is typically necessary only at process start-up)
			ds := datastore.GetDatastore()
			_, sce := ds.GetSystemCollection(parts[1])
			if sce == nil {
				break // the error is real
			}
		} else if e.Code() != errors.E_DATASTORE_NOT_SET { // datastore must be available else we retry
			break
		}
		time.Sleep(_AWR_READY_WAIT_INTRVL)
	}
	return nil, err
}

type awrCB struct {
	sync.Mutex

	enabled bool
	state   uint32

	activeConfig awrConfig
	config       awrConfig

	queueFullOmissions     int64
	maxStmtOmissions       int64
	statementDataOmissions int64
	queue                  chan *awrData
	stop                   chan bool
	workerDone             *sync.WaitGroup
	reporterDone           *sync.WaitGroup
	current                map[string]*awrUniqueStmt
	ts                     time.Time
	requests               uint64
	snapshots              uint64
	start                  time.Time
}

func (this *awrCB) Config() map[string]interface{} {
	m := make(map[string]interface{})
	m["enabled"] = this.enabled
	m["location"] = this.config.Location()
	m["queue_len"] = this.config.QueueLen()
	m["num_statements"] = this.config.NumStmts()
	m["threshold"] = util.OutputDuration(this.config.Threshold())
	m["interval"] = util.OutputDuration(this.config.Interval())
	return m
}

func (this *awrCB) Vitals(m map[string]interface{}) {
	if this.enabled {
		v := make(map[string]interface{}, 7)
		if this.state == _AWR_QUIESCENT {
			v["state"] = "quiescent"
		} else {
			v["state"] = "active"
			if this.queueFullOmissions > 0 {
				v["omissions.queue_full"] = this.queueFullOmissions
			}
			if this.maxStmtOmissions > 0 {
				v["omissions.max_statements"] = this.maxStmtOmissions
			}
			if this.statementDataOmissions > 0 {
				v["omissions.statement_data"] = this.statementDataOmissions
			}
			v["requests"] = this.requests
			v["snapshots"] = this.snapshots
			v["start"] = this.start.Format(util.DEFAULT_FORMAT)
		}
		m["awr"] = v
	}
}

func (this *awrCB) setQuiescent(quiescent bool) {
	if quiescent {
		atomic.StoreUint32(&this.state, _AWR_QUIESCENT)
		logging.Infof(_AWR_MSG + "State: quiescent")
	} else {
		atomic.StoreUint32(&this.state, _AWR_ACTIVE)
		logging.Infof(_AWR_MSG + "State: active")
	}
}

func (this *awrCB) isQuiescent() bool {
	return atomic.LoadUint32(&this.state) == _AWR_QUIESCENT
}

func (this *awrCB) SetConfig(i interface{}, distribute bool) errors.Error {

	cfg, ok := i.(map[string]interface{})
	if !ok {
		if s, ok := i.(string); ok {
			s = strings.TrimSpace(s)
			if len(s) > 0 {
				if err := json.Unmarshal([]byte(s), &cfg); err != nil {
					return errors.NewAWRError(errors.E_AWR_CONFIG, err)
				}
			}
		} else {
			return errors.NewAWRError(errors.E_AWR_CONFIG, fmt.Errorf("Invalid type ('%T') for configuration.", i))
		}
	}

	start := this.enabled
	save := this.config
	target := this.config

	if len(cfg) == 0 {
		start = false
		target = awrConfig{}
	} else {
		for k, v := range cfg {
			if va, ok := v.(value.Value); ok {
				v = va.Actual()
			}
			if f, ok := v.(float64); ok && value.IsInt(f) {
				v = int64(f)
			}
			switch k {
			case "node": // allow but ignore
				continue
			case "threshold":
				if _, ok := v.(string); ok {
					if ok, _ := checkDuration(v); !ok {
						return errors.NewAdminSettingTypeError(k, v)
					}
					target.setThreshold(getDuration(v))
				} else if n, ok := v.(int64); ok {
					target.setThreshold(time.Duration(n) * time.Millisecond)
				} else {
					return errors.NewAdminSettingTypeError(k, v)
				}
			case "interval":
				if _, ok := v.(string); ok {
					if ok, _ := checkDuration(v); !ok {
						return errors.NewAdminSettingTypeError(k, v)
					}
					target.setInterval(getDuration(v))
				} else if n, ok := v.(int64); ok {
					target.setInterval(time.Duration(n) * time.Millisecond)
				} else {
					return errors.NewAdminSettingTypeError(k, v)
				}
			case "queue_len":
				if n, ok := v.(int64); ok {
					target.setQueueLen(int(n))
				} else {
					return errors.NewAdminSettingTypeError(k, v)
				}
			case "num_statements":
				if n, ok := v.(int64); ok {
					target.setNumStmts(int(n))
				} else {
					return errors.NewAdminSettingTypeError(k, v)
				}
			case "enabled":
				if enabled, ok := v.(bool); ok {
					start = enabled
				} else {
					return errors.NewAdminSettingTypeError(k, v)
				}
			case "location":
				if ks, ok := v.(string); ok {
					if ks != "" {
						parts := algebra.ParsePath(ks)
						if parts[0] != "default" && parts[0] != "" {
							return errors.NewAWRError(errors.E_AWR_SETTING, ks, "location", fmt.Errorf("Invalid namespace"))
						} else if len(parts) != 2 && len(parts) != 4 {
							return errors.NewAWRError(errors.E_AWR_SETTING, ks, "location",
								fmt.Errorf("Invalid path (must resolve to 2 or 4 parts)"))
						} else if parts[0] == "" {
							parts[0] = "default"
						}
						path := algebra.NewPathFromElements(parts)
						target.setLocation(path.ProtectedString())
					} else {
						target.setLocation("")
					}
				} else {
					return errors.NewAdminSettingTypeError(k, v)
				}
			default:
				return errors.NewAdminUnknownSettingError(k)
			}
		}
	}
	if target.sameAs(&this.config) && this.enabled == start {
		return nil
	}

	this.Lock()
	this.config = target
	this.Unlock()

	var err errors.Error

	if start {
		if err1 := this.Start(); err1 != nil {
			err = errors.NewAWRError(errors.E_AWR_START, err1)
			this.Lock()
			this.config = save
			this.Unlock()
		}
	} else {
		this.Stop()
	}

	if distribute {
		dcfg := this.distribConfig()
		err1 := metakv.Set(_AWR_MKV_CONFIG, dcfg, nil)
		if err1 != nil && err1.Error() == "Not found" {
			err1 = metakv.Add(_AWR_MKV_CONFIG, dcfg)
		}
		if err1 != nil {
			logging.Errorf(_AWR_MSG+"Failed to distribute configuration: %v", err1)
			if err == nil {
				err = errors.NewAWRError(errors.E_AWR_DISTRIB, err1)
			}
		}
	}

	return err
}

func (this *awrCB) distribConfig() []byte {
	m := this.Config()
	m["node"] = distributed.RemoteAccess().WhoAmI()
	b, err := json.Marshal(m)
	if err != nil {
		logging.Errorf(_AWR_MSG+"Failed to marshal configuration: %v", err)
		return nil
	}
	return b
}

func (this *awrCB) Start() error {
	if this.enabled {
		logging.Debugf("Already enabled - restarting")
		this.Stop()
	}
	this.Lock()
	this.state = _AWR_QUIESCENT
	this.activeConfig = this.config
	this.snapshots = 0
	this.requests = 0
	this.queueFullOmissions = 0
	this.maxStmtOmissions = 0
	this.statementDataOmissions = 0
	this.queue = make(chan *awrData, this.activeConfig.QueueLen())
	this.newSet()
	this.stop = make(chan bool, 3)
	this.workerDone = &sync.WaitGroup{}
	this.workerDone.Add(2)
	this.reporterDone = &sync.WaitGroup{}
	this.reporterDone.Add(1)
	go this.worker()
	go this.worker()
	go this.reporter()
	this.start = time.Now()
	this.enabled = true
	this.Unlock()
	logging.Infof(_AWR_MSG+"Started. Config: %v", this.Config())
	return nil
}

func (this *awrCB) Stop() {
	if !this.enabled {
		logging.Debugf("Already disabled")
		return
	}

	this.Lock()
	this.stop <- true
	this.stop <- true
	this.stop <- true
	this.Unlock()

	this.reporterDone.Wait()

	this.Lock()
	close(this.queue)
	this.queue = nil
	this.current = nil
	this.enabled = false
	close(this.stop)
	this.stop = nil
	this.reporterDone = nil
	this.workerDone = nil
	this.state = _AWR_QUIESCENT
	this.Unlock()
	logging.Infof(_AWR_MSG+"Stopped. Omission stats: queue=%v max-statement=%v statement-data=%v", this.queueFullOmissions,
		this.maxStmtOmissions, this.statementDataOmissions)
}

func (this *awrCB) Active() bool {
	return this.enabled && !this.isQuiescent()
}

func (this *awrCB) newSet() (map[string]*awrUniqueStmt, time.Time) {
	old := this.current
	oldTs := this.ts
	this.current = make(map[string]*awrUniqueStmt, this.activeConfig.NumStmts())
	this.ts = time.Now()
	return old, oldTs
}

// This holds a copy of the data for the workload reporting from the BaseRequest
// We can't hold/use the BaseRequest itself since it is pooled and we want to free it and its resources without delay
type awrData struct {
	sqlID       string // MD5 sum of the request text; calculated for this but also included in completed_requests
	text        string // request text (may be compressed)
	qc          string // query context
	plan        []byte // execution plan outline
	totalTime   uint64
	usedMemory  uint64
	cpuTime     uint64
	phaseStats  [execution.PHASES]phaseStat
	resultCount uint64
	resultSize  uint64
	errorCount  uint64
}

func newAwrData(sqlID string, text string, br *BaseRequest, plan []byte) *awrData {
	if len(text) > _AWR_COMPRESS_THRESHOLD {
		if len(text) > _AWR_MAX_TEXT {
			text = text[:_AWR_MAX_TEXT] + "…"
		}
		var b bytes.Buffer
		e := base64.NewEncoder(base64.StdEncoding, &b)
		w := zlib.NewWriter(e)
		w.Write([]byte(text))
		w.Close()
		e.Close()
		text = b.String()
	}
	rv := &awrData{
		sqlID:       sqlID,
		text:        text,
		qc:          br.queryContext,
		plan:        plan,
		totalTime:   uint64(br.totalDuration),
		usedMemory:  uint64(br.usedMemory),
		cpuTime:     uint64(br.cpuTime),
		phaseStats:  br.phaseStats,
		resultCount: uint64(br.resultCount),
		resultSize:  uint64(br.resultSize),
		errorCount:  uint64(br.errorCount),
	}
	return rv
}

func (this *awrCB) recordWorkload(br *BaseRequest) string {

	if !this.enabled || this.isQuiescent() || br.totalDuration < this.activeConfig.Threshold() || br.Sensitive() {
		return ""
	}

	var text string
	if len(br.statement) > 0 {
		text = br.statement
	} else if br.prepared != nil {
		if len(br.prepared.Text()) > 0 {
			text = br.prepared.Text()
		} else if len(br.prepared.Name()) > 0 {
			text = br.prepared.Name()
		} else if len(br.prepared.EncodedPlan()) > 0 {
			text = br.prepared.EncodedPlan()
		} else {
			text = br.prepared.Signature().ToString()
		}
	}

	if len(text) == 0 {
		// no identifiable statement
		atomic.AddInt64(&this.statementDataOmissions, 1)
		return ""
	}

	h := md5.New()
	h.Write([]byte(text))
	if len(br.queryContext) > 0 {
		h.Write([]byte(br.queryContext)) // include the queryContext to differentiate statements
	}
	sqlID := fmt.Sprintf("%x", h.Sum(nil))

	select {
	case this.queue <- newAwrData(sqlID, text, br, planshape.Encode(br.GetTimings(), _AWR_MAX_PLAN)):
		return sqlID
	default:
		atomic.AddInt64(&this.queueFullOmissions, 1)
		return ""
	}
}

type awrStat struct {
	total uint64
	min   uint64
	max   uint64
}

func (this *awrStat) record(val uint64) (bool, bool) {
	min := false
	max := false
	this.total += val
	if this.min == 0 || this.min > val {
		this.min = val
		min = true
	}
	if this.max < val {
		this.max = val
		max = true
	}
	return min, max
}

func (this *awrStat) appendTo(a *[]interface{}) {
	*a = append(*a, this.total)
	*a = append(*a, this.min)
	*a = append(*a, this.max)
}

type awrStatID int

const (
	_STAT_TOT_TIME awrStatID = iota
	_STAT_CPU_TIME
	_STAT_MEM_USED
	_STAT_RES_COUNT
	_STAT_RES_SIZE
	_STAT_ERR_COUNT
	_STAT_RUN_TIME
	_STAT_FETCH_TIME
	_STAT_PRI_SCAN_TIME
	_STAT_SEQ_SCAN_TIME
	_STAT_PRI_SCAN_COUNT
	_STAT_SEQ_SCAN_COUNT
	_STAT_IDX_SCAN_COUNT
	_STAT_FETCH_COUNT
	_STAT_ORDER_COUNT
	_STAT_PRI_SCAN_OPS
	_STAT_SEQ_SCAN_OPS

	_STAT_SIZE_MARKER // "sizer"
)

const _CURR_STATS_VERSION = 1

type awrUniqueStmt struct {
	text    string
	qc      string
	minPlan []byte
	maxPlan []byte
	count   uint64
	elems   [_STAT_SIZE_MARKER]awrStat
}

func (this *awrUniqueStmt) record(elem awrStatID, val uint64) (bool, bool) {
	return this.elems[elem].record(val)
}

func (this *awrUniqueStmt) stats() []interface{} {
	a := make([]interface{}, 0, len(this.elems)*3)
	for i := range this.elems {
		this.elems[i].appendTo(&a)
	}
	return a
}

func (this *awrCB) processData(data *awrData) {
	var stmt *awrUniqueStmt
	var ok bool
	if stmt, ok = this.current[data.sqlID]; !ok {
		if len(this.current) >= this.activeConfig.NumStmts() {
			this.maxStmtOmissions++
			return
		}
		stmt = &awrUniqueStmt{text: data.text, qc: data.qc}
		this.current[data.sqlID] = stmt
	}
	stmt.count++
	stmt.record(_STAT_TOT_TIME, data.totalTime)
	stmt.record(_STAT_CPU_TIME, data.cpuTime)
	stmt.record(_STAT_MEM_USED, data.usedMemory)
	stmt.record(_STAT_RES_COUNT, data.resultCount)
	stmt.record(_STAT_RES_SIZE, data.resultSize)
	stmt.record(_STAT_ERR_COUNT, data.errorCount)
	ismin, ismax := stmt.record(_STAT_RUN_TIME, uint64(data.phaseStats[execution.RUN].duration))
	if ismin {
		stmt.minPlan = data.plan
	}
	if ismax {
		stmt.maxPlan = data.plan
	}
	stmt.record(_STAT_FETCH_TIME, uint64(data.phaseStats[execution.FETCH].duration))
	stmt.record(_STAT_PRI_SCAN_TIME, uint64(data.phaseStats[execution.PRIMARY_SCAN_GSI].duration))
	stmt.record(_STAT_SEQ_SCAN_TIME, uint64(data.phaseStats[execution.PRIMARY_SCAN_SEQ].duration+
		data.phaseStats[execution.INDEX_SCAN_SEQ].duration))
	stmt.record(_STAT_PRI_SCAN_COUNT, uint64(data.phaseStats[execution.PRIMARY_SCAN_GSI].count))
	stmt.record(_STAT_SEQ_SCAN_COUNT, uint64(data.phaseStats[execution.PRIMARY_SCAN_SEQ].count+
		data.phaseStats[execution.INDEX_SCAN_SEQ].count))
	stmt.record(_STAT_IDX_SCAN_COUNT, uint64(data.phaseStats[execution.INDEX_SCAN].count))
	stmt.record(_STAT_FETCH_COUNT, uint64(data.phaseStats[execution.FETCH].count))
	stmt.record(_STAT_ORDER_COUNT, uint64(data.phaseStats[execution.SORT].count))
	stmt.record(_STAT_PRI_SCAN_OPS, uint64(data.phaseStats[execution.PRIMARY_SCAN_GSI].operators))
	stmt.record(_STAT_SEQ_SCAN_OPS, uint64(data.phaseStats[execution.PRIMARY_SCAN_SEQ].operators+
		data.phaseStats[execution.INDEX_SCAN_SEQ].operators))
	atomic.AddUint64(&this.requests, 1)
}

func (this *awrCB) worker() {
	defer func() {
		e := recover()
		if e != nil {
			logging.Stackf(logging.SEVERE, _AWR_MSG+"Panic in worker: %v", e)
		}
		this.workerDone.Done()
	}()
	for {
		select {
		case data := <-this.queue:
			this.Lock()
			this.processData(data)
			this.Unlock()
		case <-this.stop:
			// drain requests but ignore new that arrive whilst this is processing
			n := len(this.queue)
			logging.Debugf("[%p] stopping (n=%v)", this, n)
			this.Lock()
			for i := 0; i < n; i++ {
				select {
				case data := <-this.queue:
					this.processData(data)
				default:
					i = n
				}
			}
			this.Unlock()
			logging.Debugf("[%p] stopped", this)
			return
		}
	}
}

func (this *awrCB) reporter() {
	defer func() {
		e := recover()
		if e != nil {
			logging.Stackf(logging.SEVERE, _AWR_MSG+"Panic in reporter: %v", e)
		}
		this.reporterDone.Done()
	}()

	ticker := time.NewTicker(this.activeConfig.Interval())
	logging.Debugf("[%p] interval %v", this, this.activeConfig.Interval())

	whoAmI := distributed.RemoteAccess().WhoAmI()
	if len(whoAmI) > 0 {
		whoAmI = whoAmI + "::"
	}

	// block until start has completed
	this.Lock()
	this.Unlock()

	// will block for process start-up if necessary
	ks, err := this.activeConfig.getStorageCollection()
	if err != nil {
		logging.Debugf("[%p] %v", this, err)
	}
	this.setQuiescent(ks == nil || err != nil)

	pqf := this.queueFullOmissions
	pms := this.maxStmtOmissions
	psd := this.statementDataOmissions

	stopProcessing := false
	for !stopProcessing {
		select {
		case <-ticker.C:
		case <-this.stop:
			logging.Debugf("[%p] stopping", this)
			stopProcessing = true
			this.workerDone.Wait() // wait for worker
		}

		ks, err := this.activeConfig.getStorageCollection()
		if ks == nil || err != nil {
			if err != nil {
				if !stopProcessing {
					logging.Errorf(_AWR_MSG+"Keyspace '%s' not found: %v", this.activeConfig.Location(), err)
				}
			} else {
				logging.Errorf(_AWR_MSG+"Keyspace '%s' not found.", this.activeConfig.Location())
			}
			if !stopProcessing && !this.isQuiescent() {
				this.setQuiescent(true)
			}
		} else if this.isQuiescent() {
			this.setQuiescent(false)
		} else {
			if pqf != this.queueFullOmissions || pms != this.maxStmtOmissions || psd != this.statementDataOmissions {
				logging.Infof(_AWR_MSG+"Omission stats: queue=%d (%+d) max-statement=%d (%+d) statement-data=%d (%+d)",
					this.queueFullOmissions, this.queueFullOmissions-pqf,
					this.maxStmtOmissions, this.maxStmtOmissions-pms,
					this.statementDataOmissions, this.statementDataOmissions-psd)
				pqf = this.queueFullOmissions
				pms = this.maxStmtOmissions
				psd = this.statementDataOmissions
			}
		}

		this.Lock()
		if len(this.current) == 0 {
			this.Unlock()
			continue
		}
		loc, ts := this.newSet()
		this.Unlock()

		if ks != nil {
			tsStart := value.NewValue(ts.UTC().UnixMilli())
			tsEnd := value.NewValue(this.ts.UTC().UnixMilli())
			keybase := fmt.Sprintf("awrs::%s::%s", ts.UTC().Format(util.DEFAULT_FORMAT), whoAmI)
			pairs := make([]value.Pair, 0, 512)
			var ins int
			tot := uint64(0)
			for k, v := range loc {
				o := make(map[string]interface{})
				o["ver"] = _CURR_STATS_VERSION
				o["from"] = tsStart
				o["to"] = tsEnd
				o["sqlID"] = k
				o["txt"] = v.text
				o["qc"] = v.qc
				o["cnt"] = v.count
				o["pln"] = append(append([]interface{}(nil), v.minPlan), v.maxPlan) // automatically base64 encoded
				o["sts"] = v.stats()
				tot += v.count

				n := len(pairs)
				pairs = pairs[:n+1]
				pairs[n].Name = fmt.Sprintf("%s%05d", keybase, ins)
				pairs[n].Value = value.NewAnnotatedValue(value.NewValue(o))
				ins++
				if len(pairs) == cap(pairs) {
					_, _, errs := ks.Insert(pairs, datastore.NULL_QUERY_CONTEXT, false)
					if errs != nil && len(errs) > 0 {
						logging.Warnf(_AWR_MSG+"Failed to insert statement snapshot: %v", errs)
					}
					pairs = pairs[:0]
				}
			}
			if len(pairs) > 0 {
				_, _, errs := ks.Insert(pairs, datastore.NULL_QUERY_CONTEXT, false)
				if errs != nil && len(errs) > 0 {
					logging.Warnf(_AWR_MSG+"Failed to insert statement snapshot: %v", errs)
				}
			}
			atomic.AddUint64(&this.snapshots, uint64(ins))
			logging.Infof(_AWR_MSG+"Saved %d statement snapshot(s) for %d requests. Key prefix: \"%s\"", ins, tot, keybase)
		} else {
			logging.Infof(_AWR_MSG+"Discarded %v statement snapshot(s).", len(loc))
		}
	}
	logging.Debugf("[%p] complete", this)
}
