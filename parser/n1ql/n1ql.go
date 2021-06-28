//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

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
		return nil, fmt.Errorf("%s", strings.Join(lex.errs, " \n "))
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
		return nil, fmt.Errorf("%s", strings.Join(lex.errs, " \n "))
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
	nex                    *Lexer
	posParam               int
	paramCount             int
	errs                   []string
	stmt                   algebra.Statement
	expr                   expression.Expression
	parsingStmt            bool
	lastScannerError       string
	text                   string
	offset                 int
	namespace              string
	createFuncQueryContext string
	queryContext           string
	hasSaved               bool
	saved                  int
	lval                   yySymType
	stop                   bool
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
	if strings.HasPrefix(s, "syntax error") {
		s = s + this.ErrorContext()
	}
	this.errs = append(this.errs, s)
}

func (this *lexer) ErrorContext() string {
	s := ""
	if len(this.nex.stack) > 0 {
		ctx := this.getContextFor(this.nex.Line(), this.nex.Column())
		if len(ctx) > 0 {
			s = fmt.Sprintf(" - line %d, column %d, near '%s', at: ", this.nex.Line()+1, this.nex.Column()+1, ctx) +
				this.nex.Text()
		} else {
			s = fmt.Sprintf(" - line %d, column %d, at: ", this.nex.Line()+1, this.nex.Column()+1) + this.nex.Text()
		}
	} else {
		s = " - at end of input"
	}
	return s
}

func (this *lexer) getContextFor(contextLine, contextColumn int) string {
	line := 0
	eoff := 0
	for eoff = 0; eoff < len(this.text); eoff++ {
		if line >= contextLine {
			break
		}
		if this.text[eoff] == '\n' {
			line++
		}
	}
	line = eoff
	eoff += contextColumn - 1
	if eoff > len(this.text) {
		eoff = len(this.text) - 1
	}
	for ; eoff > line; eoff-- {
		if this.text[eoff] != ' ' && this.text[eoff] != '\t' {
			break
		}
	}
	eoff++
	soff := eoff - 20
	if line > soff {
		soff = line
	}
	if soff < eoff {
		return this.text[soff:eoff]
	}
	return ""
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

func (this *lexer) PushQueryContext(queryContext string) {
	this.createFuncQueryContext = queryContext
}

func (this *lexer) PopQueryContext() {
	this.createFuncQueryContext = ""
}

func (this *lexer) QueryContext() string {
	if this.createFuncQueryContext != "" {
		return this.createFuncQueryContext
	}
	return this.queryContext
}
