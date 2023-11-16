//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build windows

package server

import (
	"syscall"
	"unsafe"

	"github.com/couchbase/query/logging"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	winGlobalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
)

type winMemoryStatusEx struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

func getTotalMemory() int64 {
	var ms winMemoryStatusEx

	ms.dwLength = 64
	winGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&ms)))
	logging.Debugf("MemoryStatusEx: [%d,%d,%d,%d,%d,%d,%d,%d,%d]",
		ms.dwLength,
		ms.dwMemoryLoad,
		ms.ullTotalPhys,
		ms.ullAvailPhys,
		ms.ullTotalPageFile,
		ms.ullAvailPageFile,
		ms.ullTotalVirtual,
		ms.ullAvailVirtual,
		ms.ullAvailExtendedVirtual,
	)

	return int64(ms.ullTotalPhys)
}
