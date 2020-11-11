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
	"bufio"
	"io"
	"os"
	"strings"
	"unicode"

	"github.com/couchbase/godbc/n1ql"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/shell/cbq/command"
	"github.com/peterh/liner"
)

var batch_run = false

func command_alias(line string, w io.Writer, interactive bool, liner *liner.State) (int, string) {
	// This block handles aliases
	commandkey := line[2:]
	commandkey = strings.TrimSpace(commandkey)

	val, ok := command.AliasCommand[commandkey]

	if !ok {
		return errors.NO_SUCH_ALIAS, " : " + commandkey + "\n"
	}

	// If outputting to a file, then add the statement to the file as well.
	if command.FILE_RW_MODE == true {
		_, werr := io.WriteString(command.W, val+"\n")
		if werr != nil {
			return errors.WRITER_OUTPUT, werr.Error()
		}
	}

	err_code, err_str := dispatch_command(val, w, interactive, liner)
	/* Error handling for Shell errors and errors recieved from
	   godbc/n1ql.
	*/
	if err_code != 0 {
		return err_code, err_str
	}
	return 0, ""

}

func command_shell(line string, w io.Writer, interactive bool, liner *liner.State) (int, string) {
	if len(strings.TrimSpace(line)) == 1 {
		// A single \ (with whitespaces) is used to run the input statements
		// in batch mode for AsterixDB. For such a case, we run the statements the
		// same way we execute N1QL statements.
		// Run all the commands in the buffer in 1 go.

		batch_run = true

		err_code, err_str := dispatch_command(stringBuffer.String(), w, interactive, liner)
		stringBuffer.Reset()
		if err_code != 0 {
			return err_code, err_str
		}

	} else {
		//This block handles the shell commands
		err_code, err_str := ExecShellCmd(line, liner)
		if err_code != 0 {
			return err_code, err_str
		}
	}
	return 0, ""
}

func command_query(line string, w io.Writer, liner *liner.State) (int, string) {
	command.REFRESH_URL = serverFlag
	var err error
	//This block handles N1QL statements
	// If connected to a query service then noQueryService == false.
	if noQueryService {
		//Not connected to a query service
		return errors.NO_CONNECTION, ""
	} else {
		// Check for batch mode.
		if command.BATCH == "on" && !batch_run {
			// This means we need to save the batched statements.
			// Set line to be the set of queries input in batch mode.
			// line = some buffer
			line = line + ";"
			_, err := stringBuffer.WriteString(line)
			if err != nil {
				return errors.STRING_WRITE, err.Error()
			}
		} else {
			/* If a connection already exists, then use it.
			   Else Try opening a connection to the endpoint.
			   If successful execute the n1ql command.
			   Else try to connect again.
			*/

			if command.DbN1ql == nil {
				command.DbN1ql, err = n1ql.OpenExtended(serverFlag)
				if err != nil {
					return errors.DRIVER_OPEN, err.Error()
				}
			}

			// Check if the statement needs to be executed.
			if batch_run {
				batch_run = false
			}

			if line == "" {
				// In batch mode, if we try execute without any input,
				// then dont execute anything.
				return 0, ""
			}

			retry := true
			for {
				err_code, err_str := ExecN1QLStmt(line, command.DbN1ql, w)
				if err_code != 0 {
					// If the error is a connection error then refresh DbN1ql handle
					// retry the query or not ? How many times ?
					// For now retry it once
					if strings.Contains(err_str, "Connection failed") && retry == true {
						retry = false
						err = command.Ping(serverFlag)
						if err != nil {
							// There was an issue establishing a connection. Throw the error and return
							return errors.CONNECTION_REFUSED, err.Error()
						}
					} else {
						return err_code, err_str
					}
				} else {
					break
				}
			}
		}

	}
	return 0, ""
}

