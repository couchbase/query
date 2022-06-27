//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package server

import (
	"time"

	json "github.com/couchbase/go_json"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/system"
)

const (
	_STATS_INTRVL = 30 * time.Second
	_LOG_INTRVL   = 10
)

//////////////////////////////////////////////////////////////
// Concrete Type/Struct
//////////////////////////////////////////////////////////////

type statsCollector struct {
	server *Server
	stats  *system.SystemStats
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

	// skip the first one
	collector.stats.ProcessCpuPercent()
	collector.stats.ProcessRSS()
	collector.stats.GetTotalAndFreeMem(false)

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

	count := 0
	refresh := true
	mstats := make(map[string]interface{}, 20)
	for range ticker.C {
		_, rss, total, free, err := system.GetSystemStats(c.stats, refresh, count == 0)
		if err != nil {
			logging.Debugf("statsCollector error : %v", err)
		}
		c.server.MemoryStats(refresh)

		if count == 0 {
			getQueryEngineStats(c.server, mstats, rss, total, free)
			if buf, e := json.Marshal(mstats); e == nil {
				logging.Infof("Query Engine Stats %v", string(buf))
			}
		}
		count++
		count %= _LOG_INTRVL
	}
}

func getQueryEngineStats(server *Server, mstats map[string]interface{}, rss, total, free uint64) {
	mstats["process.rss"] = rss
	mstats["host.memory.total"] = total
	mstats["host.memory.free"] = free
	mstats["loadfactor"] = server.LoadFactor()
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
