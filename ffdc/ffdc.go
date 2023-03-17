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
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/couchbase/query/logging"
)

// First Failure Data Capture (FFDC)

const _FFDC_FILE_LIMIT = 24
const _FFDC_MIN_INTERVAL = time.Minute * 10

var _path string
var pidString string
var files []string

var operations = map[string]func(io.Writer) error{
	Heap: func(w io.Writer) error {
		p := pprof.Lookup(Heap)
		if p != nil {
			return p.WriteTo(w, 0)
		}
		return nil
	},
	Stacks: func(w io.Writer) error {
		p := pprof.Lookup(Stacks)
		if p != nil {
			return p.WriteTo(w, 2)
		}
		return nil
	},
}

var whenLast = make(map[string]time.Time)

const (
	Heap      = "heap"
	Stacks    = "goroutine"
	Completed = "completed_requests"
	Active    = "active_requests"
)

const fileNamePrefix = "query_ffdc"
const defaultLogsPath = "/var/lib/couchbase/logs"
const staticConfigFile = "/etc/couchbase/static_config"

// We are not passed the path to the logs so this is a (dirty?) means of obtaining it
func getPath() string {
	if _path == "" {
		installDir := os.Args[0]
		if os.PathSeparator != '/' {
			installDir = strings.ReplaceAll(installDir, string([]byte{os.PathSeparator}), "/")
		}
		installDir = path.Dir(path.Dir(installDir))
		var p string
		f, err := os.Open(installDir + staticConfigFile)
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
			p = installDir + defaultLogsPath
		}
		if _, err := os.Stat(p); err != nil {
			p = os.TempDir()
		}
		_path = p
	}
	return _path
}

func Init() {
	pidString = fmt.Sprintf("%d", os.Getpid())
	capturePath := getPath()
	logging.Infof("FFDC: Capture path: %v", capturePath)

	d, err := os.Open(capturePath)
	if err == nil {
		for {
			ents, err := d.ReadDir(_FFDC_FILE_LIMIT)
			if err == nil {
				for i := range ents {
					if !ents[i].IsDir() && strings.HasPrefix(ents[i].Name(), fileNamePrefix) {
						files = append(files, capturePath+"/"+ents[i].Name())
					}
				}
			}
			if err != nil || len(ents) < _FFDC_FILE_LIMIT {
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
		}
		logging.Infof("FFDC: Found %v existing dump file(s).", len(files))
	}
}

func doCapture(what string, action func(io.Writer) error) {
	runtime.GC()
	name := path.Join(getPath(), strings.Join([]string{fileNamePrefix, what, pidString, time.Now().Format(time.RFC3339Nano)}, "_"))
	f, err := os.Create(name)
	if err == nil {
		files = append(files, name)
		zip := gzip.NewWriter(f)
		err = action(zip)
		zip.Close()
		f.Sync()
		f.Close()
		if err != nil {
			logging.Errorf("FFDC: Error capturing '%v' to %v: %v", what, path.Base(name), err)
		} else {
			logging.Infof("FFDC: Captured: %v", path.Base(name))
		}
	} else {
		logging.Errorf("FFDC: failed to create '%v' dump file: %v", what, err)
	}
}

func Capture(reason string, args ...string) {
	if len(reason) < 1 {
		panic("Invalid FFDC reason")
	}
	key := reason
	if n := strings.IndexByte(reason, ':'); n > 1 {
		key = key[:n]
	}
	if last, ok := whenLast[key]; ok && time.Since(last) < _FFDC_MIN_INTERVAL {
		return
	}
	whenLast[key] = time.Now()
	logging.Warnf("FFDC: %v", reason)
	for _, what := range args {
		if f, ok := operations[what]; ok && f != nil {
			doCapture(what, f)
		} else {
			logging.Errorf("FFDC: Unknown capture type: %v", what)
		}
	}
	for len(files) > _FFDC_FILE_LIMIT {
		logging.Infof("FFDC: removing dump: %v", path.Base(files[0]))
		os.Remove(files[0])
		files = files[1:]
	}
}

func Set(what string, action func(io.Writer) error) {
	if !fs.ValidPath(what) {
		panic(fmt.Sprintf("Invalid 'what' (%v)(%v) for FFDC.", what, []byte(what)))
	}
	operations[what] = action
}
