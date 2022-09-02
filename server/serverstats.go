//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package server

import (
	"sync/atomic"
	"time"

	json "github.com/couchbase/go_json"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/system"
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
	stats            *system.SystemStats
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

	// open sigar for stats
	collector.stats, err = system.NewSystemStats()
	if err != nil {
		logging.Errorf("StartStatsCollector error : %v", err)
		return err
	}
	for i := 0; i < len(collector.loadFactors); i++ {
		collector.loadFactors[i] = DEF_LOAD_FACTOR
		collector.sumOfLoadFactors += collector.loadFactors[i]
		collector.nLoadFactors += 1
	}

	// skip the first one
	collector.stats.ProcessCpuPercent()
	collector.stats.ProcessRSS()
	collector.stats.GetTotalAndFreeMem(false)
	updateQsLoadFactor(int(collector.sumOfLoadFactors / collector.nLoadFactors))

	// start stats collection
	go collector.runCollectStats()

	return nil
}

//
// Gather Cpu/Memory
//
func (c *statsCollector) runCollectStats() {
	ticker := time.NewTicker(_STATS_INTRVL)
	defer func() {
		ticker.Stop()
		// cannot panic and die
		recover()
		go c.runCollectStats()
	}()

	index := 0
	mstats := make(map[string]interface{}, 20)

	for range ticker.C {
		loadFactor := c.server.loadFactor(true)
		c.sumOfLoadFactors += (loadFactor - c.loadFactors[index])
		c.loadFactors[index] = loadFactor
		updateQsLoadFactor(int(c.sumOfLoadFactors / c.nLoadFactors))
		if (index % _LOG_INTRVL) == 0 {
			_, rss, total, free, err := system.GetSystemStats(c.stats, false, true)
			if err != nil {
				logging.Debugf("statsCollector error : %v", err)
			}
			getQueryEngineStats(c.server, mstats, rss, total, free)
			if buf, e := json.Marshal(mstats); e == nil {
				logging.Infof("Query Engine Stats %v", string(buf))
			}
		}
		index++
		index %= c.nLoadFactors
	}
}

func getQueryEngineStats(server *Server, mstats map[string]interface{}, rss, total, free uint64) {
	mstats["process.rss"] = rss
	mstats["host.memory.total"] = total
	mstats["host.memory.free"] = free
	mstats["loadfactor"] = getQsLoadFactor()
	mstats["load"] = server.Load()
	mstats["process.service.usage"] = server.ServicerUsage()
	mstats["process.percore.cpupercent"] = server.CpuUsage(false)
	mstats["process.memory.usage"] = server.MemoryUsage(false)
	mstats["request.queued.count"] = server.QueuedRequests()

	var mv map[string]interface{}
	v, e := server.AccountingStore().Vitals()
	if e == nil {
		body, e := json.Marshal(v)
		if e == nil {
			json.Unmarshal(body, &mv)
		}
	}
	for n, v := range mv {
		mstats[n] = v
	}
}

func updateQsLoadFactor(loadFactor int) {
	atomic.StoreUint32(&qsLoadFactor, uint32(loadFactor))
}

func getQsLoadFactor() int {
	return int(atomic.LoadUint32(&qsLoadFactor))
}
