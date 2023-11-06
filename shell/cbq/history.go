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
	"io"
	"os"
	"strings"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/shell/cbq/command"
	"github.com/couchbase/query/shell/liner"
)

func LoadHistory(liner *liner.State, dir string) (errors.ErrorCode, string) {
	if dir != "" {

		path := command.GetPath(dir, command.HISTFILE)
		err_code, err_str := ReadHistoryFromFile(liner, path)

		if err_code != 0 {
			return err_code, err_str
		}
		//Print path to histfile on startup.
		if !command.QUIET {
			io.WriteString(command.W, command.NewMessage(command.HISTORYMSG, path)+" \n")
		}
	}
	return 0, ""
}

func UpdateHistory(liner *liner.State, dir, line string) (errors.ErrorCode, string) {
	liner.AppendHistory(line)
	if dir != "" {
		path := command.GetPath(dir, command.HISTFILE)
		err_code, err_str := WriteHistoryToFile(liner, path)

		if err_code != 0 {
			return err_code, err_str
		}
	}
	return 0, ""
}

func WriteHistoryToFile(liner *liner.State, path string) (errors.ErrorCode, string) {

	var err error
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return errors.E_SHELL_OPEN_FILE, err.Error()
	}

	defer f.Close()

	writer := bufio.NewWriter(f)
	_, err = liner.WriteHistory(writer)
	if err != nil {
		return errors.E_SHELL_WRITE_FILE, err.Error()
	} else {
		err = writer.Flush()
		if err != nil {
			return errors.E_SHELL_WRITER_OUTPUT, err.Error()
		}
	}
	return 0, ""

}

func ReadHistoryFromFile(liner *liner.State, path string) (errors.ErrorCode, string) {

	var err error
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			f, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
			if err != nil {
				return errors.E_SHELL_OPEN_FILE, err.Error()
			}

		} else {
			return errors.E_SHELL_OPEN_FILE, err.Error()
		}
	}

	defer f.Close()

	reader := bufio.NewReader(f)
	_, err = liner.ReadHistory(reader)

	//Check for line too long errors. If the line didnt fit into the buffer
	//then dont report the error
	if err != nil && strings.Contains(err.Error(), "too long") {
		err = nil
	}

	if err != nil {
		return errors.E_SHELL_READ_FILE, err.Error()
	}

	return 0, ""
}
