//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/value"
)

/*
Represents the Create function ddl statement. Type CreateFunction is
a struct that contains fields mapping to each clause in the
create function statement. The fields refer to the function name and
and function body
*/
type CreateFunction struct {
	statementBase

	name         functions.FunctionName `json:"name"`
	body         functions.FunctionBody `json:"body"`
	replace      bool                   `json:"replace"`
	failIfExists bool                   `json:"fail_if_exists"`
}

/*
The function NewCreateFunction returns a pointer to the
CreateFunction struct with the input argument values as fields.
*/
func NewCreateFunction(name functions.FunctionName, body functions.FunctionBody, replace bool, failIfExists bool) *CreateFunction {
	rv := &CreateFunction{
		name:         name,
		body:         body,
		replace:      replace,
		failIfExists: failIfExists,
	}

	rv.stmt = rv
	return rv
}

func (this *CreateFunction) Name() functions.FunctionName {
	return this.name
}

func (this *CreateFunction) Body() functions.FunctionBody {
	return this.body
}

func (this *CreateFunction) Replace() bool {
	return this.replace
}

func (this *CreateFunction) FailIfExists() bool {
	return this.failIfExists
}

/*
It calls the VisitCreateFunction method by passing
in the receiver and returns the interface. It is a
visitor pattern.
*/
func (this *CreateFunction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateFunction(this)
}

/*
Returns nil.
*/
func (this *CreateFunction) Signature() value.Value {
	return nil
}

/*
Returns nil.
*/
func (this *CreateFunction) Formalize() error {
	return nil
}

/*
This method maps all the constituent clauses, but here non have expressions
*/
func (this *CreateFunction) MapExpressions(mapper expression.Mapper) (err error) {
	return
}

/*
Return expr from the create function statement.
*/
func (this *CreateFunction) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *CreateFunction) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	priv := functions.GetPrivilege(this.name, this.body)
	privs.Add(this.name.Key(), priv, auth.PRIV_PROPS_NONE)

	return privs, nil
}

func (this *CreateFunction) Type() string {
	return "CREATE_FUNCTION"
}

func (this *CreateFunction) String() string {

	var s strings.Builder
	s.WriteString("CREATE ")
	if this.replace {
		s.WriteString("OR REPLACE ")
	}
	s.WriteString("FUNCTION ")
	if !this.failIfExists {
		s.WriteString("IF NOT EXISTS ")
	}
	s.WriteString(this.name.ProtectedKey())
	funcbody := map[string]interface{}{}
	this.body.Body(funcbody)
	parameters := funcbody["parameters"]
	s.WriteString("(")
	if parameters != nil {
		if paramList, ok := parameters.([]value.Value); ok {
			for i, p := range paramList {
				if i > 0 {
					s.WriteString(", ")
				}
				s.WriteString(p.ToString())
			}
		}
	} else {
		s.WriteString("...")
	}
	s.WriteString(")")

	language, ok := funcbody["#language"].(string)
	if ok {
		switch language {
		case "inline":
			s.WriteString(" LANGUAGE ")
			s.WriteString(strings.ToUpper(language))
			expression := funcbody["expression"].(string)
			s.WriteString(" AS ")
			s.WriteString(expression)
		case "javascript":
			s.WriteString(" LANGUAGE ")
			s.WriteString(strings.ToUpper(language))
			if t, ok := funcbody["text"].(string); ok {
				s.WriteString(" AS \"")
				s.WriteString(t)
				s.WriteString("\"")
			} else {
				object := funcbody["object"].(string)
				library := funcbody["library"].(string)
				s.WriteString(" AS \"")
				s.WriteString(object)
				s.WriteString("\" AT \"")
				s.WriteString(library)
				s.WriteString("\"")
			}
		case "golang":
			s.WriteString(" LANGUAGE ")
			s.WriteString(strings.ToUpper(language))
			object := funcbody["object"].(string)
			library := funcbody["library"].(string)
			s.WriteString(" AS \"")
			s.WriteString(object)
			s.WriteString("\" AT \"")
			s.WriteString(library)
			s.WriteString("\"")
		default:
			// should not happen
			writeErrBody(UNEXPECTED_LANGUAGE, &s, language)
		}
	} else {
		// should not happen
		writeErrBody(MISSING_LANGUAGE, &s)
	}

	return s.String()
}

const (
	MISSING_LANGUAGE    = "missing language field in function body"
	UNEXPECTED_LANGUAGE = "unexpected language in function body, language: %s"
)

func writeErrBody(msg string, s *strings.Builder, args ...interface{}) {
	s.WriteString("{ ")
	errmap := map[string]string{
		"error": fmt.Sprintf(msg, args...),
	}
	errMsg, err := json.Marshal(errmap)
	if err != nil {
		s.WriteString(" }")
	} else {
		s.Write(errMsg)
		s.WriteString(" }")
	}
}
