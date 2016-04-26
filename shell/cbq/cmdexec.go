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

/*
This method executes the input command or statement. It
returns an error code and optionally a non empty error message.
*/
func execute_input(line string, w io.Writer, interactive bool, liner *liner.State) (int, string) {
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

		err_code, err_str := execute_input(val, w, interactive, liner)
		/* Error handling for Shell errors and errors recieved from
		   godbc/n1ql.
		*/
		if err_code != 0 {
			return err_code, err_str
		}

	} else if strings.HasPrefix(line, "\\") {
		//This block handles the shell commands
		err_code, err_str := ExecShellCmd(line, liner)
		if err_code != 0 {
			return err_code, err_str
		}

	} else {
		//This block handles N1QL statements
		// If connected to a query service then noQueryService == false.
		if noQueryService == true {
			//Not connected to a query service
			return errors.NO_CONNECTION, ""
		} else {
			/* Try opening a connection to the endpoint. If successful, ping.
			   If successful execute the n1ql command. Else try to connect
			   again.
			*/
			dBn1ql, err := n1ql.OpenExtended(serverFlag)
			if err != nil {
				return errors.DRIVER_OPEN, err.Error()
			} else {
				//Successfully logged into the server
				err_code, err_str := ExecN1QLStmt(line, dBn1ql, w)
				if err_code != 0 {
					return err_code, err_str
				}
			}

		}
	}

	return 0, ""
}

func ExecN1QLStmt(line string, dBn1ql n1ql.N1qlDB, w io.Writer) (int, string) {

	rows, err := dBn1ql.QueryRaw(line)

	if rows != nil {
		// We have output. That is what we want. We can ignore the error, even if there is one.

		_, werr := io.Copy(w, rows)

		// For any captured write error
		if werr != nil {
			return errors.WRITER_OUTPUT, werr.Error()
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
				err_str = err_str + "\n " + command.NewMessage(command.STARTUP, SERVICE_URL) + "\n"
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

			if path == "" && final_input == " " {
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

		// Print the query along with printing the
		io.WriteString(command.W, final_input+"\n")

		//Remove the ; before sending the query to execute
		final_input = strings.TrimSuffix(final_input, ";")

		// If outputting to a file, then add the statement to the file as well.
		if command.FILE_RW_MODE == true {
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
			io.WriteString(command.W, final_input+"\n")

		}

		errCode, errStr := execute_input(final_input, command.W, false, liner)
		if errCode != 0 {
			s_err := command.HandleError(errCode, errStr)
			command.PrintError(s_err)
		}
		io.WriteString(command.W, "\n\n")
		final_input = " "
	}
	return 0, ""
}
