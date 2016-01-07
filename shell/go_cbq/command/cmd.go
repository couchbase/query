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
)

const (
	SHELL_VERSION = "1.0"

	MAX_ARGS    = math.MaxInt16
	MAX_ALIASES = math.MaxInt16
	MAX_VARS    = math.MaxInt16
)

var (
	//Used to manage connections
	SERVICE_URL = ""
	//Used to disconnect from the endpoint
	DISCONNECT = false
	//Used to quit shell
	EXIT = false
	//Used to check for files
	FILE_INPUT = false
	//Total no. of commands
	MAX_COMMANDS = len(COMMAND_LIST)
	//Total number of
)

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
