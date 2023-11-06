//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build darwin

package vliner

import (
	"os/exec"
	"syscall"
)

type Termios struct {
	Iflag     uint64
	Oflag     uint64
	Cflag     uint64
	Lflag     uint64
	Cc        [20]uint8
	Pad_cgo_0 [4]byte
	Ispeed    uint64
	Ospeed    uint64
}

const (
	_TCGETS = syscall.TIOCGETA
	_TCSETS = syscall.TIOCSETA
)

func invokeEditor(args []string, attr *syscall.ProcAttr) bool {
	attr.Sys = &syscall.SysProcAttr{Setctty: true, Setsid: true}
	pid, _, err := syscall.StartProcess(args[0], args, attr)
	if nil == err {
		var ws syscall.WaitStatus
		_, err = syscall.Wait4(pid, &ws, 0, nil)
	}
	return nil == err
}

func setupPipe(cmd string) *exec.Cmd {
	return exec.Command("sh", "-c", cmd)
}
