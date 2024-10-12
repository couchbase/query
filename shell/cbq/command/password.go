//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package command

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

func PromptPassword(prompt string) ([]byte, error) {
	s := fmt.Sprintln(prompt)
	_, err := OUTPUT.WriteString(s)
	if err != nil {
		return nil, err
	}
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		os.Stderr.Write([]byte(s))
	}
	return term.ReadPassword(int(os.Stdin.Fd()))
}