/*
This method is the handler that calls execution methods based on the input command
or statement. It returns an error code and optionally a non empty error message.
*/
func dispatch_command(line string, w io.Writer, interactive bool, liner *liner.State) (int, string) {
	line = strings.TrimSpace(line)
	command.W = w

	if interactive == false {
		// Check if the line ends with a ;
		line = strings.TrimSpace(line)
		semiC := ""
		if !strings.HasSuffix(line, ";") {
			semiC = ";"
		}
		errCode, errStr := UpdateHistory(liner, homeDir, line+semiC)
		if errCode != 0 {
			s_err := command.HandleError(errCode, errStr)
			command.PrintError(s_err)
		}

	}

	if DISCONNECT == true || noQueryService == true {
		if strings.HasPrefix(strings.ToLower(line), "\\connect") {
			noQueryService = false
			command.DISCONNECT = false
			DISCONNECT = false
			SERVICE_URL = ""
		}
	}

	// Handle comments here as well. This is useful for the \source
	// command and the --file and --script options.
	if strings.HasPrefix(line, "--") || strings.HasPrefix(line, "#") {
		return 0, ""
	}

	if strings.HasPrefix(line, "\\\\") {
		// handles aliases
		errCode, errStr := command_alias(line, w, interactive, liner)
		if errCode != 0 {
			return errCode, errStr
		}

	} else if strings.HasPrefix(line, "\\") {
		// handles shell commands
		errCode, errStr := command_shell(line, w, interactive, liner)
		if errCode != 0 {
			return errCode, errStr
		}

	} else {
		// handles input queries, both n1ql and asterix
		errCode, errStr := command_query(line, w, liner)
		if errCode != 0 {
			return errCode, errStr
		}

	}

	return 0, ""
}

func ExecN1QLStmt(line string, dBn1ql n1ql.N1qlDB, w io.Writer) (int, string) {

	// Add back the ; for queries to support fully qualified
	// asterix queries along with N1QL queries.
	line = line + QRY_EOL
	rows, err := dBn1ql.QueryRaw(line)

	if rows != nil {
		// We have output. That is what we want.

		_, werr := io.Copy(w, rows)

		// For any captured write error
		if werr != nil {
			return errors.WRITER_OUTPUT, werr.Error()
		} else if err != nil {
			// Return error from godbc if there is one. This is for N1QL errors.
			return errors.DRIVER_QUERY, err.Error()
		}
		return 0, ""
	}

	if err != nil {
		return errors.DRIVER_QUERY, err.Error()
	}

	// No output, and no error. Strange, but keep going.
	return 0, ""
}

//Function to remove extra space in between words in a string.
func trimSpaceInStr(inputStr string) (outputStr string) {
	whiteSpace := false
	for _, character := range inputStr {
		if unicode.IsSpace(character) {
			if !whiteSpace {
				outputStr = outputStr + " "
			}
			whiteSpace = true
		} else {
			outputStr = outputStr + string(character)
			whiteSpace = false
		}
	}
	return
}

func ExecShellCmd(line string, liner *liner.State) (int, string) {
	line = strings.TrimSpace(line)
	arg1 := strings.Split(line, " ")
	arg1str := strings.ToLower(arg1[0])

	line = arg1str + " " + strings.Join(arg1[1:], " ")
	line = strings.TrimSpace(line)

	line = trimSpaceInStr(line)

	// Handle input strings to \echo command.
	if strings.HasPrefix(line, "\\echo") {

		count_param := strings.Count(line, "\"")
		count_param_bs := strings.Count(line, "\\\"")

		if count_param%2 == 0 && count_param_bs%2 == 0 {
			r := strings.NewReplacer("\\\"", "\\\"", "\"", "")
			line = r.Replace(line)

		} else {
			return errors.UNBALANCED_PAREN, ""
		}

	}

	cmd_args := strings.Split(line, " ")

	//Lookup Command from function registry

	Cmd, ok := command.COMMAND_LIST[cmd_args[0]]
	if ok == true {
		err_code, err_str := Cmd.ExecCommand(cmd_args[1:])
		if err_code == errors.CONNECTION_REFUSED {
			if strings.TrimSpace(SERVICE_URL) == "" {
				io.WriteString(command.W, command.NOCONNMSG)
			} else {
				err_str = err_str + "\n\n" + command.NewMessage(command.STARTUP, SERVICE_URL) + "\n"
			}
		}
		if err_code != 0 {
			return err_code, err_str
		}
	} else {
		return errors.NO_SUCH_COMMAND, ""
	}

	SERVICE_URL = command.SERVICE_URL

	// Reset the Server flag and Service_url to the current connection string.
	if SERVICE_URL != "" {
		serverFlag = SERVICE_URL
		command.SERVICE_URL = ""
	}

	DISCONNECT = command.DISCONNECT
	if DISCONNECT == true {
		noQueryService = true

	}

	EXIT = command.EXIT

	// File based input. Run all the commands as seen in the file
	// given by FILE_INPUT and then return the prompt.
	if strings.HasPrefix(line, "\\source") && command.FILE_RD_MODE == true {
		errCode, errStr := readAndExec(liner)
		if errCode != 0 {
			return errCode, errStr
		}

	} // ends main if loop for

	return 0, ""
}

