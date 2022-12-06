//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package server

import (
	"runtime"
	"sync/atomic"
	"time"

	json "github.com/couchbase/go_json"
	"github.com/couchbase/query/logging"
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
		recover()
		go c.runCollectStats()
	}()

	index := 0

	oldStats := make(map[string]interface{}, 6)
	newStats := make(map[string]interface{}, 6)
	c.server.AccountingStore().ExternalVitals(oldStats)
	for range ticker.C {
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
		newStats["allocated_values"] = value.AllocatedValuesCount()
		oldStats = c.server.AccountingStore().ExternalVitals(newStats)
		newStats = oldStats

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
	}
}

func updateQsLoadFactor(loadFactor int) {
	atomic.StoreUint32(&qsLoadFactor, uint32(loadFactor))
}

func getQsLoadFactor() int {
	return int(atomic.LoadUint32(&qsLoadFactor))
}
