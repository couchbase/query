//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package main

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/couchbase/godbc/n1ql"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/shell/cbq/command"
	"github.com/couchbase/query/shell/liner"
	"github.com/couchbase/query/shell/vliner"
)

const (
	_TXTIMEOUT = "2m"
)

/*
The following values define the query prompt for cbq.

	The expected end of line character is a ;.
*/
const (
	QRY_EOL     = ";"
	QRY_PROMPT1 = "> "
	QRY_PROMPT2 = "   > "
)

var first = false

var homeDir string

// Handle output flag

func handleOPModeFlag(outputFile **os.File, prevFile *string) {
	// If an output flag is defined
	var err error

	if outputFlag != "" {
		*prevFile = command.FILE_OUTPUT

		*outputFile, err = os.OpenFile(command.FILE_OUTPUT, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		command.SetDispVal("", "")
		if err != nil {
			s_err := command.HandleError(errors.E_SHELL_OPEN_FILE, err.Error())
			command.PrintError(s_err)
		}

		command.SetWriter(io.Writer(*outputFile))
	}
}

// Handle input flag
/*
   [MB-56912] : Handle advise flag
   			  if inputFlag is set: inputFlag is input file
              if input flag is not set: consider stdin as input file
*/
func handleIPModeFlag(liner **liner.State) {
	if inputFlag != "" || (inputFlag == "" && adviseFlag) {
		//Read each line from the file and call execute query
		var input_command string

		if inputFlag == "" && adviseFlag {
			input_command = "\\source stdin"
		} else {
			input_command = "\\source " + inputFlag
		}

		// If outputting to a file, then add the statement to the file as well.
		if command.FILE_RW_MODE == true {
			_, werr := io.WriteString(command.W, input_command+"\n")
			if werr != nil {
				s_err := command.HandleError(errors.E_SHELL_WRITER_OUTPUT, werr.Error())
				command.PrintError(s_err)
			}
		}
		errCode, errStr := dispatch_command(input_command, command.W, false, *liner)
		// If the previous run didnt error out and we are in batch mode, execute the statements.
		if errCode == 0 {
			if command.BATCH == "on" && !batch_run {
				errCode, errStr = dispatch_command("\\", command.W, false, *liner)
			}
		}

		if errCode != 0 {
			s_err := command.HandleError(errCode, errStr)
			command.PrintError(s_err)
			(*liner).Close()
			os.Clearenv()
			os.Exit(1)
		}

		(*liner).Close()
		os.Clearenv()
		os.Exit(0)
	}
}

// Handle script flag - single command mode

func handleScriptFlag(liner **liner.State) {
	// Handle the file input and script options here so as to add
	// the commands to the history.
	if len(scriptFlag) != 0 {
		//Execute the input command

		// Run all the commands
		for i := 0; i < len(scriptFlag); i++ {
			// If outputting to a file, then add the statement to the file as well.
			if command.FILE_RW_MODE == true {
				_, werr := io.WriteString(command.W, scriptFlag[i]+"\n")
				if werr != nil {
					s_err := command.HandleError(errors.E_SHELL_WRITER_OUTPUT, werr.Error())
					command.PrintError(s_err)
				}
			}

			if !command.QUIET {
				_, werr := io.WriteString(command.W, "\n "+scriptFlag[i]+"\n")
				if werr != nil {
					s_err := command.HandleError(errors.E_SHELL_WRITER_OUTPUT, werr.Error())
					command.PrintError(s_err)
				}
			}

			err_code, err_str := dispatch_command(scriptFlag[i], command.W, false, *liner)

			if err_code != 0 {
				s_err := command.HandleError(err_code, err_str)
				command.PrintError(s_err)

				if *errorExitFlag {
					_, werr := io.WriteString(command.W, command.EXITONERR)
					if werr != nil {
						s_err = command.HandleError(errors.E_SHELL_WRITER_OUTPUT, werr.Error())
						command.PrintError(s_err)
					}
					(*liner).Close()
					os.Clearenv()
					os.Exit(1)
				}
			}
		}

		(*liner).Close()
		os.Clearenv()
		os.Exit(0)
	}
}

/*
This method is used to handle user interaction with the

	cli. After combining the multi line input, it is sent to
	the execute_inpu method which parses and executes the
	input command. In the event an error is returned from the
	query execution, it is printed in red. The input prompt is
	the name of the executable.
*/
func HandleInteractiveMode(prompt string) {

	// Variables used for output to file

	outputFile := os.Stdout
	prevFile := ""
	prevreset := command.Getreset()
	prevfgRed := command.GetfgRed()

	handleOPModeFlag(&outputFile, &prevFile)
	defer outputFile.Close()

	// Find the HOME environment variable using GetHome
	var err_code = errors.E_OK
	var err_str = ""
	homeDir, err_code, err_str = command.GetHome()
	if err_code != 0 {
		s_err := command.HandleError(err_code, err_str)
		command.PrintError(s_err)
	}

	/* Create a new liner */
	liner, err := liner.NewLiner(viModeSingleLineFlag || viModeMultiLineFlag)
	if nil != err {
		command.PrintError(err)
		return
	}
	liner.SetMultiLineMode(!viModeSingleLineFlag)
	defer liner.Close()

	/* Load history from Home directory
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

	handleScriptFlag(&liner)
	handleIPModeFlag(&liner)

	// End handling the options

	n1ql.SetTxTimeout(_TXTIMEOUT)
	isTrunc := false
	for {
		line, err := liner.Prompt(fullPrompt)
		if err != nil {
			break
		}

		// Save previous length.
		lineLen := len(line)

		if lineLen == 4096 && !isTrunc {
			isTrunc = true
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Redirect command
		prevFile, outputFile = redirectTo(prevFile, prevreset, prevfgRed)

		if outputFile == os.Stdout {
			command.SetDispVal(prevreset, prevfgRed)
			command.SetWriter(os.Stdout)
		} else {
			if outputFile != nil {
				defer outputFile.Close()
				command.SetWriter(io.Writer(outputFile))
			}
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
		space := " "
		if isTrunc && lineLen < 4096 {
			space = ""
			isTrunc = false
		}

		if vliner.IsTerminatedStatement(inputLine...) {
			inputString := strings.Join(inputLine, space)
			for strings.HasSuffix(inputString, QRY_EOL) {
				inputString = strings.TrimSuffix(inputString, QRY_EOL)
			}
			if inputString != "" {
				err_code, err_string := UpdateHistory(liner, homeDir, inputString+QRY_EOL)
				if err_code != 0 {
					s_err := command.HandleError(err_code, err_string)
					command.PrintError(s_err)
				}
				// If outputting to a file, then add the statement to the file as well.
				if command.FILE_RW_MODE == true {
					_, werr := io.WriteString(command.W, "\n"+inputString+"\n")
					if werr != nil {
						s_err := command.HandleError(errors.E_SHELL_WRITER_OUTPUT, werr.Error())
						command.PrintError(s_err)
					}
				}
				err_code, err_string = dispatch_command(inputString, command.W, true, liner)
				/* Error handling for Shell errors and errors recieved from
				   godbc/n1ql.
				*/

				if err_code != 0 {
					s_err := command.HandleError(err_code, err_string)
					if err_code != errors.E_SHELL_DRIVER_QUERY_METHOD {
						// Dont print the error for query errors since we want to print the result.
						// Print all other errors
						command.PrintError(s_err)
					}

					if *errorExitFlag == true {
						if first == false {
							first = true
							_, werr := io.WriteString(command.W, command.EXITONERR)
							if werr != nil {
								s_err = command.HandleError(errors.E_SHELL_WRITER_OUTPUT, werr.Error())
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
				   the event an error is returned from dispatch_command after
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

/*
If ^C is pressed then Abort the shell. This is

	provided by the liner package.
*/
func signalCatcher(liner *liner.State) {
	liner.SetCtrlCAborts(false)

}

func redirectTo(prevFile, prevreset, prevfgRed string) (string, *os.File) {
	var err error
	var outputFile *os.File

	if command.FILE_RW_MODE == true {
		if prevFile != command.FILE_OUTPUT {
			prevFile = command.FILE_OUTPUT
			if command.FILE_APPEND_MODE {
				outputFile, err = os.OpenFile(command.FILE_OUTPUT, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
			} else {
				outputFile, err = os.OpenFile(command.FILE_OUTPUT, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
			}

			command.SetDispVal("", "")
			if err != nil {
				s_err := command.HandleError(errors.E_SHELL_OPEN_FILE, err.Error())
				command.PrintError(s_err)
				return prevFile, nil
			} else if command.FILE_APPEND_MODE && !quietFlag {
				io.WriteString(outputFile, "-- <"+time.Now().Format("2006-01-02T15:04:05.999Z07:00")+"> : opened in append mode\n")
			}
		}
	} else {
		command.SetDispVal(prevreset, prevfgRed)
		prevFile = ""
		outputFile = os.Stdout
	}
	return prevFile, outputFile
}
