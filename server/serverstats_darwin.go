//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build darwin

package server

import (
	"fmt"
	"os/exec"
)

func getTotalMemory() int64 {
	p := exec.Command("sysctl", "-n", "hw.memsize")
	o, err := p.StdoutPipe()
	if err != nil {
		return int64(0)
	}
	err = p.Start()
	if err != nil {
		o.Close()
		return int64(0)
	}
	var i, n int
	n, err = fmt.Fscan(o, &i)
	if err != nil || n != 1 {
		return int64(0)
	}
	err = p.Wait()
	if err != nil {
		return int64(0)
	}
	return int64(n)
}
