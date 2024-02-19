//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"github.com/couchbase/godbc/n1ql"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/shell/cbq/command"
	"github.com/couchbase/query/shell/liner"
)

var batch_run = false

// For the -advise option
const ADVISE_PREFIX = "ADVISE "
const ADVISOR_PREFIX = "SELECT ADVISOR(["
const ADVISOR_SUFFIX = "])"

func command_alias(line string, interactive bool, liner *liner.State) (errors.ErrorCode, string) {
	// This block handles aliases
	commandkey := line[2:]
	commandkey = strings.TrimSpace(commandkey)

	val, ok := command.AliasCommand[commandkey]

	if !ok {
		return errors.E_SHELL_NO_SUCH_ALIAS, " : " + commandkey + command.NEWLINE
	}

	_, werr := command.OUTPUT.WriteCommand(val)
	if werr != nil {
		return errors.E_SHELL_WRITER_OUTPUT, werr.Error()
	}

	err_code, err_str := dispatch_command(val, interactive, liner)
	// Error handling for Shell errors and errors recieved from godbc/n1ql.
	if err_code != 0 {
		return err_code, err_str
	}
	return 0, ""

}

func command_shell(line string, interactive bool, liner *liner.State) (errors.ErrorCode, string) {
	if len(strings.TrimSpace(line)) == 1 {
		// A single \ (with whitespaces) is used to run the input statements
		// in batch mode for AsterixDB. For such a case, we run the statements the
		// same way we execute N1QL statements.
		// Run all the commands in the buffer in 1 go.

		batch_run = true

		err_code, err_str := dispatch_command(stringBuffer.String(), interactive, liner)
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

func command_query(line string, liner *liner.State) (errors.ErrorCode, string) {
	var err error
	//This block handles N1QL statements
	// If connected to a query service then noQueryService == false.
	if noQueryService {
		//Not connected to a query service
		return errors.E_SHELL_NO_CONNECTION, ""
	} else {
		// Check for batch mode.
		if command.BATCH == "on" && !batch_run {
			// This means we need to save the batched statements.
			// Set line to be the set of queries input in batch mode.
			// line = some buffer
			line = line + ";"
			_, err := stringBuffer.WriteString(line)
			if err != nil {
				return errors.E_SHELL_STRING_WRITE, err.Error()
			}
		} else {
			/* If a connection already exists, then use it.
			   Else Try opening a connection to the endpoint.
			   If successful execute the n1ql command.
			   Else try to connect again.
			*/

			if command.DbN1ql == nil {
				command.DbN1ql, err = n1ql.OpenExtended(serverFlag, command.USER_AGENT)
				if err != nil {
					return errors.E_SHELL_DRIVER_OPEN, err.Error()
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
				err_code, err_str := ExecN1QLStmt(line, command.DbN1ql)
				if err_code != 0 {
					// If the error is a connection error then refresh DbN1ql handle
					// retry the query or not ? How many times ?
					// For now retry it once
					if strings.Contains(err_str, "Connection failed") && retry == true {
						retry = false
						err = command.Ping(serverFlag)
						if err != nil {
							for _, s := range serverList {
								if s == serverFlag {
									continue
								}
								oerr := command.Ping(s)
								if oerr == nil {
									err = nil
									serverFlag = s
									break
								}
							}
							if err != nil {
								// There was an issue establishing a connection. Throw the error and return
								return errors.E_SHELL_CONNECTION_REFUSED, err.Error()
							}
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
func dispatch_command(line string, interactive bool, liner *liner.State) (errors.ErrorCode, string) {
	command.REFRESH_URL = serverFlag
	line = strings.TrimSpace(line)

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
		errCode, errStr := command_alias(line, interactive, liner)
		if errCode != 0 {
			return errCode, errStr
		}

	} else if strings.HasPrefix(line, "\\") {
		// handles shell commands
		errCode, errStr := command_shell(line, interactive, liner)
		if errCode != 0 {
			return errCode, errStr
		}

	} else {
		// handles input queries, both n1ql and asterix
		errCode, errStr := command_query(line, liner)
		if errCode != 0 {
			return errCode, errStr
		}

	}

	return 0, ""
}

func ExecN1QLStmt(line string, dBn1ql n1ql.N1qlDB) (errors.ErrorCode, string) {

	// Add back the ; for queries to support fully qualified
	// asterix queries along with N1QL queries.
	line = line + QRY_EOL
	rows, err := dBn1ql.QueryRaw(line)

	if rows != nil {
		// We have output. That is what we want.

		command.OUTPUT.Reset(true)

		var werr error
		if command.TERSE {
			werr = terseOutput(command.OUTPUT, rows)
		} else {
			_, werr = io.Copy(command.OUTPUT, rows)
		}
		if werr == io.EOF {
			werr = nil
			rows.Close()
		}

		// if "skip to end" has been selected, write out the last page after all copying is complete
		command.OUTPUT.Write([]byte{0x4})

		command.OUTPUT.Reset(false)

		// For any captured write error
		if werr != nil {
			return errors.E_SHELL_WRITER_OUTPUT, werr.Error()
		} else if err != nil {
			// Return error from godbc if there is one. This is for N1QL errors.
			return errors.E_SHELL_DRIVER_QUERY_METHOD, err.Error()
		}
		return 0, ""
	}

	if err != nil {
		return errors.E_SHELL_DRIVER_QUERY_METHOD, err.Error()
	}

	// No output, and no error. Strange, but keep going.
	return 0, ""
}

// Function to remove extra space in between words in a string.
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

func ExecShellCmd(line string, liner *liner.State) (errors.ErrorCode, string) {
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
			return errors.E_SHELL_UNBALANCED_QUOTES, ""
		}

	}

	cmd_args := strings.Split(line, " ")

	//Lookup Command from function registry

	Cmd, ok := command.COMMAND_LIST[cmd_args[0]]
	if ok == true {
		err_code, err_str := Cmd.ExecCommand(cmd_args[1:])
		if err_code == errors.E_SHELL_CONNECTION_REFUSED {
			if strings.TrimSpace(SERVICE_URL) == "" {
				command.OUTPUT.WriteString(command.NOCONNMSG)
			} else {
				err_str = err_str + "\n\n" + command.NewMessage(command.STARTUP, SERVICE_URL) + "\n"
			}
		}
		if err_code != 0 {
			return err_code, err_str
		}
	} else {
		return errors.E_SHELL_NO_SUCH_COMMAND, ""
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
// Note: if adviseFlag is set but inputFlag is unset - stdin is the file to read from instead of FILE_INPUT
func readAndExec(liner *liner.State) (errors.ErrorCode, string) {

	var newFileReader *bufio.Reader
	var isEOF = false

	if adviseFlag && inputFlag == "" {
		f, e := os.Stdin.Stat()
		if e != nil {
			return errors.E_SHELL_OPEN_FILE, e.Error()
		}

		// if stdin is empty, return error
		// prevents a hang on reading from empty stdin
		if f.Size() == 0 {
			return errors.E_SHELL_OPEN_FILE, "stdin: stdin is empty"
		}

		// Create a new reader for the file
		newFileReader = bufio.NewReader(os.Stdin)

	} else {
		// Read input file
		inputFile, err := os.Open(command.FILE_INPUT)
		if err != nil {
			return errors.E_SHELL_OPEN_FILE, err.Error()
		}

		// Defer file close
		defer inputFile.Close()

		// Create a new reader for the file
		newFileReader = bufio.NewReader(inputFile)
	}

	// Final input command string to be executed
	final_input := " "

	// Variables for -advise option processing
	// If there is only 1 query in the file perform ADVISE
	// If there are > 1 queries in the file perform SELECT ADVISOR[".."]; on all queries
	// Note - we append any line in the file ( except comments ) to the ADVISOR query.
	// Syntax checks of the appended lines will be done in the engine.

	adviseQuery := ADVISOR_PREFIX // final advise/ advisor statement to be executed
	qCount := 0                   // number of queries so far
	adviseEnd := false            // if all queries have been consumed and we can execute the created ADVISE/ ADVISOR stmt
	prevQ := ""
	noAdvise := !adviseFlag

	// Loop through th file for every line.
	for {

		// Read the line until a new line character. If it contains a ;
		// at the end of the read then that is the query to run. If not
		// keep appending to the string until you reach the ;\n.
		path, err := newFileReader.ReadString('\n')

		if err != nil && err != io.EOF {
			return errors.E_SHELL_READ_FILE, err.Error()
		}

		// Remove leading and trailing spaces from the input
		path = strings.TrimSpace(path)

		if err == io.EOF {
			// Reached end of file. We are done. So break out of the loop.
			// Do not require the last line on the file to have a \n.

			if path == "" && final_input == " " || path == "" && final_input == ";" {
				if noAdvise {
					break
				}
				adviseEnd = true

			} else {
				//This means we have some text on the last line
				isEOF = true
			}

		}

		// Process the line read
		if noAdvise || !adviseEnd {
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

			// add the queries to the ADVISOR statement
			if adviseFlag {
				qCount++
				// Format specifier %q is used for escaping necessary characters
				// since each queries will be within ""  in the array of queries in ADVISOR statement
				if qCount == 1 {
					adviseQuery += fmt.Sprintf("%q", final_input)
				} else {
					adviseQuery += fmt.Sprintf(",%q", final_input)
				}
				prevQ = final_input
			}
		}

		// Execute the statement
		// Write statement to shell, write statement to file, etc
		if noAdvise || adviseEnd {

			if adviseFlag {
				// if only 1 query in the file - perform ADVISE on it
				if qCount == 1 {
					adviseQuery = ADVISE_PREFIX + prevQ
				} else {
					// if > 1 query in the file - perfom SELECT ADVISOR([...]) on the list of queries
					adviseQuery += ADVISOR_SUFFIX
				}
				final_input = adviseQuery
			}

			// Print the query along with printing the results, only if -q isnt specified.
			if !command.QUIET {
				command.OUTPUT.EchoCommand(string(final_input))
			}

			//Remove the ; before sending the query to execute
			final_input = strings.TrimSuffix(final_input, ";")

			command.OUTPUT.WriteCommand(final_input)

			// If outputting to a file, then add the statement to the file as well.

			errCode, errStr := dispatch_command(final_input, false, liner)
			if errCode != 0 {
				s_err := command.HandleError(errCode, errStr)
				command.PrintError(s_err)

				if *errorExitFlag {
					_, werr := command.OUTPUT.WriteString(command.EXITONERR)
					if werr != nil {
						command.PrintError(command.HandleError(errors.E_SHELL_WRITER_OUTPUT, werr.Error()))
					}
					liner.Close()
					os.Clearenv()
					os.Exit(1)
				}
			}
			command.OUTPUT.WriteString(command.NEWLINE + command.NEWLINE)

			if adviseEnd {
				break
			}
		}

		final_input = " "
	}
	return 0, ""
}

// terse output
const _INDENT = "    "
const _INITIAL_BUFFER_SIZE = 1024 * 1024

var (
	_ELEM_SEP = []byte("\n,")
	_START    = []byte("[\n")
	_END      = []byte("\n]\n")
	_NL       = []byte("\n")
)

func terseOutput(w io.Writer, rows io.ReadCloser) error {
	pretty := *prettyFlag
	if v, ok := command.QueryParam["pretty"]; ok {
		if v, ec, _ := v.Top(); ec == 0 {
			pretty = v.Truth()
		}
	}
	status := make(map[string]interface{})

	buf := make([]byte, _INITIAL_BUFFER_SIZE)
	i := 0
	s := 0
	var err error
	level := 0
	quoted := false
	res := false

	// skip everything until the results or errors section
	for {
		buf = buf[:cap(buf)]
		nr, err := rows.Read(buf[s:])
		if (err == io.EOF && nr == 0) || (err != nil && err != io.EOF) {
			return err
		}
		buf = buf[:s+nr]
		i = bytes.Index(buf, []byte("\"results\""))
		if i != -1 {
			i += len("\"results\": [\n")
			buf = buf[i:]
			break
		}
		i = bytes.Index(buf, []byte("\"errors\""))
		if i != -1 {
			// if we find the errors element before results it means there are no results and we'll fully process
			// the entire server return, so reset the start point to the beginning of the buffer
			i = 0
			goto status
		}
		// keep reading into / expanding the buffer until we find one of these
		s = len(buf)
		if s == cap(buf) {
			buf = append(buf, make([]byte, cap(buf))...)
		}
	}

	// pass on the results array uninterpreted (save for working out where it ends)
results:
	for {
		for i = 0; i < len(buf); i++ {
			switch {
			case buf[i] == '\\':
				i++
			case quoted:
				if buf[i] == '"' {
					quoted = false
				}
			case buf[i] == '"':
				quoted = true
			case buf[i] == '[':
				level++
			case buf[i] == ']':
				level--
				if level < 0 {
					j := i
					for j > 0 {
						if buf[j] == '\n' {
							break
						}
						j--
					}
					if j > 0 {
						if pretty && !res {
							w.Write(_START)
						}
						res = true
						w.Write(buf[:j])
					}
					i++
					buf[i] = '{'
					break results
				}
			}
		}
		if pretty && !res {
			w.Write(_START)
		}
		_, werr := w.Write(buf)
		if werr == io.EOF {
			return nil
		}
		res = true
		buf = buf[:cap(buf)]
		nr, err := rows.Read(buf)
		if err != nil && (err != io.EOF || nr == 0) {
			return err
		}
		buf = buf[:nr]
	}

status:
	// we should have only the meta-data to process at this point
	dec := json.NewDecoder(io.MultiReader(bytes.NewReader(buf[i:]), rows))
	var v map[string]interface{}
	err = dec.Decode(&v)
	if err != nil {
		return err
	}

	if val, ok := v["status"]; ok {
		status["status"] = val
	}
	if val, ok := v["errors"]; ok {
		status["errors"] = val
	}
	if val, ok := v["warnings"]; ok {
		status["warnings"] = val
	}
	if im, ok := v["metrics"]; ok {
		if m, ok := im.(map[string]interface{}); ok {
			if cnt, ok := m["mutationCount"]; ok {
				status["mutationCount"] = cnt
			} else if cnt, ok := m["resultCount"]; ok {
				status["resultCount"] = cnt
			}
		}
	}

	if len(status) > 1 || status["status"] != "success" {
		if res {
			w.Write(_ELEM_SEP)
		} else if pretty {
			w.Write(_START)
		}
		var b []byte
		var err error
		if pretty {
			if pretty && res {
				w.Write([]byte(_INDENT[1:]))
			} else {
				w.Write([]byte(_INDENT))
			}
			b, err = json.MarshalIndent(status, _INDENT, _INDENT)
		} else {
			b, err = json.Marshal(status)
		}
		if err != nil {
			return err
		}
		_, werr := w.Write(b)
		if werr == io.EOF {
			return nil
		}
		res = true
	}
	if res {
		if pretty {
			w.Write(_END)
		} else {
			w.Write(_NL)
		}
	}
	return nil
}
