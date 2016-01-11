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
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"os"
	"path/filepath"
	"regexp"

	go_n1ql "github.com/couchbase/go_n1ql"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/shell/go_cbq/command"
)

/*
   Command line options provided.
*/

/*
   Option        : -engine or -e
   Args          :  <url to the query service or to the cluster>
   Default value : http://localhost:8091/
   Point to the cluser/query endpoint to connect to.
*/
var ServerFlag string

func init() {
	const (
		defaultserver = "http://localhost:8091/"
		usage         = "URL to the query service/cluster. \n\t\t Default : http://localhost:8091\n\n Usage: cbq \n\t\t Connects to local couchbase instance. Same as: cbq -engine=http://localhost:8091\n\t cbq -engine=http://172.23.107.18:8093 \n\t\t Connects to query node at 172.23.107.18 Port 8093 \n\t cbq -engine=https://my.secure.node.com:8093 \n\t\t Connects to query node at my.secure.node.com:8093 using secure https protocol."
	)
	flag.StringVar(&ServerFlag, "engine", defaultserver, usage)
	flag.StringVar(&ServerFlag, "e", defaultserver, "Shorthand for -engine")
}

/*
   Option        : -no-engine or -ne
   Default value : false
   Enable/Disable startup connection to a query service/cluster endpoint.
*/
var NoQueryService bool

func init() {
	const (
		defaultval = false
		usage      = "Start shell without connecting to a query service/cluster endpoint. \n\t\t Default : false \n\t\t Possible Values : true/false"
	)
	flag.BoolVar(&NoQueryService, "no-engine", defaultval, usage)
	flag.BoolVar(&NoQueryService, "ne", defaultval, " Shorthand for -no-engine")
}

/*
   Option        : -quiet
   Default value : false
   Enable/Disable startup connection message for the shell
*/
var quietFlag bool

func init() {
	const (
		defaultval = false
		usage      = "Enable/Disable startup connection message for the shell \n\t\t Default : false \n\t\t Possible Values : true/false"
	)
	flag.BoolVar(&quietFlag, "quiet", defaultval, usage)
	flag.BoolVar(&quietFlag, "q", defaultval, " Shorthand for -quiet")
}

/*
   Option        : -timeout or -t
   Args          : <timeout value>
   Default value : "0ms"
   Query timeout parameter.
*/

var timeoutFlag string

func init() {
	const (
		defaultval = ""
		usage      = "Query timeout parameter. Units are mandatory. For Example : \"10ms\". \n\t\t Valid Units : ns (nanoseconds), us (microseconds), ms (milliseconds), s (seconds), m (minutes), h (hours) "
	)
	flag.StringVar(&timeoutFlag, "timeout", defaultval, usage)
	flag.StringVar(&timeoutFlag, "t", defaultval, " Shorthand for -timeout")
}

/*
   Option        : -user or -u
   Args          : Login username
   Login credentials for users. The shell will prompt for the password.
*/

var userFlag string

func init() {
	const (
		defaultval = ""
		usage      = "Username \n\t For Example : -u=Administrator"
	)
	flag.StringVar(&userFlag, "user", defaultval, usage)
	flag.StringVar(&userFlag, "u", defaultval, " Shorthand for -user")

}

/*
   Option        : -password or -p
   Args          : password
   Password for user given by -u. If -u is present and we provide -p, then
   do not prompt for the password. Error out if username is not provided.
*/

var pwdFlag string

func init() {
	const (
		defaultval = ""
		usage      = "Password \n\t For Example : -p=password"
	)
	flag.StringVar(&pwdFlag, "password", defaultval, usage)
	flag.StringVar(&pwdFlag, "p", defaultval, " Shorthand for -password")

}

/*
   Option        : -credentials or -c
   Args          : A list of credentials, in the form of user/password objects.
   Login credentials for users as well as SASL Buckets.
*/

var credsFlag string

func init() {
	const (
		defaultval = ""
		usage      = "A list of credentials, in the form user:password. \n\t For Example : Administrator:password, beer-sample:asdasd"
	)
	flag.StringVar(&credsFlag, "credentials", defaultval, usage)
	flag.StringVar(&credsFlag, "c", defaultval, " Shorthand for -credentials")

}

/*
   Option        : -version or -v
   Shell Version
*/

