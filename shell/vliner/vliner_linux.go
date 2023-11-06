//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build linux

package vliner

import (
	"os/exec"
	"syscall"
)

const (
	_TCGETS = syscall.TCGETS
	_TCSETS = syscall.TCSETS
)

type Termios struct {
	syscall.Termios
}

func invokeEditor(args []string, attr *syscall.ProcAttr) bool {
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
