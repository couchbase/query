//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package errors

import "fmt"

const (
	SESSIONS_IKEY       = "sessions_req"
	PROMPT_IKEY         = "prompt"
	CHATCOMPLTIONS_IKEY = "chatcompletions_req"
	NLPARAM_IKEY        = "missing_parameters"
	FAIL_IKEY           = "failed_to_generated_stmt"
	NLCONTEXT_IKEY      = "natural_context"
	RATE_LIMIT          = "rate_limit"
	SERVE_NATURAL       = "service_natural_request"
)

var naturalErrMap = map[ErrorCode][2]string{

	E_NL_CREATE_SESSIONS_REQ:              {SESSIONS_IKEY, "Failed to create a new request to %v"},
	E_NL_SEND_SESSIONS_REQ:                {SESSIONS_IKEY, "Failed to send the request to %v to get JWT"},
	E_NL_SESSIONS_AUTH:                    {SESSIONS_IKEY, "Authorization failed when establishing natural language session"},
	E_NL_SESSIONS_RESP_READ:               {SESSIONS_IKEY, "Error reading the response from %v"},
	E_NL_SESSIONS_RESP_UNMARSHAL:          {SESSIONS_IKEY, "Unmarshalling response from %v failed: "},
	E_NL_SESSIONS_PARSE_EXPIRE_TIME:       {SESSIONS_IKEY, "Error parsing \"expiresAt\": %v "},
	E_NL_PROMPT_SCHEMA_MARSHAL:            {PROMPT_IKEY, "Error marshalling schema information for prompt:"},
	E_NL_PROMPT_INFER:                     {PROMPT_IKEY, "Schema inferring failed for keyspace %v"},
	E_NL_CHATCOMPLETIONS_PROMPT_MARSHAL:   {CHATCOMPLTIONS_IKEY, "Error marshalling prompt for chat completions API request"},
	E_NL_SEND_CHATCOMPLETIONS_REQ:         {CHATCOMPLTIONS_IKEY, "Couldn't send chat completions request to %v"},
	E_NL_CHATCOMPLETIONS_REQ_FAILED:       {CHATCOMPLTIONS_IKEY, "Chat completions request failed with status %v"},
	E_NL_CHATCOMPLETIONS_READ_RESP_STREAM: {CHATCOMPLTIONS_IKEY, "Error reading response stream from chat completion API %v"},
	E_NL_CHATCOMPLETIONS_RESP_UNMARSHAL:   {CHATCOMPLTIONS_IKEY, "Error unmarshalling chat completions response"},
	E_NL_ERR_CHATCOMPLETIONS_RESP:         {CHATCOMPLTIONS_IKEY, "LLM processing failed"},
	E_NL_MISSING_NL_PARAM:                 {NLPARAM_IKEY, "Natural Language request expects %s request parameter to be set"},
	E_NL_FAIL_GENERATED_STMT:              {FAIL_IKEY, "Statement generation failed: %v"},
	E_NL_CONTEXT:                          {NLCONTEXT_IKEY, "Error in keyspace list provided for natural language processing"},
	E_NL_ORG_NOT_FOUND:                    {CHATCOMPLTIONS_IKEY, "Organization \"%v\" not found"},
	E_NL_ORG_UNAUTH: {CHATCOMPLTIONS_IKEY, "Access to organisation '%v' is not authorized " +
		"or collison in JWT refresh with an external client"},
	E_NL_CREATE_CHATCOMPLETIONS_REQ: {CHATCOMPLTIONS_IKEY, "Failed to create a new request to \"%v\""},
	E_NL_TOO_MANY_WAITERS:           {RATE_LIMIT, "Too many waiters, dropping the request"},
	E_NL_TIMEOUT:                    {RATE_LIMIT, "Timed out waiting to be processed."},
	E_NL_REQ_FEAT_DISABLED:          {SERVE_NATURAL, "Natural language request processing is disabled."},
	E_NL_TOO_MANY_KEYSPACES:         {NLCONTEXT_IKEY, "Too many keyspaces specified."},
}

func NewNaturalLanguageRequestError(code ErrorCode, args ...interface{}) Error {

	e := &err{level: EXCEPTION, ICode: code, IKey: "natural." + naturalErrMap[code][0],
		InternalMsg: naturalErrMap[code][1], InternalCaller: CallerN(1)}
	var fmtArgs []interface{}
	for _, a := range args {
		switch a := a.(type) {
		case string:
			fmtArgs = append(fmtArgs, a)
		case int:
			fmtArgs = append(fmtArgs, a)
		case Error:
			e.cause = a
		case error:
			e.cause = a
		case nil:
			// ignore
		default:
			panic(fmt.Sprintf("invalid argument (%T) to NewNaturalLanguageRequestError", a))
		}
	}
	if len(fmtArgs) > 0 {
		e.InternalMsg = fmt.Sprintf(e.InternalMsg, fmtArgs...)
	}
	return e
}
