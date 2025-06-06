//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package n1ql

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
)

var namespaces map[string]interface{}

func SetNamespaces(ns map[string]interface{}) {
	namespaces = ns
}

func ParseStatement(input string) (algebra.Statement, error) {
	return parseStatement(input, "default", "", false, nil)
}

func ParseStatement2(input string, namespace string, queryContext string, args ...logging.Log) (algebra.Statement, error) {
	var l logging.Log
	if len(args) > 0 {
		l = args[0]
	}
	return parseStatement(input, namespace, queryContext, true, l)
}

func parseStatement(input string, namespace string, queryContext string, udfCheck bool, log logging.Log) (
	algebra.Statement, error) {

	input = strings.TrimSpace(input)
	reader := strings.NewReader(input)
	lex := newLexer(NewLexer(reader), log)
	lex.parsingStmt = true
	lex.text = input
	lex.namespace = namespace
	lex.queryContext = queryContext
	lex.udfCheck = udfCheck
	lex.nex.ResetOffset()
	lex.nex.ReportError(lex.ScannerError)
	doParse(lex)

	if len(lex.errs) > 0 {
		return nil, errors.NewParseSyntaxError(lex.errs, "")
	} else if lex.stmt == nil {
		return nil, errors.NewParseInvalidInput("statement")
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
	return parseExpression(input, false)
}

func ParseExpressionUdf(input string) (expression.Expression, error) {
	return parseExpression(input, true)
}

func parseExpression(input string, udfExpr bool) (expression.Expression, error) {
	input = strings.TrimSpace(input)
	reader := strings.NewReader(input)
	lex := newLexer(NewLexer(reader), nil)
	lex.nex.ResetOffset()
	lex.text = input
	lex.udfExpr = udfExpr
	lex.nex.ReportError(lex.ScannerError)
	doParse(lex)

	if len(lex.errs) > 0 {
		return nil, errors.NewParseSyntaxError(lex.errs, "")
	} else if lex.expr == nil {
		return nil, errors.NewParseInvalidInput("expression")
	} else {
		return lex.expr, nil
	}
}

func ParseOptimHints(input string) *algebra.OptimHints {
	input = strings.TrimSpace(input)
	reader := strings.NewReader(input)
	lex := newLexer(NewLexer(reader), nil)
	lex.text = input
	lex.nex.ResetOffset()
	lex.nex.ReportError(lex.ScannerError)
	doParse(lex)

	if len(lex.errs) > 0 {
		/* ignore the '+' at the beginning */
		return algebra.InvalidOptimHints(input[1:], strings.Join(lex.errs, "\n"))
	}
	return lex.optimHints
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
	udfCheck               bool
	hasSaved               bool
	saved                  int
	lval                   yySymType
	stop                   bool
	udfExpr                bool
	optimHints             *algebra.OptimHints
	log                    logging.Log
}

func newLexer(nex *Lexer, log logging.Log) *lexer {
	if log == nil {
		log = logging.NULL_LOG
	}
	return &lexer{
		nex:    nex,
		errs:   make([]string, 0, 16),
		offset: 0,
		log:    log,
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
	if rv != IDENT && rv != DEFAULT {
		return rv
	} else if rv == DEFAULT {
		// treat it similarly as IDENT
		lval.s = this.nex.Text()
	}

	// is it a namespace?
	tok := lval.s
	if len(tok) > 1 && tok[0] == '#' {
		tok = tok[1:]
	}
	_, found := namespaces[tok]
	if !found {
		return rv
	} else {
		lval.s = tok
	}

	// save the current token value and check the next
	this.hasSaved = true
	oldLval := *lval
	this.saved = this.nex.Lex(lval)
	this.lval = *lval
	*lval = oldLval

	// not a colon, so we have an identifier
	if this.saved != COLON {
		return rv
	}

	if datastore.GetSystemstore() != nil && tok == datastore.GetSystemstore().Id() {
		return SYSTEM
	}

	return NAMESPACE_ID
}

func (this *lexer) Remainder(offset int) string {
	if offset < 0 {
		if len(this.nex.stack) > 0 {
			return this.nex.stack[len(this.nex.stack)-1].s
		}
		offset = this.nex.curOffset
		if offset < 0 || offset >= len(this.text) {
			return ""
		}
		return strings.TrimLeft(this.text[this.nex.curOffset:], " \t")
	}
	return strings.TrimLeft(this.text[offset:], " \t")
}

func (this *lexer) ErrorWithContext(s string, line int, column int) {
	if line != 0 {
		ectx := errors.NewErrorContext(line, column).Error()
		if ectx != "" {
			if strings.HasSuffix(s, ".") {
				s = s[:len(s)-1] + ectx + "."
			} else {
				s += ectx
			}
		}
	}
	this.Error(s)
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
		// add context if the error doesn't already contain context
		if ctx, _ := regexp.MatchString("(near line [0-9]+, column [0-9]+)", s); !ctx {
			s = s + this.ErrorContext()
			if len(this.nex.stack) > 0 {
				if isLexerToken(strings.ToUpper(this.nex.Text())) {
					s = s + " (reserved word)"
				}
			}
		}
	}
	this.errs = append(this.errs, s)
}

func isLexerToken(t string) bool {
	for _, tok := range yyToknames {
		if tok == t {
			return true
		}
	}
	return false
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

const _MAX_CONTEXT_LEN = 20

func (this *lexer) getContextFor(contextLine, contextColumn int) string {
	if len(this.text) == 0 {
		return ""
	}
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

	chrs := make([]int, _MAX_CONTEXT_LEN)
	ch := 0
	col := 0
	for eoff < len(this.text) && col < contextColumn {
		r, s := utf8.DecodeRuneInString(this.text[eoff:])
		if unicode.IsSpace(r) {
			chrs[ch%len(chrs)] = -1
		} else {
			chrs[ch%len(chrs)] = eoff
		}
		ch++
		eoff += s
		col++
	}
	if eoff == 0 {
		return ""
	}
	lim := ch + len(chrs)
	if ch < len(chrs) {
		lim = ch
		ch = 0
	}
	for ch < lim {
		s := chrs[ch%len(chrs)]
		if s > 0 && s != line {
			return "..." + this.text[s:eoff]
		} else if s == 0 || s == line {
			return this.text[s:eoff]
		}
		ch++
	}
	return ""
}

func (this *lexer) FatalError(s string, line int, column int) int {
	this.stop = true
	this.nex.Stop()
	if s != "" {
		this.ErrorWithContext(s, line, column)
	}
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

func (this *lexer) parsingUdfExpression() bool { return !this.parsingStmt && this.udfExpr }

func (this *lexer) getSubString(s, e int) string {
	return strings.TrimSpace(this.text[s:e])
}

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

func (this *lexer) UdfCheck() bool {
	return this.udfCheck
}

func (this *lexer) setOptimHints(optimHints *algebra.OptimHints) {
	this.optimHints = optimHints
}
