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
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"sort"
	"strings"
	"unicode"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/shell/go_cbq/command"
	"github.com/couchbase/query/value"
	"github.com/sbinet/liner"
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

	if DISCONNECT == true || NoQueryService == true {
		if strings.HasPrefix(strings.ToLower(line), "\\connect") {
			NoQueryService = false
			command.DISCONNECT = false
			DISCONNECT = false
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
		   go_n1ql.
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
		// If connected to a query service then NoQueryService == false.
		if NoQueryService == true {
			//Not connected to a query service
			return errors.NO_CONNECTION, ""
		} else {
			/* Try opening a connection to the endpoint. If successful, ping.
			   If successful execute the n1ql command. Else try to connect
			   again.
			*/
			n1ql, err := sql.Open("n1ql", ServerFlag)
			if err != nil {
				return errors.GO_N1QL_OPEN, ""
			} else {
				//Successfully logged into the server
				err_code, err_str := ExecN1QLStmt(line, n1ql, w)
				if err_code != 0 {
					return err_code, err_str
				}
			}

		}
	}

	return 0, ""
}

func WriteHelper(rows *sql.Rows, columns []string, values, valuePtrs []interface{}, rownum int, isRawOrElement bool) ([]byte, int, string) {
	//Scan the values into the respective columns
	if err := rows.Scan(valuePtrs...); err != nil {
		return nil, errors.ROWS_SCAN, err.Error()
	}

	dat := map[string]*json.RawMessage{}
	var c []byte = nil
	var b []byte = nil
	var err error = nil

	for i, col := range columns {
		var parsed *json.RawMessage

		val := values[i]

		b, _ := val.([]byte)

		// Return input from go_n1ql as is (null). This case is seen when
		// the query tries to output RAW values and one of them is missing.
		if string(b) == "null" && isRawOrElement == true {
			return b, 0, ""
		}

		if string(b) != "" {
			//Parse the sub values of the main map first.
			err = json.Unmarshal(b, &parsed)
			if err != nil {
				return nil, errors.JSON_UNMARSHAL, err.Error()
			}

			//Fill up final result object
			dat[col] = parsed

		} else {
			continue
		}

		//Remove one level of nesting for the results when we have only 1 column to project.
		if len(columns) == 1 && dat[col] != nil {
			c, err = dat[col].MarshalJSON()
			if err != nil {
				return nil, errors.JSON_MARSHAL, err.Error()
			}
		}

	}

	b = nil
	err = nil

	// The first and second row represent the metadata. Because of the
	// way the rows are returned we need to create a map with the
	// correct data.
	if rownum == 0 || rownum == 1 {
		keys := make([]string, 0, len(dat))
		for key, _ := range dat {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		if keys != nil {
			map_value := dat[keys[0]]
			b, err = map_value.MarshalJSON()
			if err != nil {
				return nil, errors.JSON_MARSHAL, err.Error()
			}

		}

	} else {
		// If there is more than 1 column being projected, then
		// marshal and appropriately handle result.
		if len(columns) != 1 {
			b, err = json.Marshal(dat)
			if err != nil {
				return nil, errors.JSON_MARSHAL, err.Error()
			}
		} else {
			b = c
		}

	}

	var obj bool = true
	if *prettyFlag == true {

		tmpval := value.NewValue(b)
		if tmpval.Type() == value.OBJECT {
			obj = true
		} else {
			obj = false
		}

		var data map[string]interface{}
		if obj == true {

			if err := json.Unmarshal(b, &data); err != nil {
				return nil, errors.JSON_UNMARSHAL, err.Error()
			}

			b, err = json.MarshalIndent(data, "        ", "    ")
			if err != nil {
				return nil, errors.JSON_MARSHAL, err.Error()
			}
		}
	}

	return b, 0, ""
}

func ExecN1QLStmt(line string, n1ql *sql.DB, w io.Writer) (int, string) {
	//if strings.HasPrefix(strings.ToLower(line), "prepare") {

	//track if we need to return raw elements from anywhere in the query.
	isRaw := strings.Contains(strings.ToLower(line), "raw")
	isElement := strings.Contains(strings.ToLower(line), "element")

	rows, err := n1ql.Query(line)

	if err != nil {
		return errors.GON1QL_QUERY, err.Error()

	} else {
		iter := 0
		rownum := 0

		var werr error
		status := ""
		var metrics []byte
		metrics = nil

		// Multi column projection
		columns, _ := rows.Columns()
		count := len(columns)
		values := make([]interface{}, count)
		valuePtrs := make([]interface{}, count)

		//Check if spacing is enough
		_, werr = io.WriteString(w, "{\n")

		var prevRowResult []byte

		for rows.Next() {

			for i, _ := range columns {
				valuePtrs[i] = &values[i]
			}

			// The first 2 rows represent the metadata. Hence they need
			// to be explicitely handled.

			if rownum == 0 {

				// Get the first row to post process.

				extras, err_code, err_string := WriteHelper(rows, columns, values, valuePtrs, rownum, false)

				if extras == nil && err_code != 0 {
					return err_code, err_string
				}

				var dat map[string]interface{}

				if err := json.Unmarshal(extras, &dat); err != nil {
					return errors.JSON_UNMARSHAL, err.Error()
				}

				_, werr = io.WriteString(w, "    \"requestID\": \""+dat["requestID"].(string)+"\",\n")

				jsonString, err := json.MarshalIndent(dat["signature"], "        ", "    ")

				if err != nil {
					return errors.JSON_MARSHAL, err.Error()
				}
				_, werr = io.WriteString(w, "    \"signature\": "+string(jsonString)+",\n")
				_, werr = io.WriteString(w, "    \"results\" : [\n\t")
				status = dat["status"].(string)
				rownum++
				continue
			}

			// Get the second row
			if rownum == 1 {

				// Get the second row to post process as the metrics

				var err_code int
				var err_string string
				metrics, err_code, err_string = WriteHelper(rows, columns, values, valuePtrs, rownum, false)

				if metrics == nil && err_code != 0 {
					return err_code, err_string
				}

				//Wait until all the rows have been written to write the metrics.
				rownum++
				continue
			}

			//if rownum >=3 then print the rows
			if rownum > 2 {
				if iter == 0 {
					iter++
				} else {
					_, werr = io.WriteString(w, ", \n\t")
				}
				_, werr = io.WriteString(w, string(prevRowResult))
			}

			var err_code int
			var err_string string

			prevRowResult, err_code, err_string = WriteHelper(rows, columns, values, valuePtrs, rownum, (isRaw || isElement))
			if prevRowResult == nil && err_code != 0 {
				return err_code, err_string
			}
			rownum++

		} //rows.Next ends here

		//Suffix to result array
		_, werr = io.WriteString(w, "\n\t],")

		// The prevRowResult contains the output of the last row.
		// This is the errors row. Process this.
		var errorRow map[string]interface{}

		// Unmarshal the results of the errors object into errorRow
		// and then output that.
		if err := json.Unmarshal(prevRowResult, &errorRow); err != nil {
			return errors.JSON_UNMARSHAL, err.Error()
		}

		if errorRow["errors"] != nil {
			//When there are errors in this row. Print them.
			c, err := json.MarshalIndent(errorRow["errors"], "        ", "    ")
			if err != nil {
				return errors.JSON_MARSHAL, err.Error()
			}
			_, werr = io.WriteString(w, "\n")
			_, werr = io.WriteString(w, "    \"errors\" : ")
			_, werr = io.WriteString(w, string(c))
			_, werr = io.WriteString(w, ",")
		}

		err = rows.Close()
		if err != nil {
			return errors.ROWS_CLOSE, err.Error()
		}

		//Write the status and the metrics
		if status != "" {
			_, werr = io.WriteString(w, "\n    \"status\": \""+status+"\"")
		}
		if metrics != nil {
			_, werr = io.WriteString(w, ",\n    \"metrics\": ")
			_, werr = io.WriteString(w, string(metrics))
		}

		_, werr = io.WriteString(w, "\n}\n")

		// For any captured write error
		if werr != nil {
			return errors.WRITER_OUTPUT, werr.Error()
		}
	}

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
		if err_code != 0 {
			return err_code, err_str
		}
	} else {
		return errors.NO_SUCH_COMMAND, ""
	}

	SERVICE_URL = command.SERVICE_URL

	if SERVICE_URL != "" {
		ServerFlag = SERVICE_URL
		command.SERVICE_URL = ""
		SERVICE_URL = ""
	}

	DISCONNECT = command.DISCONNECT
	if DISCONNECT == true {
		NoQueryService = true

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
		if err == io.EOF {
			// Reached end of file. We are done. So break out of the loop.
			break
		} else if err != nil {
			return errors.READ_FILE, err.Error()
		}
		// Remove leading and trailing spaces from the input
		path = strings.TrimSpace(path)
		if strings.HasSuffix(path, ";") {
			// The full input command has been read.
			final_input = final_input + " " + path
		} else {
			// Only part of the command has been read. Hence continue
			// reading until ; is reached.
			final_input = final_input + " " + path
			continue
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
