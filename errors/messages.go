//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package errors

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type YesNoMaybe int

const (
	NO = YesNoMaybe(iota)
	MAYBE
	YES
)

type ErrData struct {
	Code        ErrorCode
	Description string
	Reason      []string
	Action      []string
	AppliesTo   []string
	IsUser      YesNoMaybe
	IsWarning   bool
	symbol      string
}

func (this *ErrData) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{}, 6)
	m["code"] = this.Code
	m["description"] = this.Description
	if len(this.Reason) > 0 {
		a := make([]interface{}, 0, len(this.Reason))
		for i := range this.Reason {
			sa := strings.Split(this.Reason[i], "\n")
			if len(sa) == 1 {
				a = append(a, this.Reason[i])
			} else {
				a = append(a, sa)
			}
		}
		m["reason"] = a
	}
	if len(this.Action) > 0 {
		a := make([]interface{}, 0, len(this.Action))
		for i := range this.Action {
			sa := strings.Split(this.Action[i], "\n")
			if len(sa) == 1 {
				a = append(a, this.Action[i])
			} else {
				a = append(a, sa)
			}
		}
		m["user_action"] = a
	}
	if this.IsWarning {
		m["warning"] = true
	}
	switch this.IsUser {
	case YES:
		m["user_error"] = "Yes"
	case MAYBE:
		m["user_error"] = "Possibly"
	}
	if len(this.AppliesTo) > 0 {
		m["applies_to"] = strings.Join(this.AppliesTo, ", ")
	}
	return json.Marshal(m)
}

func (this *ErrData) Contains(pattern string) bool {
	re, err := regexp.Compile(pattern)
	if err == nil {
		if re.MatchString(fmt.Sprintf("%v", this.Code)) || re.MatchString(this.symbol) || re.MatchString(this.Description) {
			return true
		}
		for i := range this.Reason {
			if re.MatchString(this.Reason[i]) {
				return true
			}
		}
		for i := range this.Action {
			if re.MatchString(this.Action[i]) {
				return true
			}
		}
		if re.MatchString(strings.Join(this.AppliesTo, ", ")) {
			return true
		}
	} else {
		if strings.Contains(fmt.Sprintf("%v", this.Code), pattern) ||
			strings.Contains(this.symbol, pattern) ||
			strings.Contains(this.Description, pattern) {

			return true
		}
		for i := range this.Reason {
			if strings.Contains(this.Reason[i], pattern) {
				return true
			}
		}
		for i := range this.Action {
			if strings.Contains(this.Action[i], pattern) {
				return true
			}
		}
		if strings.Contains(strings.Join(this.AppliesTo, ", "), pattern) {
			return true
		}
	}
	return false
}

func DescribeError(c ErrorCode) *ErrData {
	for i := range errData {
		if errData[i].Code == c {
			return &errData[i]
		}
	}
	return nil
}

func SearchErrors(pattern string) []*ErrData {
	var res []*ErrData
	for i := range errData {
		if errData[i].Contains(pattern) {
			res = append(res, &errData[i])
		}
	}
	sort.Slice(res, func(i int, j int) bool {
		return int(res[i].Code) < int(res[j].Code)
	})
	return res
}

func checkErrorIsUser(c ErrorCode, isUser YesNoMaybe) bool {
	edata := DescribeError(c)
	if edata == nil || edata.IsUser != isUser {
		return false
	}

	return true
}

func IsUserError(c ErrorCode) bool {
	return checkErrorIsUser(c, YES)
}

func IsSystemError(c ErrorCode) bool {
	return checkErrorIsUser(c, NO)
}

