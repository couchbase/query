//  Copyright (c) 2015-2016 Couchbase, Inc.
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
	"io"
	"os"
	"strings"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/shell/go_cbq/command"
	"github.com/sbinet/liner"
)

/* The following values define the query prompt for cbq.
   The expected end of line character is a ;.
*/
const (
	QRY_EOL     = ";"
	QRY_PROMPT1 = "> "
	QRY_PROMPT2 = "   > "
)

var reset = "\x1b[0m"
var fgRed = "\x1b[31m"

var first = false

/* This method is used to handle user interaction with the
   cli. After combining the multi line input, it is sent to
   the execute_inpu method which parses and executes the
   input command. In the event an error is returned from the
   query execution, it is printed in red. The input prompt is
   the name of the executable.
*/
func HandleInteractiveMode(prompt string) {

	/* Find the HOME environment variable. If it isnt set then
	   try USERPROFILE for windows. If neither is found then
	   the cli cant find the history file to read from.
	*/
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = os.Getenv("USERPROFILE")
		if homeDir == "" {
			_, werr := io.WriteString(command.W, "Unable to determine home directory, history file disabled\n")
			if werr != nil {
				s_err := command.HandleError(errors.WRITER_OUTPUT, werr.Error())
				command.PrintError(s_err)
			}
		}
	}

	/* Create a new liner */
	var liner = liner.NewLiner()
	defer liner.Close()

	/* Load history from Home directory
	   TODO : Once Histfile and Histsize are introduced then change this code
	*/
	err_code, err_string := LoadHistory(liner, homeDir)
	if err_code != 0 {
		s_err := command.HandleError(err_code, err_string)
		command.PrintError(s_err)
	}

	go signalCatcher(liner)

	// state for reading a multi-line query
	inputLine := []string{}
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

		/* Check for shell comments : -- and #. Add them to the history
		   but do not send them to be parsed.
		*/
		if strings.HasPrefix(line, "--") || strings.HasPrefix(line, "#") {
			err_code, err_string := UpdateHistory(liner, homeDir, line)
			if err_code != 0 {
				s_err := command.HandleError(err_code, err_string)
				command.PrintError(s_err)
			}

			continue
		}

		// Building query string mode: set prompt, gather current line
		fullPrompt = QRY_PROMPT2
		inputLine = append(inputLine, line)

		/* If the current line ends with a QRY_EOL, join all query lines,
		   trim off trailing QRY_EOL characters, and submit the query string.
		*/
		if strings.HasSuffix(line, QRY_EOL) {
			inputString := strings.Join(inputLine, " ")
			for strings.HasSuffix(inputString, QRY_EOL) {
				inputString = strings.TrimSuffix(inputString, QRY_EOL)
			}
			if inputString != "" {
				err_code, err_string := UpdateHistory(liner, homeDir, inputString+QRY_EOL)
				if err_code != 0 {
					s_err := command.HandleError(err_code, err_string)
					command.PrintError(s_err)
				}
				err_code, err_string = execute_input(inputString, os.Stdout)
				/* Error handling for Shell errors and errors recieved from
				   go_n1ql.
				*/
				if err_code != 0 {
					s_err := command.HandleError(err_code, err_string)
					if err_code == errors.GON1QL_QUERY {
						//Dont print the error code for query errors.
						tmpstr := fmt.Sprintln(fgRed, s_err, reset)
						io.WriteString(command.W, tmpstr+"\n")

					} else {
						command.PrintError(s_err)
					}

					if *errorExitFlag == true {
						if first == false {
							first = true
							_, werr := io.WriteString(command.W, "Exiting on first error encountered\n")
							if werr != nil {
								s_err = command.HandleError(errors.WRITER_OUTPUT, werr.Error())
								command.PrintError(s_err)
							}
							liner.Close()
							os.Clearenv()
							os.Exit(1)
						}
					}
				}

				/* For the \EXIT and \QUIT shell commands we need to
				   make sure that we close the liner and then exit. In
				   the event an error is returned from execute_input after
				   the \EXIT command, then handle the error and exit with
				   exit code 1 (which is for general errors).
				*/
				if EXIT == true {
					command.EXIT = false
					liner.Close()
					if err == nil {
						os.Exit(0)
					} else {
						os.Exit(1)
					}

				}

			}

			// reset state for multi-line query
			inputLine = []string{}
			fullPrompt = prompt + QRY_PROMPT1
		}
	}

}

/* If ^C is pressed then Abort the shell. This is
   provided by the liner package.
*/
func signalCatcher(liner *liner.State) {
	liner.SetCtrlCAborts(false)

}
