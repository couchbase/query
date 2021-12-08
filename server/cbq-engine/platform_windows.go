// Copyright 2013-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
// file, in accordance with the Business Source License, use of this software
// will be governed by the Apache License, Version 2.0, included in the file
// licenses/APL2.txt.

package main

import "syscall"

// Hide console on windows without removing it unlike -H windowsgui.
func HideConsole(hide bool) {
	var k32 = syscall.NewLazyDLL("kernel32.dll")
	var cw = k32.NewProc("GetConsoleWindow")
	var u32 = syscall.NewLazyDLL("user32.dll")
	var sw = u32.NewProc("ShowWindow")
	hwnd, _, _ := cw.Call()
	if hwnd == 0 {
		return
	}
	if hide {
		var SW_HIDE uintptr = 0
		sw.Call(hwnd, SW_HIDE)
	} else {
		var SW_RESTORE uintptr = 9
		sw.Call(hwnd, SW_RESTORE)
	}
}

func setOpenFilesLimit() {
}
