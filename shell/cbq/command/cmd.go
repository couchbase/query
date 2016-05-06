//  Copyright (c) 2015-2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package command

import (
	"io"
	"math"
	"sort"
)

const (
	ALIAS_CMD      = "ALIAS"
	CONNECT_CMD    = "CONNECT"
	DISCONNECT_CMD = "DISCONNECT"
	EXIT_CMD       = "EXIT"
	QUIT_CMD       = "QUIT"
	HELP_CMD       = "HELP"
	VERSION_CMD    = "VERSION"
	COPYRIGHT_CMD  = "COPYRIGHT"
	SET_CMD        = "SET"
	PUSH_CMD       = "PUSH"
	POP_CMD        = "POP"
	UNSET_CMD      = "UNSET"
	ECHO_CMD       = "ECHO"
	UNALIAS_CMD    = "UNALIAS"
	SOURCE_CMD     = "SOURCE"
	REDIRECT_CMD   = "REDIRECT"
)

const (
	SHELL_VERSION = "1.5"

	MAX_ARGS  = math.MaxInt16
	ZERO_ARGS = 0
	ONE_ARG   = 1
	TWO_ARGS  = 2
)

var (
	//Used to manage connections
	SERVICE_URL = ""
	//Used to disconnect from the endpoint
	DISCONNECT = false
	//Used to quit shell
	EXIT = false
	//Used to check for files
	FILE_INPUT = ""
	//True if reading commands from file
	FILE_RD_MODE = false
	//True if writing commands to file
	FILE_RW_MODE = false
	//File to redirect output to
	FILE_OUTPUT = ""
	//Total no. of commands
	MAX_COMMANDS = len(COMMAND_LIST)
	//File to store History in
	HISTFILE = ".cbq_history"
	//Is this running on windows
	WINDOWS = false
	//Quiet flag used to suppress printing history path
	QUIET = false
	//Value that represents the no-ssl-verify flag
	SKIPVERIFY = false
)

/* Value to store sorted list of keys for shell commands */
var _SORTED_CMD_LIST []string

func init() {
	_SORTED_CMD_LIST = make([]string, MAX_COMMANDS, MAX_COMMANDS)
	i := 0
	for k, _ := range COMMAND_LIST {
		_SORTED_CMD_LIST[i] = k
		i++
	}
	sort.Strings(_SORTED_CMD_LIST)
}

/*
	Define a common writer to output the responses to.
*/
var W io.Writer

/*
	Used to define aliases
*/
var AliasCommand = map[string]string{
	"serverversion": "select version()",
}

/*
	Command registry : List of Shell Commands supported by cbq
*/
var COMMAND_LIST = map[string]ShellCommand{

	/* Connection Management */
	"\\connect":    &Connect{},
	"\\disconnect": &Disconnect{},
	"\\exit":       &Exit{},
	"\\quit":       &Exit{},

	/* Shell and Server Information */
	"\\help":      &Help{},
	"\\version":   &Version{},
	"\\copyright": &Copyright{},

	/* Session Management */
	"\\set":     &Set{},
	"\\push":    &Push{},
	"\\pop":     &Pop{},
	"\\unset":   &Unset{},
	"\\echo":    &Echo{},
	"\\alias":   &Alias{},
	"\\unalias": &Unalias{},

	/* Scripting Management */
	"\\source":   &Source{},
	"\\redirect": &Redirect{},
}

/*
	Interface to be implemented by shell commands.
*/
type ShellCommand interface {
	/* Name of the comand */
	Name() string
	/* Return true if included in shell command completion */
	CommandCompletion() bool
	/* Returns the Minimum number of input arguments required by the function */
	MinArgs() int
	/* Returns the Maximum number of input arguments allowed by the function */
	MaxArgs() int
	/* Method that implements the functionality */
	ExecCommand(args []string) (int, string)
	/* Print Help information for command and its usage with an example */
	PrintHelp(desc bool) (int, string)
}
