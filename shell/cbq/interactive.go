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
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/couchbaselabs/clog"
	"github.com/sbinet/liner"
)

const (
	QRY_EOL     = ";"
	QRY_PROMPT1 = "> "
	QRY_PROMPT2 = "   > "
)

func HandleInteractiveMode(tiServer, prompt string) {

	// try to find a HOME environment variable
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		// then try USERPROFILE for Windows
		homeDir = os.Getenv("USERPROFILE")
		if homeDir == "" {
			fmt.Printf("Unable to determine home directory, history file disabled\n")
		}
	}

	var liner = liner.NewLiner()
	defer liner.Close()

	LoadHistory(liner, homeDir)

	go signalCatcher(liner)

	// state for reading a multi-line query
	queryLines := []string{}
	fullPrompt := prompt + QRY_PROMPT1
	for {
		line, err := liner.Prompt(fullPrompt)
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Building query string mode: set prompt, gather current line
		fullPrompt = QRY_PROMPT2
		queryLines = append(queryLines, line)

		// If the current line ends with a QRY_EOL, join all query lines,
		// trim off trailing QRY_EOL characters, and submit the query string:
		if strings.HasSuffix(line, QRY_EOL) {
			queryString := strings.Join(queryLines, " ")
			for strings.HasSuffix(queryString, QRY_EOL) {
				queryString = strings.TrimSuffix(queryString, QRY_EOL)
			}
			if queryString != "" {
				UpdateHistory(liner, homeDir, queryString+QRY_EOL)
				err = execute_internal(tiServer, queryString, os.Stdout)
				if err != nil {
					clog.Error(err)
				}
			}
			// reset state for multi-line query
			queryLines = []string{}
			fullPrompt = prompt + QRY_PROMPT1
		}
	}

}

/**
 *  Attempt to clean up after ctrl-C otherwise
 *  terminal is left in bad shape
 */
func signalCatcher(liner *liner.State) {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT)
	<-ch
	liner.Close()
	os.Exit(0)
}
