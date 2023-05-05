package system

import (
	"math"
	"sync/atomic"
	"time"

	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
)

//////////////////////////////////////////////////////////////
// Global Variable
//////////////////////////////////////////////////////////////

// memTotal, memFree account for cgroups if they are supported. See systemStats.go.
var cpuPercent uint64  // (holds a float64) [0,GOMAXPROCS]*100% percent CPU this Go runtime is using
var cpuTime uint64     // cpuTime utime+stime
var cpuLastTime uint64 // last collection time
var rss uint64         // size in bytes of the memory-resident portion of this Go runtime
var memTotal uint64    // total memory in bytes available to this Go runtime
var memFree uint64     // free mem in bytes (EXCLUDING inactive OS kernel pages in bare node case)

var _MIN_DURATION = uint64(time.Second.Milliseconds())

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

	pid, total, now, err1 := stats.ProcessCpuStats()
	if err1 != nil {
		return getCpuPercent(), getRSS(), getMemTotal(), GetMemFree(), nil
	}
	lastTotal, lastNow, cpuPercent := getCpuStats()
	dur := now - lastNow
	if dur > _MIN_DURATION {
		cpu = 100 * (float64(total-lastTotal) / float64(dur))
		updateCpuStats(total, now, cpu)
	} else {
		cpu = cpuPercent
	}

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

func updateCpuStats(total, lastTime uint64, cpu float64) {
	atomic.StoreUint64(&cpuTime, total)
	atomic.StoreUint64(&cpuLastTime, lastTime)
	atomic.StoreUint64(&cpuPercent, math.Float64bits(cpu))
}

func getCpuStats() (uint64, uint64, float64) {
	return atomic.LoadUint64(&cpuTime), atomic.LoadUint64(&cpuLastTime), getCpuPercent()
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

var lastFreeRefresh util.Time
var freeRefresher int32

func updateMemFree(mem uint64) {
	atomic.StoreUint64(&memFree, mem)
	lastFreeRefresh = util.Now()
}

func GetMemFree() uint64 {
	return atomic.LoadUint64(&memFree)
}

func GetMemFreePercent() float64 {
	var f uint64
	if util.Now().Sub(lastFreeRefresh) > time.Second && atomic.AddInt32(&freeRefresher, 1) == 1 {
		stats, err := NewSystemStats()
		if err == nil {
			if f, err = stats.SystemFreeMem(); err == nil {
				updateMemFree(f)
			} else {
				f = 0
			}
			stats.Close()
		}
		atomic.StoreInt32(&freeRefresher, 0)
	}
	t := getMemTotal()
	if f == 0 {
		f = GetMemFree()
	}
	if t > 0 {
		return float64(f) / float64(t)
	}
	return 0
}