var versionFlag bool

func init() {
	const (
		usage = "Shell Version \n\t Usage: -version"
	)
	flag.BoolVar(&versionFlag, "version", false, usage)
	flag.BoolVar(&versionFlag, "v", false, "Shorthand for -version")

}

/*
   Option        : -script or -s
   Args          : <query>
   Single command mode
*/

var scriptFlag string

func init() {
	const (
		defaultval = ""
		usage      = "Single command mode. Execute input command and exit shell. \n\t For Example : -script=\"select * from system:keyspaces\""
	)
	flag.StringVar(&scriptFlag, "script", defaultval, usage)
	flag.StringVar(&scriptFlag, "s", defaultval, " Shorthand for -script")

}

/*
   Option        : -pretty
   Default value : false
   Pretty print output
*/

var prettyFlag = flag.Bool("pretty", true, "Pretty print the output.")

/*
   Option        : -exit-on-error
   Default value : false
   Exit shell after first error encountered.
*/

var errorExitFlag = flag.Bool("exit-on-error", false, "Exit shell after first error encountered.")

/*
   Option        : -file or -f
   Args          : <filename>
   Input file to run queries from. Exit after the queries are run.
*/

var inputFlag string

func init() {
	const (
		defaultval = ""
		usage      = "File to load commands from. \n\t For Example : -file=temp.txt"
	)
	flag.StringVar(&inputFlag, "file", defaultval, usage)
	flag.StringVar(&inputFlag, "f", defaultval, " Shorthand for -file")

}

/*
   Option        : -ouput or -o
   Args          : <filename>
   Output file to send results of queries to.
*/

var outputFlag string

func init() {
	const (
		defaultval = ""
		usage      = "File to output commands and their results. \n\t For Example : -output=temp.txt"
	)
	flag.StringVar(&outputFlag, "output", defaultval, usage)
	flag.StringVar(&outputFlag, "o", defaultval, " Shorthand for -output")

}

/*
   Option        : -log-file or -l
   Args          : <filename>
   Log commands for session.
*/

var logFlag string

func init() {
	const (
		defaultval = ""
		usage      = "File to log commands into. \n\t For Example : -log-file=temp.txt"
	)
	flag.StringVar(&logFlag, "log-file", defaultval, usage)
	flag.StringVar(&logFlag, "l", defaultval, " Shorthand for -log-file")

}

/* Define credentials as user/pass and convert into
   JSON object credentials
*/

var (
	SERVICE_URL string
	DISCONNECT  bool
	EXIT        bool
)

