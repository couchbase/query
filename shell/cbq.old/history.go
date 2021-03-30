//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/sbinet/liner"
)

func LoadHistory(liner *liner.State, dir string) {
	if dir != "" {
		ReadHistoryFromFile(liner, dir+"/.cbq_history")
	}
}

func UpdateHistory(liner *liner.State, dir, line string) {
	liner.AppendHistory(line)
	if dir != "" {
		WriteHistoryToFile(liner, dir+"/.cbq_history")
	}
}

func WriteHistoryToFile(liner *liner.State, path string) {

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return
	}

	defer f.Close()

	writer := bufio.NewWriter(f)
	_, err = liner.WriteHistory(writer)
	if err != nil {
		fmt.Printf("Error updating .cbq_history file: %v\n", err)
	} else {
		writer.Flush()
	}

}

func ReadHistoryFromFile(liner *liner.State, path string) {

	f, err := os.Open(path)
	if err != nil {
		return
	}

	defer f.Close()

	reader := bufio.NewReader(f)
	liner.ReadHistory(reader)
}
