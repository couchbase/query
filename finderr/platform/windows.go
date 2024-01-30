//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build windows

package platform

import (
	"golang.org/x/sys/windows"
)

func InitTerm() int {
	outputHandle, _ := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
	windows.SetConsoleMode(outputHandle, windows.ENABLE_PROCESSED_OUTPUT|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)

	var csbi windows.ConsoleScreenBufferInfo
	err := windows.GetConsoleScreenBufferInfo(outputHandle, &csbi)
	if err != nil {
		return 79
	}
	if int(csbi.Size.X) > 2 {
		return int(csbi.Size.X - 1)
	}
	return 1
}
