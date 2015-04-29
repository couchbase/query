package n1ql

import "strconv"
import "github.com/couchbase/clog"
import (
	"bufio"
	"io"
	"strings"
)

type intstring struct {
	i int
	s string
}
type Lexer struct {
	// The lexer runs in its own goroutine, and communicates via channel 'ch'.
	ch chan intstring
	// We record the level of nesting because the action could return, and a
	// subsequent call expects to pick up where it left off. In other words,
	// we're simulating a coroutine.
	// TODO: Support a channel-based variant that compatible with Go's yacc.
	stack []intstring
	stale bool

	// The 'l' and 'c' fields were added for
	// https://github.com/wagerlabs/docker/blob/65694e801a7b80930961d70c69cba9f2465459be/buildfile.nex
	l, c int // line number and character position
	// The following line makes it easy for scripts to insert fields in the
	// generated code.
	// [NEX_END_OF_LEXER_STRUCT]
}

// NewLexerWithInit creates a new Lexer object, runs the given callback on it,
// then returns it.
func NewLexerWithInit(in io.Reader, initFun func(*Lexer)) *Lexer {
	type dfa struct {
		acc          []bool           // Accepting states.
		f            []func(rune) int // Transitions.
		startf, endf []int            // Transitions at start and end of input.
		nest         []dfa
	}
	yylex := new(Lexer)
	if initFun != nil {
		initFun(yylex)
	}
	yylex.ch = make(chan intstring)
	var scan func(in *bufio.Reader, ch chan intstring, family []dfa)
	scan = func(in *bufio.Reader, ch chan intstring, family []dfa) {
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
		var state [][2]int
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
				var nextState [][2]int
				for _, x := range state {
					x[1] = family[x[0]].f[x[1]](r)
					if -1 == x[1] {
						continue
					}
					nextState = append(nextState, x)
					checkAccept(x[0], x[1])
				}
				state = nextState
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
				state = nil
			}

			if state == nil {
				// All DFAs stuck. Return last match if it exists, otherwise advance by one rune and restart all DFAs.
				if matchn == -1 {
					if len(buf) == 0 { // This can only happen at the end of input.
						break
					}
					buf = buf[1:]
				} else {
					text := string(buf[:matchn])
					buf = buf[matchn:]
					matchn = -1
					ch <- intstring{matchi, text}
					if len(family[matchi].nest) > 0 {
						scan(bufio.NewReader(strings.NewReader(text)), ch, family[matchi].nest)
					}
					if atEOF {
						break
					}
				}
				n = 0
				for i := 0; i < len(family); i++ {
					state = append(state, [2]int{i, 0})
				}
			}
		}
		ch <- intstring{-1, ""}
	}
	go scan(bufio.NewReader(in), yylex.ch, []dfa{
		// \"((\\\\)|(\\\")|(\\\/)|(\\b)|(\\f)|(\\n)|(\\r)|(\\t)|(\\u([0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F]){4})|[^\\\"])*\"
		{[]bool{false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 92:
					return -1
				case 102:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return -1
				case 34:
					return 1
				case 47:
					return -1
				case 98:
					return -1
				case 116:
					return -1
				case 125:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 92:
					return 2
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				case 34:
					return 4
				case 47:
					return 3
				case 98:
					return 3
				case 116:
					return 3
				case 125:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 92:
					return 5
				case 102:
					return 6
				case 110:
					return 7
				case 114:
					return 8
				case 117:
					return 9
				case 123:
					return -1
				case 52:
					return -1
				case 34:
					return 10
				case 47:
					return 11
				case 98:
					return 12
				case 116:
					return 13
				case 125:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 92:
					return 2
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				case 34:
					return 4
				case 47:
					return 3
				case 98:
					return 3
				case 116:
					return 3
				case 125:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 34:
					return -1
				case 47:
					return -1
				case 98:
					return -1
				case 116:
					return -1
				case 125:
					return -1
				case 92:
					return -1
				case 102:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 92:
					return 2
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				case 34:
					return 4
				case 47:
					return 3
				case 98:
					return 3
				case 116:
					return 3
				case 125:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 34:
					return 4
				case 47:
					return 3
				case 98:
					return 3
				case 116:
					return 3
				case 125:
					return 3
				case 92:
					return 2
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 34:
					return 4
				case 47:
					return 3
				case 98:
					return 3
				case 116:
					return 3
				case 125:
					return 3
				case 92:
					return 2
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 92:
					return 2
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				case 34:
					return 4
				case 47:
					return 3
				case 98:
					return 3
				case 116:
					return 3
				case 125:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 92:
					return -1
				case 102:
					return 14
				case 110:
					return -1
				case 114:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return 14
				case 34:
					return -1
				case 47:
					return -1
				case 98:
					return 14
				case 116:
					return -1
				case 125:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 14
				case 65 <= r && r <= 70:
					return 14
				case 97 <= r && r <= 102:
					return 14
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 92:
					return 2
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				case 34:
					return 4
				case 47:
					return 3
				case 98:
					return 3
				case 116:
					return 3
				case 125:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 34:
					return 4
				case 47:
					return 3
				case 98:
					return 3
				case 116:
					return 3
				case 125:
					return 3
				case 92:
					return 2
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 92:
					return 2
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				case 34:
					return 4
				case 47:
					return 3
				case 98:
					return 3
				case 116:
					return 3
				case 125:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 92:
					return 2
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				case 34:
					return 4
				case 47:
					return 3
				case 98:
					return 3
				case 116:
					return 3
				case 125:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 92:
					return -1
				case 102:
					return 15
				case 110:
					return -1
				case 114:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return 15
				case 34:
					return -1
				case 47:
					return -1
				case 98:
					return 15
				case 116:
					return -1
				case 125:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 15
				case 65 <= r && r <= 70:
					return 15
				case 97 <= r && r <= 102:
					return 15
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 34:
					return -1
				case 47:
					return -1
				case 98:
					return 16
				case 116:
					return -1
				case 125:
					return -1
				case 92:
					return -1
				case 102:
					return 16
				case 110:
					return -1
				case 114:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return 16
				}
				switch {
				case 48 <= r && r <= 57:
					return 16
				case 65 <= r && r <= 70:
					return 16
				case 97 <= r && r <= 102:
					return 16
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 34:
					return -1
				case 47:
					return -1
				case 98:
					return 17
				case 116:
					return -1
				case 125:
					return -1
				case 92:
					return -1
				case 102:
					return 17
				case 110:
					return -1
				case 114:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return 17
				}
				switch {
				case 48 <= r && r <= 57:
					return 17
				case 65 <= r && r <= 70:
					return 17
				case 97 <= r && r <= 102:
					return 17
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 34:
					return -1
				case 47:
					return -1
				case 98:
					return -1
				case 116:
					return -1
				case 125:
					return -1
				case 92:
					return -1
				case 102:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 117:
					return -1
				case 123:
					return 18
				case 52:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 34:
					return -1
				case 47:
					return -1
				case 98:
					return -1
				case 116:
					return -1
				case 125:
					return -1
				case 92:
					return -1
				case 102:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return 19
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 34:
					return -1
				case 47:
					return -1
				case 98:
					return -1
				case 116:
					return -1
				case 125:
					return 20
				case 92:
					return -1
				case 102:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 34:
					return 4
				case 47:
					return 3
				case 98:
					return 3
				case 116:
					return 3
				case 125:
					return 3
				case 92:
					return 2
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// '((\\\\)|(\\\/)|(\\b)|(\\f)|(\\n)|(\\r)|(\\t)|(\\u([0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F]){4})|('')|[^\\'])*'
		{[]bool{false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 39:
					return 1
				case 47:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 123:
					return -1
				case 125:
					return -1
				case 92:
					return -1
				case 98:
					return -1
				case 102:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 52:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 92:
					return 2
				case 98:
					return 3
				case 102:
					return 3
				case 116:
					return 3
				case 117:
					return 3
				case 52:
					return 3
				case 39:
					return 4
				case 47:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 123:
					return 3
				case 125:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 92:
					return 6
				case 98:
					return 7
				case 102:
					return 8
				case 116:
					return 9
				case 117:
					return 10
				case 52:
					return -1
				case 39:
					return -1
				case 47:
					return 11
				case 110:
					return 12
				case 114:
					return 13
				case 123:
					return -1
				case 125:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 39:
					return 4
				case 47:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 123:
					return 3
				case 125:
					return 3
				case 92:
					return 2
				case 98:
					return 3
				case 102:
					return 3
				case 116:
					return 3
				case 117:
					return 3
				case 52:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 92:
					return -1
				case 98:
					return -1
				case 102:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 52:
					return -1
				case 39:
					return 5
				case 47:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 123:
					return -1
				case 125:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 39:
					return 4
				case 47:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 123:
					return 3
				case 125:
					return 3
				case 92:
					return 2
				case 98:
					return 3
				case 102:
					return 3
				case 116:
					return 3
				case 117:
					return 3
				case 52:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 92:
					return 2
				case 98:
					return 3
				case 102:
					return 3
				case 116:
					return 3
				case 117:
					return 3
				case 52:
					return 3
				case 39:
					return 4
				case 47:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 123:
					return 3
				case 125:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 39:
					return 4
				case 47:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 123:
					return 3
				case 125:
					return 3
				case 92:
					return 2
				case 98:
					return 3
				case 102:
					return 3
				case 116:
					return 3
				case 117:
					return 3
				case 52:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 39:
					return 4
				case 47:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 123:
					return 3
				case 125:
					return 3
				case 92:
					return 2
				case 98:
					return 3
				case 102:
					return 3
				case 116:
					return 3
				case 117:
					return 3
				case 52:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 92:
					return 2
				case 98:
					return 3
				case 102:
					return 3
				case 116:
					return 3
				case 117:
					return 3
				case 52:
					return 3
				case 39:
					return 4
				case 47:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 123:
					return 3
				case 125:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 39:
					return -1
				case 47:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 123:
					return -1
				case 125:
					return -1
				case 92:
					return -1
				case 98:
					return 14
				case 102:
					return 14
				case 116:
					return -1
				case 117:
					return -1
				case 52:
					return 14
				}
				switch {
				case 48 <= r && r <= 57:
					return 14
				case 65 <= r && r <= 70:
					return 14
				case 97 <= r && r <= 102:
					return 14
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 92:
					return 2
				case 98:
					return 3
				case 102:
					return 3
				case 116:
					return 3
				case 117:
					return 3
				case 52:
					return 3
				case 39:
					return 4
				case 47:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 123:
					return 3
				case 125:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 39:
					return 4
				case 47:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 123:
					return 3
				case 125:
					return 3
				case 92:
					return 2
				case 98:
					return 3
				case 102:
					return 3
				case 116:
					return 3
				case 117:
					return 3
				case 52:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 92:
					return 2
				case 98:
					return 3
				case 102:
					return 3
				case 116:
					return 3
				case 117:
					return 3
				case 52:
					return 3
				case 39:
					return 4
				case 47:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 123:
					return 3
				case 125:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 92:
					return -1
				case 98:
					return 15
				case 102:
					return 15
				case 116:
					return -1
				case 117:
					return -1
				case 52:
					return 15
				case 39:
					return -1
				case 47:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 123:
					return -1
				case 125:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 15
				case 65 <= r && r <= 70:
					return 15
				case 97 <= r && r <= 102:
					return 15
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 92:
					return -1
				case 98:
					return 16
				case 102:
					return 16
				case 116:
					return -1
				case 117:
					return -1
				case 52:
					return 16
				case 39:
					return -1
				case 47:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 123:
					return -1
				case 125:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 16
				case 65 <= r && r <= 70:
					return 16
				case 97 <= r && r <= 102:
					return 16
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 92:
					return -1
				case 98:
					return 17
				case 102:
					return 17
				case 116:
					return -1
				case 117:
					return -1
				case 52:
					return 17
				case 39:
					return -1
				case 47:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 123:
					return -1
				case 125:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 17
				case 65 <= r && r <= 70:
					return 17
				case 97 <= r && r <= 102:
					return 17
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 92:
					return -1
				case 98:
					return -1
				case 102:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 52:
					return -1
				case 39:
					return -1
				case 47:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 123:
					return 18
				case 125:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 39:
					return -1
				case 47:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 123:
					return -1
				case 125:
					return -1
				case 92:
					return -1
				case 98:
					return -1
				case 102:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 52:
					return 19
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 39:
					return -1
				case 47:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 123:
					return -1
				case 125:
					return 20
				case 92:
					return -1
				case 98:
					return -1
				case 102:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 52:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 39:
					return 4
				case 47:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 123:
					return 3
				case 125:
					return 3
				case 92:
					return 2
				case 98:
					return 3
				case 102:
					return 3
				case 116:
					return 3
				case 117:
					return 3
				case 52:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// `((\\\\)|(\\\/)|(\\b)|(\\f)|(\\n)|(\\r)|(\\t)|(\\u([0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F]){4})|(``)|[^\\`])+`i
		{[]bool{false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 96:
					return 1
				case 102:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 116:
					return -1
				case 92:
					return -1
				case 47:
					return -1
				case 98:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return -1
				case 125:
					return -1
				case 105:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 92:
					return 2
				case 47:
					return 3
				case 98:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				case 125:
					return 3
				case 105:
					return 3
				case 96:
					return 4
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 116:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 92:
					return 8
				case 47:
					return 9
				case 98:
					return 10
				case 117:
					return 11
				case 123:
					return -1
				case 52:
					return -1
				case 125:
					return -1
				case 105:
					return -1
				case 96:
					return -1
				case 102:
					return 12
				case 110:
					return 13
				case 114:
					return 14
				case 116:
					return 15
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 96:
					return 6
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 116:
					return 3
				case 92:
					return 2
				case 47:
					return 3
				case 98:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				case 125:
					return 3
				case 105:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 96:
					return 5
				case 102:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 116:
					return -1
				case 92:
					return -1
				case 47:
					return -1
				case 98:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return -1
				case 125:
					return -1
				case 105:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 96:
					return 6
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 116:
					return 3
				case 92:
					return 2
				case 47:
					return 3
				case 98:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				case 125:
					return 3
				case 105:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 96:
					return 5
				case 102:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 116:
					return -1
				case 92:
					return -1
				case 47:
					return -1
				case 98:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return -1
				case 125:
					return -1
				case 105:
					return 7
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 92:
					return -1
				case 47:
					return -1
				case 98:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return -1
				case 125:
					return -1
				case 105:
					return -1
				case 96:
					return -1
				case 102:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 116:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 92:
					return 2
				case 47:
					return 3
				case 98:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				case 125:
					return 3
				case 105:
					return 3
				case 96:
					return 6
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 116:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 92:
					return 2
				case 47:
					return 3
				case 98:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				case 125:
					return 3
				case 105:
					return 3
				case 96:
					return 6
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 116:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 96:
					return 6
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 116:
					return 3
				case 92:
					return 2
				case 47:
					return 3
				case 98:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				case 125:
					return 3
				case 105:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 96:
					return -1
				case 102:
					return 16
				case 110:
					return -1
				case 114:
					return -1
				case 116:
					return -1
				case 92:
					return -1
				case 47:
					return -1
				case 98:
					return 16
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return 16
				case 125:
					return -1
				case 105:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 16
				case 65 <= r && r <= 70:
					return 16
				case 97 <= r && r <= 102:
					return 16
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 96:
					return 6
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 116:
					return 3
				case 92:
					return 2
				case 47:
					return 3
				case 98:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				case 125:
					return 3
				case 105:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 92:
					return 2
				case 47:
					return 3
				case 98:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				case 125:
					return 3
				case 105:
					return 3
				case 96:
					return 6
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 116:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 96:
					return 6
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 116:
					return 3
				case 92:
					return 2
				case 47:
					return 3
				case 98:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				case 125:
					return 3
				case 105:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 92:
					return 2
				case 47:
					return 3
				case 98:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				case 125:
					return 3
				case 105:
					return 3
				case 96:
					return 6
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 116:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 96:
					return -1
				case 102:
					return 17
				case 110:
					return -1
				case 114:
					return -1
				case 116:
					return -1
				case 92:
					return -1
				case 47:
					return -1
				case 98:
					return 17
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return 17
				case 125:
					return -1
				case 105:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 17
				case 65 <= r && r <= 70:
					return 17
				case 97 <= r && r <= 102:
					return 17
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 96:
					return -1
				case 102:
					return 18
				case 110:
					return -1
				case 114:
					return -1
				case 116:
					return -1
				case 92:
					return -1
				case 47:
					return -1
				case 98:
					return 18
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return 18
				case 125:
					return -1
				case 105:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 18
				case 65 <= r && r <= 70:
					return 18
				case 97 <= r && r <= 102:
					return 18
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 92:
					return -1
				case 47:
					return -1
				case 98:
					return 19
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return 19
				case 125:
					return -1
				case 105:
					return -1
				case 96:
					return -1
				case 102:
					return 19
				case 110:
					return -1
				case 114:
					return -1
				case 116:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 19
				case 65 <= r && r <= 70:
					return 19
				case 97 <= r && r <= 102:
					return 19
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 96:
					return -1
				case 102:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 116:
					return -1
				case 92:
					return -1
				case 47:
					return -1
				case 98:
					return -1
				case 117:
					return -1
				case 123:
					return 20
				case 52:
					return -1
				case 125:
					return -1
				case 105:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 92:
					return -1
				case 47:
					return -1
				case 98:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return 21
				case 125:
					return -1
				case 105:
					return -1
				case 96:
					return -1
				case 102:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 116:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 92:
					return -1
				case 47:
					return -1
				case 98:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return -1
				case 125:
					return 22
				case 105:
					return -1
				case 96:
					return -1
				case 102:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 116:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 92:
					return 2
				case 47:
					return 3
				case 98:
					return 3
				case 117:
					return 3
				case 123:
					return 3
				case 52:
					return 3
				case 125:
					return 3
				case 105:
					return 3
				case 96:
					return 6
				case 102:
					return 3
				case 110:
					return 3
				case 114:
					return 3
				case 116:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return 3
				case 65 <= r && r <= 70:
					return 3
				case 97 <= r && r <= 102:
					return 3
				}
				return 3
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// `((\\\\)|(\\\/)|(\\b)|(\\f)|(\\n)|(\\r)|(\\t)|(\\u([0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F]){4})|(``)|[^\\`])+`
		{[]bool{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 96:
					return 1
				case 92:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return -1
				case 47:
					return -1
				case 98:
					return -1
				case 102:
					return -1
				case 114:
					return -1
				case 125:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 47:
					return 2
				case 98:
					return 2
				case 102:
					return 2
				case 114:
					return 2
				case 125:
					return 2
				case 96:
					return 3
				case 92:
					return 4
				case 110:
					return 2
				case 116:
					return 2
				case 117:
					return 2
				case 123:
					return 2
				case 52:
					return 2
				}
				switch {
				case 48 <= r && r <= 57:
					return 2
				case 65 <= r && r <= 70:
					return 2
				case 97 <= r && r <= 102:
					return 2
				}
				return 2
			},
			func(r rune) int {
				switch r {
				case 47:
					return 2
				case 98:
					return 2
				case 102:
					return 2
				case 114:
					return 2
				case 125:
					return 2
				case 96:
					return 20
				case 92:
					return 4
				case 110:
					return 2
				case 116:
					return 2
				case 117:
					return 2
				case 123:
					return 2
				case 52:
					return 2
				}
				switch {
				case 48 <= r && r <= 57:
					return 2
				case 65 <= r && r <= 70:
					return 2
				case 97 <= r && r <= 102:
					return 2
				}
				return 2
			},
			func(r rune) int {
				switch r {
				case 47:
					return -1
				case 98:
					return -1
				case 102:
					return -1
				case 114:
					return -1
				case 125:
					return -1
				case 96:
					return 21
				case 92:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 47:
					return 5
				case 98:
					return 6
				case 102:
					return 7
				case 114:
					return 8
				case 125:
					return -1
				case 96:
					return -1
				case 92:
					return 9
				case 110:
					return 10
				case 116:
					return 11
				case 117:
					return 12
				case 123:
					return -1
				case 52:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 47:
					return 2
				case 98:
					return 2
				case 102:
					return 2
				case 114:
					return 2
				case 125:
					return 2
				case 96:
					return 20
				case 92:
					return 4
				case 110:
					return 2
				case 116:
					return 2
				case 117:
					return 2
				case 123:
					return 2
				case 52:
					return 2
				}
				switch {
				case 48 <= r && r <= 57:
					return 2
				case 65 <= r && r <= 70:
					return 2
				case 97 <= r && r <= 102:
					return 2
				}
				return 2
			},
			func(r rune) int {
				switch r {
				case 47:
					return 2
				case 98:
					return 2
				case 102:
					return 2
				case 114:
					return 2
				case 125:
					return 2
				case 96:
					return 20
				case 92:
					return 4
				case 110:
					return 2
				case 116:
					return 2
				case 117:
					return 2
				case 123:
					return 2
				case 52:
					return 2
				}
				switch {
				case 48 <= r && r <= 57:
					return 2
				case 65 <= r && r <= 70:
					return 2
				case 97 <= r && r <= 102:
					return 2
				}
				return 2
			},
			func(r rune) int {
				switch r {
				case 47:
					return 2
				case 98:
					return 2
				case 102:
					return 2
				case 114:
					return 2
				case 125:
					return 2
				case 96:
					return 20
				case 92:
					return 4
				case 110:
					return 2
				case 116:
					return 2
				case 117:
					return 2
				case 123:
					return 2
				case 52:
					return 2
				}
				switch {
				case 48 <= r && r <= 57:
					return 2
				case 65 <= r && r <= 70:
					return 2
				case 97 <= r && r <= 102:
					return 2
				}
				return 2
			},
			func(r rune) int {
				switch r {
				case 47:
					return 2
				case 98:
					return 2
				case 102:
					return 2
				case 114:
					return 2
				case 125:
					return 2
				case 96:
					return 20
				case 92:
					return 4
				case 110:
					return 2
				case 116:
					return 2
				case 117:
					return 2
				case 123:
					return 2
				case 52:
					return 2
				}
				switch {
				case 48 <= r && r <= 57:
					return 2
				case 65 <= r && r <= 70:
					return 2
				case 97 <= r && r <= 102:
					return 2
				}
				return 2
			},
			func(r rune) int {
				switch r {
				case 47:
					return 2
				case 98:
					return 2
				case 102:
					return 2
				case 114:
					return 2
				case 125:
					return 2
				case 96:
					return 20
				case 92:
					return 4
				case 110:
					return 2
				case 116:
					return 2
				case 117:
					return 2
				case 123:
					return 2
				case 52:
					return 2
				}
				switch {
				case 48 <= r && r <= 57:
					return 2
				case 65 <= r && r <= 70:
					return 2
				case 97 <= r && r <= 102:
					return 2
				}
				return 2
			},
			func(r rune) int {
				switch r {
				case 96:
					return 20
				case 92:
					return 4
				case 110:
					return 2
				case 116:
					return 2
				case 117:
					return 2
				case 123:
					return 2
				case 52:
					return 2
				case 47:
					return 2
				case 98:
					return 2
				case 102:
					return 2
				case 114:
					return 2
				case 125:
					return 2
				}
				switch {
				case 48 <= r && r <= 57:
					return 2
				case 65 <= r && r <= 70:
					return 2
				case 97 <= r && r <= 102:
					return 2
				}
				return 2
			},
			func(r rune) int {
				switch r {
				case 96:
					return 20
				case 92:
					return 4
				case 110:
					return 2
				case 116:
					return 2
				case 117:
					return 2
				case 123:
					return 2
				case 52:
					return 2
				case 47:
					return 2
				case 98:
					return 2
				case 102:
					return 2
				case 114:
					return 2
				case 125:
					return 2
				}
				switch {
				case 48 <= r && r <= 57:
					return 2
				case 65 <= r && r <= 70:
					return 2
				case 97 <= r && r <= 102:
					return 2
				}
				return 2
			},
			func(r rune) int {
				switch r {
				case 47:
					return -1
				case 98:
					return 13
				case 102:
					return 13
				case 114:
					return -1
				case 125:
					return -1
				case 96:
					return -1
				case 92:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return 13
				}
				switch {
				case 48 <= r && r <= 57:
					return 13
				case 65 <= r && r <= 70:
					return 13
				case 97 <= r && r <= 102:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 47:
					return -1
				case 98:
					return 14
				case 102:
					return 14
				case 114:
					return -1
				case 125:
					return -1
				case 96:
					return -1
				case 92:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return 14
				}
				switch {
				case 48 <= r && r <= 57:
					return 14
				case 65 <= r && r <= 70:
					return 14
				case 97 <= r && r <= 102:
					return 14
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 96:
					return -1
				case 92:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return 15
				case 47:
					return -1
				case 98:
					return 15
				case 102:
					return 15
				case 114:
					return -1
				case 125:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 15
				case 65 <= r && r <= 70:
					return 15
				case 97 <= r && r <= 102:
					return 15
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 47:
					return -1
				case 98:
					return 16
				case 102:
					return 16
				case 114:
					return -1
				case 125:
					return -1
				case 96:
					return -1
				case 92:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return 16
				}
				switch {
				case 48 <= r && r <= 57:
					return 16
				case 65 <= r && r <= 70:
					return 16
				case 97 <= r && r <= 102:
					return 16
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 47:
					return -1
				case 98:
					return -1
				case 102:
					return -1
				case 114:
					return -1
				case 125:
					return -1
				case 96:
					return -1
				case 92:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 123:
					return 17
				case 52:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 96:
					return -1
				case 92:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return 18
				case 47:
					return -1
				case 98:
					return -1
				case 102:
					return -1
				case 114:
					return -1
				case 125:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 96:
					return -1
				case 92:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return -1
				case 47:
					return -1
				case 98:
					return -1
				case 102:
					return -1
				case 114:
					return -1
				case 125:
					return 19
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 96:
					return 20
				case 92:
					return 4
				case 110:
					return 2
				case 116:
					return 2
				case 117:
					return 2
				case 123:
					return 2
				case 52:
					return 2
				case 47:
					return 2
				case 98:
					return 2
				case 102:
					return 2
				case 114:
					return 2
				case 125:
					return 2
				}
				switch {
				case 48 <= r && r <= 57:
					return 2
				case 65 <= r && r <= 70:
					return 2
				case 97 <= r && r <= 102:
					return 2
				}
				return 2
			},
			func(r rune) int {
				switch r {
				case 47:
					return -1
				case 98:
					return -1
				case 102:
					return -1
				case 114:
					return -1
				case 125:
					return -1
				case 96:
					return 21
				case 92:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 123:
					return -1
				case 52:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 96:
					return 20
				case 92:
					return 4
				case 110:
					return 2
				case 116:
					return 2
				case 117:
					return 2
				case 123:
					return 2
				case 52:
					return 2
				case 47:
					return 2
				case 98:
					return 2
				case 102:
					return 2
				case 114:
					return 2
				case 125:
					return 2
				}
				switch {
				case 48 <= r && r <= 57:
					return 2
				case 65 <= r && r <= 70:
					return 2
				case 97 <= r && r <= 102:
					return 2
				}
				return 2
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// (0|[1-9][0-9]*)\.[0-9]+([eE][+\-]?[0-9]+)?
		{[]bool{false, false, false, false, false, true, false, true, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 48:
					return 1
				case 46:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 43:
					return -1
				case 45:
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
				case 48:
					return -1
				case 46:
					return 4
				case 101:
					return -1
				case 69:
					return -1
				case 43:
					return -1
				case 45:
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
				case 46:
					return 4
				case 101:
					return -1
				case 69:
					return -1
				case 43:
					return -1
				case 45:
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
				case 48:
					return 3
				case 46:
					return 4
				case 101:
					return -1
				case 69:
					return -1
				case 43:
					return -1
				case 45:
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
				case 48:
					return 5
				case 46:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 43:
					return -1
				case 45:
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
				case 48:
					return 5
				case 46:
					return -1
				case 101:
					return 6
				case 69:
					return 6
				case 43:
					return -1
				case 45:
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
				case 48:
					return 7
				case 46:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 43:
					return 8
				case 45:
					return 8
				}
				switch {
				case 48 <= r && r <= 48:
					return 7
				case 49 <= r && r <= 57:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 48:
					return 7
				case 46:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				}
				switch {
				case 48 <= r && r <= 48:
					return 7
				case 49 <= r && r <= 57:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 48:
					return 7
				case 46:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				}
				switch {
				case 48 <= r && r <= 48:
					return 7
				case 49 <= r && r <= 57:
					return 7
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// (0|[1-9][0-9]*)[eE][+\-]?[0-9]+
		{[]bool{false, false, false, false, false, true, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 48:
					return 1
				case 101:
					return -1
				case 69:
					return -1
				case 43:
					return -1
				case 45:
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
				case 48:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				case 43:
					return -1
				case 45:
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
				case 101:
					return 4
				case 69:
					return 4
				case 43:
					return -1
				case 45:
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
				case 48:
					return 3
				case 101:
					return 4
				case 69:
					return 4
				case 43:
					return -1
				case 45:
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
				case 48:
					return 5
				case 101:
					return -1
				case 69:
					return -1
				case 43:
					return 6
				case 45:
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
				case 48:
					return 5
				case 101:
					return -1
				case 69:
					return -1
				case 43:
					return -1
				case 45:
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
				case 48:
					return 5
				case 101:
					return -1
				case 69:
					return -1
				case 43:
					return -1
				case 45:
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

		// (\/\*)([^\*]|(\*)+[^\/])*((\*)+\/)
		{[]bool{false, false, false, false, false, true, false, false, true, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 47:
					return 1
				case 42:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 47:
					return -1
				case 42:
					return 2
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 47:
					return 3
				case 42:
					return 4
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 47:
					return 3
				case 42:
					return 4
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 47:
					return 5
				case 42:
					return 6
				}
				return 7
			},
			func(r rune) int {
				switch r {
				case 47:
					return -1
				case 42:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 47:
					return 8
				case 42:
					return 6
				}
				return 9
			},
			func(r rune) int {
				switch r {
				case 47:
					return 3
				case 42:
					return 4
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 47:
					return 3
				case 42:
					return 4
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 47:
					return 3
				case 42:
					return 4
				}
				return 3
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// "--"[^\n\r]*
		{[]bool{false, false, false, false, true, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 34:
					return 1
				case 45:
					return -1
				case 10:
					return -1
				case 13:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 34:
					return -1
				case 45:
					return 2
				case 10:
					return -1
				case 13:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 34:
					return -1
				case 45:
					return 3
				case 10:
					return -1
				case 13:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 34:
					return 4
				case 45:
					return -1
				case 10:
					return -1
				case 13:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 34:
					return 5
				case 45:
					return 5
				case 10:
					return -1
				case 13:
					return -1
				}
				return 5
			},
			func(r rune) int {
				switch r {
				case 34:
					return 5
				case 45:
					return 5
				case 10:
					return -1
				case 13:
					return -1
				}
				return 5
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [ \t\n\r\f]+
		{[]bool{false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 32:
					return 1
				case 9:
					return 1
				case 10:
					return 1
				case 13:
					return 1
				case 12:
					return 1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 32:
					return 1
				case 9:
					return 1
				case 10:
					return 1
				case 13:
					return 1
				case 12:
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
				case 62:
					return 1
				case 61:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 62:
					return -1
				case 61:
					return 2
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 62:
					return -1
				case 61:
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

		// [aA][lL][lL]
		{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 97:
					return 1
				case 65:
					return 1
				case 108:
					return -1
				case 76:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 108:
					return 2
				case 76:
					return 2
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 108:
					return 3
				case 76:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

		// [aA][lL][tT][eE][rR]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 97:
					return 1
				case 116:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 65:
					return 1
				case 108:
					return -1
				case 76:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 65:
					return -1
				case 108:
					return 2
				case 76:
					return 2
				case 84:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 116:
					return 3
				case 114:
					return -1
				case 82:
					return -1
				case 65:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 84:
					return 3
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 65:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 84:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 116:
					return -1
				case 114:
					return 5
				case 82:
					return 5
				case 65:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 65:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 69:
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
				case 110:
					return -1
				case 108:
					return -1
				case 122:
					return -1
				case 101:
					return -1
				case 97:
					return 1
				case 78:
					return -1
				case 76:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 90:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 78:
					return 2
				case 76:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 90:
					return -1
				case 69:
					return -1
				case 65:
					return -1
				case 110:
					return 2
				case 108:
					return -1
				case 122:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return 3
				case 78:
					return -1
				case 76:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 90:
					return -1
				case 69:
					return -1
				case 65:
					return 3
				case 110:
					return -1
				case 108:
					return -1
				case 122:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 65:
					return -1
				case 110:
					return -1
				case 108:
					return 4
				case 122:
					return -1
				case 101:
					return -1
				case 97:
					return -1
				case 78:
					return -1
				case 76:
					return 4
				case 121:
					return -1
				case 89:
					return -1
				case 90:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 78:
					return -1
				case 76:
					return -1
				case 121:
					return 5
				case 89:
					return 5
				case 90:
					return -1
				case 69:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 108:
					return -1
				case 122:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 65:
					return -1
				case 110:
					return -1
				case 108:
					return -1
				case 122:
					return 6
				case 101:
					return -1
				case 97:
					return -1
				case 78:
					return -1
				case 76:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 90:
					return 6
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 65:
					return -1
				case 110:
					return -1
				case 108:
					return -1
				case 122:
					return -1
				case 101:
					return 7
				case 97:
					return -1
				case 78:
					return -1
				case 76:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 90:
					return -1
				case 69:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 65:
					return -1
				case 110:
					return -1
				case 108:
					return -1
				case 122:
					return -1
				case 101:
					return -1
				case 97:
					return -1
				case 78:
					return -1
				case 76:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 90:
					return -1
				case 69:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [aA][nN][dD]
		{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 97:
					return 1
				case 65:
					return 1
				case 110:
					return -1
				case 78:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 110:
					return 2
				case 78:
					return 2
				case 100:
					return -1
				case 68:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 100:
					return 3
				case 68:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

		// [aA][nN][yY]
		{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 97:
					return 1
				case 65:
					return 1
				case 110:
					return -1
				case 78:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 110:
					return 2
				case 78:
					return 2
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 121:
					return 3
				case 89:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

		// [aA][rR][rR][aA][yY]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 97:
					return 1
				case 65:
					return 1
				case 114:
					return -1
				case 82:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return 2
				case 82:
					return 2
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return 3
				case 82:
					return 3
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return 4
				case 65:
					return 4
				case 114:
					return -1
				case 82:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 121:
					return 5
				case 89:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [aA][sS]
		{[]bool{false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 97:
					return 1
				case 65:
					return 1
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 115:
					return 2
				case 83:
					return 2
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

		// [aA][sS][cC]
		{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 97:
					return 1
				case 65:
					return 1
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 115:
					return 2
				case 83:
					return 2
				case 99:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return 3
				case 67:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

		// [bB][eE][gG][iI][nN]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 66:
					return 1
				case 69:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 98:
					return 1
				case 101:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 101:
					return 2
				case 110:
					return -1
				case 78:
					return -1
				case 66:
					return -1
				case 69:
					return 2
				case 103:
					return -1
				case 71:
					return -1
				case 105:
					return -1
				case 73:
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
				case 103:
					return 3
				case 71:
					return 3
				case 105:
					return -1
				case 73:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 110:
					return -1
				case 78:
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
				case 103:
					return -1
				case 71:
					return -1
				case 105:
					return 4
				case 73:
					return 4
				case 98:
					return -1
				case 101:
					return -1
				case 110:
					return -1
				case 78:
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
				case 103:
					return -1
				case 71:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 110:
					return 5
				case 78:
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
				case 103:
					return -1
				case 71:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [bB][eE][tT][wW][eE][eE][nN]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 69:
					return -1
				case 84:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 98:
					return 1
				case 66:
					return 1
				case 101:
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
				case 84:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 101:
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
				case 84:
					return 3
				case 119:
					return -1
				case 87:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 101:
					return -1
				case 116:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 66:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 119:
					return 4
				case 87:
					return 4
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 66:
					return -1
				case 101:
					return 5
				case 116:
					return -1
				case 69:
					return 5
				case 84:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 66:
					return -1
				case 101:
					return 6
				case 116:
					return -1
				case 69:
					return 6
				case 84:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 66:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 110:
					return 7
				case 78:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 69:
					return -1
				case 84:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [bB][iI][nN][aA][rR][yY]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 98:
					return 1
				case 66:
					return 1
				case 105:
					return -1
				case 73:
					return -1
				case 82:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 78:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 105:
					return 2
				case 73:
					return 2
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 66:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 82:
					return -1
				case 110:
					return 3
				case 78:
					return 3
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 66:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 82:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 97:
					return 4
				case 65:
					return 4
				case 114:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 78:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return 5
				case 121:
					return -1
				case 89:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 82:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 78:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return -1
				case 121:
					return 6
				case 89:
					return 6
				case 98:
					return -1
				case 66:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 78:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 82:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [bB][oO][oO][lL][eE][aA][nN]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 66:
					return 1
				case 69:
					return -1
				case 97:
					return -1
				case 78:
					return -1
				case 98:
					return 1
				case 111:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 65:
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
				case 97:
					return -1
				case 78:
					return -1
				case 98:
					return -1
				case 111:
					return 2
				case 79:
					return 2
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 65:
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
				case 97:
					return -1
				case 78:
					return -1
				case 98:
					return -1
				case 111:
					return 3
				case 79:
					return 3
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 108:
					return 4
				case 76:
					return 4
				case 101:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return 5
				case 65:
					return -1
				case 110:
					return -1
				case 66:
					return -1
				case 69:
					return 5
				case 97:
					return -1
				case 78:
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
				case 97:
					return 6
				case 78:
					return -1
				case 98:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 65:
					return 6
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
				case 97:
					return -1
				case 78:
					return 7
				case 98:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 65:
					return -1
				case 110:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 66:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 78:
					return -1
				case 98:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [bB][rR][eE][aA][kK]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 98:
					return 1
				case 66:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 75:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 107:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 66:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 75:
					return -1
				case 114:
					return 2
				case 82:
					return 2
				case 101:
					return -1
				case 69:
					return -1
				case 107:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return 3
				case 69:
					return 3
				case 107:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 75:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 107:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 97:
					return 4
				case 65:
					return 4
				case 75:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 107:
					return 5
				case 98:
					return -1
				case 66:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 75:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 107:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 75:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [bB][uU][cC][kK][eE][tT]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 98:
					return 1
				case 66:
					return 1
				case 117:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 85:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 66:
					return -1
				case 117:
					return 2
				case 99:
					return -1
				case 67:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 85:
					return 2
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 66:
					return -1
				case 117:
					return -1
				case 99:
					return 3
				case 67:
					return 3
				case 69:
					return -1
				case 116:
					return -1
				case 85:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 85:
					return -1
				case 107:
					return 4
				case 75:
					return 4
				case 101:
					return -1
				case 84:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 117:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 66:
					return -1
				case 117:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 69:
					return 5
				case 116:
					return -1
				case 85:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return 5
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 66:
					return -1
				case 117:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 69:
					return -1
				case 116:
					return 6
				case 85:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return -1
				case 84:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 66:
					return -1
				case 117:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 85:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [bB][uU][iI][lL][dD]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 98:
					return 1
				case 66:
					return 1
				case 117:
					return -1
				case 85:
					return -1
				case 105:
					return -1
				case 76:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 73:
					return -1
				case 108:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 66:
					return -1
				case 117:
					return 2
				case 85:
					return 2
				case 105:
					return -1
				case 76:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 73:
					return -1
				case 108:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 66:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 105:
					return 3
				case 76:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 73:
					return 3
				case 108:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 66:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 105:
					return -1
				case 76:
					return 4
				case 100:
					return -1
				case 68:
					return -1
				case 73:
					return -1
				case 108:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 73:
					return -1
				case 108:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 105:
					return -1
				case 76:
					return -1
				case 100:
					return 5
				case 68:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 66:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 105:
					return -1
				case 76:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 73:
					return -1
				case 108:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [bB][yY]
		{[]bool{false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 98:
					return 1
				case 66:
					return 1
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 66:
					return -1
				case 121:
					return 2
				case 89:
					return 2
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 66:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

		// [cC][aA][lL][lL]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 99:
					return 1
				case 67:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 97:
					return 2
				case 65:
					return 2
				case 108:
					return -1
				case 76:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 108:
					return 3
				case 76:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 108:
					return 4
				case 76:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [cC][aA][sS][eE]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 99:
					return 1
				case 67:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 97:
					return 2
				case 65:
					return 2
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 115:
					return 3
				case 83:
					return 3
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [cC][aA][sS][tT]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 99:
					return 1
				case 67:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 97:
					return 2
				case 65:
					return 2
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 115:
					return 3
				case 83:
					return 3
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return 4
				case 84:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [cC][lL][uU][sS][tT][eE][rR]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 99:
					return 1
				case 67:
					return 1
				case 115:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return 2
				case 76:
					return 2
				case 85:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 115:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 83:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 115:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 85:
					return 3
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return 3
				case 116:
					return -1
				case 114:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 83:
					return 4
				case 99:
					return -1
				case 67:
					return -1
				case 115:
					return 4
				case 84:
					return -1
				case 82:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 83:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 115:
					return -1
				case 84:
					return 5
				case 82:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 116:
					return 5
				case 114:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 115:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 85:
					return -1
				case 101:
					return 6
				case 69:
					return 6
				case 117:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 116:
					return -1
				case 114:
					return 7
				case 83:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 115:
					return -1
				case 84:
					return -1
				case 82:
					return 7
				case 108:
					return -1
				case 76:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 115:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [cC][oO][lL][lL][aA][tT][eE]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 67:
					return 1
				case 108:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 99:
					return 1
				case 111:
					return -1
				case 79:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 111:
					return 2
				case 79:
					return 2
				case 76:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 67:
					return -1
				case 108:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 67:
					return -1
				case 108:
					return 3
				case 116:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 76:
					return 3
				case 97:
					return -1
				case 65:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 67:
					return -1
				case 108:
					return 4
				case 116:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 76:
					return 4
				case 97:
					return -1
				case 65:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 76:
					return -1
				case 97:
					return 5
				case 65:
					return 5
				case 101:
					return -1
				case 69:
					return -1
				case 67:
					return -1
				case 108:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 67:
					return -1
				case 108:
					return -1
				case 116:
					return 6
				case 84:
					return 6
				case 99:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 67:
					return -1
				case 108:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 101:
					return 7
				case 69:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 67:
					return -1
				case 108:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [cC][oO][lL][lL][eE][cC][tT][iI][oO][nN]
		{[]bool{false, false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 105:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 73:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 99:
					return 1
				case 67:
					return 1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
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
				case 84:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 79:
					return 2
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 110:
					return -1
				case 111:
					return 2
				case 105:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 73:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 69:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 79:
					return -1
				case 108:
					return 3
				case 76:
					return 3
				case 116:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 105:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 73:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 69:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 79:
					return -1
				case 108:
					return 4
				case 76:
					return 4
				case 116:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 105:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 73:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 105:
					return -1
				case 78:
					return -1
				case 101:
					return 5
				case 73:
					return -1
				case 69:
					return 5
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 105:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 73:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 99:
					return 6
				case 67:
					return 6
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
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
				case 84:
					return 7
				case 99:
					return -1
				case 67:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return 7
				case 110:
					return -1
				case 111:
					return -1
				case 105:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 73:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 105:
					return 8
				case 78:
					return -1
				case 101:
					return -1
				case 73:
					return 8
				case 69:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 73:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 79:
					return 9
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 110:
					return -1
				case 111:
					return 9
				case 105:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 73:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 110:
					return 10
				case 111:
					return -1
				case 105:
					return -1
				case 78:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 105:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 73:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 110:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [cC][oO][mM][mM][iI][tT]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 79:
					return -1
				case 109:
					return -1
				case 116:
					return -1
				case 99:
					return 1
				case 67:
					return 1
				case 111:
					return -1
				case 77:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 111:
					return 2
				case 77:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 84:
					return -1
				case 79:
					return 2
				case 109:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 77:
					return 3
				case 105:
					return -1
				case 73:
					return -1
				case 84:
					return -1
				case 79:
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
				case 99:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 77:
					return 4
				case 105:
					return -1
				case 73:
					return -1
				case 84:
					return -1
				case 79:
					return -1
				case 109:
					return 4
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 79:
					return -1
				case 109:
					return -1
				case 116:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 77:
					return -1
				case 105:
					return 5
				case 73:
					return 5
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 77:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 84:
					return 6
				case 79:
					return -1
				case 109:
					return -1
				case 116:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 77:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 84:
					return -1
				case 79:
					return -1
				case 109:
					return -1
				case 116:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [cC][oO][nN][nN][eE][cC][tT]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 79:
					return -1
				case 78:
					return -1
				case 84:
					return -1
				case 99:
					return 1
				case 67:
					return 1
				case 111:
					return -1
				case 110:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 79:
					return 2
				case 78:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 111:
					return 2
				case 110:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 79:
					return -1
				case 78:
					return 3
				case 84:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 110:
					return 3
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 110:
					return 4
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 79:
					return -1
				case 78:
					return 4
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 110:
					return -1
				case 101:
					return 5
				case 69:
					return 5
				case 116:
					return -1
				case 79:
					return -1
				case 78:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return 6
				case 67:
					return 6
				case 111:
					return -1
				case 110:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 79:
					return -1
				case 78:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 110:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return 7
				case 79:
					return -1
				case 78:
					return -1
				case 84:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 79:
					return -1
				case 78:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 110:
					return -1
				case 101:
					return -1
				case 69:
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
				case 111:
					return -1
				case 78:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 99:
					return 1
				case 79:
					return -1
				case 110:
					return -1
				case 73:
					return -1
				case 67:
					return 1
				case 69:
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
				case 99:
					return -1
				case 79:
					return 2
				case 110:
					return -1
				case 73:
					return -1
				case 67:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 111:
					return 2
				case 78:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 85:
					return -1
				case 101:
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
				case 116:
					return -1
				case 117:
					return -1
				case 111:
					return -1
				case 78:
					return 3
				case 84:
					return -1
				case 105:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 99:
					return -1
				case 79:
					return -1
				case 110:
					return 3
				case 73:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 78:
					return -1
				case 84:
					return 4
				case 105:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 99:
					return -1
				case 79:
					return -1
				case 110:
					return -1
				case 73:
					return -1
				case 67:
					return -1
				case 69:
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
				case 111:
					return -1
				case 78:
					return -1
				case 84:
					return -1
				case 105:
					return 5
				case 85:
					return -1
				case 101:
					return -1
				case 99:
					return -1
				case 79:
					return -1
				case 110:
					return -1
				case 73:
					return 5
				case 67:
					return -1
				case 69:
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
				case 99:
					return -1
				case 79:
					return -1
				case 110:
					return 6
				case 73:
					return -1
				case 67:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 111:
					return -1
				case 78:
					return 6
				case 84:
					return -1
				case 105:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 117:
					return 7
				case 111:
					return -1
				case 78:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 85:
					return 7
				case 101:
					return -1
				case 99:
					return -1
				case 79:
					return -1
				case 110:
					return -1
				case 73:
					return -1
				case 67:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 117:
					return -1
				case 111:
					return -1
				case 78:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 85:
					return -1
				case 101:
					return 8
				case 99:
					return -1
				case 79:
					return -1
				case 110:
					return -1
				case 73:
					return -1
				case 67:
					return -1
				case 69:
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
				case 116:
					return -1
				case 117:
					return -1
				case 111:
					return -1
				case 78:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 99:
					return -1
				case 79:
					return -1
				case 110:
					return -1
				case 73:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [cC][rR][eE][aA][tT][eE]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 67:
					return 1
				case 69:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				case 99:
					return 1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 97:
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
				case 65:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 114:
					return 2
				case 82:
					return 2
				case 101:
					return -1
				case 97:
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
					return 3
				case 65:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return 3
				case 97:
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
				case 65:
					return 4
				case 84:
					return -1
				case 99:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 97:
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
					return -1
				case 65:
					return -1
				case 84:
					return 5
				case 99:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 97:
					return -1
				case 116:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 67:
					return -1
				case 69:
					return 6
				case 65:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return 6
				case 97:
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
				case 65:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [dD][aA][tT][aA][bB][aA][sS][eE]
		{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 100:
					return 1
				case 65:
					return -1
				case 84:
					return -1
				case 83:
					return -1
				case 68:
					return 1
				case 97:
					return -1
				case 116:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 115:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 65:
					return 2
				case 84:
					return -1
				case 83:
					return -1
				case 68:
					return -1
				case 97:
					return 2
				case 116:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 115:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 65:
					return -1
				case 84:
					return 3
				case 83:
					return -1
				case 68:
					return -1
				case 97:
					return -1
				case 116:
					return 3
				case 98:
					return -1
				case 66:
					return -1
				case 115:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 65:
					return 4
				case 84:
					return -1
				case 83:
					return -1
				case 68:
					return -1
				case 97:
					return 4
				case 116:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 115:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				case 83:
					return -1
				case 68:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 98:
					return 5
				case 66:
					return 5
				case 115:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 68:
					return -1
				case 97:
					return 6
				case 116:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 115:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 100:
					return -1
				case 65:
					return 6
				case 84:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 68:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 115:
					return 7
				case 101:
					return -1
				case 69:
					return -1
				case 100:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				case 83:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				case 83:
					return -1
				case 68:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 115:
					return -1
				case 101:
					return 8
				case 69:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 68:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 115:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 100:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				case 83:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [dD][aA][tT][aA][sS][eE][tT]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 68:
					return 1
				case 116:
					return -1
				case 84:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 100:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 68:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 100:
					return -1
				case 97:
					return 2
				case 65:
					return 2
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 116:
					return 3
				case 84:
					return 3
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 97:
					return 4
				case 65:
					return 4
				case 69:
					return -1
				case 68:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 68:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 115:
					return 5
				case 83:
					return 5
				case 101:
					return -1
				case 100:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 68:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return 6
				case 100:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 69:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 116:
					return 7
				case 84:
					return 7
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [dD][aA][tT][aA][sS][tT][oO][rR][eE]
		{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 68:
					return 1
				case 65:
					return -1
				case 83:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 79:
					return -1
				case 100:
					return 1
				case 115:
					return -1
				case 111:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 115:
					return -1
				case 111:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 65:
					return 2
				case 83:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 97:
					return 2
				case 116:
					return -1
				case 84:
					return -1
				case 79:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 101:
					return -1
				case 97:
					return -1
				case 116:
					return 3
				case 84:
					return 3
				case 79:
					return -1
				case 100:
					return -1
				case 115:
					return -1
				case 111:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 65:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return 4
				case 116:
					return -1
				case 84:
					return -1
				case 79:
					return -1
				case 100:
					return -1
				case 115:
					return -1
				case 111:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 65:
					return 4
				case 83:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 115:
					return 5
				case 111:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 65:
					return -1
				case 83:
					return 5
				case 114:
					return -1
				case 101:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 79:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 101:
					return -1
				case 97:
					return -1
				case 116:
					return 6
				case 84:
					return 6
				case 79:
					return -1
				case 100:
					return -1
				case 115:
					return -1
				case 111:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 65:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 101:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 79:
					return 7
				case 100:
					return -1
				case 115:
					return -1
				case 111:
					return 7
				case 82:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 65:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return 8
				case 101:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 79:
					return -1
				case 100:
					return -1
				case 115:
					return -1
				case 111:
					return -1
				case 82:
					return 8
				case 69:
					return -1
				case 68:
					return -1
				case 65:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 79:
					return -1
				case 100:
					return -1
				case 115:
					return -1
				case 111:
					return -1
				case 82:
					return -1
				case 69:
					return 9
				case 68:
					return -1
				case 65:
					return -1
				case 83:
					return -1
				case 114:
					return -1
				case 101:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 115:
					return -1
				case 111:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 65:
					return -1
				case 83:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 79:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [dD][eE][cC][lL][aA][rR][eE]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				case 100:
					return 1
				case 68:
					return 1
				case 69:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return 2
				case 65:
					return -1
				case 82:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return 2
				case 99:
					return -1
				case 67:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 99:
					return 3
				case 67:
					return 3
				case 108:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 108:
					return 4
				case 76:
					return 4
				case 97:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 65:
					return 5
				case 82:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 97:
					return 5
				case 114:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 114:
					return 6
				case 101:
					return -1
				case 65:
					return -1
				case 82:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return 7
				case 65:
					return -1
				case 82:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return 7
				case 99:
					return -1
				case 67:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 97:
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
				case 77:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				case 100:
					return 1
				case 69:
					return -1
				case 68:
					return 1
				case 101:
					return -1
				case 99:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 69:
					return 2
				case 68:
					return -1
				case 101:
					return 2
				case 99:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 84:
					return -1
				case 67:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 99:
					return 3
				case 109:
					return -1
				case 110:
					return -1
				case 84:
					return -1
				case 67:
					return 3
				case 77:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 99:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 84:
					return -1
				case 67:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 114:
					return 4
				case 82:
					return 4
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 69:
					return 5
				case 68:
					return -1
				case 101:
					return 5
				case 99:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 84:
					return -1
				case 67:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 68:
					return -1
				case 101:
					return -1
				case 99:
					return -1
				case 109:
					return 6
				case 110:
					return -1
				case 84:
					return -1
				case 67:
					return -1
				case 77:
					return 6
				case 116:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				case 100:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				case 100:
					return -1
				case 69:
					return 7
				case 68:
					return -1
				case 101:
					return 7
				case 99:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 84:
					return -1
				case 67:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 78:
					return 8
				case 100:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 99:
					return -1
				case 109:
					return -1
				case 110:
					return 8
				case 84:
					return -1
				case 67:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				case 100:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 99:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 84:
					return 9
				case 67:
					return -1
				case 77:
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
				case 77:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				case 100:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 99:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [dD][eE][lL][eE][tT][eE]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 100:
					return 1
				case 68:
					return 1
				case 101:
					return -1
				case 69:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return 2
				case 69:
					return 2
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 108:
					return 3
				case 76:
					return 3
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return 5
				case 84:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return 6
				case 69:
					return 6
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [dD][eE][rR][iI][vV][eE][dD]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 82:
					return -1
				case 73:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				case 100:
					return 1
				case 68:
					return 1
				case 69:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return 2
				case 82:
					return -1
				case 73:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return 2
				case 114:
					return -1
				case 105:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 114:
					return 3
				case 105:
					return -1
				case 101:
					return -1
				case 82:
					return 3
				case 73:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 82:
					return -1
				case 73:
					return 4
				case 118:
					return -1
				case 86:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 105:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 82:
					return -1
				case 73:
					return -1
				case 118:
					return 5
				case 86:
					return 5
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return 6
				case 114:
					return -1
				case 105:
					return -1
				case 101:
					return 6
				case 82:
					return -1
				case 73:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 82:
					return -1
				case 73:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				case 100:
					return 7
				case 68:
					return 7
				case 69:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 82:
					return -1
				case 73:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [dD][eE][sS][cC]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 100:
					return 1
				case 68:
					return 1
				case 101:
					return -1
				case 69:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return 2
				case 69:
					return 2
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 115:
					return 3
				case 83:
					return 3
				case 99:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return 4
				case 67:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [dD][eE][sS][cC][rR][iI][bB][eE]
		{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 68:
					return 1
				case 69:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				case 98:
					return -1
				case 100:
					return 1
				case 101:
					return -1
				case 115:
					return -1
				case 67:
					return -1
				case 99:
					return -1
				case 82:
					return -1
				case 66:
					return -1
				case 83:
					return -1
				case 73:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 101:
					return 2
				case 115:
					return -1
				case 67:
					return -1
				case 99:
					return -1
				case 82:
					return -1
				case 66:
					return -1
				case 83:
					return -1
				case 73:
					return -1
				case 68:
					return -1
				case 69:
					return 2
				case 114:
					return -1
				case 105:
					return -1
				case 98:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 82:
					return -1
				case 66:
					return -1
				case 83:
					return 3
				case 73:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				case 98:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 115:
					return 3
				case 67:
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
				case 114:
					return -1
				case 105:
					return -1
				case 98:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 115:
					return -1
				case 67:
					return 4
				case 99:
					return 4
				case 82:
					return -1
				case 66:
					return -1
				case 83:
					return -1
				case 73:
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
				case 114:
					return 5
				case 105:
					return -1
				case 98:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 115:
					return -1
				case 67:
					return -1
				case 99:
					return -1
				case 82:
					return 5
				case 66:
					return -1
				case 83:
					return -1
				case 73:
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
				case 114:
					return -1
				case 105:
					return 6
				case 98:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 115:
					return -1
				case 67:
					return -1
				case 99:
					return -1
				case 82:
					return -1
				case 66:
					return -1
				case 83:
					return -1
				case 73:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 82:
					return -1
				case 66:
					return 7
				case 83:
					return -1
				case 73:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				case 98:
					return 7
				case 100:
					return -1
				case 101:
					return -1
				case 115:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 68:
					return -1
				case 69:
					return 8
				case 114:
					return -1
				case 105:
					return -1
				case 98:
					return -1
				case 100:
					return -1
				case 101:
					return 8
				case 115:
					return -1
				case 67:
					return -1
				case 99:
					return -1
				case 82:
					return -1
				case 66:
					return -1
				case 83:
					return -1
				case 73:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 101:
					return -1
				case 115:
					return -1
				case 67:
					return -1
				case 99:
					return -1
				case 82:
					return -1
				case 66:
					return -1
				case 83:
					return -1
				case 73:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				case 98:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [dD][iI][sS][tT][iI][nN][cC][tT]
		{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 100:
					return 1
				case 68:
					return 1
				case 73:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 105:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 73:
					return 2
				case 115:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 105:
					return 2
				case 116:
					return -1
				case 84:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 78:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 73:
					return -1
				case 115:
					return 3
				case 83:
					return 3
				case 110:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 116:
					return 4
				case 84:
					return 4
				case 78:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 73:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return 5
				case 116:
					return -1
				case 84:
					return -1
				case 78:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 73:
					return 5
				case 115:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 73:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 110:
					return 6
				case 99:
					return -1
				case 67:
					return -1
				case 105:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 78:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 78:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 73:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				case 99:
					return 7
				case 67:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 116:
					return 8
				case 84:
					return 8
				case 78:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 73:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 78:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 73:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [dD][oO]
		{[]bool{false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 100:
					return 1
				case 68:
					return 1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 111:
					return 2
				case 79:
					return 2
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

		// [dD][rR][oO][pP]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 100:
					return 1
				case 68:
					return 1
				case 114:
					return -1
				case 82:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 114:
					return 2
				case 82:
					return 2
				case 111:
					return -1
				case 79:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 111:
					return 3
				case 79:
					return 3
				case 112:
					return -1
				case 80:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 112:
					return 4
				case 80:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 68:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [eE][aA][cC][hH]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 101:
					return 1
				case 69:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return 2
				case 65:
					return 2
				case 99:
					return -1
				case 67:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 99:
					return 3
				case 67:
					return 3
				case 104:
					return -1
				case 72:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 104:
					return 4
				case 72:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [eE][lL][eE][mM][eE][nN][tT]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 101:
					return 1
				case 69:
					return 1
				case 76:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return 2
				case 109:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return 2
				case 110:
					return -1
				case 78:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 101:
					return 3
				case 69:
					return 3
				case 76:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 109:
					return 4
				case 77:
					return 4
				case 116:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return 5
				case 69:
					return 5
				case 76:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 84:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				case 110:
					return 6
				case 78:
					return 6
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 116:
					return 7
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 84:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [eE][lL][sS][eE]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 101:
					return 1
				case 69:
					return 1
				case 108:
					return -1
				case 76:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 108:
					return 2
				case 76:
					return 2
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 115:
					return 3
				case 83:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return 4
				case 69:
					return 4
				case 108:
					return -1
				case 76:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [eE][nN][dD]
		{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 101:
					return 1
				case 69:
					return 1
				case 110:
					return -1
				case 78:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return 2
				case 78:
					return 2
				case 100:
					return -1
				case 68:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 100:
					return 3
				case 68:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

		// [eE][vV][eE][rR][yY]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 101:
					return 1
				case 69:
					return 1
				case 118:
					return -1
				case 86:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 118:
					return 2
				case 86:
					return 2
				case 114:
					return -1
				case 82:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return 3
				case 69:
					return 3
				case 118:
					return -1
				case 86:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				case 114:
					return 4
				case 82:
					return 4
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 121:
					return 5
				case 89:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [eE][xX][cC][eE][pP][tT]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 116:
					return -1
				case 101:
					return 1
				case 69:
					return 1
				case 120:
					return -1
				case 88:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 116:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 120:
					return 2
				case 88:
					return 2
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return 3
				case 67:
					return 3
				case 112:
					return -1
				case 80:
					return -1
				case 116:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return 4
				case 69:
					return 4
				case 120:
					return -1
				case 88:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 112:
					return 5
				case 80:
					return 5
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				case 84:
					return 6
				case 99:
					return -1
				case 67:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 116:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 116:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [eE][xX][cC][lL][uU][dD][eE]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 101:
					return 1
				case 69:
					return 1
				case 120:
					return -1
				case 88:
					return -1
				case 99:
					return -1
				case 85:
					return -1
				case 67:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 120:
					return 2
				case 88:
					return 2
				case 99:
					return -1
				case 85:
					return -1
				case 67:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 67:
					return 3
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				case 99:
					return 3
				case 85:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				case 99:
					return -1
				case 85:
					return -1
				case 67:
					return -1
				case 108:
					return 4
				case 76:
					return 4
				case 117:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 67:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return 5
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				case 99:
					return -1
				case 85:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 67:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 100:
					return 6
				case 68:
					return 6
				case 101:
					return -1
				case 69:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				case 99:
					return -1
				case 85:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 67:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return 7
				case 69:
					return 7
				case 120:
					return -1
				case 88:
					return -1
				case 99:
					return -1
				case 85:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 67:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				case 99:
					return -1
				case 85:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [eE][xX][eE][cC][uU][tT][eE]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 101:
					return 1
				case 69:
					return 1
				case 88:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 117:
					return -1
				case 116:
					return -1
				case 120:
					return -1
				case 85:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 88:
					return 2
				case 99:
					return -1
				case 67:
					return -1
				case 117:
					return -1
				case 116:
					return -1
				case 120:
					return 2
				case 85:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 120:
					return -1
				case 85:
					return -1
				case 84:
					return -1
				case 101:
					return 3
				case 69:
					return 3
				case 88:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 117:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 120:
					return -1
				case 85:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 88:
					return -1
				case 99:
					return 4
				case 67:
					return 4
				case 117:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 120:
					return -1
				case 85:
					return 5
				case 84:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 88:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 117:
					return 5
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 120:
					return -1
				case 85:
					return -1
				case 84:
					return 6
				case 101:
					return -1
				case 69:
					return -1
				case 88:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 117:
					return -1
				case 116:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return 7
				case 69:
					return 7
				case 88:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 117:
					return -1
				case 116:
					return -1
				case 120:
					return -1
				case 85:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 120:
					return -1
				case 85:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 88:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 117:
					return -1
				case 116:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [eE][xX][iI][sS][tT][sS]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 88:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return 1
				case 69:
					return 1
				case 120:
					return -1
				case 105:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 88:
					return 2
				case 73:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 120:
					return 2
				case 105:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 88:
					return -1
				case 73:
					return 3
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 120:
					return -1
				case 105:
					return 3
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 120:
					return -1
				case 105:
					return -1
				case 115:
					return 4
				case 83:
					return 4
				case 88:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 120:
					return -1
				case 105:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 88:
					return -1
				case 73:
					return -1
				case 116:
					return 5
				case 84:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 88:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 120:
					return -1
				case 105:
					return -1
				case 115:
					return 6
				case 83:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 88:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 120:
					return -1
				case 105:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [eE][xX][pP][lL][aA][iI][nN]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 120:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 97:
					return -1
				case 101:
					return 1
				case 69:
					return 1
				case 88:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 120:
					return 2
				case 112:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 97:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 88:
					return 2
				case 105:
					return -1
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 88:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 120:
					return -1
				case 112:
					return 3
				case 80:
					return 3
				case 65:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 88:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 108:
					return 4
				case 76:
					return 4
				case 120:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 120:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 65:
					return 5
				case 73:
					return -1
				case 78:
					return -1
				case 97:
					return 5
				case 101:
					return -1
				case 69:
					return -1
				case 88:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 120:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				case 73:
					return 6
				case 78:
					return -1
				case 97:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 88:
					return -1
				case 105:
					return 6
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 120:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				case 73:
					return -1
				case 78:
					return 7
				case 97:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 88:
					return -1
				case 105:
					return -1
				case 110:
					return 7
				case 108:
					return -1
				case 76:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 88:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 120:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 97:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [fF][aA][lL][sS][eE]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 102:
					return 1
				case 65:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 70:
					return 1
				case 97:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 70:
					return -1
				case 97:
					return 2
				case 108:
					return -1
				case 76:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 102:
					return -1
				case 65:
					return 2
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 65:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 97:
					return -1
				case 108:
					return 3
				case 76:
					return 3
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 70:
					return -1
				case 97:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 115:
					return 4
				case 83:
					return 4
				case 102:
					return -1
				case 65:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 70:
					return -1
				case 97:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 102:
					return -1
				case 65:
					return -1
				case 101:
					return 5
				case 69:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 65:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 97:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [fF][iI][rR][sS][tT]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 102:
					return 1
				case 114:
					return -1
				case 82:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 70:
					return 1
				case 105:
					return -1
				case 73:
					return -1
				case 83:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 70:
					return -1
				case 105:
					return 2
				case 73:
					return 2
				case 83:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 114:
					return 3
				case 82:
					return 3
				case 115:
					return -1
				case 116:
					return -1
				case 70:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 83:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 115:
					return 4
				case 116:
					return -1
				case 70:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 83:
					return 4
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 70:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 83:
					return -1
				case 84:
					return 5
				case 102:
					return -1
				case 114:
					return -1
				case 82:
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
				case 105:
					return -1
				case 73:
					return -1
				case 83:
					return -1
				case 84:
					return -1
				case 102:
					return -1
				case 114:
					return -1
				case 82:
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
				case 102:
					return 1
				case 108:
					return -1
				case 76:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 110:
					return -1
				case 70:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 116:
					return -1
				case 69:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 70:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 116:
					return -1
				case 69:
					return -1
				case 78:
					return -1
				case 102:
					return -1
				case 108:
					return 2
				case 76:
					return 2
				case 84:
					return -1
				case 101:
					return -1
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 70:
					return -1
				case 97:
					return 3
				case 65:
					return 3
				case 116:
					return -1
				case 69:
					return -1
				case 78:
					return -1
				case 102:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 70:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 116:
					return 4
				case 69:
					return -1
				case 78:
					return -1
				case 102:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 84:
					return 4
				case 101:
					return -1
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 70:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 116:
					return 5
				case 69:
					return -1
				case 78:
					return -1
				case 102:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 84:
					return 5
				case 101:
					return -1
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 70:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 116:
					return -1
				case 69:
					return 6
				case 78:
					return -1
				case 102:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 84:
					return -1
				case 101:
					return 6
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 110:
					return 7
				case 70:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 116:
					return -1
				case 69:
					return -1
				case 78:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 110:
					return -1
				case 70:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 116:
					return -1
				case 69:
					return -1
				case 78:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [fF][oO][rR]
		{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 102:
					return 1
				case 70:
					return 1
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 70:
					return -1
				case 111:
					return 2
				case 79:
					return 2
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 70:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return 3
				case 82:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 70:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
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
				case 102:
					return 1
				case 70:
					return 1
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 99:
					return -1
				case 101:
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
				case 102:
					return -1
				case 70:
					return -1
				case 111:
					return 2
				case 79:
					return 2
				case 114:
					return -1
				case 82:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 70:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return 3
				case 82:
					return 3
				case 99:
					return -1
				case 101:
					return -1
				case 67:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 70:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 99:
					return 4
				case 101:
					return -1
				case 67:
					return 4
				case 69:
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
				case 102:
					return -1
				case 70:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 99:
					return -1
				case 101:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 70:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 67:
					return -1
				case 69:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [fF][rR][oO][mM]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 102:
					return 1
				case 70:
					return 1
				case 114:
					return -1
				case 82:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 70:
					return -1
				case 114:
					return 2
				case 82:
					return 2
				case 111:
					return -1
				case 79:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 70:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 111:
					return 3
				case 79:
					return 3
				case 109:
					return -1
				case 77:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 70:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 109:
					return 4
				case 77:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 70:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [fF][uU][nN][cC][tT][iI][oO][nN]
		{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 102:
					return 1
				case 117:
					return -1
				case 70:
					return 1
				case 85:
					return -1
				case 110:
					return -1
				case 99:
					return -1
				case 116:
					return -1
				case 78:
					return -1
				case 67:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 70:
					return -1
				case 85:
					return 2
				case 110:
					return -1
				case 99:
					return -1
				case 116:
					return -1
				case 78:
					return -1
				case 67:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 102:
					return -1
				case 117:
					return 2
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 117:
					return -1
				case 70:
					return -1
				case 85:
					return -1
				case 110:
					return 3
				case 99:
					return -1
				case 116:
					return -1
				case 78:
					return 3
				case 67:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 70:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 99:
					return 4
				case 116:
					return -1
				case 78:
					return -1
				case 67:
					return 4
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 102:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 102:
					return -1
				case 117:
					return -1
				case 70:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 99:
					return -1
				case 116:
					return 5
				case 78:
					return -1
				case 67:
					return -1
				case 84:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return 6
				case 73:
					return 6
				case 111:
					return -1
				case 79:
					return -1
				case 102:
					return -1
				case 117:
					return -1
				case 70:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 99:
					return -1
				case 116:
					return -1
				case 78:
					return -1
				case 67:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 111:
					return 7
				case 79:
					return 7
				case 102:
					return -1
				case 117:
					return -1
				case 70:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 99:
					return -1
				case 116:
					return -1
				case 78:
					return -1
				case 67:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 117:
					return -1
				case 70:
					return -1
				case 85:
					return -1
				case 110:
					return 8
				case 99:
					return -1
				case 116:
					return -1
				case 78:
					return 8
				case 67:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 78:
					return -1
				case 67:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 102:
					return -1
				case 117:
					return -1
				case 70:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 99:
					return -1
				case 116:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [gG][rR][aA][nN][tT]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 71:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 103:
					return 1
				case 114:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 103:
					return -1
				case 114:
					return 2
				case 82:
					return 2
				case 78:
					return -1
				case 84:
					return -1
				case 71:
					return -1
				case 97:
					return -1
				case 65:
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
				case 103:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				case 84:
					return -1
				case 71:
					return -1
				case 97:
					return 3
				case 65:
					return 3
				case 110:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 103:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 78:
					return 4
				case 84:
					return -1
				case 71:
					return -1
				case 97:
					return -1
				case 65:
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
				case 103:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				case 84:
					return 5
				case 71:
					return -1
				case 97:
					return -1
				case 65:
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
				case 71:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 103:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [gG][rR][oO][uU][pP]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 103:
					return 1
				case 82:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 71:
					return 1
				case 114:
					return -1
				case 79:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 80:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 71:
					return -1
				case 114:
					return 2
				case 79:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 80:
					return -1
				case 103:
					return -1
				case 82:
					return 2
				case 111:
					return -1
				case 112:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 71:
					return -1
				case 114:
					return -1
				case 79:
					return 3
				case 117:
					return -1
				case 85:
					return -1
				case 80:
					return -1
				case 103:
					return -1
				case 82:
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
				case 103:
					return -1
				case 82:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 71:
					return -1
				case 114:
					return -1
				case 79:
					return -1
				case 117:
					return 4
				case 85:
					return 4
				case 80:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 103:
					return -1
				case 82:
					return -1
				case 111:
					return -1
				case 112:
					return 5
				case 71:
					return -1
				case 114:
					return -1
				case 79:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 80:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 103:
					return -1
				case 82:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 71:
					return -1
				case 114:
					return -1
				case 79:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 80:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [gG][sS][iI]
		{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 103:
					return 1
				case 71:
					return 1
				case 115:
					return -1
				case 83:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 103:
					return -1
				case 71:
					return -1
				case 115:
					return 2
				case 83:
					return 2
				case 105:
					return -1
				case 73:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 103:
					return -1
				case 71:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 105:
					return 3
				case 73:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 103:
					return -1
				case 71:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

		// [hH][aA][vV][iI][nN][gG]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 104:
					return 1
				case 97:
					return -1
				case 86:
					return -1
				case 105:
					return -1
				case 78:
					return -1
				case 71:
					return -1
				case 72:
					return 1
				case 65:
					return -1
				case 118:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 72:
					return -1
				case 65:
					return 2
				case 118:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 104:
					return -1
				case 97:
					return 2
				case 86:
					return -1
				case 105:
					return -1
				case 78:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 72:
					return -1
				case 65:
					return -1
				case 118:
					return 3
				case 73:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 104:
					return -1
				case 97:
					return -1
				case 86:
					return 3
				case 105:
					return -1
				case 78:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 104:
					return -1
				case 97:
					return -1
				case 86:
					return -1
				case 105:
					return 4
				case 78:
					return -1
				case 71:
					return -1
				case 72:
					return -1
				case 65:
					return -1
				case 118:
					return -1
				case 73:
					return 4
				case 110:
					return -1
				case 103:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 104:
					return -1
				case 97:
					return -1
				case 86:
					return -1
				case 105:
					return -1
				case 78:
					return 5
				case 71:
					return -1
				case 72:
					return -1
				case 65:
					return -1
				case 118:
					return -1
				case 73:
					return -1
				case 110:
					return 5
				case 103:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 104:
					return -1
				case 97:
					return -1
				case 86:
					return -1
				case 105:
					return -1
				case 78:
					return -1
				case 71:
					return 6
				case 72:
					return -1
				case 65:
					return -1
				case 118:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 103:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 104:
					return -1
				case 97:
					return -1
				case 86:
					return -1
				case 105:
					return -1
				case 78:
					return -1
				case 71:
					return -1
				case 72:
					return -1
				case 65:
					return -1
				case 118:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [iI][fF]
		{[]bool{false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 105:
					return 1
				case 73:
					return 1
				case 102:
					return -1
				case 70:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 102:
					return 2
				case 70:
					return 2
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

		// [iI][gG][nN][oO][rR][eE]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 73:
					return 1
				case 103:
					return -1
				case 71:
					return -1
				case 78:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 82:
					return -1
				case 105:
					return 1
				case 110:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 103:
					return 2
				case 71:
					return 2
				case 78:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 110:
					return 3
				case 114:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 78:
					return 3
				case 111:
					return -1
				case 79:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 73:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 78:
					return -1
				case 111:
					return 4
				case 79:
					return 4
				case 82:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 73:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 78:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 82:
					return 5
				case 105:
					return -1
				case 110:
					return -1
				case 114:
					return 5
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 73:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 78:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 101:
					return 6
				case 69:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 73:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 78:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [iI][lL][iI][kK][eE]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 105:
					return 1
				case 73:
					return 1
				case 108:
					return -1
				case 76:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 108:
					return 2
				case 76:
					return 2
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return 3
				case 73:
					return 3
				case 108:
					return -1
				case 76:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 107:
					return 4
				case 75:
					return 4
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return 5
				case 69:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [iI][nN]
		{[]bool{false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 105:
					return 1
				case 73:
					return 1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return 2
				case 78:
					return 2
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

		// [iI][nN][cC][lL][uU][dD][eE]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 105:
					return 1
				case 67:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				case 73:
					return 1
				case 110:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 78:
					return 2
				case 99:
					return -1
				case 73:
					return -1
				case 110:
					return 2
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 69:
					return -1
				case 105:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 73:
					return -1
				case 110:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 69:
					return -1
				case 105:
					return -1
				case 67:
					return 3
				case 78:
					return -1
				case 99:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 67:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 108:
					return 4
				case 76:
					return 4
				case 117:
					return -1
				case 85:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 67:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return 5
				case 85:
					return 5
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 78:
					return -1
				case 99:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 100:
					return 6
				case 68:
					return 6
				case 101:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 69:
					return -1
				case 105:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 69:
					return 7
				case 105:
					return -1
				case 67:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 69:
					return -1
				case 105:
					return -1
				case 67:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [iI][nN][cC][rR][eE][mM][eE][nN][tT]
		{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 77:
					return -1
				case 105:
					return 1
				case 73:
					return 1
				case 101:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 110:
					return -1
				case 67:
					return -1
				case 109:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 78:
					return 2
				case 99:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 77:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 110:
					return 2
				case 67:
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
				case 105:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 110:
					return -1
				case 67:
					return 3
				case 109:
					return -1
				case 114:
					return -1
				case 78:
					return -1
				case 99:
					return 3
				case 82:
					return -1
				case 69:
					return -1
				case 77:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return 4
				case 78:
					return -1
				case 99:
					return -1
				case 82:
					return 4
				case 69:
					return -1
				case 77:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 110:
					return -1
				case 67:
					return -1
				case 109:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 78:
					return -1
				case 99:
					return -1
				case 82:
					return -1
				case 69:
					return 5
				case 77:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 101:
					return 5
				case 116:
					return -1
				case 84:
					return -1
				case 110:
					return -1
				case 67:
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
				case 110:
					return -1
				case 67:
					return -1
				case 109:
					return 6
				case 114:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 77:
					return 6
				case 105:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 67:
					return -1
				case 109:
					return -1
				case 114:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				case 82:
					return -1
				case 69:
					return 7
				case 77:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 101:
					return 7
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 110:
					return 8
				case 67:
					return -1
				case 109:
					return -1
				case 114:
					return -1
				case 78:
					return 8
				case 99:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 77:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 78:
					return -1
				case 99:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 77:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 116:
					return 9
				case 84:
					return 9
				case 110:
					return -1
				case 67:
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
				case 105:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 110:
					return -1
				case 67:
					return -1
				case 109:
					return -1
				case 114:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 77:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [iI][nN][dD][eE][xX]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 105:
					return 1
				case 73:
					return 1
				case 78:
					return -1
				case 101:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				case 110:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return 2
				case 101:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				case 110:
					return 2
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 100:
					return 3
				case 68:
					return 3
				case 69:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return 4
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 101:
					return 4
				case 120:
					return -1
				case 88:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 120:
					return 5
				case 88:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [iI][nN][lL][iI][nN][eE]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 105:
					return 1
				case 73:
					return 1
				case 110:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return 2
				case 78:
					return 2
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 108:
					return 3
				case 76:
					return 3
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return 4
				case 73:
					return 4
				case 110:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return 5
				case 78:
					return 5
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return 6
				case 69:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [iI][nN][nN][eE][rR]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 105:
					return 1
				case 73:
					return 1
				case 110:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return 2
				case 78:
					return 2
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return 3
				case 78:
					return 3
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return 5
				case 82:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [iI][nN][sS][eE][rR][tT]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 73:
					return 1
				case 110:
					return -1
				case 78:
					return -1
				case 115:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 105:
					return 1
				case 83:
					return -1
				case 101:
					return -1
				case 82:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 82:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 73:
					return -1
				case 110:
					return 2
				case 78:
					return 2
				case 115:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 83:
					return 3
				case 101:
					return -1
				case 82:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 115:
					return 3
				case 69:
					return -1
				case 114:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 83:
					return -1
				case 101:
					return 4
				case 82:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 115:
					return -1
				case 69:
					return 4
				case 114:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 115:
					return -1
				case 69:
					return -1
				case 114:
					return 5
				case 105:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 82:
					return 5
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 115:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 82:
					return -1
				case 116:
					return 6
				case 84:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 82:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 115:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [iI][nN][tT][eE][rR][sS][eE][cC][tT]
		{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 73:
					return 1
				case 78:
					return -1
				case 114:
					return -1
				case 67:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 99:
					return -1
				case 105:
					return 1
				case 110:
					return -1
				case 116:
					return -1
				case 69:
					return -1
				case 82:
					return -1
				case 83:
					return -1
				case 115:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 73:
					return -1
				case 78:
					return 2
				case 114:
					return -1
				case 67:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 99:
					return -1
				case 105:
					return -1
				case 110:
					return 2
				case 116:
					return -1
				case 69:
					return -1
				case 82:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 84:
					return 3
				case 101:
					return -1
				case 99:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 116:
					return 3
				case 69:
					return -1
				case 82:
					return -1
				case 83:
					return -1
				case 115:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 114:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 69:
					return 4
				case 82:
					return -1
				case 83:
					return -1
				case 115:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 114:
					return -1
				case 67:
					return -1
				case 84:
					return -1
				case 101:
					return 4
				case 99:
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
				case 114:
					return 5
				case 67:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 99:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 69:
					return -1
				case 82:
					return 5
				case 83:
					return -1
				case 115:
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
				case 114:
					return -1
				case 67:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 99:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 69:
					return -1
				case 82:
					return -1
				case 83:
					return 6
				case 115:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 114:
					return -1
				case 67:
					return -1
				case 84:
					return -1
				case 101:
					return 7
				case 99:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 69:
					return 7
				case 82:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 69:
					return -1
				case 82:
					return -1
				case 83:
					return -1
				case 115:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 114:
					return -1
				case 67:
					return 8
				case 84:
					return -1
				case 101:
					return -1
				case 99:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 73:
					return -1
				case 78:
					return -1
				case 114:
					return -1
				case 67:
					return -1
				case 84:
					return 9
				case 101:
					return -1
				case 99:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 116:
					return 9
				case 69:
					return -1
				case 82:
					return -1
				case 83:
					return -1
				case 115:
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
				case 114:
					return -1
				case 67:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 99:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 69:
					return -1
				case 82:
					return -1
				case 83:
					return -1
				case 115:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [iI][nN][tT][oO]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 105:
					return 1
				case 73:
					return 1
				case 110:
					return -1
				case 78:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return 2
				case 78:
					return 2
				case 116:
					return -1
				case 84:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 116:
					return 3
				case 84:
					return 3
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 111:
					return 4
				case 79:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [iI][sS]
		{[]bool{false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 105:
					return 1
				case 73:
					return 1
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 115:
					return 2
				case 83:
					return 2
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

		// [jJ][oO][iI][nN]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 106:
					return 1
				case 74:
					return 1
				case 111:
					return -1
				case 79:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 106:
					return -1
				case 74:
					return -1
				case 111:
					return 2
				case 79:
					return 2
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 106:
					return -1
				case 74:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 105:
					return 3
				case 73:
					return 3
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 106:
					return -1
				case 74:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return 4
				case 78:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 106:
					return -1
				case 74:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [kK][eE][yY]
		{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 107:
					return 1
				case 75:
					return 1
				case 101:
					return -1
				case 69:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return 2
				case 69:
					return 2
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 121:
					return 3
				case 89:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

		// [kK][eE][yY][sS]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 107:
					return 1
				case 75:
					return 1
				case 101:
					return -1
				case 69:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return 2
				case 69:
					return 2
				case 121:
					return -1
				case 89:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 121:
					return 3
				case 89:
					return 3
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 115:
					return 4
				case 83:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [kK][eE][yY][sS][pP][aA][cC][eE]
		{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 121:
					return -1
				case 97:
					return -1
				case 67:
					return -1
				case 75:
					return 1
				case 69:
					return -1
				case 89:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				case 107:
					return 1
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return 2
				case 121:
					return -1
				case 97:
					return -1
				case 67:
					return -1
				case 75:
					return -1
				case 69:
					return 2
				case 89:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				case 107:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 121:
					return 3
				case 97:
					return -1
				case 67:
					return -1
				case 75:
					return -1
				case 69:
					return -1
				case 89:
					return 3
				case 112:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				case 107:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 121:
					return -1
				case 97:
					return -1
				case 67:
					return -1
				case 75:
					return -1
				case 69:
					return -1
				case 89:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				case 107:
					return -1
				case 115:
					return 4
				case 83:
					return 4
				case 99:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 75:
					return -1
				case 69:
					return -1
				case 89:
					return -1
				case 112:
					return 5
				case 80:
					return 5
				case 65:
					return -1
				case 107:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 121:
					return -1
				case 97:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 107:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 121:
					return -1
				case 97:
					return 6
				case 67:
					return -1
				case 75:
					return -1
				case 69:
					return -1
				case 89:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 65:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 75:
					return -1
				case 69:
					return -1
				case 89:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				case 107:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return 7
				case 101:
					return -1
				case 121:
					return -1
				case 97:
					return -1
				case 67:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 107:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 101:
					return 8
				case 121:
					return -1
				case 97:
					return -1
				case 67:
					return -1
				case 75:
					return -1
				case 69:
					return 8
				case 89:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 107:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 121:
					return -1
				case 97:
					return -1
				case 67:
					return -1
				case 75:
					return -1
				case 69:
					return -1
				case 89:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [lL][aA][sS][tT]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 108:
					return 1
				case 76:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 97:
					return 2
				case 65:
					return 2
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 115:
					return 3
				case 83:
					return 3
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return 4
				case 84:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [lL][eE][fF][tT]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 108:
					return 1
				case 76:
					return 1
				case 101:
					return -1
				case 69:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return 2
				case 69:
					return 2
				case 102:
					return -1
				case 70:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 102:
					return 3
				case 70:
					return 3
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				case 116:
					return 4
				case 84:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [lL][eE][tT]
		{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 108:
					return 1
				case 76:
					return 1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return 2
				case 69:
					return 2
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return 3
				case 84:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

		// [lL][eE][tT][tT][iI][nN][gG]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 76:
					return 1
				case 101:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 108:
					return 1
				case 116:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 76:
					return -1
				case 101:
					return 2
				case 69:
					return 2
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 116:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 84:
					return 3
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 116:
					return 3
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 84:
					return 4
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 116:
					return 4
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 116:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 105:
					return 5
				case 73:
					return 5
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return 6
				case 108:
					return -1
				case 116:
					return -1
				case 110:
					return 6
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 116:
					return -1
				case 110:
					return -1
				case 103:
					return 7
				case 71:
					return 7
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 116:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [lL][iI][kK][eE]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 108:
					return 1
				case 76:
					return 1
				case 105:
					return -1
				case 73:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 105:
					return 2
				case 73:
					return 2
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 107:
					return 3
				case 75:
					return 3
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [lL][iI][mM][iI][tT]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 108:
					return 1
				case 76:
					return 1
				case 105:
					return -1
				case 73:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 105:
					return 2
				case 73:
					return 2
				case 109:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 109:
					return 3
				case 77:
					return 3
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 105:
					return 4
				case 73:
					return 4
				case 109:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 116:
					return 5
				case 84:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [lL][sS][mM]
		{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 108:
					return 1
				case 76:
					return 1
				case 115:
					return -1
				case 83:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 115:
					return 2
				case 83:
					return 2
				case 109:
					return -1
				case 77:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 109:
					return 3
				case 77:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 76:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

		// [mM][aA][pP]
		{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 109:
					return 1
				case 77:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 77:
					return -1
				case 97:
					return 2
				case 65:
					return 2
				case 112:
					return -1
				case 80:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 112:
					return 3
				case 80:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

		// [mM][aA][pP][pP][iI][nN][gG]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 109:
					return 1
				case 77:
					return 1
				case 80:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 112:
					return -1
				case 105:
					return -1
				case 78:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 77:
					return -1
				case 80:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 97:
					return 2
				case 65:
					return 2
				case 112:
					return -1
				case 105:
					return -1
				case 78:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 112:
					return 3
				case 105:
					return -1
				case 78:
					return -1
				case 71:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 80:
					return 3
				case 73:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 77:
					return -1
				case 80:
					return 4
				case 73:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 112:
					return 4
				case 105:
					return -1
				case 78:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 77:
					return -1
				case 80:
					return -1
				case 73:
					return 5
				case 110:
					return -1
				case 103:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 112:
					return -1
				case 105:
					return 5
				case 78:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 77:
					return -1
				case 80:
					return -1
				case 73:
					return -1
				case 110:
					return 6
				case 103:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 112:
					return -1
				case 105:
					return -1
				case 78:
					return 6
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 77:
					return -1
				case 80:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 103:
					return 7
				case 97:
					return -1
				case 65:
					return -1
				case 112:
					return -1
				case 105:
					return -1
				case 78:
					return -1
				case 71:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 77:
					return -1
				case 80:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 112:
					return -1
				case 105:
					return -1
				case 78:
					return -1
				case 71:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [mM][aA][tT][cC][hH][eE][dD]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 109:
					return 1
				case 116:
					return -1
				case 84:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 104:
					return -1
				case 101:
					return -1
				case 77:
					return 1
				case 72:
					return -1
				case 100:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 104:
					return -1
				case 101:
					return -1
				case 77:
					return -1
				case 72:
					return -1
				case 100:
					return -1
				case 97:
					return 2
				case 65:
					return 2
				case 109:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 109:
					return -1
				case 116:
					return 3
				case 84:
					return 3
				case 69:
					return -1
				case 68:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 104:
					return -1
				case 101:
					return -1
				case 77:
					return -1
				case 72:
					return -1
				case 100:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 109:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 99:
					return 4
				case 67:
					return 4
				case 104:
					return -1
				case 101:
					return -1
				case 77:
					return -1
				case 72:
					return -1
				case 100:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 104:
					return 5
				case 101:
					return -1
				case 77:
					return -1
				case 72:
					return 5
				case 100:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 77:
					return -1
				case 72:
					return -1
				case 100:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 109:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 69:
					return 6
				case 68:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 104:
					return -1
				case 101:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 77:
					return -1
				case 72:
					return -1
				case 100:
					return 7
				case 97:
					return -1
				case 65:
					return -1
				case 109:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 69:
					return -1
				case 68:
					return 7
				case 99:
					return -1
				case 67:
					return -1
				case 104:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 77:
					return -1
				case 72:
					return -1
				case 100:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 109:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 104:
					return -1
				case 101:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [mM][aA][tT][eE][rR][iI][aA][lL][iI][zZ][eE][dD]
		{[]bool{false, false, false, false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 109:
					return 1
				case 77:
					return 1
				case 101:
					return -1
				case 69:
					return -1
				case 122:
					return -1
				case 68:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				case 90:
					return -1
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 100:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 122:
					return -1
				case 68:
					return -1
				case 97:
					return 2
				case 65:
					return 2
				case 82:
					return -1
				case 90:
					return -1
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 100:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				case 90:
					return -1
				case 84:
					return 3
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return 3
				case 114:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 100:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 122:
					return -1
				case 68:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 100:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				case 122:
					return -1
				case 68:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				case 90:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 82:
					return 5
				case 90:
					return -1
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 114:
					return 5
				case 105:
					return -1
				case 73:
					return -1
				case 100:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 122:
					return -1
				case 68:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 105:
					return 6
				case 73:
					return 6
				case 100:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 122:
					return -1
				case 68:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				case 90:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return 7
				case 65:
					return 7
				case 82:
					return -1
				case 90:
					return -1
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 100:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 122:
					return -1
				case 68:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 84:
					return -1
				case 108:
					return 8
				case 76:
					return 8
				case 116:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 100:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 122:
					return -1
				case 68:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				case 90:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				case 90:
					return -1
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 105:
					return 9
				case 73:
					return 9
				case 100:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 122:
					return -1
				case 68:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 100:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 122:
					return 10
				case 68:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				case 90:
					return 10
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 100:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return 11
				case 69:
					return 11
				case 122:
					return -1
				case 68:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				case 90:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 122:
					return -1
				case 68:
					return 12
				case 97:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				case 90:
					return -1
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 100:
					return 12
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				case 90:
					return -1
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 100:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 122:
					return -1
				case 68:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [mM][eE][rR][gG][eE]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 109:
					return 1
				case 77:
					return 1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return 2
				case 69:
					return 2
				case 114:
					return -1
				case 82:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return 3
				case 82:
					return 3
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 103:
					return 4
				case 71:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return 5
				case 69:
					return 5
				case 114:
					return -1
				case 82:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [mM][iI][nN][uU][sS]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 109:
					return 1
				case 105:
					return -1
				case 117:
					return -1
				case 77:
					return 1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 85:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 77:
					return -1
				case 73:
					return 2
				case 110:
					return -1
				case 78:
					return -1
				case 85:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 109:
					return -1
				case 105:
					return 2
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 77:
					return -1
				case 73:
					return -1
				case 110:
					return 3
				case 78:
					return 3
				case 85:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 109:
					return -1
				case 105:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 77:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 85:
					return 4
				case 115:
					return -1
				case 83:
					return -1
				case 109:
					return -1
				case 105:
					return -1
				case 117:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 105:
					return -1
				case 117:
					return -1
				case 77:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 85:
					return -1
				case 115:
					return 5
				case 83:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 105:
					return -1
				case 117:
					return -1
				case 77:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 85:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [mM][iI][sS][sS][iI][nN][gG]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 109:
					return 1
				case 77:
					return 1
				case 73:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 105:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return 2
				case 115:
					return -1
				case 83:
					return -1
				case 78:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 73:
					return 2
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 115:
					return 3
				case 83:
					return 3
				case 78:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 115:
					return 4
				case 83:
					return 4
				case 78:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return 5
				case 115:
					return -1
				case 83:
					return -1
				case 78:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 73:
					return 5
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 78:
					return 6
				case 109:
					return -1
				case 77:
					return -1
				case 73:
					return -1
				case 110:
					return 6
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 77:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 103:
					return 7
				case 71:
					return 7
				case 105:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 77:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 105:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 78:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [nN][aA][mM][eE][sS][pP][aA][cC][eE]
		{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 110:
					return 1
				case 97:
					return -1
				case 83:
					return -1
				case 67:
					return -1
				case 69:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 115:
					return -1
				case 112:
					return -1
				case 99:
					return -1
				case 78:
					return 1
				case 101:
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
				case 65:
					return 2
				case 109:
					return -1
				case 77:
					return -1
				case 115:
					return -1
				case 112:
					return -1
				case 99:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 110:
					return -1
				case 97:
					return 2
				case 83:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 78:
					return -1
				case 101:
					return -1
				case 110:
					return -1
				case 97:
					return -1
				case 83:
					return -1
				case 67:
					return -1
				case 69:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				case 109:
					return 3
				case 77:
					return 3
				case 115:
					return -1
				case 112:
					return -1
				case 99:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 65:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 115:
					return -1
				case 112:
					return -1
				case 99:
					return -1
				case 78:
					return -1
				case 101:
					return 4
				case 110:
					return -1
				case 97:
					return -1
				case 83:
					return -1
				case 67:
					return -1
				case 69:
					return 4
				case 80:
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
				case 65:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 115:
					return 5
				case 112:
					return -1
				case 99:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 110:
					return -1
				case 97:
					return -1
				case 83:
					return 5
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 78:
					return -1
				case 101:
					return -1
				case 110:
					return -1
				case 97:
					return -1
				case 83:
					return -1
				case 67:
					return -1
				case 69:
					return -1
				case 80:
					return 6
				case 65:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 115:
					return -1
				case 112:
					return 6
				case 99:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 78:
					return -1
				case 101:
					return -1
				case 110:
					return -1
				case 97:
					return 7
				case 83:
					return -1
				case 67:
					return -1
				case 69:
					return -1
				case 80:
					return -1
				case 65:
					return 7
				case 109:
					return -1
				case 77:
					return -1
				case 115:
					return -1
				case 112:
					return -1
				case 99:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 65:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 115:
					return -1
				case 112:
					return -1
				case 99:
					return 8
				case 78:
					return -1
				case 101:
					return -1
				case 110:
					return -1
				case 97:
					return -1
				case 83:
					return -1
				case 67:
					return 8
				case 69:
					return -1
				case 80:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 69:
					return 9
				case 80:
					return -1
				case 65:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 115:
					return -1
				case 112:
					return -1
				case 99:
					return -1
				case 78:
					return -1
				case 101:
					return 9
				case 110:
					return -1
				case 97:
					return -1
				case 83:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 78:
					return -1
				case 101:
					return -1
				case 110:
					return -1
				case 97:
					return -1
				case 83:
					return -1
				case 67:
					return -1
				case 69:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 115:
					return -1
				case 112:
					return -1
				case 99:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [nN][eE][sS][tT]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 110:
					return 1
				case 78:
					return 1
				case 101:
					return -1
				case 69:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 78:
					return -1
				case 101:
					return 2
				case 69:
					return 2
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 115:
					return 3
				case 83:
					return 3
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return 4
				case 84:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [nN][oO][tT]
		{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 110:
					return 1
				case 78:
					return 1
				case 111:
					return -1
				case 79:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 78:
					return -1
				case 111:
					return 2
				case 79:
					return 2
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 78:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 116:
					return 3
				case 84:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 78:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

		// [nN][uU][lL][lL]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 110:
					return 1
				case 78:
					return 1
				case 117:
					return -1
				case 85:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 78:
					return -1
				case 117:
					return 2
				case 85:
					return 2
				case 108:
					return -1
				case 76:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 78:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 108:
					return 3
				case 76:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 78:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 108:
					return 4
				case 76:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 78:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [nN][uN][mM][bB][eE][rR]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 110:
					return 1
				case 78:
					return 1
				case 109:
					return -1
				case 77:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 117:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 78:
					return 2
				case 109:
					return -1
				case 77:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 117:
					return 2
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 78:
					return -1
				case 109:
					return 3
				case 77:
					return 3
				case 98:
					return -1
				case 66:
					return -1
				case 117:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 98:
					return 4
				case 66:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 101:
					return 5
				case 69:
					return 5
				case 114:
					return -1
				case 82:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return 6
				case 82:
					return 6
				case 110:
					return -1
				case 78:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [oO][bB][jJ][eE][cC][tT]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 111:
					return 1
				case 98:
					return -1
				case 74:
					return -1
				case 67:
					return -1
				case 84:
					return -1
				case 79:
					return 1
				case 66:
					return -1
				case 106:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 98:
					return 2
				case 74:
					return -1
				case 67:
					return -1
				case 84:
					return -1
				case 79:
					return -1
				case 66:
					return 2
				case 106:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 79:
					return -1
				case 66:
					return -1
				case 106:
					return 3
				case 101:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 116:
					return -1
				case 111:
					return -1
				case 98:
					return -1
				case 74:
					return 3
				case 67:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 98:
					return -1
				case 74:
					return -1
				case 67:
					return -1
				case 84:
					return -1
				case 79:
					return -1
				case 66:
					return -1
				case 106:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				case 99:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 98:
					return -1
				case 74:
					return -1
				case 67:
					return 5
				case 84:
					return -1
				case 79:
					return -1
				case 66:
					return -1
				case 106:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 99:
					return 5
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 79:
					return -1
				case 66:
					return -1
				case 106:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 116:
					return 6
				case 111:
					return -1
				case 98:
					return -1
				case 74:
					return -1
				case 67:
					return -1
				case 84:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 79:
					return -1
				case 66:
					return -1
				case 106:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 116:
					return -1
				case 111:
					return -1
				case 98:
					return -1
				case 74:
					return -1
				case 67:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [oO][fF][fF][sS][eE][tT]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 111:
					return 1
				case 79:
					return 1
				case 70:
					return -1
				case 69:
					return -1
				case 102:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return 2
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 70:
					return 2
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 79:
					return -1
				case 70:
					return 3
				case 69:
					return -1
				case 102:
					return 3
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 79:
					return -1
				case 70:
					return -1
				case 69:
					return -1
				case 102:
					return -1
				case 115:
					return 4
				case 83:
					return 4
				case 101:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return 5
				case 116:
					return -1
				case 84:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 70:
					return -1
				case 69:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 116:
					return 6
				case 84:
					return 6
				case 111:
					return -1
				case 79:
					return -1
				case 70:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 79:
					return -1
				case 70:
					return -1
				case 69:
					return -1
				case 102:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [oO][nN]
		{[]bool{false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 111:
					return 1
				case 79:
					return 1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 79:
					return -1
				case 110:
					return 2
				case 78:
					return 2
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 79:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

		// [oO][pP][tT][iI][oO][nN]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 111:
					return 1
				case 79:
					return 1
				case 112:
					return -1
				case 80:
					return -1
				case 84:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 116:
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
				case 111:
					return -1
				case 79:
					return -1
				case 112:
					return 2
				case 80:
					return 2
				case 84:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 116:
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
				case 111:
					return -1
				case 79:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 84:
					return 3
				case 73:
					return -1
				case 78:
					return -1
				case 116:
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
				case 116:
					return -1
				case 105:
					return 4
				case 110:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 84:
					return -1
				case 73:
					return 4
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return 5
				case 79:
					return 5
				case 112:
					return -1
				case 80:
					return -1
				case 84:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 116:
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
				case 116:
					return -1
				case 105:
					return -1
				case 110:
					return 6
				case 111:
					return -1
				case 79:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 84:
					return -1
				case 73:
					return -1
				case 78:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 79:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 84:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 116:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [oO][rR]
		{[]bool{false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 111:
					return 1
				case 79:
					return 1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return 2
				case 82:
					return 2
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

		// [oO][rR][dD][eE][rR]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 111:
					return 1
				case 79:
					return 1
				case 114:
					return -1
				case 82:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return 2
				case 82:
					return 2
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 100:
					return 3
				case 68:
					return 3
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return 5
				case 82:
					return 5
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [oO][uU][tT][eE][rR]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 79:
					return 1
				case 117:
					return -1
				case 116:
					return -1
				case 101:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 111:
					return 1
				case 85:
					return -1
				case 84:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 79:
					return -1
				case 117:
					return 2
				case 116:
					return -1
				case 101:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 111:
					return -1
				case 85:
					return 2
				case 84:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 85:
					return -1
				case 84:
					return 3
				case 69:
					return -1
				case 79:
					return -1
				case 117:
					return -1
				case 116:
					return 3
				case 101:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 79:
					return -1
				case 117:
					return -1
				case 116:
					return -1
				case 101:
					return 4
				case 114:
					return -1
				case 82:
					return -1
				case 111:
					return -1
				case 85:
					return -1
				case 84:
					return -1
				case 69:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 85:
					return -1
				case 84:
					return -1
				case 69:
					return -1
				case 79:
					return -1
				case 117:
					return -1
				case 116:
					return -1
				case 101:
					return -1
				case 114:
					return 5
				case 82:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 79:
					return -1
				case 117:
					return -1
				case 116:
					return -1
				case 101:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 111:
					return -1
				case 85:
					return -1
				case 84:
					return -1
				case 69:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [oO][vV][eE][rR]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 111:
					return 1
				case 79:
					return 1
				case 118:
					return -1
				case 86:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 79:
					return -1
				case 118:
					return 2
				case 86:
					return 2
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 79:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				case 101:
					return 3
				case 69:
					return 3
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 79:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return 4
				case 82:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 79:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [pP][aA][rR][tT][iI][tT][iI][oO][nN]
		{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 80:
					return 1
				case 116:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 112:
					return 1
				case 82:
					return -1
				case 78:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				case 65:
					return 2
				case 110:
					return -1
				case 97:
					return 2
				case 114:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 80:
					return -1
				case 116:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 114:
					return 3
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 80:
					return -1
				case 116:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 112:
					return -1
				case 82:
					return 3
				case 78:
					return -1
				case 65:
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
				case 110:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 84:
					return 4
				case 105:
					return -1
				case 73:
					return -1
				case 80:
					return -1
				case 116:
					return 4
				case 111:
					return -1
				case 79:
					return -1
				case 112:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 84:
					return -1
				case 105:
					return 5
				case 73:
					return 5
				case 80:
					return -1
				case 116:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 65:
					return -1
				case 110:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 84:
					return 6
				case 105:
					return -1
				case 73:
					return -1
				case 80:
					return -1
				case 116:
					return 6
				case 111:
					return -1
				case 79:
					return -1
				case 112:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 80:
					return -1
				case 116:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 112:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 84:
					return -1
				case 105:
					return 7
				case 73:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 80:
					return -1
				case 116:
					return -1
				case 111:
					return 8
				case 79:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 114:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 80:
					return -1
				case 116:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 112:
					return -1
				case 82:
					return -1
				case 78:
					return 9
				case 65:
					return -1
				case 110:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 65:
					return -1
				case 110:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 80:
					return -1
				case 116:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 112:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [pP][aA][sS][sS][wW][oO][rR][dD]
		{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 79:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 80:
					return 1
				case 65:
					return -1
				case 111:
					return -1
				case 82:
					return -1
				case 112:
					return 1
				case 97:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 114:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 65:
					return 2
				case 111:
					return -1
				case 82:
					return -1
				case 112:
					return -1
				case 97:
					return 2
				case 119:
					return -1
				case 87:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 79:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 80:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 97:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 114:
					return -1
				case 115:
					return 3
				case 83:
					return 3
				case 79:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				case 111:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return 4
				case 83:
					return 4
				case 79:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				case 111:
					return -1
				case 82:
					return -1
				case 112:
					return -1
				case 97:
					return -1
				case 119:
					return -1
				case 87:
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
				case 111:
					return -1
				case 82:
					return -1
				case 112:
					return -1
				case 97:
					return -1
				case 119:
					return 5
				case 87:
					return 5
				case 114:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 79:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 80:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 97:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 79:
					return 6
				case 100:
					return -1
				case 68:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				case 111:
					return 6
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 80:
					return -1
				case 65:
					return -1
				case 111:
					return -1
				case 82:
					return 7
				case 112:
					return -1
				case 97:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 114:
					return 7
				case 115:
					return -1
				case 83:
					return -1
				case 79:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 80:
					return -1
				case 65:
					return -1
				case 111:
					return -1
				case 82:
					return -1
				case 112:
					return -1
				case 97:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 79:
					return -1
				case 100:
					return 8
				case 68:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 79:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				case 111:
					return -1
				case 82:
					return -1
				case 112:
					return -1
				case 97:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 114:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [pP][aA][tT][hH]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 112:
					return 1
				case 80:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 97:
					return 2
				case 65:
					return 2
				case 116:
					return -1
				case 84:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 116:
					return 3
				case 84:
					return 3
				case 104:
					return -1
				case 72:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 104:
					return 4
				case 72:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [pP][oO][oO][lL]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 112:
					return 1
				case 80:
					return 1
				case 111:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 111:
					return 2
				case 79:
					return 2
				case 108:
					return -1
				case 76:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 111:
					return 3
				case 79:
					return 3
				case 108:
					return -1
				case 76:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 108:
					return 4
				case 76:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [pP][rR][eE][pP][aA][rR][eE]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 112:
					return 1
				case 80:
					return 1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 114:
					return 2
				case 82:
					return 2
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return 3
				case 69:
					return 3
				case 97:
					return -1
				case 65:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return 4
				case 80:
					return 4
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return 5
				case 65:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 114:
					return 6
				case 82:
					return 6
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return 7
				case 69:
					return 7
				case 97:
					return -1
				case 65:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [pP][rR][iI][mM][aA][rR][yY]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 80:
					return 1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 109:
					return -1
				case 65:
					return -1
				case 121:
					return -1
				case 112:
					return 1
				case 73:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 89:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 73:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 89:
					return -1
				case 80:
					return -1
				case 114:
					return 2
				case 82:
					return 2
				case 105:
					return -1
				case 109:
					return -1
				case 65:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 73:
					return 3
				case 77:
					return -1
				case 97:
					return -1
				case 89:
					return -1
				case 80:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return 3
				case 109:
					return -1
				case 65:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 73:
					return -1
				case 77:
					return 4
				case 97:
					return -1
				case 89:
					return -1
				case 80:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 109:
					return 4
				case 65:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 73:
					return -1
				case 77:
					return -1
				case 97:
					return 5
				case 89:
					return -1
				case 80:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 109:
					return -1
				case 65:
					return 5
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 73:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 89:
					return -1
				case 80:
					return -1
				case 114:
					return 6
				case 82:
					return 6
				case 105:
					return -1
				case 109:
					return -1
				case 65:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 73:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 89:
					return 7
				case 80:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 109:
					return -1
				case 65:
					return -1
				case 121:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 73:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 89:
					return -1
				case 80:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 109:
					return -1
				case 65:
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
				case 118:
					return -1
				case 86:
					return -1
				case 101:
					return -1
				case 80:
					return 1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 112:
					return 1
				case 73:
					return -1
				case 84:
					return -1
				case 69:
					return -1
				case 65:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 118:
					return -1
				case 86:
					return -1
				case 101:
					return -1
				case 80:
					return -1
				case 114:
					return 2
				case 82:
					return 2
				case 105:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 112:
					return -1
				case 73:
					return -1
				case 84:
					return -1
				case 69:
					return -1
				case 65:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 65:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				case 101:
					return -1
				case 80:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return 3
				case 97:
					return -1
				case 116:
					return -1
				case 112:
					return -1
				case 73:
					return 3
				case 84:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 80:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 112:
					return -1
				case 73:
					return -1
				case 84:
					return -1
				case 69:
					return -1
				case 65:
					return -1
				case 118:
					return 4
				case 86:
					return 4
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 80:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 97:
					return 5
				case 116:
					return -1
				case 112:
					return -1
				case 73:
					return -1
				case 84:
					return -1
				case 69:
					return -1
				case 65:
					return 5
				case 118:
					return -1
				case 86:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 73:
					return -1
				case 84:
					return 6
				case 69:
					return -1
				case 65:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				case 101:
					return -1
				case 80:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 97:
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
				case 118:
					return -1
				case 86:
					return -1
				case 101:
					return 7
				case 80:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 112:
					return -1
				case 73:
					return -1
				case 84:
					return -1
				case 69:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 118:
					return -1
				case 86:
					return -1
				case 101:
					return -1
				case 80:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 112:
					return -1
				case 73:
					return -1
				case 84:
					return -1
				case 69:
					return -1
				case 65:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [pP][rR][iI][vV][iI][lL][eE][gG][eE]
		{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 118:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 69:
					return -1
				case 71:
					return -1
				case 112:
					return 1
				case 80:
					return 1
				case 82:
					return -1
				case 86:
					return -1
				case 103:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 82:
					return 2
				case 86:
					return -1
				case 103:
					return -1
				case 101:
					return -1
				case 114:
					return 2
				case 105:
					return -1
				case 73:
					return -1
				case 118:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 69:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 114:
					return -1
				case 105:
					return 3
				case 73:
					return 3
				case 118:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 69:
					return -1
				case 71:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 82:
					return -1
				case 86:
					return -1
				case 103:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 82:
					return -1
				case 86:
					return 4
				case 103:
					return -1
				case 101:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 118:
					return 4
				case 108:
					return -1
				case 76:
					return -1
				case 69:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 114:
					return -1
				case 105:
					return 5
				case 73:
					return 5
				case 118:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 69:
					return -1
				case 71:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 82:
					return -1
				case 86:
					return -1
				case 103:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 118:
					return -1
				case 108:
					return 6
				case 76:
					return 6
				case 69:
					return -1
				case 71:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 82:
					return -1
				case 86:
					return -1
				case 103:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 118:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 69:
					return 7
				case 71:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 82:
					return -1
				case 86:
					return -1
				case 103:
					return -1
				case 101:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 82:
					return -1
				case 86:
					return -1
				case 103:
					return 8
				case 101:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 118:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 69:
					return -1
				case 71:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return 9
				case 114:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 118:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 69:
					return 9
				case 71:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 82:
					return -1
				case 86:
					return -1
				case 103:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 118:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 69:
					return -1
				case 71:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 82:
					return -1
				case 86:
					return -1
				case 103:
					return -1
				case 101:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [pP][rR][oO][cC][eE][dE][uU][rR][eE]
		{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 85:
					return -1
				case 112:
					return 1
				case 80:
					return 1
				case 82:
					return -1
				case 67:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 82:
					return 2
				case 67:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 114:
					return 2
				case 111:
					return -1
				case 79:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 85:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 82:
					return -1
				case 67:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 114:
					return -1
				case 111:
					return 3
				case 79:
					return 3
				case 99:
					return -1
				case 100:
					return -1
				case 85:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 82:
					return -1
				case 67:
					return 4
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 114:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 99:
					return 4
				case 100:
					return -1
				case 85:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 82:
					return -1
				case 67:
					return -1
				case 101:
					return 5
				case 69:
					return 5
				case 117:
					return -1
				case 114:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 85:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 82:
					return -1
				case 67:
					return -1
				case 101:
					return -1
				case 69:
					return 6
				case 117:
					return -1
				case 114:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 99:
					return -1
				case 100:
					return 6
				case 85:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 82:
					return -1
				case 67:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return 7
				case 114:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 85:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return 8
				case 111:
					return -1
				case 79:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 85:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 82:
					return 8
				case 67:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 85:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 82:
					return -1
				case 67:
					return -1
				case 101:
					return 9
				case 69:
					return 9
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 82:
					return -1
				case 67:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 114:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 85:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [pP][uU][bB][lL][iI][cC]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 80:
					return 1
				case 108:
					return -1
				case 76:
					return -1
				case 112:
					return 1
				case 117:
					return -1
				case 85:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 117:
					return 2
				case 85:
					return 2
				case 98:
					return -1
				case 66:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 80:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 98:
					return 3
				case 66:
					return 3
				case 105:
					return -1
				case 73:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 80:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 80:
					return -1
				case 108:
					return 4
				case 76:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 80:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 112:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 105:
					return 5
				case 73:
					return 5
				case 99:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 99:
					return 6
				case 67:
					return 6
				case 80:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 80:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 112:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [rR][aA][wW]
		{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 114:
					return 1
				case 82:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return 2
				case 65:
					return 2
				case 119:
					return -1
				case 87:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 119:
					return 3
				case 87:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

		// [rR][eE][aA][lL][mM]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 77:
					return -1
				case 114:
					return 1
				case 82:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 109:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 109:
					return -1
				case 101:
					return 2
				case 69:
					return 2
				case 77:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 77:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return 3
				case 65:
					return 3
				case 108:
					return -1
				case 76:
					return -1
				case 109:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 108:
					return 4
				case 76:
					return 4
				case 109:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 77:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 109:
					return 5
				case 101:
					return -1
				case 69:
					return -1
				case 77:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 77:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 109:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [rR][eE][dD][uU][cC][eE]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 114:
					return 1
				case 82:
					return 1
				case 100:
					return -1
				case 117:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 85:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 100:
					return -1
				case 117:
					return -1
				case 99:
					return -1
				case 101:
					return 2
				case 69:
					return 2
				case 68:
					return -1
				case 85:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 68:
					return 3
				case 85:
					return -1
				case 67:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 100:
					return 3
				case 117:
					return -1
				case 99:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 85:
					return 4
				case 67:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 100:
					return -1
				case 117:
					return 4
				case 99:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 100:
					return -1
				case 117:
					return -1
				case 99:
					return 5
				case 101:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 85:
					return -1
				case 67:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 100:
					return -1
				case 117:
					return -1
				case 99:
					return -1
				case 101:
					return 6
				case 69:
					return 6
				case 68:
					return -1
				case 85:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 68:
					return -1
				case 85:
					return -1
				case 67:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 100:
					return -1
				case 117:
					return -1
				case 99:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [rR][eE][nN][aA][mM][eE]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 114:
					return 1
				case 110:
					return -1
				case 78:
					return -1
				case 82:
					return 1
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 82:
					return -1
				case 101:
					return 2
				case 69:
					return 2
				case 97:
					return -1
				case 65:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 110:
					return 3
				case 78:
					return 3
				case 82:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 82:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return 4
				case 65:
					return 4
				case 109:
					return -1
				case 77:
					return -1
				case 114:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 109:
					return 5
				case 77:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 82:
					return -1
				case 101:
					return 6
				case 69:
					return 6
				case 97:
					return -1
				case 65:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 114:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 82:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 114:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [rR][eE][tT][uU][rR][nN]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 82:
					return 1
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 78:
					return -1
				case 114:
					return 1
				case 101:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 82:
					return -1
				case 69:
					return 2
				case 116:
					return -1
				case 84:
					return -1
				case 78:
					return -1
				case 114:
					return -1
				case 101:
					return 2
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 101:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 116:
					return 3
				case 84:
					return 3
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 82:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 78:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 117:
					return 4
				case 85:
					return 4
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 82:
					return 5
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 78:
					return -1
				case 114:
					return 5
				case 101:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 82:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 78:
					return 6
				case 114:
					return -1
				case 101:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 82:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 78:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [rR][eE][tT][uU][rR][nN][iI][nN][gG]
		{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 114:
					return 1
				case 73:
					return -1
				case 101:
					return -1
				case 84:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 105:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 116:
					return -1
				case 82:
					return 1
				case 69:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return 2
				case 84:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 105:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 116:
					return -1
				case 82:
					return -1
				case 69:
					return 2
				case 78:
					return -1
				case 114:
					return -1
				case 73:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return 3
				case 82:
					return -1
				case 69:
					return -1
				case 78:
					return -1
				case 114:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 84:
					return 3
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 105:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 84:
					return -1
				case 117:
					return 4
				case 85:
					return 4
				case 110:
					return -1
				case 105:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 116:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 84:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 105:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 116:
					return -1
				case 82:
					return 5
				case 69:
					return -1
				case 78:
					return -1
				case 114:
					return 5
				case 73:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 78:
					return 6
				case 114:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 84:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return 6
				case 105:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 73:
					return 7
				case 101:
					return -1
				case 84:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 105:
					return 7
				case 103:
					return -1
				case 71:
					return -1
				case 116:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 84:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return 8
				case 105:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 116:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 78:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 84:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 105:
					return -1
				case 103:
					return 9
				case 71:
					return 9
				case 116:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 84:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 105:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 116:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 78:
					return -1
				case 114:
					return -1
				case 73:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [rR][eE][vV][oO][kK][eE]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 114:
					return 1
				case 101:
					return -1
				case 118:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 82:
					return 1
				case 69:
					return -1
				case 86:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 101:
					return 2
				case 118:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 82:
					return -1
				case 69:
					return 2
				case 86:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 82:
					return -1
				case 69:
					return -1
				case 86:
					return 3
				case 114:
					return -1
				case 101:
					return -1
				case 118:
					return 3
				case 111:
					return -1
				case 79:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 101:
					return -1
				case 118:
					return -1
				case 111:
					return 4
				case 79:
					return 4
				case 107:
					return -1
				case 75:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 86:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 82:
					return -1
				case 69:
					return -1
				case 86:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 118:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 107:
					return 5
				case 75:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 82:
					return -1
				case 69:
					return 6
				case 86:
					return -1
				case 114:
					return -1
				case 101:
					return 6
				case 118:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 82:
					return -1
				case 69:
					return -1
				case 86:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 118:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [rR][iI][gG][hH][tT]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 114:
					return 1
				case 82:
					return 1
				case 72:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 104:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return 2
				case 73:
					return 2
				case 103:
					return -1
				case 71:
					return -1
				case 104:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 72:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 72:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 103:
					return 3
				case 71:
					return 3
				case 104:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 72:
					return 4
				case 116:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 104:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 104:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 72:
					return -1
				case 116:
					return 5
				case 84:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 73:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 104:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 72:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [rR][oO][lL][eE]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 114:
					return 1
				case 82:
					return 1
				case 111:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 111:
					return 2
				case 79:
					return 2
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 108:
					return 3
				case 76:
					return 3
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [rR][oO][lL][lL][bB][aA][cC][kK]
		{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 82:
					return 1
				case 98:
					return -1
				case 99:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 111:
					return -1
				case 66:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 114:
					return 1
				case 79:
					return -1
				case 108:
					return -1
				case 65:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 82:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 111:
					return 2
				case 66:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 79:
					return 2
				case 108:
					return -1
				case 65:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 79:
					return -1
				case 108:
					return 3
				case 65:
					return -1
				case 67:
					return -1
				case 82:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 111:
					return -1
				case 66:
					return -1
				case 76:
					return 3
				case 97:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 66:
					return -1
				case 76:
					return 4
				case 97:
					return -1
				case 114:
					return -1
				case 79:
					return -1
				case 108:
					return 4
				case 65:
					return -1
				case 67:
					return -1
				case 82:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 76:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 65:
					return -1
				case 67:
					return -1
				case 82:
					return -1
				case 98:
					return 5
				case 99:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 111:
					return -1
				case 66:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 65:
					return 6
				case 67:
					return -1
				case 82:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 111:
					return -1
				case 66:
					return -1
				case 76:
					return -1
				case 97:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 66:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 65:
					return -1
				case 67:
					return 7
				case 82:
					return -1
				case 98:
					return -1
				case 99:
					return 7
				case 107:
					return -1
				case 75:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 66:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 65:
					return -1
				case 67:
					return -1
				case 82:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 107:
					return 8
				case 75:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 66:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 65:
					return -1
				case 67:
					return -1
				case 82:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [sS][aA][tT][iI][sS][fF][iI][eE][sS]
		{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 83:
					return 1
				case 65:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				case 69:
					return -1
				case 115:
					return 1
				case 97:
					return -1
				case 116:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 97:
					return 2
				case 116:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 83:
					return -1
				case 65:
					return 2
				case 84:
					return -1
				case 105:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 83:
					return -1
				case 65:
					return -1
				case 84:
					return 3
				case 105:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				case 69:
					return -1
				case 115:
					return -1
				case 97:
					return -1
				case 116:
					return 3
				case 73:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 83:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				case 105:
					return 4
				case 102:
					return -1
				case 70:
					return -1
				case 69:
					return -1
				case 115:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 73:
					return 4
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return 5
				case 97:
					return -1
				case 116:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 83:
					return 5
				case 65:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 83:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 102:
					return 6
				case 70:
					return 6
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 83:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				case 105:
					return 7
				case 102:
					return -1
				case 70:
					return -1
				case 69:
					return -1
				case 115:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 73:
					return 7
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 83:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				case 69:
					return 8
				case 115:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 73:
					return -1
				case 101:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 83:
					return 9
				case 65:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				case 69:
					return -1
				case 115:
					return 9
				case 97:
					return -1
				case 116:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 83:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				case 69:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [sS][cC][hH][eE][mM][aA]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 115:
					return 1
				case 83:
					return 1
				case 67:
					return -1
				case 104:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 99:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 109:
					return -1
				case 65:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return 2
				case 72:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 109:
					return -1
				case 65:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 67:
					return 2
				case 104:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 72:
					return 3
				case 101:
					return -1
				case 69:
					return -1
				case 109:
					return -1
				case 65:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 67:
					return -1
				case 104:
					return 3
				case 77:
					return -1
				case 97:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 67:
					return -1
				case 104:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 99:
					return -1
				case 72:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				case 109:
					return -1
				case 65:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 67:
					return -1
				case 104:
					return -1
				case 77:
					return 5
				case 97:
					return -1
				case 99:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 109:
					return 5
				case 65:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 67:
					return -1
				case 104:
					return -1
				case 77:
					return -1
				case 97:
					return 6
				case 99:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 109:
					return -1
				case 65:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 109:
					return -1
				case 65:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 67:
					return -1
				case 104:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [sS][eE][lL][eE][cC][tT]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 108:
					return -1
				case 67:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 115:
					return 1
				case 83:
					return 1
				case 69:
					return -1
				case 76:
					return -1
				case 99:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return 2
				case 108:
					return -1
				case 67:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 69:
					return 2
				case 76:
					return -1
				case 99:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 108:
					return 3
				case 67:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 69:
					return -1
				case 76:
					return 3
				case 99:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 69:
					return 4
				case 76:
					return -1
				case 99:
					return -1
				case 101:
					return 4
				case 108:
					return -1
				case 67:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 108:
					return -1
				case 67:
					return 5
				case 116:
					return -1
				case 84:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				case 99:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 108:
					return -1
				case 67:
					return -1
				case 116:
					return 6
				case 84:
					return 6
				case 115:
					return -1
				case 83:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				case 99:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 108:
					return -1
				case 67:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				case 99:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [sS][eE][lL][fF]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 115:
					return 1
				case 83:
					return 1
				case 101:
					return -1
				case 69:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return 2
				case 69:
					return 2
				case 108:
					return -1
				case 76:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 108:
					return 3
				case 76:
					return 3
				case 102:
					return -1
				case 70:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 102:
					return 4
				case 70:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [sS][eE][tT]
		{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 115:
					return 1
				case 83:
					return 1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return 2
				case 69:
					return 2
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return 3
				case 84:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

		// [sS][hH][oO][wW]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 115:
					return 1
				case 83:
					return 1
				case 104:
					return -1
				case 72:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 104:
					return 2
				case 72:
					return 2
				case 111:
					return -1
				case 79:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 111:
					return 3
				case 79:
					return 3
				case 119:
					return -1
				case 87:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 119:
					return 4
				case 87:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [sS][oO][mM][eE]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 115:
					return 1
				case 83:
					return 1
				case 111:
					return -1
				case 79:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 111:
					return 2
				case 79:
					return 2
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 109:
					return 3
				case 77:
					return 3
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [sS][tT][aA][rR][tT]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 115:
					return 1
				case 83:
					return 1
				case 116:
					return -1
				case 84:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return 2
				case 84:
					return 2
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 97:
					return 3
				case 65:
					return 3
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return 4
				case 82:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return 5
				case 84:
					return 5
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [sS][tT][aA][tT][iI][sS][tT][iI][cC][sS]
		{[]bool{false, false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 99:
					return -1
				case 115:
					return 1
				case 83:
					return 1
				case 84:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return 2
				case 99:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 84:
					return 2
				case 97:
					return -1
				case 65:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 84:
					return -1
				case 97:
					return 3
				case 65:
					return 3
				case 105:
					return -1
				case 73:
					return -1
				case 67:
					return -1
				case 116:
					return -1
				case 99:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 84:
					return 4
				case 97:
					return -1
				case 65:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 67:
					return -1
				case 116:
					return 4
				case 99:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 99:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 84:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 105:
					return 5
				case 73:
					return 5
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 99:
					return -1
				case 115:
					return 6
				case 83:
					return 6
				case 84:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return 7
				case 99:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 84:
					return 7
				case 97:
					return -1
				case 65:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 84:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 105:
					return 8
				case 73:
					return 8
				case 67:
					return -1
				case 116:
					return -1
				case 99:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 84:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 67:
					return 9
				case 116:
					return -1
				case 99:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return 10
				case 83:
					return 10
				case 84:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 67:
					return -1
				case 116:
					return -1
				case 99:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 99:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 84:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 67:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [sS][tT][rR][iI][nN][gG]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 84:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 115:
					return 1
				case 83:
					return 1
				case 116:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return 2
				case 114:
					return -1
				case 82:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 71:
					return -1
				case 84:
					return 2
				case 105:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 84:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 114:
					return 3
				case 82:
					return 3
				case 73:
					return -1
				case 78:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 84:
					return -1
				case 105:
					return 4
				case 110:
					return -1
				case 103:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 73:
					return 4
				case 78:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 73:
					return -1
				case 78:
					return 5
				case 71:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 110:
					return 5
				case 103:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 71:
					return 6
				case 84:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 103:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 84:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 71:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [sS][yY][sS][tT][eE][mM]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 115:
					return 1
				case 84:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 83:
					return 1
				case 121:
					return -1
				case 89:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 83:
					return -1
				case 121:
					return 2
				case 89:
					return 2
				case 116:
					return -1
				case 115:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 83:
					return 3
				case 121:
					return -1
				case 89:
					return -1
				case 116:
					return -1
				case 115:
					return 3
				case 84:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 84:
					return 4
				case 101:
					return -1
				case 69:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 83:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 116:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 83:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 116:
					return -1
				case 115:
					return -1
				case 84:
					return -1
				case 101:
					return 5
				case 69:
					return 5
				case 109:
					return -1
				case 77:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 109:
					return 6
				case 77:
					return 6
				case 83:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 83:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 116:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [tT][hH][eE][nN]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 116:
					return 1
				case 84:
					return 1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 84:
					return -1
				case 104:
					return 2
				case 72:
					return 2
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 84:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return 3
				case 69:
					return 3
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 84:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return 4
				case 78:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 84:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [tT][oO]
		{[]bool{false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 116:
					return 1
				case 84:
					return 1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 84:
					return -1
				case 111:
					return 2
				case 79:
					return 2
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 84:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

		// [tT][rR][aA][nN][sS][aA][cC][tT][iI][oO][nN]
		{[]bool{false, false, false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 116:
					return 1
				case 84:
					return 1
				case 82:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 67:
					return -1
				case 105:
					return -1
				case 114:
					return -1
				case 97:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				case 73:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 73:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return 2
				case 65:
					return -1
				case 110:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 67:
					return -1
				case 105:
					return -1
				case 114:
					return 2
				case 97:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 65:
					return 3
				case 110:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 67:
					return -1
				case 105:
					return -1
				case 114:
					return -1
				case 97:
					return 3
				case 78:
					return -1
				case 99:
					return -1
				case 73:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 65:
					return -1
				case 110:
					return 4
				case 115:
					return -1
				case 83:
					return -1
				case 67:
					return -1
				case 105:
					return -1
				case 114:
					return -1
				case 97:
					return -1
				case 78:
					return 4
				case 99:
					return -1
				case 73:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 115:
					return 5
				case 83:
					return 5
				case 67:
					return -1
				case 105:
					return -1
				case 114:
					return -1
				case 97:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				case 73:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 73:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 65:
					return 6
				case 110:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 67:
					return -1
				case 105:
					return -1
				case 114:
					return -1
				case 97:
					return 6
				case 78:
					return -1
				case 99:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 67:
					return 7
				case 105:
					return -1
				case 114:
					return -1
				case 97:
					return -1
				case 78:
					return -1
				case 99:
					return 7
				case 73:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 97:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				case 73:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 116:
					return 8
				case 84:
					return 8
				case 82:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 67:
					return -1
				case 105:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 73:
					return 9
				case 111:
					return -1
				case 79:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 67:
					return -1
				case 105:
					return 9
				case 114:
					return -1
				case 97:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 97:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				case 73:
					return -1
				case 111:
					return 10
				case 79:
					return 10
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 67:
					return -1
				case 105:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 65:
					return -1
				case 110:
					return 11
				case 115:
					return -1
				case 83:
					return -1
				case 67:
					return -1
				case 105:
					return -1
				case 114:
					return -1
				case 97:
					return -1
				case 78:
					return 11
				case 99:
					return -1
				case 73:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 67:
					return -1
				case 105:
					return -1
				case 114:
					return -1
				case 97:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				case 73:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [tT][rR][iI][gG][gG][eE][rR]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 116:
					return 1
				case 84:
					return 1
				case 82:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 103:
					return -1
				case 114:
					return -1
				case 71:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return 2
				case 71:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return 2
				case 105:
					return -1
				case 73:
					return -1
				case 103:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 71:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 105:
					return 3
				case 73:
					return 3
				case 103:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 71:
					return 4
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 103:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 71:
					return 5
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 103:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 103:
					return -1
				case 114:
					return -1
				case 71:
					return -1
				case 101:
					return 6
				case 69:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return 7
				case 105:
					return -1
				case 73:
					return -1
				case 103:
					return -1
				case 114:
					return 7
				case 71:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 71:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 103:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [tT][rR][uU][eE]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 116:
					return 1
				case 84:
					return 1
				case 114:
					return -1
				case 82:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 84:
					return -1
				case 114:
					return 2
				case 82:
					return 2
				case 117:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 84:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 117:
					return 3
				case 85:
					return 3
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 84:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 84:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [tT][rR][uU][nN][cC][aA][tT][eE]
		{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 116:
					return 1
				case 82:
					return -1
				case 99:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				case 84:
					return 1
				case 117:
					return -1
				case 110:
					return -1
				case 101:
					return -1
				case 114:
					return -1
				case 85:
					return -1
				case 78:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 82:
					return 2
				case 99:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 117:
					return -1
				case 110:
					return -1
				case 101:
					return -1
				case 114:
					return 2
				case 85:
					return -1
				case 78:
					return -1
				case 67:
					return -1
				case 97:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 85:
					return 3
				case 78:
					return -1
				case 67:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 82:
					return -1
				case 99:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 117:
					return 3
				case 110:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 85:
					return -1
				case 78:
					return 4
				case 67:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 82:
					return -1
				case 99:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 117:
					return -1
				case 110:
					return 4
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 116:
					return -1
				case 82:
					return -1
				case 99:
					return 5
				case 65:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 117:
					return -1
				case 110:
					return -1
				case 101:
					return -1
				case 114:
					return -1
				case 85:
					return -1
				case 78:
					return -1
				case 67:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 82:
					return -1
				case 99:
					return -1
				case 65:
					return 6
				case 69:
					return -1
				case 84:
					return -1
				case 117:
					return -1
				case 110:
					return -1
				case 101:
					return -1
				case 114:
					return -1
				case 85:
					return -1
				case 78:
					return -1
				case 67:
					return -1
				case 97:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 85:
					return -1
				case 78:
					return -1
				case 67:
					return -1
				case 97:
					return -1
				case 116:
					return 7
				case 82:
					return -1
				case 99:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				case 84:
					return 7
				case 117:
					return -1
				case 110:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 85:
					return -1
				case 78:
					return -1
				case 67:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 82:
					return -1
				case 99:
					return -1
				case 65:
					return -1
				case 69:
					return 8
				case 84:
					return -1
				case 117:
					return -1
				case 110:
					return -1
				case 101:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 85:
					return -1
				case 78:
					return -1
				case 67:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 82:
					return -1
				case 99:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 117:
					return -1
				case 110:
					return -1
				case 101:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [uU][nN][dD][eE][rR]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 117:
					return 1
				case 110:
					return -1
				case 78:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 82:
					return -1
				case 85:
					return 1
				case 68:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 110:
					return 2
				case 78:
					return 2
				case 100:
					return -1
				case 101:
					return -1
				case 82:
					return -1
				case 85:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 85:
					return -1
				case 68:
					return 3
				case 69:
					return -1
				case 114:
					return -1
				case 117:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 100:
					return 3
				case 101:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 85:
					return -1
				case 68:
					return -1
				case 69:
					return 4
				case 114:
					return -1
				case 117:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 100:
					return -1
				case 101:
					return 4
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 82:
					return 5
				case 85:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 114:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 82:
					return -1
				case 85:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [uU][nN][iI][oO][nN]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 117:
					return 1
				case 85:
					return 1
				case 110:
					return -1
				case 78:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return 2
				case 78:
					return 2
				case 105:
					return -1
				case 73:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 105:
					return 3
				case 73:
					return 3
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 111:
					return 4
				case 79:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return 5
				case 78:
					return 5
				case 105:
					return -1
				case 73:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [uU][nN][iI][qQ][uU][eE]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 85:
					return 1
				case 110:
					return -1
				case 78:
					return -1
				case 73:
					return -1
				case 81:
					return -1
				case 117:
					return 1
				case 105:
					return -1
				case 113:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 105:
					return -1
				case 113:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 85:
					return -1
				case 110:
					return 2
				case 78:
					return 2
				case 73:
					return -1
				case 81:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 85:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 73:
					return 3
				case 81:
					return -1
				case 117:
					return -1
				case 105:
					return 3
				case 113:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 105:
					return -1
				case 113:
					return 4
				case 101:
					return -1
				case 69:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 73:
					return -1
				case 81:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return 5
				case 105:
					return -1
				case 113:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 85:
					return 5
				case 110:
					return -1
				case 78:
					return -1
				case 73:
					return -1
				case 81:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 105:
					return -1
				case 113:
					return -1
				case 101:
					return 6
				case 69:
					return 6
				case 85:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 73:
					return -1
				case 81:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 85:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 73:
					return -1
				case 81:
					return -1
				case 117:
					return -1
				case 105:
					return -1
				case 113:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [uU][nN][nN][eE][sS][tT]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 78:
					return -1
				case 69:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 117:
					return 1
				case 85:
					return 1
				case 110:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 78:
					return 2
				case 69:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return 2
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return 3
				case 101:
					return -1
				case 78:
					return 3
				case 69:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 78:
					return -1
				case 69:
					return 4
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 101:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 101:
					return -1
				case 78:
					return -1
				case 69:
					return -1
				case 115:
					return 5
				case 83:
					return 5
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 101:
					return -1
				case 78:
					return -1
				case 69:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return 6
				case 84:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 101:
					return -1
				case 78:
					return -1
				case 69:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [uU][nN][sS][eE][tT]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 117:
					return 1
				case 85:
					return 1
				case 78:
					return -1
				case 83:
					return -1
				case 84:
					return -1
				case 110:
					return -1
				case 115:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 78:
					return 2
				case 83:
					return -1
				case 84:
					return -1
				case 110:
					return 2
				case 115:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 78:
					return -1
				case 83:
					return 3
				case 84:
					return -1
				case 110:
					return -1
				case 115:
					return 3
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 115:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				case 116:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 78:
					return -1
				case 83:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 78:
					return -1
				case 83:
					return -1
				case 84:
					return 5
				case 110:
					return -1
				case 115:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 115:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 78:
					return -1
				case 83:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [uU][pP][dD][aA][tT][eE]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 85:
					return 1
				case 100:
					return -1
				case 68:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 117:
					return 1
				case 112:
					return -1
				case 80:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 112:
					return 2
				case 80:
					return 2
				case 97:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				case 85:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 85:
					return -1
				case 100:
					return 3
				case 68:
					return 3
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 117:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 97:
					return 4
				case 65:
					return 4
				case 69:
					return -1
				case 85:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 85:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 116:
					return 5
				case 84:
					return 5
				case 101:
					return -1
				case 117:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 69:
					return 6
				case 85:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				case 85:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [uU][pP][sS][eE][rR][tT]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 117:
					return 1
				case 85:
					return 1
				case 112:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 80:
					return -1
				case 115:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 112:
					return 2
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 80:
					return 2
				case 115:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 112:
					return -1
				case 83:
					return 3
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 80:
					return -1
				case 115:
					return 3
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 112:
					return -1
				case 83:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				case 116:
					return -1
				case 84:
					return -1
				case 80:
					return -1
				case 115:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 80:
					return -1
				case 115:
					return -1
				case 114:
					return 5
				case 82:
					return 5
				case 117:
					return -1
				case 85:
					return -1
				case 112:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 80:
					return -1
				case 115:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 112:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return 6
				case 84:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 80:
					return -1
				case 115:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 112:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [uU][sS][eE]
		{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 117:
					return 1
				case 85:
					return 1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 115:
					return 2
				case 83:
					return 2
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return 3
				case 69:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

		// [uU][sS][eE][rR]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 117:
					return 1
				case 85:
					return 1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 115:
					return 2
				case 83:
					return 2
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return 3
				case 69:
					return 3
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return 4
				case 82:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [uU][sS][iI][nN][gG]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 117:
					return 1
				case 83:
					return -1
				case 110:
					return -1
				case 85:
					return 1
				case 115:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 83:
					return 2
				case 110:
					return -1
				case 85:
					return -1
				case 115:
					return 2
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				case 85:
					return -1
				case 115:
					return -1
				case 105:
					return 3
				case 73:
					return 3
				case 78:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 83:
					return -1
				case 110:
					return 4
				case 85:
					return -1
				case 115:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return 4
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 85:
					return -1
				case 115:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 103:
					return 5
				case 71:
					return 5
				case 117:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				case 85:
					return -1
				case 115:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [vV][aA][lL][uU][eE]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 86:
					return 1
				case 76:
					return -1
				case 85:
					return -1
				case 69:
					return -1
				case 118:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 108:
					return -1
				case 117:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 86:
					return -1
				case 76:
					return -1
				case 85:
					return -1
				case 69:
					return -1
				case 118:
					return -1
				case 97:
					return 2
				case 65:
					return 2
				case 108:
					return -1
				case 117:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 86:
					return -1
				case 76:
					return 3
				case 85:
					return -1
				case 69:
					return -1
				case 118:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 108:
					return 3
				case 117:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 118:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 108:
					return -1
				case 117:
					return 4
				case 101:
					return -1
				case 86:
					return -1
				case 76:
					return -1
				case 85:
					return 4
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 86:
					return -1
				case 76:
					return -1
				case 85:
					return -1
				case 69:
					return 5
				case 118:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 108:
					return -1
				case 117:
					return -1
				case 101:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 118:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 108:
					return -1
				case 117:
					return -1
				case 101:
					return -1
				case 86:
					return -1
				case 76:
					return -1
				case 85:
					return -1
				case 69:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [vV][aA][lL][uU][eE][dD]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 118:
					return 1
				case 65:
					return -1
				case 108:
					return -1
				case 117:
					return -1
				case 69:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 86:
					return 1
				case 97:
					return -1
				case 76:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 86:
					return -1
				case 97:
					return 2
				case 76:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 118:
					return -1
				case 65:
					return 2
				case 108:
					return -1
				case 117:
					return -1
				case 69:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 86:
					return -1
				case 97:
					return -1
				case 76:
					return 3
				case 85:
					return -1
				case 101:
					return -1
				case 118:
					return -1
				case 65:
					return -1
				case 108:
					return 3
				case 117:
					return -1
				case 69:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 86:
					return -1
				case 97:
					return -1
				case 76:
					return -1
				case 85:
					return 4
				case 101:
					return -1
				case 118:
					return -1
				case 65:
					return -1
				case 108:
					return -1
				case 117:
					return 4
				case 69:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 86:
					return -1
				case 97:
					return -1
				case 76:
					return -1
				case 85:
					return -1
				case 101:
					return 5
				case 118:
					return -1
				case 65:
					return -1
				case 108:
					return -1
				case 117:
					return -1
				case 69:
					return 5
				case 100:
					return -1
				case 68:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 118:
					return -1
				case 65:
					return -1
				case 108:
					return -1
				case 117:
					return -1
				case 69:
					return -1
				case 100:
					return 6
				case 68:
					return 6
				case 86:
					return -1
				case 97:
					return -1
				case 76:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 118:
					return -1
				case 65:
					return -1
				case 108:
					return -1
				case 117:
					return -1
				case 69:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 86:
					return -1
				case 97:
					return -1
				case 76:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [vV][aA][lL][uU][eE][sS]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 86:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 83:
					return -1
				case 118:
					return 1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 69:
					return -1
				case 115:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 86:
					return -1
				case 97:
					return 2
				case 65:
					return 2
				case 85:
					return -1
				case 101:
					return -1
				case 83:
					return -1
				case 118:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 69:
					return -1
				case 115:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 86:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 83:
					return -1
				case 118:
					return -1
				case 108:
					return 3
				case 76:
					return 3
				case 117:
					return -1
				case 69:
					return -1
				case 115:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 118:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return 4
				case 69:
					return -1
				case 115:
					return -1
				case 86:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 85:
					return 4
				case 101:
					return -1
				case 83:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 86:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 85:
					return -1
				case 101:
					return 5
				case 83:
					return -1
				case 118:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 69:
					return 5
				case 115:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 86:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 83:
					return 6
				case 118:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 69:
					return -1
				case 115:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 118:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 69:
					return -1
				case 115:
					return -1
				case 86:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 83:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [vV][iI][eE][wW]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 118:
					return 1
				case 86:
					return 1
				case 105:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 118:
					return -1
				case 86:
					return -1
				case 105:
					return 2
				case 73:
					return 2
				case 101:
					return -1
				case 69:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 118:
					return -1
				case 86:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 101:
					return 3
				case 69:
					return 3
				case 119:
					return -1
				case 87:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 118:
					return -1
				case 86:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 119:
					return 4
				case 87:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 118:
					return -1
				case 86:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [wW][hH][eE][nN]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 119:
					return 1
				case 87:
					return 1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 104:
					return 2
				case 72:
					return 2
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return 3
				case 69:
					return 3
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return 4
				case 78:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [wW][hH][eE][rR][eE]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 119:
					return 1
				case 87:
					return 1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 104:
					return 2
				case 72:
					return 2
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return 3
				case 69:
					return 3
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return 4
				case 82:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return 5
				case 69:
					return 5
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [wW][hH][iI][lL][eE]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 72:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 119:
					return 1
				case 87:
					return 1
				case 104:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 104:
					return 2
				case 105:
					return -1
				case 108:
					return -1
				case 69:
					return -1
				case 72:
					return 2
				case 73:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 72:
					return -1
				case 73:
					return 3
				case 76:
					return -1
				case 101:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 104:
					return -1
				case 105:
					return 3
				case 108:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 104:
					return -1
				case 105:
					return -1
				case 108:
					return 4
				case 69:
					return -1
				case 72:
					return -1
				case 73:
					return -1
				case 76:
					return 4
				case 101:
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
				case 76:
					return -1
				case 101:
					return 5
				case 119:
					return -1
				case 87:
					return -1
				case 104:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 69:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 72:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 104:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 69:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [wW][iI][tT][hH]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 119:
					return 1
				case 87:
					return 1
				case 105:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 105:
					return 2
				case 73:
					return 2
				case 116:
					return -1
				case 84:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 116:
					return 3
				case 84:
					return 3
				case 104:
					return -1
				case 72:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 104:
					return 4
				case 72:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [wW][iI][tT][hH][iI][nN]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 119:
					return 1
				case 87:
					return 1
				case 73:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 104:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 105:
					return -1
				case 72:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 73:
					return 2
				case 116:
					return -1
				case 84:
					return -1
				case 104:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 105:
					return 2
				case 72:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 73:
					return -1
				case 116:
					return 3
				case 84:
					return 3
				case 104:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 105:
					return -1
				case 72:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 72:
					return 4
				case 119:
					return -1
				case 87:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 104:
					return 4
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return 5
				case 72:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 73:
					return 5
				case 116:
					return -1
				case 84:
					return -1
				case 104:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 104:
					return -1
				case 110:
					return 6
				case 78:
					return 6
				case 105:
					return -1
				case 72:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 72:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 104:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [wW][oO][rR][kK]
		{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 119:
					return 1
				case 87:
					return 1
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 111:
					return 2
				case 79:
					return 2
				case 114:
					return -1
				case 82:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return 3
				case 82:
					return 3
				case 107:
					return -1
				case 75:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 107:
					return 4
				case 75:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 119:
					return -1
				case 87:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// [xX][oO][rR]
		{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 120:
					return 1
				case 88:
					return 1
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 120:
					return -1
				case 88:
					return -1
				case 111:
					return 2
				case 79:
					return 2
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 120:
					return -1
				case 88:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return 3
				case 82:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 120:
					return -1
				case 88:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
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

		// \$[a-zA-Z_][a-zA-Z0-9_]*
		{[]bool{false, false, true, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 36:
					return 1
				case 95:
					return -1
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
				case 95:
					return 2
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
				case 95:
					return 3
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
				case 95:
					return 3
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

		// \$[1-9][0-9]*
		{[]bool{false, false, true, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 36:
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
	})
	return yylex
}
func NewLexer(in io.Reader) *Lexer {
	return NewLexerWithInit(in, nil)
}
func (yylex *Lexer) Text() string {
	return yylex.stack[len(yylex.stack)-1].s
}
func (yylex *Lexer) next(lvl int) int {
	if lvl == len(yylex.stack) {
		yylex.stack = append(yylex.stack, intstring{0, ""})
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
	yylex.stack = yylex.stack[:len(yylex.stack)-1]
}
func (yylex Lexer) Error(e string) {
	panic(e)
}
func (yylex *Lexer) Lex(lval *yySymType) int {
	for {
		switch yylex.next(0) {
		case 0:
			{
				lval.s, _ = UnmarshalDoubleQuoted(yylex.Text())
				logToken("STRING - %s", lval.s)
				return STRING
			}
			continue
		case 1:
			{
				lval.s, _ = UnmarshalSingleQuoted(yylex.Text())
				logToken("STRING - %s", lval.s)
				return STRING
			}
			continue
		case 2:
			{
				// Case-insensitive identifier
				text := yylex.Text()
				text = text[0 : len(text)-1]
				lval.s, _ = UnmarshalBackQuoted(text)
				logToken("IDENTIFIER_ICASE - %s", lval.s)
				return IDENTIFIER_ICASE
			}
			continue
		case 3:
			{
				// Escaped identifier
				lval.s, _ = UnmarshalBackQuoted(yylex.Text())
				logToken("IDENTIFIER - %s", lval.s)
				return IDENTIFIER
			}
			continue
		case 4:
			{
				// We differentiate NUMBER from INT
				lval.f, _ = strconv.ParseFloat(yylex.Text(), 64)
				logToken("NUMBER - %f", lval.f)
				return NUMBER
			}
			continue
		case 5:
			{
				// We differentiate NUMBER from INT
				lval.f, _ = strconv.ParseFloat(yylex.Text(), 64)
				logToken("NUMBER - %f", lval.f)
				return NUMBER
			}
			continue
		case 6:
			{
				// We differentiate NUMBER from INT
				lval.n, _ = strconv.Atoi(yylex.Text())
				logToken("INT - %d", lval.n)
				return INT
			}
			continue
		case 7:
			{
				logToken("BLOCK_COMMENT (length=%d)", len(yylex.Text())) /* eat up block comment */
			}
			continue
		case 8:
			{
				logToken("LINE_COMMENT (length=%d)", len(yylex.Text())) /* eat up line comment */
			}
			continue
		case 9:
			{
				logToken("WHITESPACE (count=%d)", len(yylex.Text())) /* eat up whitespace */
			}
			continue
		case 10:
			{
				logToken("DOT")
				return DOT
			}
			continue
		case 11:
			{
				logToken("PLUS")
				return PLUS
			}
			continue
		case 12:
			{
				logToken("MINUS")
				return MINUS
			}
			continue
		case 13:
			{
				logToken("MULT")
				return STAR
			}
			continue
		case 14:
			{
				logToken("DIV")
				return DIV
			}
			continue
		case 15:
			{
				logToken("MOD")
				return MOD
			}
			continue
		case 16:
			{
				logToken("DEQ")
				return DEQ
			}
			continue
		case 17:
			{
				logToken("EQ")
				return EQ
			}
			continue
		case 18:
			{
				logToken("NE")
				return NE
			}
			continue
		case 19:
			{
				logToken("NE")
				return NE
			}
			continue
		case 20:
			{
				logToken("LT")
				return LT
			}
			continue
		case 21:
			{
				logToken("LTE")
				return LE
			}
			continue
		case 22:
			{
				logToken("GT")
				return GT
			}
			continue
		case 23:
			{
				logToken("GTE")
				return GE
			}
			continue
		case 24:
			{
				logToken("CONCAT")
				return CONCAT
			}
			continue
		case 25:
			{
				logToken("LPAREN")
				return LPAREN
			}
			continue
		case 26:
			{
				logToken("RPAREN")
				return RPAREN
			}
			continue
		case 27:
			{
				logToken("LBRACE")
				return LBRACE
			}
			continue
		case 28:
			{
				logToken("RBRACE")
				return RBRACE
			}
			continue
		case 29:
			{
				logToken("COMMA")
				return COMMA
			}
			continue
		case 30:
			{
				logToken("COLON")
				return COLON
			}
			continue
		case 31:
			{
				logToken("LBRACKET")
				return LBRACKET
			}
			continue
		case 32:
			{
				logToken("RBRACKET")
				return RBRACKET
			}
			continue
		case 33:
			{
				logToken("RBRACKET_ICASE")
				return RBRACKET_ICASE
			}
			continue
		case 34:
			{
				logToken("SEMI")
				return SEMI
			}
			continue
		case 35:
			{
				logToken("ALL")
				return ALL
			}
			continue
		case 36:
			{
				logToken("ALTER")
				return ALTER
			}
			continue
		case 37:
			{
				logToken("ANALYZE")
				return ANALYZE
			}
			continue
		case 38:
			{
				logToken("AND")
				return AND
			}
			continue
		case 39:
			{
				logToken("ANY")
				return ANY
			}
			continue
		case 40:
			{
				logToken("ARRAY")
				return ARRAY
			}
			continue
		case 41:
			{
				logToken("AS")
				return AS
			}
			continue
		case 42:
			{
				logToken("ASC")
				return ASC
			}
			continue
		case 43:
			{
				logToken("BEGIN")
				return BEGIN
			}
			continue
		case 44:
			{
				logToken("BETWEEN")
				return BETWEEN
			}
			continue
		case 45:
			{
				logToken("BINARY")
				return BINARY
			}
			continue
		case 46:
			{
				logToken("BOOLEAN")
				return BOOLEAN
			}
			continue
		case 47:
			{
				logToken("BREAK")
				return BREAK
			}
			continue
		case 48:
			{
				logToken("BUCKET")
				return BUCKET
			}
			continue
		case 49:
			{
				logToken("BUILD")
				return BUILD
			}
			continue
		case 50:
			{
				logToken("BY")
				return BY
			}
			continue
		case 51:
			{
				logToken("CALL")
				return CALL
			}
			continue
		case 52:
			{
				logToken("CASE")
				return CASE
			}
			continue
		case 53:
			{
				logToken("CAST")
				return CAST
			}
			continue
		case 54:
			{
				logToken("CLUSTER")
				return CLUSTER
			}
			continue
		case 55:
			{
				logToken("COLLATE")
				return COLLATE
			}
			continue
		case 56:
			{
				logToken("COLLECTION")
				return COLLECTION
			}
			continue
		case 57:
			{
				logToken("COMMIT")
				return COMMIT
			}
			continue
		case 58:
			{
				logToken("CONNECT")
				return CONNECT
			}
			continue
		case 59:
			{
				logToken("CONTINUE")
				return CONTINUE
			}
			continue
		case 60:
			{
				logToken("CREATE")
				return CREATE
			}
			continue
		case 61:
			{
				logToken("DATABASE")
				return DATABASE
			}
			continue
		case 62:
			{
				logToken("DATASET")
				return DATASET
			}
			continue
		case 63:
			{
				logToken("DATASTORE")
				return DATASTORE
			}
			continue
		case 64:
			{
				logToken("DECLARE")
				return DECLARE
			}
			continue
		case 65:
			{
				logToken("DECREMENT")
				return DECREMENT
			}
			continue
		case 66:
			{
				logToken("DELETE")
				return DELETE
			}
			continue
		case 67:
			{
				logToken("DERIVED")
				return DERIVED
			}
			continue
		case 68:
			{
				logToken("DESC")
				return DESC
			}
			continue
		case 69:
			{
				logToken("DESCRIBE")
				return DESCRIBE
			}
			continue
		case 70:
			{
				logToken("DISTINCT")
				return DISTINCT
			}
			continue
		case 71:
			{
				logToken("DO")
				return DO
			}
			continue
		case 72:
			{
				logToken("DROP")
				return DROP
			}
			continue
		case 73:
			{
				logToken("EACH")
				return EACH
			}
			continue
		case 74:
			{
				logToken("ELEMENT")
				return ELEMENT
			}
			continue
		case 75:
			{
				logToken("ELSE")
				return ELSE
			}
			continue
		case 76:
			{
				logToken("END")
				return END
			}
			continue
		case 77:
			{
				logToken("EVERY")
				return EVERY
			}
			continue
		case 78:
			{
				logToken("EXCEPT")
				return EXCEPT
			}
			continue
		case 79:
			{
				logToken("EXCLUDE")
				return EXCLUDE
			}
			continue
		case 80:
			{
				logToken("EXECUTE")
				return EXECUTE
			}
			continue
		case 81:
			{
				logToken("EXISTS")
				return EXISTS
			}
			continue
		case 82:
			{
				logToken("EXPLAIN")
				return EXPLAIN
			}
			continue
		case 83:
			{
				logToken("FALSE")
				return FALSE
			}
			continue
		case 84:
			{
				logToken("FIRST")
				return FIRST
			}
			continue
		case 85:
			{
				logToken("FLATTEN")
				return FLATTEN
			}
			continue
		case 86:
			{
				logToken("FOR")
				return FOR
			}
			continue
		case 87:
			{
				logToken("FORCE")
				return FORCE
			}
			continue
		case 88:
			{
				logToken("FROM")
				return FROM
			}
			continue
		case 89:
			{
				logToken("FUNCTION")
				return FUNCTION
			}
			continue
		case 90:
			{
				logToken("GRANT")
				return GRANT
			}
			continue
		case 91:
			{
				logToken("GROUP")
				return GROUP
			}
			continue
		case 92:
			{
				logToken("GSI")
				return GSI
			}
			continue
		case 93:
			{
				logToken("HAVING")
				return HAVING
			}
			continue
		case 94:
			{
				logToken("IF")
				return IF
			}
			continue
		case 95:
			{
				logToken("IGNORE")
				return IGNORE
			}
			continue
		case 96:
			{
				logToken("ILIKE")
				return ILIKE
			}
			continue
		case 97:
			{
				logToken("IN")
				return IN
			}
			continue
		case 98:
			{
				logToken("INCLUDE")
				return INCLUDE
			}
			continue
		case 99:
			{
				logToken("INCREMENT")
				return INCREMENT
			}
			continue
		case 100:
			{
				logToken("INDEX")
				return INDEX
			}
			continue
		case 101:
			{
				logToken("INLINE")
				return INLINE
			}
			continue
		case 102:
			{
				logToken("INNER")
				return INNER
			}
			continue
		case 103:
			{
				logToken("INSERT")
				return INSERT
			}
			continue
		case 104:
			{
				logToken("INTERSECT")
				return INTERSECT
			}
			continue
		case 105:
			{
				logToken("INTO")
				return INTO
			}
			continue
		case 106:
			{
				logToken("IS")
				return IS
			}
			continue
		case 107:
			{
				logToken("JOIN")
				return JOIN
			}
			continue
		case 108:
			{
				logToken("KEY")
				return KEY
			}
			continue
		case 109:
			{
				logToken("KEYS")
				return KEYS
			}
			continue
		case 110:
			{
				logToken("KEYSPACE")
				return KEYSPACE
			}
			continue
		case 111:
			{
				logToken("LAST")
				return LAST
			}
			continue
		case 112:
			{
				logToken("LEFT")
				return LEFT
			}
			continue
		case 113:
			{
				logToken("LET")
				return LET
			}
			continue
		case 114:
			{
				logToken("LETTING")
				return LETTING
			}
			continue
		case 115:
			{
				logToken("LIKE")
				return LIKE
			}
			continue
		case 116:
			{
				logToken("LIMIT")
				return LIMIT
			}
			continue
		case 117:
			{
				logToken("LSM")
				return LSM
			}
			continue
		case 118:
			{
				logToken("MAP")
				return MAP
			}
			continue
		case 119:
			{
				logToken("MAPPING")
				return MAPPING
			}
			continue
		case 120:
			{
				logToken("MATCHED")
				return MATCHED
			}
			continue
		case 121:
			{
				logToken("MATERIALIZED")
				return MATERIALIZED
			}
			continue
		case 122:
			{
				logToken("MERGE")
				return MERGE
			}
			continue
		case 123:
			{
				logToken("MINUS")
				return MINUS
			}
			continue
		case 124:
			{
				logToken("MISSING")
				return MISSING
			}
			continue
		case 125:
			{
				logToken("NAMESPACE")
				return NAMESPACE
			}
			continue
		case 126:
			{
				logToken("NEST")
				return NEST
			}
			continue
		case 127:
			{
				logToken("NOT")
				return NOT
			}
			continue
		case 128:
			{
				logToken("NULL")
				return NULL
			}
			continue
		case 129:
			{
				logToken("NUMBER")
				return NUMBER
			}
			continue
		case 130:
			{
				logToken("OBJECT")
				return OBJECT
			}
			continue
		case 131:
			{
				logToken("OFFSET")
				return OFFSET
			}
			continue
		case 132:
			{
				logToken("ON")
				return ON
			}
			continue
		case 133:
			{
				logToken("OPTION")
				return OPTION
			}
			continue
		case 134:
			{
				logToken("OR")
				return OR
			}
			continue
		case 135:
			{
				logToken("ORDER")
				return ORDER
			}
			continue
		case 136:
			{
				logToken("OUTER")
				return OUTER
			}
			continue
		case 137:
			{
				logToken("OVER")
				return OVER
			}
			continue
		case 138:
			{
				logToken("PARTITION")
				return PARTITION
			}
			continue
		case 139:
			{
				logToken("PASSWORD")
				return PASSWORD
			}
			continue
		case 140:
			{
				logToken("PATH")
				return PATH
			}
			continue
		case 141:
			{
				logToken("POOL")
				return POOL
			}
			continue
		case 142:
			{
				logToken("PREPARE")
				return PREPARE
			}
			continue
		case 143:
			{
				logToken("PRIMARY")
				return PRIMARY
			}
			continue
		case 144:
			{
				logToken("PRIVATE")
				return PRIVATE
			}
			continue
		case 145:
			{
				logToken("PRIVILEGE")
				return PRIVILEGE
			}
			continue
		case 146:
			{
				logToken("PROCEDURE")
				return PROCEDURE
			}
			continue
		case 147:
			{
				logToken("PUBLIC")
				return PUBLIC
			}
			continue
		case 148:
			{
				logToken("RAW")
				return RAW
			}
			continue
		case 149:
			{
				logToken("REALM")
				return REALM
			}
			continue
		case 150:
			{
				logToken("REDUCE")
				return REDUCE
			}
			continue
		case 151:
			{
				logToken("RENAME")
				return RENAME
			}
			continue
		case 152:
			{
				logToken("RETURN")
				return RETURN
			}
			continue
		case 153:
			{
				logToken("RETURNING")
				return RETURNING
			}
			continue
		case 154:
			{
				logToken("REVOKE")
				return REVOKE
			}
			continue
		case 155:
			{
				logToken("RIGHT")
				return RIGHT
			}
			continue
		case 156:
			{
				logToken("ROLE")
				return ROLE
			}
			continue
		case 157:
			{
				logToken("ROLLBACK")
				return ROLLBACK
			}
			continue
		case 158:
			{
				logToken("SATISFIES")
				return SATISFIES
			}
			continue
		case 159:
			{
				logToken("SCHEMA")
				return SCHEMA
			}
			continue
		case 160:
			{
				logToken("SELECT")
				return SELECT
			}
			continue
		case 161:
			{
				logToken("SELF")
				return SELF
			}
			continue
		case 162:
			{
				logToken("SET")
				return SET
			}
			continue
		case 163:
			{
				logToken("SHOW")
				return SHOW
			}
			continue
		case 164:
			{
				logToken("SOME")
				return SOME
			}
			continue
		case 165:
			{
				logToken("START")
				return START
			}
			continue
		case 166:
			{
				logToken("STATISTICS")
				return STATISTICS
			}
			continue
		case 167:
			{
				logToken("STRING")
				return STRING
			}
			continue
		case 168:
			{
				logToken("SYSTEM")
				return SYSTEM
			}
			continue
		case 169:
			{
				logToken("THEN")
				return THEN
			}
			continue
		case 170:
			{
				logToken("TO")
				return TO
			}
			continue
		case 171:
			{
				logToken("TRANSACTION")
				return TRANSACTION
			}
			continue
		case 172:
			{
				logToken("TRIGGER")
				return TRIGGER
			}
			continue
		case 173:
			{
				logToken("TRUE")
				return TRUE
			}
			continue
		case 174:
			{
				logToken("TRUNCATE")
				return TRUNCATE
			}
			continue
		case 175:
			{
				logToken("UNDER")
				return UNDER
			}
			continue
		case 176:
			{
				logToken("UNION")
				return UNION
			}
			continue
		case 177:
			{
				logToken("UNIQUE")
				return UNIQUE
			}
			continue
		case 178:
			{
				logToken("UNNEST")
				return UNNEST
			}
			continue
		case 179:
			{
				logToken("UNSET")
				return UNSET
			}
			continue
		case 180:
			{
				logToken("UPDATE")
				return UPDATE
			}
			continue
		case 181:
			{
				logToken("UPSERT")
				return UPSERT
			}
			continue
		case 182:
			{
				logToken("USE")
				return USE
			}
			continue
		case 183:
			{
				logToken("USER")
				return USER
			}
			continue
		case 184:
			{
				logToken("USING")
				return USING
			}
			continue
		case 185:
			{
				logToken("VALUE")
				return VALUE
			}
			continue
		case 186:
			{
				logToken("VALUED")
				return VALUED
			}
			continue
		case 187:
			{
				logToken("VALUES")
				return VALUES
			}
			continue
		case 188:
			{
				logToken("VIEW")
				return VIEW
			}
			continue
		case 189:
			{
				logToken("WHEN")
				return WHEN
			}
			continue
		case 190:
			{
				logToken("WHERE")
				return WHERE
			}
			continue
		case 191:
			{
				logToken("WHILE")
				return WHILE
			}
			continue
		case 192:
			{
				logToken("WITH")
				return WITH
			}
			continue
		case 193:
			{
				logToken("WITHIN")
				return WITHIN
			}
			continue
		case 194:
			{
				logToken("WORK")
				return WORK
			}
			continue
		case 195:
			{
				logToken("XOR")
				return XOR
			}
			continue
		case 196:
			{
				lval.s = yylex.Text()
				logToken("IDENTIFIER - %s", lval.s)
				return IDENTIFIER
			}
			continue
		case 197:
			{
				lval.s = yylex.Text()[1:]
				logToken("NAMED_PARAM - %s", lval.s)
				return NAMED_PARAM
			}
			continue
		case 198:
			{
				lval.n, _ = strconv.Atoi(yylex.Text()[1:])
				logToken("POSITIONAL_PARAM - %d", lval.n)
				return POSITIONAL_PARAM
			}
			continue
		case 199:
			{
				lval.n = 0 // Handled by parser
				logToken("NEXT_PARAM - ?")
				return NEXT_PARAM
			}
			continue
		}
		break
	}
	yylex.pop()

	return 0
}
func logToken(format string, v ...interface{}) {
	clog.To("LEXER", format, v...)
}
