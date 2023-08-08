//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package n1ql

import "math"
import "strconv"
import "github.com/couchbase/clog"
import (
	"bufio"
	"io"
	"strings"
)

type frame struct {
	i            int
	s            string
	line, column int
}

type Lexer struct {
	// The lexer runs in its own goroutine, and communicates via channel 'ch'.
	ch      chan frame
	ch_stop chan bool
	// We record the level of nesting because the action could return, and a
	// subsequent call expects to pick up where it left off. In other words,
	// we're simulating a coroutine.
	// TODO: Support a channel-based variant that compatible with Go's yacc.
	stack []frame
	stale bool

	// The 'l' and 'c' fields were added for
	// https://github.com/wagerlabs/docker/blob/65694e801a7b80930961d70c69cba9f2465459be/buildfile.nex
	// Now used to record last seen line & column from the stack.
	l, c int

	parseResult interface{}

	// The following line makes it easy for scripts to insert fields in the
	// generated code.
	curOffset   int
	reportError func(what string)
	// [NEX_END_OF_LEXER_STRUCT]
}

// NewLexerWithInit creates a new Lexer object, runs the given callback on it,
// then returns it.
func NewLexerWithInit(in io.Reader, initFun func(*Lexer)) *Lexer {
	yylex := new(Lexer)
	if initFun != nil {
		initFun(yylex)
	}
	yylex.ch = make(chan frame)
	yylex.ch_stop = make(chan bool, 1)
	var scan func(in *bufio.Reader, ch chan frame, ch_stop chan bool, family []dfa, line, column int)
	scan = func(in *bufio.Reader, ch chan frame, ch_stop chan bool, family []dfa, line, column int) {
		// Index of DFA and length of highest-precedence match so far.
		matchi, matchn := 0, -1
		var buf []rune
		n := 0
		checkAccept := func(i int, st int) bool {
			// Higher precedence match? DFAs are run in parallel, so matchn is at most len(buf), hence we may omit the length equality check.
			if family[i].acc[st] && (matchn < n || matchi > i) {
				matchi, matchn = i, n
				return true
			}
			return false
		}
		stateCap := len(family)
		if stateCap == 0 {
			stateCap = 1
		}
		state := make([][2]int, 0, stateCap)
		for i := 0; i < len(family); i++ {
			mark := make([]bool, len(family[i].startf))
			// Every DFA starts at state 0.
			st := 0
			for {
				state = append(state, [2]int{i, st})
				mark[st] = true
				// As we're at the start of input, follow all ^ transitions and append to our list of start states.
				st = family[i].startf[st]
				if -1 == st || mark[st] {
					break
				}
				// We only check for a match after at least one transition.
				checkAccept(i, st)
			}
		}
		atEOF := false
		stopped := false

	loop:
		for {
			if n == len(buf) && !atEOF {
				r, _, err := in.ReadRune()
				switch err {
				case io.EOF:
					atEOF = true
				case nil:
					buf = append(buf, r)
				default:
					panic(err)
				}
			}
			if !atEOF {
				r := buf[n]
				n++
				d := 0
				for _, x := range state {
					x[1] = family[x[0]].f[x[1]](r)
					if -1 == x[1] {
						continue
					}
					state[d] = x
					d++
					checkAccept(x[0], x[1])
				}
				state = state[:d]
			} else {
			dollar: // Handle $.
				for _, x := range state {
					mark := make([]bool, len(family[x[0]].endf))
					for {
						mark[x[1]] = true
						x[1] = family[x[0]].endf[x[1]]
						if -1 == x[1] || mark[x[1]] {
							break
						}
						if checkAccept(x[0], x[1]) {
							// Unlike before, we can break off the search. Now that we're at the end, there's no need to maintain the state of each DFA.
							break dollar
						}
					}
				}
				state = state[:0]
			}

			if len(state) == 0 {
				lcUpdate := func(r rune) {
					if r == '\n' {
						line++
						column = 0
					} else {
						column++
					}
				}
				// All DFAs stuck. Return last match if it exists, otherwise advance by one rune and restart all DFAs.
				if matchn == -1 {
					if len(buf) == 0 { // This can only happen at the end of input.
						break
					}
					lcUpdate(buf[0])
					buf = buf[1:]
				} else {
					text := string(buf[:matchn])
					buf = buf[matchn:]
					matchn = -1

					select {
					case <-ch_stop:
						stopped = true
						break loop
					default:
					}
					select {
					case ch <- frame{matchi, text, line, column}:
					case <-ch_stop:
						stopped = true
						break loop
					}
					if len(family[matchi].nest) > 0 {
						scan(bufio.NewReader(strings.NewReader(text)), ch, ch_stop, family[matchi].nest, line, column)
					}
					if atEOF {
						break
					}
					for _, r := range text {
						lcUpdate(r)
					}
				}
				n = 0
				if len(family) > cap(state) {
					state = make([][2]int, 0, len(family))
				}
				for i := 0; i < len(family); i++ {
					state = append(state, [2]int{i, 0})
				}
			}
		}
		select {
		case <-ch_stop:
			stopped = true
		default:
		}
		if !stopped {
			select {
			case ch <- frame{-1, "", line, column}:

			case <-ch_stop:
			}
		}
	}
	go scan(bufio.NewReader(in), yylex.ch, yylex.ch_stop, dfas, 0, 0)
	return yylex
}

type dfa struct {
	acc          []bool           // Accepting states.
	f            []func(rune) int // Transitions.
	startf, endf []int            // Transitions at start and end of input.
	nest         []dfa
}

