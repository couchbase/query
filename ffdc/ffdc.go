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

	go_errors "errors"

	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/encryption"
	"github.com/couchbase/query/encryption/keymgmt"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
)

// First Failure Data Capture (FFDC)

const _OCCURENCE_LIMIT = 30
const FFDC_MIN_INTERVAL = time.Second * 10
const _MAX_CAPTURE_WAIT_TIME = time.Second * 10
const _CPU_PROFILE_TIME = time.Second * 10
const _ENCRYPTION_BUFFER_SIZE = 64 * util.KiB
const _UNSET_KEY_ID = "UNSET"

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
const reencryptPrefix = "reencrypt_"
const reencryptFileNamePrefix = reencryptPrefix + fileNamePrefix
const unencryptedFileExtension = ".gz" // Encrypted files will not have the ".gz" extension

var pidString string
var cbLogDir string

// Initializing a concrete structure because actions can be set in the operations map post-initialization i.e post Init()
// But initialization requires awareness of what actions are sensitive in order to correctly track the scanned FFDC files on disk
var sensitiveActions = map[string]bool{
	Completed: true,
	Active:    true,
	Vitals:    true,
}

type operationConfig struct {
	op        func(io.Writer) error
	sensitive bool
	async     bool
}

// some actions require external dependencies and are therefore set via the Set() function
var operations = map[string]operationConfig{
	Heap: {
		op: func(w io.Writer) error {
			p := pprof.Lookup("heap")
			if p != nil {
				return p.WriteTo(w, 0)
			}
			return nil
		},
	},
	MemStats: {
		op: func(w io.Writer) error {
			var s runtime.MemStats
			runtime.ReadMemStats(&s)
			fmt.Fprintf(w, "Alloc........... %v\n", Human(s.Alloc))
			fmt.Fprintf(w, "TotalAlloc...... %v\n", Human(s.TotalAlloc))
			fmt.Fprintf(w, "Sys............. %v\n", Human(s.Sys))
			fmt.Fprintf(w, "Lookups......... %v\n", s.Lookups)
			fmt.Fprintf(w, "Mallocs......... %v\n", s.Mallocs)
			fmt.Fprintf(w, "Frees........... %v\n", s.Frees)
			fmt.Fprintf(w, "HeapAlloc....... %v\n", Human(s.HeapAlloc))
			fmt.Fprintf(w, "HeapSys......... %v\n", Human(s.HeapSys))
			fmt.Fprintf(w, "HeapIdle........ %v\n", Human(s.HeapIdle))
			fmt.Fprintf(w, "HeapInuse....... %v\n", Human(s.HeapInuse))
			fmt.Fprintf(w, "HeapReleased.... %v\n", Human(s.HeapReleased))
			fmt.Fprintf(w, "HeapObjects..... %v\n", s.HeapObjects)
			fmt.Fprintf(w, "Stack in use.... %v\n", Human(s.StackInuse))
			fmt.Fprintf(w, "Stack sys....... %v\n", Human(s.StackSys))
			fmt.Fprintf(w, "MSpan in use.... %v\n", Human(s.MSpanInuse))
			fmt.Fprintf(w, "MSpan sys....... %v\n", Human(s.MSpanSys))
			fmt.Fprintf(w, "MCache in use... %v\n", Human(s.MCacheInuse))
			fmt.Fprintf(w, "MCache sys...... %v\n", Human(s.MCacheSys))
			fmt.Fprintf(w, "BuckHashSys..... %v\n", Human(s.BuckHashSys))
			fmt.Fprintf(w, "GCSys........... %v\n", Human(s.GCSys))
			fmt.Fprintf(w, "OtherSys........ %v\n", Human(s.OtherSys))
			fmt.Fprintf(w, "NextGC.......... %v\n", s.NextGC)
			fmt.Fprintf(w, "LastGC.......... %v %v\n", s.LastGC, time.Unix(0, int64(s.LastGC)))
			fmt.Fprintf(w, "GCPauses........ [PauseEnd         PauseNs]\n                 ")
			start := int((s.NumGC + 255) % 256)
			if start < 0 {
				start = 255
			}
			c := 0
			for i := start; ; {
				if c > 0 {
					if c == 4 {
						fmt.Fprintf(w, "\n                 ")
						c = 0
					} else {
						fmt.Fprintf(w, " ")
					}
				}
				fmt.Fprintf(w, "[%s %7d]", time.Unix(0, int64(s.PauseEnd[i])).Format("150405.000000000"), s.PauseNs[i])
				c++
				i--
				if i < 0 {
					i = 255
				}
				if i == start {
					break
				}
			}
			fmt.Fprintf(w, "\n")
			fmt.Fprintf(w, "NumGC........... %v\n", s.NumGC)
			fmt.Fprintf(w, "NumForcedGC..... %v\n", s.NumForcedGC)
			fmt.Fprintf(w, "GCCPUFraction... %v\n", s.GCCPUFraction)
			fmt.Fprintf(w, "DebugGC......... %v\n", s.DebugGC)
			return nil
		},
	},
	Stacks: {
		op: func(w io.Writer) error {
			p := pprof.Lookup("goroutine")
			if p != nil {
				return p.WriteTo(w, 2)
			}
			return nil
		},
	},
	Netstat: {
		op: func(w io.Writer) error {
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
	},
	CPU: {
		op: func(w io.Writer) error {
			if err := pprof.StartCPUProfile(w); err != nil {
				return err
			}
			time.Sleep(_CPU_PROFILE_TIME)
			pprof.StopCPUProfile()
			return nil
		},
		async: true,
	},
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

type occurrenceState uint32

const (
	_IDLE occurrenceState = iota
	_CAPTURING
	_CLEANING
	_REENCRYPTING
)

type occurrence struct {
	when time.Time
	ts   string
	id   int64

	// Is an in memory list of Query generated FFDC files on disk associated with this occurrence
	// Will be the source of truth for key management and file operations
	files     []*ffdcFile
	filesLock sync.RWMutex

	state           atomic.Int32
	pendingCaptures atomic.Int32
}

func (this *occurrence) capture(event string, what string) {
	handoffCompletion := false
	defer func() {
		if !handoffCompletion {
			this.completeAction()
		}
	}()

	var opConfig operationConfig
	if op, ok := operations[what]; ok {
		opConfig = op
	} else {
		// If this is the case, something is wrong with the event definition
		logging.Errorf("FFDC: [%#x] Unknown operation: %v", this.id, what)
		return
	}

	var encryptionKey *encryption.EaRKey
	if opConfig.sensitive {
		var encErr errors.Error
		encryptionKey, encErr = ffdcMgr.getActiveKey()
		if encErr != nil {
			logging.Errorf("FFDC: [%#x] Error obtaining active encryption key: %v", this.id, encErr)
			return
		}
	}

	encrypt := encryptionKey != nil && opConfig.sensitive
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

	if !encrypt {
		nameBuilder.WriteString(unencryptedFileExtension)
	}

	name := nameBuilder.String()

	f, err := os.Create(path.Join(GetPath(), name))
	if err == nil {
		ffdcFile := &ffdcFile{
			name:        name,
			sensitive:   opConfig.sensitive,
			targetKeyId: _UNSET_KEY_ID,
		}

		if encryptionKey != nil {
			ffdcFile.currentKeyId = encryptionKey.Id
		} else {
			ffdcFile.currentKeyId = encryption.UNENCRYPTED_KEY_ID
		}

		this.filesLock.Lock()
		this.files = append(this.files, ffdcFile)
		this.filesLock.Unlock()

		if opConfig.async {
			handoffCompletion = true
			go func() {
				defer this.completeAction()
				var err error
				if encrypt {
					err = this.writeEncryptedFFDCFile(f, opConfig, encryptionKey)
				} else {
					err = this.writeUnencryptedFFDCFile(f, opConfig)
				}

				f.Sync()
				f.Close()
				if err != nil {
					logging.Errorf("FFDC: [%#x] Error capturing '%v' to %v: %v", this.id, what, name, err)
				} else {
					msg := fmt.Sprintf("FFDC: [%#x] Captured: %v", this.id, path.Base(name))
					if encrypt {
						msg += fmt.Sprintf(" encrypted with keyId: %v", encryptionKey.Id)
					}
					logging.Infof(msg)
				}
			}()
			logging.Infof("FFDC: [%#x] Started capture of: %v", this.id, path.Base(name))
		} else {
			if encrypt {
				err = this.writeEncryptedFFDCFile(f, opConfig, encryptionKey)
			} else {
				err = this.writeUnencryptedFFDCFile(f, opConfig)
			}

			f.Sync()
			f.Close()
			if err != nil {
				logging.Errorf("FFDC: [%#x] Error capturing '%v' to %v: %v", this.id, what, name, err)
			} else {
				msg := fmt.Sprintf("FFDC: [%#x] Captured: %v", this.id, path.Base(name))
				if encrypt {
					msg += fmt.Sprintf(" encrypted with keyId: %v", encryptionKey.Id)
				}
				logging.Infof(msg)
			}
		}
	} else {
		logging.Errorf("FFDC: [%#x] failed to create '%v' dump file: %v - %v", this.id, what, name, err)
	}
}

func (this *occurrence) cleanup(inaccessibleOnly bool) {
	this.filesLock.Lock()
	defer this.filesLock.Unlock()
	for i := 0; i < len(this.files); {
		name := this.files[i].Name(true)
		if inaccessibleOnly {
			if _, err := os.Stat(path.Join(GetPath(), name)); err != nil {
				logging.Infof("FFDC: [%#x] dump has been removed: %v", this.id, name)
				if i+1 < len(this.files) {
					copy(this.files[i:], this.files[i+1:])
				}
				this.files = this.files[:len(this.files)-1]
			} else {
				i++
			}
		} else {
			logging.Infof("FFDC: [%#x] removing dump: %v", this.id, name)
			err := os.Remove(path.Join(GetPath(), name))
			if err != nil && !go_errors.Is(err, os.ErrNotExist) {
				ffdcMgr.trackOrphanFile(this.files[i])
			}
			i++
		}
	}
	if !inaccessibleOnly {
		this.files = nil
	}
}

func (this *occurrence) completeAction() {
	if this.pendingCaptures.Add(-1) == 0 {
		this.state.Store(int32(_IDLE))
	}
}

type reason struct {
	sync.RWMutex
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
	if len(this.actions) > 0 {
		occ.state.Store(int32(_CAPTURING))
		occ.pendingCaptures.Store(int32(len(this.actions)))
	}
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
func periodicActions(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer func() {
		ticker.Stop()
		// cannot panic and die
		err := recover()
		logging.Debugf("FFDC: Periodic reset routine failed with error: %v. Restarting.", err)
		go periodicActions(interval)
	}()

	for range ticker.C {
		// Reset the count of every type of FFDC event
		for _, r := range reasons {
			r.reset()
		}

		// Clean up orphaned files
		ffdcMgr.cleanupOrphanFiles()
	}

}

func (this *reason) cleanup() {
	for i := 0; i < len(this.occurrences); {

		if !this.occurrences[i].state.CompareAndSwap(int32(_IDLE), int32(_CLEANING)) {
			i++
			continue
		}

		// remove references to inaccessible files
		this.occurrences[i].cleanup(true)

		this.occurrences[i].filesLock.RLock()
		nFiles := len(this.occurrences[i].files)
		this.occurrences[i].filesLock.RUnlock()

		if nFiles == 0 {
			if i+1 < len(this.occurrences) {
				copy(this.occurrences[i:], this.occurrences[i+1:])
			}
			this.occurrences = this.occurrences[:len(this.occurrences)-1]
		} else {
			// Change state for the occurrence only for those occurrences that remain in the list
			this.occurrences[i].state.Store(int32(_IDLE))
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

	if !occ.state.CompareAndSwap(int32(_IDLE), int32(_CLEANING)) {
		return
	}

	copy(this.occurrences[n:], this.occurrences[n+1:])
	this.occurrences = this.occurrences[:len(this.occurrences)-1]
	occ.cleanup(false)
}

func (this *reason) getOccurence(ts string, fileName string) *occurrence {
	if len(this.occurrences) > 0 {
		occ := this.occurrences[len(this.occurrences)-1]
		if occ.ts == ts {
			return occ
		}
	}
	occ := &occurrence{ts: ts}

	// Occurrences in a given reason will always have different timestamps, because we do not allow captures within FFDC_MIN_INTERVAL
	// of the last occurrence
	this.occurrences = append(this.occurrences, occ)
	return occ
}

func (this *reason) processForKeyDrop(keyIdToDrop string, ffdcMgr *ffdcManager) error {
	// Create a snapshot of occurrences
	this.Lock()
	occList := make([]*occurrence, 0, len(this.occurrences))
	for _, occ := range this.occurrences {
		if occ.state.CompareAndSwap(int32(_IDLE), int32(_REENCRYPTING)) {
			occList = append(occList, occ)
		}
	}
	this.Unlock()

	var dropErr error

	// Track error and continue processing other files even if a failure occurs
	// This is to maximize the number of files that get re-encrypted in this attempt.
	for _, occ := range occList {
		activeKey, err := ffdcMgr.getActiveKey()
		if err != nil {
			logging.Errorf("FFDC: [%#x] Failed to get active key during drop key operation: %v", occ.id, err)
			dropErr = err
			occ.state.Store(int32(_IDLE))
			continue
		}

		// Active key drop is not allowed.
		if (activeKey == nil && keyIdToDrop == encryption.UNENCRYPTED_KEY_ID) || (activeKey != nil && keyIdToDrop == activeKey.Id) {
			logging.Errorf("FFDC: [%#x] Attempt to drop active key", occ.id)
			dropErr = fmt.Errorf("Attempt to drop active key")
			occ.state.Store(int32(_IDLE))
			continue
		}

		// Can iterate over the occurrence's files list without a lock as no appends/deletes can occur on the list
		// when it is in _REENCRYPTING state
		for _, ffdc := range occ.files {

			name, sensitive, currentKeyId, _ := ffdc.AllFields(true)
			if !sensitive || currentKeyId != keyIdToDrop {
				continue
			}

			dropErr = ffdc.transformForKeyDrop(keyIdToDrop, activeKey, ffdcMgr)
			if dropErr != nil {
				logging.Errorf("FFDC: [%#x] Failed to transform file %v: %v", occ.id, name, dropErr)
				dropErr = fmt.Errorf("Failed to transform file %v in occurrence %v: %v", name, occ.id, dropErr)
				continue
			}
		}

		occ.state.Store(int32(_IDLE))
	}

	return dropErr
}

// Get the path to the Couchbase log directory.
func GetPath() string {
	return cbLogDir
}

func Init(logDir string) keymgmt.TrackedEncryptor {
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
		var files []*ffdcFile // Should only contain files with prefix "query_ffdc"
		sz := int64(0)
		for {
			ents, err := d.ReadDir(10)
			if err == nil {
				for i := range ents {

					if ents[i].IsDir() {
						continue
					}

					name := ents[i].Name()

					// Delete if file is staging ffdc file. If cannot delete, track as an orphan file
					if strings.HasPrefix(name, reencryptFileNamePrefix) {
						err := os.Remove(path.Join(GetPath(), name))
						if err != nil && !go_errors.Is(err, os.ErrNotExist) {
							ffdcMgr.trackOrphanFileFromName(name)
						}
						continue
					} else if strings.HasPrefix(name, fileNamePrefix) {
						ffdcFile, queryGenerated, err := genFfdcFile(name)
						if err != nil {
							ffdcMgr.trackOrphanFileFromName(name)
							continue
						}
						if !queryGenerated || ffdcFile == nil {
							continue
						}
						files = append(files, ffdcFile)
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
				a := strings.LastIndexByte(files[i].name, '_')
				b := strings.LastIndexByte(files[j].name, '_')
				return strings.TrimSuffix(files[i].name[a:], unencryptedFileExtension) < strings.TrimSuffix(files[j].name[b:], unencryptedFileExtension)
			})
			for i := range files {
				parts := strings.Split(files[i].name[len(fileNamePrefix)+1:], "_")
				// This check is fine as the files list will only contain query_ffdc files
				if len(parts) < 4 {
					continue
				}
				var occ *occurrence
				if reas, ok := reasons[parts[0]]; ok {
					ts := strings.TrimSuffix(parts[len(parts)-1], unencryptedFileExtension)
					occ = reas.getOccurence(ts, files[i].name)
				}
				if occ != nil {
					occ.files = append(occ.files, files[i])
				}
			}
		}
		logging.Infof("FFDC: Found %v existing dump file(s); %v bytes.", len(files), sz)
	}

	go periodicActions(15 * time.Minute)

	return ffdcMgr
}

func Set(what string, action func(io.Writer) error) {
	if !fs.ValidPath(what) {
		panic(fmt.Sprintf("Invalid 'what' (%v)(%v) for FFDC.", what, []byte(what)))
	}

	opConfig := operationConfig{
		op: action,
	}

	if what == CPU {
		opConfig.async = true
	}

	if ok := sensitiveActions[what]; ok {
		opConfig.sensitive = true
	}

	operations[what] = opConfig
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

func (this *occurrence) writeEncryptedFFDCFile(f *os.File, opConfig operationConfig, key *encryption.EaRKey) error {
	ew, err := encryption.NewCBEFWriterSize(f, key, encryption.CBEF_ZLIB, _ENCRYPTION_BUFFER_SIZE)
	if err != nil {
		logging.Errorf("FFDC: [%#x] Error creating encryption writer: %v", this.id, err)
		return err
	}

	err1 := opConfig.op(ew)

	fErr := ew.Flush()
	if fErr != nil {
		logging.Errorf("FFDC: [%#x] Error flushing encryption writer: %v", this.id, fErr)
	}

	cErr := ew.Close()
	if cErr != nil {
		logging.Errorf("FFDC: [%#x] Error closing encryption writer: %v", this.id, cErr)
	}
	return err1
}

func (this *occurrence) writeUnencryptedFFDCFile(f *os.File, opConfig operationConfig) error {
	zip := gzip.NewWriter(f)
	bufWriter := bufio.NewWriter(zip)
	err := opConfig.op(bufWriter)
	bufWriter.Flush()
	zip.Close()
	return err
}
