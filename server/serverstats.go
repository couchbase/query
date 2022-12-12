//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package server

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"sync/atomic"
	"time"

	json "github.com/couchbase/go_json"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/memory"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

const (
	_STATS_INTRVL   = 30 * time.Second // load factor interval
	_LOG_INTRVL     = 10               // log interval 5min
	_MOVING_WINDOW  = 30               // 15min, load factor moving average of 15min i.e 30 values
	DEF_LOAD_FACTOR = 35               // default load factor above 30% so that at start no nodes will be added
)

var qsLoadFactor uint32 // Query Service moving average Load Factor

//////////////////////////////////////////////////////////////
// Concrete Type/Struct
//////////////////////////////////////////////////////////////

type statsCollector struct {
	server           *Server
	loadFactors      [_MOVING_WINDOW]int
	sumOfLoadFactors int
	nLoadFactors     int
}

//////////////////////////////////////////////////////////////
// Cpu/Memory Collector
//////////////////////////////////////////////////////////////

//
// Start Stats collection
//
func (this *Server) StartStatsCollector() (err error) {

	collector := &statsCollector{server: this}

	for i := 0; i < len(collector.loadFactors); i++ {
		collector.loadFactors[i] = DEF_LOAD_FACTOR
		collector.sumOfLoadFactors += collector.loadFactors[i]
		collector.nLoadFactors += 1
	}

	updateQsLoadFactor(int(collector.sumOfLoadFactors / collector.nLoadFactors))

	// start stats collection
	go collector.runCollectStats()

	return nil
}

//
// Gather Cpu/Memory
//
func (c *statsCollector) runCollectStats() {
	var lastGC uint64
	ticker := time.NewTicker(_STATS_INTRVL)
	defer func() {
		ticker.Stop()
		// cannot panic and die
		e := recover()
		logging.Debugf("System stats collector failed with: %v.  Restarting.", e)
		go c.runCollectStats()
	}()

	index := 0

	lastDumpTime := util.Time(0) // temporary addition

	oldStats := make(map[string]interface{}, 6)
	newStats := make(map[string]interface{}, 6)
	c.server.AccountingStore().ExternalVitals(oldStats)
	tickerFunc := func() {
		loadFactor := c.server.loadFactor(true)
		c.sumOfLoadFactors += (loadFactor - c.loadFactors[index])
		c.loadFactors[index] = loadFactor
		updateQsLoadFactor(int(c.sumOfLoadFactors / c.nLoadFactors))

		newStats["loadfactor"] = getQsLoadFactor()
		newStats["load"] = c.server.Load()
		newStats["process.service.usage"] = c.server.ServicerUsage()
		newStats["process.percore.cpupercent"] = c.server.CpuUsage(false)
		newStats["process.memory.usage"], lastGC = c.server.MemoryUsage(false)
		newStats["request.queued.count"] = c.server.QueuedRequests()
		newStats["node.allocated.values"] = value.AllocatedValuesCount()
		m := memory.AllocatedMemory()
		if m > 0 {
			newStats["node.memory.usage"] = m
		} else {
			delete(newStats, "node.memory.usage")
		}
		if tenant.IsServerless() {
			tenants := make(map[string]interface{})
			tenant.Foreach(func(n string, m memory.MemoryManager) {
				tenants[n] = m.AllocatedMemory()
			})
			newStats["tenant.memory.usage"] = tenants
		}
		oldStats = c.server.AccountingStore().ExternalVitals(newStats)
		newStats = oldStats

		// Start: temporary addition hence literal constants
		if newStats != nil {
			if pmu, ok := newStats["process.memory.usage"]; ok {
				if mu, ok := pmu.(uint64); ok && mu >= 80 {
					if util.Since(lastDumpTime) > time.Minute*10 {
						dumpHeap()
						lastDumpTime = util.Now()
					}
				}
			}
		}
		// End: temporary addition

		if (index % _LOG_INTRVL) == 0 {
			mstats, _ := c.server.AccountingStore().Vitals()
			if buf, e := json.Marshal(mstats); e == nil {
				logging.Infof("Query Engine Stats %v", string(buf))
			}
		}
		index++
		index %= c.nLoadFactors

		util.ResyncTime()
		if util.Now().UnixNano()-int64(lastGC) > int64(_STATS_INTRVL) {
			logging.Debugf("Running GC")
			runtime.GC()
		}

		// TODO expire tenants if required
		if false && tenant.IsServerless() {
			tenant.Foreach(func(n string, m memory.MemoryManager) {
				m.Expire()
			})
		}
	}

	tickerFunc()
	index--
	for range ticker.C {
		tickerFunc()
	}
}

func updateQsLoadFactor(loadFactor int) {
	atomic.StoreUint32(&qsLoadFactor, uint32(loadFactor))
}

func getQsLoadFactor() int {
	return int(atomic.LoadUint32(&qsLoadFactor))
}

// start: temporary addition
func dumpHeap() {
	ts := time.Now().Format(time.RFC3339Nano)
	name := fmt.Sprintf("%s/ffdcheap_%v_%v", os.TempDir(), os.Getpid(), ts)
	logging.Infof("FFDC: threshold exceeded, attempting heap dump to: %v", name)
	runtime.GC()
	f, err := os.Create(name)
	if err == nil {
		pprof.WriteHeapProfile(f)
		f.Sync()
		f.Close()
		logging.Infof("FFDC: heap dumped")
	} else {
		logging.Infof("FFDC: failed to create heap output file: %v", err)
	}
}

// end temporary addition