var dfas = []dfa{
	// \"(\\\"|\\[^\"]|[^\"\\])*\"?
	{[]bool{false, true, true, false, true, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 34:
				return 1
			case 92:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 34:
				return 2
			case 92:
				return 3
			}
			return 4
		},
		func(r rune) int {
			switch r {
			case 34:
				return -1
			case 92:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 34:
				return 5
			case 92:
				return 6
			}
			return 6
		},
		func(r rune) int {
			switch r {
			case 34:
				return 2
			case 92:
				return 3
			}
			return 4
		},
		func(r rune) int {
			switch r {
			case 34:
				return 2
			case 92:
				return 3
			}
			return 4
		},
		func(r rune) int {
			switch r {
			case 34:
				return 2
			case 92:
				return 3
			}
			return 4
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// '(\'\'|\\'|\\[^']|[^'\\])*'?
	{[]bool{false, true, true, false, true, true, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 39:
				return 1
			case 92:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 2
			case 92:
				return 3
			}
			return 4
		},
		func(r rune) int {
			switch r {
			case 39:
				return 7
			case 92:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 5
			case 92:
				return 6
			}
			return 6
		},
		func(r rune) int {
			switch r {
			case 39:
				return 2
			case 92:
				return 3
			}
			return 4
		},
		func(r rune) int {
			switch r {
			case 39:
				return 2
			case 92:
				return 3
			}
			return 4
		},
		func(r rune) int {
			switch r {
			case 39:
				return 2
			case 92:
				return 3
			}
			return 4
		},
		func(r rune) int {
			switch r {
			case 39:
				return 2
			case 92:
				return 3
			}
			return 4
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// `((\`\`|\\`)|\\[^`]|[^`\\])*`?i
	{[]bool{false, false, false, false, true, false, false, true, false, false}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 92:
				return -1
			case 96:
				return 1
			case 105:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 92:
				return 2
			case 96:
				return 3
			case 105:
				return 4
			}
			return 5
		},
		func(r rune) int {
			switch r {
			case 92:
				return 8
			case 96:
				return 9
			case 105:
				return 8
			}
			return 8
		},
		func(r rune) int {
			switch r {
			case 92:
				return -1
			case 96:
				return 6
			case 105:
				return 7
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 92:
				return 2
			case 96:
				return 3
			case 105:
				return 4
			}
			return 5
		},
		func(r rune) int {
			switch r {
			case 92:
				return 2
			case 96:
				return 3
			case 105:
				return 4
			}
			return 5
		},
		func(r rune) int {
			switch r {
			case 92:
				return 2
			case 96:
				return 3
			case 105:
				return 4
			}
			return 5
		},
		func(r rune) int {
			switch r {
			case 92:
				return -1
			case 96:
				return -1
			case 105:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 92:
				return 2
			case 96:
				return 3
			case 105:
				return 4
			}
			return 5
		},
		func(r rune) int {
			switch r {
			case 92:
				return 2
			case 96:
				return 3
			case 105:
				return 4
			}
			return 5
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// `((\`\`|\\`)|\\[^`]|[^`\\])*`?
	{[]bool{false, true, false, true, true, true, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 92:
				return -1
			case 96:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 92:
				return 2
			case 96:
				return 3
			}
			return 4
		},
		func(r rune) int {
			switch r {
			case 92:
				return 6
			case 96:
				return 7
			}
			return 6
		},
		func(r rune) int {
			switch r {
			case 92:
				return -1
			case 96:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 92:
				return 2
			case 96:
				return 3
			}
			return 4
		},
		func(r rune) int {
			switch r {
			case 92:
				return 2
			case 96:
				return 3
			}
			return 4
		},
		func(r rune) int {
			switch r {
			case 92:
				return 2
			case 96:
				return 3
			}
			return 4
		},
		func(r rune) int {
			switch r {
			case 92:
				return 2
			case 96:
				return 3
			}
			return 4
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// (0|[1-9][0-9]*)\.[0-9]+([eE][+\-]?[0-9]+)?
	{[]bool{false, false, false, false, false, true, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return 1
			case 69:
				return -1
			case 101:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return -1
			case 49 <= r && r <= 57:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return 3
			case 48:
				return -1
			case 69:
				return -1
			case 101:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return -1
			case 49 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return 3
			case 48:
				return 4
			case 69:
				return -1
			case 101:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return 4
			case 49 <= r && r <= 57:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return 5
			case 69:
				return -1
			case 101:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return 5
			case 49 <= r && r <= 57:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return 3
			case 48:
				return 4
			case 69:
				return -1
			case 101:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return 4
			case 49 <= r && r <= 57:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return 5
			case 69:
				return 6
			case 101:
				return 6
			}
			switch {
			case 48 <= r && r <= 48:
				return 5
			case 49 <= r && r <= 57:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return 7
			case 45:
				return 7
			case 46:
				return -1
			case 48:
				return 8
			case 69:
				return -1
			case 101:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return 8
			case 49 <= r && r <= 57:
				return 8
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return 8
			case 69:
				return -1
			case 101:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return 8
			case 49 <= r && r <= 57:
				return 8
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return 8
			case 69:
				return -1
			case 101:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return 8
			case 49 <= r && r <= 57:
				return 8
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// (0|[1-9][0-9]*)[eE][+\-]?[0-9]+
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 48:
				return 1
			case 69:
				return -1
			case 101:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return -1
			case 49 <= r && r <= 57:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 48:
				return -1
			case 69:
				return 4
			case 101:
				return 4
			}
			switch {
			case 48 <= r && r <= 48:
				return -1
			case 49 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 48:
				return 3
			case 69:
				return 4
			case 101:
				return 4
			}
			switch {
			case 48 <= r && r <= 48:
				return 3
			case 49 <= r && r <= 57:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 48:
				return 3
			case 69:
				return 4
			case 101:
				return 4
			}
			switch {
			case 48 <= r && r <= 48:
				return 3
			case 49 <= r && r <= 57:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return 5
			case 45:
				return 5
			case 48:
				return 6
			case 69:
				return -1
			case 101:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return 6
			case 49 <= r && r <= 57:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 48:
				return 6
			case 69:
				return -1
			case 101:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return 6
			case 49 <= r && r <= 57:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 48:
				return 6
			case 69:
				return -1
			case 101:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return 6
			case 49 <= r && r <= 57:
				return 6
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// 0|[1-9][0-9]*
	{[]bool{false, true, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 48:
				return 1
			}
			switch {
			case 48 <= r && r <= 48:
				return -1
			case 49 <= r && r <= 57:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return -1
			case 49 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return 3
			}
			switch {
			case 48 <= r && r <= 48:
				return 3
			case 49 <= r && r <= 57:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return 3
			}
			switch {
			case 48 <= r && r <= 48:
				return 3
			case 49 <= r && r <= 57:
				return 3
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [0-9][0-9]*[a-zA-Z_][0-9a-zA-Z_]*
	{[]bool{false, false, true, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 95:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 1
			case 65 <= r && r <= 90:
				return -1
			case 97 <= r && r <= 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 95:
				return 2
			}
			switch {
			case 48 <= r && r <= 57:
				return 3
			case 65 <= r && r <= 90:
				return 2
			case 97 <= r && r <= 122:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 95:
				return 4
			}
			switch {
			case 48 <= r && r <= 57:
				return 4
			case 65 <= r && r <= 90:
				return 4
			case 97 <= r && r <= 122:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 95:
				return 2
			}
			switch {
			case 48 <= r && r <= 57:
				return 3
			case 65 <= r && r <= 90:
				return 2
			case 97 <= r && r <= 122:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 95:
				return 4
			}
			switch {
			case 48 <= r && r <= 57:
				return 4
			case 65 <= r && r <= 90:
				return 4
			case 97 <= r && r <= 122:
				return 4
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// \/\*\+[^*]?(([^*\/])|(\*+[^\/])|([^*]\/))*\*+\/
	{[]bool{false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 42:
				return -1
			case 43:
				return -1
			case 47:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 42:
				return 2
			case 43:
				return -1
			case 47:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 42:
				return -1
			case 43:
				return 3
			case 47:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 42:
				return 4
			case 43:
				return 5
			case 47:
				return 6
			}
			return 5
		},
		func(r rune) int {
			switch r {
			case 42:
				return 9
			case 43:
				return 10
			case 47:
				return 11
			}
			return 10
		},
		func(r rune) int {
			switch r {
			case 42:
				return 4
			case 43:
				return 7
			case 47:
				return 8
			}
			return 7
		},
		func(r rune) int {
			switch r {
			case 42:
				return 4
			case 43:
				return 7
			case 47:
				return 8
			}
			return 7
		},
		func(r rune) int {
			switch r {
			case 42:
				return 4
			case 43:
				return 7
			case 47:
				return 8
			}
			return 7
		},
		func(r rune) int {
			switch r {
			case 42:
				return 4
			case 43:
				return 7
			case 47:
				return 8
			}
			return 7
		},
		func(r rune) int {
			switch r {
			case 42:
				return 9
			case 43:
				return 14
			case 47:
				return 15
			}
			return 14
		},
		func(r rune) int {
			switch r {
			case 42:
				return 4
			case 43:
				return 7
			case 47:
				return 12
			}
			return 7
		},
		func(r rune) int {
			switch r {
			case 42:
				return -1
			case 43:
				return -1
			case 47:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 42:
				return -1
			case 43:
				return -1
			case 47:
				return 13
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 42:
				return 4
			case 43:
				return 7
			case 47:
				return 12
			}
			return 7
		},
		func(r rune) int {
			switch r {
			case 42:
				return 4
			case 43:
				return 7
			case 47:
				return 8
			}
			return 7
		},
		func(r rune) int {
			switch r {
			case 42:
				return -1
			case 43:
				return -1
			case 47:
				return 13
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// --\+[^\n\r]*
	{[]bool{false, false, false, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 13:
				return -1
			case 43:
				return -1
			case 45:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 13:
				return -1
			case 43:
				return -1
			case 45:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 13:
				return -1
			case 43:
				return 3
			case 45:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 13:
				return -1
			case 43:
				return 4
			case 45:
				return 4
			}
			return 4
		},
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 13:
				return -1
			case 43:
				return 4
			case 45:
				return 4
			}
			return 4
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// \/\*[^*]?(([^*\/])|(\*+[^\/])|([^*]\/))*\*+\/
	{[]bool{false, false, false, false, false, false, false, false, false, true, false, false, false, true, false}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 42:
				return -1
			case 47:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 42:
				return 2
			case 47:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 42:
				return 3
			case 47:
				return 4
			}
			return 5
		},
		func(r rune) int {
			switch r {
			case 42:
				return 8
			case 47:
				return 9
			}
			return 10
		},
		func(r rune) int {
			switch r {
			case 42:
				return 3
			case 47:
				return 6
			}
			return 7
		},
		func(r rune) int {
			switch r {
			case 42:
				return 3
			case 47:
				return 6
			}
			return 7
		},
		func(r rune) int {
			switch r {
			case 42:
				return 3
			case 47:
				return 6
			}
			return 7
		},
		func(r rune) int {
			switch r {
			case 42:
				return 3
			case 47:
				return 6
			}
			return 7
		},
		func(r rune) int {
			switch r {
			case 42:
				return 8
			case 47:
				return 13
			}
			return 14
		},
		func(r rune) int {
			switch r {
			case 42:
				return -1
			case 47:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 42:
				return 3
			case 47:
				return 11
			}
			return 7
		},
		func(r rune) int {
			switch r {
			case 42:
				return -1
			case 47:
				return 12
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 42:
				return 3
			case 47:
				return 11
			}
			return 7
		},
		func(r rune) int {
			switch r {
			case 42:
				return -1
			case 47:
				return 12
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 42:
				return 3
			case 47:
				return 6
			}
			return 7
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// --[^\n\r]*
	{[]bool{false, false, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 13:
				return -1
			case 45:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 13:
				return -1
			case 45:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 13:
				return -1
			case 45:
				return 3
			}
			return 3
		},
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 13:
				return -1
			case 45:
				return 3
			}
			return 3
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [ \t\n\r\f]+
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 9:
				return 1
			case 10:
				return 1
			case 12:
				return 1
			case 13:
				return 1
			case 32:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 9:
				return 1
			case 10:
				return 1
			case 12:
				return 1
			case 13:
				return 1
			case 32:
				return 1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \.
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 46:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 46:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \+
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 43:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// -
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 45:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 45:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \*
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 42:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 42:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \/
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 47:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 47:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// %
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 37:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 37:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \^
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 94:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 94:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \=\=
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 61:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 61:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 61:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// \=
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 61:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 61:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \!\=
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 33:
				return 1
			case 61:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 61:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 61:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// \<\>
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 60:
				return 1
			case 62:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 60:
				return -1
			case 62:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 60:
				return -1
			case 62:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// \<
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 60:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 60:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \<\=
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 60:
				return 1
			case 61:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 60:
				return -1
			case 61:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 60:
				return -1
			case 61:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// \>
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 62:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 62:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \>\=
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 61:
				return -1
			case 62:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 61:
				return 2
			case 62:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 61:
				return -1
			case 62:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// \|\|
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 124:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 124:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 124:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// \(
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 40:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 40:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \)
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 41:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 41:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \{
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 123:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 123:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \}
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 125:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 125:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \,
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 44:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 44:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \:
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 58:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 58:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \[
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 91:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 91:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \]
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 93:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 93:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \]i
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 93:
				return 1
			case 105:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 93:
				return -1
			case 105:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 93:
				return -1
			case 105:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// ;
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 59:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 59:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \!
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 33:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// [_][iI][nN][dD][eE][xX][_][cC][oO][nN][dD][iI][tT][iI][oO][nN]
	{[]bool{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 88:
				return -1
			case 95:
				return 1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return 2
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 88:
				return -1
			case 95:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return 2
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return 3
			case 79:
				return -1
			case 84:
				return -1
			case 88:
				return -1
			case 95:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return 3
			case 111:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return 4
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 88:
				return -1
			case 95:
				return -1
			case 99:
				return -1
			case 100:
				return 4
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 5
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 88:
				return -1
			case 95:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 5
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 88:
				return 6
			case 95:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 120:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 88:
				return -1
			case 95:
				return 7
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 8
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 88:
				return -1
			case 95:
				return -1
			case 99:
				return 8
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return 9
			case 84:
				return -1
			case 88:
				return -1
			case 95:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return 9
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return 10
			case 79:
				return -1
			case 84:
				return -1
			case 88:
				return -1
			case 95:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return 10
			case 111:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return 11
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 88:
				return -1
			case 95:
				return -1
			case 99:
				return -1
			case 100:
				return 11
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return 12
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 88:
				return -1
			case 95:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return 12
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return 13
			case 88:
				return -1
			case 95:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return 13
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return 14
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 88:
				return -1
			case 95:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return 14
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return 15
			case 84:
				return -1
			case 88:
				return -1
			case 95:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return 15
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return 16
			case 79:
				return -1
			case 84:
				return -1
			case 88:
				return -1
			case 95:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return 16
			case 111:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 88:
				return -1
			case 95:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [_][iI][nN][dD][eE][xX][_][kK][eE][yY]
	{[]bool{false, false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 78:
				return -1
			case 88:
				return -1
			case 89:
				return -1
			case 95:
				return 1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 110:
				return -1
			case 120:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return 2
			case 75:
				return -1
			case 78:
				return -1
			case 88:
				return -1
			case 89:
				return -1
			case 95:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return 2
			case 107:
				return -1
			case 110:
				return -1
			case 120:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 78:
				return 3
			case 88:
				return -1
			case 89:
				return -1
			case 95:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 110:
				return 3
			case 120:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return 4
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 78:
				return -1
			case 88:
				return -1
			case 89:
				return -1
			case 95:
				return -1
			case 100:
				return 4
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 110:
				return -1
			case 120:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return 5
			case 73:
				return -1
			case 75:
				return -1
			case 78:
				return -1
			case 88:
				return -1
			case 89:
				return -1
			case 95:
				return -1
			case 100:
				return -1
			case 101:
				return 5
			case 105:
				return -1
			case 107:
				return -1
			case 110:
				return -1
			case 120:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 78:
				return -1
			case 88:
				return 6
			case 89:
				return -1
			case 95:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 110:
				return -1
			case 120:
				return 6
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 78:
				return -1
			case 88:
				return -1
			case 89:
				return -1
			case 95:
				return 7
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 110:
				return -1
			case 120:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return 8
			case 78:
				return -1
			case 88:
				return -1
			case 89:
				return -1
			case 95:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return 8
			case 110:
				return -1
			case 120:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return 9
			case 73:
				return -1
			case 75:
				return -1
			case 78:
				return -1
			case 88:
				return -1
			case 89:
				return -1
			case 95:
				return -1
			case 100:
				return -1
			case 101:
				return 9
			case 105:
				return -1
			case 107:
				return -1
			case 110:
				return -1
			case 120:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 78:
				return -1
			case 88:
				return -1
			case 89:
				return 10
			case 95:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 110:
				return -1
			case 120:
				return -1
			case 121:
				return 10
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 78:
				return -1
			case 88:
				return -1
			case 89:
				return -1
			case 95:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 110:
				return -1
			case 120:
				return -1
			case 121:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [aA][dD][vV][iI][sS][eE]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return 1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 83:
				return -1
			case 86:
				return -1
			case 97:
				return 1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return 2
			case 69:
				return -1
			case 73:
				return -1
			case 83:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 100:
				return 2
			case 101:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 83:
				return -1
			case 86:
				return 3
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 118:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return 4
			case 83:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return 4
			case 115:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 83:
				return 5
			case 86:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 115:
				return 5
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return 6
			case 73:
				return -1
			case 83:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return 6
			case 105:
				return -1
			case 115:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 83:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [aA][lL][lL]
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return 1
			case 76:
				return -1
			case 97:
				return 1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 76:
				return 2
			case 97:
				return -1
			case 108:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 76:
				return 3
			case 97:
				return -1
			case 108:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 76:
				return -1
			case 97:
				return -1
			case 108:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [aA][lL][tT][eE][rR]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return 1
			case 69:
				return -1
			case 76:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return 1
			case 101:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return 2
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return 2
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 82:
				return -1
			case 84:
				return 3
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 116:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 4
			case 76:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return 4
			case 108:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 82:
				return 5
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 114:
				return 5
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [aA][nN][aA][lL][yY][zZ][eE]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return 1
			case 69:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 89:
				return -1
			case 90:
				return -1
			case 97:
				return 1
			case 101:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 78:
				return 2
			case 89:
				return -1
			case 90:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 110:
				return 2
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 3
			case 69:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 89:
				return -1
			case 90:
				return -1
			case 97:
				return 3
			case 101:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return 4
			case 78:
				return -1
			case 89:
				return -1
			case 90:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return 4
			case 110:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 89:
				return 5
			case 90:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 121:
				return 5
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 89:
				return -1
			case 90:
				return 6
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 121:
				return -1
			case 122:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 7
			case 76:
				return -1
			case 78:
				return -1
			case 89:
				return -1
			case 90:
				return -1
			case 97:
				return -1
			case 101:
				return 7
			case 108:
				return -1
			case 110:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 89:
				return -1
			case 90:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [aA][nN][dD]
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return 1
			case 68:
				return -1
			case 78:
				return -1
			case 97:
				return 1
			case 100:
				return -1
			case 110:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 78:
				return 2
			case 97:
				return -1
			case 100:
				return -1
			case 110:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return 3
			case 78:
				return -1
			case 97:
				return -1
			case 100:
				return 3
			case 110:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 78:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 110:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [aA][nN][yY]
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return 1
			case 78:
				return -1
			case 89:
				return -1
			case 97:
				return 1
			case 110:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 78:
				return 2
			case 89:
				return -1
			case 97:
				return -1
			case 110:
				return 2
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 78:
				return -1
			case 89:
				return 3
			case 97:
				return -1
			case 110:
				return -1
			case 121:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 78:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 110:
				return -1
			case 121:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [aA][rR][rR][aA][yY]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return 1
			case 82:
				return -1
			case 89:
				return -1
			case 97:
				return 1
			case 114:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 82:
				return 2
			case 89:
				return -1
			case 97:
				return -1
			case 114:
				return 2
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 82:
				return 3
			case 89:
				return -1
			case 97:
				return -1
			case 114:
				return 3
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 4
			case 82:
				return -1
			case 89:
				return -1
			case 97:
				return 4
			case 114:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 82:
				return -1
			case 89:
				return 5
			case 97:
				return -1
			case 114:
				return -1
			case 121:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 82:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 114:
				return -1
			case 121:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [aA][sS]
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return 1
			case 83:
				return -1
			case 97:
				return 1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 83:
				return 2
			case 97:
				return -1
			case 115:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// [aA][sS][cC]
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return 1
			case 67:
				return -1
			case 83:
				return -1
			case 97:
				return 1
			case 99:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 83:
				return 2
			case 97:
				return -1
			case 99:
				return -1
			case 115:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return 3
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return 3
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [aA][tT]
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return 1
			case 84:
				return -1
			case 97:
				return 1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 84:
				return 2
			case 97:
				return -1
			case 116:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// [bB][eE][gG][iI][nN]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 66:
				return 1
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 98:
				return 1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return 2
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 98:
				return -1
			case 101:
				return 2
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return -1
			case 71:
				return 3
			case 73:
				return -1
			case 78:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 103:
				return 3
			case 105:
				return -1
			case 110:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return 4
			case 78:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return 4
			case 110:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return 5
			case 98:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [bB][eE][tT][wW][eE][eE][nN]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 66:
				return 1
			case 69:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 87:
				return -1
			case 98:
				return 1
			case 101:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return 2
			case 78:
				return -1
			case 84:
				return -1
			case 87:
				return -1
			case 98:
				return -1
			case 101:
				return 2
			case 110:
				return -1
			case 116:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 84:
				return 3
			case 87:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 116:
				return 3
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 87:
				return 4
			case 98:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			case 119:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return 5
			case 78:
				return -1
			case 84:
				return -1
			case 87:
				return -1
			case 98:
				return -1
			case 101:
				return 5
			case 110:
				return -1
			case 116:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return 6
			case 78:
				return -1
			case 84:
				return -1
			case 87:
				return -1
			case 98:
				return -1
			case 101:
				return 6
			case 110:
				return -1
			case 116:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return -1
			case 78:
				return 7
			case 84:
				return -1
			case 87:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 110:
				return 7
			case 116:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 87:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			case 119:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [bB][iI][nN][aA][rR][yY]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return 1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 98:
				return 1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 73:
				return 2
			case 78:
				return -1
			case 82:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 105:
				return 2
			case 110:
				return -1
			case 114:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 73:
				return -1
			case 78:
				return 3
			case 82:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 105:
				return -1
			case 110:
				return 3
			case 114:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 4
			case 66:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 89:
				return -1
			case 97:
				return 4
			case 98:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return 5
			case 89:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return 5
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 89:
				return 6
			case 97:
				return -1
			case 98:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 121:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 121:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [bB][oO][oO][lL][eE][aA][nN]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return 1
			case 69:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 97:
				return -1
			case 98:
				return 1
			case 101:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return 2
			case 97:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return 3
			case 97:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 76:
				return 4
			case 78:
				return -1
			case 79:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 108:
				return 4
			case 110:
				return -1
			case 111:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return 5
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 101:
				return 5
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 6
			case 66:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 97:
				return 6
			case 98:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 78:
				return 7
			case 79:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 110:
				return 7
			case 111:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [bB][rR][eE][aA][kK]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return 1
			case 69:
				return -1
			case 75:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 98:
				return 1
			case 101:
				return -1
			case 107:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 75:
				return -1
			case 82:
				return 2
			case 97:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 107:
				return -1
			case 114:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return 3
			case 75:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 101:
				return 3
			case 107:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 4
			case 66:
				return -1
			case 69:
				return -1
			case 75:
				return -1
			case 82:
				return -1
			case 97:
				return 4
			case 98:
				return -1
			case 101:
				return -1
			case 107:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 75:
				return 5
			case 82:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 107:
				return 5
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 75:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 107:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [bB][uU][cC][kK][eE][tT]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 66:
				return 1
			case 67:
				return -1
			case 69:
				return -1
			case 75:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 98:
				return 1
			case 99:
				return -1
			case 101:
				return -1
			case 107:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 75:
				return -1
			case 84:
				return -1
			case 85:
				return 2
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 107:
				return -1
			case 116:
				return -1
			case 117:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return 3
			case 69:
				return -1
			case 75:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 98:
				return -1
			case 99:
				return 3
			case 101:
				return -1
			case 107:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 75:
				return 4
			case 84:
				return -1
			case 85:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 107:
				return 4
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return 5
			case 75:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return 5
			case 107:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 75:
				return -1
			case 84:
				return 6
			case 85:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 107:
				return -1
			case 116:
				return 6
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 75:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 107:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [bB][uU][iI][lL][dD]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 66:
				return 1
			case 68:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 98:
				return 1
			case 100:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 68:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 85:
				return 2
			case 98:
				return -1
			case 100:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 117:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 68:
				return -1
			case 73:
				return 3
			case 76:
				return -1
			case 85:
				return -1
			case 98:
				return -1
			case 100:
				return -1
			case 105:
				return 3
			case 108:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 68:
				return -1
			case 73:
				return -1
			case 76:
				return 4
			case 85:
				return -1
			case 98:
				return -1
			case 100:
				return -1
			case 105:
				return -1
			case 108:
				return 4
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 68:
				return 5
			case 73:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 98:
				return -1
			case 100:
				return 5
			case 105:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 68:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 98:
				return -1
			case 100:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [bB][yY]
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 66:
				return 1
			case 89:
				return -1
			case 98:
				return 1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 89:
				return 2
			case 98:
				return -1
			case 121:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 89:
				return -1
			case 98:
				return -1
			case 121:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// [cC][aA][lL][lL]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return 1
			case 76:
				return -1
			case 97:
				return -1
			case 99:
				return 1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 67:
				return -1
			case 76:
				return -1
			case 97:
				return 2
			case 99:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 76:
				return 3
			case 97:
				return -1
			case 99:
				return -1
			case 108:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 76:
				return 4
			case 97:
				return -1
			case 99:
				return -1
			case 108:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 76:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 108:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [cC][aA][sS][eE]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return 1
			case 69:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return 1
			case 101:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 67:
				return -1
			case 69:
				return -1
			case 83:
				return -1
			case 97:
				return 2
			case 99:
				return -1
			case 101:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 83:
				return 3
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 115:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return 4
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return 4
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [cC][aA][sS][tT]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return 1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return 1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 67:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return 2
			case 99:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 83:
				return 3
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 115:
				return 3
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 83:
				return -1
			case 84:
				return 4
			case 97:
				return -1
			case 99:
				return -1
			case 115:
				return -1
			case 116:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [cC][lL][uU][sS][tT][eE][rR]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return 1
			case 69:
				return -1
			case 76:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return 1
			case 101:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return 2
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return 2
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return 3
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 82:
				return -1
			case 83:
				return 4
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return 4
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 5
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 5
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 6
			case 76:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return 6
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 82:
				return 7
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 114:
				return 7
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [cC][oO][lL][lL][aA][tT][eE]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return 1
			case 69:
				return -1
			case 76:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return 1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 79:
				return 2
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return 2
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return 3
			case 79:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return 3
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return 4
			case 79:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return 4
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 5
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 97:
				return 5
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 79:
				return -1
			case 84:
				return 6
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 116:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return 7
			case 76:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return 7
			case 108:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [cC][oO][lL][lL][eE][cC][tT][iI][oO][nN]
	{[]bool{false, false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return 1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return 1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return 2
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return 2
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return 3
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return 3
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return 4
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return 4
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 5
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return 5
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 6
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return 6
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return 7
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return 7
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return 8
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return 8
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return 9
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return 9
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return 10
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return 10
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [cC][oO][mM][mM][iI][tT]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return 1
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return 1
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return 2
			case 84:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return 2
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 73:
				return -1
			case 77:
				return 3
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 109:
				return 3
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 73:
				return -1
			case 77:
				return 4
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 109:
				return 4
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 73:
				return 5
			case 77:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 105:
				return 5
			case 109:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 84:
				return 6
			case 99:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 116:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [cC][oO][mM][mM][iI][tT][tT][eE][dD]
	{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return 1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return 1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return 2
			case 84:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return 2
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 77:
				return 3
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 109:
				return 3
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 77:
				return 4
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 109:
				return 4
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return 5
			case 77:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return 5
			case 109:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 84:
				return 6
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 116:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 84:
				return 7
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 116:
				return 7
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 8
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 8
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return 9
			case 69:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 100:
				return 9
			case 101:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [cC][oO][nN][nN][eE][cC][tT]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return 1
			case 69:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return 1
			case 101:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 79:
				return 2
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 111:
				return 2
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return 3
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return 3
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return 4
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return 4
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 5
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return 5
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 6
			case 69:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return 6
			case 101:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return 7
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return 7
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [cC][oO][nN][tT][iI][nN][uU][eE]
	{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return 1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return 1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return 2
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return 2
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return 3
			case 79:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return 3
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return 4
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return 4
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return 5
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return 5
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return 6
			case 79:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return 6
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 85:
				return 7
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return 7
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 8
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return 8
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [cC][oO][rR][rR][eE][lL][aA][tT][eE][dD]
	{[]bool{false, false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return 1
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return 1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 79:
				return 2
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return 2
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 79:
				return -1
			case 82:
				return 3
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 114:
				return 3
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 79:
				return -1
			case 82:
				return 4
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 114:
				return 4
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 5
			case 76:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 5
			case 108:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return 6
			case 79:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return 6
			case 111:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 7
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return 7
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 84:
				return 8
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 116:
				return 8
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 9
			case 76:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 9
			case 108:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return 10
			case 69:
				return -1
			case 76:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return 10
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [cC][oO][vV][eE][rR]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return 1
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 99:
				return 1
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 79:
				return 2
			case 82:
				return -1
			case 86:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 111:
				return 2
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 86:
				return 3
			case 99:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 118:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 4
			case 79:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 99:
				return -1
			case 101:
				return 4
			case 111:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return 5
			case 86:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return 5
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [cC][rR][eE][aA][tT][eE]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return 1
			case 69:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return 1
			case 101:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 82:
				return 2
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 114:
				return 2
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return 3
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return 3
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 4
			case 67:
				return -1
			case 69:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return 4
			case 99:
				return -1
			case 101:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 82:
				return -1
			case 84:
				return 5
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 114:
				return -1
			case 116:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return 6
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return 6
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [cC][uU][rR][rR][eE][nN][tT]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return 1
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return 1
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return 2
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return 3
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return 3
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return 4
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return 4
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 5
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return 5
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return 6
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return 6
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return 7
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return 7
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [cC][yY][cC][lL][eE]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return 1
			case 69:
				return -1
			case 76:
				return -1
			case 89:
				return -1
			case 99:
				return 1
			case 101:
				return -1
			case 108:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 89:
				return 2
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 121:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 3
			case 69:
				return -1
			case 76:
				return -1
			case 89:
				return -1
			case 99:
				return 3
			case 101:
				return -1
			case 108:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return 4
			case 89:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return 4
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 5
			case 76:
				return -1
			case 89:
				return -1
			case 99:
				return -1
			case 101:
				return 5
			case 108:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 89:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 121:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [dD][aA][tT][aA][bB][aA][sS][eE]
	{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 68:
				return 1
			case 69:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 100:
				return 1
			case 101:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 66:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return 2
			case 98:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 83:
				return -1
			case 84:
				return 3
			case 97:
				return -1
			case 98:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 115:
				return -1
			case 116:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 4
			case 66:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return 4
			case 98:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return 5
			case 68:
				return -1
			case 69:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 98:
				return 5
			case 100:
				return -1
			case 101:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 6
			case 66:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return 6
			case 98:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 83:
				return 7
			case 84:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 115:
				return 7
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 68:
				return -1
			case 69:
				return 8
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 100:
				return -1
			case 101:
				return 8
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [dD][aA][tT][aA][sS][eE][tT]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return 1
			case 69:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 100:
				return 1
			case 101:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 68:
				return -1
			case 69:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return 2
			case 100:
				return -1
			case 101:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 83:
				return -1
			case 84:
				return 3
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 115:
				return -1
			case 116:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 4
			case 68:
				return -1
			case 69:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return 4
			case 100:
				return -1
			case 101:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 83:
				return 5
			case 84:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 115:
				return 5
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return 6
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return 6
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 83:
				return -1
			case 84:
				return 7
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 115:
				return -1
			case 116:
				return 7
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [dD][aA][tT][aA][sS][tT][oO][rR][eE]
	{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return 1
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 100:
				return 1
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 68:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return 2
			case 100:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 3
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 4
			case 68:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return 4
			case 100:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return 5
			case 84:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return 5
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 6
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 79:
				return 7
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 111:
				return 7
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return 8
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return 8
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return 9
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return 9
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [dD][eE][cC][lL][aA][rR][eE]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return 1
			case 69:
				return -1
			case 76:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return 1
			case 101:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 2
			case 76:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 2
			case 108:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return 3
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 99:
				return 3
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return 4
			case 82:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return 4
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 5
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 82:
				return -1
			case 97:
				return 5
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 82:
				return 6
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 114:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 7
			case 76:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 7
			case 108:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [dD][eE][cC][rR][eE][mM][eE][nN][tT]
	{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return 1
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 100:
				return 1
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 2
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 2
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 3
			case 68:
				return -1
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 99:
				return 3
			case 100:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return 4
			case 84:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return 4
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 5
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 5
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 77:
				return 6
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 109:
				return 6
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 7
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 7
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return 8
			case 82:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return 8
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return 9
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return 9
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [dD][eE][fF][aA][uU][lL][tT]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return 1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 100:
				return 1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return 2
			case 70:
				return -1
			case 76:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return 2
			case 102:
				return -1
			case 108:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 70:
				return 3
			case 76:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return 3
			case 108:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 4
			case 68:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return 4
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 84:
				return -1
			case 85:
				return 5
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 116:
				return -1
			case 117:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return 6
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return 6
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 84:
				return 7
			case 85:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 116:
				return 7
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [dD][eE][lL][eE][tT][eE]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 68:
				return 1
			case 69:
				return -1
			case 76:
				return -1
			case 84:
				return -1
			case 100:
				return 1
			case 101:
				return -1
			case 108:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return 2
			case 76:
				return -1
			case 84:
				return -1
			case 100:
				return -1
			case 101:
				return 2
			case 108:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return 3
			case 84:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return 3
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return 4
			case 76:
				return -1
			case 84:
				return -1
			case 100:
				return -1
			case 101:
				return 4
			case 108:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 84:
				return 5
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 116:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return 6
			case 76:
				return -1
			case 84:
				return -1
			case 100:
				return -1
			case 101:
				return 6
			case 108:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 84:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [dD][eE][rR][iI][vV][eE][dD]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 68:
				return 1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 100:
				return 1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return 2
			case 73:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 100:
				return -1
			case 101:
				return 2
			case 105:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return 3
			case 86:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return 3
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return 4
			case 82:
				return -1
			case 86:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return 4
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 86:
				return 5
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 118:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return 6
			case 73:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 100:
				return -1
			case 101:
				return 6
			case 105:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return 7
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 100:
				return 7
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [dD][eE][sS][cC]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return 1
			case 69:
				return -1
			case 83:
				return -1
			case 99:
				return -1
			case 100:
				return 1
			case 101:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 2
			case 83:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 2
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 83:
				return 3
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 115:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 4
			case 68:
				return -1
			case 69:
				return -1
			case 83:
				return -1
			case 99:
				return 4
			case 100:
				return -1
			case 101:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 83:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [dD][eE][sS][cC][rR][iI][bB][eE]
	{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 68:
				return 1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return 1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 2
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 2
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return 3
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return 4
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 98:
				return -1
			case 99:
				return 4
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return 5
			case 83:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return 5
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return 6
			case 82:
				return -1
			case 83:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return 6
			case 114:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return 7
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 98:
				return 7
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 8
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 8
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [dD][iI][sS][tT][iI][nN][cC][tT]
	{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return 1
			case 73:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 100:
				return 1
			case 105:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 73:
				return 2
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 105:
				return 2
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 83:
				return 3
			case 84:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 115:
				return 3
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return 4
			case 99:
				return -1
			case 100:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 73:
				return 5
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 105:
				return 5
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 73:
				return -1
			case 78:
				return 6
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 105:
				return -1
			case 110:
				return 6
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 7
			case 68:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return 7
			case 100:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return 8
			case 99:
				return -1
			case 100:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return 8
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [dD][oO]
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 68:
				return 1
			case 79:
				return -1
			case 100:
				return 1
			case 111:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 79:
				return 2
			case 100:
				return -1
			case 111:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 79:
				return -1
			case 100:
				return -1
			case 111:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// [dD][rR][oO][pP]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 68:
				return 1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 100:
				return 1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return 2
			case 100:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 79:
				return 3
			case 80:
				return -1
			case 82:
				return -1
			case 100:
				return -1
			case 111:
				return 3
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 79:
				return -1
			case 80:
				return 4
			case 82:
				return -1
			case 100:
				return -1
			case 111:
				return -1
			case 112:
				return 4
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 100:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [eE][aA][cC][hH]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return 1
			case 72:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return 1
			case 104:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 67:
				return -1
			case 69:
				return -1
			case 72:
				return -1
			case 97:
				return 2
			case 99:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return 3
			case 69:
				return -1
			case 72:
				return -1
			case 97:
				return -1
			case 99:
				return 3
			case 101:
				return -1
			case 104:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 72:
				return 4
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 104:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 72:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [eE][lL][eE][mM][eE][nN][tT]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return 1
			case 76:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 101:
				return 1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 76:
				return 2
			case 77:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 108:
				return 2
			case 109:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 3
			case 76:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 101:
				return 3
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 76:
				return -1
			case 77:
				return 4
			case 78:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 109:
				return 4
			case 110:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 5
			case 76:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 101:
				return 5
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 78:
				return 6
			case 84:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return 6
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 84:
				return 7
			case 101:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 116:
				return 7
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [eE][lL][sS][eE]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return 1
			case 76:
				return -1
			case 83:
				return -1
			case 101:
				return 1
			case 108:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 76:
				return 2
			case 83:
				return -1
			case 101:
				return -1
			case 108:
				return 2
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 76:
				return -1
			case 83:
				return 3
			case 101:
				return -1
			case 108:
				return -1
			case 115:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 4
			case 76:
				return -1
			case 83:
				return -1
			case 101:
				return 4
			case 108:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 76:
				return -1
			case 83:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [eE][nN][dD]
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return 1
			case 78:
				return -1
			case 100:
				return -1
			case 101:
				return 1
			case 110:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return 2
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return 3
			case 69:
				return -1
			case 78:
				return -1
			case 100:
				return 3
			case 101:
				return -1
			case 110:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [eE][sS][cC][aA][pP][eE]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return 1
			case 80:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return 1
			case 112:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 80:
				return -1
			case 83:
				return 2
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 115:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return 3
			case 69:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return 3
			case 101:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 4
			case 67:
				return -1
			case 69:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 97:
				return 4
			case 99:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 80:
				return 5
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 112:
				return 5
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return 6
			case 80:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return 6
			case 112:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [eE][vV][eE][rR][yY]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return 1
			case 82:
				return -1
			case 86:
				return -1
			case 89:
				return -1
			case 101:
				return 1
			case 114:
				return -1
			case 118:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 82:
				return -1
			case 86:
				return 2
			case 89:
				return -1
			case 101:
				return -1
			case 114:
				return -1
			case 118:
				return 2
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 3
			case 82:
				return -1
			case 86:
				return -1
			case 89:
				return -1
			case 101:
				return 3
			case 114:
				return -1
			case 118:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 82:
				return 4
			case 86:
				return -1
			case 89:
				return -1
			case 101:
				return -1
			case 114:
				return 4
			case 118:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 89:
				return 5
			case 101:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			case 121:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 89:
				return -1
			case 101:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			case 121:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [eE][xX][cC][eE][pP][tT]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 1
			case 80:
				return -1
			case 84:
				return -1
			case 88:
				return -1
			case 99:
				return -1
			case 101:
				return 1
			case 112:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 80:
				return -1
			case 84:
				return -1
			case 88:
				return 2
			case 99:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 116:
				return -1
			case 120:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 3
			case 69:
				return -1
			case 80:
				return -1
			case 84:
				return -1
			case 88:
				return -1
			case 99:
				return 3
			case 101:
				return -1
			case 112:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 4
			case 80:
				return -1
			case 84:
				return -1
			case 88:
				return -1
			case 99:
				return -1
			case 101:
				return 4
			case 112:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 80:
				return 5
			case 84:
				return -1
			case 88:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 112:
				return 5
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 80:
				return -1
			case 84:
				return 6
			case 88:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 116:
				return 6
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 80:
				return -1
			case 84:
				return -1
			case 88:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [eE][xX][cC][lL][uU][dD][eE]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 1
			case 76:
				return -1
			case 85:
				return -1
			case 88:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 1
			case 108:
				return -1
			case 117:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 88:
				return 2
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			case 120:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 3
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 88:
				return -1
			case 99:
				return 3
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return 4
			case 85:
				return -1
			case 88:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return 4
			case 117:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 85:
				return 5
			case 88:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 117:
				return 5
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return 6
			case 69:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 88:
				return -1
			case 99:
				return -1
			case 100:
				return 6
			case 101:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 7
			case 76:
				return -1
			case 85:
				return -1
			case 88:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 7
			case 108:
				return -1
			case 117:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 88:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			case 120:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [eE][xX][eE][cC][uU][tT][eE]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 1
			case 84:
				return -1
			case 85:
				return -1
			case 88:
				return -1
			case 99:
				return -1
			case 101:
				return 1
			case 116:
				return -1
			case 117:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 88:
				return 2
			case 99:
				return -1
			case 101:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 120:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 3
			case 84:
				return -1
			case 85:
				return -1
			case 88:
				return -1
			case 99:
				return -1
			case 101:
				return 3
			case 116:
				return -1
			case 117:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 4
			case 69:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 88:
				return -1
			case 99:
				return 4
			case 101:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 84:
				return -1
			case 85:
				return 5
			case 88:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 116:
				return -1
			case 117:
				return 5
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 84:
				return 6
			case 85:
				return -1
			case 88:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 116:
				return 6
			case 117:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 7
			case 84:
				return -1
			case 85:
				return -1
			case 88:
				return -1
			case 99:
				return -1
			case 101:
				return 7
			case 116:
				return -1
			case 117:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 88:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 120:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [eE][xX][iI][sS][tT][sS]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return 1
			case 73:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 88:
				return -1
			case 101:
				return 1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 88:
				return 2
			case 101:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 120:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return 3
			case 83:
				return -1
			case 84:
				return -1
			case 88:
				return -1
			case 101:
				return -1
			case 105:
				return 3
			case 115:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 83:
				return 4
			case 84:
				return -1
			case 88:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 115:
				return 4
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 83:
				return -1
			case 84:
				return 5
			case 88:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return 5
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 83:
				return 6
			case 84:
				return -1
			case 88:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 115:
				return 6
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 88:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [eE][xX][pP][lL][aA][iI][nN]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 97:
				return -1
			case 101:
				return 1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 88:
				return 2
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 120:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 80:
				return 3
			case 88:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 112:
				return 3
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return 4
			case 78:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return 4
			case 110:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 5
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 97:
				return 5
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 73:
				return 6
			case 76:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return 6
			case 108:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return 7
			case 80:
				return -1
			case 88:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return 7
			case 112:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [fF][aA][lL][sS][eE]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return 1
			case 76:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return 1
			case 108:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 83:
				return -1
			case 97:
				return 2
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return 3
			case 83:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return 3
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 83:
				return 4
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 115:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 5
			case 70:
				return -1
			case 76:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 101:
				return 5
			case 102:
				return -1
			case 108:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [fF][eE][tT][cC][hH]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 70:
				return 1
			case 72:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 102:
				return 1
			case 104:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 2
			case 70:
				return -1
			case 72:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return 2
			case 102:
				return -1
			case 104:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 72:
				return -1
			case 84:
				return 3
			case 99:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 104:
				return -1
			case 116:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 4
			case 69:
				return -1
			case 70:
				return -1
			case 72:
				return -1
			case 84:
				return -1
			case 99:
				return 4
			case 101:
				return -1
			case 102:
				return -1
			case 104:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 72:
				return 5
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 104:
				return 5
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 72:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 104:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [fF][iI][lL][tT][eE][rR]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return 1
			case 73:
				return -1
			case 76:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 102:
				return 1
			case 105:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return -1
			case 73:
				return 2
			case 76:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return 2
			case 108:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return -1
			case 73:
				return -1
			case 76:
				return 3
			case 82:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 108:
				return 3
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 82:
				return -1
			case 84:
				return 4
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 116:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 5
			case 70:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 101:
				return 5
			case 102:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 82:
				return 6
			case 84:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 114:
				return 6
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [fF][iI][rR][sS][tT]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 70:
				return 1
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 102:
				return 1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 73:
				return 2
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 102:
				return -1
			case 105:
				return 2
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 73:
				return -1
			case 82:
				return 3
			case 83:
				return -1
			case 84:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 114:
				return 3
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return 4
			case 84:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return 4
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 5
			case 102:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [fF][lL][aA][tT][tT][eE][nN]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return 1
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return 1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return 2
			case 78:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return 2
			case 110:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 3
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 97:
				return 3
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return 4
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return 5
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 6
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return 6
			case 102:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return 7
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 110:
				return 7
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [fF][lL][aA][tT][tT][eE][nN][_][kK][eE][yY][sS]
	{[]bool{false, false, false, false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return 1
			case 75:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 89:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return 1
			case 107:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 75:
				return -1
			case 76:
				return 2
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 89:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 107:
				return -1
			case 108:
				return 2
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 3
			case 69:
				return -1
			case 70:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 89:
				return -1
			case 95:
				return -1
			case 97:
				return 3
			case 101:
				return -1
			case 102:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return 4
			case 89:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return 4
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return 5
			case 89:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return 5
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 6
			case 70:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 89:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 101:
				return 6
			case 102:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 78:
				return 7
			case 83:
				return -1
			case 84:
				return -1
			case 89:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 110:
				return 7
			case 115:
				return -1
			case 116:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 89:
				return -1
			case 95:
				return 8
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 75:
				return 9
			case 76:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 89:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 107:
				return 9
			case 108:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 10
			case 70:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 89:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 101:
				return 10
			case 102:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 89:
				return 11
			case 95:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 121:
				return 11
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 83:
				return 12
			case 84:
				return -1
			case 89:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 115:
				return 12
			case 116:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 89:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 121:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [fF][lL][uU][sS][hH]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 70:
				return 1
			case 72:
				return -1
			case 76:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 102:
				return 1
			case 104:
				return -1
			case 108:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 72:
				return -1
			case 76:
				return 2
			case 83:
				return -1
			case 85:
				return -1
			case 102:
				return -1
			case 104:
				return -1
			case 108:
				return 2
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 83:
				return -1
			case 85:
				return 3
			case 102:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 115:
				return -1
			case 117:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 83:
				return 4
			case 85:
				return -1
			case 102:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 115:
				return 4
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 72:
				return 5
			case 76:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 102:
				return -1
			case 104:
				return 5
			case 108:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 102:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [fF][oO][lL][lL][oO][wW][iI][nN][gG]
	{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 70:
				return 1
			case 71:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 87:
				return -1
			case 102:
				return 1
			case 103:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return 2
			case 87:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return 2
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 76:
				return 3
			case 78:
				return -1
			case 79:
				return -1
			case 87:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 108:
				return 3
			case 110:
				return -1
			case 111:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 76:
				return 4
			case 78:
				return -1
			case 79:
				return -1
			case 87:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 108:
				return 4
			case 110:
				return -1
			case 111:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return 5
			case 87:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return 5
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 87:
				return 6
			case 102:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 119:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 71:
				return -1
			case 73:
				return 7
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 87:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 105:
				return 7
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return 8
			case 79:
				return -1
			case 87:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return 8
			case 111:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 71:
				return 9
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 87:
				return -1
			case 102:
				return -1
			case 103:
				return 9
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 87:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 119:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [fF][oO][rR]
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 70:
				return 1
			case 79:
				return -1
			case 82:
				return -1
			case 102:
				return 1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 79:
				return 2
			case 82:
				return -1
			case 102:
				return -1
			case 111:
				return 2
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 79:
				return -1
			case 82:
				return 3
			case 102:
				return -1
			case 111:
				return -1
			case 114:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 102:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [fF][oO][rR][cC][eE]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 70:
				return 1
			case 79:
				return -1
			case 82:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 102:
				return 1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 79:
				return 2
			case 82:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 111:
				return 2
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 79:
				return -1
			case 82:
				return 3
			case 99:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 111:
				return -1
			case 114:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 4
			case 69:
				return -1
			case 70:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 99:
				return 4
			case 101:
				return -1
			case 102:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 5
			case 70:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 99:
				return -1
			case 101:
				return 5
			case 102:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [fF][rR][oO][mM]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 70:
				return 1
			case 77:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 102:
				return 1
			case 109:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 82:
				return 2
			case 102:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 114:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 77:
				return -1
			case 79:
				return 3
			case 82:
				return -1
			case 102:
				return -1
			case 109:
				return -1
			case 111:
				return 3
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 77:
				return 4
			case 79:
				return -1
			case 82:
				return -1
			case 102:
				return -1
			case 109:
				return 4
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 102:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [fF][tT][sS]
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 70:
				return 1
			case 83:
				return -1
			case 84:
				return -1
			case 102:
				return 1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 83:
				return -1
			case 84:
				return 2
			case 102:
				return -1
			case 115:
				return -1
			case 116:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 83:
				return 3
			case 84:
				return -1
			case 102:
				return -1
			case 115:
				return 3
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 102:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [fF][uU][nN][cC][tT][iI][oO][nN]
	{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 70:
				return 1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 102:
				return 1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 70:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 85:
				return 2
			case 99:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 70:
				return -1
			case 73:
				return -1
			case 78:
				return 3
			case 79:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 110:
				return 3
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 4
			case 70:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return 4
			case 102:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 70:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return 5
			case 85:
				return -1
			case 99:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return 5
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 70:
				return -1
			case 73:
				return 6
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 102:
				return -1
			case 105:
				return 6
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 70:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return 7
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return 7
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 70:
				return -1
			case 73:
				return -1
			case 78:
				return 8
			case 79:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 110:
				return 8
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 70:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [gG][oO][lL][aA][nN][gG]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return 1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 97:
				return -1
			case 103:
				return 1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return 2
			case 97:
				return -1
			case 103:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return -1
			case 76:
				return 3
			case 78:
				return -1
			case 79:
				return -1
			case 97:
				return -1
			case 103:
				return -1
			case 108:
				return 3
			case 110:
				return -1
			case 111:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 4
			case 71:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 97:
				return 4
			case 103:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return -1
			case 76:
				return -1
			case 78:
				return 5
			case 79:
				return -1
			case 97:
				return -1
			case 103:
				return -1
			case 108:
				return -1
			case 110:
				return 5
			case 111:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return 6
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 97:
				return -1
			case 103:
				return 6
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 97:
				return -1
			case 103:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [gG][rR][aA][nN][tT]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return 1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 103:
				return 1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return -1
			case 78:
				return -1
			case 82:
				return 2
			case 84:
				return -1
			case 97:
				return -1
			case 103:
				return -1
			case 110:
				return -1
			case 114:
				return 2
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 3
			case 71:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return 3
			case 103:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return -1
			case 78:
				return 4
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 103:
				return -1
			case 110:
				return 4
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return 5
			case 97:
				return -1
			case 103:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 103:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [gG][rR][oO][uU][pP]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 71:
				return 1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 85:
				return -1
			case 103:
				return 1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return 2
			case 85:
				return -1
			case 103:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return 2
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 79:
				return 3
			case 80:
				return -1
			case 82:
				return -1
			case 85:
				return -1
			case 103:
				return -1
			case 111:
				return 3
			case 112:
				return -1
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 85:
				return 4
			case 103:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 117:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 79:
				return -1
			case 80:
				return 5
			case 82:
				return -1
			case 85:
				return -1
			case 103:
				return -1
			case 111:
				return -1
			case 112:
				return 5
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 85:
				return -1
			case 103:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [gG][rR][oO][uU][pP][sS]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 71:
				return 1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 103:
				return 1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return 2
			case 83:
				return -1
			case 85:
				return -1
			case 103:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return 2
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 79:
				return 3
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 103:
				return -1
			case 111:
				return 3
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 85:
				return 4
			case 103:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 117:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 79:
				return -1
			case 80:
				return 5
			case 82:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 103:
				return -1
			case 111:
				return -1
			case 112:
				return 5
			case 114:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return 6
			case 85:
				return -1
			case 103:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return 6
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 103:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [gG][sS][iI]
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 71:
				return 1
			case 73:
				return -1
			case 83:
				return -1
			case 103:
				return 1
			case 105:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 73:
				return -1
			case 83:
				return 2
			case 103:
				return -1
			case 105:
				return -1
			case 115:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 73:
				return 3
			case 83:
				return -1
			case 103:
				return -1
			case 105:
				return 3
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 73:
				return -1
			case 83:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [hH][aA][sS][hH]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 72:
				return 1
			case 83:
				return -1
			case 97:
				return -1
			case 104:
				return 1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 72:
				return -1
			case 83:
				return -1
			case 97:
				return 2
			case 104:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 72:
				return -1
			case 83:
				return 3
			case 97:
				return -1
			case 104:
				return -1
			case 115:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 72:
				return 4
			case 83:
				return -1
			case 97:
				return -1
			case 104:
				return 4
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 72:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 104:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [hH][aA][vV][iI][nN][gG]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return -1
			case 72:
				return 1
			case 73:
				return -1
			case 78:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 103:
				return -1
			case 104:
				return 1
			case 105:
				return -1
			case 110:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 71:
				return -1
			case 72:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 86:
				return -1
			case 97:
				return 2
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 86:
				return 3
			case 97:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 118:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 73:
				return 4
			case 78:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return 4
			case 110:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 73:
				return -1
			case 78:
				return 5
			case 86:
				return -1
			case 97:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 110:
				return 5
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return 6
			case 72:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 103:
				return 6
			case 104:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [iI][fF]
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 73:
				return 1
			case 102:
				return -1
			case 105:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return 2
			case 73:
				return -1
			case 102:
				return 2
			case 105:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 70:
				return -1
			case 73:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// [iI][gG][nN][oO][rR][eE]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return 1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return 1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return 2
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 101:
				return -1
			case 103:
				return 2
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return 3
			case 79:
				return -1
			case 82:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return 3
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return 4
			case 82:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return 4
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return 5
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 6
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 101:
				return 6
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [iI][lL][iI][kK][eE]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return 1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return 1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return 2
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return 3
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return 3
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return 4
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return 4
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 5
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return 5
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [iI][nN]
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 73:
				return 1
			case 78:
				return -1
			case 105:
				return 1
			case 110:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return 2
			case 105:
				return -1
			case 110:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// [iI][nN][cC][lL][uU][dD][eE]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return 1
			case 76:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return 1
			case 108:
				return -1
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return 2
			case 85:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return 2
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 3
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 99:
				return 3
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return 4
			case 78:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return 4
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 85:
				return 5
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 117:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return 6
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 100:
				return 6
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 7
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 7
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [iI][nN][cC][rR][eE][mM][eE][nN][tT]
	{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return 1
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return 1
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 78:
				return 2
			case 82:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 110:
				return 2
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 3
			case 69:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 99:
				return 3
			case 101:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return 4
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return 4
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 5
			case 73:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return 5
			case 105:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 77:
				return 6
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 109:
				return 6
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 7
			case 73:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return 7
			case 105:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 78:
				return 8
			case 82:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 110:
				return 8
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return 9
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return 9
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [iI][nN][dD][eE][xX]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return 1
			case 78:
				return -1
			case 88:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return 1
			case 110:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return 2
			case 88:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return 2
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return 3
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 88:
				return -1
			case 100:
				return 3
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return 4
			case 73:
				return -1
			case 78:
				return -1
			case 88:
				return -1
			case 100:
				return -1
			case 101:
				return 4
			case 105:
				return -1
			case 110:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 88:
				return 5
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 120:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 88:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 120:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [iI][nN][fF][eE][rR]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return -1
			case 73:
				return 1
			case 78:
				return -1
			case 82:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return 1
			case 110:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return -1
			case 73:
				return -1
			case 78:
				return 2
			case 82:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 110:
				return 2
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return 3
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 101:
				return -1
			case 102:
				return 3
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 4
			case 70:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 101:
				return 4
			case 102:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return 5
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [iI][nN][lL][iI][nN][eE]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return 1
			case 76:
				return -1
			case 78:
				return -1
			case 101:
				return -1
			case 105:
				return 1
			case 108:
				return -1
			case 110:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return 2
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return 3
			case 78:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return 3
			case 110:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return 4
			case 76:
				return -1
			case 78:
				return -1
			case 101:
				return -1
			case 105:
				return 4
			case 108:
				return -1
			case 110:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return 5
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 6
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 101:
				return 6
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [iI][nN][nN][eE][rR]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return 1
			case 78:
				return -1
			case 82:
				return -1
			case 101:
				return -1
			case 105:
				return 1
			case 110:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return 2
			case 82:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return 2
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return 3
			case 82:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return 3
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 4
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 101:
				return 4
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return 5
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [iI][nN][sS][eE][rR][tT]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return 1
			case 78:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 105:
				return 1
			case 110:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return 2
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return 2
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 83:
				return 3
			case 84:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 115:
				return 3
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 4
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return 4
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return 5
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return 5
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 6
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [iI][nN][tT][eE][rR][sS][eE][cC][tT]
	{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return 1
			case 78:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return 1
			case 110:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return 2
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return 2
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 3
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 4
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return 4
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return 5
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return 5
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 83:
				return 6
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 115:
				return 6
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 7
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return 7
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 8
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return 8
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 9
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 9
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [iI][nN][tT][oO]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 73:
				return 1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 105:
				return 1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return 2
			case 79:
				return -1
			case 84:
				return -1
			case 105:
				return -1
			case 110:
				return 2
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return 3
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return 4
			case 84:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return 4
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [iI][sS]
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 73:
				return 1
			case 83:
				return -1
			case 105:
				return 1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 83:
				return 2
			case 105:
				return -1
			case 115:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 83:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// [iI][sS][oO][lL][aA][tT][iI][oO][nN]
	{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return 1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 105:
				return 1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 83:
				return 2
			case 84:
				return -1
			case 97:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 115:
				return 2
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return 3
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return 3
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 76:
				return 4
			case 78:
				return -1
			case 79:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 105:
				return -1
			case 108:
				return 4
			case 110:
				return -1
			case 111:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 5
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return 5
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 83:
				return -1
			case 84:
				return 6
			case 97:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 115:
				return -1
			case 116:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return 7
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 105:
				return 7
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return 8
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return 8
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return 9
			case 79:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return 9
			case 111:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [jJ][aA][vV][aA][sS][cC][rR][iI][pP][tT]
	{[]bool{false, false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 74:
				return 1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 106:
				return 1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 67:
				return -1
			case 73:
				return -1
			case 74:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return 2
			case 99:
				return -1
			case 105:
				return -1
			case 106:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 74:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 86:
				return 3
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 106:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 4
			case 67:
				return -1
			case 73:
				return -1
			case 74:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return 4
			case 99:
				return -1
			case 105:
				return -1
			case 106:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 74:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return 5
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 106:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return 5
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return 6
			case 73:
				return -1
			case 74:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 99:
				return 6
			case 105:
				return -1
			case 106:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 74:
				return -1
			case 80:
				return -1
			case 82:
				return 7
			case 83:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 106:
				return -1
			case 112:
				return -1
			case 114:
				return 7
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return 8
			case 74:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return 8
			case 106:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 74:
				return -1
			case 80:
				return 9
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 106:
				return -1
			case 112:
				return 9
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 74:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 10
			case 86:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 106:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 10
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 74:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 106:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [jJ][oO][iI][nN]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 74:
				return 1
			case 78:
				return -1
			case 79:
				return -1
			case 105:
				return -1
			case 106:
				return 1
			case 110:
				return -1
			case 111:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 74:
				return -1
			case 78:
				return -1
			case 79:
				return 2
			case 105:
				return -1
			case 106:
				return -1
			case 110:
				return -1
			case 111:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return 3
			case 74:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 105:
				return 3
			case 106:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 74:
				return -1
			case 78:
				return 4
			case 79:
				return -1
			case 105:
				return -1
			case 106:
				return -1
			case 110:
				return 4
			case 111:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 74:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 105:
				return -1
			case 106:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [kK][eE][yY]
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 75:
				return 1
			case 89:
				return -1
			case 101:
				return -1
			case 107:
				return 1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 2
			case 75:
				return -1
			case 89:
				return -1
			case 101:
				return 2
			case 107:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 75:
				return -1
			case 89:
				return 3
			case 101:
				return -1
			case 107:
				return -1
			case 121:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 75:
				return -1
			case 89:
				return -1
			case 101:
				return -1
			case 107:
				return -1
			case 121:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [kK][eE][yY][sS]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 75:
				return 1
			case 83:
				return -1
			case 89:
				return -1
			case 101:
				return -1
			case 107:
				return 1
			case 115:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 2
			case 75:
				return -1
			case 83:
				return -1
			case 89:
				return -1
			case 101:
				return 2
			case 107:
				return -1
			case 115:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 75:
				return -1
			case 83:
				return -1
			case 89:
				return 3
			case 101:
				return -1
			case 107:
				return -1
			case 115:
				return -1
			case 121:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 75:
				return -1
			case 83:
				return 4
			case 89:
				return -1
			case 101:
				return -1
			case 107:
				return -1
			case 115:
				return 4
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 75:
				return -1
			case 83:
				return -1
			case 89:
				return -1
			case 101:
				return -1
			case 107:
				return -1
			case 115:
				return -1
			case 121:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [kK][eE][yY][sS][pP][aA][cC][eE]
	{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 75:
				return 1
			case 80:
				return -1
			case 83:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 107:
				return 1
			case 112:
				return -1
			case 115:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return 2
			case 75:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return 2
			case 107:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 75:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 89:
				return 3
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 107:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			case 121:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 75:
				return -1
			case 80:
				return -1
			case 83:
				return 4
			case 89:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 107:
				return -1
			case 112:
				return -1
			case 115:
				return 4
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 75:
				return -1
			case 80:
				return 5
			case 83:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 107:
				return -1
			case 112:
				return 5
			case 115:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 6
			case 67:
				return -1
			case 69:
				return -1
			case 75:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 89:
				return -1
			case 97:
				return 6
			case 99:
				return -1
			case 101:
				return -1
			case 107:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return 7
			case 69:
				return -1
			case 75:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 99:
				return 7
			case 101:
				return -1
			case 107:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return 8
			case 75:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return 8
			case 107:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 75:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 107:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			case 121:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [kK][nN][oO][wW][nN]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 75:
				return 1
			case 78:
				return -1
			case 79:
				return -1
			case 87:
				return -1
			case 107:
				return 1
			case 110:
				return -1
			case 111:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 75:
				return -1
			case 78:
				return 2
			case 79:
				return -1
			case 87:
				return -1
			case 107:
				return -1
			case 110:
				return 2
			case 111:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 75:
				return -1
			case 78:
				return -1
			case 79:
				return 3
			case 87:
				return -1
			case 107:
				return -1
			case 110:
				return -1
			case 111:
				return 3
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 75:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 87:
				return 4
			case 107:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 119:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 75:
				return -1
			case 78:
				return 5
			case 79:
				return -1
			case 87:
				return -1
			case 107:
				return -1
			case 110:
				return 5
			case 111:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 75:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 87:
				return -1
			case 107:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 119:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [lL][aA][nN][gG][uU][aA][gG][eE]
	{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 76:
				return 1
			case 78:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 108:
				return 1
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 69:
				return -1
			case 71:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 97:
				return 2
			case 101:
				return -1
			case 103:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 76:
				return -1
			case 78:
				return 3
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 108:
				return -1
			case 110:
				return 3
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 71:
				return 4
			case 76:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 103:
				return 4
			case 108:
				return -1
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 85:
				return 5
			case 97:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 117:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 6
			case 69:
				return -1
			case 71:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 97:
				return 6
			case 101:
				return -1
			case 103:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 71:
				return 7
			case 76:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 103:
				return 7
			case 108:
				return -1
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 8
			case 71:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return 8
			case 103:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [lL][aA][sS][tT]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 76:
				return 1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 108:
				return 1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 76:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return 2
			case 108:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 76:
				return -1
			case 83:
				return 3
			case 84:
				return -1
			case 97:
				return -1
			case 108:
				return -1
			case 115:
				return 3
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 76:
				return -1
			case 83:
				return -1
			case 84:
				return 4
			case 97:
				return -1
			case 108:
				return -1
			case 115:
				return -1
			case 116:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 76:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 108:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [lL][aA][tT][eE][rR][aA][lL]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return 1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return 1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 69:
				return -1
			case 76:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return 2
			case 101:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 82:
				return -1
			case 84:
				return 3
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 116:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 4
			case 76:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return 4
			case 108:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 82:
				return 5
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 114:
				return 5
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 6
			case 69:
				return -1
			case 76:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return 6
			case 101:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return 7
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return 7
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [lL][eE][fF][tT]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return 1
			case 84:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return 1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 2
			case 70:
				return -1
			case 76:
				return -1
			case 84:
				return -1
			case 101:
				return 2
			case 102:
				return -1
			case 108:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return 3
			case 76:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 102:
				return 3
			case 108:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 84:
				return 4
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 116:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [lL][eE][tT]
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 76:
				return 1
			case 84:
				return -1
			case 101:
				return -1
			case 108:
				return 1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 2
			case 76:
				return -1
			case 84:
				return -1
			case 101:
				return 2
			case 108:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 76:
				return -1
			case 84:
				return 3
			case 101:
				return -1
			case 108:
				return -1
			case 116:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 76:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [lL][eE][tT][tT][iI][nN][gG]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 76:
				return 1
			case 78:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 108:
				return 1
			case 110:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 2
			case 71:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 101:
				return 2
			case 103:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return 3
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return 4
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return 5
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return 5
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return 6
			case 84:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return 6
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return 7
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 103:
				return 7
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [lL][eE][vV][eE][lL]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 76:
				return 1
			case 86:
				return -1
			case 101:
				return -1
			case 108:
				return 1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 2
			case 76:
				return -1
			case 86:
				return -1
			case 101:
				return 2
			case 108:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 76:
				return -1
			case 86:
				return 3
			case 101:
				return -1
			case 108:
				return -1
			case 118:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 4
			case 76:
				return -1
			case 86:
				return -1
			case 101:
				return 4
			case 108:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 76:
				return 5
			case 86:
				return -1
			case 101:
				return -1
			case 108:
				return 5
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 76:
				return -1
			case 86:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [lL][iI][kK][eE]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return 1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return 2
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return 2
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return 3
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return 3
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 4
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return 4
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [lL][iI][mM][iI][tT]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 76:
				return 1
			case 77:
				return -1
			case 84:
				return -1
			case 105:
				return -1
			case 108:
				return 1
			case 109:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return 2
			case 76:
				return -1
			case 77:
				return -1
			case 84:
				return -1
			case 105:
				return 2
			case 108:
				return -1
			case 109:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 76:
				return -1
			case 77:
				return 3
			case 84:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return 3
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return 4
			case 76:
				return -1
			case 77:
				return -1
			case 84:
				return -1
			case 105:
				return 4
			case 108:
				return -1
			case 109:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 84:
				return 5
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 116:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 84:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [lL][sS][mM]
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 76:
				return 1
			case 77:
				return -1
			case 83:
				return -1
			case 108:
				return 1
			case 109:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 77:
				return -1
			case 83:
				return 2
			case 108:
				return -1
			case 109:
				return -1
			case 115:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 77:
				return 3
			case 83:
				return -1
			case 108:
				return -1
			case 109:
				return 3
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 77:
				return -1
			case 83:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [mM][aA][pP]
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 77:
				return 1
			case 80:
				return -1
			case 97:
				return -1
			case 109:
				return 1
			case 112:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 77:
				return -1
			case 80:
				return -1
			case 97:
				return 2
			case 109:
				return -1
			case 112:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 77:
				return -1
			case 80:
				return 3
			case 97:
				return -1
			case 109:
				return -1
			case 112:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 77:
				return -1
			case 80:
				return -1
			case 97:
				return -1
			case 109:
				return -1
			case 112:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [mM][aA][pP][pP][iI][nN][gG]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 77:
				return 1
			case 78:
				return -1
			case 80:
				return -1
			case 97:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 109:
				return 1
			case 110:
				return -1
			case 112:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 71:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 97:
				return 2
			case 103:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 80:
				return 3
			case 97:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 112:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 80:
				return 4
			case 97:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 112:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return -1
			case 73:
				return 5
			case 77:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 97:
				return -1
			case 103:
				return -1
			case 105:
				return 5
			case 109:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 78:
				return 6
			case 80:
				return -1
			case 97:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 110:
				return 6
			case 112:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return 7
			case 73:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 97:
				return -1
			case 103:
				return 7
			case 105:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 97:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [mM][aA][tT][cC][hH][eE][dD]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 72:
				return -1
			case 77:
				return 1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 109:
				return 1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 72:
				return -1
			case 77:
				return -1
			case 84:
				return -1
			case 97:
				return 2
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 109:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 72:
				return -1
			case 77:
				return -1
			case 84:
				return 3
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 109:
				return -1
			case 116:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return 4
			case 68:
				return -1
			case 69:
				return -1
			case 72:
				return -1
			case 77:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return 4
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 109:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 72:
				return 5
			case 77:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return 5
			case 109:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 6
			case 72:
				return -1
			case 77:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 6
			case 104:
				return -1
			case 109:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return 7
			case 69:
				return -1
			case 72:
				return -1
			case 77:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return 7
			case 101:
				return -1
			case 104:
				return -1
			case 109:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 72:
				return -1
			case 77:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 109:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [mM][aA][tT][eE][rR][iI][aA][lL][iI][zZ][eE][dD]
	{[]bool{false, false, false, false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 77:
				return 1
			case 82:
				return -1
			case 84:
				return -1
			case 90:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return 1
			case 114:
				return -1
			case 116:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 90:
				return -1
			case 97:
				return 2
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 82:
				return -1
			case 84:
				return 3
			case 90:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 114:
				return -1
			case 116:
				return 3
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return 4
			case 73:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 90:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return 4
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 82:
				return 5
			case 84:
				return -1
			case 90:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 114:
				return 5
			case 116:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return 6
			case 76:
				return -1
			case 77:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 90:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return 6
			case 108:
				return -1
			case 109:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 7
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 90:
				return -1
			case 97:
				return 7
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return 8
			case 77:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 90:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return 8
			case 109:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return 9
			case 76:
				return -1
			case 77:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 90:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return 9
			case 108:
				return -1
			case 109:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 90:
				return 10
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 122:
				return 10
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return 11
			case 73:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 90:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return 11
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return 12
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 90:
				return -1
			case 97:
				return -1
			case 100:
				return 12
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 90:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 122:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [mM][eE][rR][gG][eE]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 77:
				return 1
			case 82:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 109:
				return 1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 2
			case 71:
				return -1
			case 77:
				return -1
			case 82:
				return -1
			case 101:
				return 2
			case 103:
				return -1
			case 109:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 77:
				return -1
			case 82:
				return 3
			case 101:
				return -1
			case 103:
				return -1
			case 109:
				return -1
			case 114:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return 4
			case 77:
				return -1
			case 82:
				return -1
			case 101:
				return -1
			case 103:
				return 4
			case 109:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 5
			case 71:
				return -1
			case 77:
				return -1
			case 82:
				return -1
			case 101:
				return 5
			case 103:
				return -1
			case 109:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 77:
				return -1
			case 82:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 109:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [mM][iI][sS][sS][iI][nN][gG]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 73:
				return -1
			case 77:
				return 1
			case 78:
				return -1
			case 83:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 109:
				return 1
			case 110:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 73:
				return 2
			case 77:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 103:
				return -1
			case 105:
				return 2
			case 109:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 83:
				return 3
			case 103:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 115:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 83:
				return 4
			case 103:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 115:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 73:
				return 5
			case 77:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 103:
				return -1
			case 105:
				return 5
			case 109:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 78:
				return 6
			case 83:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 110:
				return 6
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return 7
			case 73:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 103:
				return 7
			case 105:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [nN][aA][mM][eE][sS][pP][aA][cC][eE]
	{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return 1
			case 80:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return 1
			case 112:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 67:
				return -1
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 97:
				return 2
			case 99:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 77:
				return 3
			case 78:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 109:
				return 3
			case 110:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return 4
			case 77:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return 4
			case 109:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 83:
				return 5
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 115:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 80:
				return 6
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 112:
				return 6
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 7
			case 67:
				return -1
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 97:
				return 7
			case 99:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return 8
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return 8
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return 9
			case 77:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return 9
			case 109:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [nN][eE][sS][tT]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return 1
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 110:
				return 1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 2
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return 2
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return -1
			case 83:
				return 3
			case 84:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 115:
				return 3
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return 4
			case 101:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [nN][lL]
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 78:
				return 1
			case 108:
				return -1
			case 110:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return 2
			case 78:
				return -1
			case 108:
				return 2
			case 110:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 78:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// [nN][oO]
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 78:
				return 1
			case 79:
				return -1
			case 110:
				return 1
			case 111:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 78:
				return -1
			case 79:
				return 2
			case 110:
				return -1
			case 111:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 78:
				return -1
			case 79:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// [nN][oO][tT]
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 78:
				return 1
			case 79:
				return -1
			case 84:
				return -1
			case 110:
				return 1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 78:
				return -1
			case 79:
				return 2
			case 84:
				return -1
			case 110:
				return -1
			case 111:
				return 2
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return 3
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [nN][tT][hH][_][vV][aA][lL][uU][eE]
	{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return 1
			case 84:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return 1
			case 116:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return 2
			case 85:
				return -1
			case 86:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return 2
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 72:
				return 3
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 104:
				return 3
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 95:
				return 4
			case 97:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 86:
				return 5
			case 95:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 118:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 6
			case 69:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 95:
				return -1
			case 97:
				return 6
			case 101:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 72:
				return -1
			case 76:
				return 7
			case 78:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 108:
				return 7
			case 110:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 85:
				return 8
			case 86:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			case 117:
				return 8
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 9
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 101:
				return 9
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [nN][uU][lL][lL]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 78:
				return 1
			case 85:
				return -1
			case 108:
				return -1
			case 110:
				return 1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 78:
				return -1
			case 85:
				return 2
			case 108:
				return -1
			case 110:
				return -1
			case 117:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return 3
			case 78:
				return -1
			case 85:
				return -1
			case 108:
				return 3
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return 4
			case 78:
				return -1
			case 85:
				return -1
			case 108:
				return 4
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [nN][uU][lL][lL][sS]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 78:
				return 1
			case 83:
				return -1
			case 85:
				return -1
			case 108:
				return -1
			case 110:
				return 1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 85:
				return 2
			case 108:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 117:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return 3
			case 78:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 108:
				return 3
			case 110:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return 4
			case 78:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 108:
				return 4
			case 110:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 78:
				return -1
			case 83:
				return 5
			case 85:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 115:
				return 5
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [nN][uN][mM][bB][eE][rR]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return 1
			case 82:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return 1
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return 2
			case 82:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 117:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return -1
			case 77:
				return 3
			case 78:
				return -1
			case 82:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 109:
				return 3
			case 110:
				return -1
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return 4
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 98:
				return 4
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return 5
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 98:
				return -1
			case 101:
				return 5
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return 6
			case 98:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return 6
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [oO][bB][jJ][eE][cC][tT]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 74:
				return -1
			case 79:
				return 1
			case 84:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 106:
				return -1
			case 111:
				return 1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return 2
			case 67:
				return -1
			case 69:
				return -1
			case 74:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 98:
				return 2
			case 99:
				return -1
			case 101:
				return -1
			case 106:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 74:
				return 3
			case 79:
				return -1
			case 84:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 106:
				return 3
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return 4
			case 74:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return 4
			case 106:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return 5
			case 69:
				return -1
			case 74:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 98:
				return -1
			case 99:
				return 5
			case 101:
				return -1
			case 106:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 74:
				return -1
			case 79:
				return -1
			case 84:
				return 6
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 106:
				return -1
			case 111:
				return -1
			case 116:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 74:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 106:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [oO][fF][fF][sS][eE][tT]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return -1
			case 79:
				return 1
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 111:
				return 1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return 2
			case 79:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 102:
				return 2
			case 111:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return 3
			case 79:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 102:
				return 3
			case 111:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return -1
			case 79:
				return -1
			case 83:
				return 4
			case 84:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 111:
				return -1
			case 115:
				return 4
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 5
			case 70:
				return -1
			case 79:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return 5
			case 102:
				return -1
			case 111:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return -1
			case 79:
				return -1
			case 83:
				return -1
			case 84:
				return 6
			case 101:
				return -1
			case 102:
				return -1
			case 111:
				return -1
			case 115:
				return -1
			case 116:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return -1
			case 79:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 111:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [oO][nN]
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 78:
				return -1
			case 79:
				return 1
			case 110:
				return -1
			case 111:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 78:
				return 2
			case 79:
				return -1
			case 110:
				return 2
			case 111:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 78:
				return -1
			case 79:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// [oO][pP][tT][iI][oO][nN]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return 1
			case 80:
				return -1
			case 84:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return 1
			case 112:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return 2
			case 84:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return 2
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 84:
				return 3
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 116:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return 4
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 84:
				return -1
			case 105:
				return 4
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return 5
			case 80:
				return -1
			case 84:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return 5
			case 112:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return 6
			case 79:
				return -1
			case 80:
				return -1
			case 84:
				return -1
			case 105:
				return -1
			case 110:
				return 6
			case 111:
				return -1
			case 112:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 84:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [oO][pP][tT][iI][oO][nN][sS]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return 1
			case 80:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return 1
			case 112:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return 2
			case 83:
				return -1
			case 84:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return 2
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 84:
				return 3
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			case 116:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return 4
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 105:
				return 4
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return 5
			case 80:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return 5
			case 112:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return 6
			case 79:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 105:
				return -1
			case 110:
				return 6
			case 111:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 83:
				return 7
			case 84:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 115:
				return 7
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [oO][rR]
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 79:
				return 1
			case 82:
				return -1
			case 111:
				return 1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 79:
				return -1
			case 82:
				return 2
			case 111:
				return -1
			case 114:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 79:
				return -1
			case 82:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// [oO][rR][dD][eE][rR]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 79:
				return 1
			case 82:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 111:
				return 1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return 2
			case 100:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return 3
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 100:
				return 3
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return 4
			case 79:
				return -1
			case 82:
				return -1
			case 100:
				return -1
			case 101:
				return 4
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return 5
			case 100:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [oO][tT][hH][eE][rR][sS]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return -1
			case 79:
				return 1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 111:
				return 1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 2
			case 101:
				return -1
			case 104:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return 3
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 104:
				return 3
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 4
			case 72:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return 4
			case 104:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return -1
			case 79:
				return -1
			case 82:
				return 5
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 111:
				return -1
			case 114:
				return 5
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return 6
			case 84:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return 6
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [oO][uU][tT][eE][rR]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 79:
				return 1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 111:
				return 1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return 2
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 84:
				return 3
			case 85:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 116:
				return 3
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 4
			case 79:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return 4
			case 111:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return 5
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return 5
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [oO][vV][eE][rR]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 79:
				return 1
			case 82:
				return -1
			case 86:
				return -1
			case 101:
				return -1
			case 111:
				return 1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 86:
				return 2
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 118:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 3
			case 79:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 101:
				return 3
			case 111:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return 4
			case 86:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return 4
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [pP][aA][rR][sS][eE]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 80:
				return 1
			case 82:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 112:
				return 1
			case 114:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 69:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 97:
				return 2
			case 101:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 80:
				return -1
			case 82:
				return 3
			case 83:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 114:
				return 3
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return 4
			case 97:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 5
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 101:
				return 5
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [pP][aA][rR][tT][iI][tT][iI][oO][nN]
	{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return 1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return 1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return 2
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return 3
			case 84:
				return -1
			case 97:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return 3
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return 4
			case 97:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return 5
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 105:
				return 5
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return 6
			case 97:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return 7
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 105:
				return 7
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return 8
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return 8
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 78:
				return 9
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 105:
				return -1
			case 110:
				return 9
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [pP][aA][sS][sS][wW][oO][rR][dD]
	{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 79:
				return -1
			case 80:
				return 1
			case 82:
				return -1
			case 83:
				return -1
			case 87:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 111:
				return -1
			case 112:
				return 1
			case 114:
				return -1
			case 115:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 68:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 87:
				return -1
			case 97:
				return 2
			case 100:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return 3
			case 87:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return 3
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return 4
			case 87:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return 4
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 87:
				return 5
			case 97:
				return -1
			case 100:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 119:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 79:
				return 6
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 87:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 111:
				return 6
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return 7
			case 83:
				return -1
			case 87:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return 7
			case 115:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return 8
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 87:
				return -1
			case 97:
				return -1
			case 100:
				return 8
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 87:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 119:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [pP][aA][tT][hH]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 72:
				return -1
			case 80:
				return 1
			case 84:
				return -1
			case 97:
				return -1
			case 104:
				return -1
			case 112:
				return 1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 72:
				return -1
			case 80:
				return -1
			case 84:
				return -1
			case 97:
				return 2
			case 104:
				return -1
			case 112:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 72:
				return -1
			case 80:
				return -1
			case 84:
				return 3
			case 97:
				return -1
			case 104:
				return -1
			case 112:
				return -1
			case 116:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 72:
				return 4
			case 80:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 104:
				return 4
			case 112:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 72:
				return -1
			case 80:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 104:
				return -1
			case 112:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [pP][oO][oO][lL]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 79:
				return -1
			case 80:
				return 1
			case 108:
				return -1
			case 111:
				return -1
			case 112:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 79:
				return 2
			case 80:
				return -1
			case 108:
				return -1
			case 111:
				return 2
			case 112:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 79:
				return 3
			case 80:
				return -1
			case 108:
				return -1
			case 111:
				return 3
			case 112:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return 4
			case 79:
				return -1
			case 80:
				return -1
			case 108:
				return 4
			case 111:
				return -1
			case 112:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [pP][rR][eE][cC][eE][dD][iI][nN][gG]
	{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 80:
				return 1
			case 82:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 112:
				return 1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return 2
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 3
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 3
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 4
			case 68:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 99:
				return 4
			case 100:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 5
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 5
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return 6
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 99:
				return -1
			case 100:
				return 6
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return 7
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return 7
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return 8
			case 80:
				return -1
			case 82:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return 8
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 71:
				return 9
			case 73:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 103:
				return 9
			case 105:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [pP][rR][eE][pP][aA][rR][eE]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 80:
				return 1
			case 82:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 112:
				return 1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 80:
				return -1
			case 82:
				return 2
			case 97:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 114:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 3
			case 80:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 101:
				return 3
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 80:
				return 4
			case 82:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 112:
				return 4
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 5
			case 69:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 97:
				return 5
			case 101:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 80:
				return -1
			case 82:
				return 6
			case 97:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 114:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 7
			case 80:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 101:
				return 7
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [pP][rR][iI][mM][aA][rR][yY]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 80:
				return 1
			case 82:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 112:
				return 1
			case 114:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 80:
				return -1
			case 82:
				return 2
			case 89:
				return -1
			case 97:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 112:
				return -1
			case 114:
				return 2
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return 3
			case 77:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 105:
				return 3
			case 109:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 77:
				return 4
			case 80:
				return -1
			case 82:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 105:
				return -1
			case 109:
				return 4
			case 112:
				return -1
			case 114:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 5
			case 73:
				return -1
			case 77:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 89:
				return -1
			case 97:
				return 5
			case 105:
				return -1
			case 109:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 80:
				return -1
			case 82:
				return 6
			case 89:
				return -1
			case 97:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 112:
				return -1
			case 114:
				return 6
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 89:
				return 7
			case 97:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 121:
				return 7
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 121:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [pP][rR][iI][vV][aA][tT][eE]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 80:
				return 1
			case 82:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 112:
				return 1
			case 114:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 80:
				return -1
			case 82:
				return 2
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 112:
				return -1
			case 114:
				return 2
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 73:
				return 3
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return 3
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 86:
				return 4
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 118:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 5
			case 69:
				return -1
			case 73:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return 5
			case 101:
				return -1
			case 105:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return 6
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return 6
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 7
			case 73:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return 7
			case 105:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [pP][rR][iI][vV][iI][lL][eE][gG][eE]
	{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 80:
				return 1
			case 82:
				return -1
			case 86:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 112:
				return 1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 82:
				return 2
			case 86:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 114:
				return 2
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return 3
			case 76:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return 3
			case 108:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 86:
				return 4
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 118:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return 5
			case 76:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return 5
			case 108:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 76:
				return 6
			case 80:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 108:
				return 6
			case 112:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 7
			case 71:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 101:
				return 7
			case 103:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return 8
			case 73:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 101:
				return -1
			case 103:
				return 8
			case 105:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 9
			case 71:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 101:
				return 9
			case 103:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [pP][rR][oO][cC][eE][dD][uU][rR][eE]
	{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 80:
				return 1
			case 82:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 112:
				return 1
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return 2
			case 85:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return 2
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 79:
				return 3
			case 80:
				return -1
			case 82:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 111:
				return 3
			case 112:
				return -1
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 4
			case 68:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 85:
				return -1
			case 99:
				return 4
			case 100:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 5
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 5
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return 6
			case 69:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 100:
				return 6
			case 101:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 85:
				return 7
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 117:
				return 7
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return 8
			case 85:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return 8
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 9
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 9
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [pP][rR][oO][bB][eE]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 80:
				return 1
			case 82:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 112:
				return 1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return 2
			case 98:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return -1
			case 79:
				return 3
			case 80:
				return -1
			case 82:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 111:
				return 3
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return 4
			case 69:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 98:
				return 4
			case 101:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return 5
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 98:
				return -1
			case 101:
				return 5
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [pP][uU][bB][lL][iI][cC]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 80:
				return 1
			case 85:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 112:
				return 1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 85:
				return 2
			case 98:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 117:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return 3
			case 67:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 85:
				return -1
			case 98:
				return 3
			case 99:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 76:
				return 4
			case 80:
				return -1
			case 85:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 108:
				return 4
			case 112:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 73:
				return 5
			case 76:
				return -1
			case 80:
				return -1
			case 85:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 105:
				return 5
			case 108:
				return -1
			case 112:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return 6
			case 73:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 85:
				return -1
			case 98:
				return -1
			case 99:
				return 6
			case 105:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 85:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [rR][aA][nN][gG][eE]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 78:
				return -1
			case 82:
				return 1
			case 97:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 110:
				return -1
			case 114:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 69:
				return -1
			case 71:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 97:
				return 2
			case 101:
				return -1
			case 103:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 78:
				return 3
			case 82:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 110:
				return 3
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 71:
				return 4
			case 78:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 103:
				return 4
			case 110:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 5
			case 71:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 101:
				return 5
			case 103:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [rR][aA][wW]
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 82:
				return 1
			case 87:
				return -1
			case 97:
				return -1
			case 114:
				return 1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 82:
				return -1
			case 87:
				return -1
			case 97:
				return 2
			case 114:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 82:
				return -1
			case 87:
				return 3
			case 97:
				return -1
			case 114:
				return -1
			case 119:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 82:
				return -1
			case 87:
				return -1
			case 97:
				return -1
			case 114:
				return -1
			case 119:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [rR][eE][aA][dD]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 82:
				return 1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 114:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return 2
			case 82:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return 2
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 3
			case 68:
				return -1
			case 69:
				return -1
			case 82:
				return -1
			case 97:
				return 3
			case 100:
				return -1
			case 101:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return 4
			case 69:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 100:
				return 4
			case 101:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [rR][eE][aA][lL][mM]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 82:
				return 1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 114:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 2
			case 76:
				return -1
			case 77:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 101:
				return 2
			case 108:
				return -1
			case 109:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 3
			case 69:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 82:
				return -1
			case 97:
				return 3
			case 101:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return 4
			case 77:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return 4
			case 109:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 77:
				return 5
			case 82:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 109:
				return 5
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [rR][eE][cC][uU][rR][sS][iI][vV][eE]
	{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return 1
			case 83:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return 1
			case 115:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 2
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 99:
				return -1
			case 101:
				return 2
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 3
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 99:
				return 3
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 85:
				return 4
			case 86:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 117:
				return 4
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return 5
			case 83:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return 5
			case 115:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return 6
			case 85:
				return -1
			case 86:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return 6
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return 7
			case 82:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return 7
			case 114:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 86:
				return 8
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			case 118:
				return 8
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 9
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 99:
				return -1
			case 101:
				return 9
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [rR][eE][dD][uU][cC][eE]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 82:
				return 1
			case 85:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 114:
				return 1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 2
			case 82:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 2
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return 3
			case 69:
				return -1
			case 82:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 100:
				return 3
			case 101:
				return -1
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 82:
				return -1
			case 85:
				return 4
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 114:
				return -1
			case 117:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 5
			case 68:
				return -1
			case 69:
				return -1
			case 82:
				return -1
			case 85:
				return -1
			case 99:
				return 5
			case 100:
				return -1
			case 101:
				return -1
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 6
			case 82:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 6
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 82:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [rR][eE][nN][aA][mM][eE]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return 1
			case 97:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 2
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 101:
				return 2
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return 3
			case 82:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return 3
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 4
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 97:
				return 4
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 77:
				return 5
			case 78:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 109:
				return 5
			case 110:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 6
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 101:
				return 6
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [rR][eE][pP][lL][aA][cC][eE]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 82:
				return 1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 114:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return 2
			case 76:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return 2
			case 108:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 80:
				return 3
			case 82:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 112:
				return 3
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return 4
			case 80:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return 4
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 5
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 97:
				return 5
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return 6
			case 69:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 99:
				return 6
			case 101:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return 7
			case 76:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return 7
			case 108:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [rR][eE][sS][pP][eE][cC][tT]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 80:
				return -1
			case 82:
				return 1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 114:
				return 1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 2
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return 2
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return 3
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return 3
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 80:
				return 4
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 112:
				return 4
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 5
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return 5
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 6
			case 69:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return 6
			case 101:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 7
			case 99:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 7
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [rR][eE][sS][tT][rR][iI][cC][tT]
	{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return 1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return 1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 2
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return 2
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return 3
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return 3
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 4
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return 5
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return 5
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return 6
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return 6
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 7
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return 7
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 8
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 8
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [rR][eE][tT][uU][rR][nN]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return 1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return 1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 2
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return 2
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return 3
			case 85:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return 3
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return 4
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return 5
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return 5
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return 6
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 110:
				return 6
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [rR][eE][tT][uU][rR][nN][iI][nN][gG]
	{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return 1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return 1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 2
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return 2
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return 3
			case 85:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return 3
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return 4
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return 5
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return 5
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return 6
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return 6
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return 7
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return 7
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return 8
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return 8
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return 9
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 103:
				return 9
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [rR][eE][vV][oO][kK][eE]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 75:
				return -1
			case 79:
				return -1
			case 82:
				return 1
			case 86:
				return -1
			case 101:
				return -1
			case 107:
				return -1
			case 111:
				return -1
			case 114:
				return 1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 2
			case 75:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 101:
				return 2
			case 107:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 75:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 86:
				return 3
			case 101:
				return -1
			case 107:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 118:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 75:
				return -1
			case 79:
				return 4
			case 82:
				return -1
			case 86:
				return -1
			case 101:
				return -1
			case 107:
				return -1
			case 111:
				return 4
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 75:
				return 5
			case 79:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 101:
				return -1
			case 107:
				return 5
			case 111:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 6
			case 75:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 101:
				return 6
			case 107:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 75:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 101:
				return -1
			case 107:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [rR][iI][gG][hH][tT]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 72:
				return -1
			case 73:
				return -1
			case 82:
				return 1
			case 84:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 114:
				return 1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 72:
				return -1
			case 73:
				return 2
			case 82:
				return -1
			case 84:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return 2
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return 3
			case 72:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 103:
				return 3
			case 104:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 72:
				return 4
			case 73:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 103:
				return -1
			case 104:
				return 4
			case 105:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 72:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 84:
				return 5
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 116:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 72:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [rR][oO][lL][eE]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 76:
				return -1
			case 79:
				return -1
			case 82:
				return 1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 114:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 76:
				return -1
			case 79:
				return 2
			case 82:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return 2
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 76:
				return 3
			case 79:
				return -1
			case 82:
				return -1
			case 101:
				return -1
			case 108:
				return 3
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 4
			case 76:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 101:
				return 4
			case 108:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 76:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [rR][oO][lL][lL][bB][aA][cC][kK]
	{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 79:
				return -1
			case 82:
				return 1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 114:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 79:
				return 2
			case 82:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 111:
				return 2
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 75:
				return -1
			case 76:
				return 3
			case 79:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 107:
				return -1
			case 108:
				return 3
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 75:
				return -1
			case 76:
				return 4
			case 79:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 107:
				return -1
			case 108:
				return 4
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return 5
			case 67:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 98:
				return 5
			case 99:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 6
			case 66:
				return -1
			case 67:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 97:
				return 6
			case 98:
				return -1
			case 99:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return 7
			case 75:
				return -1
			case 76:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return 7
			case 107:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 75:
				return 8
			case 76:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 107:
				return 8
			case 108:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [rR][oO][wW]
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 79:
				return -1
			case 82:
				return 1
			case 87:
				return -1
			case 111:
				return -1
			case 114:
				return 1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 79:
				return 2
			case 82:
				return -1
			case 87:
				return -1
			case 111:
				return 2
			case 114:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 79:
				return -1
			case 82:
				return -1
			case 87:
				return 3
			case 111:
				return -1
			case 114:
				return -1
			case 119:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 79:
				return -1
			case 82:
				return -1
			case 87:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 119:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [rR][oO][wW][sS]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 79:
				return -1
			case 82:
				return 1
			case 83:
				return -1
			case 87:
				return -1
			case 111:
				return -1
			case 114:
				return 1
			case 115:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 79:
				return 2
			case 82:
				return -1
			case 83:
				return -1
			case 87:
				return -1
			case 111:
				return 2
			case 114:
				return -1
			case 115:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 87:
				return 3
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 119:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return 4
			case 87:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return 4
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 87:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 119:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [sS][aA][tT][iI][sS][fF][iI][eE][sS]
	{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 73:
				return -1
			case 83:
				return 1
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 115:
				return 1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 69:
				return -1
			case 70:
				return -1
			case 73:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return 2
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 73:
				return -1
			case 83:
				return -1
			case 84:
				return 3
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 73:
				return 4
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return 4
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 73:
				return -1
			case 83:
				return 5
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 115:
				return 5
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return 6
			case 73:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return 6
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 73:
				return 7
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return 7
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 8
			case 70:
				return -1
			case 73:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return 8
			case 102:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 73:
				return -1
			case 83:
				return 9
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 115:
				return 9
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 73:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [sS][aA][vV][eE][pP][oO][iI][nN][tT]
	{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 83:
				return 1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 115:
				return 1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return 2
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 86:
				return 3
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 4
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return 4
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return 5
			case 83:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return 5
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return 6
			case 80:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return 6
			case 112:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 73:
				return 7
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return 7
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return 8
			case 79:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return 8
			case 111:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 84:
				return 9
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			case 116:
				return 9
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [sS][cC][hH][eE][mM][aA]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 72:
				return -1
			case 77:
				return -1
			case 83:
				return 1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 109:
				return -1
			case 115:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return 2
			case 69:
				return -1
			case 72:
				return -1
			case 77:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return 2
			case 101:
				return -1
			case 104:
				return -1
			case 109:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 72:
				return 3
			case 77:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 104:
				return 3
			case 109:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return 4
			case 72:
				return -1
			case 77:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return 4
			case 104:
				return -1
			case 109:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 72:
				return -1
			case 77:
				return 5
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 109:
				return 5
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 6
			case 67:
				return -1
			case 69:
				return -1
			case 72:
				return -1
			case 77:
				return -1
			case 83:
				return -1
			case 97:
				return 6
			case 99:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 109:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 72:
				return -1
			case 77:
				return -1
			case 83:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 109:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [sS][cC][oO][pP][eE]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 83:
				return 1
			case 99:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 115:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 2
			case 69:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 99:
				return 2
			case 101:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 79:
				return 3
			case 80:
				return -1
			case 83:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 111:
				return 3
			case 112:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 80:
				return 4
			case 83:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 112:
				return 4
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 5
			case 79:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 99:
				return -1
			case 101:
				return 5
			case 111:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 83:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [sS][eE][lL][eE][cC][tT]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 83:
				return 1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 115:
				return 1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 2
			case 76:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return 2
			case 108:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return 3
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return 3
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return 4
			case 76:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return 4
			case 108:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return 5
			case 69:
				return -1
			case 76:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return 5
			case 101:
				return -1
			case 108:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 83:
				return -1
			case 84:
				return 6
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 115:
				return -1
			case 116:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [sS][eE][lL][fF]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 83:
				return 1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 115:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 2
			case 70:
				return -1
			case 76:
				return -1
			case 83:
				return -1
			case 101:
				return 2
			case 102:
				return -1
			case 108:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return 3
			case 83:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return 3
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return 4
			case 76:
				return -1
			case 83:
				return -1
			case 101:
				return -1
			case 102:
				return 4
			case 108:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 83:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [sS][eE][tT]
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 83:
				return 1
			case 84:
				return -1
			case 101:
				return -1
			case 115:
				return 1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 2
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return 2
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 83:
				return -1
			case 84:
				return 3
			case 101:
				return -1
			case 115:
				return -1
			case 116:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [sS][hH][oO][wW]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 72:
				return -1
			case 79:
				return -1
			case 83:
				return 1
			case 87:
				return -1
			case 104:
				return -1
			case 111:
				return -1
			case 115:
				return 1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 72:
				return 2
			case 79:
				return -1
			case 83:
				return -1
			case 87:
				return -1
			case 104:
				return 2
			case 111:
				return -1
			case 115:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 72:
				return -1
			case 79:
				return 3
			case 83:
				return -1
			case 87:
				return -1
			case 104:
				return -1
			case 111:
				return 3
			case 115:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 72:
				return -1
			case 79:
				return -1
			case 83:
				return -1
			case 87:
				return 4
			case 104:
				return -1
			case 111:
				return -1
			case 115:
				return -1
			case 119:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 72:
				return -1
			case 79:
				return -1
			case 83:
				return -1
			case 87:
				return -1
			case 104:
				return -1
			case 111:
				return -1
			case 115:
				return -1
			case 119:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [sS][oO][mM][eE]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 83:
				return 1
			case 101:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 115:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 77:
				return -1
			case 79:
				return 2
			case 83:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 111:
				return 2
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 77:
				return 3
			case 79:
				return -1
			case 83:
				return -1
			case 101:
				return -1
			case 109:
				return 3
			case 111:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 4
			case 77:
				return -1
			case 79:
				return -1
			case 83:
				return -1
			case 101:
				return 4
			case 109:
				return -1
			case 111:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 83:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [sS][tT][aA][rR][tT]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 82:
				return -1
			case 83:
				return 1
			case 84:
				return -1
			case 97:
				return -1
			case 114:
				return -1
			case 115:
				return 1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 2
			case 97:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 3
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return 3
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 82:
				return 4
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 114:
				return 4
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 5
			case 97:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [sS][tT][aA][tT][iI][sS][tT][iI][cC][sS]
	{[]bool{false, false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 83:
				return 1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 115:
				return 1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 83:
				return -1
			case 84:
				return 2
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 3
			case 67:
				return -1
			case 73:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return 3
			case 99:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 83:
				return -1
			case 84:
				return 4
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return 5
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return 5
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 83:
				return 6
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 115:
				return 6
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 83:
				return -1
			case 84:
				return 7
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return 7
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return 8
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return 8
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return 9
			case 73:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return 9
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 83:
				return 10
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 115:
				return 10
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [sS][tT][rR][iI][nN][gG]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 83:
				return 1
			case 84:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 115:
				return 1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 2
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return 3
			case 83:
				return -1
			case 84:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return 3
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 73:
				return 4
			case 78:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 103:
				return -1
			case 105:
				return 4
			case 110:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return 5
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return 5
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return 6
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 103:
				return 6
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [sS][yY][sS][tT][eE][mM]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 77:
				return -1
			case 83:
				return 1
			case 84:
				return -1
			case 89:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 115:
				return 1
			case 116:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 77:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 89:
				return 2
			case 101:
				return -1
			case 109:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 121:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 77:
				return -1
			case 83:
				return 3
			case 84:
				return -1
			case 89:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 115:
				return 3
			case 116:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 77:
				return -1
			case 83:
				return -1
			case 84:
				return 4
			case 89:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 115:
				return -1
			case 116:
				return 4
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 5
			case 77:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 89:
				return -1
			case 101:
				return 5
			case 109:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 77:
				return 6
			case 83:
				return -1
			case 84:
				return -1
			case 89:
				return -1
			case 101:
				return -1
			case 109:
				return 6
			case 115:
				return -1
			case 116:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 77:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 89:
				return -1
			case 101:
				return -1
			case 109:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 121:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [tT][hH][eE][nN]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return -1
			case 78:
				return -1
			case 84:
				return 1
			case 101:
				return -1
			case 104:
				return -1
			case 110:
				return -1
			case 116:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return 2
			case 78:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 104:
				return 2
			case 110:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 3
			case 72:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 101:
				return 3
			case 104:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return -1
			case 78:
				return 4
			case 84:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 110:
				return 4
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [tT][iI][eE][sS]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 83:
				return -1
			case 84:
				return 1
			case 101:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return 2
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 105:
				return 2
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 3
			case 73:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return 3
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 83:
				return 4
			case 84:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 115:
				return 4
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [tT][oO]
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 79:
				return -1
			case 84:
				return 1
			case 111:
				return -1
			case 116:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 79:
				return 2
			case 84:
				return -1
			case 111:
				return 2
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 79:
				return -1
			case 84:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// [tT][rR][aA][nN]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return 1
			case 97:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 78:
				return -1
			case 82:
				return 2
			case 84:
				return -1
			case 97:
				return -1
			case 110:
				return -1
			case 114:
				return 2
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 3
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return 3
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 78:
				return 4
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 110:
				return 4
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [tT][rR][aA][nN][sS][aA][cC][tT][iI][oO][nN]
	{[]bool{false, false, false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 1
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return 2
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return 2
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 3
			case 67:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return 3
			case 99:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 78:
				return 4
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 110:
				return 4
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return 5
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return 5
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 6
			case 67:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return 6
			case 99:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return 7
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return 7
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 8
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 8
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return 9
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return 9
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return 10
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return 10
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 78:
				return 11
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 110:
				return 11
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [tT][rR][iI][gG][gG][eE][rR]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 84:
				return 1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 116:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 82:
				return 2
			case 84:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 114:
				return 2
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return 3
			case 82:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return 3
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return 4
			case 73:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 103:
				return 4
			case 105:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return 5
			case 73:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 103:
				return 5
			case 105:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 6
			case 71:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 101:
				return 6
			case 103:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 82:
				return 7
			case 84:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 114:
				return 7
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [tT][rR][uU][eE]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 82:
				return -1
			case 84:
				return 1
			case 85:
				return -1
			case 101:
				return -1
			case 114:
				return -1
			case 116:
				return 1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 82:
				return 2
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 114:
				return 2
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return 3
			case 101:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 4
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return 4
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [tT][rR][uU][nN][cC][aA][tT][eE]
	{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return 1
			case 85:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return 1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return 2
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return 2
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return 3
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return 4
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return 4
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return 5
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 99:
				return 5
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 6
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return 6
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return 7
			case 85:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return 7
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return 8
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return 8
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [uU][nN][bB][oO][uU][nN][dD][eE][dD]
	{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 85:
				return 1
			case 98:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 117:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return 2
			case 79:
				return -1
			case 85:
				return -1
			case 98:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return 2
			case 111:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return 3
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 85:
				return -1
			case 98:
				return 3
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 79:
				return 4
			case 85:
				return -1
			case 98:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 111:
				return 4
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 85:
				return 5
			case 98:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 117:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return 6
			case 79:
				return -1
			case 85:
				return -1
			case 98:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return 6
			case 111:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 68:
				return 7
			case 69:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 85:
				return -1
			case 98:
				return -1
			case 100:
				return 7
			case 101:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 68:
				return -1
			case 69:
				return 8
			case 78:
				return -1
			case 79:
				return -1
			case 85:
				return -1
			case 98:
				return -1
			case 100:
				return -1
			case 101:
				return 8
			case 110:
				return -1
			case 111:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 68:
				return 9
			case 69:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 85:
				return -1
			case 98:
				return -1
			case 100:
				return 9
			case 101:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 66:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 85:
				return -1
			case 98:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [uU][nN][dD][eE][rR]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 85:
				return 1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 117:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return 2
			case 82:
				return -1
			case 85:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return 2
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return 3
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 85:
				return -1
			case 100:
				return 3
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return 4
			case 78:
				return -1
			case 82:
				return -1
			case 85:
				return -1
			case 100:
				return -1
			case 101:
				return 4
			case 110:
				return -1
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return 5
			case 85:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return 5
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 85:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [uU][nN][iI][oO][nN]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 85:
				return 1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 117:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return 2
			case 79:
				return -1
			case 85:
				return -1
			case 105:
				return -1
			case 110:
				return 2
			case 111:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return 3
			case 78:
				return -1
			case 79:
				return -1
			case 85:
				return -1
			case 105:
				return 3
			case 110:
				return -1
			case 111:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return 4
			case 85:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return 4
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return 5
			case 79:
				return -1
			case 85:
				return -1
			case 105:
				return -1
			case 110:
				return 5
			case 111:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 85:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [uU][nN][iI][qQ][uU][eE]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 81:
				return -1
			case 85:
				return 1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 113:
				return -1
			case 117:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return 2
			case 81:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return 2
			case 113:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return 3
			case 78:
				return -1
			case 81:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 105:
				return 3
			case 110:
				return -1
			case 113:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 81:
				return 4
			case 85:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 113:
				return 4
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 81:
				return -1
			case 85:
				return 5
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 113:
				return -1
			case 117:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 6
			case 73:
				return -1
			case 78:
				return -1
			case 81:
				return -1
			case 85:
				return -1
			case 101:
				return 6
			case 105:
				return -1
			case 110:
				return -1
			case 113:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 81:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 113:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [uU][nN][kK][nN][oO][wW][nN]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 75:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 85:
				return 1
			case 87:
				return -1
			case 107:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 117:
				return 1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 75:
				return -1
			case 78:
				return 2
			case 79:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 107:
				return -1
			case 110:
				return 2
			case 111:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 75:
				return 3
			case 78:
				return -1
			case 79:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 107:
				return 3
			case 110:
				return -1
			case 111:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 75:
				return -1
			case 78:
				return 4
			case 79:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 107:
				return -1
			case 110:
				return 4
			case 111:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 75:
				return -1
			case 78:
				return -1
			case 79:
				return 5
			case 85:
				return -1
			case 87:
				return -1
			case 107:
				return -1
			case 110:
				return -1
			case 111:
				return 5
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 75:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 85:
				return -1
			case 87:
				return 6
			case 107:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 117:
				return -1
			case 119:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 75:
				return -1
			case 78:
				return 7
			case 79:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 107:
				return -1
			case 110:
				return 7
			case 111:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 75:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 107:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [uU][nN][nN][eE][sS][tT]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return 1
			case 101:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return 2
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 110:
				return 2
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return 3
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 110:
				return 3
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 4
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return 4
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return -1
			case 83:
				return 5
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 115:
				return 5
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return 6
			case 85:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return 6
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [uU][nN][sS][eE][tT]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return 1
			case 101:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return 2
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 110:
				return 2
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return -1
			case 83:
				return 3
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 115:
				return 3
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 4
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return 4
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return 5
			case 85:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return 5
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [uU][pP][dD][aA][tT][eE]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 80:
				return -1
			case 84:
				return -1
			case 85:
				return 1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 116:
				return -1
			case 117:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 80:
				return 2
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 112:
				return 2
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return 3
			case 69:
				return -1
			case 80:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 100:
				return 3
			case 101:
				return -1
			case 112:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 4
			case 68:
				return -1
			case 69:
				return -1
			case 80:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return 4
			case 100:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 80:
				return -1
			case 84:
				return 5
			case 85:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 116:
				return 5
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return 6
			case 80:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return 6
			case 112:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 80:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [uU][pP][sS][eE][rR][tT]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return 1
			case 101:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 80:
				return 2
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 112:
				return 2
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return 3
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return 3
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 4
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return 4
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 80:
				return -1
			case 82:
				return 5
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 114:
				return 5
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 6
			case 85:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 6
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [uU][sS][eE]
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 83:
				return -1
			case 85:
				return 1
			case 101:
				return -1
			case 115:
				return -1
			case 117:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 83:
				return 2
			case 85:
				return -1
			case 101:
				return -1
			case 115:
				return 2
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 3
			case 83:
				return -1
			case 85:
				return -1
			case 101:
				return 3
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [uU][sS][eE][rR]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 85:
				return 1
			case 101:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 117:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 82:
				return -1
			case 83:
				return 2
			case 85:
				return -1
			case 101:
				return -1
			case 114:
				return -1
			case 115:
				return 2
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 3
			case 82:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 101:
				return 3
			case 114:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 82:
				return 4
			case 83:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 114:
				return 4
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 101:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [uU][sS][iI][nN][gG]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 85:
				return 1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 117:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 83:
				return 2
			case 85:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 115:
				return 2
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 73:
				return 3
			case 78:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 103:
				return -1
			case 105:
				return 3
			case 110:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return 4
			case 83:
				return -1
			case 85:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return 4
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return 5
			case 73:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 103:
				return 5
			case 105:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [vV][aA][lL][iI][dD][aA][tT][eE]
	{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 84:
				return -1
			case 86:
				return 1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 116:
				return -1
			case 118:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return 2
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return 3
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return 3
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return 4
			case 76:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return 4
			case 108:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return 5
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 100:
				return 5
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 6
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return 6
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 84:
				return 7
			case 86:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 116:
				return 7
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return 8
			case 73:
				return -1
			case 76:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return 8
			case 105:
				return -1
			case 108:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [vV][aA][lL][uU][eE]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 86:
				return 1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			case 118:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 69:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 97:
				return 2
			case 101:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return 3
			case 85:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return 3
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 85:
				return 4
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 117:
				return 4
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 5
			case 76:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return 5
			case 108:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [vV][aA][lL][uU][eE][dD]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 86:
				return 1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			case 118:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 97:
				return 2
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return 3
			case 85:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return 3
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 85:
				return 4
			case 86:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 117:
				return 4
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return 5
			case 76:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return 5
			case 108:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return 6
			case 69:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 100:
				return 6
			case 101:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [vV][aA][lL][uU][eE][sS]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 86:
				return 1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			case 118:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 2
			case 69:
				return -1
			case 76:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 97:
				return 2
			case 101:
				return -1
			case 108:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return 3
			case 83:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return 3
			case 115:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 83:
				return -1
			case 85:
				return 4
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 115:
				return -1
			case 117:
				return 4
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return 5
			case 76:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return 5
			case 108:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 83:
				return 6
			case 85:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 115:
				return 6
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [vV][iI][aA]
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 86:
				return 1
			case 97:
				return -1
			case 105:
				return -1
			case 118:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return 2
			case 86:
				return -1
			case 97:
				return -1
			case 105:
				return 2
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 3
			case 73:
				return -1
			case 86:
				return -1
			case 97:
				return 3
			case 105:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 73:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 105:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [vV][iI][eE][wW]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 86:
				return 1
			case 87:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 118:
				return 1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return 2
			case 86:
				return -1
			case 87:
				return -1
			case 101:
				return -1
			case 105:
				return 2
			case 118:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 3
			case 73:
				return -1
			case 86:
				return -1
			case 87:
				return -1
			case 101:
				return 3
			case 105:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 86:
				return -1
			case 87:
				return 4
			case 101:
				return -1
			case 105:
				return -1
			case 118:
				return -1
			case 119:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 86:
				return -1
			case 87:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [wW][hH][eE][nN]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return -1
			case 78:
				return -1
			case 87:
				return 1
			case 101:
				return -1
			case 104:
				return -1
			case 110:
				return -1
			case 119:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return 2
			case 78:
				return -1
			case 87:
				return -1
			case 101:
				return -1
			case 104:
				return 2
			case 110:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 3
			case 72:
				return -1
			case 78:
				return -1
			case 87:
				return -1
			case 101:
				return 3
			case 104:
				return -1
			case 110:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return -1
			case 78:
				return 4
			case 87:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 110:
				return 4
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return -1
			case 78:
				return -1
			case 87:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 110:
				return -1
			case 119:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [wW][hH][eE][rR][eE]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return -1
			case 82:
				return -1
			case 87:
				return 1
			case 101:
				return -1
			case 104:
				return -1
			case 114:
				return -1
			case 119:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return 2
			case 82:
				return -1
			case 87:
				return -1
			case 101:
				return -1
			case 104:
				return 2
			case 114:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 3
			case 72:
				return -1
			case 82:
				return -1
			case 87:
				return -1
			case 101:
				return 3
			case 104:
				return -1
			case 114:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return -1
			case 82:
				return 4
			case 87:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 114:
				return 4
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 5
			case 72:
				return -1
			case 82:
				return -1
			case 87:
				return -1
			case 101:
				return 5
			case 104:
				return -1
			case 114:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return -1
			case 82:
				return -1
			case 87:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 114:
				return -1
			case 119:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [wW][hH][iI][lL][eE]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 87:
				return 1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 119:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return 2
			case 73:
				return -1
			case 76:
				return -1
			case 87:
				return -1
			case 101:
				return -1
			case 104:
				return 2
			case 105:
				return -1
			case 108:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return -1
			case 73:
				return 3
			case 76:
				return -1
			case 87:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return 3
			case 108:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return -1
			case 73:
				return -1
			case 76:
				return 4
			case 87:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return 4
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return 5
			case 72:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 87:
				return -1
			case 101:
				return 5
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 72:
				return -1
			case 73:
				return -1
			case 76:
				return -1
			case 87:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 119:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [wW][iI][nN][dD][oO][wW]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 87:
				return 1
			case 100:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 119:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 73:
				return 2
			case 78:
				return -1
			case 79:
				return -1
			case 87:
				return -1
			case 100:
				return -1
			case 105:
				return 2
			case 110:
				return -1
			case 111:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 73:
				return -1
			case 78:
				return 3
			case 79:
				return -1
			case 87:
				return -1
			case 100:
				return -1
			case 105:
				return -1
			case 110:
				return 3
			case 111:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return 4
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 87:
				return -1
			case 100:
				return 4
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return 5
			case 87:
				return -1
			case 100:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return 5
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 87:
				return 6
			case 100:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 119:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 87:
				return -1
			case 100:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 119:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [wW][iI][tT][hH]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 72:
				return -1
			case 73:
				return -1
			case 84:
				return -1
			case 87:
				return 1
			case 104:
				return -1
			case 105:
				return -1
			case 116:
				return -1
			case 119:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 72:
				return -1
			case 73:
				return 2
			case 84:
				return -1
			case 87:
				return -1
			case 104:
				return -1
			case 105:
				return 2
			case 116:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 72:
				return -1
			case 73:
				return -1
			case 84:
				return 3
			case 87:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 116:
				return 3
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 72:
				return 4
			case 73:
				return -1
			case 84:
				return -1
			case 87:
				return -1
			case 104:
				return 4
			case 105:
				return -1
			case 116:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 72:
				return -1
			case 73:
				return -1
			case 84:
				return -1
			case 87:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 116:
				return -1
			case 119:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [wW][iI][tT][hH][iI][nN]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 72:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 87:
				return 1
			case 104:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			case 119:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 72:
				return -1
			case 73:
				return 2
			case 78:
				return -1
			case 84:
				return -1
			case 87:
				return -1
			case 104:
				return -1
			case 105:
				return 2
			case 110:
				return -1
			case 116:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 72:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 84:
				return 3
			case 87:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 116:
				return 3
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 72:
				return 4
			case 73:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 87:
				return -1
			case 104:
				return 4
			case 105:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 72:
				return -1
			case 73:
				return 5
			case 78:
				return -1
			case 84:
				return -1
			case 87:
				return -1
			case 104:
				return -1
			case 105:
				return 5
			case 110:
				return -1
			case 116:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 72:
				return -1
			case 73:
				return -1
			case 78:
				return 6
			case 84:
				return -1
			case 87:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 110:
				return 6
			case 116:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 72:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 87:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			case 119:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [wW][oO][rR][kK]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 75:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 87:
				return 1
			case 107:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 119:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 75:
				return -1
			case 79:
				return 2
			case 82:
				return -1
			case 87:
				return -1
			case 107:
				return -1
			case 111:
				return 2
			case 114:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 75:
				return -1
			case 79:
				return -1
			case 82:
				return 3
			case 87:
				return -1
			case 107:
				return -1
			case 111:
				return -1
			case 114:
				return 3
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 75:
				return 4
			case 79:
				return -1
			case 82:
				return -1
			case 87:
				return -1
			case 107:
				return 4
			case 111:
				return -1
			case 114:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 75:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 87:
				return -1
			case 107:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 119:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [xX][oO][rR]
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 79:
				return -1
			case 82:
				return -1
			case 88:
				return 1
			case 111:
				return -1
			case 114:
				return -1
			case 120:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 79:
				return 2
			case 82:
				return -1
			case 88:
				return -1
			case 111:
				return 2
			case 114:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 79:
				return -1
			case 82:
				return 3
			case 88:
				return -1
			case 111:
				return -1
			case 114:
				return 3
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 79:
				return -1
			case 82:
				return -1
			case 88:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 120:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [a-zA-Z_][a-zA-Z0-9_]*
	{[]bool{false, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 95:
				return 1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 90:
				return 1
			case 97 <= r && r <= 122:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 95:
				return 2
			}
			switch {
			case 48 <= r && r <= 57:
				return 2
			case 65 <= r && r <= 90:
				return 2
			case 97 <= r && r <= 122:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 95:
				return 2
			}
			switch {
			case 48 <= r && r <= 57:
				return 2
			case 65 <= r && r <= 90:
				return 2
			case 97 <= r && r <= 122:
				return 2
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// [$|@][a-zA-Z_][a-zA-Z0-9_]*
	{[]bool{false, false, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 36:
				return 1
			case 64:
				return 1
			case 95:
				return -1
			case 124:
				return 1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 90:
				return -1
			case 97 <= r && r <= 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 36:
				return -1
			case 64:
				return -1
			case 95:
				return 2
			case 124:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 90:
				return 2
			case 97 <= r && r <= 122:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 36:
				return -1
			case 64:
				return -1
			case 95:
				return 3
			case 124:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 3
			case 65 <= r && r <= 90:
				return 3
			case 97 <= r && r <= 122:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 36:
				return -1
			case 64:
				return -1
			case 95:
				return 3
			case 124:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 3
			case 65 <= r && r <= 90:
				return 3
			case 97 <= r && r <= 122:
				return 3
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [$|@][1-9][0-9]*
	{[]bool{false, false, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 36:
				return 1
			case 64:
				return 1
			case 124:
				return 1
			}
			switch {
			case 48 <= r && r <= 48:
				return -1
			case 49 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 36:
				return -1
			case 64:
				return -1
			case 124:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return -1
			case 49 <= r && r <= 57:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 36:
				return -1
			case 64:
				return -1
			case 124:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return 3
			case 49 <= r && r <= 57:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 36:
				return -1
			case 64:
				return -1
			case 124:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return 3
			case 49 <= r && r <= 57:
				return 3
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// \?\?
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 63:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 63:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 63:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// \?
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 63:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 63:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 32:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \t
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 9:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 9:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \n
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 10:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 10:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// .
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			return 1
		},
		func(r rune) int {
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},
}

func NewLexer(in io.Reader) *Lexer {
	return NewLexerWithInit(in, nil)
}

func (yyLex *Lexer) Stop() {
	select {
	case yyLex.ch_stop <- true:
	default:
	}
}

// Text returns the matched text.
func (yylex *Lexer) Text() string {
	return yylex.stack[len(yylex.stack)-1].s
}

// Line returns the current line number.
// The first line is 0.
func (yylex *Lexer) Line() int {
	if len(yylex.stack) == 0 {
		return yylex.l
	}
	return yylex.stack[len(yylex.stack)-1].line
}

// Column returns the current column number.
// The first column is 0.
func (yylex *Lexer) Column() int {
	if len(yylex.stack) == 0 {
		return yylex.c
	}
	return yylex.stack[len(yylex.stack)-1].column
}

func (yylex *Lexer) next(lvl int) int {
	if lvl == len(yylex.stack) {
		l, c := 0, 0
		if lvl > 0 {
			l, c = yylex.stack[lvl-1].line, yylex.stack[lvl-1].column
		}
		yylex.stack = append(yylex.stack, frame{0, "", l, c})
	}
	if lvl == len(yylex.stack)-1 {
		p := &yylex.stack[lvl]
		*p = <-yylex.ch
		yylex.stale = false
	} else {
		yylex.stale = true
	}
	return yylex.stack[lvl].i
}
func (yylex *Lexer) pop() {
	l := len(yylex.stack) - 1
	yylex.l, yylex.c = yylex.stack[l].line, yylex.stack[l].column
	yylex.stack = yylex.stack[:l]
}
func (yylex Lexer) Error(e string) {
	panic(e)
}

// Lex runs the lexer.
// When the -s option is given, this function is not generated;
// instead, the NN_FUN macro runs the lexer.
// yySymType is expected to include the int fields, line and column.
func (yylex *Lexer) Lex(lval *yySymType) int {
OUTER0:
	for {
		next := yylex.next(0)
		lval.line = yylex.Line() + 1
		lval.column = yylex.Column() + 1
		switch next {
		case 0:
			{
				var e error

				lval.s, e = ProcessEscapeSequences(yylex.Text())
				yylex.logToken(yylex.Text(), "STR - [%s]", lval.s)
				if e != nil {
					yylex.reportError("invalid quoted string - " + e.Error())
					return _ERROR_
				}
				return STR
			}
		case 1:
			{
				var e error

				lval.s, e = ProcessEscapeSequences(yylex.Text())
				yylex.logToken(yylex.Text(), "STR - [%s]", lval.s)
				if e != nil {
					yylex.reportError("invalid quoted string - " + e.Error())
					return _ERROR_
				}
				return STR
			}
		case 2:
			{
				// Case-insensitive identifier
				var e error

				text := yylex.Text()
				text = text[0 : len(text)-1]
				lval.s, e = ProcessEscapeSequences(text)
				yylex.logToken(yylex.Text(), "IDENT_ICASE - %s", lval.s)
				if e != nil {
					yylex.reportError("invalid case insensitive identifier - " + e.Error())
					return _ERROR_
				}
				return IDENT_ICASE
			}
		case 3:
			{
				// Escaped identifier
				var e error

				lval.s, e = ProcessEscapeSequences(yylex.Text())
				yylex.logToken(yylex.Text(), "IDENT - %s", lval.s)
				if e != nil {
					yylex.reportError("invalid escaped identifier - " + e.Error())
					return _ERROR_
				}
				return IDENT
			}
		case 4:
			{
				// We differentiate NUM from INT
				lval.f, _ = strconv.ParseFloat(yylex.Text(), 64)
				yylex.logToken(yylex.Text(), "NUM - %f", lval.f)
				return NUM
			}
		case 5:
			{
				// We differentiate NUM from INT
				lval.f, _ = strconv.ParseFloat(yylex.Text(), 64)
				yylex.logToken(yylex.Text(), "NUM - %f", lval.f)
				return NUM
			}
		case 6:
			{
				// We differentiate NUM from INT
				lval.n, _ = strconv.ParseInt(yylex.Text(), 10, 64)
				if (lval.n > math.MinInt64 && lval.n < math.MaxInt64) || strconv.FormatInt(lval.n, 10) == yylex.Text() {
					yylex.logToken(yylex.Text(), "INT - %d", lval.n)
					return INT
				} else {
					lval.f, _ = strconv.ParseFloat(yylex.Text(), 64)
					yylex.logToken(yylex.Text(), "NUM - %f", lval.f)
					return NUM
				}
			}
		case 7:
			{
				yylex.reportError("invalid number")
				return _ERROR_
			}
		case 8:
			{
				s := yylex.Text()
				lval.s = s[2 : len(s)-2]
				return OPTIM_HINTS
			}
		case 9:
			{
				s := yylex.Text()
				lval.s = s[2:]
				return OPTIM_HINTS
			}
		case 10:
			{
				yylex.logToken(yylex.Text(), "BLOCK_COMMENT (length=%d)", len(yylex.Text())) /* eat up block comment */
			}
		case 11:
			{
				yylex.logToken(yylex.Text(), "LINE_COMMENT (length=%d)", len(yylex.Text())) /* eat up line comment */
			}
		case 12:
			{
				yylex.logToken(yylex.Text(), "WHITESPACE (count=%d)", len(yylex.Text())) /* eat up whitespace */
			}
		case 13:
			{
				yylex.logToken(yylex.Text(), "DOT")
				return DOT
			}
		case 14:
			{
				yylex.logToken(yylex.Text(), "PLUS")
				return PLUS
			}
		case 15:
			{
				yylex.logToken(yylex.Text(), "MINUS")
				return MINUS
			}
		case 16:
			{
				yylex.logToken(yylex.Text(), "MULT")
				return STAR
			}
		case 17:
			{
				yylex.logToken(yylex.Text(), "DIV")
				return DIV
			}
		case 18:
			{
				yylex.logToken(yylex.Text(), "MOD")
				return MOD
			}
		case 19:
			{
				yylex.logToken(yylex.Text(), "POW")
				return POW
			}
		case 20:
			{
				yylex.logToken(yylex.Text(), "DEQ")
				return DEQ
			}
		case 21:
			{
				yylex.logToken(yylex.Text(), "EQ")
				return EQ
			}
		case 22:
			{
				yylex.logToken(yylex.Text(), "NE")
				return NE
			}
		case 23:
			{
				yylex.logToken(yylex.Text(), "NE")
				return NE
			}
		case 24:
			{
				yylex.logToken(yylex.Text(), "LT")
				return LT
			}
		case 25:
			{
				yylex.logToken(yylex.Text(), "LTE")
				return LE
			}
		case 26:
			{
				yylex.logToken(yylex.Text(), "GT")
				return GT
			}
		case 27:
			{
				yylex.logToken(yylex.Text(), "GTE")
				return GE
			}
		case 28:
			{
				yylex.logToken(yylex.Text(), "CONCAT")
				return CONCAT
			}
		case 29:
			{
				yylex.logToken(yylex.Text(), "LPAREN")
				return LPAREN
			}
		case 30:
			{
				yylex.logToken(yylex.Text(), "RPAREN")
				return RPAREN
			}
		case 31:
			{
				yylex.logToken(yylex.Text(), "LBRACE")
				lval.tokOffset = yylex.curOffset
				return LBRACE
			}
		case 32:
			{
				lval.tokOffset = yylex.curOffset
				yylex.logToken(yylex.Text(), "RBRACE")
				return RBRACE
			}
		case 33:
			{
				yylex.logToken(yylex.Text(), "COMMA")
				return COMMA
			}
		case 34:
			{
				yylex.logToken(yylex.Text(), "COLON")
				return COLON
			}
		case 35:
			{
				yylex.logToken(yylex.Text(), "LBRACKET")
				return LBRACKET
			}
		case 36:
			{
				yylex.logToken(yylex.Text(), "RBRACKET")
				return RBRACKET
			}
		case 37:
			{
				yylex.logToken(yylex.Text(), "RBRACKET_ICASE")
				return RBRACKET_ICASE
			}
		case 38:
			{
				yylex.logToken(yylex.Text(), "SEMI")
				return SEMI
			}
		case 39:
			{
				yylex.logToken(yylex.Text(), "NOT_A_TOKEN")
				return NOT_A_TOKEN
			}
		case 40:
			{
				yylex.logToken(yylex.Text(), "_INDEX_CONDITION")
				return _INDEX_CONDITION
			}
		case 41:
			{
				yylex.logToken(yylex.Text(), "_INDEX_KEY")
				return _INDEX_KEY
			}
		case 42:
			{
				yylex.logToken(yylex.Text(), "ADVISE")
				lval.tokOffset = yylex.curOffset
				return ADVISE
			}
		case 43:
			{
				yylex.logToken(yylex.Text(), "ALL")
				return ALL
			}
		case 44:
			{
				yylex.logToken(yylex.Text(), "ALTER")
				return ALTER
			}
		case 45:
			{
				yylex.logToken(yylex.Text(), "ANALYZE")
				return ANALYZE
			}
		case 46:
			{
				yylex.logToken(yylex.Text(), "AND")
				return AND
			}
		case 47:
			{
				yylex.logToken(yylex.Text(), "ANY")
				return ANY
			}
		case 48:
			{
				yylex.logToken(yylex.Text(), "ARRAY")
				return ARRAY
			}
		case 49:
			{
				yylex.logToken(yylex.Text(), "AS")
				lval.tokOffset = yylex.curOffset
				return AS
			}
		case 50:
			{
				yylex.logToken(yylex.Text(), "ASC")
				return ASC
			}
		case 51:
			{
				yylex.logToken(yylex.Text(), "AT")
				return AT
			}
		case 52:
			{
				yylex.logToken(yylex.Text(), "BEGIN")
				return BEGIN
			}
		case 53:
			{
				yylex.logToken(yylex.Text(), "BETWEEN")
				return BETWEEN
			}
		case 54:
			{
				yylex.logToken(yylex.Text(), "BINARY")
				return BINARY
			}
		case 55:
			{
				yylex.logToken(yylex.Text(), "BOOLEAN")
				return BOOLEAN
			}
		case 56:
			{
				yylex.logToken(yylex.Text(), "BREAK")
				return BREAK
			}
		case 57:
			{
				yylex.logToken(yylex.Text(), "BUCKET")
				return BUCKET
			}
		case 58:
			{
				yylex.logToken(yylex.Text(), "BUILD")
				return BUILD
			}
		case 59:
			{
				yylex.logToken(yylex.Text(), "BY")
				return BY
			}
		case 60:
			{
				yylex.logToken(yylex.Text(), "CALL")
				return CALL
			}
		case 61:
			{
				yylex.logToken(yylex.Text(), "CASE")
				return CASE
			}
		case 62:
			{
				yylex.logToken(yylex.Text(), "CAST")
				return CAST
			}
		case 63:
			{
				yylex.logToken(yylex.Text(), "CLUSTER")
				return CLUSTER
			}
		case 64:
			{
				yylex.logToken(yylex.Text(), "COLLATE")
				return COLLATE
			}
		case 65:
			{
				yylex.logToken(yylex.Text(), "COLLECTION")
				return COLLECTION
			}
		case 66:
			{
				yylex.logToken(yylex.Text(), "COMMIT")
				return COMMIT
			}
		case 67:
			{
				yylex.logToken(yylex.Text(), "COMMITTED")
				return COMMITTED
			}
		case 68:
			{
				yylex.logToken(yylex.Text(), "CONNECT")
				return CONNECT
			}
		case 69:
			{
				yylex.logToken(yylex.Text(), "CONTINUE")
				return CONTINUE
			}
		case 70:
			{
				yylex.logToken(yylex.Text(), "_CORRELATED")
				return _CORRELATED
			}
		case 71:
			{
				yylex.logToken(yylex.Text(), "_COVER")
				return _COVER
			}
		case 72:
			{
				yylex.logToken(yylex.Text(), "CREATE")
				return CREATE
			}
		case 73:
			{
				yylex.logToken(yylex.Text(), "CURRENT")
				return CURRENT
			}
		case 74:
			{
				yylex.logToken(yylex.Text(), "CYCLE")
				return CYCLE
			}
		case 75:
			{
				yylex.logToken(yylex.Text(), "DATABASE")
				return DATABASE
			}
		case 76:
			{
				yylex.logToken(yylex.Text(), "DATASET")
				return DATASET
			}
		case 77:
			{
				yylex.logToken(yylex.Text(), "DATASTORE")
				return DATASTORE
			}
		case 78:
			{
				yylex.logToken(yylex.Text(), "DECLARE")
				return DECLARE
			}
		case 79:
			{
				yylex.logToken(yylex.Text(), "DECREMENT")
				return DECREMENT
			}
		case 80:
			{
				yylex.logToken(yylex.Text(), "DEFAULT")
				return DEFAULT
			}
		case 81:
			{
				yylex.logToken(yylex.Text(), "DELETE")
				return DELETE
			}
		case 82:
			{
				yylex.logToken(yylex.Text(), "DERIVED")
				return DERIVED
			}
		case 83:
			{
				yylex.logToken(yylex.Text(), "DESC")
				return DESC
			}
		case 84:
			{
				yylex.logToken(yylex.Text(), "DESCRIBE")
				return DESCRIBE
			}
		case 85:
			{
				yylex.logToken(yylex.Text(), "DISTINCT")
				return DISTINCT
			}
		case 86:
			{
				yylex.logToken(yylex.Text(), "DO")
				return DO
			}
		case 87:
			{
				yylex.logToken(yylex.Text(), "DROP")
				return DROP
			}
		case 88:
			{
				yylex.logToken(yylex.Text(), "EACH")
				return EACH
			}
		case 89:
			{
				yylex.logToken(yylex.Text(), "ELEMENT")
				return ELEMENT
			}
		case 90:
			{
				yylex.logToken(yylex.Text(), "ELSE")
				return ELSE
			}
		case 91:
			{
				yylex.logToken(yylex.Text(), "END")
				return END
			}
		case 92:
			{
				yylex.logToken(yylex.Text(), "ESCAPE")
				return ESCAPE
			}
		case 93:
			{
				yylex.logToken(yylex.Text(), "EVERY")
				return EVERY
			}
		case 94:
			{
				yylex.logToken(yylex.Text(), "EXCEPT")
				return EXCEPT
			}
		case 95:
			{
				yylex.logToken(yylex.Text(), "EXCLUDE")
				return EXCLUDE
			}
		case 96:
			{
				yylex.logToken(yylex.Text(), "EXECUTE")
				return EXECUTE
			}
		case 97:
			{
				yylex.logToken(yylex.Text(), "EXISTS")
				return EXISTS
			}
		case 98:
			{
				yylex.logToken(yylex.Text(), "EXPLAIN")
				lval.tokOffset = yylex.curOffset
				return EXPLAIN
			}
		case 99:
			{
				yylex.logToken(yylex.Text(), "FALSE")
				return FALSE
			}
		case 100:
			{
				yylex.logToken(yylex.Text(), "FETCH")
				return FETCH
			}
		case 101:
			{
				yylex.logToken(yylex.Text(), "FILTER")
				return FILTER
			}
		case 102:
			{
				yylex.logToken(yylex.Text(), "FIRST")
				return FIRST
			}
		case 103:
			{
				yylex.logToken(yylex.Text(), "FLATTEN")
				return FLATTEN
			}
		case 104:
			{
				yylex.logToken(yylex.Text(), "FLATTEN_KEYS")
				return FLATTEN_KEYS
			}
		case 105:
			{
				yylex.logToken(yylex.Text(), "FLUSH")
				return FLUSH
			}
		case 106:
			{
				yylex.logToken(yylex.Text(), "FOLLOWING")
				return FOLLOWING
			}
		case 107:
			{
				yylex.logToken(yylex.Text(), "FOR")
				return FOR
			}
		case 108:
			{
				yylex.logToken(yylex.Text(), "FORCE")
				lval.tokOffset = yylex.curOffset
				return FORCE
			}
		case 109:
			{
				yylex.logToken(yylex.Text(), "FROM")
				lval.tokOffset = yylex.curOffset
				return FROM
			}
		case 110:
			{
				yylex.logToken(yylex.Text(), "FTS")
				return FTS
			}
		case 111:
			{
				yylex.logToken(yylex.Text(), "FUNCTION")
				return FUNCTION
			}
		case 112:
			{
				yylex.logToken(yylex.Text(), "GOLANG")
				return GOLANG
			}
		case 113:
			{
				yylex.logToken(yylex.Text(), "GRANT")
				return GRANT
			}
		case 114:
			{
				yylex.logToken(yylex.Text(), "GROUP")
				return GROUP
			}
		case 115:
			{
				yylex.logToken(yylex.Text(), "GROUPS")
				return GROUPS
			}
		case 116:
			{
				yylex.logToken(yylex.Text(), "GSI")
				return GSI
			}
		case 117:
			{
				yylex.logToken(yylex.Text(), "HASH")
				return HASH
			}
		case 118:
			{
				yylex.logToken(yylex.Text(), "HAVING")
				return HAVING
			}
		case 119:
			{
				yylex.logToken(yylex.Text(), "IF")
				return IF
			}
		case 120:
			{
				yylex.logToken(yylex.Text(), "IGNORE")
				return IGNORE
			}
		case 121:
			{
				yylex.logToken(yylex.Text(), "ILIKE")
				return ILIKE
			}
		case 122:
			{
				yylex.logToken(yylex.Text(), "IN")
				return IN
			}
		case 123:
			{
				yylex.logToken(yylex.Text(), "INCLUDE")
				return INCLUDE
			}
		case 124:
			{
				yylex.logToken(yylex.Text(), "INCREMENT")
				return INCREMENT
			}
		case 125:
			{
				yylex.logToken(yylex.Text(), "INDEX")
				return INDEX
			}
		case 126:
			{
				yylex.logToken(yylex.Text(), "INFER")
				return INFER
			}
		case 127:
			{
				yylex.logToken(yylex.Text(), "INLINE")
				return INLINE
			}
		case 128:
			{
				yylex.logToken(yylex.Text(), "INNER")
				return INNER
			}
		case 129:
			{
				yylex.logToken(yylex.Text(), "INSERT")
				return INSERT
			}
		case 130:
			{
				yylex.logToken(yylex.Text(), "INTERSECT")
				return INTERSECT
			}
		case 131:
			{
				yylex.logToken(yylex.Text(), "INTO")
				return INTO
			}
		case 132:
			{
				yylex.logToken(yylex.Text(), "IS")
				return IS
			}
		case 133:
			{
				yylex.logToken(yylex.Text(), "ISOLATION")
				return ISOLATION
			}
		case 134:
			{
				yylex.logToken(yylex.Text(), "JAVASCRIPT")
				return JAVASCRIPT
			}
		case 135:
			{
				yylex.logToken(yylex.Text(), "JOIN")
				return JOIN
			}
		case 136:
			{
				yylex.logToken(yylex.Text(), "KEY")
				return KEY
			}
		case 137:
			{
				yylex.logToken(yylex.Text(), "KEYS")
				return KEYS
			}
		case 138:
			{
				yylex.logToken(yylex.Text(), "KEYSPACE")
				return KEYSPACE
			}
		case 139:
			{
				yylex.logToken(yylex.Text(), "KNOWN")
				return KNOWN
			}
		case 140:
			{
				yylex.logToken(yylex.Text(), "LANGUAGE")
				return LANGUAGE
			}
		case 141:
			{
				yylex.logToken(yylex.Text(), "LAST")
				return LAST
			}
		case 142:
			{
				yylex.logToken(yylex.Text(), "LATERAL")
				return LATERAL
			}
		case 143:
			{
				yylex.logToken(yylex.Text(), "LEFT")
				return LEFT
			}
		case 144:
			{
				yylex.logToken(yylex.Text(), "LET")
				return LET
			}
		case 145:
			{
				yylex.logToken(yylex.Text(), "LETTING")
				return LETTING
			}
		case 146:
			{
				yylex.logToken(yylex.Text(), "LEVEL")
				return LEVEL
			}
		case 147:
			{
				yylex.logToken(yylex.Text(), "LIKE")
				return LIKE
			}
		case 148:
			{
				yylex.logToken(yylex.Text(), "LIMIT")
				return LIMIT
			}
		case 149:
			{
				yylex.logToken(yylex.Text(), "LSM")
				return LSM
			}
		case 150:
			{
				yylex.logToken(yylex.Text(), "MAP")
				return MAP
			}
		case 151:
			{
				yylex.logToken(yylex.Text(), "MAPPING")
				return MAPPING
			}
		case 152:
			{
				yylex.logToken(yylex.Text(), "MATCHED")
				return MATCHED
			}
		case 153:
			{
				yylex.logToken(yylex.Text(), "MATERIALIZED")
				return MATERIALIZED
			}
		case 154:
			{
				yylex.logToken(yylex.Text(), "MERGE")
				return MERGE
			}
		case 155:
			{
				yylex.logToken(yylex.Text(), "MISSING")
				return MISSING
			}
		case 156:
			{
				yylex.logToken(yylex.Text(), "NAMESPACE")
				return NAMESPACE
			}
		case 157:
			{
				yylex.logToken(yylex.Text(), "NEST")
				return NEST
			}
		case 158:
			{
				yylex.logToken(yylex.Text(), "NL")
				return NL
			}
		case 159:
			{
				yylex.logToken(yylex.Text(), "NO")
				return NO
			}
		case 160:
			{
				yylex.logToken(yylex.Text(), "NOT")
				return NOT
			}
		case 161:
			{
				yylex.logToken(yylex.Text(), "NTH_VALUE")
				return NTH_VALUE
			}
		case 162:
			{
				yylex.logToken(yylex.Text(), "NULL")
				return NULL
			}
		case 163:
			{
				yylex.logToken(yylex.Text(), "NULLS")
				return NULLS
			}
		case 164:
			{
				yylex.logToken(yylex.Text(), "NUMBER")
				return NUMBER
			}
		case 165:
			{
				yylex.logToken(yylex.Text(), "OBJECT")
				return OBJECT
			}
		case 166:
			{
				yylex.logToken(yylex.Text(), "OFFSET")
				return OFFSET
			}
		case 167:
			{
				yylex.logToken(yylex.Text(), "ON")
				return ON
			}
		case 168:
			{
				yylex.logToken(yylex.Text(), "OPTION")
				return OPTION
			}
		case 169:
			{
				yylex.logToken(yylex.Text(), "OPTIONS")
				return OPTIONS
			}
		case 170:
			{
				yylex.logToken(yylex.Text(), "OR")
				return OR
			}
		case 171:
			{
				yylex.logToken(yylex.Text(), "ORDER")
				return ORDER
			}
		case 172:
			{
				yylex.logToken(yylex.Text(), "OTHERS")
				return OTHERS
			}
		case 173:
			{
				yylex.logToken(yylex.Text(), "OUTER")
				return OUTER
			}
		case 174:
			{
				yylex.logToken(yylex.Text(), "OVER")
				return OVER
			}
		case 175:
			{
				yylex.logToken(yylex.Text(), "PARSE")
				return PARSE
			}
		case 176:
			{
				yylex.logToken(yylex.Text(), "PARTITION")
				return PARTITION
			}
		case 177:
			{
				yylex.logToken(yylex.Text(), "PASSWORD")
				return PASSWORD
			}
		case 178:
			{
				yylex.logToken(yylex.Text(), "PATH")
				return PATH
			}
		case 179:
			{
				yylex.logToken(yylex.Text(), "POOL")
				return POOL
			}
		case 180:
			{
				yylex.logToken(yylex.Text(), "PRECEDING")
				return PRECEDING
			}
		case 181:
			{
				yylex.logToken(yylex.Text(), "PREPARE")
				lval.tokOffset = yylex.curOffset
				return PREPARE
			}
		case 182:
			{
				yylex.logToken(yylex.Text(), "PRIMARY")
				return PRIMARY
			}
		case 183:
			{
				yylex.logToken(yylex.Text(), "PRIVATE")
				return PRIVATE
			}
		case 184:
			{
				yylex.logToken(yylex.Text(), "PRIVILEGE")
				return PRIVILEGE
			}
		case 185:
			{
				yylex.logToken(yylex.Text(), "PROCEDURE")
				return PROCEDURE
			}
		case 186:
			{
				yylex.logToken(yylex.Text(), "PROBE")
				return PROBE
			}
		case 187:
			{
				yylex.logToken(yylex.Text(), "PUBLIC")
				return PUBLIC
			}
		case 188:
			{
				yylex.logToken(yylex.Text(), "RANGE")
				return RANGE
			}
		case 189:
			{
				yylex.logToken(yylex.Text(), "RAW")
				return RAW
			}
		case 190:
			{
				yylex.logToken(yylex.Text(), "READ")
				return READ
			}
		case 191:
			{
				yylex.logToken(yylex.Text(), "REALM")
				return REALM
			}
		case 192:
			{
				yylex.logToken(yylex.Text(), "RECURSIVE")
				return RECURSIVE
			}
		case 193:
			{
				yylex.logToken(yylex.Text(), "REDUCE")
				return REDUCE
			}
		case 194:
			{
				yylex.logToken(yylex.Text(), "RENAME")
				return RENAME
			}
		case 195:
			{
				yylex.logToken(yylex.Text(), "REPLACE")
				lval.s = yylex.Text()
				return REPLACE
			}
		case 196:
			{
				yylex.logToken(yylex.Text(), "RESPECT")
				return RESPECT
			}
		case 197:
			{
				yylex.logToken(yylex.Text(), "RESTRICT")
				return RESTRICT
			}
		case 198:
			{
				yylex.logToken(yylex.Text(), "RETURN")
				return RETURN
			}
		case 199:
			{
				yylex.logToken(yylex.Text(), "RETURNING")
				return RETURNING
			}
		case 200:
			{
				yylex.logToken(yylex.Text(), "REVOKE")
				return REVOKE
			}
		case 201:
			{
				yylex.logToken(yylex.Text(), "RIGHT")
				return RIGHT
			}
		case 202:
			{
				yylex.logToken(yylex.Text(), "ROLE")
				return ROLE
			}
		case 203:
			{
				yylex.logToken(yylex.Text(), "ROLLBACK")
				return ROLLBACK
			}
		case 204:
			{
				yylex.logToken(yylex.Text(), "ROW")
				return ROW
			}
		case 205:
			{
				yylex.logToken(yylex.Text(), "ROWS")
				return ROWS
			}
		case 206:
			{
				yylex.logToken(yylex.Text(), "SATISFIES")
				return SATISFIES
			}
		case 207:
			{
				yylex.logToken(yylex.Text(), "SAVEPOINT")
				return SAVEPOINT
			}
		case 208:
			{
				yylex.logToken(yylex.Text(), "SCHEMA")
				return SCHEMA
			}
		case 209:
			{
				yylex.logToken(yylex.Text(), "SCOPE")
				return SCOPE
			}
		case 210:
			{
				yylex.logToken(yylex.Text(), "SELECT")
				return SELECT
			}
		case 211:
			{
				yylex.logToken(yylex.Text(), "SELF")
				return SELF
			}
		case 212:
			{
				yylex.logToken(yylex.Text(), "SET")
				return SET
			}
		case 213:
			{
				yylex.logToken(yylex.Text(), "SHOW")
				return SHOW
			}
		case 214:
			{
				yylex.logToken(yylex.Text(), "SOME")
				return SOME
			}
		case 215:
			{
				yylex.logToken(yylex.Text(), "START")
				return START
			}
		case 216:
			{
				yylex.logToken(yylex.Text(), "STATISTICS")
				return STATISTICS
			}
		case 217:
			{
				yylex.logToken(yylex.Text(), "STRING")
				return STRING
			}
		case 218:
			{
				yylex.logToken(yylex.Text(), "SYSTEM")
				return SYSTEM
			}
		case 219:
			{
				yylex.logToken(yylex.Text(), "THEN")
				return THEN
			}
		case 220:
			{
				yylex.logToken(yylex.Text(), "TIES")
				return TIES
			}
		case 221:
			{
				yylex.logToken(yylex.Text(), "TO")
				return TO
			}
		case 222:
			{
				yylex.logToken(yylex.Text(), "TRAN")
				return TRAN
			}
		case 223:
			{
				yylex.logToken(yylex.Text(), "TRANSACTION")
				return TRANSACTION
			}
		case 224:
			{
				yylex.logToken(yylex.Text(), "TRIGGER")
				return TRIGGER
			}
		case 225:
			{
				yylex.logToken(yylex.Text(), "TRUE")
				return TRUE
			}
		case 226:
			{
				yylex.logToken(yylex.Text(), "TRUNCATE")
				return TRUNCATE
			}
		case 227:
			{
				yylex.logToken(yylex.Text(), "UNBOUNDED")
				return UNBOUNDED
			}
		case 228:
			{
				yylex.logToken(yylex.Text(), "UNDER")
				return UNDER
			}
		case 229:
			{
				yylex.logToken(yylex.Text(), "UNION")
				return UNION
			}
		case 230:
			{
				yylex.logToken(yylex.Text(), "UNIQUE")
				return UNIQUE
			}
		case 231:
			{
				yylex.logToken(yylex.Text(), "UNKNOWN")
				return UNKNOWN
			}
		case 232:
			{
				yylex.logToken(yylex.Text(), "UNNEST")
				return UNNEST
			}
		case 233:
			{
				yylex.logToken(yylex.Text(), "UNSET")
				return UNSET
			}
		case 234:
			{
				yylex.logToken(yylex.Text(), "UPDATE")
				return UPDATE
			}
		case 235:
			{
				yylex.logToken(yylex.Text(), "UPSERT")
				return UPSERT
			}
		case 236:
			{
				yylex.logToken(yylex.Text(), "USE")
				return USE
			}
		case 237:
			{
				yylex.logToken(yylex.Text(), "USER")
				return USER
			}
		case 238:
			{
				yylex.logToken(yylex.Text(), "USING")
				return USING
			}
		case 239:
			{
				yylex.logToken(yylex.Text(), "VALIDATE")
				return VALIDATE
			}
		case 240:
			{
				yylex.logToken(yylex.Text(), "VALUE")
				return VALUE
			}
		case 241:
			{
				yylex.logToken(yylex.Text(), "VALUED")
				return VALUED
			}
		case 242:
			{
				yylex.logToken(yylex.Text(), "VALUES")
				return VALUES
			}
		case 243:
			{
				yylex.logToken(yylex.Text(), "VIA")
				return VIA
			}
		case 244:
			{
				yylex.logToken(yylex.Text(), "VIEW")
				return VIEW
			}
		case 245:
			{
				yylex.logToken(yylex.Text(), "WHEN")
				return WHEN
			}
		case 246:
			{
				yylex.logToken(yylex.Text(), "WHERE")
				return WHERE
			}
		case 247:
			{
				yylex.logToken(yylex.Text(), "WHILE")
				return WHILE
			}
		case 248:
			{
				yylex.logToken(yylex.Text(), "WINDOW")
				return WINDOW
			}
		case 249:
			{
				yylex.logToken(yylex.Text(), "WITH")
				return WITH
			}
		case 250:
			{
				yylex.logToken(yylex.Text(), "WITHIN")
				return WITHIN
			}
		case 251:
			{
				yylex.logToken(yylex.Text(), "WORK")
				return WORK
			}
		case 252:
			{
				yylex.logToken(yylex.Text(), "XOR")
				return XOR
			}
		case 253:
			{
				lval.s = yylex.Text()
				yylex.logToken(yylex.Text(), "IDENT - %s", lval.s)
				return IDENT
			}
		case 254:
			{
				lval.s = yylex.Text()[1:]
				yylex.logToken(yylex.Text(), "NAMED_PARAM - %s", lval.s)
				return NAMED_PARAM
			}
		case 255:
			{
				lval.n, _ = strconv.ParseInt(yylex.Text()[1:], 10, 64)
				yylex.logToken(yylex.Text(), "POSITIONAL_PARAM - %d", lval.n)
				return POSITIONAL_PARAM
			}
		case 256:
			{
				yylex.logToken(yylex.Text(), "RANDOM_ELEMENT - ??")
				return RANDOM_ELEMENT
			}
		case 257:
			{
				lval.n = 0 // Handled by parser
				yylex.logToken(yylex.Text(), "NEXT_PARAM - ?")
				return NEXT_PARAM
			}
		case 258:
			{
				yylex.curOffset++
			}
		case 259:
			{
				yylex.curOffset++
			}
		case 260:
			{
				yylex.curOffset++
			}
		case 261:
			{
				/* this we don't know what it is: we'll let
				   the parser handle it (and most probably throw a syntax error
				*/
				yylex.logToken(yylex.Text(), "UNKNOWN token")
				return int([]byte(yylex.Text())[0])
			}
		default:
			break OUTER0
		}
		continue
	}
	yylex.pop()

	return 0
}
func (yylex *Lexer) logToken(text string, format string, v ...interface{}) {
	yylex.curOffset += len(text)
	clog.To("LEXER", format, v...)
}

func (yylex *Lexer) ResetOffset() {
	yylex.curOffset = 0
}

func (yylex *Lexer) ReportError(reportError func(what string)) {
	yylex.reportError = reportError
}
