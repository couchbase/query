//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package ffdc

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/logging"
)

// First Failure Data Capture (FFDC)

const _OCCURENCE_LIMIT = 30
const FFDC_MIN_INTERVAL = time.Second * 10
const _MAX_CAPTURE_WAIT_TIME = time.Second * 10
const _CPU_PROFILE_TIME = time.Second * 10

const (
	Heap      = "heap"
	MemStats  = "mems"
	Stacks    = "grtn"
	Completed = "creq"
	Active    = "areq"
	Netstat   = "nets"
	Vitals    = "vita"
	CPU       = "prof"
)

const fileNamePrefix = "query_ffdc"
const defaultLogsPath = "var/lib/couchbase/logs"
const staticConfigFile = "etc/couchbase/static_config"

var _path string
var pidString string
var cbLogDir string

// some actions require external dependencies and are therefore set via the Set() function
var operations = map[string]func(io.Writer) error{
	Heap: func(w io.Writer) error {
		p := pprof.Lookup("heap")
		if p != nil {
			return p.WriteTo(w, 0)
		}
		return nil
	},
	MemStats: func(w io.Writer) error {
		bufWriter := bufio.NewWriter(w)
		var s runtime.MemStats
		runtime.ReadMemStats(&s)
		fmt.Fprintf(bufWriter, "Alloc........... %v\n", Human(s.Alloc))
		fmt.Fprintf(bufWriter, "TotalAlloc...... %v\n", Human(s.TotalAlloc))
		fmt.Fprintf(bufWriter, "Sys............. %v\n", Human(s.Sys))
		fmt.Fprintf(bufWriter, "Lookups......... %v\n", s.Lookups)
		fmt.Fprintf(bufWriter, "Mallocs......... %v\n", s.Mallocs)
		fmt.Fprintf(bufWriter, "Frees........... %v\n", s.Frees)
		fmt.Fprintf(bufWriter, "HeapAlloc....... %v\n", Human(s.HeapAlloc))
		fmt.Fprintf(bufWriter, "HeapSys......... %v\n", Human(s.HeapSys))
		fmt.Fprintf(bufWriter, "HeapIdle........ %v\n", Human(s.HeapIdle))
		fmt.Fprintf(bufWriter, "HeapInuse....... %v\n", Human(s.HeapInuse))
		fmt.Fprintf(bufWriter, "HeapReleased.... %v\n", Human(s.HeapReleased))
		fmt.Fprintf(bufWriter, "HeapObjects..... %v\n", s.HeapObjects)
		fmt.Fprintf(bufWriter, "Stack in use.... %v\n", Human(s.StackInuse))
		fmt.Fprintf(bufWriter, "Stack sys....... %v\n", Human(s.StackSys))
		fmt.Fprintf(bufWriter, "MSpan in use.... %v\n", Human(s.MSpanInuse))
		fmt.Fprintf(bufWriter, "MSpan sys....... %v\n", Human(s.MSpanSys))
		fmt.Fprintf(bufWriter, "MCache in use... %v\n", Human(s.MCacheInuse))
		fmt.Fprintf(bufWriter, "MCache sys...... %v\n", Human(s.MCacheSys))
		fmt.Fprintf(bufWriter, "BuckHashSys..... %v\n", Human(s.BuckHashSys))
		fmt.Fprintf(bufWriter, "GCSys........... %v\n", Human(s.GCSys))
		fmt.Fprintf(bufWriter, "OtherSys........ %v\n", Human(s.OtherSys))
		fmt.Fprintf(bufWriter, "NextGC.......... %v\n", s.NextGC)
		fmt.Fprintf(bufWriter, "LastGC.......... %v %v\n", s.LastGC, time.Unix(0, int64(s.LastGC)))
		fmt.Fprintf(bufWriter, "GCPauses........ [PauseEnd         PauseNs]\n                 ")
		start := int((s.NumGC + 255) % 256)
		if start < 0 {
			start = 255
		}
		c := 0
		for i := start; ; {
			if c > 0 {
				if c == 4 {
					fmt.Fprintf(bufWriter, "\n                 ")
					c = 0
				} else {
					fmt.Fprintf(bufWriter, " ")
				}
			}
			fmt.Fprintf(bufWriter, "[%s %7d]", time.Unix(0, int64(s.PauseEnd[i])).Format("150405.000000000"), s.PauseNs[i])
			c++
			i--
			if i < 0 {
				i = 255
			}
			if i == start {
				break
			}
		}
		fmt.Fprintf(bufWriter, "\n")
		fmt.Fprintf(bufWriter, "NumGC........... %v\n", s.NumGC)
		fmt.Fprintf(bufWriter, "NumForcedGC..... %v\n", s.NumForcedGC)
		fmt.Fprintf(bufWriter, "GCCPUFraction... %v\n", s.GCCPUFraction)
		fmt.Fprintf(bufWriter, "DebugGC......... %v\n", s.DebugGC)
		return bufWriter.Flush()
	},
	Stacks: func(w io.Writer) error {
		p := pprof.Lookup("goroutine")
		if p != nil {
			return p.WriteTo(w, 2)
		}
		return nil
	},
	Netstat: func(w io.Writer) error {
		switch runtime.GOOS {
		case "linux":
			if runCommand(w, "netstat", "-atnp") == nil {
				return nil
			}
		case "windows":
			return runCommand(w, "netstat.exe", "-atno")
		}
		return runCommand(w, "netstat", "-an")
	},
}

