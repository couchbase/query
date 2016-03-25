//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
