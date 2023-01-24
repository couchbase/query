//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build !solaris
// +build !solaris

package main

import (
	"fmt"
	"io"

	"github.com/couchbase/query/shell/cbq/command"
	"golang.org/x/crypto/ssh/terminal"
)

func promptPassword(prompt string) ([]byte, error) {
	s := fmt.Sprintln(prompt)
	_, err := io.WriteString(command.W, s)
	if err != nil {
		return nil, err
	}
	return terminal.ReadPassword(0)
}
