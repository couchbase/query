//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package n1ql

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
)

var namespaces map[string]interface{}

func SetNamespaces(ns map[string]interface{}) {
	namespaces = ns
}

func ParseStatement(input string) (algebra.Statement, error) {
	return ParseStatement2(input, "default", "")
}

func ParseStatement2(input string, namespace string, queryContext string) (algebra.Statement, error) {
	input = strings.TrimSpace(input)
	reader := strings.NewReader(input)
	lex := newLexer(NewLexer(reader))
	lex.parsingStmt = true
	lex.text = input
	lex.namespace = namespace
	lex.queryContext = queryContext
	lex.nex.ResetOffset()
	lex.nex.ReportError(lex.ScannerError)
	doParse(lex)

	if len(lex.errs) > 0 {
		return nil, fmt.Errorf(strings.Join(lex.errs, " \n "))
	} else if lex.stmt == nil {
		return nil, fmt.Errorf("Input was not a statement.")
	} else {
		err := lex.stmt.Formalize()
		if err != nil {
			return nil, err
		}

		lex.stmt.SetParamsCount(lex.paramCount)
		return lex.stmt, nil
	}
}

func ParseExpression(input string) (expression.Expression, error) {
	input = strings.TrimSpace(input)
	reader := strings.NewReader(input)
	lex := newLexer(NewLexer(reader))
	lex.nex.ResetOffset()
	lex.nex.ReportError(lex.ScannerError)
	doParse(lex)

	if len(lex.errs) > 0 {
		return nil, fmt.Errorf(strings.Join(lex.errs, " \n "))
	} else if lex.expr == nil {
		return nil, fmt.Errorf("Input was not an expression.")
	} else {
		return lex.expr, nil
	}
}

func doParse(lex *lexer) {
	defer func() {
		r := recover()
		if r != nil {
			lex.Error(fmt.Sprintf("Error while parsing: %v", r))

			// Log this error
			buf := make([]byte, 2048)
			n := runtime.Stack(buf, false)
			logging.Errorf("Error while parsing: %v %s", r, string(buf[0:n]))
		}
		if !lex.stop {
			lex.nex.Stop()
		}
	}()

	yyParse(lex)
}

type lexer struct {
	nex              *Lexer
	posParam         int
	paramCount       int
	errs             []string
	stmt             algebra.Statement
	expr             expression.Expression
	parsingStmt      bool
	lastScannerError string
	text             string
	offset           int
	namespace        string
	queryContext     string
	hasSaved         bool
	saved            int
	lval             yySymType
	stop             bool
}

func newLexer(nex *Lexer) *lexer {
	return &lexer{
		nex:    nex,
		errs:   make([]string, 0, 16),
		offset: 0,
	}
}

func (this *lexer) Lex(lval *yySymType) int {
	if this.stop {
		return 0
	}

	// if we had peeked, return that peeked token
	if this.hasSaved {
		rv := this.saved
		*lval = this.lval
		this.hasSaved = false
		return rv
	}

	rv := this.nex.Lex(lval)

	// we are going to treat identifiers specially to resolve
	// shift reduce conflicts on namespaces
	if rv != IDENT {
		return rv
	}

	// is it a namespace?
	_, found := namespaces[lval.s]
	if !found {
		return IDENT
	}

	// save the current token value and check the next
	this.hasSaved = true
	oldLval := *lval
	this.saved = this.nex.Lex(lval)
	this.lval = *lval
	*lval = oldLval

	// not a colon, so we have an identifier
	if this.saved != COLON {
		return IDENT
	}

	return NAMESPACE_ID
}

func (this *lexer) Remainder(offset int) string {
	return strings.TrimLeft(this.text[offset:], " \t")
}

func (this *lexer) Error(s string) {
	if s == "syntax error" && this.stop {
		return
	}
	if this.lastScannerError != "" {
		s = s + ": " + this.lastScannerError
		this.lastScannerError = ""
	}
	if len(this.nex.stack) > 0 {
		s = s + " - at " + this.nex.Text()
	} else {
		s = s + " - at end of input"
	}

	this.errs = append(this.errs, s)
}

func (this *lexer) FatalError(s string) int {
	this.stop = true
	this.nex.Stop()
	this.Error(s)
	return 1
}

func (this *lexer) ScannerError(s string) {
	this.lastScannerError = s
}

func (this *lexer) setStatement(stmt algebra.Statement) {
	this.stmt = stmt
}

func (this *lexer) setOffset(offset int) {
	this.offset = offset
}

func (this *lexer) getOffset() int {
	return this.offset
}

func (this *lexer) setExpression(expr expression.Expression) {
	this.expr = expr
}

func (this *lexer) parsingStatement() bool { return this.parsingStmt }

func (this *lexer) getText() string { return this.text }

func (this *lexer) nextParam() int {
	this.posParam++
	this.paramCount++
	return this.posParam
}

func (this *lexer) countParam() {
	this.paramCount++
}

func (this *lexer) Namespace() string {
	return this.namespace
}

func (this *lexer) QueryContext() string {
	return this.queryContext
}