var errData = []ErrData{
	{
		Code:        E_SHELL_CONNECTION_REFUSED, // 100
		symbol:      "E_SHELL_CONNECTION_REFUSED",
		Description: "A connection was refused.",
		Reason: []string{
			"A connection the cbq-shell was trying to make was refused by the remote partner.",
		},
		Action: []string{
			"Verify the connection URL and try again.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_UNSUPPORTED_PROTOCOL, // 101
		symbol:      "E_SHELL_UNSUPPORTED_PROTOCOL",
		Description: "Unsupported protocol scheme «scheme»",
		Reason: []string{
			"The protocol scheme in the cbq-shell connection URL is not supported.",
		},
		Action: []string{
			"Correct the URL ensuring only a supported scheme is used.\nSchemes: http, https, couchbase, couchbases",
		},
		IsUser: YES,
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_NO_SUCH_HOST, // 102
		symbol:      "E_SHELL_NO_SUCH_HOST",
		Description: "No such host «host»",
		Reason: []string{
			"The noted host could not be found by the cbq-shell.",
		},
		Action: []string{
			"Correct the host in the connection URL and try again.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_NO_HOST_IN_REQUEST_URL, // 103
		symbol:      "E_SHELL_NO_HOST_IN_REQUEST_URL",
		Description: "No host in request URL",
		Reason: []string{
			"The cbq-shell connection URL does not contain a host name or IP address.",
		},
		Action: []string{
			"Correct the the connection URL and try again.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_UNKNOWN_PORT_TCP, // 104
		symbol:      "E_SHELL_UNKNOWN_PORT_TCP",
		Description: "Unknown port «port»",
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_NO_ROUTE_TO_HOST, // 105
		symbol:      "E_SHELL_NO_ROUTE_TO_HOST",
		Description: "No route to host",
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_UNREACHABLE_NETWORK, // 106
		symbol:      "E_SHELL_UNREACHABLE_NETWORK",
		Description: "Network is unreachable.",
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_NO_CONNECTION, // 107
		symbol:      "E_SHELL_NO_CONNECTION",
		Description: "Not connected to any cluster. Use \\CONNECT command.",
		Reason: []string{
			"A connection in the cbq-shell has not been attempted or has failed.",
		},
		Action: []string{
			"Issue the connect command to connect to a server.",
		},
		IsUser:    YES,
		IsWarning: true,
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_DRIVER_OPEN, // 108
		symbol:      "E_SHELL_DRIVER_OPEN",
		Description: "Failed to open a connection to the server endpoint.",
		Action: []string{
			"Review the details of the error reported.",
			"Contact support.",
		},
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_INVALID_URL, // 109
		symbol:      "E_SHELL_INVALID_URL",
		Description: "Invalid input URL «url»",
		Reason: []string{
			"The URL could not be properly parsed.",
			"The URL contains an invalid host.",
			"The URL contains an invalid port.",
			"A port number was specified with couchbase:// or couchbases:// protocol scheme in the URL.",
		},
		Action: []string{
			"Correct the URL and retry.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_READ_FILE, // 116
		symbol:      "E_SHELL_READ_FILE",
		Description: "Error during file read «details»",
		Reason: []string{
			"The cbq-shell input commands file could not be read.",
			"The cbq-shell command history file could not be read.",
		},
		Action: []string{
			"Review the details reported and take corrective action.",
		},
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_WRITE_FILE, // 117
		symbol:      "E_SHELL_WRITE_FILE",
		Description: "Error during file write «details»",
		Reason: []string{
			"The cbq-shell command history file could not be written to."},
		Action: []string{
			"Review the details reported and take corrective action.",
		},
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_OPEN_FILE, // 118
		symbol:      "E_SHELL_OPEN_FILE",
		Description: "Unable to open file «file»",
		Reason: []string{
			"The cbq-shell input commands file could not be opened.",
			"The size of the cbq-shell input stream could not be determined.",
			"The the cbq-shell input stream was empty.",
			"The cbq-shell command history file could not be opened.",
		},
		Action: []string{
			"Review the details reported and take corrective action.",
		},
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_CLOSE_FILE, // 119
		symbol:      "E_SHELL_CLOSE_FILE",
		Description: "Unable to close file «file»",
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_INVALID_PASSWORD, // 121
		symbol:      "E_SHELL_INVALID_PASSWORD",
		Description: "Invalid password",
		Reason: []string{
			"An empty password was entered at the cbq-shell password prompt.",
			"A password entered at the cbq-shell prompt contains invalid characters.",
			"An error occurred reading from the terminal for the cbq-shell password prompt.",
		},
		Action: []string{
			"Enter only a valid password at the prompt.",
			"review the details reported and take corrective action.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_INVALID_USERNAME, // 122
		symbol:      "E_SHELL_INVALID_USERNAME",
		Description: "Invalid username. ",
		Reason: []string{
			"The cbq-shell command line flags include the password but not the user name.",
		},
		Action: []string{
			"Pass both user and password on the cbq-shell command line.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_MISSING_CREDENTIAL, // 123
		symbol:      "E_SHELL_MISSING_CREDENTIAL",
		Description: "Username missing in -credentials/-c option.",
		Reason: []string{
			"The username could not be found in the credentials cbq-shell option value.",
		},
		Action: []string{
			"Correct the value and retry.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_INVALID_CREDENTIAL, // 124
		symbol:      "E_SHELL_INVALID_CREDENTIAL",
		Description: "Invalid format for credentials. Separate username and password with a colon (':').",
		Reason: []string{
			"The credentials cbq-shell option value was not in the correct format.",
		},
		Action: []string{
			"Correct the value and retry.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_NO_SUCH_COMMAND, // 136
		symbol:      "E_SHELL_NO_SUCH_COMMAND",
		Description: "Command does not exist.",
		Reason: []string{
			"The command entered at the cbq-shell prompt was invalid.",
			"The cbq-shell help command could not find the command specified.",
		},
		Action: []string{
			"Correct the command and retry.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_NO_SUCH_PARAM, // 137
		symbol:      "E_SHELL_NO_SUCH_PARAM",
		Description: "Parameter does not exist",
		Reason: []string{
			"An attempt was made access a parameter that doesn't exist via cbq-shell commands.",
		},
		Action: []string{
			"Verify the parameter is correctly named and the sequence of commands means it is defined when expected.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_TOO_MANY_ARGS, // 138
		symbol:      "E_SHELL_TOO_MANY_ARGS",
		Description: "Too many input arguments to command.",
		Reason: []string{
			"A cbq-shell command was attempted but too many arguments were supplied.",
		},
		Action: []string{
			"Consult the command help facility, correct the command and retry.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_TOO_FEW_ARGS, // 139
		symbol:      "E_SHELL_TOO_FEW_ARGS",
		Description: "Too few input arguments to command.",
		Reason: []string{
			"A cbq-shell command was attempted with insufficient arguments.",
		},
		Action: []string{
			"Consult the command help facility, correct the command and retry.",
		},
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_STACK_EMPTY, // 140
		symbol:      "E_SHELL_STACK_EMPTY",
		Description: "Stack empty.",
		Reason: []string{
			"The cbq-shell value stack was empty and an attempt to pop or set a value was attempted.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_NO_SUCH_ALIAS, // 141
		symbol:      "E_SHELL_NO_SUCH_ALIAS",
		Description: "Alias does not exist «alias»",
		Reason: []string{
			"An attempt was made to list cbq-shell command aliases and no aliases exist.",
		},
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_BATCH_MODE, // 142
		symbol:      "E_SHELL_BATCH_MODE",
		Description: "Error when running in batch mode for Analytics. Incorrect input value",
		Reason: []string{
			"The cbq-shell batch command line option was set to an invalid value.",
		},
		Action: []string{
			"Correct the command line option and retry.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_STRING_WRITE, // 143
		symbol:      "E_SHELL_STRING_WRITE",
		Description: "Cannot write to string buffer.",
		Reason: []string{
			"Operating in batch mode, cbq-shell failed to write the command to the batch file.",
		},
		Action: []string{
			"Review the details reported and take corrective action.",
		},
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_OPERATION_TIMEOUT, // 170
		symbol:      "E_SHELL_OPERATION_TIMEOUT",
		Description: "Operation timed out. Check query service url",
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_ROWS_SCAN, // 171
		symbol:      "E_SHELL_ROWS_SCAN",
		Description: "Retired.",
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_JSON_MARSHAL, // 172
		symbol:      "E_SHELL_JSON_MARSHAL",
		Description: "An error occurred writing data in JSON format.",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_JSON_UNMARSHAL, // 173
		symbol:      "E_SHELL_JSON_UNMARSHAL",
		Description: "An error occurred reading data in JSON format.",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_DRIVER_QUERY_METHOD, // 174
		symbol:      "E_SHELL_DRIVER_QUERY_METHOD",
		Description: "An error occurred in the Query driver.",
		Action: []string{
			"Review the reported error.",
			"Contact support.",
		},
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_WRITER_OUTPUT, // 175
		symbol:      "E_SHELL_WRITER_OUTPUT",
		Description: "Error with io Writer.",
		Reason: []string{
			"The cbq-shell was trying to write output and encountered an error.",
		},
		Action: []string{
			"Review the details reported and take corrective action.",
		},
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_UNBALANCED_QUOTES, // 176
		symbol:      "E_SHELL_UNBALANCED_QUOTES",
		Description: "Unbalanced quotes in the input.",
		Reason: []string{
			"The cbq-shell echo contained an unequal number of double quotation marks.",
		},
		Action: []string{
			"Correct the command and retry.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_ROWS_CLOSE, // 177
		symbol:      "E_SHELL_ROWS_CLOSE",
		Description: "Retired.",
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_CMD_LINE_ARGS, // 178
		symbol:      "E_SHELL_CMD_LINE_ARGS",
		Description: "Place input argument URL at the end, after input flags.",
		Reason: []string{
			"The cbq-shell connection URL was not passed as the engine argument and was not the last argument.",
		},
		Action: []string{
			"Pass the connection URL using the engine argument flag or as the final argument on the command line.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_INVALID_INPUT_ARGUMENTS, // 179
		symbol:      "E_SHELL_INVALID_INPUT_ARGUMENTS",
		Description: "Input Argument format is invalid.",
		Reason: []string{
			"The argument to a cbq-shell command was not a supported format.",
			"The value specified for a cbq-shell predefined value was not valid.",
		},
		Action: []string{
			"Correct the argument or value type.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_ON_REFRESH, // 180
		symbol:      "E_SHELL_ON_REFRESH",
		Description: "Query APIs cannot be initialized from Cluster Map.",
		Reason: []string{
			"The cbq-shell failed to obtain the cluster map from the server.",
		},
		Action: []string{
			"Review the connection URL is correct and still valid.",
		},
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_INVALID_ARGUMENT, // 181
		symbol:      "E_SHELL_INVALID_ARGUMENT",
		Description: "Invalid argument.",
		Reason: []string{
			"An invalid argument was supplied to the cbq-shell redirect command.",
		},
		Action: []string{
			"Consult the command help facility, correct the command and retry.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_INIT_FAILURE, // 182
		symbol:      "E_SHELL_INIT_FAILURE",
		Description: "Terminal set-up failed (check not legacy console)",
		Reason: []string{
			"cbq-shell failed to initialise the terminal for vi emulation mode.",
		},
		Action: []string{
			"(Windows) Ensure cbq-shell is not being run in a legacy console window.",
			"Don't use cbq-shell's vi emulation mode.",
		},
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_INVALID_PROTOCOL, // 183
		symbol:      "E_SHELL_INVALID_PROTOCOL",
		Description: "Invalid protocol. Mixed protocols are not permitted in engine list.",
		Reason: []string{
			"Multiple endpoints were listed in the cbq-shell connection URL with differing protocols.",
		},
		Action: []string{
			"Ensure all endpoints listed are using the same protocol.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SHELL_UNKNOWN, // 199
		symbol:      "E_SHELL_UNKNOWN",
		Description: "A non-specific error occurred in the cbq-shell.",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"cbq-shell",
		},
	},
	{
		Code:        E_SERVICE_READONLY, // 1000
		symbol:      "E_SERVICE_READONLY",
		Description: "The server or request is read-only and cannot accept this write statement.",
		Reason: []string{
			"A request was submitted using the GET method and attempted a statement that modifies data.",
			"A request was received with the ˝readonly˝ parameter set to true and attempted a statement that modifies data.",
			"A PREPARE statement preparing a statement that modifies data was received and the ˝auto_execute˝ was set to true.",
		},
		Action: []string{
			"Use POST to submit write statements and ensure the ˝readonly˝ request parameter is not set or is set to false.",
			"Ensure ˝auto_execute˝ is false when ˝readonly˝ is true (or when using the GET method) and preparing statements " +
				"that modify data.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_HTTP_UNSUPPORTED_METHOD, // 1010
		symbol:      "E_SERVICE_HTTP_UNSUPPORTED_METHOD",
		Description: "Unsupported http method:«METHOD»",
		Reason: []string{
			"The service endpoint supports only GET & POST HTTP methods.\nAll other HTTP methods are not supported.",
		},
		Action: []string{
			"Use a supported method to submit requests.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_NOT_IMPLEMENTED, // 1020
		symbol:      "E_SERVICE_NOT_IMPLEMENTED",
		Description: "«feature» «value» not implemented",
		Reason: []string{
			"The noted feature and value combination is reserved but is not implemented.",
		},
		Action: []string{
			"Use only supported feature and value combinations.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_UNRECOGNIZED_VALUE, // 1030
		symbol:      "E_SERVICE_UNRECOGNIZED_VALUE",
		Description: "Unknown «parameter» value: «value»",
		Reason: []string{
			"The value supplied for the noted parameter is unknown.",
		},
		Action: []string{
			"Ensure the value supplied is a supported value in the required format for the request parameter noted.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_BAD_VALUE, // 1040
		symbol:      "E_SERVICE_BAD_VALUE",
		Description: "Error processing «message»",
		Reason: []string{
			"There was an error in processing as detailed in the message.\n" +
				"e.g. a non-numeric string value passed as the value for a request parameter that is expected to be numeric.",
		},
		Action: []string{
			"Where the error is derived from user controlled data, correct the data.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_MISSING_VALUE, // 1050
		symbol:      "E_SERVICE_MISSING_VALUE",
		Description: "No «parameter» value",
		Reason: []string{
			"A value was not supplied for the required parameter.",
		},
		Action: []string{
			"Provide valid values for all required parameters.\ne.g. ensure a user and password are supplied for all " +
				"requests and a scan_vector is supplied for requests using AT_PLUS consistency level.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_MULTIPLE_VALUES, // 1060
		symbol:      "E_SERVICE_MULTIPLE_VALUES",
		Description: "Multiple values for «parameters»",
		Reason: []string{
			"Multiple values have been supplied for a parameter or two mutually exclusive parameters are both enabled.",
		},
		Action: []string{
			"Ensure all request parameters including named statement parameters, are unique and supplied only once.",
			"Ensure mutually exclusive parameters are not simultaneously enabled.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_UNRECOGNIZED_PARAMETER, // 1065
		symbol:      "E_SERVICE_UNRECOGNIZED_PARAMETER",
		Description: "Unrecognized parameter in request: «parameter»",
		Reason: []string{
			"An unknown request parameter was received.",
		},
		Action: []string{
			"Pass only valid request parameters.",
			"Check parameter names for typographical errors.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_TYPE_MISMATCH, // 1070
		symbol:      "E_SERVICE_TYPE_MISMATCH",
		Description: "«feature» has to be of type «expected»",
		Reason: []string{
			"The value supplied for «feature» was not of the expected type.",
		},
		Action: []string{
			"Correct the value and re-submit the request.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_TIMEOUT, // 1080
		symbol:      "E_SERVICE_TIMEOUT",
		Description: "Timeout «duration» exceeded",
		Reason: []string{
			"The specified request time-out was reached.",
		},
		Action: []string{
			"Check the statement is correctly constructed and using the expected plan.",
			"Revise the time-out upward to accommodate the statement.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_INVALID_VALUE, // 1090
		symbol:      "E_SERVICE_INVALID_VALUE",
		Description: "«parameter» = «value» is invalid. «message»",
		Reason: []string{
			"The named parameter's value was invalid for the reason noted in the message.",
		},
		Action: []string{
			"Set the parameter to a valid value for the request.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_INVALID_JSON, // 1100
		symbol:      "E_SERVICE_INVALID_JSON",
		Description: "Invalid JSON in results",
		Reason: []string{
			"An error occurred whilst writing results to the output stream.",
		},
		Action: []string{
			"Please contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_CLIENTID, // 1110
		symbol:      "E_SERVICE_CLIENTID",
		Description: "forbidden character (\\\\ or \\\") in client_context_id",
		Reason: []string{
			"The request parameter client_context_id contains one or more of the noted invalid characters.",
		},
		Action: []string{
			"Revise the value for client_context_id.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_MEDIA_TYPE, // 1120
		symbol:      "E_SERVICE_MEDIA_TYPE",
		Description: "Unsupported media type: «mediaType»",
		Reason: []string{
			"The HTTP request header field ˝Accept˝ was not set to a supported value.",
		},
		Action: []string{
			"Change the header field to ˝*/*˝, ˝application/json˝ or ˝application/xml˝.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_HTTP_REQ, // 1130
		symbol:      "E_SERVICE_HTTP_REQ",
		Description: "Request «id» is not a http request",
		Reason: []string{
			"The request identified by «id» does not exist.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_SCAN_VECTOR_BAD_LENGTH, // 1140
		symbol:      "E_SERVICE_SCAN_VECTOR_BAD_LENGTH",
		Description: "Array «scan_entry» should be of length 2",
		Reason: []string{
			"An invalid scan vector array element was found.",
		},
		Action: []string{
			"Correct the scan vector.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_SCAN_VECTOR_BAD_SEQUENCE_NUMBER, // 1150
		symbol:      "E_SERVICE_SCAN_VECTOR_BAD_SEQUENCE_NUMBER",
		Description: "Bad sequence number «seqno». Expected an unsigned 64-bit integer.",
		Reason: []string{
			"An entry in the scan vector contained a sequence number that was not an unsigned 64-bit integer.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_SCAN_VECTOR_BADUUID, // 1155
		symbol:      "E_SERVICE_SCAN_VECTOR_BADUUID",
		Description: "Bad UUID «vbucket_uuid». Expected a string.",
		Reason: []string{
			"An entry in the scan vector contained a v-bucket UUID that was not a string value.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_DECODE_NIL, // 1160
		symbol:      "E_SERVICE_DECODE_NIL",
		Description: "Failed to decode nil value.",
		Reason: []string{
			"A request requiring a body did not include one.",
		},
		Action: []string{
			"Resubmit a valid request.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_HTTP_METHOD, // 1170
		symbol:      "E_SERVICE_HTTP_METHOD",
		Description: "Unsupported method «method»",
		Reason: []string{
			"The HTTP request method noted is not supported by the endpoint.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_SHUTTING_DOWN, // 1180
		symbol:      "E_SERVICE_SHUTTING_DOWN",
		Description: "Indicates the service on the node is in the process of shutting down.",
		Reason: []string{
			"A topology change was in the process of removing the node.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_SHUT_DOWN, // 1181
		symbol:      "E_SERVICE_SHUT_DOWN",
		Description: "Indicates the service on the node has been shut down and is waiting to be terminated.",
		Reason: []string{
			"A topology change was in the process of removing the node.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_UNAVAILABLE, // 1182
		symbol:      "E_SERVICE_UNAVAILABLE",
		Description: "Service cannot handle requests",
		Reason: []string{
			"A ping request has determined the service was not healthy.",
		},
		Action: []string{
			"Examine the diagnostic logs to ascertain the reason for this state.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_USER_REQUEST_EXCEEDED, // 1191
		symbol:      "E_SERVICE_USER_REQUEST_EXCEEDED",
		Description: "User has more requests running than allowed",
		Reason: []string{
			"Currently unused.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_USER_REQUEST_RATE_EXCEEDED, // 1192
		symbol:      "E_SERVICE_USER_REQUEST_RATE_EXCEEDED",
		Description: "User has exceeded request rate limit",
		Reason: []string{
			"Currently unused.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_USER_REQUEST_SIZE_EXCEEDED, // 1193
		symbol:      "E_SERVICE_USER_REQUEST_SIZE_EXCEEDED",
		Description: "User has exceeded input network traffic limit",
		Reason: []string{
			"Currently unused.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_USER_RESULT_SIZE_EXCEEDED, // 1194
		symbol:      "E_SERVICE_USER_RESULT_SIZE_EXCEEDED",
		Description: "User has exceeded results size limit",
		Reason: []string{
			"Currently unused.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_REQUEST_ERROR_LIMIT, // 1195
		symbol:      "E_REQUEST_ERROR_LIMIT",
		Description: "Request execution aborted as the number of errors raised has reached the maximum permitted.",
		Reason: []string{
			"The number of errors raised has reached the limit.",
		},
		Action: []string{
			"Consult the errors to ensure the statement is operating as expected.",
			"Revise the ˝error_limit˝ request parameter as necessary.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_TENANT_THROTTLED, // 1196
		symbol:      "E_SERVICE_TENANT_THROTTLED",
		Description: "Request has been declined with «reason»",
		Reason: []string{
			"The request breached a limit for the tenant.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_TENANT_MISSING, // 1197
		symbol:      "E_SERVICE_TENANT_MISSING",
		Description: "Request does not have a valid tenant",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_TENANT_NOT_AUTHORIZED, // 1198
		symbol:      "E_SERVICE_TENANT_NOT_AUTHORIZED",
		Description: "Request is not authorized for tenant «tenant»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_TENANT_REJECTED, // 1199
		symbol:      "E_SERVICE_TENANT_REJECTED",
		Description: "Request rejected due to limiting or throttling. «retry»",
		Action: []string{
			"Retry the request in accordance with «retry».",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_TENANT_NOT_FOUND, // 1200
		symbol:      "E_SERVICE_TENANT_NOT_FOUND",
		Description: "Tenant not found «tenant»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_REQUEST_QUEUE_FULL, // 1201
		symbol:      "E_SERVICE_REQUEST_QUEUE_FULL",
		Description: "Request queue full",
		Reason: []string{
			"The request queue has reached its limit",
		},
		Action: []string{
			"Verify the server is processing requests.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_NO_CLIENT, // 1202
		symbol:      "E_SERVICE_NO_CLIENT",
		Description: "Client disconnected",
		Reason: []string{
			"The server aborts servicing a request when it detects the client has closed its connection.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SERVICE_SLOW_CLIENT, // 1203
		symbol:      "E_SERVICE_SLOW_CLIENT",
		Description: "Slow/stalled client write timed out",
		Reason: []string{
			"A write to the request output stream timed out.  Individual writes that make up the response must not block " +
				"indefinitely, which typically occurs when the client isn't reading the response stream.",
		},
		Action: []string{
			"Check the application is reading response stream fast enough to avoid blocking writes.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_CONNECTION, // 2000
		symbol:      "E_ADMIN_CONNECTION",
		Description: "Error connecting to «what»",
		Reason: []string{
			"The server encountered an error when establishing a connection to «what».",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_START, // 2001
		symbol:      "E_ADMIN_START",
		Description: "Error accounting manager: «reason».",
		Reason: []string{
			"«reason» prevented correct start-up of the service statistics monitor.",
		},
		Action: []string{
			"Contact support",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_INVALIDURL, // 2010
		symbol:      "E_ADMIN_INVALIDURL",
		Description: "Invalid «component» URL: «URL»",
		Reason: []string{
			"An invalid URL was encountered for the noted component.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_DECODING, // 2020
		symbol:      "E_ADMIN_DECODING",
		Description: "Error in JSON decoding",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_ENCODING, // 2030
		symbol:      "E_ADMIN_ENCODING",
		Description: "Error in JSON encoding",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_UNKNOWN_SETTING, // 2031
		symbol:      "E_ADMIN_UNKNOWN_SETTING",
		Description: "Unknown setting: «setting»",
		Reason: []string{
			"An unknown setting was supplied in a request to the settings rest endpoint.",
		},
		Action: []string{
			"Provide only valid settings in the request.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_SETTING_TYPE, // 2032
		symbol:      "E_ADMIN_SETTING_TYPE",
		Description: "Incorrect value «value» for setting: «name»",
		Reason: []string{
			"The value provided for the noted setting was not of the correct type.",
		},
		Action: []string{
			"Correct the value and re-submit the request.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_GET_CLUSTER, // 2040
		symbol:      "E_ADMIN_GET_CLUSTER",
		Description: "Error retrieving cluster «message»",
		Action: []string{
			"Contact support.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_ADD_CLUSTER, // 2050
		symbol:      "E_ADMIN_ADD_CLUSTER",
		Description: "Error adding cluster «message»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_REMOVE_CLUSTER, // 2060
		symbol:      "E_ADMIN_REMOVE_CLUSTER",
		Description: "Error removing cluster «message»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_GET_NODE, // 2070
		symbol:      "E_ADMIN_GET_NODE",
		Description: "Error retrieving node «message»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_NO_NODE, // 2080
		symbol:      "E_ADMIN_NO_NODE",
		Description: "No such node «message»",
		Action: []string{
			"Contact support",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_ADD_NODE, // 2090
		symbol:      "E_ADMIN_ADD_NODE",
		Description: "Error adding node «message»",
		Action: []string{
			"Contact support",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_REMOVE_NODE, // 2100
		symbol:      "E_ADMIN_REMOVE_NODE",
		Description: "Error removing node «message»",
		Action: []string{
			"Contact support",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_MAKE_METRIC, // 2110
		symbol:      "E_ADMIN_MAKE_METRIC",
		Description: "Error creating metric «message»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_AUTH, // 2120
		symbol:      "E_ADMIN_AUTH",
		Description: "Error authorizing against cluster «message»",
		Reason: []string{
			"Request received without suitable credentials.",
			"Failure to authenticate with given credentials.",
			"Authenticated user lacks required privileges.",
		},
		Action: []string{
			"Review the embedded «message» information for more detail on why the operation failed.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_ENDPOINT, // 2130
		symbol:      "E_ADMIN_ENDPOINT",
		Description: "The admin endpoint encountered an error.",
		Action: []string{
			"Contact support",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_SSL_NOT_ENABLED, // 2140
		symbol:      "E_ADMIN_SSL_NOT_ENABLED",
		Description: "server is not ssl enabled",
		Reason: []string{
			"An attempt has been made to update the SSL certificate but the server does not have SSL enabled.",
		},
		Action: []string{
			"Review the server configuration.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_CREDS, // 2150
		symbol:      "E_ADMIN_CREDS",
		Description: "Not a proper creds JSON array of user/pass structures: «creds»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_COMPLETED_QUALIFIER_EXISTS, // 2160
		symbol:      "E_COMPLETED_QUALIFIER_EXISTS",
		Description: "Completed requests qualifier already set: «qualifier»",
		Action: []string{
			"Define a different qualifier or update the existing one.",
			"Refer to the ˝logging qualifiers section of the documentation.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_COMPLETED_QUALIFIER_UNKNOWN, // 2170
		symbol:      "E_COMPLETED_QUALIFIER_UNKNOWN",
		Description: "Completed requests qualifier unknown: «qualifier»",
		Reason: []string{
			"An attempt was made to add a qualifier that is not a known.",
		},
		Action: []string{
			"Check the qualifier specified is valid.",
			"Refer to the ˝logging qualifiers section of the documentation.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_COMPLETED_QUALIFIER_NOT_FOUND, // 2180
		symbol:      "E_COMPLETED_QUALIFIER_NOT_FOUND",
		Description: "Completed requests qualifier not set: «qualifier»",
		Reason: []string{
			"An attempt was made to access a qualifier that was not set.",
		},
		Action: []string{
			"Ensure the intended qualifier has been set.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_COMPLETED_QUALIFIER_NOT_UNIQUE, // 2190
		symbol:      "E_COMPLETED_QUALIFIER_NOT_UNIQUE",
		Description: "Non-unique completed requests qualifier «qualifier» cannot be updated",
		Reason: []string{
			"A attempt was made to update a qualifier that isn't unique.",
		},
		Action: []string{
			"Only attempt to update unique qualifiers. Non-unique qualifiers may only be added/removed.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_COMPLETED_QUALIFIER_INVALID_ARGUMENT, // 2200
		symbol:      "E_COMPLETED_QUALIFIER_INVALID_ARGUMENT",
		Description: "Completed requests qualifier «qualifier» cannot accept argument «value»",
		Reason: []string{
			"The data type of the «value» was incompatible with the qualifier.",
		},
		Action: []string{
			"Correct the value for the qualifier and re-submit the request.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_COMPLETED_BAD_MAX_SIZE, // 2201
		symbol:      "E_COMPLETED_BAD_MAX_SIZE",
		Description: "Completed requests maximum plan size («size») is invalid.",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_BAD_SERVICE_PORT, // 2210
		symbol:      "E_ADMIN_BAD_SERVICE_PORT",
		Description: "Invalid service port: «port»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_BODY, // 2220
		symbol:      "E_ADMIN_BODY",
		Description: "Error getting request body",
		Reason: []string{
			"A prepareds endpoint PUT request was received but the body was empty or could not be read.",
		},
		Action: []string{
			"contact support",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_FFDC, // 2230
		symbol:      "E_ADMIN_FFDC",
		Description: "FFDC invocation failed.",
		Reason: []string{
			"An error occurred with a manual First Failure Data Capture (FFDC) invocation.",
		},
		Action: []string{
			"Wait until the specified reported minimum time before attempting a further invocation.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADMIN_LOG, // 2240
		symbol:      "E_ADMIN_LOG",
		Description: "Error accessing log",
		Reason: []string{
			"A request was made to the diagnostic log endpoint and there was a error accessing the file.",
		},
		Action: []string{
			"Ensure the diagnostic log file being accessed exists for the duration of the request.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_AWR_START, // 2500
		symbol:      "E_AWR_START",
		Description: "Failed to start workload reporting",
		Reason: []string{
			"An error occurred when starting request capture for workload reporting.",
		},
		Action: []string{
			"Review the error and correct any configuration issues.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_AWR_SETTING, // 2501
		symbol:      "E_AWR_SETTING",
		Description: "Invalid value «value» for workload setting «setting»",
		Reason: []string{
			"The value provided for the setting is invalid.",
		},
		Action: []string{
			"Consult the documentation for the valid values for the workload settings." +
				" Review the configuration and correct the value.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_AWR_CONFIG, // 2502
		symbol:      "E_AWR_CONFIG",
		Description: "Error processing workload configuration",
		Reason: []string{
			"The value provided for AWR configuration is invalid.",
		},
		Action: []string{
			"Review the configuration and correct the value.",
			"The value must be a valid object or a string encoding a valid JSON object.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_AWR_DISTRIB, // 2503
		symbol:      "E_AWR_DISTRIB",
		Description: "Error distributing workload settings",
		Reason: []string{
			"An error occurred distributing the workload settings to other Query nodes.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_PARSE_SYNTAX, // 3000
		symbol:      "E_PARSE_SYNTAX",
		Description: "Indicates a syntax error occurred during statement parsing.",
		Action: []string{
			"Correct the syntax and re-submit the request.  Look for incorrectly spelled keywords, use of reserved words " +
				"as identifiers, incorrect or omitted punctuation and delimiters or invalid grammar.",
			"If using the cbq-shell, the \\syntax command may help with grammar issues.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ERROR_CONTEXT, // 3005
		symbol:      "E_ERROR_CONTEXT",
		Description: "Details the location in the statement text of errors encountered during parsing.",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_PARSE_INVALID_ESCAPE_SEQUENCE, // 3006
		symbol:      "E_PARSE_INVALID_ESCAPE_SEQUENCE",
		Description: "invalid escape sequence",
		Reason: []string{
			"An invalid escape sequence was encountered whilst parsing a string value.  Escape sequences are introduced " +
				"with a backslash (Reverse Solidus, U+005C) and literal backslashes must be escaped.",
		},
		Action: []string{
			"Valid escape sequences are: \\b, \\f, \\n, \\r, \\t, \\/, \\\\, \\\", \\`, \\u#### (where #### is a Unicode " +
				"symbol number in hexadecimal).",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_PARSE_INVALID_STRING, // 3007
		symbol:      "E_PARSE_INVALID_STRING",
		Description: "invalid string",
		Reason: []string{
			"An opening quotation mark defining a string was encountered without any further characters.",
		},
		Action: []string{
			"Correctly delimit all string values in statements.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_PARSE_MISSING_CLOSING_QUOTE, // 3008
		symbol:      "E_PARSE_MISSING_CLOSING_QUOTE",
		Description: "missing closing quote",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_PARSE_UNESCAPED_EMBEDDED_QUOTE, // 3009
		symbol:      "E_PARSE_UNESCAPED_EMBEDDED_QUOTE",
		Description: "unescaped embedded quote",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_AMBIGUOUS_REFERENCE, // 3080
		symbol:      "E_AMBIGUOUS_REFERENCE",
		Description: "Ambiguous reference to field «field»",
		Reason: []string{
			"A field reference in the statement was not fully qualified and there were multiple keyspaces it could have " +
				"referred to.\ne.g. SELECT a FROM b, c WHERE ...",
		},
		Action: []string{
			"Fully qualify references when the potential for ambiguity exists.\ne.g. SELECT b.a FROM b,c WHERE ... ",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DUPLICATE_VARIABLE, // 3081
		symbol:      "E_DUPLICATE_VARIABLE",
		Description: "Duplicate variable: «identifier» already in the scope «context»",
		Reason: []string{
			"There was a non-unique binding name in a LET or WITH clause.",
		},
		Action: []string{
			"Use unique names for all bindings in a statement.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FORMALIZER_INTERNAL, // 3082
		symbol:      "E_FORMALIZER_INTERNAL",
		Description: "Formalizer internal error: «details»",
		Reason: []string{
			"A statement included a correlated reference that was not permitted.",
		},
		Action: []string{
			"If encountered with a existing prepared statement, re-prepare the statement.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_PARSE_INVALID_INPUT, // 3083
		symbol:      "E_PARSE_INVALID_INPUT",
		Description: "Invalid input.",
		Reason: []string{
			"Invalid input was submitted, either a statement or expression, depending on context.",
		},
		Action: []string{
			"Submit only valid SQL++ statements or expressions.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SEMANTICS, // 3100
		symbol:      "E_SEMANTICS",
		Description: "A semantic error is present in the statement.",
		Reason: []string{
			"The statement includes portions that violate semantic constraints.",
		},
		Action: []string{
			"The cause will contain more detail on the violation; revise the statement and re-submit.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SEMANTICS_INTERNAL, // 3101
		symbol:      "E_SEMANTICS_INTERNAL",
		Description: "Semantic error: «what»",
		Reason: []string{
			"An internal error occurred during semantics check for the query.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_JOIN_NEST_NO_JOIN_HINT, // 3110
		symbol:      "E_JOIN_NEST_NO_JOIN_HINT",
		Description: "«op» on «alias» cannot have join hint (USE HASH or USE NL)",
		Reason: []string{
			"Join type hints are only supported for ANSI join and nest operations.",
		},
		Action: []string{
			"Review the statement and revise the operation or omit the hints.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_JOIN_NEST_NO_USE_KEYS, // 3120
		symbol:      "E_JOIN_NEST_NO_USE_KEYS",
		Description: "«operation» on «alias» cannot have USE KEYS.",
		Reason: []string{
			"USE KEYS is not supported in this context.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_JOIN_NEST_NO_USE_INDEX, // 3130
		symbol:      "E_JOIN_NEST_NO_USE_INDEX",
		Description: "«operation» on «alias» cannot have USE INDEX.",
		Reason: []string{
			"USE INDEX is not supported in this context.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_MERGE_INSERT_NO_KEY, // 3150
		symbol:      "E_MERGE_INSERT_NO_KEY",
		Description: "MERGE with ON KEY clause cannot have document key specification in INSERT action.",
		Reason: []string{
			"A lookup merge statement specified a document key.\n" +
				"e.g. MERGE INTO default USING [{},{}] AS source ON KEY 'aaa' WHEN NOT MATCHED THEN INSERT ('key',{})",
		},
		Action: []string{
			"Refer to the documentation for lookup merge statements.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_MERGE_INSERT_MISSING_KEY, // 3160
		symbol:      "E_MERGE_INSERT_MISSING_KEY",
		Description: "MERGE with ON clause must have document key specification in INSERT action",
		Reason: []string{
			"An ANSI merge statement did not include the document key specification.\n" +
				"e.g. MERGE INTO default USING [{},{}] AS source ON default.id IS VALUED WHEN NOT MATCHED THEN INSERT ({})",
		},
		Action: []string{
			"Refer to the documentation for ANSI merge statements.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_MERGE_MISSING_SOURCE, // 3170
		symbol:      "E_MERGE_MISSING_SOURCE",
		Description: "MERGE is missing source.",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_MERGE_NO_INDEX_HINT, // 3180
		symbol:      "E_MERGE_NO_INDEX_HINT",
		Description: "MERGE with ON KEY clause cannot have USE INDEX hint specified on target.",
		Reason: []string{
			"The USE INDEX hint is not supported with lookup merge statement targets.\n" +
				"e.g. MERGE INTO default USE INDEX (ix) USING [{},{}] AS source ON KEY 'aaa' WHEN NOT MATCHED THEN INSERT ({})",
		},
		Action: []string{
			"Refer to the documentation for lookup merge statements.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_MERGE_NO_JOIN_HINT, // 3190
		symbol:      "E_MERGE_NO_JOIN_HINT",
		Description: "MERGE with ON KEY clause cannot have join hint specified on source.",
		Reason: []string{
			"The USE INDEX hint is not supported with lookup merge statement source.",
		},
		Action: []string{
			"Refer to the documentation for lookup merge statements.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_MIXED_JOIN, // 3200
		symbol:      "E_MIXED_JOIN",
		Description: "Cannot mix «op1» on «alias1» with «op2» on «alias2».",
		Reason: []string{
			"Mixing ANSI and non-ANSI joins.\ne.g. SELECT * FROM default d1 JOIN default d2 ON d1.id = d2.id " +
				"JOIN default d3 ON KEYS 'aaa'",
			"Mixing ANSI and non-ANSI NEST statements.",
		},
		Action: []string{
			"Revise the statement to use only one type of operation.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_WINDOW_SEMANTIC, // 3220
		symbol:      "E_WINDOW_SEMANTIC",
		Description: "«name» window function «clause» «reason»",
		Reason: []string{
			"A violation of the window function semantic restrictions was present in the statement.",
		},
		Action: []string{
			"Revise the statement to remove the violation.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ENTERPRISE_FEATURE, // 3230
		symbol:      "E_ENTERPRISE_FEATURE",
		Description: "«feature» is an enterprise level feature.",
		Reason: []string{
			"An attempt was made to use the noted feature that is only available in the Enterprise Edition of the product.",
		},
		Action: []string{
			"Consult the documentation for the feature you're trying to use.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Community Edition",
		},
	},
	{
		Code:        E_ADVISE_UNSUPPORTED_STMT, // 3250
		symbol:      "E_ADVISE_UNSUPPORTED_STMT",
		Description: "Advise supports SELECT, MERGE, UPDATE and DELETE statements only.",
		Reason: []string{
			"An attempt was made to run advise on an unsupported statement.",
		},
		Action: []string{
			"Refer to the documentation for ADVISE.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADVISOR_PROJ_ONLY, // 3255
		symbol:      "E_ADVISOR_PROJ_ONLY",
		Description: "Advisor function is only allowed in projection clause",
		Reason: []string{
			"An attempt was made to use the ADVISOR() function out side of a select statement's projection.",
		},
		Action: []string{
			"Refer to the documentation for ADVISE.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADVISOR_NO_FROM, // 3256
		symbol:      "E_ADVISOR_NO_FROM",
		Description: "FROM clause is not allowed when Advisor function is present in projection clause.",
		Reason: []string{
			"An attempt was made to use the advisor function on the results from a keyspace fetch.",
		},
		Action: []string{
			"Refer to the documentation for ADVISE.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_MHDP_ONLY_FEATURE, // 3260
		symbol:      "E_MHDP_ONLY_FEATURE",
		Description: "«what» is only supported in Developer Preview Mode.",
		IsUser:      YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_MISSING_USE_KEYS, // 3261
		symbol:      "E_MISSING_USE_KEYS",
		Description: "«type» term must have USE KEYS",
		Reason: []string{
			"A keyspace in the statement was not an explicit path and there was no USE KEYS clause.",
		},
		Action: []string{
			"Revise the statement to include an explicit path or a USE KEYS clause as appropriate.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_HAS_USE_INDEXES, // 3262
		symbol:      "E_HAS_USE_INDEXES",
		Description: "«type» term should not have USE INDEX",
		Reason: []string{
			"A keyspace in the statement was not an explicit path and there was a USE INDEX clause.",
		},
		Action: []string{
			"Revise the statement to include an explicit path or remove the USE INDEX clause as appropriate.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_UPDATE_STAT_INVALID_INDEX_TYPE, // 3270
		symbol:      "E_UPDATE_STAT_INVALID_INDEX_TYPE",
		Description: "UPDATE STATISTICS (ANALYZE) supports GSI indexes only for INDEX option.",
		Reason: []string{
			"An attempt was made to run UPDATE STATISTICS for a non-GSI index.",
		},
		Action: []string{
			"Do not run UPDATE STATISTICS on a non-GSI index.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_UPDATE_STAT_INDEX_ALL_COLLECTION_ONLY, // 3271
		symbol:      "E_UPDATE_STAT_INDEX_ALL_COLLECTION_ONLY",
		Description: "INDEX ALL option for UPDATE STATISTICS (ANALYZE) can only be used for a collection.",
		Reason: []string{
			"A statistics update was attempted using the INDEX ALL clause on a bucket.",
		},
		Action: []string{
			"Do not run UPDATE STATISTICS with INDEX ALL on buckets.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_UPDATE_STAT_SELF_NOTALLOWED, // 3272
		symbol:      "E_UPDATE_STAT_SELF_NOTALLOWED",
		Description: "UPDATE STATISTICS of 'self' is not allowed",
		Reason: []string{
			"A statistics update was attempted on an index expression including ˝self˝.",
		},
		Action: []string{
			"Revise the index expression to not include ˝self˝.",
			"Refer to the documentation for UPDATE STATISTICS.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CREATE_INDEX_NOT_INDEXABLE, // 3280
		symbol:      "E_CREATE_INDEX_NOT_INDEXABLE",
		Description: "«index key expression» is not indexable",
		Reason: []string{
			"An expression in the index definition was not indexable (e.g. a constant).",
		},
		Action: []string{
			"Revise the definition to include only indexable expressions.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CREATE_INDEX_ATTRIBUTE_MISSING, // 3281
		symbol:      "E_CREATE_INDEX_ATTRIBUTE_MISSING",
		Description: "«message» «location» MISSING attribute not allowed (Only allowed with gsi leading key).",
		Reason: []string{
			"An attempt was made to create a GSI index and INCLUDE MISSING was specified for a non-leading key.",
			"An attempt was made to create a non-GSI index and INCLUDE MISSING was specified.",
			"An attempt was made to create an index using FLATTEN_KEYS, and INCLUDE MISSING was specified for an argument " +
				"other than the initial argument.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CREATE_INDEX_ATTRIBUTE, // 3282
		symbol:      "E_CREATE_INDEX_ATTRIBUTE",
		Description: "Attributes are not allowed on «details» «location» of flatten_keys.",
		Reason: []string{
			"Attributes specified for FLATTEN_KEYS.",
		},
		Action: []string{
			"Revise the statement to remove the attributes on the FLATTEN_KEYS() expression.",
			"NOTE: Arguments passed to FLATTEN_KEYS() may have attributes.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FLATTEN_KEYS, // 3283
		symbol:      "E_FLATTEN_KEYS",
		Description: "«flatten keys expression» «location» is not allowed in this context",
		Reason: []string{
			"FLATTEN_KEYS specified outside of CREATE INDEX or UPDATE STATISTICS or was surrounded by a function.",
		},
		Action: []string{
			"Refer to the documentation for flatten keys.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ALL_DISTINCT_NOT_ALLOWED, // 3284
		symbol:      "E_ALL_DISTINCT_NOT_ALLOWED",
		Description: "ALL/DISTINCT is not allowed in «expression» «location»",
		Reason: []string{
			"ALL and/or DISTINCT used in an invalid location.",
		},
		Action: []string{
			"Revise the statement.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CREATE_INDEX_SELF_NOTALLOWED, // 3285
		symbol:      "E_CREATE_INDEX_SELF_NOTALLOWED",
		Description: "Index of «expression» «location» is not allowed as a index key",
		Reason: []string{
			"An attempt to use SELF as an index key was made.",
		},
		Action: []string{
			"Remove SELF from the the index definition.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INDEX_NOT_ALLOWED, // 3286
		symbol:      "E_INDEX_NOT_ALLOWED",
		Description: "PRIMARY INDEX is not allowed using FTS",
		Reason: []string{
			"FTS was specified as the index provider for a primary index.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_JOIN_HINT_FIRST_FROM_TERM, // 3290
		symbol:      "E_JOIN_HINT_FIRST_FROM_TERM",
		Description: "Join hint (USE HASH or USE NL) cannot be specified on the first from term «term»",
		Reason: []string{
			"A join hint was specified on the first term of a join.",
		},
		Action: []string{
			"Revise the statement to remove the hint on the first join term.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ORDER_BY_VALIDATION_FAIL, // 3291
		symbol:      "E_ORDER_BY_VALIDATION_FAIL",
		Description: "«what» «expression» is not a valid constant, named, positional or function parameter.",
		Reason: []string{
			"The ORDER BY direction or NULLS position was not a valid constant, named, positional or function parameter.",
		},
		Action: []string{
			"Revise the ORDER BY direction or NULLS position to be a valid constant, named, positional or function parameter.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_RECURSIVE_WITH_SEMANTIC, // 3300
		symbol:      "E_RECURSIVE_WITH_SEMANTIC",
		Description: "recursive_with semantics: «cause»",
		Reason: []string{
			"The statement specifies restricted syntax in a recursive common table expression definition.",
		},
		Action: []string{
			"Revise the statement removing the restricted syntax.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ANCHOR_RECURSIVE_REF, // 3301
		symbol:      "E_ANCHOR_RECURSIVE_REF",
		Description: "Anchor Clause cannot have recursive reference in FROM Expression : «alias»",
		Reason: []string{
			"The statement includes a recursive common table expression that references itself in the first branch of the " +
				"defining UNION.\ne.g. WITH RECURSIVE rcte AS (SELECT * FROM rcte UNION SELECT * FROM rcte) SELECT 1",
		},
		Action: []string{
			"Correct the recursive common table expression definition.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_MORE_THAN_ONE_RECURSIVE_REF, // 3302
		symbol:      "E_MORE_THAN_ONE_RECURSIVE_REF",
		Description: "Recursive reference «alias» must not appear more than once in the FROM clause",
		Reason: []string{
			"The statement includes a recursive common table expression that references itself more than once in the " +
				"recursive branch of the defining UNION.\n" +
				"e.g. WITH RECURSIVE rcte AS (SELECT * FROM default UNION SELECT * FROM rcte, rcte) SELECT 1",
		},
		Action: []string{
			"Revise the statement removing the duplicate reference.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CONFIG_INVALID_OPTION, // 3303
		symbol:      "E_CONFIG_INVALID_OPTION",
		Description: "Invalid config option «option»",
		Reason: []string{
			"The statement includes a recursive common table expression with an OPTIONS clause object containing an invalid " +
				"option.\ne.g. WITH RECURSIVE rcte AS (SELECT * FROM default UNION SELECT * FROM rcte) OPTIONS {'bad':1} SELECT 1",
		},
		Action: []string{
			"Refer to the documentation for permitted options.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_RECURSION_UNSUPPORTED, // 3304
		symbol:      "E_RECURSION_UNSUPPORTED",
		Description: "recursive_with_unsupported: «reason»",
		Reason: []string{
			"A recursive common table expression was specified in a NEST clause.",
			"A recursive common table expression was specified in an UNNEST clause.",
			"A recursive common table expression was specified in an OUTER JOIN clause.",
		},
		Action: []string{
			"Revise the statement to remove the unsupported reference.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_RECURSIVE_IMPLICIT_DOC_LIMIT, // 3305
		symbol:      "E_RECURSIVE_IMPLICIT_DOC_LIMIT",
		Description: "Recursive WITH «alias» limited to «limit» documents as no explicit document count limit or memory quota set",
		Reason: []string{
			"The request without a memory quota set contained a recursive common table expression without an explicit document " +
				"limit that produced more results than the implicit limit and was stopped.",
		},
		Action: []string{
			"Review the statement and its control of the recursion.\nUse a memory quota to guard against runaway recursion or " +
				"specify an explicit document limit for the common table expression.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_RECURSIVE_IMPLICIT_DEPTH_LIMIT, // 3306
		symbol:      "E_RECURSIVE_IMPLICIT_DEPTH_LIMIT",
		Description: "Recursive WITH «alias» stopped at «depth» level as no explicit level limit or memory quota set",
		Reason: []string{
			"The request without a memory quota set contained a recursive common table expression without an explicit level " +
				"limit exceeded the implicit limit and was stopped.",
		},
		Action: []string{
			"Review the statement and its control of the recursion.\nUse a memory quota to guard against runaway recursion or " +
				"specify an explicit level limit for the common table expression.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CYCLE_FIELDS_VALIDATION_FAILED, // 3307
		symbol:      "E_CYCLE_FIELDS_VALIDATION_FAILED",
		Description: "Cycle fields validation failed for with term: «alias»",
		Reason: []string{
			"The expression specified in the cycle clause is not an identifier or path term.",
		},
		Action: []string{
			"Revise statement removing or modifying the invalid cycle clause expression.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VECTOR_SEMANTIC, // 3400
		symbol:      "E_VECTOR_SEMANTIC",
		Description: "Semantic error in query with vector search function: <<msg>>.",
		Reason: []string{
			"A vector search function cannot be used together with certain features of a query, e.g. GROUP BY clause or " +
				"Window function.",
		},
		Action: []string{
			"Revise the statement to remove the offending features of the query.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VECTOR_INDEX_ATTRIBUTE, // 3401
		symbol:      "E_VECTOR_INDEX_ATTRIBUTE",
		Description: "Invalid index attributes specified for index key <<key>> in CREATE INDEX statement.",
		Reason: []string{
			"Cannot mix index attribute VECTOR with <<attr>> for index key <<key>> in CREATE INDEX statement.",
		},
		Action: []string{
			"Revise the statement to remove the offending index attribute.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VECTOR_INDEX_SINGLE_VECTOR, // 3402
		symbol:      "E_VECTOR_INDEX_SINGLE_VECTOR",
		Description: "Multiple VECTOR index key specified in CREATE INDEX statement for index <<name>>.",
		Reason: []string{
			"Only a single index key with VECTOR attribute is supported in CREATE INDEX statement.",
		},
		Action: []string{
			"Revise the statement to include only a single index key with VECTOR attribute.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VECTOR_INDEX_SINGLE_KEY, // 3403
		symbol:      "E_VECTOR_INDEX_SINGLE_KEY",
		Description: "Multiple index keys specified in CREATE VECTOR INDEX statement for index <<name>>.",
		Reason: []string{
			"Only a single index key (with VECTOR attribute) is supported in CREATE VECTOR INDEX statement.",
		},
		Action: []string{
			"Revise the statement to include only a single index key (with VECTOR attribute).",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VECTOR_INDEX_NO_VECTOR, // 3404
		symbol:      "E_VECTOR_INDEX_NO_VECTOR",
		Description: "No index key with VECTOR attribute specified in CREATE VECTOR INDEX statement for index <<name>>.",
		Reason: []string{
			"An index key with VECTOR attribute must be included in CREATE VECTOR INDEX statement.",
		},
		Action: []string{
			"Revise the statement to include an index key with VECTOR attribute.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VECTOR_FUNC_ORDER_CONST, // 3405
		symbol:      "E_VECTOR_FUNC_ORDER_CONST",
		Description: "Vector function (<<term>>) in ORDER BY clause must use a constant for <<option>>.",
		Reason: []string{
			"A vector function (<<term>>) in ORDER BY clause uses a non-constant for specifying <<option>>.",
		},
		Action: []string{
			"Revise the statement to use a constant for the order option.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VECTOR_FUNC_ORDER_OPTION, // 3406
		symbol:      "E_VECTOR_FUNC_ORDER_OPTION",
		Description: "Vector function (<<term>>) in ORDER BY clause must use ASC and NULLS LAST.",
		Reason: []string{
			"A vector function (<<term>>) in ORDER BY clause uses an invalid option <<option>>.",
		},
		Action: []string{
			"Revise the statement to use only ASC and NULLS LAST order options.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VECTOR_DISTINCT_ARRAY_KEY, // 3407
		symbol:      "E_VECTOR_DISTINCT_ARRAY_KEY",
		Description: "Cannot use DISTINCT in an array index key with VECTOR attribute in CREATE INDEX statement.",
		Reason: []string{
			"An array index key with VECTOR attribute is specified using DISTINCT in CREATE INDEX statement.",
		},
		Action: []string{
			"Revise the statement to remove DISTINCT in array index key with VECTOR attribute.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:   E_VECTOR_CONSTANT_INDEX_KEY, // 3408
		symbol: "E_VECTOR_CONSTANT_INDEX_KEY",
		Description: "Cannot use a constant construct (object or array) in an index key with VECTOR attribute in " +
			"CREATE INDEX statement.",
		Reason: []string{
			"An index key (<<name>>) with VECTOR attribute is specified as an object construct or array construct in " +
				"CREATE INDEX statement.",
		},
		Action: []string{
			"Revise the statement to not use a constant construct in index key with VECTOR attribute.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_PLAN, // 4000
		symbol:      "E_PLAN",
		Description: "A planning error occurred.",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_REPREPARE, // 4001
		symbol:      "E_REPREPARE",
		Description: "Reprepare error",
		Reason: []string{
			"A parsing error occurred when re-preparing a statement.",
			"There was an error building the plan when re-preparing a statement.",
			"There was an error storing the re-prepared plan in the cache.",
		},
		Action: []string{
			"Prepare the statement under a new name.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NO_TERM_NAME, // 4010
		symbol:      "E_NO_TERM_NAME",
		Description: "From Term must have a name or alias.",
		Reason: []string{
			"The statement includes an unnamed FROM term.",
		},
		Action: []string{
			"Revise the statement aliasing terms that are unnamed (i.e. non-keyspace path terms).",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DUPLICATE_ALIAS, // 4020
		symbol:      "E_DUPLICATE_ALIAS",
		Description: "Duplicate alias «alias» «location»",
		Reason: []string{
			"The statement defines the alias multiple times for different elements.",
		},
		Action: []string{
			"Revise the statement to ensure aliases are unique.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DUPLICATE_WITH_ALIAS, // 4021
		symbol:      "E_DUPLICATE_WITH_ALIAS",
		Description: "Duplicate WITH alias reference in «term»: «alias» «location»",
		Reason: []string{
			"The statement contains a duplicate reference to the noted alias in a FROM expression.",
		},
		Action: []string{
			"Revise the statement to ensure alias references are unique.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_UNKNOWN_FOR, // 4025
		symbol:      "E_UNKNOWN_FOR",
		Description: "Unknow alias in : ON KEY «expr» FOR «alias». ",
		Reason: []string{
			"The statement contains an index join FOR clause referencing an unknown alias.",
		},
		Action: []string{
			"Revise the statement correcting the alias.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SUBQUERY_MISSING_KEYS, // 4030
		symbol:      "E_SUBQUERY_MISSING_KEYS",
		Description: "FROM in correlated subquery must have USE KEYS clause: FROM «keyspace».",
		IsUser:      YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SUBQUERY_MISSING_INDEX, // 4035
		symbol:      "E_SUBQUERY_MISSING_INDEX",
		Description: "No secondary index available for keyspace «keyspace» in correlated subquery.",
		Reason: []string{
			"A correlated sub-query was specified but a suitable index on the keyspace was not available to support it.",
		},
		Action: []string{
			"Create the necessary index.",
			"Check the expected index is online.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:   E_SUBQUERY_PRIMARY_DOCS_EXCEEDED, // 4036
		symbol: "E_SUBQUERY_PRIMARY_DOCS_EXCEEDED",
		Description: "Correlated subquery's keyspace «keyspace» cannot have more than «number» documents without " +
			"appropriate secondary index",
		Reason: []string{
			"A primary scan supporting a correlated sub-query returned more keys than permitted.",
		},
		Action: []string{
			"Create an appropriate secondary index to support the sub-query.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NO_SUCH_PREPARED, // 4040
		symbol:      "E_NO_SUCH_PREPARED",
		Description: "No such prepared statement: «name»",
		Reason: []string{
			"The prepared statement referenced in the request doesn't exist.",
		},
		Action: []string{
			"Verify that a valid prepared statement name is specified.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_UNRECOGNIZED_PREPARED, // 4050
		symbol:      "E_UNRECOGNIZED_PREPARED",
		Description: "JSON unmarshalling error: «details»",
		Reason: []string{
			"A request with a non-character string ˝prepared˝ parameter value was received.",
			"Automatic execution (auto_execute) failed to produce a prepared statement.",
			"Inter-node prepared statement distribution failed.",
		},
		Action: []string{
			"Ensure a valid value is passed for ˝prepared˝.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_PREPARED_NAME, // 4060
		symbol:      "E_PREPARED_NAME",
		Description: "Unable to add name: «reason»",
		Reason: []string{
			"A prepared statement with the same name was already defined.",
		},
		Action: []string{
			"Use a unique name for each prepared statement.",
			"Delete unwanted prepared statements.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_PREPARED_DECODING, // 4070
		symbol:      "E_PREPARED_DECODING",
		Description: "Unable to decode prepared statement",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_PREPARED_ENCODING_MISMATCH, // 4080
		symbol:      "E_PREPARED_ENCODING_MISMATCH",
		Description: "Encoded plan parameter does not match encoded plan of «name»",
		Reason: []string{
			"The ˝encoded_plan˝ parameter received for a prepared statement didn't match the cached plan for the statement.",
			"Different nodes had a plan of the same name but their plans differed.",
		},
		Action: []string{
			"Resubmit the request and the cached plan will be used.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ENCODING_NAME_MISMATCH, // 4090
		symbol:      "E_ENCODING_NAME_MISMATCH",
		Description: "Mismatching name in encoded plan, expecting: «expected», found: «found»",
		Reason: []string{
			"The name in an encoded plan doesn't match the prepared statement's name.",
		},
		Action: []string{
			"Correct the request if passing an encoded plan.",
			"Delete the entry from the prepareds cache.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ENCODING_CONTEXT_MISMATCH, // 4091
		symbol:      "E_ENCODING_CONTEXT_MISMATCH",
		Description: "Mismatching query_context in encoded plan",
		Reason: []string{
			"The query context in an encoded plan doesn't match the prepared statement's query context.",
		},
		Action: []string{
			"Correct the request if passing an encoded plan.",
			"Delete the entry from the prepareds cache.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_PREDEFINED_PREPARED_NAME, // 4092
		symbol:      "E_PREDEFINED_PREPARED_NAME",
		Description: "Prepared name «name» is predefined (reserved).",
		Action: []string{
			"Don't use predefined names for prepared statements.\nPredefined names have a double underscore ('__') " +
				"prefix and include: '__get','__insert','__upsert','__update' and '__delete'.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NO_INDEX_JOIN, // 4100
		symbol:      "E_NO_INDEX_JOIN",
		Description: "No index available for join term «term»",
		Reason: []string{
			"There was no available index to support the index join or NEST on the noted term.",
		},
		Action: []string{
			"Create an appropriate secondary index to support the join or NEST statement.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_USE_KEYS_USE_INDEXES, // 4110
		symbol:      "E_USE_KEYS_USE_INDEXES",
		Description: "From Expression Term cannot have USE KEYS or USE INDEX Clause",
		Reason: []string{
			"An expression term in a from clause specifies USE KEYS or USE INDEX.\n" +
				"e.g. SELECT * FROM (SELECT * FROM default) a USE KEYS['key']",
		},
		Action: []string{
			"Revise the statement removing the invalid clauses or expression term.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NO_INDEX_SERVICE, // 4115
		symbol:      "E_NO_INDEX_SERVICE",
		Description: "Index service not available.",
		Reason: []string{
			"No active Index service nodes were found in this cluster.",
		},
		Action: []string{
			"Ensure the Index service is defined and operational in the cluster before attempting index operations.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:   E_NO_PRIMARY_INDEX, // 4120
		symbol: "E_NO_PRIMARY_INDEX",
		Description: "No index available on keyspace «keyspace» that matches your query. Use CREATE PRIMARY INDEX ON " +
			"«keyspace» to create a primary index, or check that your expected index is online.",
		Reason: []string{
			"The statement was attempting to scan the keyspace but there was no index available to support the scan.",
		},
		Action: []string{
			"Create a appropriate index to support the scan.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_PRIMARY_INDEX_OFFLINE, // 4125
		symbol:      "E_PRIMARY_INDEX_OFFLINE",
		Description: "Primary index «indexname» not online.",
		Reason: []string{
			"A statement was attempting to scan a keyspace using a primary index however the index was not online.",
		},
		Action: []string{
			"Check the state of the index and build it if necessary.",
			"Create a appropriate index to support the scan.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_LIST_SUBQUERIES, // 4130
		symbol:      "E_LIST_SUBQUERIES",
		Description: "Error listing sub-queries.",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NOT_GROUP_KEY_OR_AGG, // 4210
		symbol:      "E_NOT_GROUP_KEY_OR_AGG",
		Description: "Expression «expression» must depend only on group keys or aggregates.",
		Reason: []string{
			"The statement contained grouping and an expression in the projection referenced a value that was not a " +
				"grouping key or an aggregate.\ne.g. SELECT a FROM b GROUP BY c",
		},
		Action: []string{
			"Revise the statement to use only grouping keys or aggregates in the projection.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INDEX_ALREADY_EXISTS, // 4300
		symbol:      "E_INDEX_ALREADY_EXISTS",
		Description: "The index «name» already exists.",
		Reason: []string{
			"An attempt was made to create an index with a name that already exists.",
		},
		Action: []string{
			"Verify that existing index has the desired definition.",
			"Use unique index names.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_AMBIGUOUS_META, // 4310
		symbol:      "E_AMBIGUOUS_META",
		Description: "«meta-term» in query with multiple FROM terms requires an argument",
		Reason: []string{
			"A statement with multiple FROM terms includes meta-data function without a qualifying argument.",
		},
		Action: []string{
			"Add the term qualification argument to the meta-data function.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INDEXER_DESC_COLLATION, // 4320
		symbol:      "E_INDEXER_DESC_COLLATION",
		Description: "DESC option is not supported by the indexer.",
		Reason: []string{
			"The GSI indexer doesn't support descending key ordering.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_PLAN_INTERNAL, // 4321
		symbol:      "E_PLAN_INTERNAL",
		Description: "Plan error: «what»",
		Reason: []string{
			"An internal error occurred whilst generating the query plan.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ALTER_INDEX, // 4322
		symbol:      "E_ALTER_INDEX",
		Description: "ALTER INDEX not supported",
		Reason: []string{
			"An ALTER INDEX statement was attempted but is not supported by the indexer.",
		},
		Action: []string{
			"Drop and re-create to alter an index.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_PLAN_NO_PLACEHOLDER, // 4323
		symbol:      "E_PLAN_NO_PLACEHOLDER",
		Description: "Placeholder is not allowed in keyspace",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NO_ANSI_JOIN, // 4330
		symbol:      "E_NO_ANSI_JOIN",
		Description: "No index available for ANSI «type» term «alias»",
		Reason: []string{
			"The statement contains an ANSI JOIN with the noted alias and no index exists to support it.",
			"The statement contains an ANSI NEST with the noted alias and no index exists to support it.",
		},
		Action: []string{
			"Create a appropriate index to support the operation.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_PARTITION_INDEX_NOT_SUPPORTED, // 4340
		symbol:      "E_PARTITION_INDEX_NOT_SUPPORTED",
		Description: "PARTITION index is not supported by indexer.",
		Reason: []string{
			"The statement includes an index partitioning clause that isn't supported by the indexer.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_GSI, // 4350
		symbol:      "E_GSI",
		Description: "An error occurred in GSI",
		Reason: []string{
			"An attempt was made to define an index with a duplicate name.",
			"An attempt was made to manage an index that was not defined.",
			"An operation the user lacked necessary permissions for was attempted on an index.",
			"An internal error occurred in the GSI sub-component.",
		},
		Action: []string{
			"Review the reported error for more detail on why the operation failed and possible user actions.",
		},
		IsUser: MAYBE,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_GSI_TRANSIENT, // 4360
		symbol:      "W_GSI_TRANSIENT",
		Description: "A transient error occurred in GSI",
		Action: []string{
			"Review the error details for possible user actions.\n" +
				"GSI will typically handle this condition and retry the operation automatically when appropriate.\n" +
				"The state of index build operations can be monitored via the GSI endpoint or system:indexes collection.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_GSI_TEMP_FILE_SIZE, // 4370
		symbol:      "E_GSI_TEMP_FILE_SIZE",
		Description: "«request» temp file size exceeded limit «limit», «size»",
		Reason: []string{
			"GSI was unable to write to a temporary file for the request as the configured temporary disk space limit was reached.",
		},
		Action: []string{
			"Check the settings indicated permit sufficient space for the operation.",
			"Review active statements and their temporary space requirements.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NO_KNN_SEARCH_INDEX, // 4380
		symbol:      "E_NO_KNN_SEARCH_INDEX",
		Description: "Search() function with KNN has no search index",
		Reason: []string{
			"Query uses Search() as predicate with KNN, but there is no matching FTS index",
		},
		Action: []string{
			"Create appropriate FTS index",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ENCODED_PLAN_NOT_ALLOWED, // 4400
		symbol:      "E_ENCODED_PLAN_NOT_ALLOWED",
		Description: "Encoded plan use is not allowed in serverless mode.",
		Reason: []string{
			"The server was operating in ˝serverless˝ mode and the request attempted to pass an encoded plan for execution.",
		},
		Action: []string{
			"Submit the statement text for planning and execution.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CBO, // 4600
		symbol:      "E_CBO",
		Description: "Error occurred during cost-based optimization: «what»",
		Action:      []string{},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INDEX_STAT, // 4610
		symbol:      "E_INDEX_STAT",
		Description: "Invalid index statistics for index «name» : «what»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_DOCUMENT_KEY_TYPE, // 4998
		symbol:      "W_DOCUMENT_KEY_TYPE",
		Description: "Document key must be a string: «key»",
		Reason: []string{
			"A key in a USE KEYS clause was not a string.\ne.g. SELECT * FROM default USE KEYS[123];",
		},
		Action: []string{
			"Correct the key in the statement.",
		},
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_GENERIC, // 4999
		symbol:      "W_GENERIC",
		Description: "A non-specific warning was raised.",
		Reason: []string{
			"Clustering was unable to add a Query service node due to the detailed incompatibility.",
		},
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INTERNAL, // 5000
		symbol:      "E_INTERNAL",
		Description: "An internal error occurred.",
		Reason: []string{
			"A sub-component such as GSI may be reporting an error.",
			"An internal error occurred.",
		},
		Action: []string{
			"If reported by a sub-component, review the error details for appropriate actions.",
			"Contact support",
		},
		IsUser: MAYBE,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_EXECUTION_PANIC, // 5001
		symbol:      "E_EXECUTION_PANIC",
		Description: "A panic occurred during execution.",
		Reason: []string{
			"The server encountered an internal error that resulted in a panic which halted request processing.",
		},
		Action: []string{
			"Contact support",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_EXECUTION_INTERNAL, // 5002
		symbol:      "E_EXECUTION_INTERNAL",
		Description: "Execution internal error: «what»",
		Action: []string{
			"Contact support",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_EXECUTION_PARAMETER, // 5003
		symbol:      "E_EXECUTION_PARAMETER",
		Description: "Execution parameter error: «reason»",
		Reason: []string{
			"The request has a USING clause and provides parameters.",
			"The request has a USING clause that includes a non-static value.",
		},
		Action: []string{
			"Either provide parameters or use a USING clause.",
			"Use only static value in a USING clause.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_PARSING, // 5004
		symbol:      "E_PARSING",
		Description: "Expression parsing «expression» failed",
		Reason: []string{
			"A projection EXCLUDE clause in the statement contains a string reference that cannot be parsed.",
		},
		Action: []string{
			"Revise the statement to use only valid expressions in projection EXCLUDE clauses.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TEMP_FILE_QUOTA, // 5005
		symbol:      "E_TEMP_FILE_QUOTA",
		Description: "Temporary file quota exceeded",
		Reason: []string{
			"An order by operation was unable to spill to disk as the temporary space quota was hit.",
			"A group by operation was unable to spill to disk as the temporary space quota was hit.",
			"An ordered sequential scan was unable to spill to disk as the temporary space quota was hit.",
		},
		Action: []string{
			"Review the Query service temporary space quota setting meets with requirements.",
			"Review statements to reduce the amount of data sorted and grouped.",
			"Create a primary index to support ordered document key operations.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_EXECUTION_KEY_VALIDATION, // 5006
		symbol:      "E_EXECUTION_KEY_VALIDATION",
		Description: "Out of key validation space.",
		Reason: []string{
			"The INSERT operation was using a sequential scan exhausted the space reserved to exclude new keys.",
			"The space reserved to record keys processed by the UPSERT statement was exhausted.",
		},
		Action: []string{
			"Divide the statement into portions that don't exceed the key validation space.",
			"Create a suitable secondary index to support the statement.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_EXECUTION_CURL, // 5007
		symbol:      "E_EXECUTION_CURL",
		Description: "Error executing CURL function",
		Reason: []string{
			"An attempt was made to access a URL outside of the configured list.",
			"An attempt was made to access a restricted URL using the CURL() function.",
			"An attempt was made to access an invalid URL.",
			"The CURL() function failed to complete within the specified time limit.",
			"An invalid option was passed to the CURL() function.",
			"The CURL() function failed to access the URL for the reason noted.",
		},
		Action: []string{
			"Check the server configuration permits access to the URL.",
			"Don't attempt to access restricted URLs.",
			"Ensure the URL is correctly formed with a valid, supported scheme.",
			"Ensure the time limit specified is suitable for the URL response time.",
			"Refer to the documentation for valid options.",
			"Depending on the error, contact your administrator or support.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_EXECUTION_STATEMENT_STOPPED, // 5008
		symbol:      "E_EXECUTION_STATEMENT_STOPPED",
		Description: "Execution of statement has been stopped.",
		Reason: []string{
			"A nested statement was stopped when the nesting statement stopped.",
		},
		Action: []string{
			"Verify it is expected that the nesting statement was stopped.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_EVALUATION, // 5010
		symbol:      "E_EVALUATION",
		Description: "Error evaluating «what»",
		Reason: []string{
			"The noted error occurred during the evaluation of the indicated item.",
		},
		Action: []string{
			"Review the indicated error and it's actions.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_EVALUATION_ABORT, // 5011
		symbol:      "E_EVALUATION_ABORT",
		Description: "Abort: «reason»",
		Reason: []string{
			"The SQL++ abort() function was called in the statement.\ne.g. SELECT abort('An example cause')",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_EXPLAIN, // 5015
		symbol:      "E_EXPLAIN",
		Description: "EXPLAIN: Error marshalling JSON.",
		Action: []string{
			"Contact support",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_EXPLAIN_FUNCTION, // 5017
		symbol:      "E_EXPLAIN_FUNCTION",
		Description: "EXPLAIN FUNCTION: «reason»",
		Reason: []string{
			"The statement failed to build a query plan for an embedded query.",
			"An internal error occurred writing the plan as JSON.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_GROUP_UPDATE, // 5020
		symbol:      "E_GROUP_UPDATE",
		Description: "Error updating «phase» GROUP value",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_DATE, // 5021
		symbol:      "W_DATE",
		Description: "Date error",
		IsUser:      YES,
		IsWarning:   true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_DATE_OVERFLOW, // 5022
		symbol:      "W_DATE_OVERFLOW",
		Description: "Date error: Overflow",
		Action: []string{
			"Ensure date values are in the range -9999-01-01 12:00:00.000000000 UTC to 9999-12-31 09:59:59.999000000 UTC.",
		},
		IsUser:    YES,
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_DATE_INVALID_FORMAT, // 5023
		symbol:      "W_DATE_INVALID_FORMAT",
		Description: "Date error: Invalid format",
		Action: []string{
			"Correct the date format specification.",
		},
		IsUser:    YES,
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_DATE_INVALID_DATE_STRING, // 5024
		symbol:      "W_DATE_INVALID_DATE_STRING",
		Description: "Date error: Invalid date string",
		Reason: []string{
			"A date function converting a string to a date encountered an invalid element in the string, such as a numeric " +
				"month outside the range 1-12.",
		},
		Action: []string{
			"Correct the date string value.",
		},
		IsUser:    YES,
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_DATE_PARSE_FAILED, // 5025
		symbol:      "W_DATE_PARSE_FAILED",
		Description: "Date error: Failed to parse",
		Reason: []string{
			"A date function converting a string to a date failed to parse the input string according to the format.",
		},
		Action: []string{
			"Correct the date string value.",
		},
		IsUser:    YES,
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_DATE_INVALID_COMPONENT, // 5026
		symbol:      "W_DATE_INVALID_COMPONENT",
		Description: "Date error: Invalid component",
		Reason: []string{
			"A date function operating on elements in a date value was given an invalid date component.",
		},
		Action: []string{
			"Correct the date component specified.",
		},
		IsUser:    YES,
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_DATE_NON_INT_VALUE, // 5027
		symbol:      "W_DATE_NON_INT_VALUE",
		Description: "Date error: Value is not an integer",
		Reason: []string{
			"A date function manipulating a component of a date value was provided a non-integer number.",
		},
		Action: []string{
			"Correct the argument to the date function.",
		},
		IsUser:    YES,
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_DATE_INVALID_ARGUMENT, // 5028
		symbol:      "W_DATE_INVALID_ARGUMENT",
		Description: "Date error: Invalid argument",
		Reason: []string{
			"A date function argument was not of the correct type.",
		},
		Action: []string{
			"Correct the argument to the date function.",
		},
		IsUser:    YES,
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_DATE_INVALID_TIMEZONE, // 5029
		symbol:      "W_DATE_INVALID_TIMEZONE",
		Description: "Date error: Invalid time zone",
		Reason: []string{
			"A date value or function argument includes an unknown or invalid timezone.",
		},
		Action: []string{
			"Correct the timezone in the date value or function argument.\n" +
				"Absolute offsets are recommended in place of time zone names.",
		},
		IsUser:    YES,
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INVALID_VALUE, // 5030
		symbol:      "E_INVALID_VALUE",
		Description: "An invalid value was encountered.",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INVALID_EXPRESSION, // 5031
		symbol:      "E_INVALID_EXPRESSION",
		Description: "Invalid expression",
		Reason: []string{
			"An expression in an EXCLUDE clause was invalid.",
			"An expression in an argument to OBJECT_REMOVE_FIELDS() was invalid.",
		},
		Action: []string{
			"Revise the statement providing valid a expression.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_UNSUPPORTED_EXPRESSION, // 5032
		symbol:      "E_UNSUPPORTED_EXPRESSION",
		Description: "iUnsupported expression",
		Reason: []string{
			"An expression in an EXCLUDE clause is not supported.",
			"An expression in an argument to OBJECT_REMOVE_FIELDS() is not supported.",
		},
		Action: []string{
			"Revise the statement providing valid a expression.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_RANGE, // 5035
		symbol:      "E_RANGE",
		Description: "Out of range evaluating «term»",
		Reason: []string{
			"An ARRAY_RANGE() call exceeds the permitted limit for the number of elements produced.",
			"An ARRAY_REPEAT() call exceeds the permitted limit for the number of elements produced.",
			"A REPEAT() call exceeds the permitted limit for the resulting string's size.",
			"A DATE_RANGE_STR() call exceeds the permitted limit for the number of values produced.",
			"A DATE_RANGE_MILLIS() call exceeds the permitted limit for the number of values produced.",
		},
		Action: []string{
			"Revise the function arguments to produce results within the permitted limit.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_DIVIDE_BY_ZERO, // 5036
		symbol:      "W_DIVIDE_BY_ZERO",
		Description: "Division by 0.",
		Reason: []string{
			"An arithmetic operation dividing by zero was encountered during the statement evaluation.",
		},
		Action: []string{
			"If required, revise the statement as necessary to avoid such operations.",
		},
		IsUser:    YES,
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DUPLICATE_FINAL_GROUP, // 5040
		symbol:      "E_DUPLICATE_FINAL_GROUP",
		Description: "Duplicate Final Group.",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INSERT_KEY, // 5050
		symbol:      "E_INSERT_KEY",
		Description: "No INSERT key for «document»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INSERT_VALUE, // 5060
		symbol:      "E_INSERT_VALUE",
		Description: "No INSERT value for «document»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INSERT_KEY_TYPE, // 5070
		symbol:      "E_INSERT_KEY_TYPE",
		Description: "Cannot INSERT non-string key «key» of type «type»",
		Reason: []string{
			"The statement includes an INSERT operation with a non-string key value.\n" +
				"e.g. INSERT INTO default VALUES(1,{'the':'value'})",
		},
		Action: []string{
			"Revise the statement to ensure keys are always string values.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INSERT_OPTIONS_TYPE, // 5071
		symbol:      "E_INSERT_OPTIONS_TYPE",
		Description: "Cannot INSERT non-OBJECT options «options» of type «type»",
		Reason: []string{
			"The statement includes an INSERT operation with a non-OBJECT options value.\n" +
				"e.g. INSERT INTO default VALUES('the_key',{'the':'value'},null)",
		},
		Action: []string{
			"Revise the statement to ensure insert options are always provided as an object value or omitted if unneeded.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_UPSERT_KEY, // 5072
		symbol:      "E_UPSERT_KEY",
		Description: "No UPSERT key for «value»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_UPSERT_KEY_ALREADY_MUTATED, // 5073
		symbol:      "E_UPSERT_KEY_ALREADY_MUTATED",
		Description: "Cannot act on the same key multiple times in an UPSERT statement",
		Reason: []string{
			"The UPSERT statement was trying to modify the same key multiple times.\n" +
				"e.g. UPSERT INTO default VALUES ('key0',{}),('key0',{})",
		},
		Action: []string{
			"Revise the statement to ensure that keys are unique.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_UPSERT_VALUE, // 5075
		symbol:      "E_UPSERT_VALUE",
		Description: "No UPSERT value for «value»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_UPSERT_KEY_TYPE, // 5078
		symbol:      "E_UPSERT_KEY_TYPE",
		Description: "Cannot UPSERT non-string key «key» of type «type».",
		Reason: []string{
			"The statement includes an UPSERT operation with a non-string key value.\n" +
				"e.g. UPSERT INTO default VALUES(1,{'the':'value'})",
		},
		Action: []string{
			"Revise the statement to ensure keys are always string values.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_UPSERT_OPTIONS_TYPE, // 5079
		symbol:      "E_UPSERT_OPTIONS_TYPE",
		Description: "Cannot UPSERT non-OBJECT options «value» of type «type».",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DELETE_ALIAS_MISSING, // 5080
		symbol:      "E_DELETE_ALIAS_MISSING",
		Description: "DELETE alias «alias» not found in item «value»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DELETE_ALIAS_METADATA, // 5090
		symbol:      "E_DELETE_ALIAS_METADATA",
		Description: "DELETE alias «alias» has no metadata in item.",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_UPDATE_ALIAS_MISSING, // 5100
		symbol:      "E_UPDATE_ALIAS_MISSING",
		Description: "UPDATE alias «alias» not found in item",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_UPDATE_ALIAS_METADATA, // 5110
		symbol:      "E_UPDATE_ALIAS_METADATA",
		Description: "UPDATE alias «alias» has no metadata in item.",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_UPDATE_MISSING_CLONE, // 5120
		symbol:      "E_UPDATE_MISSING_CLONE",
		Description: "Missing UPDATE clone.",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_UPDATE_INVALID_FIELD, // 5130
		symbol:      "E_UPDATE_INVALID_FIELD",
		Description: "Invalid field update.",
		Reason: []string{
			"An attempt was made to update a field that doesn't support updating or to set an unsupported value for the field.",
		},
		Action: []string{
			"Revise statement to not update the field.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_UNNEST_INVALID_POSITION, // 5180
		symbol:      "E_UNNEST_INVALID_POSITION",
		Description: "Invalid UNNEST position of type «type»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:   E_SCAN_VECTOR_TOO_MANY_SCANNED_BUCKETS, // 5190
		symbol: "E_SCAN_VECTOR_TOO_MANY_SCANNED_BUCKETS",
		Description: "The scan_vector parameter should not be used for queries accessing more than one keyspace.  Use " +
			"scan_vectors instead. Keyspaces: «list»",
		Reason: []string{
			"The request parameter ˝scan_vector˝ was specified for the request which references multiple keyspaces.",
		},
		Action: []string{
			"Use the ˝scan_vectors˝ request parameter with statements referring to multiple keyspaces.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DYNAMIC_AUTH, // 5201
		symbol:      "E_DYNAMIC_AUTH",
		Description: "Dynamic auth error",
		Reason: []string{
			"The determination of dynamic privileges required by the request failed.",
		},
		Action: []string{
			"Revise the request to ensure all keyspace references can be resolved enabling dynamic authorisation to proceed.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TRANSACTIONAL_AUTH, // 5202
		symbol:      "E_TRANSACTIONAL_AUTH",
		Description: "Transactional auth error",
		Reason: []string{
			"The determination of transaction privileges required by the request failed.",
		},
		Action: []string{
			"Verify the user is permitted to run transaction statements.",
		},
		IsUser: MAYBE,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_USER_NOT_FOUND, // 5210
		symbol:      "E_USER_NOT_FOUND",
		Description: "Unable to find user «user».",
		Reason: []string{
			"A role grant or revoke statement referred to a user that didn't exist.",
			"A user alter or drop statement referred to a user that didn't exist.",
		},
		Action: []string{
			"Verify the user referenced in the statement exists prior to executing the statement.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_USER_EXISTS, // 5211
		symbol:      "E_USER_EXISTS",
		Description: "User «name» already exists.",
		Reason: []string{
			"A create user command was attempted but the user specified already existed.",
		},
		Action: []string{
			"Revise the statement if intending to create a different user.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_USER_ATTRIBUTE, // 5212
		symbol:      "E_USER_ATTRIBUTE",
		Description: "Attribute «attribute» «reason» for «domain» users.",
		Reason: []string{
			"A create user statement creating a ˝local˝ domain user did not specify the password attribute.",
			"A create or alter user statement for a ˝remote˝ domain user specified the password attribute.",
		},
		Action: []string{
			"Specify the password when creating ˝local˝ domain users.",
			"Do not specify the password when creating or altering ˝remote˝ domain users.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_GROUP_EXISTS, // 5213
		symbol:      "E_GROUP_EXISTS",
		Description: "Group «name» already exists.",
		Reason: []string{
			"A create group command was attempted but the group specified already existed.",
		},
		Action: []string{
			"Revise the statement if intending to create a different group.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_GROUP_NOT_FOUND, // 5214
		symbol:      "E_GROUP_NOT_FOUND",
		Description: "Unable to find group «name».",
		Reason: []string{
			"A group role grant or revoke statement specified a group that was not defined.",
			"A group alter or drop statement specified a group that was not defined.",
			"A create or alter user statement specified a group that was not defined.",
		},
		Action: []string{
			"Ensure the group exists before attempting role operations.",
			"Ensure the group exists before attempting to alter or drop it.",
			"Ensure the group exists before assigning to a user.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_GROUP_ATTRIBUTE, // 5215
		symbol:      "E_GROUP_ATTRIBUTE",
		Description: "Attribute «attribute» «reason» for groups.",
		Reason: []string{
			"A create group statement did not specify any roles.",
		},
		Action: []string{
			"Specify at least one role when creating a group.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_MISSING_ATTRIBUTES, // 5216
		symbol:      "E_MISSING_ATTRIBUTES",
		Description: "Missing attributes for «what».",
		Reason: []string{
			"A create user statement did not specify any attributes.",
			"An alter user statement did not specify any attributes.",
			"A create group statement did not specify any attributes.",
			"An alter group statement did not specify any attributes.",
		},
		Action: []string{
			"Specify at least one attribute when creating users or groups.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ROLE_REQUIRES_KEYSPACE, // 5220
		symbol:      "E_ROLE_REQUIRES_KEYSPACE",
		Description: "Role «role» requires a keyspace.",
		Reason: []string{
			"A role in a grant or revoke statement requires qualification with a keyspace.",
			"A role in a group create or alter statement requires qualification with a keyspace.",
		},
		Action: []string{
			"Revise the statement to provide the necessary keyspace qualification.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ROLE_INCORRECT_LEVEL, // 5221
		symbol:      "E_ROLE_INCORRECT_LEVEL",
		Description: "Role «role» cannot be specified at the «level» level.",
		Reason: []string{
			"An attempt was made to specify a scope role as a collection role.",
		},
		Action: []string{
			"Correct the role qualification in the statement.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ROLE_TAKES_NO_KEYSPACE, // 5230
		symbol:      "E_ROLE_TAKES_NO_KEYSPACE",
		Description: "Role «role» does not take a keyspace.",
		Reason: []string{
			"A keyspace qualification has been provided for a role that isn't qualified by keyspace in a grant or revoke role " +
				"statement.",
			"A keyspace qualification has been provided for a role that isn't qualified by keyspace in a group create or alter " +
				"statement.",
		},
		Action: []string{
			"Remove the qualification from the role that isn't qualified by keyspace.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NO_SUCH_KEYSPACE, // 5240
		symbol:      "E_NO_SUCH_KEYSPACE",
		Description: "Keyspace «keyspace» is not valid.",
		Reason: []string{
			"A keyspace qualification provided in a grant or revoke role statement was invalid.",
			"A keyspace qualification provided in a group create or alter statement was invalid.",
		},
		Action: []string{
			"Revise the statement to provide a valid keyspace qualification.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NO_SUCH_SCOPE, // 5241
		symbol:      "E_NO_SUCH_SCOPE",
		Description: "Scope «scope» is not valid.",
		Reason: []string{
			"The scope provided to qualify ˝scope_admin˝ role in a grant or revoke statement was not valid.",
		},
		Action: []string{
			"Revise the statement to provide a valid scope qualification.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NO_SUCH_BUCKET, // 5242
		symbol:      "E_NO_SUCH_BUCKET",
		Description: "Bucket «bucket» is not valid.",
		Reason: []string{
			"The bucket provided to qualify a role in a grant or revoke statement was not valid.",
		},
		Action: []string{
			"Revise the statement to provide a valid bucket qualification.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ROLE_NOT_FOUND, // 5250
		symbol:      "E_ROLE_NOT_FOUND",
		Description: "Role «role» is not valid.",
		Reason: []string{
			"An invalid role was provided in a grant or revoke role statement.",
			"An invalid role was provided in a group create or alter statement.",
		},
		Action: []string{
			"Revise the statement to provide a valid role.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_ROLE_ALREADY_PRESENT, // 5260
		symbol:      "W_ROLE_ALREADY_PRESENT",
		Description: "User «name» already has role «role» «bucket»",
		IsUser:      YES,
		IsWarning:   true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_ROLE_NOT_PRESENT, // 5270
		symbol:      "W_ROLE_NOT_PRESENT",
		Description: "«entity» «name» did not have role «role»",
		Reason: []string{
			"An attempt was made to revoke a role from a user or group that was not held by the user or group.",
		},
		IsUser:    YES,
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_USER_WITH_NO_ROLES, // 5280
		symbol:      "W_USER_WITH_NO_ROLES",
		Description: "User «name» has no roles. Connecting with this user may not be possible",
		Reason: []string{
			"The user has had all roles revoked.",
		},
		IsUser:    YES,
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_GROUP_WITH_NO_ROLES, // 5281
		symbol:      "W_GROUP_WITH_NO_ROLES",
		Description: "Group «name» has no roles.",
		Reason: []string{
			"The group has had all roles revoked.",
		},
		IsUser:    YES,
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_HASH_TABLE_PUT, // 5300
		symbol:      "E_HASH_TABLE_PUT",
		Description: "Hash Table Put failed",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_HASH_TABLE_GET, // 5310
		symbol:      "E_HASH_TABLE_GET",
		Description: "Hash Table Get failed",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_MERGE_MULTI_UPDATE, // 5320
		symbol:      "E_MERGE_MULTI_UPDATE",
		Description: "Multiple UPDATE/DELETE of the same document (document key «key») in a MERGE statement",
		Reason: []string{
			"During a merge statement a key was detected multiple times in update and/or delete operations.",
		},
		Action: []string{
			"Contact support.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_MERGE_MULTI_INSERT, // 5330
		symbol:      "E_MERGE_MULTI_INSERT",
		Description: "Multiple INSERT of the same document (document key «key») in a MERGE statement",
		Reason: []string{
			"The INSERT action of the MERGE statement had previously inserted the noted key.\ne.g. MERGE INTO default" +
				"\n     USING [{},{}] AS source\n     ON default.id IS VALUED\n     WHEN NOT MATCHED THEN\n     INSERT ('key',{})" +
				"\n     ;",
		},
		Action: []string{
			"Revise the statement logic to ensure only unique keys are produced.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_WINDOW_EVALUATION, // 5340
		symbol:      "E_WINDOW_EVALUATION",
		Description: "An error occurred during WINDOW evaluation",
		Reason: []string{
			"An expression in the window aggregate did not evaluate to a number.",
			"The window order clause evaluation failed.",
			"The window partitioning clause evaluation failed.",
		},
		Action: []string{
			"Contact support.",
		},
		IsUser: MAYBE,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADVISE_INDEX, // 5350
		symbol:      "E_ADVISE_INDEX",
		Description: "AdviseIndex: Error marshalling JSON.",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADVISE_INVALID_RESULTS, // 5351
		symbol:      "E_ADVISE_INVALID_RESULTS",
		Description: "Invalid advise results",
		Reason: []string{
			"The results from an advisor session are not valid.",
		},
		Action: []string{
			"Repeat the session.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_UPDATE_STATISTICS, // 5360
		symbol:      "E_UPDATE_STATISTICS",
		Description: "An internal error occurred during update statistics processing.",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SUBQUERY_BUILD, // 5370
		symbol:      "E_SUBQUERY_BUILD",
		Description: "Unable to run subquery",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INDEX_LEADING_KEY_MISSING_NOT_SUPPORTED, // 5380
		symbol:      "E_INDEX_LEADING_KEY_MISSING_NOT_SUPPORTED",
		Description: "Indexing leading key MISSING values are not supported by indexer.",
		Reason: []string{
			"The index definition includes missing values for the leading key and this feature is not supported by the indexer.",
		},
		Action: []string{
			"Revise the index definition to not include missing leading key values.",
			"Contact support.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INDEX_NOT_IN_MEMORY, // 5390
		symbol:      "E_INDEX_NOT_IN_MEMORY",
		Description: "Index «name» is not in memory",
		Reason: []string{
			"The cost of an index scan could not be calculated as the index was not in memory.",
			"Statistics for the index could not be updated as it was not in memory.",
		},
		Action: []string{
			"Re-submit the request.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_MISSING_SYSTEMCBO_STATS, // 5400
		symbol:      "E_MISSING_SYSTEMCBO_STATS",
		Description: "System Collection 'N1QL_CBO_STATS' is required for UPDATE STATISTICS (ANALYZE)",
		Reason: []string{
			"The system bucket N1QL_CBO_STATS does not exist and could not be created.",
		},
		Action: []string{
			"Manually create the bucket and re-submit the request.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INVALID_INDEX_NAME, // 5410
		symbol:      "E_INVALID_INDEX_NAME",
		Description: "index name «name» must be a string",
		Reason: []string{
			"An index name in the index build statement was not a string value compliant with index naming requirements.",
			"An index name in the update statistics statement was not a string value compliant with index naming requirements.",
		},
		Action: []string{
			"Revise the statement to provide a valid index name.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INDEX_NOT_FOUND, // 5411
		symbol:      "E_INDEX_NOT_FOUND",
		Description: "index «name» is not found",
		Reason: []string{
			"The index specified in an index build statement does not exist.",
			"The index specified in an update statistics statement does not exist.",
		},
		Action: []string{
			"Revise the statement to provide a valid index name.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INDEX_UPD_STATS, // 5415
		symbol:      "E_INDEX_UPD_STATS",
		Description: "Error with UPDATE STATISTICS for indexes («names»): «reason»",
		Action: []string{
			"Contact support",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TIME_PARSE, // 5416
		symbol:      "E_TIME_PARSE",
		Description: "Error parsing time string «string»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:   E_JOIN_ON_PRIMARY_DOCS_EXCEEDED, // 5420
		symbol: "E_JOIN_ON_PRIMARY_DOCS_EXCEEDED",
		Description: "Inner of nested-loop join «keyspace» cannot have more than 1000 documents without appropriate " +
			"secondary index",
		Reason: []string{
			"A nested loop join using a primary scan produced more than 1000 documents.\nThis limit is imposed for " +
				"resource usage and performance considerations as the inner leg of such a join may be executed repeatedly.",
		},
		Action: []string{
			"Limit the number of documents accessed in this manner or create an appropriate secondary index to support the join.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INVALID_QUERY_VECTOR, // 5430
		symbol:      "E_INVALID_QUERY_VECTOR",
		Description: "Invalid parameter (query vector) specified for vector function: <<msg>>.",
		Reason: []string{
			"An invalid parameter (query vector) is specified for vector function: <<msg>>.",
		},
		Action: []string{
			"Revise the vector function to use a proper parameter (query vector).",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INVALID_PROBES, // 5431
		symbol:      "E_INVALID_PROBES",
		Description: "Invalid parameter (probes) specified for vector function: <<msg>>.",
		Reason: []string{
			"An invalid parameter (probes) is specified for vector function: <<msg>>.",
		},
		Action: []string{
			"Revise the vector function to use a proper parameter (probes).",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INVALID_RERANK, // 5432
		symbol:      "E_INVALID_RERANK",
		Description: "Invalid parameter (rerank) specified for vector function: <<msg>>.",
		Reason: []string{
			"An invalid parameter (rerank) is specified for vector function: <<msg>>.",
		},
		Action: []string{
			"Revise the vector function to use a proper parameter (rerank).",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_MAXHEAP_SIZE_EXCEEDED, // 5433
		symbol:      "E_MAXHEAP_SIZE_EXCEEDED",
		Description: "Total heap size for (Limit + Offset) exceeded maximum heap size allowed for vector index <<index>>.",
		Reason: []string{
			"Limit and/or Offset specified as query parameters have values that exceeded maximum allowed values for vector index.",
		},
		Action: []string{
			"Specify Limit/Offset as constants and re-submit the query.",
			"Use values that do not exceed the maximum allowed value and re-submit the query.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_MEMORY_QUOTA_EXCEEDED, // 5500
		symbol:      "E_MEMORY_QUOTA_EXCEEDED",
		Description: "Request has exceeded memory quota.",
		Reason: []string{
			"The tracked memory required for the request exceeded the quota set for it.",
		},
		Action: []string{
			"Review the statement execution plan for efficiency.",
			"Review the volume of data the statement is expected to process and the operations used.",
			"Increase the memory quota if appropriate.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NIL_EVALUATE_PARAM, // 5501
		symbol:      "E_NIL_EVALUATE_PARAM",
		Description: "nil «param» parameter for evaluation",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_BUCKET_ACTION, // 5502
		symbol:      "E_BUCKET_ACTION",
		Description: "Unable to complete action after «count» attempts",
		Reason: []string{
			"The indicated action was attempted a number of times without success.",
		},
		Action: []string{
			"If the associated error indicates an infrastructure issue, verify all cluster resources are accessible, available " +
				"and online.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_MISSING_KEY, // 5503
		symbol:      "W_MISSING_KEY",
		Description: "Key(s) in USE KEYS hint not found",
		Reason: []string{
			"A key in a USE KEYS clause was not found.",
		},
		Action: []string{
			"If necessary, revise the keys listed.",
		},
		IsUser:    YES,
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NODE_QUOTA_EXCEEDED, // 5600
		symbol:      "E_NODE_QUOTA_EXCEEDED",
		Description: "Query node has run out of memory",
		Reason: []string{
			"A node level memory quota was in effect and while trying to execute the request the limit was reached",
		},
		Action: []string{
			"Verify the memory requirements of the request, possibly setting a request memory quota.",
			"Retry the request with less concurrent activity.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TENANT_QUOTA_EXCEEDED, // 5601
		symbol:      "E_TENANT_QUOTA_EXCEEDED",
		Description: "«entity» has run out of memory: requested «size», limit «limit»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VALUE_RECONSTRUCT, // 5700
		symbol:      "E_VALUE_RECONSTRUCT",
		Description: "Failed to reconstruct value",
		Reason: []string{
			"An error occurred reconstructing a value data spilled temporarily to disk during the request processing.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VALUE_INVALID, // 5701
		symbol:      "E_VALUE_INVALID",
		Description: "Invalid reconstructed value",
		Reason: []string{
			"An error occurred reconstructing a value data spilled temporarily to disk during the request processing.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VALUE_SPILL_CREATE, // 5702
		symbol:      "E_VALUE_SPILL_CREATE",
		Description: "Failed to create spill file",
		Reason: []string{
			"The request processing requires date be temporarily spilled to disk and the temporary file creation failed.",
		},
		Action: []string{
			"Review the Query service temporary data directory and validate the filesystem is in good order with sufficient " +
				"space to support the node's temporary data requirements.",
			"Review the request memory quota, if in effect, and increase it if appropriate to avoid spilling.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VALUE_SPILL_READ, // 5703
		symbol:      "E_VALUE_SPILL_READ",
		Description: "Failed to read from spill file",
		Reason: []string{
			"An error occurred reading data spilled temporarily to disk during the request processing.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VALUE_SPILL_WRITE, // 5704
		symbol:      "E_VALUE_SPILL_WRITE",
		Description: "Failed to write to spill file",
		Reason: []string{
			"An error occurred writing data temporarily to disk during the request processing.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VALUE_SPILL_SIZE, // 5705
		symbol:      "E_VALUE_SPILL_SIZE",
		Description: "Failed to determine spill file size",
		Reason: []string{
			"An error occurred accessing the file used for data temporarily spilled to disk during the request processing.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VALUE_SPILL_SEEK, // 5706
		symbol:      "E_VALUE_SPILL_SEEK",
		Description: "Failed to seek in spill file",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VALUE_SPILL_MAX_FILES, // 5707
		symbol:      "E_VALUE_SPILL_MAX_FILES",
		Description: "Too many spill files",
		Reason: []string{
			"The operation is attempting to use more files for temporarily spilling data to disk than is permitted.",
		},
		Action: []string{
			"Increase your request memory quota.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SCHEDULER, // 6001
		symbol:      "E_SCHEDULER",
		Description: "The scheduler encountered an error in generating uuid for the task entry",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DUPLICATE_TASK, // 6002
		symbol:      "E_DUPLICATE_TASK",
		Description: "Task already exists «task_id»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TASK_RUNNING, // 6003
		symbol:      "E_TASK_RUNNING",
		Description: "Task «id» is currently executing and cannot be deleted",
		Reason: []string{
			"An attempt was made to delete a task that was active at the time.",
		},
		Action: []string{
			"Restrict task deletion attempts to tasks that have completed.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TASK_NOT_FOUND, // 6004
		symbol:      "E_TASK_NOT_FOUND",
		Description: "the task «id» was not found",
		Reason: []string{
			"An attempt was made to delete a task that did not exist.",
		},
		Action: []string{
			"Ensure the task specified exists before attempting to delete it.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TASK_INVALID_PARAMETER, // 6005
		symbol:      "E_TASK_INVALID_PARAMETER",
		Description: "Task parameter «param» not provided.",
		Reason: []string{
			"An attempt was made to perform a task action without a valid task parameter.",
		},
		Action: []string{
			"Attempt to execute the task again. If unsuccessful, contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_REWRITE, // 6500
		symbol:      "E_REWRITE",
		Description: "An error occurred during query rewrite.",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INFER_INVALID_OPTION, // 7000
		symbol:      "E_INFER_INVALID_OPTION",
		Description: "Invalid INFER option argument.",
		Reason: []string{
			"A non-object value was passed as the options parameter to INFER.",
			"An invalid field was specified in the INFER options parameter object.",
		},
		Action: []string{
			"Pass only an object value containing documented fields as the options to INFER.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INFER_OPTION_MUST_BE_NUMERIC, // 7001
		symbol:      "E_INFER_OPTION_MUST_BE_NUMERIC",
		Description: "Option «option» must be numeric.",
		Reason: []string{
			"The INFER option noted was passed a non-numeric value.",
		},
		Action: []string{
			"Revise the option values used in the statement.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INFER_READING_NUMBER, // 7002
		symbol:      "E_INFER_READING_NUMBER",
		Description: "Error reading option «option».",
		Reason: []string{
			"A string value was passed as the ˝flags˝ option to INFER and it did not parse as a number.",
		},
		Action: []string{
			"Revise the statement to pass a valid number, a string that parsed as a number (e.g. ˝0x10˝) or an array of " +
				"valid flag-name strings.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INFER_NO_KEYSPACE_DOCUMENTS, // 7003
		symbol:      "E_INFER_NO_KEYSPACE_DOCUMENTS",
		Description: "Keyspace has no documents, schema inference not possible.",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INFER_CREATE_RETRIEVER, // 7004
		symbol:      "E_INFER_CREATE_RETRIEVER",
		Description: "Error creating document retriever.",
		Reason: []string{
			"Flags used in an INFER statement did not permit any method of sampling the data.",
		},
		Action: []string{
			"Revise the flags option to permit at least one type of access for data sampling.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INFER_NO_RANDOM_ENTRY, // 7005
		symbol:      "E_INFER_NO_RANDOM_ENTRY",
		Description: "Keyspace does not support random document retrieval.",
		Reason: []string{
			"An INFER statement attempted to use a random entry document interface to sample the data but this was not " +
				"supported by the datastore.",
		},
		Action: []string{
			"Ensure flags permit other document sampling mechanisms to be tried.",
			"Contact support.",
		},
		IsUser: MAYBE,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INFER_NO_RANDOM_DOCS, // 7006
		symbol:      "E_INFER_NO_RANDOM_DOCS",
		Description: "Keyspace will not return random documents.",
		Reason: []string{
			"An INFER statement attempted to use a random document interface to sample the data but this was not returning " +
				"any documents.",
		},
		Action: []string{
			"Ensure flags permit other document sampling mechanisms to be tried.",
			"Contact support.",
		},
		IsUser: MAYBE,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INFER_MISSING_CONTEXT, // 7007
		symbol:      "E_INFER_MISSING_CONTEXT",
		Description: "Missing expression context.",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INFER_EXPRESSION_EVAL, // 7008
		symbol:      "E_INFER_EXPRESSION_EVAL",
		Description: "Expression evaluation failed.",
		Reason: []string{
			"Expression evaluation for an INFER statement failed with the error noted.",
		},
		Action: []string{
			"Review the error details for appropriate actions.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INFER_KEYSPACE_ERROR, // 7009
		symbol:      "E_INFER_KEYSPACE_ERROR",
		Description: "Keyspace error.",
		Reason: []string{
			"An INFER statement encountered the noted error retrieving the keyspace document count.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INFER_NO_SUITABLE_PRIMARY_INDEX, // 7010
		symbol:      "E_INFER_NO_SUITABLE_PRIMARY_INDEX",
		Description: "No suitable primary index found.",
		Reason: []string{
			"An INFER statement attempted to use a primary index from which to sample documents but no suitable index was found.",
		},
		Action: []string{
			"Review the INFER options to permit other document sampling methods.",
			"Create a suitable primary index.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INFER_NO_SUITABLE_SECONDARY_INDEX, // 7011
		symbol:      "E_INFER_NO_SUITABLE_SECONDARY_INDEX",
		Description: "No suitable secondary index found.",
		Reason: []string{
			"An INFER statement attempted to use a secondary index from which to sample documents but no suitable index was found.",
		},
		Action: []string{
			"Review the INFER options to permit other document sampling methods.",
			"Create a suitable index.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_INFER_TIMEOUT, // 7012
		symbol:      "W_INFER_TIMEOUT",
		Description: "Stopped after exceeding infer_timeout. Schema may be incomplete.",
		Reason: []string{
			"An INFER statement reached the specified time limit before completion.",
		},
		Action: []string{
			"Review and adjust the ˝infer_timeout˝ INFER option.",
		},
		IsUser:    YES,
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_INFER_SIZE_LIMIT, // 7013
		symbol:      "W_INFER_SIZE_LIMIT",
		Description: "Stopped after exceeding max_schema_MB. Schema may be incomplete.",
		Reason: []string{
			"The data produced by an INFER statement reached the size limit specified before completion.",
		},
		Action: []string{
			"Review and adjust the ˝max_schema_MB˝ INFER option.",
		},
		IsUser:    YES,
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INFER_NO_DOCUMENTS, // 7014
		symbol:      "E_INFER_NO_DOCUMENTS",
		Description: "No documents found, unable to infer schema.",
		Action: []string{
			"Limit INFER operations to keyspaces that contain data.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INFER_CONNECT, // 7015
		symbol:      "E_INFER_CONNECT",
		Description: "Failed to connect to the server.",
		Reason: []string{
			"The stand-alone INFER tool failed to connect to the server for the reason given.",
		},
		AppliesTo: []string{
			"Infer tool",
		},
	},
	{
		Code:        E_INFER_GET_POOL, // 7016
		symbol:      "E_INFER_GET_POOL",
		Description: "Failed to access pool 'default'.",
		Reason: []string{
			"The stand-alone INFER tool failed to access the default namespace for the reason given.",
		},
		AppliesTo: []string{
			"Infer tool",
		},
	},
	{
		Code:        E_INFER_GET_BUCKET, // 7017
		symbol:      "E_INFER_GET_BUCKET",
		Description: "Failed to access bucket.",
		Reason: []string{
			"The stand-alone INFER tool failed to access the bucket for the reason given.",
		},
		AppliesTo: []string{
			"Infer tool",
		},
	},
	{
		Code:        W_INFER_INDEX, // 7018
		symbol:      "W_INFER_INDEX",
		Description: "Index scanning only; document sample may not be representative.",
		Reason: []string{
			"An INFER statement used only index access for document sampling and as these may not contain all keys, the " +
				"sampling was only of a sub-set of the data.",
		},
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INFER_GET_RANDOM, // 7019
		symbol:      "E_INFER_GET_RANDOM",
		Description: "Failed to get random document.",
		Reason: []string{
			"The random entry interface used by an INFER statement to sample documents failed.",
		},
		Action: []string{
			"Review the referenced error and take appropriate action if possible.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INFER_NO_RANDOM_SCAN, // 7020
		symbol:      "E_INFER_NO_RANDOM_SCAN",
		Description: "Keyspace does not support random key scans",
		Reason: []string{
			"An INFER statement attempted to use a random scan to sample the data but this was not supported by the datastore.",
		},
		Action: []string{
			"Ensure flags permit other document sampling mechanisms to be tried.",
			"Contact support.",
		},
		IsUser: MAYBE,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INFER_NO_SEQUENTIAL_SCAN, // 7021
		symbol:      "E_INFER_NO_SEQUENTIAL_SCAN",
		Description: "Sequential scan not available.",
		Reason: []string{
			"An INFER statement attempted to use a sequential scan to sample the data but no available scan mechanism was " +
				"available in the data store.",
		},
		Action: []string{
			"Ensure flags permit other document sampling mechanisms to be tried.",
			"Contact support.",
		},
		IsUser: MAYBE,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INFER_NO_RETRIEVERS, // 7022
		symbol:      "E_INFER_NO_RETRIEVERS",
		Description: "No document retrievers available.",
		Reason: []string{
			"An INFER statement's options, possibly combined with a lack of suitable indexes, meant that there was no document " +
				"sampling mechanism available.",
		},
		Action: []string{
			"Ensure flags permit at least one supported document sampling mechanism and that any required indexes are available.",
		},
		IsUser: MAYBE,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INFER_OPTIONS, // 7023
		symbol:      "E_INFER_OPTIONS",
		Description: "Options must be provided",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INFER_NEXT_DOCUMENT, // 7024
		symbol:      "E_INFER_NEXT_DOCUMENT",
		Description: "NextDocument failed",
		Reason: []string{
			"An error occurred retrieving a document from a sub-query for an INFER statement.",
		},
		Action: []string{
			"Review the referenced error for appropriate actions.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_INFER_INVALID_FLAGS, // 7025
		symbol:      "W_INFER_INVALID_FLAGS",
		Description: "'flags' must be a number, a string or an array not: «type»",
		Reason: []string{
			"An invalid value was passed as the INFER statement's ˝flags˝ option.",
		},
		Action: []string{
			"Revise the statement and provide a valid value for the option.",
		},
		IsUser:    YES,
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_INFER_INVALID_FLAG, // 7026
		symbol:      "W_INFER_INVALID_FLAG",
		Description: "'flags' array element «element» is invalid",
		Reason: []string{
			"An invalid value included in the array passed as the INFER statement's ˝flags˝ option.",
		},
		Action: []string{
			"Revise the statement and provide a valid values in the array.",
		},
		IsUser:    YES,
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_MIGRATION, // 7200
		symbol:      "E_MIGRATION",
		Description: "Error occurred during «what» migration «details»",
		Reason: []string{
			"An upgrade from an earlier version triggered migration of the component which encountered an error.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_MIGRATION_INTERNAL, // 7201
		symbol:      "E_MIGRATION_INTERNAL",
		Description: "Unexpected error occurred during «what» migration «details»",
		Reason: []string{
			"An upgrade from an earlier version triggered migration of the component which encountered an error.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_BACKUP_NOT_POSSIBLE, // 7300
		symbol:      "E_BACKUP_NOT_POSSIBLE",
		Description: "Metadata backup not possible.",
		Reason: []string{
			"A backup has been attempted whilst metadata migration is underway.",
		},
		Action: []string{
			"Wait for the migration to complete before attempting to backup the Query node(s).",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DATASTORE_AUTHORIZATION, // 10000
		symbol:      "E_DATASTORE_AUTHORIZATION",
		Description: "Unable to authorize user.",
		Reason: []string{
			"Authorisation for the user failed with the indicated error.",
		},
		Action: []string{
			"Ensure the request is using the correct credentials and the user holds the permissions necessary for the request.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FTS_MISSING_PORT_ERR, // 10003
		symbol:      "E_FTS_MISSING_PORT_ERR",
		Description: "Missing or Incorrect port in input url.",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NODE_INFO_ACCESS_ERR, // 10004
		symbol:      "E_NODE_INFO_ACCESS_ERR",
		Description: "Issue with accessing node information for rest endpoint «endpoint»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NODE_SERVICE_ERR, // 10005
		symbol:      "E_NODE_SERVICE_ERR",
		Description: "No FTS node in server «server»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FUNCTIONS_NOT_SUPPORTED, // 10100
		symbol:      "E_FUNCTIONS_NOT_SUPPORTED",
		Description: "Functions of type «type» are only supported in Enterprise Edition.",
		IsUser:      YES,
		AppliesTo: []string{
			"Community Edition",
		},
	},
	{
		Code:        E_MISSING_FUNCTION, // 10101
		symbol:      "E_MISSING_FUNCTION",
		Description: "Function «name» not found",
		Reason: []string{
			"An attempt was made to drop the named function but it did not exist.",
		},
		Action: []string{
			"Ensure the function exists before attempting to drop it.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DUPLICATE_FUNCTION, // 10102
		symbol:      "E_DUPLICATE_FUNCTION",
		Description: "Function «name» already exists",
		Reason: []string{
			"An attempt was made to create a function with a name that was already defined.",
		},
		Action: []string{
			"Use the OR REPLACE clause if the intention is to redefine the function.",
			"Change the function name to be unique.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INTERNAL_FUNCTION, // 10103
		symbol:      "E_INTERNAL_FUNCTION",
		Description: "Operation on function «name» encountered an unexpected error: «details».",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ARGUMENTS_MISMATCH, // 10104
		symbol:      "E_ARGUMENTS_MISMATCH",
		Description: "Incorrect number of arguments supplied to function «name»",
		Action: []string{
			"Revise the statement to include the correct number of arguments to the function.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INVALID_FUNCTION_NAME, // 10105
		symbol:      "E_INVALID_FUNCTION_NAME",
		Description: "Invalid function name «name»",
		Reason: []string{
			"The namespace in the function name was invalid.",
			"The scope in the function name did not exist.",
			"The function name had an incorrect number of components.",
		},
		Action: []string{
			"Revise the statement providing a valid function name.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FUNCTIONS_STORAGE, // 10106
		symbol:      "E_FUNCTIONS_STORAGE",
		Description: "Could not access function definition for «where» because «what»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FUNCTION_ENCODING, // 10107
		symbol:      "E_FUNCTION_ENCODING",
		Description: "Could not «operation» function definition for «function»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FUNCTIONS_DISABLED, // 10108
		symbol:      "E_FUNCTIONS_DISABLED",
		Description: "«type» functions are disabled.",
		Reason: []string{
			"The cluster's feature control flags disable the type of functions.",
			"Problems with initialising the environment to run functions of the type prevent them being enabled.",
			"Restrictions in the cluster deployment model disable the type of functions.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FUNCTION_EXECUTION, // 10109
		symbol:      "E_FUNCTION_EXECUTION",
		Description: "Error executing function «name» «details»",
		Reason: []string{
			"The error noted occurred whilst executing the function.",
		},
		Action: []string{
			"Review the error for possible user actions and revise the statement and/or function as appropriate.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TOO_MANY_NESTED_FUNCTIONS, // 10112
		symbol:      "E_TOO_MANY_NESTED_FUNCTIONS",
		Description: "Error executing function: «name»: «num» nested javascript calls",
		Reason: []string{
			"Function execution reached the maximum permitted number of nested calls and was halted.",
		},
		Action: []string{
			"Review your function code to ensure you don't have excess recursion.",
			"Review your functions, flattening where possible so as to not exceed the limit.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INNER_FUNCTION_EXECUTION, // 10113
		symbol:      "E_INNER_FUNCTION_EXECUTION",
		Description: "An error occurred executing an inner function.",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:   E_LIBRARY_PATH_ERROR, // 10114
		symbol: "E_LIBRARY_PATH_ERROR",
		Description: "Invalid javascript library path: «path». Use a root level path, the same path as the function scope, " +
			"or a local path ('./library')",
		Reason: []string{
			"The path specified in a Javascript function creation statement was invalid.",
		},
		Action: []string{
			"Revise the statement providing a path valid for Javascript libraries.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FUNCTION_LOADING, // 10115
		symbol:      "E_FUNCTION_LOADING",
		Description: "Error loading function «name»",
		Reason: []string{
			"An error occurred loading the body of a Javascript function.",
		},
		Action: []string{
			"Review the error referenced for appropriate actions.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FUNCTIONS_UNSUPPORTED_ACTION, // 10118
		symbol:      "E_FUNCTIONS_UNSUPPORTED_ACTION",
		Description: "«operation» is not supported for functions of type «type»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FUNCTION_STATEMENTS, // 10119
		symbol:      "E_FUNCTION_STATEMENTS",
		Description: "Error getting queries inside function «name». «details»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DATASTORE_INVALID_BUCKET_PARTS, // 10200
		symbol:      "E_DATASTORE_INVALID_BUCKET_PARTS",
		Description: "«entity» resolves to «path» - «num» path parts are expected",
		Reason: []string{
			"A bucket path was expected but the provided information did not resolve to a two part path.",
			"A scope path was expected but the provided information did not resolve to a three part path.",
			"A collection path was expected but the provided information did not resolve to a four part path.",
			"A keyspace path was expected but the provided information did not resolve to a two or four part path.",
		},
		Action: []string{
			"Review the request's ˝query_context˝ setting.",
			"Revise the statement to provide a correct path.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_QUERY_CONTEXT, // 10201
		symbol:      "E_QUERY_CONTEXT",
		Description: "Invalid query_context specified: «details»",
		Reason: []string{
			"The request's ˝query_context˝ contains the noted error.",
			"The request's ˝query_context˝ contains too many parts.",
		},
		Action: []string{
			"Correct the ˝query_context˝ value and try the request again.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_BUCKET_NO_DEFAULT_COLLECTION, // 10202
		symbol:      "E_BUCKET_NO_DEFAULT_COLLECTION",
		Description: "Bucket «name» does not have a default collection",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NO_DATASTORE, // 10203
		symbol:      "E_NO_DATASTORE",
		Description: "No datastore is available",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_BUCKET_UPDATER_MAX_ERRORS, // 10300
		symbol:      "E_BUCKET_UPDATER_MAX_ERRORS",
		Description: "Max failures reached. Last error: «error»",
		Reason: []string{
			"The process responsible for synchronising changes to the bucket in the node encountered more failures than the " +
				"maximum tolerated.",
			"The bucket was dropped outside of the node and the synchronisation endpoint was no longer available.",
		},
		Action: []string{
			"If the bucket was dropped this error is expected.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_BUCKET_UPDATER_NO_HEALTHY_NODES, // 10301
		symbol:      "E_BUCKET_UPDATER_NO_HEALTHY_NODES",
		Description: "No healthy nodes found.",
		Reason: []string{
			"The process responsible for synchronising changes to the bucket was unable to find a healthy node on which to find " +
				"the bucket.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_BUCKET_UPDATER_STREAM_ERROR, // 10302
		symbol:      "E_BUCKET_UPDATER_STREAM_ERROR",
		Description: "Streaming error",
		Reason: []string{
			"The process responsible for synchronising changes to the bucket encountered an error reading the information " +
				"stream from the orchestrator.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_BUCKET_UPDATER_AUTH_ERROR, // 10303
		symbol:      "E_BUCKET_UPDATER_AUTH_ERROR",
		Description: "Authentication error: «details»",
		Reason: []string{
			"The process responsible for synchronising changes to the bucket failed to connect to the orchestrator.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_BUCKET_UPDATER_CONNECTION_FAILED, // 10304
		symbol:      "E_BUCKET_UPDATER_CONNECTION_FAILED",
		Description: "Failed to connect to host.",
		Reason: []string{
			"The process responsible for synchronising changes to the bucket failed to connect to the orchestrator.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_BUCKET_UPDATER_ERROR_MAPPING, // 10305
		symbol:      "E_BUCKET_UPDATER_ERROR_MAPPING",
		Description: "Mapping error: «details»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_BUCKET_UPDATER_EP_NOT_FOUND, // 10306
		symbol:      "E_BUCKET_UPDATER_EP_NOT_FOUND",
		Description: "Streaming endpoint not found",
		Reason: []string{
			"The process responsible for synchronising changes to the bucket in the node was unable to find the orchestrator " +
				"endpoint for the bucket.",
			"The bucket was dropped outside of the node and the synchronisation endpoint was no longer available.",
		},
		Action: []string{
			"If the bucket was dropped this error is expected.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADVISOR_SESSION_NOT_FOUND, // 10500
		symbol:      "E_ADVISOR_SESSION_NOT_FOUND",
		Description: "Advisor: Session not found.",
		Reason: []string{
			"An advisor function call was made with the action as ˝stop˝ and an unknown session specified.",
		},
		Action: []string{
			"Verify the correct session is specified when stopping an index advisor session.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADVISOR_INVALID_ACTION, // 10501
		symbol:      "E_ADVISOR_INVALID_ACTION",
		Description: "Advisor: Invalid value for 'action",
		Reason: []string{
			"An advisor function call was made with an invalid value for the ˝action˝ field.",
		},
		Action: []string{
			"Refer to the documentation for valid values to pass to the advisor function.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADVISOR_ACTION_MISSING, // 10502
		symbol:      "E_ADVISOR_ACTION_MISSING",
		Description: "Advisor: missing argument for 'action",
		Reason: []string{
			"An advisor function call was made with an object argument that was missing the ˝action˝ field.",
		},
		Action: []string{
			"Refer to the documentation for valid arguments to pass to the advisor function.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ADVISOR_INVALID_ARGS, // 10503
		symbol:      "E_ADVISOR_INVALID_ARGS",
		Description: "Advisor: Invalid arguments.",
		Reason: []string{
			"An advisor function call was made with invalid arguments.",
		},
		Action: []string{
			"Refer to the documentation for valid arguments to pass to the advisor function.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VECTOR_FUNC_INVALID_METRIC, // 10510
		symbol:      "E_VECTOR_FUNC_INVALID_METRIC",
		Description: "Vector function <<name>> has invalid metric specification (<<metric>>).",
		Reason: []string{
			"An invalid metric specification (<<metric>>) is used in vector function <<name>>.",
		},
		Action: []string{
			"Revise the vector function to use a supported metric specification.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VECTOR_FUNC_INVALID_FIELD, // 10511
		symbol:      "E_VECTOR_FUNC_INVALID_FIELD",
		Description: "Vector function <<name>> has invalid field specification (<<field>>).",
		Reason: []string{
			"An invalid field specification (<<field>>) is used in vector function <<name>>.",
		},
		Action: []string{
			"Revise the vector function to use a valid field specification.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_IS_VECTOR_INVALID_DIMENSION, // 10512
		symbol:      "E_IS_VECTOR_INVALID_DIMENSION",
		Description: "IsVector() function has invalid dimension specification (<<dimension>>).",
		Reason: []string{
			"An invalid dimension specification (<<dimension>>) is used in IsVector() function.",
		},
		Action: []string{
			"Revise the function parameter to use an integer for dimension specification.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_IS_VECTOR_INVALID_ARG, // 10513
		symbol:      "E_IS_VECTOR_INVALID_ARG",
		Description: "IsVector() function has invalid argument (<<msg>>).",
		Reason: []string{
			"An invalid argument (<<msg>>) is used in IsVector() function.",
		},
		Action: []string{
			"Revise the function parameter to use a supported argument.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SYSTEM_DATASTORE, // 11000
		symbol:      "E_SYSTEM_DATASTORE",
		Description: "System datastore error «details»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SYSTEM_KEYSPACE_NOT_FOUND, // 11002
		symbol:      "E_SYSTEM_KEYSPACE_NOT_FOUND",
		Description: "Keyspace not found in system namespace",
		Reason: []string{
			"A reference was made to a keyspace that doesn't exist in the system namespace.",
		},
		Action: []string{
			"Refer to the documentation for valid keyspaces in the system namespace.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SYSTEM_NOT_IMPLEMENTED, // 11003
		symbol:      "E_SYSTEM_NOT_IMPLEMENTED",
		Description: "System datastore :  Not implemented «what»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SYSTEM_NOT_SUPPORTED, // 11004
		symbol:      "E_SYSTEM_NOT_SUPPORTED",
		Description: "System datastore : Not supported «details»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SYSTEM_IDX_NOT_FOUND, // 11005
		symbol:      "E_SYSTEM_IDX_NOT_FOUND",
		Description: "System datastore : Index not found «details»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SYSTEM_IDX_NO_DROP, // 11006
		symbol:      "E_SYSTEM_IDX_NO_DROP",
		Description: "System datastore : This index cannot be dropped «details»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SYSTEM_STMT_NOT_FOUND, // 11007
		symbol:      "E_SYSTEM_STMT_NOT_FOUND",
		Description: "System datastore : Statement not found «details»",
		Reason: []string{
			"An attempt was made to delete an unknown request from completed requests.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_SYSTEM_REMOTE, // 11008
		symbol:      "W_SYSTEM_REMOTE",
		Description: "System datastore : «details»",
		Reason: []string{
			"An operation on a remote node failed as detailed.",
		},
		Action: []string{
			"Contact support.",
		},
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SYSTEM_UNABLE_TO_RETRIEVE, // 11009
		symbol:      "E_SYSTEM_UNABLE_TO_RETRIEVE",
		Description: "System datastore : unable to retrieve «what» from server",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SYSTEM_UNABLE_TO_UPDATE, // 11010
		symbol:      "E_SYSTEM_UNABLE_TO_UPDATE",
		Description: "System datastore : unable to update «what» information in server",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:   W_SYSTEM_FILTERED_ROWS, // 11011
		symbol: "W_SYSTEM_FILTERED_ROWS",
		Description: "One or more documents were excluded from the «namespace» bucket because of insufficient user permissions. " +
			"In an EE system, add the query_system_catalog role to see all rows. In a CE system, add the administrator role to " +
			"see all rows.",
		Reason: []string{
			"The request attempted to access the contents of a restricted access system keyspace and the user was not permitted " +
				"to see all data.",
		},
		Action: []string{
			"Ensure the request is run as a user with the correct privileges.",
		},
		IsUser:    YES,
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SYSTEM_MALFORMED_KEY, // 11012
		symbol:      "E_SYSTEM_MALFORMED_KEY",
		Description: "System datastore : key «key» is not of the correct format for keyspace «keyspace»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SYSTEM_NO_BUCKETS, // 11013
		symbol:      "E_SYSTEM_NO_BUCKETS",
		Description: "The system namespace contains no buckets that contain scopes.",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_SYSTEM_REMOTE_NODE_NOT_FOUND, // 11015
		symbol:      "W_SYSTEM_REMOTE_NODE_NOT_FOUND",
		Description: "Node «node» not found",
		Action: []string{
			"Contact support.",
		},
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_CONNECTION, // 12000
		symbol:      "E_CB_CONNECTION",
		Description: "Cannot connect «details»",
		Reason: []string{
			"On start-up the Query service was unable to connect to the authorisation service.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_NAMESPACE_NOT_FOUND, // 12002
		symbol:      "E_CB_NAMESPACE_NOT_FOUND",
		Description: "Namespace not found in CB datastore: «details»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_KEYSPACE_NOT_FOUND, // 12003
		symbol:      "E_CB_KEYSPACE_NOT_FOUND",
		Description: "Keyspace not found in CB datastore: «details»",
		Reason: []string{
			"A keyspace referenced in the statement did not exist.",
		},
		Action: []string{
			"Check all expected keyspaces have been created and are correctly referenced in the statement.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_PRIMARY_INDEX_NOT_FOUND, // 12004
		symbol:      "E_CB_PRIMARY_INDEX_NOT_FOUND",
		Description: "Primary Index not found «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_INDEXER_NOT_IMPLEMENTED, // 12005
		symbol:      "E_CB_INDEXER_NOT_IMPLEMENTED",
		Description: "Indexer not implemented «details»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_KEYSPACE_COUNT, // 12006
		symbol:      "E_CB_KEYSPACE_COUNT",
		Description: "Failed to get count for keyspace «details»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_BULK_GET, // 12008
		symbol:      "E_CB_BULK_GET",
		Description: "Error performing bulk get operation «details»",
		Reason: []string{
			"An error occurred retrieving documents from the data service.",
		},
		Action: []string{
			"Review the error detailed for possible user actions.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_DML, // 12009
		symbol:      "E_CB_DML",
		Description: "DML Error, possible causes include «reason»",
		Reason: []string{
			"An attempt was made to update a document but it was concurrently updated by another request.",
		},
		Action: []string{
			"Review concurrent document update logic and retry the operation as appropriate.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_DELETE_FAILED, // 12011
		symbol:      "E_CB_DELETE_FAILED",
		Description: "Failed to perform «operation» on key «key»",
		Reason: []string{
			"A request to the data service to delete a key failed.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_LOAD_INDEXES, // 12012
		symbol:      "E_CB_LOAD_INDEXES",
		Description: "Failed to load indexes «indexes»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_BUCKET_TYPE_NOT_SUPPORTED, // 12013
		symbol:      "E_CB_BUCKET_TYPE_NOT_SUPPORTED",
		Description: "This bucket type is not supported «details»",
		Reason: []string{
			"An attempt was made to access a bucket of an an unsupported type through the Query service.",
		},
		Action: []string{
			"Migrate all buckets to currently supported types.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_INDEX_SCAN_TIMEOUT, // 12015
		symbol:      "E_CB_INDEX_SCAN_TIMEOUT",
		Description: "Index scan timed out",
		Reason: []string{
			"The maximum time for a primary index scan was reached before producing any keys.",
		},
		Action: []string{
			"Review the state and performance of the indexing nodes in the cluster.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_INDEX_NOT_FOUND, // 12016
		symbol:      "E_CB_INDEX_NOT_FOUND",
		Description: "Index Not Found",
		Reason: []string{
			"An attempt was made to alter or drop an index that didn't exist.",
		},
		Action: []string{
			"Check the index exists before attempting to alter or drop it.",
			"Check for a concurrent deletion of the index.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_GET_RANDOM_ENTRY, // 12017
		symbol:      "E_CB_GET_RANDOM_ENTRY",
		Description: "Error getting random entry from keyspace",
		Reason: []string{
			"An attempt to retrieve a document using the random document interface failed.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_UNABLE_TO_INIT_CB_AUTH, // 12018
		symbol:      "E_UNABLE_TO_INIT_CB_AUTH",
		Description: "Unable to initialize authorization system as required",
		Reason: []string{
			"The authorisation service was not initialised but is required by the datastore.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_AUDIT_STREAM_HANDLER_FAILED, // 12019
		symbol:      "E_AUDIT_STREAM_HANDLER_FAILED",
		Description: "Audit stream handler failed",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_BUCKET_NOT_FOUND, // 12020
		symbol:      "E_CB_BUCKET_NOT_FOUND",
		Description: "Bucket not found in CB datastore «bucket»",
		Reason: []string{
			"An attempt was made to alter or drop a bucket that did not exist.",
		},
		Action: []string{
			"Review the statement and ensure buckets exist beforehand.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_SCOPE_NOT_FOUND, // 12021
		symbol:      "E_CB_SCOPE_NOT_FOUND",
		Description: "Scope not found in CB datastore «scope»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_KEYSPACE_SIZE, // 12022
		symbol:      "E_CB_KEYSPACE_SIZE",
		Description: "Failed to get size for keyspace «details»",
		Reason: []string{
			"An error was encountered acquiring the bucket statistics from the data service.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_SECURITY_CONFIG_NOT_PROVIDED, // 12023
		symbol:      "E_CB_SECURITY_CONFIG_NOT_PROVIDED",
		Description: "Connection security config not provided. Unable to load bucket «bucket»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_CREATE_SYSTEM_BUCKET, // 12024
		symbol:      "E_CB_CREATE_SYSTEM_BUCKET",
		Description: "Error while creating system bucket «details»",
		Reason: []string{
			"Creation of the N1QL_SYSTEM_BUCKET used for storing statistics for query planning failed.",
		},
		Action: []string{
			"Review the error detailed for possible user actions.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_BUCKET_CREATE_SCOPE, // 12025
		symbol:      "E_CB_BUCKET_CREATE_SCOPE",
		Description: "Error while creating scope «details»",
		Reason: []string{
			"An attempt was made to create a scope with a name that already existed.",
		},
		Action: []string{
			"Review the statement and ensure scope names are unique within the bucket.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_BUCKET_DROP_SCOPE, // 12026
		symbol:      "E_CB_BUCKET_DROP_SCOPE",
		Description: "Error while dropping scope «details»",
		Reason: []string{
			"An attempt was made to create a scope with a name that did not exist.",
		},
		Action: []string{
			"Review the statement and ensure scopes exist beforehand.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_BUCKET_CREATE_COLLECTION, // 12027
		symbol:      "E_CB_BUCKET_CREATE_COLLECTION",
		Description: "Error while creating collection «name»",
		Reason: []string{
			"Invalid options were passed in a CREATE COLLECTION statement.",
			"An invalid value was passed for the ˝maxTTL˝ option in a CREATE COLLECTION statement.",
		},
		Action: []string{
			"Revise the statement to provide a valid options.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_BUCKET_DROP_COLLECTION, // 12028
		symbol:      "E_CB_BUCKET_DROP_COLLECTION",
		Description: "Error while dropping collection «details»",
		Reason: []string{
			"An attempt was made to drop a collection that did not exist.",
		},
		Action: []string{
			"Review the statement and ensure collections exist beforehand.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_BUCKET_FLUSH_COLLECTION, // 12029
		symbol:      "E_CB_BUCKET_FLUSH_COLLECTION",
		Description: "Error while flushing collection «name»",
		Reason: []string{
			"An attempt was made to flush a collection but it encountered an error.",
		},
		Action: []string{
			"Review the error detailed for possible user actions.",
			"Contact support.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_BINARY_DOCUMENT_MUTATION, // 12030
		symbol:      "E_BINARY_DOCUMENT_MUTATION",
		Description: "«operation» of binary document is not supported",
		Reason: []string{
			"An attempt was made to operate on a binary document.",
		},
		Action: []string{
			"Revise the statement to ensure only JSON documents are included.",
			"Check the collection contains documents of the expected format.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DURABILITY_NOT_SUPPORTED, // 12031
		symbol:      "E_DURABILITY_NOT_SUPPORTED",
		Description: "Durability is not supported.",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_PRESERVE_EXPIRY_NOT_SUPPORTED, // 12032
		symbol:      "E_PRESERVE_EXPIRY_NOT_SUPPORTED",
		Description: "Preserve expiration is not supported.",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CAS_MISMATCH, // 12033
		symbol:      "E_CAS_MISMATCH",
		Description: "CAS mismatch",
		Reason: []string{
			"A concurrent update of a document was detected.",
		},
		Action: []string{
			"Retry the operation.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DML_MC, // 12034
		symbol:      "E_DML_MC",
		Description: "MC error «details»",
		Reason: []string{
			"A data service operation failed.",
		},
		Action: []string{
			"Review the error for possible user actions.",
			"Retry the request if appropriate.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_NOT_PRIMARY_INDEX, // 12035
		symbol:      "E_CB_NOT_PRIMARY_INDEX",
		Description: "Index «name» exists but is not a primary index",
		Reason: []string{
			"A DROP PRIMARY INDEX indicated a specific index but that index was not a primary index.",
		},
		Action: []string{
			"Review the statement and use the PRIMARY qualifier only when dropping a primary index.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DML_INSERT, // 12036
		symbol:      "E_DML_INSERT",
		Description: "Error in INSERT of key: «key»",
		Reason: []string{
			"A data service error occurred whilst adding a document.",
		},
		Action: []string{
			"Review the error for possible user actions.",
			"Retry the request if appropriate.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ACCESS_DENIED, // 12037
		symbol:      "E_ACCESS_DENIED",
		Description: "User does not have access to «entity»",
		Reason: []string{
			"An attempt was made to access a bucket the user does not have access too.",
		},
		Action: []string{
			"Review the request and bucket access requirements.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_WITH_INVALID_OPTION, // 12038
		symbol:      "E_WITH_INVALID_OPTION",
		Description: "Invalid option «option»",
		Reason: []string{
			"An invalid option was specified in a create or alter bucket statement.",
			"An invalid option was specified in a create or alter sequence statement.",
		},
		Action: []string{
			"Review the statement and correct the options passed.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_WITH_INVALID_TYPE, // 12039
		symbol:      "E_WITH_INVALID_TYPE",
		Description: "Invalid value for «option»",
		Reason: []string{
			"An invalid option value was specified in a create or alter bucket statement.",
			"An invalid option value was specified in a create or alter sequence statement.",
			"A non-integer value was specified for the ˝maxTTL˝ option in a create collection statement.",
		},
		Action: []string{
			"Review the statement and correct the options passed.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INVALID_COMPRESSED_VALUE, // 12040
		symbol:      "E_INVALID_COMPRESSED_VALUE",
		Description: "Invalid compressed document received from datastore",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_BUCKET_CLOSED, // 12041
		symbol:      "E_CB_BUCKET_CLOSED",
		Description: "Bucket is closed: «message»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_SUBDOC_GET, // 12042
		symbol:      "E_CB_SUBDOC_GET",
		Description: "Sub-doc get operation failed",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_SUBDOC_SET, // 12043
		symbol:      "E_CB_SUBDOC_SET",
		Description: "Sub-doc set operation failed",
		Reason: []string{
			"A sub-document update of a sequence document failed.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_DROP_SYSTEM_BUCKET, // 12044
		symbol:      "E_CB_DROP_SYSTEM_BUCKET",
		Description: "Error while dropping system bucket «details»",
		Reason: []string{
			"An error occurred dropping the system bucket ˝N1QL_SYSTEM_BUCKET˝.",
		},
		Action: []string{
			"Review the error detailed for possible user actions.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_BUCKET_EXISTS, // 12045
		symbol:      "E_CB_BUCKET_EXISTS",
		Description: "Bucket «bucket» already exists.",
		Reason: []string{
			"A CREATE BUCKET statement attempted to create a bucket that already existed.",
		},
		Action: []string{
			"Review the statement and ensure bucket names are unique.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INDEXER_VERSION, // 12046
		symbol:      "E_INDEXER_VERSION",
		Description: "All indexer nodes must be version <<ver>> or later (<<cause>>).",
		Reason: []string{
			"An indexer with version lower than <<ver>> is found, cannot support '<<cause>>'.",
		},
		Action: []string{
			"Upgrade all indexer nodes to be at least version <<ver>>.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_CB_SYS_COLLECTION_PRIMARY_INDEX, // 12047
		symbol:      "E_CB_SYS_COLLECTION_PRIMARY_INDEX",
		Description: "Primary index on system collection not available for bucket «bucket»",
		Reason: []string{
			"Primary index on system collection for bucket <<bucket>> is taking longer than expected to be created.",
		},
		Action: []string{
			"Retry action.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DATASTORE_CLUSTER, // 13012
		symbol:      "E_DATASTORE_CLUSTER",
		Description: "Error retrieving cluster «what»",
		Reason: []string{
			"An error occurred obtaining the cluster information from the orchestrator.",
		},
		Action: []string{
			"Review the cluster state and diagnostic logs.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DATASTORE_UNABLE_TO_RETRIEVE_ROLES, // 13013
		symbol:      "E_DATASTORE_UNABLE_TO_RETRIEVE_ROLES",
		Description: "Unable to retrieve roles from server.",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DATASTORE_INSUFFICIENT_CREDENTIALS, // 13014
		symbol:      "E_DATASTORE_INSUFFICIENT_CREDENTIALS",
		Description: "User does not have credentials to «action». Add role «role»  to allow the statement to run.",
		Action: []string{
			"Submit the request as a user with the necessary privileges.",
			"Review the user's privileges.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DATASTORE_UNABLE_TO_RETRIEVE_BUCKETS, // 13015
		symbol:      "E_DATASTORE_UNABLE_TO_RETRIEVE_BUCKETS",
		Description: "Unable to retrieve buckets from server.",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DATASTORE_NO_ADMIN, // 13016
		symbol:      "E_DATASTORE_NO_ADMIN",
		Description: "Unable to determine admin credentials",
		Reason: []string{
			"The indicated error occurred obtaining the correct cluster administration credentials.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DATASTORE_NOT_SET, // 13017
		symbol:      "E_DATASTORE_NOT_SET",
		Description: "Datastore not set",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DATASTORE_INVALID_URI, // 13018
		symbol:      "E_DATASTORE_INVALID_URI",
		Description: "Invalid datastore uri: «uri»",
		Reason: []string{
			"The datastore URI received from the orchestrator was invalid.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INDEX_SCAN_SIZE, // 14000
		symbol:      "E_INDEX_SCAN_SIZE",
		Description: "Unacceptable size for index scan: «size»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FILE_DATASTORE, // 15000
		symbol:      "E_FILE_DATASTORE",
		Description: "Error in file datastore «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FILE_NAMESPACE_NOT_FOUND, // 15001
		symbol:      "E_FILE_NAMESPACE_NOT_FOUND",
		Description: "Namespace not found in file store «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FILE_KEYSPACE_NOT_FOUND, // 15002
		symbol:      "E_FILE_KEYSPACE_NOT_FOUND",
		Description: "Keyspace not found «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FILE_DUPLICATE_NAMESPACE, // 15003
		symbol:      "E_FILE_DUPLICATE_NAMESPACE",
		Description: "Duplicate Namespace «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FILE_DUPLICATE_KEYSPACE, // 15004
		symbol:      "E_FILE_DUPLICATE_KEYSPACE",
		Description: "Duplicate Keyspace «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FILE_NO_KEYS_INSERT, // 15005
		symbol:      "E_FILE_NO_KEYS_INSERT",
		Description: "No keys to insert «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FILE_KEY_EXISTS, // 15006
		symbol:      "E_FILE_KEY_EXISTS",
		Description: "Key Exists «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FILE_DML, // 15007
		symbol:      "E_FILE_DML",
		Description: "DML Error «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FILE_KEYSPACE_NOT_DIR, // 15008
		symbol:      "E_FILE_KEYSPACE_NOT_DIR",
		Description: "Keyspace path must be a directory «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FILE_IDX_NOT_FOUND, // 15009
		symbol:      "E_FILE_IDX_NOT_FOUND",
		Description: "Index not found «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FILE_NOT_SUPPORTED, // 15010
		symbol:      "E_FILE_NOT_SUPPORTED",
		Description: "Operation not supported «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_FILE_PRIMARY_IDX_NO_DROP, // 15011
		symbol:      "E_FILE_PRIMARY_IDX_NO_DROP",
		Description: "Primary Index cannot be dropped «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_OTHER_DATASTORE, // 16000
		symbol:      "E_OTHER_DATASTORE",
		Description: "Error in datastore «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_OTHER_NAMESPACE_NOT_FOUND, // 16001
		symbol:      "E_OTHER_NAMESPACE_NOT_FOUND",
		Description: "Namespace Not Found «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_OTHER_KEYSPACE_NOT_FOUND, // 16002
		symbol:      "E_OTHER_KEYSPACE_NOT_FOUND",
		Description: "Keyspace Not Found «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_OTHER_NOT_IMPLEMENTED, // 16003
		symbol:      "E_OTHER_NOT_IMPLEMENTED",
		Description: "Not Implemented «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_OTHER_IDX_NOT_FOUND, // 16004
		symbol:      "E_OTHER_IDX_NOT_FOUND",
		Description: "Index not found «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_OTHER_IDX_NO_DROP, // 16005
		symbol:      "E_OTHER_IDX_NO_DROP",
		Description: "Index Cannot be dropped «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_OTHER_NOT_SUPPORTED, // 16006
		symbol:      "E_OTHER_NOT_SUPPORTED",
		Description: "Not supported for this datastore «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_OTHER_KEY_NOT_FOUND, // 16007
		symbol:      "E_OTHER_KEY_NOT_FOUND",
		Description: "Key not found «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INFERENCER_NOT_FOUND, // 16020
		symbol:      "E_INFERENCER_NOT_FOUND",
		Description: "Inferencer not found «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_OTHER_NO_BUCKETS, // 16021
		symbol:      "E_OTHER_NO_BUCKETS",
		Description: "Datastore «name» contains no buckets that contain scopes.",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SCOPES_NOT_SUPPORTED, // 16022
		symbol:      "E_SCOPES_NOT_SUPPORTED",
		Description: "Keyspace does not support scopes: «scopes»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_STAT_UPDATER_NOT_FOUND, // 16030
		symbol:      "E_STAT_UPDATER_NOT_FOUND",
		Description: "StatUpdater not found",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NO_FLUSH, // 16040
		symbol:      "E_NO_FLUSH",
		Description: "Keyspace does not support flush: «keyspace»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SS_IDX_NOT_FOUND, // 16050
		symbol:      "E_SS_IDX_NOT_FOUND",
		Description: "Index not found",
		Reason: []string{
			"Sequential scans were disabled using the feature control flags.",
		},
		Action: []string{
			"If desired, revise the feature control flags to enable sequential scans.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SS_NOT_SUPPORTED, // 16051
		symbol:      "E_SS_NOT_SUPPORTED",
		Description: "«operation» not supported for scan",
		Reason: []string{
			"The sequential scan indexer does not support index management and maintenance operations.",
			"A sequential scan doesn't support aggregate operations.",
		},
		Action: []string{
			"Don't attempt index management & maintenance statements using the sequential scan indexer.",
			"Contact support.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SS_INACTIVE, // 16052
		symbol:      "E_SS_INACTIVE",
		Description: "Inactive scan in Fetch",
		Action: []string{
			"Contact Support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SS_INVALID, // 16053
		symbol:      "E_SS_INVALID",
		Description: "Invalid scan in «operation»",
		Action: []string{
			"Contact Support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SS_CONTINUE, // 16054
		symbol:      "E_SS_CONTINUE",
		Description: "Scan continuation failed",
		Reason: []string{
			"A KV range scan operation could not be continued.",
		},
		Action: []string{
			"Contact Support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SS_CREATE, // 16055
		symbol:      "E_SS_CREATE",
		Description: "Scan creation failed",
		Reason: []string{
			"A KV range scan operation could not be created to support a sequential scan.",
		},
		Action: []string{
			"Review the associated error for appropriate actions.",
			"Consider creating an index to avoid the sequential scan.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SS_CANCEL, // 16056
		symbol:      "E_SS_CANCEL",
		Description: "Scan cancellation failed",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SS_TIMEOUT, // 16057
		symbol:      "E_SS_TIMEOUT",
		Description: "Scan exceeded permitted duration",
		Reason: []string{
			"A sequential scan operation did not complete within the permitted time.",
		},
		Action: []string{
			"Review the request ˝timeout˝ parameter.",
			"Review any associated error for appropriate actions.",
			"Review cluster availability and load.",
			"Confirm end client is consuming results produced by the Query service in a timely manner.",
		},
		IsUser: MAYBE,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SS_CID_GET, // 16058
		symbol:      "E_SS_CID_GET",
		Description: "Failed to get collection ID for scan",
		Action: []string{
			"Contact support",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SS_CONN, // 16059
		symbol:      "E_SS_CONN",
		Description: "Failed to get connection for scan",
		Action: []string{
			"Contact support",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SS_FETCH_WAIT_TIMEOUT, // 16060
		symbol:      "E_SS_FETCH_WAIT_TIMEOUT",
		Description: "Timed out polling scan for data",
		Action: []string{
			"Contact support",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SS_WORKER_ABORT, // 16061
		symbol:      "E_SS_WORKER_ABORT",
		Description: "A fatal error occurred in scan processing",
		Action: []string{
			"Contact support",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SS_FAILED, // 16062
		symbol:      "E_SS_FAILED",
		Description: "Scan failed",
		Reason: []string{
			"The v-bucket map available to a sequential scan was incomplete or contained errors.",
		},
		Action: []string{
			"Review concurrent cluster management operations.",
			"Review the diagnostic logs for further information.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SS_SPILL, // 16063
		symbol:      "E_SS_SPILL",
		Description: "Operation failed on scan spill file",
		Reason: []string{
			"A sorted sequential scan needed to spill data to disk temporarily but could not do so.",
		},
		Action: []string{
			"Review the Query service temporary data directory and validate the filesystem is in good order with sufficient " +
				"space to support the node's temporary data requirements.",
			"Contact support",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SS_VALIDATE, // 16064
		symbol:      "E_SS_VALIDATE",
		Description: "Failed to validate document key",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SS_BAD_RESPONSE, // 16065
		symbol:      "E_SS_BAD_RESPONSE",
		Description: "Invalid scan response received",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TRAN_DATASTORE_NOT_SUPPORTED, // 17001
		symbol:      "E_TRAN_DATASTORE_NOT_SUPPORTED",
		Description: "Transactions are not supported on «type» store",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TRAN_STATEMENT_NOT_SUPPORTED, // 17002
		symbol:      "E_TRAN_STATEMENT_NOT_SUPPORTED",
		Description: "«statement» statement is not supported «qualifier» transaction",
		Reason: []string{
			"A statement was issued to start a transaction whilst already in a transaction.",
			"A statement was issued to end or modify a transaction whilst not in a transaction.",
		},
		Action: []string{
			"Review the application's order of operations.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TRAN_FUNCTION_NOT_SUPPORTED, // 17003
		symbol:      "E_TRAN_FUNCTION_NOT_SUPPORTED",
		Description: "advisor function is not supported within the transaction",
		Reason: []string{
			"An attempt was made to use the advisor function whilst in a transaction.",
		},
		Action: []string{
			"Review the application's order of operations.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TRANSACTION_CONTEXT, // 17004
		symbol:      "E_TRANSACTION_CONTEXT",
		Description: "Transaction context error",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TRAN_STATEMENT_OUT_OF_ORDER, // 17005
		symbol:      "E_TRAN_STATEMENT_OUT_OF_ORDER",
		Description: "Transaction statement is out of order",
		Reason: []string{
			"A request was attempted with a value for the ˝txstmtnum˝ that was not in relative order within the transaction.",
		},
		Action: []string{
			"Review the application logic to ensure a correctly ordered ˝txstmtnum˝.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_START_TRANSACTION, // 17006
		symbol:      "E_START_TRANSACTION",
		Description: "Start Transaction statement error «details»",
		Action: []string{
			"Review the included details for appropriate actions.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_COMMIT_TRANSACTION, // 17007
		symbol:      "E_COMMIT_TRANSACTION",
		Description: "Commit Transaction statement error «details»",
		Action: []string{
			"Review the included details for appropriate actions.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_ROLLBACK_TRANSACTION, // 17008
		symbol:      "E_ROLLBACK_TRANSACTION",
		Description: "Rollback Transaction statement error «details»",
		Action: []string{
			"Review the included details for appropriate actions.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NO_SAVEPOINT, // 17009
		symbol:      "E_NO_SAVEPOINT",
		Description: "«name» savepoint is not defined",
		Reason: []string{
			"A save point referenced in a transaction rollback statement did not exist.",
		},
		Action: []string{
			"Review the application's order of operations.",
			"Ensure the a valid save point name is used.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TRANSACTION_EXPIRED, // 17010
		symbol:      "E_TRANSACTION_EXPIRED",
		Description: "Transaction timeout",
		Reason: []string{
			"A transaction was active for longer than the permitted maximum time.",
		},
		Action: []string{
			"Review the operations in the transaction ensuring they can be completed within the time limit.",
			"Review the transaction time limit setting.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TRANSACTION_RELEASED, // 17011
		symbol:      "E_TRANSACTION_RELEASED",
		Description: "Transaction is released",
		Reason: []string{
			"An attempt was made to access a transaction that had been released.",
		},
		Action: []string{
			"Review other errors raised during the transaction for possible actions.",
			"Review the application's order of operations.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DUPLICATE_KEY, // 17012
		symbol:      "E_DUPLICATE_KEY",
		Description: "Duplicate Key «details»",
		Reason: []string{
			"A key in an insert statement already existed.",
		},
		Action: []string{
			"Ensure uniqueness of document keys.",
			"Review concurrent activity that may lead to duplicates.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TRANSACTION_INUSE, // 17013
		symbol:      "E_TRANSACTION_INUSE",
		Description: "Parallel execution of the statements are not allowed within the transaction",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_KEY_NOT_FOUND, // 17014
		symbol:      "E_KEY_NOT_FOUND",
		Description: "Key not found",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SCAS_MISMATCH, // 17015
		symbol:      "E_SCAS_MISMATCH",
		Description: "«op» cas (actual: «actual», expected:«expected») mismatch for key: «key»",
		Reason: []string{
			"The operation encountered a check-and-set value mismatch for the noted key.\n" +
				"This indicates another update was successful between the document being read and this update operation.",
		},
		Action: []string{
			"Review the transaction operations and concurrent activity.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TRANSACTION_MEMORY_QUOTA_EXCEEDED, // 17016
		symbol:      "E_TRANSACTION_MEMORY_QUOTA_EXCEEDED",
		Description: "Transaction memory («used») exceeded quota («quota»)",
		Action: []string{
			"Review the transaction operations and quota setting.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TRANSACTION_FETCH, // 17017
		symbol:      "E_TRANSACTION_FETCH",
		Description: "Transaction fetch error",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_POST_COMMIT_TRANSACTION, // 17018
		symbol:      "E_POST_COMMIT_TRANSACTION",
		Description: "Failed post commit",
		Reason: []string{
			"A transaction failed during post commit operations for the given reason",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_AMBIGUOUS_COMMIT_TRANSACTION, // 17019
		symbol:      "E_AMBIGUOUS_COMMIT_TRANSACTION",
		Description: "Commit was ambiguous",
		Reason: []string{
			"A transaction commit operation could not be completed ensuring a consistent, precise outcome.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TRANSACTION_STAGING, // 17020
		symbol:      "E_TRANSACTION_STAGING",
		Description: "Transaction staging error",
		Reason: []string{
			"The staging of a write for a transaction failed.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TRANSACTION_QUEUE_FULL, // 17021
		symbol:      "E_TRANSACTION_QUEUE_FULL",
		Description: "Transaction queue is full",
		Reason: []string{
			"Another request for the transaction was executing and server could not queue any more transaction requests when " +
				"this request was received.",
		},
		Action: []string{
			"Revise the concurrent requests submitted for a single transaction.",
			"Review the duration of requests in the transaction.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_POST_COMMIT_TRANSACTION, // 17022
		symbol:      "W_POST_COMMIT_TRANSACTION",
		Description: "Failed post commit",
		Reason: []string{
			"A transaction failed during post commit operations for the given reason",
		},
		Action: []string{
			"Review the details for possible user action.",
		},
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TRANSACTION_XATTRS, // 17023
		symbol:      "E_TRANSACTION_XATTRS",
		Description: "XATTRs not permitted in a transaction",
		Reason: []string{
			"Document XATTRs may not be used in a transaction.",
		},
		Action: []string{
			"Confirm OPTIONS for INSERT/UPSERT do not specify XATTRs.",
			"Remove XATTRs from the statement and re-submit.",
			"Submit the statement outside of a transaction.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_GC_AGENT, // 17096
		symbol:      "E_GC_AGENT",
		Description: "GC agent error",
		Reason: []string{
			"An agent handling the transaction encountered an error during the noted operation.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TRAN_CE_NOTSUPPORTED, // 17097
		symbol:      "E_TRAN_CE_NOTSUPPORTED",
		Description: "Transactions are not supported in Community Edition",
		Reason: []string{
			"A transaction operation was attempted in a Community Edition server.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Community Edition",
		},
	},
	{
		Code:        E_MEMORY_ALLOCATION, // 17098
		symbol:      "E_MEMORY_ALLOCATION",
		Description: "Memory allocation error: «details»",
		Reason: []string{
			"An internal memory pool used for transactions was exhausted.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_TRANSACTION, // 17099
		symbol:      "E_TRANSACTION",
		Description: "A transaction error occurred",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DICT_INTERNAL, // 18010
		symbol:      "E_DICT_INTERNAL",
		Description: "Unexpected error in dictionary: «error»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INVALID_GSI_INDEXER, // 18020
		symbol:      "E_INVALID_GSI_INDEXER",
		Description: "GSI Indexer does not support collections - «reason»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_INVALID_GSI_INDEX, // 18030
		symbol:      "E_INVALID_GSI_INDEX",
		Description: "GSI Index «name» does not support collections",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SYSTEM_COLLECTION, // 18040
		symbol:      "E_SYSTEM_COLLECTION",
		Description: "Error accessing system collection - «details»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DICTIONARY_ENCODING, // 18050
		symbol:      "E_DICTIONARY_ENCODING",
		Description: "Cound not «what» data dictionary entry for «name» due to «reason»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DICT_KEYSPACE_MISMATCH, // 18060
		symbol:      "E_DICT_KEYSPACE_MISMATCH",
		Description: "Decoded dictionary entry for keyspace «keyspace» does not match «keyspace»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_DICT_MISSING_FIELD, // 18070
		symbol:      "E_DICT_MISSING_FIELD",
		Description: "Dictionary entry «entry» for <<name>> is missing field <<field>>",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VIRTUAL_KS_NOT_SUPPORTED, // 19000
		symbol:      "E_VIRTUAL_KS_NOT_SUPPORTED",
		Description: "Virtual Keyspace : Not supported «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VIRTUAL_KS_NOT_IMPLEMENTED, // 19001
		symbol:      "E_VIRTUAL_KS_NOT_IMPLEMENTED",
		Description: "Virtual Keyspace : Not yet implemented «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VIRTUAL_KS_IDXER_NOT_FOUND, // 19002
		symbol:      "E_VIRTUAL_KS_IDXER_NOT_FOUND",
		Description: "Virtual keyspace : Indexer not found «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VIRTUAL_IDX_NOT_FOUND, // 19003
		symbol:      "E_VIRTUAL_IDX_NOT_FOUND",
		Description: "Virtual indexer : Index not found «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VIRTUAL_IDXER_NOT_SUPPORTED, // 19004
		symbol:      "E_VIRTUAL_IDXER_NOT_SUPPORTED",
		Description: "Virtual Indexer : Not supported «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VIRTUAL_IDX_NOT_IMPLEMENTED, // 19005
		symbol:      "E_VIRTUAL_IDX_NOT_IMPLEMENTED",
		Description: "Virtual index : Not yet implemented «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VIRTUAL_IDX_NOT_SUPPORTED, // 19006
		symbol:      "E_VIRTUAL_IDX_NOT_SUPPORTED",
		Description: "Virtual Index : Not supported «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VIRTUAL_SCOPE_NOT_FOUND, // 19007
		symbol:      "E_VIRTUAL_SCOPE_NOT_FOUND",
		Description: "Scope not found in virtual datastore «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VIRTUAL_BUCKET_CREATE_SCOPE, // 19009
		symbol:      "E_VIRTUAL_BUCKET_CREATE_SCOPE",
		Description: "Error while creating scope «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VIRTUAL_BUCKET_DROP_SCOPE, // 19010
		symbol:      "E_VIRTUAL_BUCKET_DROP_SCOPE",
		Description: "Error while dropping scope «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VIRTUAL_KEYSPACE_NOT_FOUND, // 19011
		symbol:      "E_VIRTUAL_KEYSPACE_NOT_FOUND",
		Description: "Keyspace not found in CB datastore: «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VIRTUAL_BUCKET_CREATE_COLLECTION, // 19012
		symbol:      "E_VIRTUAL_BUCKET_CREATE_COLLECTION",
		Description: "Error while creating collection «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_VIRTUAL_BUCKET_DROP_COLLECTION, // 19013
		symbol:      "E_VIRTUAL_BUCKET_DROP_COLLECTION",
		Description: "Error while dropping collection «details»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SEQUENCE_NOT_ENABLED, // 19100
		symbol:      "E_SEQUENCE_NOT_ENABLED",
		Description: "Sequence support is not enabled for «bucket»",
		Reason: []string{
			"An attempt was made to define a sequence in a bucket that lacks a system collection.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SEQUENCE_CREATE, // 19101
		symbol:      "E_SEQUENCE_CREATE",
		Description: "Create failed for sequence «name»",
		Reason: []string{
			"Invalid options were specified in a CREATE SEQUENCE statement.",
			"An error occurred storing the sequence information in the system collection.",
		},
		Action: []string{
			"Correct the sequence options according to the referenced error.",
			"Contact support.",
		},
		IsUser: MAYBE,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SEQUENCE_ALTER, // 19102
		symbol:      "E_SEQUENCE_ALTER",
		Description: "Alter failed for sequence «name»",
		Reason: []string{
			"Invalid options were specified in an ALTER SEQUENCE statement.",
			"An error occurred storing the sequence information in the system collection.",
		},
		Action: []string{
			"Correct the sequence options according to the referenced error.",
			"Contact support.",
		},
		IsUser: MAYBE,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SEQUENCE_DROP, // 19103
		symbol:      "E_SEQUENCE_DROP",
		Description: "Drop failed for sequence «name»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SEQUENCE_INVALID_RANGE, // 19104
		symbol:      "E_SEQUENCE_INVALID_RANGE",
		Description: "Invalid range «min»-«max»",
		Reason: []string{
			"The range specified in the sequence options was invalid.",
		},
		Action: []string{
			"Review the statement and correct the range specification.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SEQUENCE_INVALID_CACHE, // 19105
		symbol:      "E_SEQUENCE_INVALID_CACHE",
		Description: "Invalid cache value «value»",
		Reason: []string{
			"A sequence cache value less than one was specified.",
		},
		Action: []string{
			"Review the statement and correct the cache specification.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SEQUENCE_NOT_FOUND, // 19106
		symbol:      "E_SEQUENCE_NOT_FOUND",
		Description: "Sequence «name» not found",
		Action: []string{
			"Review the statement ensuring the correct sequence is referenced and that it exists.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SEQUENCE, // 19107
		symbol:      "E_SEQUENCE",
		Description: "Error accessing sequence",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SEQUENCE_ALREADY_EXISTS, // 19108
		symbol:      "E_SEQUENCE_ALREADY_EXISTS",
		Description: "Sequence «name» already exists ",
		Reason: []string{
			"An attempt was made to create a sequence with the same name as an existing sequence.",
		},
		Action: []string{
			"Review the statement ensuring a unique name is used.",
			"Revise the statement to include the IF NOT EXISTS syntax.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SEQUENCE_METAKV, // 19109
		symbol:      "E_SEQUENCE_METAKV",
		Description: "Error accessing sequences cache monitor data",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SEQUENCE_INVALID_DATA, // 19110
		symbol:      "E_SEQUENCE_INVALID_DATA",
		Description: "Invalid sequence data",
		Reason: []string{
			"The persisted data for a sequence was invalid.",
		},
		Action: []string{
			"Ensure no user activity has directly altered any data in the system collection.",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SEQUENCE_EXHAUSTED, // 19111
		symbol:      "E_SEQUENCE_EXHAUSTED",
		Description: "Sequence «name» has reached its limit",
		Reason: []string{
			"A sequence was defined to not cycle and was unable to generate further values having reached its defined limit.",
		},
		Action: []string{
			"Review the sequence definition and usage.\nA sequence may be altered to change the limits for generated values.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SEQUENCE_CYCLE, // 19112
		symbol:      "E_SEQUENCE_CYCLE",
		Description: "Cycle failed for sequence «name»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SEQUENCE_INVALID_NAME, // 19113
		symbol:      "E_SEQUENCE_INVALID_NAME",
		Description: "Invalid sequence name «name»",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SEQUENCE_READ_ONLY_REQ, // 19114
		symbol:      "E_SEQUENCE_READ_ONLY_REQ",
		Description: "Sequences cannot be used in read-only requests",
		Reason: []string{
			"A read-only request attempted to use a sequence.\n" +
				"Use of a sequence updates meta-data contravening the read-only request specification.",
		},
		Action: []string{
			"Avoid sequences in read-only requests.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_SEQUENCE_CACHE_SIZE, // 19115
		symbol:      "W_SEQUENCE_CACHE_SIZE",
		Description: "Cache size «size» below recommended minimum",
		Reason: []string{
			"A sequence cache value below the recommended minimum size was specified.\n" +
				"Sequences with smaller caches result in higher I/O and may suffer with increased latency as a result.",
		},
		Action: []string{
			"Review your sequence requirements and adjust the cache size as required.",
		},
		IsUser:    YES,
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SEQUENCE_NAME_PARTS, // 19116
		symbol:      "E_SEQUENCE_NAME_PARTS",
		Description: "Sequence name resolves to «name» - check query_context?",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_SEQUENCE_DROP_ALL, // 19117
		symbol:      "E_SEQUENCE_DROP_ALL",
		Description: "Drop failed for sequences «sequences»",
		Reason: []string{
			"The clean-up operation for sequences belonging to a scope that has been dropped encountered the noted error.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        W_SEQUENCE_NO_PREV_VALUE, // 19118
		symbol:      "W_SEQUENCE_NO_PREV_VALUE",
		Description: "Sequence previous value cannot be accessed before next value generation.",
		Reason: []string{
			"A statement attempted to access the previous sequence value before the first value has been generated for it on the " +
				"Query service node.",
		},
		Action: []string{
			"Review your logic and if necessary ensure that sequences generate values before attempting to access the previous " +
				"value.",
		},
		IsUser:    YES,
		IsWarning: true,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_CREATE_SESSIONS_REQ, // 19200,
		symbol:      "E_NL_CREATE_SESSIONS_REQ",
		Description: "Failed to create a new request to «sessions url»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_SEND_SESSIONS_REQ, // 19201
		symbol:      "E_NL_SEND_SESSIONS_REQ",
		Description: "Failed to send the request to «sessions api» to get JWT",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_SESSIONS_AUTH, // 19202
		symbol:      "E_NL_SESSIONS_AUTH",
		Description: "Authorization failed when establishing natural language session",
		IsUser:      YES,
		Action: []string{
			"Verify the natural language processing credentials supplied in the request.",
			"Create a Couchbase cloud account if necessary.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_SESSIONS_RESP_READ, // 19203
		symbol:      "E_NL_SESSIONS_RESP_READ",
		Description: "Error reading the response from «sessions api»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_SESSIONS_RESP_UNMARSHAL, // 19204
		symbol:      "E_NL_SESSIONS_RESP_UNMARSHAL",
		Description: "Unmarshalling response from «sessions api» failed: ",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_SESSIONS_PARSE_EXPIRE_TIME, // 19205
		symbol:      "E_NL_SESSIONS_PARSE_EXPIRE_TIME",
		Description: "Error parsing \"expiresAt\": «expiresAt»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_PROMPT_SCHEMA_MARSHAL, // 19206
		symbol:      "E_NL_PROMPT_SCHEMA_MARSHAL",
		Description: "Error marshalling schema information for prompt:",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_CHATCOMPLETIONS_PROMPT_MARSHAL, // 19207
		symbol:      "E_NL_CHATCOMPLETIONS_PROMPT_MARSHAL",
		Description: "Error marshalling prompt for chat completions API request",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_SEND_CHATCOMPLETIONS_REQ, // 19208
		symbol:      "E_NL_SEND_CHATCOMPLETIONS_REQ",
		Description: "Couldn't send chat completions request to «chat completions api»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_CHATCOMPLETIONS_REQ_FAILED, // 19209
		symbol:      "E_NL_CHATCOMPLETIONS_REQ_FAILED",
		Description: "Chat completions request failed with status «http-status-code»",
		IsUser:      YES,
		Reason: []string{
			"Status 429: Rate limited. The natural language processing facilities are limiting the number of requests.",
			"Status 404: Unauthorized. Authorization for natural language processing failed.",
		},
		Action: []string{
			"Status 429: Retry later.",
			"Status 404: Verify the credentials provided for natural language processing.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_CHATCOMPLETIONS_READ_RESP_STREAM, // 19210
		symbol:      "E_NL_CHATCOMPLETIONS_READ_RESP_STREAM",
		Description: "Error reading response stream from chat completion API «url»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_CHATCOMPLETIONS_RESP_UNMARSHAL, // 19211
		symbol:      "E_NL_CHATCOMPLETIONS_RESP_UNMARSHAL",
		Description: "Error unmarshalling chat completions response",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_ERR_CHATCOMPLETIONS_RESP, // 19212
		symbol:      "E_NL_ERR_CHATCOMPLETIONS_RESP",
		Description: "LLM processing failed",
		IsUser:      MAYBE,
		Reason: []string{
			"\"natural\" parameter is not a valid prompt or doesn't prompt for a SELECT query.",
			"The natural language statement is not a valid prompt or doesn't prompt for a SELECT query.",
		},
		Action: []string{
			"Review the embedded \"reason\" field for more information on the failure.",
			"Try rewording your request or revising the keyspace information provided.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_MISSING_NL_PARAM, // 19213
		symbol:      "E_NL_MISSING_NL_PARAM",
		Description: "Natural Language request expects «param» request parameter to be set",
		IsUser:      YES,
		Reason: []string{
			"\"natural_cred\", \"natural_context\" and \"natural_orgid\" parameters are required when sending a request " +
				"using the \"natural\" parameter",
			"The options \"cred\", \"keyspaces\" and \"orgid\" are required in the statement when the \"natural_\" parameters " +
				"are not supplied.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_FAIL_GENERATED_STMT, // 19214
		symbol:      "E_NL_FAIL_GENERATED_STMT",
		Description: "Statement generation failed: «failure»",
		IsUser:      MAYBE,
		Reason: []string{
			"Syntax error in generated statement.",
			"LLM returned an empty response",
		},
		Action: []string{
			"Examine the «failure», adjust and re-submit as a direct statement execution request if possible.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_CONTEXT, // 19215
		symbol:      "E_NL_CONTEXT",
		Description: "Error in keyspace list provided for natural language processing",
		IsUser:      YES,
		Reason: []string{
			"Validation of the \"natural_context\" parameter failed for the reason specified.",
			"Validation of the \"keyspaces\" option failed for the reason specified.",
		},
		Action: []string{
			"Revise the \"natural_context\" parameter.",
			"Revise the \"keyspaces\" option.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_PROMPT_INFER, // 19216
		symbol:      "E_NL_PROMPT_INFER",
		Description: "Schema inferring failed for keyspace «keyspace»",
		IsUser:      YES,
		Reason: []string{
			"A keyspace the list of keyspaces passed for natural language processing doesn't exist in the cluster.",
		},
		Action: []string{
			"Ensure all keyspaces provided for natural language processing exist.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_ORG_NOT_FOUND, // 19217
		symbol:      "E_NL_ORG_NOT_FOUND",
		Description: "Organization «organization» not found",
		IsUser:      YES,
		Reason: []string{
			"The organisation specified in the \"natural_orgid\" parameter was not found by the chat completions API.",
			"The organisation specified in the \"orgid\" option was not found by the chat completions API.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:   E_NL_ORG_UNAUTH, // 19218
		symbol: "E_NL_ORG_UNAUTH",
		Description: "Access to organisation «organization» is not authorized " +
			"or collison in JWT refresh with an external client",
		IsUser: MAYBE,
		Reason: []string{
			"Organisation exists but the natural language processing credentials lack permission to access it.",
			"Concurrent JWT refresh by external clients.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_CREATE_CHATCOMPLETIONS_REQ, // 19219
		symbol:      "E_NL_CREATE_CHATCOMPLETIONS_REQ",
		Description: "Failed to create a new request to «chat completions api»",
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_TOO_MANY_WAITERS, // 19220
		symbol:      "E_NL_TOO_MANY_WAITERS",
		Description: "Too many waiters, dropping the request",
		Reason: []string{
			"Natural language requests are throttled as there are no more free slots in the waiting queue",
		},
		Action: []string{
			"Retry the request later.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_TIMEOUT, // 19221
		symbol:      "E_NL_TIMEOUT",
		Description: "Timed out waiting to be processed.",
		Reason: []string{
			"Natural language request timed out waiting to be processed",
		},
		Action: []string{
			"Retry the request later.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_REQ_FEAT_DISABLED, // 19222
		symbol:      "E_NL_REQ_FEAT_DISABLED",
		Description: "Natural language request processing is disabled.",
		Reason: []string{
			"The processing of natural language requests has been disabled.",
		},
		Action: []string{
			"Enable natural language request processing before submitting a natural language request.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_NL_TOO_MANY_KEYSPACES, // 19223
		symbol:      "E_NL_TOO_MANY_KEYSPACES",
		Description: "Too many keyspaces specified.",
		IsUser:      YES,
		Reason: []string{
			"The \"natural_context\" parameter specifies more than the maximum permitted number of keyspaces.",
			"The \"keyspaces\" option specifies more than the maximum permitted number of keyspaces.",
		},
		Action: []string{
			"Revise the \"natural_context\" parameter or \"keyspaces\" option.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_AUS_NOT_SUPPORTED, // 20000
		symbol:      "E_AUS_NOT_SUPPORTED",
		Description: "Auto Update Statistics is not supported in Community Edition. It is an enterprise level feature.",
		Reason: []string{
			"An Auto Update Statistics related operation was attempted on a Community Edition Couchbase cluster.",
		},
		Action: []string{
			"Consult the documentation for the feature you are trying to use.",
		},
		AppliesTo: []string{
			"Community Edition",
		},
	},
	{
		Code:   E_AUS_NOT_INITIALIZED, // 20001
		symbol: "E_AUS_NOT_INITIALIZED",
		Description: "Auto Update Statistics is not initialized for the node. It is only available on clusters migrated to " +
			"a supported version.",
		Reason: []string{
			"An Auto Update Statistics related operation was attempted on a cluster that is not fully migrated to a version " +
				"that supports it.",
		},
		Action: []string{
			"Migrate the Couchbase cluster to a version that supports Auto Update Statistics.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_AUS_STORAGE, // 20002
		symbol:      "E_AUS_STORAGE",
		Description: "Error accessing Auto Update Statistics information from storage.",
		Action: []string{
			"Retry the operation again. Or contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_AUS_INVALID_DOCUMENT_SCHEMA, // 20003
		symbol:      "E_AUS_INVALID_DOCUMENT_SCHEMA",
		Description: "Invalid schema detected in the Auto Update Statistics settings document.",
		Reason: []string{
			"The schema validation check failed when an attempt was made to INSERT/UPSERT/UPDATE a document in system:aus " +
				"or system:aus_settings.",
		},
		Action: []string{
			"Consult the documentation on the valid schema for Auto Update Statistics settings documents.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_AUS_SETTINGS_ENCODING, // 20004
		symbol:      "E_AUS_SETTINGS_ENCODING",
		Description: "Error «action» Automatic Update Statistics settings document.",
		Action: []string{
			"Retry the operation again. Or contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_AUS_STORAGE_INVALID_KEY, // 20005
		symbol:      "E_AUS_STORAGE_INVALID_KEY",
		Description: "Invalid document key «key» for Auto Update Statistics settings document.",
		Reason: []string{
			"An invalid document key was detected when an operation or SQL++ statement was run against system:aus or " +
				"system:aus_settings.",
		},
		Action: []string{
			"Consult the documentation on the valid document key format for Auto Update Statistics settings documents.",
		},
		IsUser: YES,
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_AUS_SCHEDULING, // 20006
		symbol:      "E_AUS_SCHEDULING",
		Description: "Error during scheduling the Auto Update Statistics task.",
		Reason: []string{
			"An error occurred during scheduling the Auto Update Statistics task.",
		},
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_AUS_TASK, // 20007
		symbol:      "E_AUS_TASK",
		Description: "Error during «operation» of Auto Update Statistics task.",
		Action: []string{
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_AUS_EVALUATION_PHASE, // 20008
		symbol:      "E_AUS_EVALUATION_PHASE",
		Description: "Auto Update Statistics task's Evaluation phase for «keyspace» encountered an error.",
		Action: []string{
			"Observe if the error occurs again in future runs of the Auto Update Statistics task. " +
				"If it occurs frequently, contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_AUS_UPDATE_PHASE, // 20009
		symbol:      "E_AUS_UPDATE_PHASE",
		Description: "Auto Update Statistics task's Update phase for «keyspace» encountered an error.",
		Action: []string{
			"Observe if the error occurs again in future runs of the Auto Update Statistics task. " +
				"If it occurs frequently, contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_AUS_TASK_NOT_STARTED, // 20010
		symbol:      "E_AUS_TASK_NOT_STARTED",
		Description: "The Auto Update Statistics task was not started due to existing load on the node.",
		Reason: []string{
			"The Auto Update Statistics task was not started as the load factor of the Query node was too high to handle " +
				"the additional workload of the task.",
		},
		Action: []string{
			"Observe if the error occurs again in future runs of the Auto Update Statistics task. If it occurs frequently, " +
				"the set schedule for Auto Update Statistics might not be suitable for the workload.",
			"Approach revising the schedule. ",
			"Contact support.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
	{
		Code:        E_AUS_TASK_TIMEOUT, // 20011
		symbol:      "E_AUS_TASK_TIMEOUT",
		Description: "Scheduled window of the Auto Update Statistics task exceeded.",
		Action: []string{
			"Observe if the error occurs again in future runs of the Auto Update Statistics task. If it occurs frequently, " +
				"the set scheduled window for Auto Update Statistics might not be long enough. Approach revising the start and " +
				"end time of the schedule.",
		},
		AppliesTo: []string{
			"Server",
		},
	},
}
