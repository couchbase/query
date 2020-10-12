//  Copyright (c) 2015-2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

Package message provides user-visible messages for cbq. These
include message codes and will eventually provide multi-language
messages.

*/

package command

import "fmt"

const (
	//Usage messages for flags. U-> Usage
	USERVERFLAG = " URL to the query service/cluster. \n\t\t Default : http://localhost:8091\n\t\t For example : ./cbq -e couchbase://172.6.23.2\n\t\t\t       ./cbq -e http://172.23.107.18:8091\n"
	UNOENGINE   = " Start shell without connecting to a query service/cluster endpoint. \n\t\t Default : false \n\t\t Possible values : true,false"
	UQUIET      = " Enable/Disable startup connection message for the shell \n\t\t Default : false \n\t\t Possible values : true,false"
	UTIMEOUT    = " Query timeout parameter. Units are mandatory. \n\t\tFor example : -timeout \"10ms\". \n\t\tValid units : ns, us, ms, s, m, h"
	UUSER       = " Username \n\t For example : -u Administrator"
	UPWD        = " Password \n\t For example : -p password"
	UCREDS      = " A list of credentials, in the form user:password. \n\t For example : -c beer-sample:pass"
	UVERSION    = " Shell Version \n\t Usage: -version"
	USCRIPT     = " Single command mode. Execute input command and exit shell. \n\t For example : -script \"select * from system:keyspaces\""
	UPRETTY     = " Pretty print the output."
	UEXIT       = " Exit shell after first error encountered."
	UINPUT      = " File to load commands from. \n\t For example : -file temp.txt"
	UOUTPUT     = " File to output commands and their results. \n\t For example : -output temp.txt"
	USSLVERIFY  = " Skip verification of Certificates. "
	UBATCH      = " Batch mode for sending queries to the Analytics service. Values : on/off"
	UANALYTICS  = " Auto discover analytics server from input cluster. Batch mode on. "
	UNETWORK    = " Choose network address support. \n\tAuto - Use either alternate address or not depending on input. \n\t Default - Use internal address. \n\t external - Use alternate addresses. "
	UCACERT     = " Path to root ca certificate to verify identity of server. \n\t For example : -cacert ./root/ca.pem"
	UCERTFILE   = " Path to chain certificate. \n\t For example : -cert ./client/client/chain.pem"
	UKEYFILE    = " Path to client key file. \n\t For example : -key ./client/client/client.key"

	//Shorthand message for flags
	SHORTHAND = " Shorthand for "

	//User facing messages from package main
	PWDMSG       = " Enter Password: "
	STARTUPCREDS = " No input credentials. In order to connect to a server with authentication, please provide credentials.\n"
	STARTUP      = " Connected to"
	EXITMSG      = ". Type Ctrl-D or \\QUIT to exit.\n"
	EXITONERR    = "\n Exiting on first error encountered.\n"
	HISTORYMSG   = "\n Path to history file for the shell"
	NOCONNMSG    = "\n Couchbase query shell not connected to any endpoint. Use \\CONNECT command to connect.\n"

	//Messages for each command

	ERRHOME = "\nUnable to determine home directory, history file disabled.\n"

	//COPYRIGHT, VERSION
	COPYRIGHTMSG = "\nCopyright (c) 2016 Couchbase, Inc. Licensed under the Apache License, " +
		"Version 2.0 (the \"License\"); \nyou may not use this file except in " +
		"compliance with the License. You may obtain a copy of the \nLicense at " +
		"http://www.apache.org/licenses/LICENSE-2.0\nUnless required by applicable " +
		"law or agreed to in writing, software distributed under the\nLicense is " +
		"distributed on an \"AS IS\" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF " +
		"ANY KIND,\neither express or implied. See the License for the specific " +
		"language governing permissions\nand limitations under the License.\n"

	VERSIONMSG       = " SHELL VERSION "
	SERVERVERSIONMSG = "\n Use N1QL queries select version(); or select min_version(); to display server version.\n"

	//SET
	QUERYP   = " Query Parameters : \n"
	NAMEDP   = " Named Parameters : \n"
	PREDEFP  = " Predefined Session Parameters : \n"
	USERDEFP = " User Defined Session Parameters : \n"
	PNAME    = " Parameter name"
	PVAL     = " Value"

	//SSL
	SSLVERIFY_FALSE = "\n If you are using self signed certificates you can rerun this command with the " +
		"-no-ssl-verify flag.\n Note however that disabling SSL verification means that cbq will be " +
		"vulnerable to man-in-the-middle attacks.\n\n"

	SSLVERIFY_TRUE = "\n Disabling SSL verification means that cbq will be vulnerable to man-in-the-middle attacks.\n\n"

	INVALIDPORT = "When specifying couchbase:// or couchbases://, do not specify the port.\n"
	INVALIDHOST = " Empty host name.\n"

	//HELP H-> Help
	HELPMSG             = "\nHelp information for all shell commands.\n\n"
	HALIAS              = "\\ALIAS [ name value ]\n"
	HUNALIAS            = "\\UNALIAS name ...\n"
	HCONNECT            = "\\CONNECT url\n"
	HDISCONNECT         = "\\DISCONNECT\n"
	HCOPYRIGHT          = "\\COPYRIGHT\n"
	HVERSION            = "\\VERSION\n"
	HECHO               = "\\ECHO args ...\n"
	HEXIT               = "\\QUIT \n\\EXIT\n"
	HHELP               = "\\HELP [ args ... ]\n"
	HSET                = "\\SET [ parameter value ]\n"
	HPUSH               = "\\PUSH [ parameter value ]\n"
	HUNSET              = "\\UNSET parameter\n"
	HPOP                = "\\POP [ parameter ]\n"
	HREDIRECT           = "\\REDIRECT OFF | filename \n"
	HSOURCE             = "\\SOURCE filename\n"
	HREFRESH_CLUSTERMAP = "\\REFRESH_CLUSTER_MAP\n"

	//Messages to print description of shell commands. D-> Description
	DALIAS = " Create an alias (name) for input value. value can be shell command, " +
		"query statement or string.\nIf no arguments are given, list all existing alias.\n" +
		"\tExample : \n\t        \\ALIAS serverversion \"select version(), min_version()\"" +
		" ;\n\t        \\ALIAS \"\\SET -max-parallelism 8\";\n"

	DCONNECT = "Connect to the query service or cluster endpoint URL.\n" +
		"Default : http://localhost:8091\n" +
		"\tExample : \n\t        \\CONNECT couchbase://172.6.23.2 ; \n\t        " +
		"\\CONNECT http://172.6.23.2:8091 ;\n\t        " +
		"\\CONNECT https://my.secure.node.com:18093 ;\n"

	DCOPYRIGHT = "Print Couchbase copyright information.\n" +
		"\tExample : \n\t        \\COPYRIGHT;\n"

	DDISCONNECT = "Disconnect from the query service or cluster endpoint.\n" +
		"\tExample : \n\t        \\DISCONNECT;\n"

	DECHO = "Echo the input value. args can be a name (a prefixed-parameter), an alias " +
		"(command alias) or \na value (any input statement).\n" +
		"\tExample : \n\t        \\ECHO -$r ;\n\t        \\ECHO \\\\tempalias; \n"

	DEXIT = "Exit the shell.\n" +
		"\tExample : \n\t        \\EXIT; \n\t        \\QUIT;\n"

	DHELP = "Display help information for input shell commands. If no args are given, " +
		"it lists all existing shell commands.\n" +
		"\tExample : \n\t        \\HELP VERSION; \n\t        \\HELP EXIT DISCONNECT VERSION; \n\t        \\HELP;\n"

	DPOP = "Pop the value of the given parameter from the input parameter stack. " +
		"parameter is a prefixed name (-creds, -$rate, $user, histfile).\nIf no " +
		"arguments are given, it pops the stack for each parameter excluding the pre-defined parameters.\n" +
		"\tExample : \n\t        \\POP -$r ;\n\t        \\POP $Val ; \n\t        \\POP ;\n"

	DPUSH = "Push the value of the given parameter to the input parameter stack. parameter is a prefixed " +
		"name (-creds, -$rate, $user, histfile).\nIf no arguments are given, it pushes the top " +
		"value onto the respective stack for each parameter excluding the pre-defined parameters.\n" +
		"\tExample : \n\t        \\PUSH -$r 9.5 ;\n\t        \\PUSH $Val -$r; \n\t        \\PUSH ;\n"

	DSET = "Set the value of the given parameter to the input value. parameter is a prefixed name " +
		"(-creds, -$rate, $user, histfile).\nIf no arguments are given, list all the existing parameters.\n" +
		"\tExample : \n\t        \\SET -$r 9.5 ;\n\t        \\SET $Val -$r ;\n"

	DSOURCE = "Load input file into shell.\n\tExample : \n\t \\SOURCE temp1.txt ;\n"

	DUNALIAS = "Delete the input alias.\n\tExample : \n\t        \\UNALIAS serverversion;" +
		"\n\t        \\UNALIAS subcommand1 subcommand2 serverversion;\n"

	DUNSET = "Unset the value of the given parameter. parameter is a prefixed name " +
		"(-creds, -$rate, $user, histfile). \n" +
		"\tExample : \n\t        \\UNSET -$r ;\n\t        \\UNSET $Val ;\n"

	DVERSION = "Print the Shell Version.\n\tExample : \n\t        \\VERSION;\n"

	DREDIRECT = "Write output of commands to file (\\REDIRECT filename). " +
		"To return to STDOUT, execute \\REDIRECT OFF .\n" +
		"\tExample : \n\t\t \\REDIRECT temp1.txt ;\n\t\t select * from `beer-sample`;\n\t\t \\REDIRECT OFF;"

	DDEFAULT            = "Fix : Does not exist.\n"
	DREFRESH_CLUSTERMAP = "Refresh the list of query APIs to reflect input service url as cluster. " +
		"\tExample : \n\t\t \\REFRESH_CLUSTER_MAP;"
)

func NewMessage(message string, args ...string) string {
	message = message + " :"
	for _, v := range args {
		message = message + fmt.Sprintf(" %v", v)
	}
	return message
}

func NewShorthandMsg(message string) string {
	return fmt.Sprintf(SHORTHAND+" %v", message)
}

/* Messages are made up of message and args
type message interface {
	Message() string
}

type msg struct {
	InternalMsg string
	Args        []string
}

func (m *msg) Message() string {
	// Fill args into the message.
	return NewMessage(m.InternalMsg, m.Args...)
} */
