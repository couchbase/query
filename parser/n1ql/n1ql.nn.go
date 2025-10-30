//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package n1ql

import "fmt"
import "math"
import "strconv"
import "github.com/couchbase/query/logging"
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

	// [cC][aA][cC][hH][eE]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return 1
			case 69:
				return -1
			case 72:
				return -1
			case 97:
				return -1
			case 99:
				return 1
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
				return 5
			case 72:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return 5
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
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

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

	// [mM][aA][xX][vV][aA][lL][uU][eE]
	{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 77:
				return 1
			case 85:
				return -1
			case 86:
				return -1
			case 88:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 109:
				return 1
			case 117:
				return -1
			case 118:
				return -1
			case 120:
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
			case 77:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 88:
				return -1
			case 97:
				return 2
			case 101:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 117:
				return -1
			case 118:
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
			case 76:
				return -1
			case 77:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 88:
				return 3
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			case 120:
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
			case 76:
				return -1
			case 77:
				return -1
			case 85:
				return -1
			case 86:
				return 4
			case 88:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 117:
				return -1
			case 118:
				return 4
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
			case 76:
				return -1
			case 77:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 88:
				return -1
			case 97:
				return 5
			case 101:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 117:
				return -1
			case 118:
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
			case 76:
				return 6
			case 77:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 88:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return 6
			case 109:
				return -1
			case 117:
				return -1
			case 118:
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
			case 76:
				return -1
			case 77:
				return -1
			case 85:
				return 7
			case 86:
				return -1
			case 88:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 117:
				return 7
			case 118:
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
				return 8
			case 76:
				return -1
			case 77:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 88:
				return -1
			case 97:
				return -1
			case 101:
				return 8
			case 108:
				return -1
			case 109:
				return -1
			case 117:
				return -1
			case 118:
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
			case 76:
				return -1
			case 77:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 88:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			case 120:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

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

	// [mM][iI][nN][vV][aA][lL][uU][eE]
	{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
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
			case 77:
				return 1
			case 78:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return 1
			case 110:
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
			case 73:
				return 2
			case 76:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return 2
			case 108:
				return -1
			case 109:
				return -1
			case 110:
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
			case 73:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 78:
				return 3
			case 85:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
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
			case 73:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 86:
				return 4
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 117:
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
			case 76:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 97:
				return 5
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
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
			case 73:
				return -1
			case 76:
				return 6
			case 77:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return 6
			case 109:
				return -1
			case 110:
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
			case 73:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 85:
				return 7
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 117:
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
			case 69:
				return 8
			case 73:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return 8
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
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
			case 73:
				return -1
			case 76:
				return -1
			case 77:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 117:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

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

	// [nN][eE][xX][tT]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return 1
			case 84:
				return -1
			case 88:
				return -1
			case 101:
				return -1
			case 110:
				return 1
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
				return 2
			case 78:
				return -1
			case 84:
				return -1
			case 88:
				return -1
			case 101:
				return 2
			case 110:
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
			case 78:
				return -1
			case 84:
				return -1
			case 88:
				return 3
			case 101:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			case 120:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 78:
				return -1
			case 84:
				return 4
			case 88:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 116:
				return 4
			case 120:
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
			case 84:
				return -1
			case 88:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [nN][eE][xX][tT][vV][aA][lL]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 78:
				return 1
			case 84:
				return -1
			case 86:
				return -1
			case 88:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 110:
				return 1
			case 116:
				return -1
			case 118:
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
				return 2
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 88:
				return -1
			case 97:
				return -1
			case 101:
				return 2
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			case 118:
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
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 88:
				return 3
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 120:
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
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return 4
			case 86:
				return -1
			case 88:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return 4
			case 118:
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
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 86:
				return 5
			case 88:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			case 118:
				return 5
			case 120:
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
			case 78:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 88:
				return -1
			case 97:
				return 6
			case 101:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			case 118:
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
			case 76:
				return 7
			case 78:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 88:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return 7
			case 110:
				return -1
			case 116:
				return -1
			case 118:
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
			case 76:
				return -1
			case 78:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 88:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 120:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

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

	// [pP][rR][eE][vV]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 80:
				return 1
			case 82:
				return -1
			case 86:
				return -1
			case 101:
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
			case 80:
				return -1
			case 82:
				return 2
			case 86:
				return -1
			case 101:
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
				return 3
			case 80:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 101:
				return 3
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
			case 80:
				return -1
			case 82:
				return -1
			case 86:
				return 4
			case 101:
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
			case 80:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 101:
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
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// [pP][rR][eE][vV][iI][oO][uU][sS]
	{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return -1
			case 79:
				return -1
			case 80:
				return 1
			case 82:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 111:
				return -1
			case 112:
				return 1
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
			case 69:
				return -1
			case 73:
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
			case 86:
				return -1
			case 101:
				return -1
			case 105:
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
			case 118:
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
			case 86:
				return -1
			case 101:
				return 3
			case 105:
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
			case 118:
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
			case 86:
				return 4
			case 101:
				return -1
			case 105:
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
			case 118:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 69:
				return -1
			case 73:
				return 5
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
			case 86:
				return -1
			case 101:
				return -1
			case 105:
				return 5
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
			case 118:
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
			case 79:
				return 6
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 86:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 111:
				return 6
			case 112:
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
			case 69:
				return -1
			case 73:
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
				return 7
			case 86:
				return -1
			case 101:
				return -1
			case 105:
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
				return 7
			case 118:
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
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return 8
			case 85:
				return -1
			case 86:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return 8
			case 117:
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
			case 73:
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
			case 86:
				return -1
			case 101:
				return -1
			case 105:
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
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [pP][rR][eE][vV][vV][aA][lL]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 80:
				return 1
			case 82:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
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
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 82:
				return 2
			case 86:
				return -1
			case 97:
				return -1
			case 101:
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
			case 65:
				return -1
			case 69:
				return 3
			case 76:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
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
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 86:
				return 4
			case 97:
				return -1
			case 101:
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
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 86:
				return 5
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 112:
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
			case 65:
				return 6
			case 69:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 97:
				return 6
			case 101:
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
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return 7
			case 80:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 108:
				return 7
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
			case 65:
				return -1
			case 69:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
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

	// [rR][eE][sS][tT][aA][rR][tT]
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 82:
				return 1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 101:
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
			case 65:
				return -1
			case 69:
				return 2
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 101:
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
			case 65:
				return -1
			case 69:
				return -1
			case 82:
				return -1
			case 83:
				return 3
			case 84:
				return -1
			case 97:
				return -1
			case 101:
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
			case 65:
				return -1
			case 69:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 4
			case 97:
				return -1
			case 101:
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
			case 65:
				return 5
			case 69:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return 5
			case 101:
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
			case 69:
				return -1
			case 82:
				return 6
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 114:
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
			case 69:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 7
			case 97:
				return -1
			case 101:
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
			case 65:
				return -1
			case 69:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 97:
				return -1
			case 101:
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

	// [rR][oO][lL][eE][sS]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
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
			case 83:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 114:
				return 1
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
			case 79:
				return 2
			case 82:
				return -1
			case 83:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return 2
			case 114:
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
				return 3
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 101:
				return -1
			case 108:
				return 3
			case 111:
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
			case 69:
				return 4
			case 76:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 101:
				return 4
			case 108:
				return -1
			case 111:
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
			case 69:
				return -1
			case 76:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return 5
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return 5
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
			case 83:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

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

	// [sS][aA][vV][eE]
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 69:
				return -1
			case 83:
				return 1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 115:
				return 1
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
			case 83:
				return -1
			case 86:
				return -1
			case 97:
				return 2
			case 101:
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
			case 69:
				return -1
			case 83:
				return -1
			case 86:
				return 3
			case 97:
				return -1
			case 101:
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
			case 69:
				return 4
			case 83:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
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
			case 69:
				return -1
			case 83:
				return -1
			case 86:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 115:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

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

	// [sS][eE][qQ][uU][eE][nN][cC][eE]
	{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 81:
				return -1
			case 83:
				return 1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 113:
				return -1
			case 115:
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
			case 69:
				return 2
			case 78:
				return -1
			case 81:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return 2
			case 110:
				return -1
			case 113:
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
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 81:
				return 3
			case 83:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 113:
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
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 81:
				return -1
			case 83:
				return -1
			case 85:
				return 4
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 113:
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
			case 67:
				return -1
			case 69:
				return 5
			case 78:
				return -1
			case 81:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return 5
			case 110:
				return -1
			case 113:
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
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return 6
			case 81:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return 6
			case 113:
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
			case 67:
				return 7
			case 69:
				return -1
			case 78:
				return -1
			case 81:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 99:
				return 7
			case 101:
				return -1
			case 110:
				return -1
			case 113:
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
			case 67:
				return -1
			case 69:
				return 8
			case 78:
				return -1
			case 81:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return 8
			case 110:
				return -1
			case 113:
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
			case 67:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 81:
				return -1
			case 83:
				return -1
			case 85:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 113:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

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

	// [uU][sS][eE][rR][sS]
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
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
				return 5
			case 85:
				return -1
			case 101:
				return -1
			case 114:
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
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

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

	// [vV][eE][cC][tT][oO][rR]
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
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
			case 84:
				return -1
			case 86:
				return 1
			case 99:
				return -1
			case 101:
				return -1
			case 111:
				return -1
			case 114:
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
			case 67:
				return -1
			case 69:
				return 2
			case 79:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 99:
				return -1
			case 101:
				return 2
			case 111:
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
			case 67:
				return 3
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 99:
				return 3
			case 101:
				return -1
			case 111:
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
			case 67:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 84:
				return 4
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
			case 116:
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
			case 79:
				return 5
			case 82:
				return -1
			case 84:
				return -1
			case 86:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 111:
				return 5
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
			case 67:
				return -1
			case 69:
				return -1
			case 79:
				return -1
			case 82:
				return 6
			case 84:
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
				return 6
			case 116:
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
			case 84:
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
			case 116:
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
				yylex.curOffset += len(yylex.Text())
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
				yylex.curOffset += len(yylex.Text())
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
				yylex.curOffset += len(yylex.Text())
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
				yylex.curOffset += len(yylex.Text())
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
				yylex.curOffset += len(yylex.Text())
				return NUM
			}
		case 5:
			{
				// We differentiate NUM from INT
				lval.f, _ = strconv.ParseFloat(yylex.Text(), 64)
				yylex.curOffset += len(yylex.Text())
				return NUM
			}
		case 6:
			{
				// We differentiate NUM from INT
				yylex.curOffset += len(yylex.Text())
				lval.n, _ = strconv.ParseInt(yylex.Text(), 10, 64)
				if (lval.n > math.MinInt64 && lval.n < math.MaxInt64) || strconv.FormatInt(lval.n, 10) == yylex.Text() {
					return INT
				} else {
					lval.f, _ = strconv.ParseFloat(yylex.Text(), 64)
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
				yylex.curOffset += len(s)
				return OPTIM_HINTS
			}
		case 9:
			{
				s := yylex.Text()
				lval.s = s[2:]
				yylex.curOffset += len(s)
				return OPTIM_HINTS
			}
		case 10:
			{ /* eat up block comment */
				yylex.curOffset += len(yylex.Text())
				yylex.logToken(yylex.Text(), "BLOCK_COMMENT (length=%d)", len(yylex.Text()))
			}
		case 11:
			{ /* eat up line comment */
				yylex.curOffset += len(yylex.Text())
				yylex.logToken(yylex.Text(), "LINE_COMMENT (length=%d)", len(yylex.Text()))
			}
		case 12:
			{ /* eat up whitespace */
				yylex.curOffset += len(yylex.Text())
			}
		case 13:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 1
				return DOT
			}
		case 14:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 1
				return PLUS
			}
		case 15:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 1
				return MINUS
			}
		case 16:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 1
				return STAR
			}
		case 17:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 1
				return DIV
			}
		case 18:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 1
				return MOD
			}
		case 19:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 1
				return POW
			}
		case 20:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 2
				return DEQ
			}
		case 21:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 1
				return EQ
			}
		case 22:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 2
				return NE
			}
		case 23:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 2
				return NE
			}
		case 24:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 1
				return LT
			}
		case 25:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 2
				return LE
			}
		case 26:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 1
				return GT
			}
		case 27:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 2
				return GE
			}
		case 28:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 2
				return CONCAT
			}
		case 29:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 1
				return LPAREN
			}
		case 30:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 1
				return RPAREN
			}
		case 31:
			{
				lval.s = yylex.Text()
				yylex.curOffset++
				lval.tokOffset = yylex.curOffset
				return LBRACE
			}
		case 32:
			{
				lval.tokOffset = yylex.curOffset
				lval.s = yylex.Text()
				yylex.curOffset++
				return RBRACE
			}
		case 33:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 1
				return COMMA
			}
		case 34:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 1
				return COLON
			}
		case 35:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 1
				return LBRACKET
			}
		case 36:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 1
				return RBRACKET
			}
		case 37:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 2
				return RBRACKET_ICASE
			}
		case 38:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 1
				return SEMI
			}
		case 39:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 1
				return NOT_A_TOKEN
			}
		case 40:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 16
				return _INDEX_CONDITION
			}
		case 41:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 10
				return _INDEX_KEY
			}
		case 42:
			{
				yylex.curOffset += 6
				lval.tokOffset = yylex.curOffset
				return ADVISE
			}
		case 43:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 3
				return ALL
			}
		case 44:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return ALTER
			}
		case 45:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return ANALYZE
			}
		case 46:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 3
				return AND
			}
		case 47:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 3
				return ANY
			}
		case 48:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return ARRAY
			}
		case 49:
			{
				yylex.curOffset += 2
				lval.tokOffset = yylex.curOffset
				return AS
			}
		case 50:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 3
				return ASC
			}
		case 51:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 2
				return AT
			}
		case 52:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return BEGIN
			}
		case 53:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return BETWEEN
			}
		case 54:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return BINARY
			}
		case 55:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return BOOLEAN
			}
		case 56:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return BREAK
			}
		case 57:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return BUCKET
			}
		case 58:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return BUILD
			}
		case 59:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 2
				return BY
			}
		case 60:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return CALL
			}
		case 61:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return CACHE
			}
		case 62:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return CASE
			}
		case 63:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return CAST
			}
		case 64:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return CLUSTER
			}
		case 65:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return COLLATE
			}
		case 66:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 10
				return COLLECTION
			}
		case 67:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return COMMIT
			}
		case 68:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 9
				return COMMITTED
			}
		case 69:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return CONNECT
			}
		case 70:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 8
				return CONTINUE
			}
		case 71:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 10
				return _CORRELATED
			}
		case 72:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return _COVER
			}
		case 73:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return CREATE
			}
		case 74:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return CURRENT
			}
		case 75:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return CYCLE
			}
		case 76:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 8
				return DATABASE
			}
		case 77:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return DATASET
			}
		case 78:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 9
				return DATASTORE
			}
		case 79:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return DECLARE
			}
		case 80:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 9
				return DECREMENT
			}
		case 81:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return DEFAULT
			}
		case 82:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return DELETE
			}
		case 83:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return DERIVED
			}
		case 84:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return DESC
			}
		case 85:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 8
				return DESCRIBE
			}
		case 86:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 8
				return DISTINCT
			}
		case 87:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 2
				return DO
			}
		case 88:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return DROP
			}
		case 89:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return EACH
			}
		case 90:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return ELEMENT
			}
		case 91:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return ELSE
			}
		case 92:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 3
				return END
			}
		case 93:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return ESCAPE
			}
		case 94:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return EVERY
			}
		case 95:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return EXCEPT
			}
		case 96:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return EXCLUDE
			}
		case 97:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return EXECUTE
			}
		case 98:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return EXISTS
			}
		case 99:
			{
				yylex.curOffset += 7
				lval.tokOffset = yylex.curOffset
				return EXPLAIN
			}
		case 100:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return FALSE
			}
		case 101:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return FETCH
			}
		case 102:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return FILTER
			}
		case 103:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return FIRST
			}
		case 104:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return FLATTEN
			}
		case 105:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 12
				return FLATTEN_KEYS
			}
		case 106:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return FLUSH
			}
		case 107:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 9
				return FOLLOWING
			}
		case 108:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 3
				return FOR
			}
		case 109:
			{
				yylex.curOffset += 5
				lval.tokOffset = yylex.curOffset
				return FORCE
			}
		case 110:
			{
				yylex.curOffset += 4
				lval.tokOffset = yylex.curOffset
				return FROM
			}
		case 111:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 3
				return FTS
			}
		case 112:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 8
				return FUNCTION
			}
		case 113:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return GOLANG
			}
		case 114:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return GRANT
			}
		case 115:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return GROUP
			}
		case 116:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return GROUPS
			}
		case 117:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 3
				return GSI
			}
		case 118:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return HASH
			}
		case 119:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return HAVING
			}
		case 120:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 2
				return IF
			}
		case 121:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return IGNORE
			}
		case 122:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return ILIKE
			}
		case 123:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 2
				return IN
			}
		case 124:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return INCLUDE
			}
		case 125:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 9
				return INCREMENT
			}
		case 126:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return INDEX
			}
		case 127:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return INFER
			}
		case 128:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return INLINE
			}
		case 129:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return INNER
			}
		case 130:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return INSERT
			}
		case 131:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 9
				return INTERSECT
			}
		case 132:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return INTO
			}
		case 133:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 2
				return IS
			}
		case 134:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 9
				return ISOLATION
			}
		case 135:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 10
				return JAVASCRIPT
			}
		case 136:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return JOIN
			}
		case 137:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 3
				return KEY
			}
		case 138:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return KEYS
			}
		case 139:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 8
				return KEYSPACE
			}
		case 140:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return KNOWN
			}
		case 141:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 8
				return LANGUAGE
			}
		case 142:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return LAST
			}
		case 143:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return LATERAL
			}
		case 144:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return LEFT
			}
		case 145:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 3
				return LET
			}
		case 146:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return LETTING
			}
		case 147:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return LEVEL
			}
		case 148:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return LIKE
			}
		case 149:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return LIMIT
			}
		case 150:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 3
				return LSM
			}
		case 151:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 3
				return MAP
			}
		case 152:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return MAPPING
			}
		case 153:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return MATCHED
			}
		case 154:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 12
				return MATERIALIZED
			}
		case 155:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 8
				return MAXVALUE
			}
		case 156:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return MERGE
			}
		case 157:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 8
				return MINVALUE
			}
		case 158:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return MISSING
			}
		case 159:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 9
				return NAMESPACE
			}
		case 160:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return NEST
			}
		case 161:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return NEXT
			}
		case 162:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return NEXTVAL
			}
		case 163:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 2
				return NL
			}
		case 164:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 2
				return NO
			}
		case 165:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 3
				return NOT
			}
		case 166:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 9
				return NTH_VALUE
			}
		case 167:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return NULL
			}
		case 168:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return NULLS
			}
		case 169:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return NUMBER
			}
		case 170:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return OBJECT
			}
		case 171:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return OFFSET
			}
		case 172:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 2
				return ON
			}
		case 173:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return OPTION
			}
		case 174:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return OPTIONS
			}
		case 175:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 2
				return OR
			}
		case 176:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return ORDER
			}
		case 177:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return OTHERS
			}
		case 178:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return OUTER
			}
		case 179:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return OVER
			}
		case 180:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return PARSE
			}
		case 181:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 9
				return PARTITION
			}
		case 182:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 8
				return PASSWORD
			}
		case 183:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return PATH
			}
		case 184:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return POOL
			}
		case 185:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 9
				return PRECEDING
			}
		case 186:
			{
				yylex.curOffset += 7
				lval.tokOffset = yylex.curOffset
				return PREPARE
			}
		case 187:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return PREV
			}
		case 188:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 8
				return PREV
			}
		case 189:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return PREVVAL
			}
		case 190:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return PRIMARY
			}
		case 191:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return PRIVATE
			}
		case 192:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 9
				return PRIVILEGE
			}
		case 193:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 9
				return PROCEDURE
			}
		case 194:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return PROBE
			}
		case 195:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return PUBLIC
			}
		case 196:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return RANGE
			}
		case 197:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 3
				return RAW
			}
		case 198:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return READ
			}
		case 199:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return REALM
			}
		case 200:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 9
				return RECURSIVE
			}
		case 201:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return REDUCE
			}
		case 202:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return RENAME
			}
		case 203:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return REPLACE
			}
		case 204:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return RESPECT
			}
		case 205:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return RESTART
			}
		case 206:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 8
				return RESTRICT
			}
		case 207:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return RETURN
			}
		case 208:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 9
				return RETURNING
			}
		case 209:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return REVOKE
			}
		case 210:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return RIGHT
			}
		case 211:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return ROLE
			}
		case 212:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return ROLES
			}
		case 213:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 8
				return ROLLBACK
			}
		case 214:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 3
				return ROW
			}
		case 215:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return ROWS
			}
		case 216:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 9
				return SATISFIES
			}
		case 217:
			{
				yylex.curOffset += 4
				lval.tokOffset = yylex.curOffset
				return SAVE
			}
		case 218:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 9
				return SAVEPOINT
			}
		case 219:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return SCHEMA
			}
		case 220:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return SCOPE
			}
		case 221:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return SELECT
			}
		case 222:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return SELF
			}
		case 223:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 8
				return SEQUENCE
			}
		case 224:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 3
				return SET
			}
		case 225:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return SHOW
			}
		case 226:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return SOME
			}
		case 227:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return START
			}
		case 228:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 10
				return STATISTICS
			}
		case 229:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return STRING
			}
		case 230:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return SYSTEM
			}
		case 231:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return THEN
			}
		case 232:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return TIES
			}
		case 233:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 2
				return TO
			}
		case 234:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return TRAN
			}
		case 235:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 11
				return TRANSACTION
			}
		case 236:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return TRIGGER
			}
		case 237:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return TRUE
			}
		case 238:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 8
				return TRUNCATE
			}
		case 239:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 9
				return UNBOUNDED
			}
		case 240:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return UNDER
			}
		case 241:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return UNION
			}
		case 242:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return UNIQUE
			}
		case 243:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 7
				return UNKNOWN
			}
		case 244:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return UNNEST
			}
		case 245:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return UNSET
			}
		case 246:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return UPDATE
			}
		case 247:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return UPSERT
			}
		case 248:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 3
				return USE
			}
		case 249:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return USER
			}
		case 250:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return USERS
			}
		case 251:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return USING
			}
		case 252:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 8
				return VALIDATE
			}
		case 253:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return VALUE
			}
		case 254:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return VALUED
			}
		case 255:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return VALUES
			}
		case 256:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return VECTOR
			}
		case 257:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 3
				return VIA
			}
		case 258:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return VIEW
			}
		case 259:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return WHEN
			}
		case 260:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return WHERE
			}
		case 261:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 5
				return WHILE
			}
		case 262:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return WINDOW
			}
		case 263:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return WITH
			}
		case 264:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 6
				return WITHIN
			}
		case 265:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 4
				return WORK
			}
		case 266:
			{
				lval.s = yylex.Text()
				yylex.curOffset += 3
				return XOR
			}
		case 267:
			{
				lval.s = yylex.Text()
				yylex.curOffset += len(lval.s)
				return IDENT
			}
		case 268:
			{
				lval.s = yylex.Text()[1:]
				yylex.curOffset += len(yylex.Text())
				return NAMED_PARAM
			}
		case 269:
			{
				lval.n, _ = strconv.ParseInt(yylex.Text()[1:], 10, 64)
				yylex.curOffset += len(yylex.Text())
				return POSITIONAL_PARAM
			}
		case 270:
			{
				yylex.curOffset += 2
				return RANDOM_ELEMENT
			}
		case 271:
			{
				lval.n = 0 // Handled by parser
				yylex.curOffset++
				return NEXT_PARAM
			}
		case 272:
			{
				yylex.curOffset++
			}
		case 273:
			{
				yylex.curOffset++
			}
		case 274:
			{
				yylex.curOffset++
			}
		case 275:
			{
				/* this we don't know what it is: we'll let
				   the parser handle it (and most probably throw a syntax error
				*/
				yylex.curOffset += len(yylex.Text())
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
	if logging.LogLevel() == logging.TRACE {
		s := fmt.Sprintf(format, v...)
		logging.Tracef("Token: >>%s<< - %s", text, s)
	}
}

func (yylex *Lexer) ResetOffset() {
	yylex.curOffset = 0
}

func (yylex *Lexer) ReportError(reportError func(what string)) {
	yylex.reportError = reportError
}
