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

	if strings.HasPrefix(line, "#") || (strings.HasPrefix(line, "--") && strings.Index(line, "\n") == -1) {
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
func trimSpaceInStr(inputStr string) string {
	whiteSpace := false
	var builder strings.Builder
	builder.Grow(len(inputStr))
	for _, character := range inputStr {
		if unicode.IsSpace(character) {
			if !whiteSpace {
				builder.WriteRune(' ')
			}
			whiteSpace = true
		} else {
			builder.WriteRune(character)
			whiteSpace = false
		}
	}
	return builder.String()
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

	var input *bufio.Reader

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
		input = bufio.NewReader(os.Stdin)

	} else {
		// Read input file
		inputFile, err := os.Open(command.FILE_INPUT)
		if err != nil {
			return errors.E_SHELL_OPEN_FILE, err.Error()
		}

		// Defer file close
		defer inputFile.Close()

		// Create a new reader for the file
		input = bufio.NewReader(inputFile)
	}

	// Variables for -advise option processing
	// If there is only 1 query in the file perform ADVISE
	// If there are multiple queries in the file perform SELECT ADVISOR[".."]; on them all
	// Note - we complete statements in the file to the ADVISOR query.
	// Syntax checks for complete statements will be performed in the engine.

	var adviseStmts []string
	var sb bytes.Buffer

	// read the input file extracting statements to run
	for eof := false; !eof; {
		lastStar := false
		quote := rune(0)
		comment := rune(0)
		pr := rune(0)
		sb.Reset()
		content := false
		lineStart := 0
		lineContent := false
		for done := false; !done; {
			r, _, err := input.ReadRune()
			if err != nil && err != io.EOF {
				return errors.E_SHELL_READ_FILE, err.Error()
			} else if err == io.EOF {
				eof = true
				done = true
				continue
			}
			switch {
			case comment == '#': // script only comment until EOL
				if r == '\n' {
					comment = rune(0)
				}
				r = rune(0)
			case comment == '*':
				if lastStar && r == '/' {
					comment = rune(0)
				}
				// as pr will be rune(0) when the adviseFlag is set, we must independently track seeing a '*' character
				// specifically for this comment style since it needs the two characters for the terminator
				lastStar = (r == '*')
				if adviseFlag {
					r = rune(0)
				}
			case comment == '-':
				if r == '\n' {
					comment = rune(0)
					if !adviseFlag {
						lineContent = true
					}
				}
				if adviseFlag && r != '\n' {
					r = rune(0)
				}
			case r == quote:
				quote = rune(0)
			case quote != rune(0):
			case r == '`' || r == '\'' || r == '"':
				quote = r
			case pr == '/' && r == '*':
				comment = r
				lastStar = false
				if adviseFlag {
					pr = rune(0)
					r = rune(0)
				}
			case pr == '-' && r == '-':
				comment = r
				if adviseFlag {
					pr = rune(0)
					r = rune(0)
				}
			case r == '#':
				comment = r
				r = rune(0)
			case r == ';':
				if !lineContent {
					sb.Truncate(lineStart)
				}
				done = true
			case pr == '\n':
				if !lineContent && adviseFlag {
					sb.Truncate(lineStart)
				}
				lineStart = sb.Len()
				lineContent = false
			default:
				if pr != rune(0) && !unicode.IsSpace(pr) {
					content = true
					lineContent = true
				}
			}
			if pr != rune(0) {
				sb.WriteRune(pr)
			}
			pr = r
		}

		stmt := strings.TrimSpace(sb.String())
		if eof && !adviseFlag && (stmt == "" || !content) {
			break
		} else if !eof && !adviseFlag && (stmt == "" || !content) {
			continue
		} else if adviseFlag {
			if stmt != "" && content {
				adviseStmts = append(adviseStmts, stmt)
			}
			if !eof {
				continue
			}
		}

		// Execute the statement
		// Write statement to shell, write statement to file, etc
		if adviseFlag {
			if len(adviseStmts) == 1 {
				stmt = ADVISE_PREFIX + adviseStmts[0]
			} else {
				var adviseStmt strings.Builder
				adviseStmt.WriteString(ADVISOR_PREFIX)
				for i := range adviseStmts {
					if i > 0 {
						adviseStmt.WriteRune(',')
					}
					adviseStmt.WriteString(fmt.Sprintf("%q", adviseStmts[i]))
				}
				adviseStmt.WriteString(ADVISOR_SUFFIX)
				stmt = adviseStmt.String()
			}
		}

		if !command.QUIET {
			command.OUTPUT.EchoCommand(string(stmt) + ";")
		}
		command.OUTPUT.WriteCommand(stmt)

		errCode, errStr := dispatch_command(stmt, false, liner)
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
