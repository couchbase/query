//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package command

import (
	"math"
	"sort"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/shell/pager"
)

const (
	ALIAS_CMD               = "ALIAS"
	CONNECT_CMD             = "CONNECT"
	DISCONNECT_CMD          = "DISCONNECT"
	EXIT_CMD                = "EXIT"
	QUIT_CMD                = "QUIT"
	HELP_CMD                = "HELP"
	VERSION_CMD             = "VERSION"
	COPYRIGHT_CMD           = "COPYRIGHT"
	SET_CMD                 = "SET"
	PUSH_CMD                = "PUSH"
	POP_CMD                 = "POP"
	UNSET_CMD               = "UNSET"
	ECHO_CMD                = "ECHO"
	UNALIAS_CMD             = "UNALIAS"
	SOURCE_CMD              = "SOURCE"
	REDIRECT_CMD            = "REDIRECT"
	REFRESH_CLUSTER_MAP_CMD = "REFRESH_CLUSTER_MAP"
	SYNTAX_CMD              = "SYNTAX"
)

const (
	MAX_ARGS  = math.MaxInt16
	ZERO_ARGS = 0
	ONE_ARG   = 1
	TWO_ARGS  = 2
)

var (
	//Used to get build version
	SHELL_VERSION = "unset: build issue" // is set by correct build process
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
	//Batch flag is used to send queries to the Asterix backend.
	BATCH = "off"
	//Terse output
	TERSE = false
	//Paged output
	PAGER = false
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

var OUTPUT = pager.NewPager()

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
	"\\syntax":    &Syntax{},

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

	"\\refresh_cluster_map": &Refresh_cluster_map{},
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
	ExecCommand(args []string) (errors.ErrorCode, string)
	/* Print Help information for command and its usage with an example */
	PrintHelp(desc bool) (errors.ErrorCode, string)
}