// Helper function to read file based input. Run all the commands as
// seen in the file given by FILE_INPUT and then return the prompt.
func readAndExec(liner *liner.State) (int, string) {

	var isEOF = false

	// Read input file
	inputFile, err := os.Open(command.FILE_INPUT)
	if err != nil {
		return errors.FILE_OPEN, err.Error()
	}

	// Defer file close
	defer inputFile.Close()

	// Create a new reader for the file
	newFileReader := bufio.NewReader(inputFile)

	// Final input command string to be executed
	final_input := " "

	// For redirect command
	outputFile := os.Stdout
	prevFile := ""
	prevreset := command.Getreset()
	prevfgRed := command.GetfgRed()

	// Loop through th file for every line.
	for {

		// Read the line until a new line character. If it contains a ;
		// at the end of the read then that is the query to run. If not
		// keep appending to the string until you reach the ;\n.
		path, err := newFileReader.ReadString('\n')

		if err != nil && err != io.EOF {
			return errors.READ_FILE, err.Error()
		}

		// Remove leading and trailing spaces from the input
		path = strings.TrimSpace(path)

		if err == io.EOF {
			// Reached end of file. We are done. So break out of the loop.
			// Do not require the last line on the file to have a \n.

			if path == "" && final_input == " " || path == "" && final_input == ";" {
				break
			} else {
				//This means we have some text on the last line
				isEOF = true
			}

		}

		// If any line has a comment, dont count that line for the input,
		// add it to the history and then discard it.

		if strings.HasPrefix(path, "--") || strings.HasPrefix(path, "#") {
			errCode, errStr := UpdateHistory(liner, homeDir, path)
			if errCode != 0 {
				return errCode, errStr
			}
			continue
		}

		if strings.HasSuffix(path, ";") {
			// The full input command has been read.
			final_input = final_input + " " + path
		} else {
			if isEOF {
				// The last line of the file has been read and it doesnt
				// contain a ;. Hence append one and then run.
				final_input = final_input + " " + path + ";"

			} else {
				// Only part of the command has been read. Hence continue
				// reading until ; is reached.
				final_input = final_input + " " + path
				continue
			}
		}

		// Populate the final string to execute
		final_input = strings.TrimSpace(final_input)

		if final_input == ";" {
			continue
		}

		// Print the query along with printing the results, only if -q isnt specified.
		if !command.QUIET {
			io.WriteString(command.W, final_input+"\n")
		}

		//Remove the ; before sending the query to execute
		final_input = strings.TrimSuffix(final_input, ";")

		// If outputting to a file, then add the statement to the file as well.

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

		if command.FILE_RW_MODE == true {
			io.WriteString(command.W, final_input+"\n")
		}

		errCode, errStr := dispatch_command(final_input, command.W, false, liner)
		if errCode != 0 {
			s_err := command.HandleError(errCode, errStr)
			command.PrintError(s_err)

			if *errorExitFlag {
				_, werr := io.WriteString(command.W, command.EXITONERR)
				if werr != nil {
					command.PrintError(command.HandleError(errors.WRITER_OUTPUT, werr.Error()))
				}
				liner.Close()
				os.Clearenv()
				os.Exit(1)
			}
		}
		io.WriteString(command.W, "\n\n")
		final_input = " "
	}
	return 0, ""
}
