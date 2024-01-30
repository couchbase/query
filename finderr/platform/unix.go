//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build linux || darwin

package platform

import (
	"os"

	"golang.org/x/term"
)

func InitTerm() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width, _, err = term.GetSize(int(os.Stdin.Fd()))
		if err != nil {
			return 79
		}
	}
	return width - 1
}