var asyncOperations = map[string]func(io.Writer) error{
	CPU: func(w io.Writer) error {
		if err := pprof.StartCPUProfile(w); err != nil {
			return err
		}
		time.Sleep(_CPU_PROFILE_TIME)
		pprof.StopCPUProfile()
		return nil
	},
}

func StartFFDC() {
	go periodicReset(15 * time.Minute)
}

func runCommand(w io.Writer, path string, options string) error {
	var cmd *exec.Cmd
	if options != "" {
		cmd = exec.Command(path, options)
	} else {
		cmd = exec.Command(path)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err = cmd.Start(); err != nil {
		return err
	}
	io.Copy(w, stdout)
	if err = cmd.Wait(); err != nil {
		return err
	}
	return nil
}

const (
	GiB = 1 << 30
	MiB = 1 << 20
	KiB = 1 << 10
)

func Human(v uint64) string {
	if v > GiB {
		return fmt.Sprintf("%v (%.3f GiB)", v, float64(v)/float64(GiB))
	} else if v > MiB {
		return fmt.Sprintf("%v (%.3f MiB)", v, float64(v)/float64(MiB))
	} else if v > KiB {
		return fmt.Sprintf("%v (%.3f KiB)", v, float64(v)/float64(KiB))
	} else {
		return fmt.Sprintf("%d", v)
	}
}

type occurrence struct {
	when  time.Time
	ts    string
	id    int64
	files []string
}

func (this *occurrence) capture(event string, what string) {
	var nameBuilder strings.Builder
	nameBuilder.WriteString(fileNamePrefix)
	nameBuilder.WriteByte('_')
	nameBuilder.WriteString(event)
	nameBuilder.WriteByte('_')
	nameBuilder.WriteString(what)
	nameBuilder.WriteByte('_')
	nameBuilder.WriteString(pidString)
	nameBuilder.WriteByte('_')
	nameBuilder.WriteString(this.ts)
	nameBuilder.WriteString(".gz")
	name := nameBuilder.String()

	f, err := os.Create(path.Join(GetPath(), name))
	if err == nil {
		this.files = append(this.files, name)
		zip := gzip.NewWriter(f)
		// Add buffered writer wrapper
		bufWriter := bufio.NewWriter(zip)
		if op, ok := asyncOperations[what]; ok {
			go func() {
				err = op(bufWriter)
				// Flush buffer before closing writers
				bufWriter.Flush()
				zip.Close()
				f.Sync()
				f.Close()
				if err != nil {
					logging.Errorf("FFDC: [%#x] Error capturing '%v' to %v: %v", this.id, what, name, err)
				} else {
					logging.Infof("FFDC: [%#x] Captured: %v", this.id, path.Base(name))
				}
			}()
			logging.Infof("FFDC: [%#x] Started capture of: %v", this.id, path.Base(name))
		} else {
			err = operations[what](bufWriter) // if it panics it is because there is a problem with the event definition
			bufWriter.Flush()                 // Flush buffer before closing
			zip.Close()
			f.Sync()
			f.Close()
			if err != nil {
				logging.Errorf("FFDC: [%#x] Error capturing '%v' to %v: %v", this.id, what, name, err)
			} else {
				logging.Infof("FFDC: [%#x] Captured: %v", this.id, path.Base(name))
			}
		}
	} else {
		logging.Errorf("FFDC: [%#x] failed to create '%v' dump file: %v - %v", this.id, what, name, err)
	}
}

func (this *occurrence) cleanup(inaccessibleOnly bool) {
	for i := 0; i < len(this.files); {
		if inaccessibleOnly {
			if _, err := os.Stat(path.Join(GetPath(), this.files[i])); err != nil {
				logging.Infof("FFDC: [%#x] dump has been removed: %v", this.id, this.files[i])
				if i+1 < len(this.files) {
					copy(this.files[i:], this.files[i+1:])
				}
				this.files = this.files[:len(this.files)-1]
			} else {
				i++
			}
		} else {
			logging.Infof("FFDC: [%#x] removing dump: %v", this.id, this.files[i])
			os.Remove(path.Join(GetPath(), this.files[i]))
			i++
		}
	}
	if !inaccessibleOnly {
		this.files = nil
	}
}

type reason struct {
	sync.Mutex
	count       int64
	event       string
	msg         string
	actions     []string
	occurrences []*occurrence
	totalCount  int64
}

func (this *reason) shouldCapture() *occurrence {
	logging.Debugf("FFDC: [%s] \"%v\".shouldCapture(): count: %v, len(occ): %v", this.event, this.msg, this.count,
		len(this.occurrences))
	if atomic.AddInt64(&this.count, 1) > 2 {
		// don't change count; reset() will reset it
		return nil
	}
	now := time.Now()
	if len(this.occurrences) > 0 {
		if now.Sub(this.occurrences[len(this.occurrences)-1].when) < FFDC_MIN_INTERVAL {
			// Only decrement count if not reset
			if atomic.LoadInt64(&this.count) > 0 {
				atomic.AddInt64(&this.count, -1)
			}
			return nil
		}
	}
	this.totalCount++
	switch this.event {
	case RequestQueueFull:
		accounting.UpdateCounter(accounting.FFDC_RQF)
	case PlusQueueFull:
		accounting.UpdateCounter(accounting.FFDC_PQF)
	case StalledQueue:
		accounting.UpdateCounter(accounting.FFDC_SQP)
	case MemoryThreshold:
		accounting.UpdateCounter(accounting.FFDC_MTE)
	case SigTerm:
		accounting.UpdateCounter(accounting.FFDC_SIG)
	case Shutdown:
		accounting.UpdateCounter(accounting.FFDC_SDN)
	case MemoryRate:
		accounting.UpdateCounter(accounting.FFDC_MRE)
	case Manual:
		accounting.UpdateCounter(accounting.FFDC_MAN)
	case MemoryLimit:
		accounting.UpdateCounter(accounting.FFDC_SML)
	}
	accounting.UpdateCounter(accounting.FFDC_TOTAL)
	this.cleanup()
	occ := &occurrence{when: now, id: now.UnixMilli(), ts: now.Format("2006-01-02-150405.000")}
	this.occurrences = append(this.occurrences, occ)
	return occ
}

func (this *reason) capture(ch chan bool) {
	locked := false
	ret := false
	defer func() {
		e := recover()
		if e != nil {
			logging.Stackf(logging.ERROR, "FFDC: [%s] Panic during capture: %v", this.event, e)
		}
		select {
		case ch <- ret:
		default:
		}
		close(ch)
		if locked {
			this.Unlock()
		}
	}()
	this.Lock()
	locked = true
	occ := this.shouldCapture()
	this.Unlock()
	locked = false
	if occ != nil {
		ret = true
		logging.Warnf("FFDC: [%#x] %s", occ.id, this.msg)
		for i := range this.actions {
			occ.capture(this.event, this.actions[i])
		}
	}
}

func (this *reason) reset() {
	atomic.StoreInt64(&this.count, 0)
}

// Periodically reset the event
func periodicReset(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer func() {
		ticker.Stop()
		// cannot panic and die
		err := recover()
		logging.Debugf("FFDC: Periodic reset routine failed with error: %v. Restarting.", err)
		go periodicReset(interval)
	}()

	for range ticker.C {
		// Reset the count of every type of FFDC event
		for _, r := range reasons {
			r.reset()
		}
	}

}

func (this *reason) cleanup() {
	for i := 0; i < len(this.occurrences); {
		// remove references to inaccessible files
		this.occurrences[i].cleanup(true)
		if len(this.occurrences[i].files) == 0 {
			if i+1 < len(this.occurrences) {
				copy(this.occurrences[i:], this.occurrences[i+1:])
			}
			this.occurrences = this.occurrences[:len(this.occurrences)-1]
		} else {
			i++
		}
	}

	if len(this.occurrences) < _OCCURENCE_LIMIT {
		return
	}
	n := _OCCURENCE_LIMIT / 2
	if time.Now().AddDate(0, -1, 0).After(this.occurrences[0].when) {
		n = 0
	}
	occ := this.occurrences[n]
	copy(this.occurrences[n:], this.occurrences[n+1:])
	this.occurrences = this.occurrences[:len(this.occurrences)-1]
	occ.cleanup(false)
}

func (this *reason) getOccurence(ts string) *occurrence {
	if len(this.occurrences) > 0 {
		occ := this.occurrences[len(this.occurrences)-1]
		if len(occ.files) == 0 || strings.HasSuffix(occ.files[0], ts) {
			return occ
		}
	}
	occ := &occurrence{ts: ts}
	this.occurrences = append(this.occurrences, occ)
	return occ
}

// Get the path to the Couchbase log directory.
func GetPath() string {
	return cbLogDir
}

func Init(logDir string) {
	defer func() {
		e := recover()
		if e != nil {
			logging.Stackf(logging.ERROR, "Panic initialising FFDC: %v", e)
		}
	}()

	cbLogDir = filepath.Clean(logDir)

	// This should not happen. But logging an error in case it does.
	if cbLogDir == "" {
		logging.Errorf("FFDC: No log directory specified. FFDC files have no capture path.")
	}

	pidString = fmt.Sprintf("%08d", os.Getpid())
	capturePath := GetPath()
	logging.Infof("FFDC: Capture path: %v", capturePath)
	d, err := os.Open(capturePath)
	if err == nil {
		var files []string
		sz := int64(0)
		for {
			ents, err := d.ReadDir(10)
			if err == nil {
				for i := range ents {
					if !ents[i].IsDir() && strings.HasPrefix(ents[i].Name(), fileNamePrefix) {
						files = append(files, ents[i].Name())
						if i, err := ents[i].Info(); err == nil {
							sz += i.Size()
						}
					}
				}
			}
			if err != nil || len(ents) < 10 {
				break
			}
		}
		d.Close()
		if len(files) > 0 {
			sort.Slice(files, func(i int, j int) bool {
				a := strings.LastIndexByte(files[i], '_')
				b := strings.LastIndexByte(files[j], '_')
				return files[i][a:] < files[j][b:]
			})
			for i := range files {
				parts := strings.Split(files[i][len(fileNamePrefix)+1:], "_")
				if len(parts) < 4 {
					continue
				}
				var occ *occurrence
				if reas, ok := reasons[parts[0]]; ok {
					occ = reas.getOccurence(parts[len(parts)-1])
				}
				if occ != nil {
					occ.files = append(occ.files, files[i])
				}
			}
		}
		logging.Infof("FFDC: Found %v existing dump file(s); %v bytes.", len(files), sz)
	}
}

func Set(what string, action func(io.Writer) error) {
	if !fs.ValidPath(what) {
		panic(fmt.Sprintf("Invalid 'what' (%v)(%v) for FFDC.", what, []byte(what)))
	}
	operations[what] = action
}

const (
	RequestQueueFull = "RQF"
	PlusQueueFull    = "PQF"
	StalledQueue     = "SQP"
	MemoryThreshold  = "MTE"
	SigTerm          = "SIG"
	Shutdown         = "SDN"
	MemoryRate       = "MRE"
	Manual           = "MAN"
	MemoryLimit      = "SML"
)

var reasons = map[string]*reason{
	RequestQueueFull: &reason{
		event:   RequestQueueFull,
		actions: []string{Vitals, Stacks, Active, Completed},
		msg:     "Request queue full",
	},
	PlusQueueFull: &reason{
		event:   PlusQueueFull,
		actions: []string{Vitals, Stacks, Active, Completed},
		msg:     "Plus queue full",
	},
	StalledQueue: &reason{
		event:   StalledQueue,
		actions: []string{Vitals, Stacks, Active, Completed, Netstat},
		msg:     "Stalled queue processing",
	},
	MemoryThreshold: &reason{
		event:   MemoryThreshold,
		actions: []string{MemStats, Heap, Stacks, Vitals, Active, Completed, Netstat},
		msg:     "Memory threshold exceeded",
	},
	SigTerm: &reason{
		event:   SigTerm,
		actions: []string{MemStats, Heap, Stacks, Active, Completed},
		msg:     "SIGTERM received",
	},
	Shutdown: &reason{
		event:   Shutdown,
		actions: []string{Active},
		msg:     "Graceful shutdown threshold exceeded",
	},
	MemoryRate: &reason{
		event:   MemoryRate,
		actions: []string{MemStats, Heap, Active, Stacks, Vitals},
		msg:     "Memory growth rate threshold exceeded",
	},
	Manual: &reason{
		event:   Manual,
		actions: []string{MemStats, Heap, Active, Completed, Stacks, Vitals, Netstat, CPU},
		msg:     "Manual invocation",
	},
	MemoryLimit: &reason{
		event:   MemoryLimit,
		actions: []string{MemStats, Heap, Active},
		msg:     "Server memory limit",
	},
}

func Capture(event string) bool {
	rv := false
	r, ok := reasons[event]
	if !ok {
		logging.Stackf(logging.ERROR, "FFDC: Invalid event: %s", event)
	} else {
		// expense of creation here is low compared to actually running the FFDC
		done := make(chan bool, 1)
		go r.capture(done)
		select {
		case rv = <-done:
		case <-time.After(_MAX_CAPTURE_WAIT_TIME):
			logging.Warnf("FFDC: Maximum wait time reached for event: %s", event)
		}
	}
	return rv
}

func Reset(event string) {
	r, ok := reasons[event]
	if !ok {
		logging.Stackf(logging.ERROR, "FFDC: Invalid event: %s", event)
	} else {
		r.reset()
	}
}

func Stats(prefix string, res map[string]interface{}, details bool) {
	tot := int64(0)
	for k, v := range reasons {
		tot += v.totalCount
		if details {
			res[prefix+k] = v.totalCount
		}
	}
	res[prefix+"total"] = tot
}
