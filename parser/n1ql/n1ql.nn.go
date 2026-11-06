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

	parseResult any

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
				// Keywords are matched here through a map lookup (see keywords.go)
				lval.s = yylex.Text()
				yylex.curOffset += len(lval.s)
				if t, ok := lookupKeyword(lval.s); ok {
					// ADVISE, AS, EXPLAIN, FORCE, FROM, PREPARE and SAVE need the offset
					// past the keyword (Remainder(), setOffset(), inline UDF bodies);
					// setting it for every keyword is harmless
					lval.tokOffset = yylex.curOffset
					return t
				}
				return IDENT
			}
		case 41:
			{
				lval.s = yylex.Text()[1:]
				yylex.curOffset += len(yylex.Text())
				return NAMED_PARAM
			}
		case 42:
			{
				lval.n, _ = strconv.ParseInt(yylex.Text()[1:], 10, 64)
				yylex.curOffset += len(yylex.Text())
				return POSITIONAL_PARAM
			}
		case 43:
			{
				yylex.curOffset += 2
				return RANDOM_ELEMENT
			}
		case 44:
			{
				lval.n = 0 // Handled by parser
				yylex.curOffset++
				return NEXT_PARAM
			}
		case 45:
			{
				yylex.curOffset++
			}
		case 46:
			{
				yylex.curOffset++
			}
		case 47:
			{
				yylex.curOffset++
			}
		case 48:
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
func (yylex *Lexer) logToken(text string, format string, v ...any) {
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
