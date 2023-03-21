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
	"path"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/couchbase/query/logging"
)

// First Failure Data Capture (FFDC)

const _FFDC_OCCURENCE_LIMIT = 10
const _FFDC_MIN_INTERVAL = time.Second * 10

const (
	Heap      = "heap"
	Stacks    = "grtn"
	Completed = "creq"
	Active    = "areq"
)

const fileNamePrefix = "query_ffdc"
const defaultLogsPath = "var/lib/couchbase/logs"
const staticConfigFile = "etc/couchbase/static_config"

var _path string
var pidString string

var operations = map[string]func(io.Writer) error{
	Heap: func(w io.Writer) error {
		p := pprof.Lookup("heap")
		if p != nil {
			return p.WriteTo(w, 0)
		}
		return nil
	},
	Stacks: func(w io.Writer) error {
		p := pprof.Lookup("goroutine")
		if p != nil {
			return p.WriteTo(w, 2)
		}
		return nil
	},
}

type occurrence struct {
	when  time.Time
	ts    string
	files []string
}

func (this *occurrence) capture(event string, what string) {
	name := strings.Join([]string{fileNamePrefix, event, what, pidString, this.ts}, "_")
	f, err := os.Create(path.Join(getPath(), name))
	if err == nil {
		this.files = append(this.files, name)
		zip := gzip.NewWriter(f)
		err = operations[what](zip) // if it panics it is because there is a problem with the event definition
		zip.Close()
		f.Sync()
		f.Close()
		if err != nil {
			logging.Errorf("FFDC: Error capturing '%v' to %v: %v", what, name, err)
		} else {
			logging.Infof("FFDC: Captured: %v", path.Base(name))
		}
	} else {
		logging.Errorf("FFDC: failed to create '%v' dump file: %v - %v", what, name, err)
	}
}

type reason struct {
	sync.Mutex
	count       int64
	event       string
	msg         string
	actions     []string
	occurrences []*occurrence
}

func (this *reason) shouldCapture() *occurrence {
	if atomic.AddInt64(&this.count, 1) > 2 {
		return nil
	}
	now := time.Now()
	if len(this.occurrences) > 0 {
		if now.Sub(this.occurrences[len(this.occurrences)-1].when) < _FFDC_MIN_INTERVAL {
			atomic.AddInt64(&this.count, -1)
			return nil
		}
	}
	if len(this.occurrences) >= _FFDC_OCCURENCE_LIMIT {
		this.cleanup()
	}
	occ := &occurrence{when: now, ts: now.Format(time.RFC3339Nano)}
	this.occurrences = append(this.occurrences, occ)
	return occ
}

func (this *reason) capture() {
	this.Lock()
	occ := this.shouldCapture()
	this.Unlock()
	if occ != nil {
		logging.Warnf("FFDC: %s", this.msg)
		for i := range this.actions {
			occ.capture(this.event, this.actions[i])
		}
	}
}

func (this *reason) reset() {
	atomic.StoreInt64(&this.count, 0)
}

func (this *reason) cleanup() {
	// remove references to inaccessible files
	for i := 0; i < len(this.occurrences); {
		for j := 0; j < len(this.occurrences[i].files); {
			if _, err := os.Stat(path.Join(getPath(), this.occurrences[i].files[j])); err != nil {
				logging.Infof("FFDC: dump has been removed: %v", this.occurrences[i].files[j])
				if j+1 < len(this.occurrences[i].files) {
					copy(this.occurrences[i].files[j:], this.occurrences[i].files[j+1:])
				}
				this.occurrences[i].files = this.occurrences[i].files[:len(this.occurrences[i].files)-1]
			} else {
				j++
			}
		}
		if len(this.occurrences[i].files) == 0 {
			if i+1 < len(this.occurrences) {
				copy(this.occurrences[i:], this.occurrences[i+1:])
			}
			this.occurrences = this.occurrences[:len(this.occurrences)-1]
		} else {
			i++
		}
	}
	if len(this.occurrences) < _FFDC_OCCURENCE_LIMIT {
		return
	}
	// drop from the middle
	n := _FFDC_OCCURENCE_LIMIT / 2
	occ := this.occurrences[n]
	copy(this.occurrences[n:], this.occurrences[n+1:])
	this.occurrences = this.occurrences[:len(this.occurrences)-1]
	for i := range occ.files {
		logging.Infof("FFDC: removing dump: %v", occ.files[i])
		os.Remove(path.Join(getPath(), occ.files[i]))
	}
	occ.files = nil
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

// We are not passed the path to the logs so this is a (dirty?) means of obtaining it
func getPath() string {
	if _path == "" {
		installDir := os.Args[0]
		if os.PathSeparator != '/' {
			installDir = strings.ReplaceAll(installDir, string([]byte{os.PathSeparator}), "/")
		}
		installDir = path.Dir(path.Dir(installDir))
		var p string
		f, err := os.Open(path.Join(installDir, staticConfigFile))
		if err == nil {
			s := bufio.NewScanner(f)
			s.Split(func(data []byte, atEOF bool) (int, []byte, error) {
				var err error
				var extra, s, i int
				if atEOF {
					err = bufio.ErrFinalToken
				}
				for s < len(data) && !(data[s] == '"' || data[s] == '_' || unicode.In(rune(data[s]), unicode.L, unicode.N)) {
					s++
				}
				if s == len(data) {
					return s, nil, err
				}
				if data[s] == '"' {
					s++
					for i = s; i < len(data) && data[i] != '"'; i++ {
					}
					extra = 1
				} else {
					for i = s + 1; i < len(data); i++ {
						if !unicode.In(rune(data[i]), unicode.L, unicode.N) && data[i] != '_' {
							break
						}
					}
				}
				return i + extra, data[s:i], err
			})
			for s.Scan() {
				if s.Text() == "error_logger_mf_dir" && s.Scan() {
					p = s.Text()
					break
				}
			}
			f.Close()
		}
		if p == "" {
			p = path.Join(installDir, defaultLogsPath)
		}
		if _, err := os.Stat(p); err != nil {
			p = os.TempDir()
		}
		_path = p
	}
	return _path
}

func Init() {
	defer func() {
		e := recover()
		if e != nil {
			logging.Stackf(logging.ERROR, "Panic initialising FFDC: %v", e)
		}
	}()
	pidString = fmt.Sprintf("%08d", os.Getpid())
	capturePath := getPath()
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
)

var reasons = map[string]*reason{
	RequestQueueFull: &reason{
		event:   RequestQueueFull,
		actions: []string{Stacks, Active, Completed},
		msg:     "Request queue full",
	},
	PlusQueueFull: &reason{
		event:   PlusQueueFull,
		actions: []string{Stacks, Active, Completed},
		msg:     "Plus queue full",
	},
	StalledQueue: &reason{
		event:   StalledQueue,
		actions: []string{Stacks, Active, Completed},
		msg:     "Stalled queue processing",
	},
	MemoryThreshold: &reason{
		event:   MemoryThreshold,
		actions: []string{Heap, Stacks, Active, Completed},
		msg:     "Memory threshold exceeded",
	},
	SigTerm: &reason{
		event:   SigTerm,
		actions: []string{Heap, Stacks, Active, Completed},
		msg:     "SIGTERM received",
	},
}

func Capture(event string) {
	r, ok := reasons[event]
	if !ok {
		panic("FFDC: Invalid event")
	}
	r.capture()
}

func Reset(event string) {
	r, ok := reasons[event]
	if !ok {
		panic("FFDC: Invalid event")
	}
	r.reset()
}