func main() {

	flag.Parse()
	command.W = os.Stdout

	/* Handle options and what they should do */

	// TODO : Readd ...
	//Taken out so as to connect to both cluster and query service
	//using go_n1ql.
	/*
		if strings.HasPrefix(ServerFlag, "http://") == false {
			ServerFlag = "http://" + ServerFlag
		}

		urlRegex := "^(https?://)[-a-zA-Z0-9+&@#/%?=~_|!:,.;]*[-a-zA-Z0-9+&@#/%=~_|]"
		match, _ := regexp.MatchString(urlRegex, ServerFlag)
		if match == false {
			//TODO Isha : Add error code. Throw invalid url error
			fmt.Println("Invalid url please check" + ServerFlag)
		}


		//-engine
		if strings.HasSuffix(ServerFlag, "/") == false {
			ServerFlag = ServerFlag + "/"
		}
	*/

	/* Check for input url argument
	 */

	args := flag.Args()
	if len(args) > 1 {
		s_err := command.HandleError(errors.CMD_LINE_ARG, "")
		command.PrintError(s_err)
		os.Exit(1)
	} else {
		if len(args) == 1 {
			urlRegex := "^(https?://)[-a-zA-Z0-9+&@#/%?=~_|!:,.;]*[-a-zA-Z0-9+&@#/%=~_|]"
			match, _ := regexp.MatchString(urlRegex, args[0])
			if match == false {
				s_err := command.HandleError(errors.INVALID_URL, args[0])
				command.PrintError(s_err)
			} else {
				ServerFlag = args[0]
			}
		}
	}

	/* -quiet : Display Message only if flag not specified
	 */
	if !quietFlag && NoQueryService == false {
		s := fmt.Sprintln("Connect to " + ServerFlag + ". Type Ctrl-D to exit.\n")
		_, werr := io.WriteString(command.W, s)
		if werr != nil {
			s_err := command.HandleError(errors.WRITER_OUTPUT, werr.Error())
			command.PrintError(s_err)
		}
	}

	/* -version : Display the version of the shell and then exit.
	 */
	if versionFlag == true {
		dummy := []string{}
		cmd := command.Version{}
		cmd.ExecCommand(dummy)
		os.Exit(0)
	}

	/* -user : Accept Admin credentials. Prompt for password and set
	   the n1ql_creds. Append to creds so that user can also define
	   bucket credentials using -credentials if they need to.
	*/
	var creds command.Credentials
	var err error
	var password []byte

	if userFlag != "" {
		//Check if there is a -password option.
		if pwdFlag != "" {
			password = []byte(pwdFlag)
			err = nil
		} else {
			// If no -p option then prompt for the password
			s := fmt.Sprintln("Enter Password: ")
			_, werr := io.WriteString(command.W, s)
			if werr != nil {
				s_err := command.HandleError(errors.WRITER_OUTPUT, werr.Error())
				command.PrintError(s_err)
			}
			password, err = terminal.ReadPassword(0)
		}

		if err == nil {
			if string(password) == "" {
				s_err := command.HandleError(errors.INVALID_PASSWORD, "")
				command.PrintError(s_err)
				os.Exit(1)
			} else {
				creds = append(creds, command.Credential{"user": userFlag, "pass": string(password)})
			}
		} else {
			s_err := command.HandleError(errors.INVALID_PASSWORD, err.Error())
			command.PrintError(s_err)
			os.Exit(1)
		}
	} else {
		// If the -u option isnt specified and the -p option is specified
		// then Invalid Username error.
		if pwdFlag != "" {
			s_err := command.HandleError(errors.INVALID_USERNAME, "")
			command.PrintError(s_err)
			os.Exit(1)
		}
	}

	/* -credentials : Accept credentials to pass to the n1ql endpoint.
	   Ensure that the user inputs credentials in the form a:b.
	   It is important to apend these credentials to those given by
	   -user.
	*/
	if userFlag == "" && credsFlag == "" {
		// No credentials exist. This can still be used to connect to
		// un-authenticated servers.
		// Dont output the statement if we are running in single command
		// mode.
		if scriptFlag == "" {
			_, werr := io.WriteString(command.W, "No Input Credentials. In order to connect to a server with authentication, please provide credentials.\n")

			if werr != nil {
				s_err := command.HandleError(errors.WRITER_OUTPUT, werr.Error())
				command.PrintError(s_err)
			}
		}

	} else if credsFlag != "" {

		creds_ret, err_code, err_string := command.ToCreds(credsFlag)
		if err_code != 0 {
			s_err := command.HandleError(err_code, err_string)
			command.PrintError(s_err)
		}
		for _, v := range creds_ret {
			creds = append(creds, v)
		}

	}
	//Append empty credentials. This is used for cases where one of the buckets
	//is a SASL bucket, and we need to access the other unprotected buckets.
	//CBauth works this way.

	//if credsFlag == "" && userFlag != "" {
	creds = append(creds, command.Credential{"user": "", "pass": ""})
	//}

	/* Add the credentials set by -user and -credentials to the
	   go_n1ql creds parameter.
	*/
	if creds != nil {
		ac, err := json.Marshal(creds)
		if err != nil {
			//Error while Marshalling
			s_err := command.HandleError(errors.JSON_MARSHAL, err.Error())
			command.PrintError(s_err)
			os.Exit(1)
		}
		go_n1ql.SetQueryParams("creds", string(ac))
	}

	if scriptFlag != "" {
		go_n1ql.SetPassthroughMode(true)
		err_code, err_str := execute_input(scriptFlag, os.Stdout)
		if err_code != 0 {
			s_err := command.HandleError(err_code, err_str)
			command.PrintError(s_err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if timeoutFlag != "0ms" {
		go_n1ql.SetQueryParams("timeout", timeoutFlag)
	}

	if inputFlag != "" {
		//Read each line from the file and call execute query

	}

	go_n1ql.SetPassthroughMode(true)
	HandleInteractiveMode(filepath.Base(os.Args[0]))
}
