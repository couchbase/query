//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build linux

package server

import (
	"fmt"
	"os"

	"github.com/couchbase/query/util"
)

func getTotalMemory() int64 {
	rv := int64(0)
	file, err := os.Open("/proc/meminfo")
	if err == nil {
		defer file.Close()
		var w, u string
		var v int
		for {
			n, err := fmt.Fscanf(file, "%s %d %s", &w, &v, &u)
			if n != 3 || err != nil {
				return rv
			}
			if w == "MemTotal:" {
				rv = int64(v)
				switch u {
				case "kB":
					rv *= util.KiB
				case "mB":
					rv *= util.MiB
				}
				return rv
			}
		}
	}
	return int64(0)
}
