package system

import (
	"math"
	"sync/atomic"

	"github.com/couchbase/query/logging"
)

//////////////////////////////////////////////////////////////
// Global Variable
//////////////////////////////////////////////////////////////

// memTotal, memFree account for cgroups if they are supported. See systemStats.go.
var cpuPercent uint64 // (holds a float64) [0,GOMAXPROCS]*100% percent CPU this Go runtime is using
var rss uint64        // size in bytes of the memory-resident portion of this Go runtime
var memTotal uint64   // total memory in bytes available to this Go runtime
var memFree uint64    // free mem in bytes (EXCLUDING inactive OS kernel pages in bare node case)

func GetSystemStats(stats *SystemStats, refresh, log bool) (cpu float64, rss, total, free uint64, err error) {
	if !refresh {
		return getCpuPercent(), getRSS(), getMemTotal(), GetMemFree(), nil
	}

	if stats == nil {
		// open sigar for stats
		stats, err = NewSystemStats()
		if err != nil {
			return
		}
		defer stats.Close()
	}

	pid, cpu1, err1 := stats.ProcessCpuPercent()
	cpu, err = cpu1, err1
	if err != nil {
		return
	}
	updateCpuPercent(cpu)

	if _, rss, err = stats.ProcessRSS(); err != nil {
		return
	}
	updateRSS(rss)

	var cGroupValues bool
	if total, free, cGroupValues, err = stats.GetTotalAndFreeMem(false); err != nil {
		return
	}
	updateMemTotal(total)
	updateMemFree(free)

	if log {
		s := "system"
		if cGroupValues {
			s = "cGroup"
		}
		logging.Debugf("cpuCollector: cpu percent %.2f, RSS %v for pid %v", cpu, rss, pid)
		logging.Debugf("cpuCollector[%s]: memory free %v, memory total %v", s, free, total)
	}
	return
}

//////////////////////////////////////////////////////////////
// Global Function
//////////////////////////////////////////////////////////////

func updateCpuPercent(cpu float64) {
	atomic.StoreUint64(&cpuPercent, math.Float64bits(cpu))
}

func getCpuPercent() float64 {
	bits := atomic.LoadUint64(&cpuPercent)
	return math.Float64frombits(bits)
}

func updateRSS(mem uint64) {
	atomic.StoreUint64(&rss, mem)
}

func getRSS() uint64 {
	return atomic.LoadUint64(&rss)
}

func updateMemTotal(mem uint64) {
	atomic.StoreUint64(&memTotal, mem)
}

func getMemTotal() uint64 {
	return atomic.LoadUint64(&memTotal)
}

func updateMemFree(mem uint64) {
	atomic.StoreUint64(&memFree, mem)
}

func GetMemFree() uint64 {
	return atomic.LoadUint64(&memFree)
}
