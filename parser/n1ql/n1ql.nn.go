package n1ql

import "math"
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
		// \"((\\\")|[^\"])*\"
		{[]bool{false, false, true, false, false, true}, []func(rune) int{ // Transitions
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
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// '(('')|[^'])*'
		{[]bool{false, false, true, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 39:
					return 1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 39:
					return 2
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 39:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 39:
					return 2
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 39:
					return 2
				}
				return 3
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// `((``)|[^`])+`i
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 96:
					return 1
				case 105:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 96:
					return 2
				case 105:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 96:
					return 5
				case 105:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 96:
					return 4
				case 105:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 96:
					return 5
				case 105:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 96:
					return 4
				case 105:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 96:
					return -1
				case 105:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// `((``)|[^`])+`
		{[]bool{false, false, false, false, true, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 96:
					return 1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 96:
					return 2
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 96:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 96:
					return 4
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 96:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 96:
					return 4
				}
				return 3
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

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
				case 65:
					return 1
				case 108:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return 1
				case 76:
					return -1
				case 116:
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
				case 65:
					return -1
				case 108:
					return 2
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return -1
				case 76:
					return 2
				case 116:
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
				case 65:
					return -1
				case 108:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return -1
				case 76:
					return -1
				case 116:
					return 3
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
				case 76:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				case 65:
					return -1
				case 108:
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
				case 65:
					return -1
				case 108:
					return -1
				case 114:
					return 5
				case 82:
					return 5
				case 97:
					return -1
				case 76:
					return -1
				case 116:
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
				case 65:
					return -1
				case 108:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return -1
				case 76:
					return -1
				case 116:
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
				case 76:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 90:
					return -1
				case 97:
					return 1
				case 65:
					return 1
				case 110:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 122:
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
				case 76:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 90:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 110:
					return 2
				case 78:
					return 2
				case 108:
					return -1
				case 122:
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
					return 3
				case 65:
					return 3
				case 110:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 122:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 90:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 76:
					return 4
				case 121:
					return -1
				case 89:
					return -1
				case 90:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 108:
					return 4
				case 122:
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
				case 65:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 122:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				case 121:
					return 5
				case 89:
					return 5
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
				case 110:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 122:
					return 6
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 90:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 76:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 90:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 122:
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
				case 97:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 122:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 90:
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
				case 69:
					return -1
				case 71:
					return -1
				case 73:
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
				case 103:
					return -1
				case 105:
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
					return 2
				case 103:
					return -1
				case 105:
					return -1
				case 69:
					return 2
				case 71:
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
				case 69:
					return -1
				case 71:
					return 3
				case 73:
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
				case 103:
					return 3
				case 105:
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
					return 4
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
				case 103:
					return -1
				case 105:
					return 4
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
				case 103:
					return -1
				case 105:
					return -1
				case 69:
					return -1
				case 71:
					return -1
				case 73:
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
				case 98:
					return -1
				case 66:
					return -1
				case 101:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 69:
					return -1
				case 71:
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
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [bB][eE][tT][wW][eE][eE][nN]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 69:
					return -1
				case 116:
					return -1
				case 119:
					return -1
				case 110:
					return -1
				case 98:
					return 1
				case 66:
					return 1
				case 101:
					return -1
				case 84:
					return -1
				case 87:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 69:
					return 2
				case 116:
					return -1
				case 119:
					return -1
				case 110:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 101:
					return 2
				case 84:
					return -1
				case 87:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 69:
					return -1
				case 116:
					return 3
				case 119:
					return -1
				case 110:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 101:
					return -1
				case 84:
					return 3
				case 87:
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
				case 84:
					return -1
				case 87:
					return 4
				case 78:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 119:
					return 4
				case 110:
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
				case 84:
					return -1
				case 87:
					return -1
				case 78:
					return -1
				case 69:
					return 5
				case 116:
					return -1
				case 119:
					return -1
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 69:
					return 6
				case 116:
					return -1
				case 119:
					return -1
				case 110:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 101:
					return 6
				case 84:
					return -1
				case 87:
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
				case 84:
					return -1
				case 87:
					return -1
				case 78:
					return 7
				case 69:
					return -1
				case 116:
					return -1
				case 119:
					return -1
				case 110:
					return 7
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
				case 84:
					return -1
				case 87:
					return -1
				case 78:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 119:
					return -1
				case 110:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [bB][iI][nN][aA][rR][yY]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 66:
					return 1
				case 105:
					return -1
				case 110:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 121:
					return -1
				case 98:
					return 1
				case 73:
					return -1
				case 78:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				case 89:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 66:
					return -1
				case 105:
					return 2
				case 110:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 121:
					return -1
				case 98:
					return -1
				case 73:
					return 2
				case 78:
					return -1
				case 65:
					return -1
				case 82:
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
				case 73:
					return -1
				case 78:
					return 3
				case 65:
					return -1
				case 82:
					return -1
				case 89:
					return -1
				case 66:
					return -1
				case 105:
					return -1
				case 110:
					return 3
				case 97:
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
				case 98:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 65:
					return 4
				case 82:
					return -1
				case 89:
					return -1
				case 66:
					return -1
				case 105:
					return -1
				case 110:
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
				case 98:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 65:
					return -1
				case 82:
					return 5
				case 89:
					return -1
				case 66:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 97:
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
				case 98:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				case 89:
					return 6
				case 66:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 97:
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
				case 98:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				case 89:
					return -1
				case 66:
					return -1
				case 105:
					return -1
				case 110:
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
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [bB][oO][oO][lL][eE][aA][nN]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 98:
					return 1
				case 66:
					return 1
				case 108:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
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
				case 111:
					return 2
				case 79:
					return 2
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 108:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return 3
				case 79:
					return 3
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 108:
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
				case 76:
					return 4
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 108:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 66:
					return -1
				case 108:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 76:
					return -1
				case 101:
					return 5
				case 69:
					return 5
				case 97:
					return -1
				case 65:
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
				case 108:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return 6
				case 65:
					return 6
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
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 110:
					return 7
				case 78:
					return 7
				case 98:
					return -1
				case 66:
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
				case 108:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 76:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 78:
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
				case 69:
					return -1
				case 97:
					return -1
				case 66:
					return 1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 65:
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
				case 98:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 66:
					return -1
				case 114:
					return 2
				case 82:
					return 2
				case 101:
					return -1
				case 65:
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
				case 66:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return 3
				case 65:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 98:
					return -1
				case 69:
					return 3
				case 97:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 69:
					return -1
				case 97:
					return 4
				case 66:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 65:
					return 4
				case 107:
					return -1
				case 75:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 66:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 65:
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
				case 66:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 65:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 98:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [bB][uU][cC][kK][eE][tT]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 85:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 98:
					return 1
				case 66:
					return 1
				case 67:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return 2
				case 85:
					return 2
				case 99:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 67:
					return -1
				case 107:
					return -1
				case 75:
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
				case 99:
					return 3
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 67:
					return 3
				case 107:
					return -1
				case 75:
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
				case 67:
					return -1
				case 107:
					return 4
				case 75:
					return 4
				case 84:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 99:
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
				case 98:
					return -1
				case 66:
					return -1
				case 67:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 84:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 99:
					return -1
				case 101:
					return 5
				case 69:
					return 5
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
				case 67:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 84:
					return 6
				case 117:
					return -1
				case 85:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
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
				case 67:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 84:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 99:
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
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [bB][uU][iI][lL][dD]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 98:
					return 1
				case 117:
					return -1
				case 73:
					return -1
				case 108:
					return -1
				case 66:
					return 1
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
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 117:
					return 2
				case 73:
					return -1
				case 108:
					return -1
				case 66:
					return -1
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
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 117:
					return -1
				case 73:
					return 3
				case 108:
					return -1
				case 66:
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
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return -1
				case 117:
					return -1
				case 73:
					return -1
				case 108:
					return 4
				case 66:
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
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 66:
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
				case 98:
					return -1
				case 117:
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
				case 66:
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
				case 98:
					return -1
				case 117:
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
				case 76:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 114:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 82:
					return -1
				case 67:
					return 1
				case 117:
					return -1
				case 116:
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
				case 67:
					return -1
				case 117:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 76:
					return 2
				case 85:
					return -1
				case 101:
					return -1
				case 114:
					return -1
				case 108:
					return 2
				case 115:
					return -1
				case 83:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 76:
					return -1
				case 85:
					return 3
				case 101:
					return -1
				case 114:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 82:
					return -1
				case 67:
					return -1
				case 117:
					return 3
				case 116:
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
				case 67:
					return -1
				case 117:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 76:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 114:
					return -1
				case 108:
					return -1
				case 115:
					return 4
				case 83:
					return 4
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 76:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 114:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 82:
					return -1
				case 67:
					return -1
				case 117:
					return -1
				case 116:
					return 5
				case 84:
					return 5
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 82:
					return -1
				case 67:
					return -1
				case 117:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 69:
					return 6
				case 99:
					return -1
				case 76:
					return -1
				case 85:
					return -1
				case 101:
					return 6
				case 114:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 67:
					return -1
				case 117:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 76:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 114:
					return 7
				case 108:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 82:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 114:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 82:
					return -1
				case 67:
					return -1
				case 117:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 76:
					return -1
				case 85:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [cC][oO][lL][lL][aA][tT][eE]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 99:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 67:
					return 1
				case 111:
					return -1
				case 79:
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
				case 67:
					return -1
				case 111:
					return 2
				case 79:
					return 2
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 99:
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
				case 111:
					return -1
				case 79:
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
				case 99:
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
				case 111:
					return -1
				case 79:
					return -1
				case 108:
					return 4
				case 76:
					return 4
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
				case 111:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 97:
					return 5
				case 65:
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
				case 99:
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
				case 111:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
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
				case 67:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 99:
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
				case 111:
					return -1
				case 79:
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
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [cC][oO][lL][lL][eE][cC][tT][iI][oO][nN]
		{[]bool{false, false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 67:
					return 1
				case 111:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 99:
					return 1
				case 76:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 76:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 67:
					return -1
				case 111:
					return 2
				case 79:
					return 2
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return 3
				case 67:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 99:
					return -1
				case 76:
					return 3
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return 4
				case 67:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 99:
					return -1
				case 76:
					return 4
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 116:
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
				case 78:
					return -1
				case 108:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 101:
					return 5
				case 69:
					return 5
				case 110:
					return -1
				case 99:
					return -1
				case 76:
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
				case 99:
					return 6
				case 76:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 67:
					return 6
				case 111:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 99:
					return -1
				case 76:
					return -1
				case 84:
					return 7
				case 105:
					return -1
				case 73:
					return -1
				case 116:
					return 7
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 99:
					return -1
				case 76:
					return -1
				case 84:
					return -1
				case 105:
					return 8
				case 73:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 67:
					return -1
				case 111:
					return 9
				case 79:
					return 9
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 99:
					return -1
				case 76:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 76:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 78:
					return 10
				case 108:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 99:
					return -1
				case 76:
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
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [cC][oO][mM][mM][iI][tT]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 99:
					return 1
				case 67:
					return 1
				case 111:
					return -1
				case 79:
					return -1
				case 77:
					return -1
				case 105:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 111:
					return 2
				case 79:
					return 2
				case 77:
					return -1
				case 105:
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
				case 79:
					return -1
				case 77:
					return 3
				case 105:
					return -1
				case 84:
					return -1
				case 109:
					return 3
				case 73:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return 4
				case 73:
					return -1
				case 116:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 77:
					return 4
				case 105:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 73:
					return 5
				case 116:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 77:
					return -1
				case 105:
					return 5
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 73:
					return -1
				case 116:
					return 6
				case 99:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 77:
					return -1
				case 105:
					return -1
				case 84:
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
				case 79:
					return -1
				case 77:
					return -1
				case 105:
					return -1
				case 84:
					return -1
				case 109:
					return -1
				case 73:
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
				case 67:
					return 1
				case 111:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 99:
					return 1
				case 79:
					return -1
				case 110:
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
				case 79:
					return 2
				case 110:
					return -1
				case 69:
					return -1
				case 67:
					return -1
				case 111:
					return 2
				case 78:
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
				case 99:
					return -1
				case 79:
					return -1
				case 110:
					return 3
				case 69:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 78:
					return 3
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
				case 67:
					return -1
				case 111:
					return -1
				case 78:
					return 4
				case 101:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 79:
					return -1
				case 110:
					return 4
				case 69:
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
					return -1
				case 69:
					return 5
				case 67:
					return -1
				case 111:
					return -1
				case 78:
					return -1
				case 101:
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
				case 67:
					return 6
				case 111:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 99:
					return 6
				case 79:
					return -1
				case 110:
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
				case 111:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 116:
					return 7
				case 84:
					return 7
				case 99:
					return -1
				case 79:
					return -1
				case 110:
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
				case 111:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 79:
					return -1
				case 110:
					return -1
				case 69:
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
				case 111:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 73:
					return -1
				case 85:
					return -1
				case 79:
					return -1
				case 110:
					return -1
				case 105:
					return -1
				case 99:
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
				case 79:
					return 2
				case 110:
					return -1
				case 105:
					return -1
				case 99:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 67:
					return -1
				case 111:
					return 2
				case 78:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 73:
					return -1
				case 85:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 78:
					return 3
				case 101:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 73:
					return -1
				case 85:
					return -1
				case 79:
					return -1
				case 110:
					return 3
				case 105:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 67:
					return -1
				case 111:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 84:
					return 4
				case 73:
					return -1
				case 85:
					return -1
				case 79:
					return -1
				case 110:
					return -1
				case 105:
					return -1
				case 99:
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
				case 79:
					return -1
				case 110:
					return -1
				case 105:
					return 5
				case 99:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 73:
					return 5
				case 85:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 78:
					return 6
				case 101:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 73:
					return -1
				case 85:
					return -1
				case 79:
					return -1
				case 110:
					return 6
				case 105:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 84:
					return -1
				case 73:
					return -1
				case 85:
					return 7
				case 79:
					return -1
				case 110:
					return -1
				case 105:
					return -1
				case 99:
					return -1
				case 116:
					return -1
				case 117:
					return 7
				case 67:
					return -1
				case 111:
					return -1
				case 78:
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
				case 84:
					return -1
				case 73:
					return -1
				case 85:
					return -1
				case 79:
					return -1
				case 110:
					return -1
				case 105:
					return -1
				case 99:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 67:
					return -1
				case 111:
					return -1
				case 78:
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
				case 67:
					return -1
				case 111:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 73:
					return -1
				case 85:
					return -1
				case 79:
					return -1
				case 110:
					return -1
				case 105:
					return -1
				case 99:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [cC][oO][rR][rR][eE][lL][aA][tT][eE]
		{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 79:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 82:
					return -1
				case 108:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				case 99:
					return 1
				case 67:
					return 1
				case 114:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return 2
				case 79:
					return 2
				case 97:
					return -1
				case 116:
					return -1
				case 82:
					return -1
				case 108:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 82:
					return 3
				case 108:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 114:
					return 3
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				case 111:
					return -1
				case 79:
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
				case 99:
					return -1
				case 67:
					return -1
				case 114:
					return 4
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 82:
					return 4
				case 108:
					return -1
				case 65:
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
				case 114:
					return -1
				case 101:
					return 5
				case 69:
					return 5
				case 76:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 82:
					return -1
				case 108:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return 6
				case 65:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return 6
				case 111:
					return -1
				case 79:
					return -1
				case 97:
					return -1
				case 116:
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
				case 97:
					return 7
				case 116:
					return -1
				case 82:
					return -1
				case 108:
					return -1
				case 65:
					return 7
				case 84:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 82:
					return -1
				case 108:
					return -1
				case 65:
					return -1
				case 84:
					return 8
				case 99:
					return -1
				case 67:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 97:
					return -1
				case 116:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 99:
					return -1
				case 67:
					return -1
				case 114:
					return -1
				case 101:
					return 9
				case 69:
					return 9
				case 76:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 82:
					return -1
				case 108:
					return -1
				case 65:
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
				case 65:
					return -1
				case 84:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 82:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [cC][oO][vV][eE][rR]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 79:
					return -1
				case 118:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 99:
					return 1
				case 67:
					return 1
				case 86:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return 2
				case 79:
					return 2
				case 118:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 99:
					return -1
				case 67:
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
				case 111:
					return -1
				case 79:
					return -1
				case 118:
					return 3
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 86:
					return 3
				case 101:
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
				case 69:
					return 4
				case 114:
					return -1
				case 82:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 86:
					return -1
				case 101:
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
				case 69:
					return -1
				case 114:
					return 5
				case 82:
					return 5
				case 99:
					return -1
				case 67:
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
				case 111:
					return -1
				case 79:
					return -1
				case 118:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 86:
					return -1
				case 101:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [cC][rR][eE][aA][tT][eE]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 99:
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
				case 116:
					return -1
				case 67:
					return 1
				case 65:
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
				case 116:
					return -1
				case 67:
					return -1
				case 65:
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
				case 116:
					return -1
				case 67:
					return -1
				case 65:
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
				case 69:
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
				case 99:
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
				case 116:
					return 5
				case 67:
					return -1
				case 65:
					return -1
				case 84:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 67:
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
					return 6
				case 69:
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
				case 69:
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
				case 97:
					return -1
				case 98:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 100:
					return 1
				case 68:
					return 1
				case 65:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 66:
					return -1
				case 115:
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
				case 65:
					return 2
				case 116:
					return -1
				case 84:
					return -1
				case 66:
					return -1
				case 115:
					return -1
				case 97:
					return 2
				case 98:
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
				case 97:
					return -1
				case 98:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 65:
					return -1
				case 116:
					return 3
				case 84:
					return 3
				case 66:
					return -1
				case 115:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return 4
				case 98:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 65:
					return 4
				case 116:
					return -1
				case 84:
					return -1
				case 66:
					return -1
				case 115:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 98:
					return 5
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 65:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 66:
					return 5
				case 115:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return 6
				case 98:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 65:
					return 6
				case 116:
					return -1
				case 84:
					return -1
				case 66:
					return -1
				case 115:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 98:
					return -1
				case 83:
					return 7
				case 101:
					return -1
				case 69:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 65:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 66:
					return -1
				case 115:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 98:
					return -1
				case 83:
					return -1
				case 101:
					return 8
				case 69:
					return 8
				case 100:
					return -1
				case 68:
					return -1
				case 65:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 66:
					return -1
				case 115:
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
				case 65:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 66:
					return -1
				case 115:
					return -1
				case 97:
					return -1
				case 98:
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
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [dD][aA][tT][aA][sS][eE][tT]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 100:
					return 1
				case 97:
					return -1
				case 84:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 68:
					return 1
				case 65:
					return -1
				case 116:
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
				case 65:
					return 2
				case 116:
					return -1
				case 69:
					return -1
				case 100:
					return -1
				case 97:
					return 2
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
				case 100:
					return -1
				case 97:
					return -1
				case 84:
					return 3
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 68:
					return -1
				case 65:
					return -1
				case 116:
					return 3
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 68:
					return -1
				case 65:
					return 4
				case 116:
					return -1
				case 69:
					return -1
				case 100:
					return -1
				case 97:
					return 4
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
				case 100:
					return -1
				case 97:
					return -1
				case 84:
					return -1
				case 115:
					return 5
				case 83:
					return 5
				case 101:
					return -1
				case 68:
					return -1
				case 65:
					return -1
				case 116:
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
				case 65:
					return -1
				case 116:
					return -1
				case 69:
					return 6
				case 100:
					return -1
				case 97:
					return -1
				case 84:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
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
				case 84:
					return 7
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 68:
					return -1
				case 65:
					return -1
				case 116:
					return 7
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 68:
					return -1
				case 65:
					return -1
				case 116:
					return -1
				case 69:
					return -1
				case 100:
					return -1
				case 97:
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
				case 116:
					return -1
				case 115:
					return -1
				case 69:
					return -1
				case 111:
					return -1
				case 68:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				case 83:
					return -1
				case 82:
					return -1
				case 100:
					return 1
				case 79:
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
				case 111:
					return -1
				case 68:
					return -1
				case 97:
					return 2
				case 65:
					return 2
				case 84:
					return -1
				case 83:
					return -1
				case 82:
					return -1
				case 100:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 115:
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
					return -1
				case 65:
					return -1
				case 84:
					return 3
				case 83:
					return -1
				case 82:
					return -1
				case 100:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 116:
					return 3
				case 115:
					return -1
				case 69:
					return -1
				case 111:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 68:
					return -1
				case 97:
					return 4
				case 65:
					return 4
				case 84:
					return -1
				case 83:
					return -1
				case 82:
					return -1
				case 100:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 115:
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
				case 79:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 115:
					return 5
				case 69:
					return -1
				case 111:
					return -1
				case 68:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				case 83:
					return 5
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 116:
					return 6
				case 115:
					return -1
				case 69:
					return -1
				case 111:
					return -1
				case 68:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 84:
					return 6
				case 83:
					return -1
				case 82:
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
				case 65:
					return -1
				case 84:
					return -1
				case 83:
					return -1
				case 82:
					return -1
				case 100:
					return -1
				case 79:
					return 7
				case 114:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 115:
					return -1
				case 69:
					return -1
				case 111:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return -1
				case 115:
					return -1
				case 69:
					return -1
				case 111:
					return -1
				case 68:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				case 83:
					return -1
				case 82:
					return 8
				case 100:
					return -1
				case 79:
					return -1
				case 114:
					return 8
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 111:
					return -1
				case 68:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				case 83:
					return -1
				case 82:
					return -1
				case 100:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 101:
					return 9
				case 116:
					return -1
				case 115:
					return -1
				case 69:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 68:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				case 83:
					return -1
				case 82:
					return -1
				case 100:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 115:
					return -1
				case 69:
					return -1
				case 111:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [dD][eE][cC][lL][aA][rR][eE]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 100:
					return 1
				case 101:
					return -1
				case 67:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				case 68:
					return 1
				case 69:
					return -1
				case 99:
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
				case 100:
					return -1
				case 101:
					return 2
				case 67:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				case 68:
					return -1
				case 69:
					return 2
				case 99:
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
				case 68:
					return -1
				case 69:
					return -1
				case 99:
					return 3
				case 108:
					return -1
				case 114:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 67:
					return 3
				case 76:
					return -1
				case 97:
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
				case 101:
					return -1
				case 67:
					return -1
				case 76:
					return 4
				case 97:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 99:
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
				case 68:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 108:
					return -1
				case 114:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 67:
					return -1
				case 76:
					return -1
				case 97:
					return 5
				case 65:
					return 5
				case 82:
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
				case 67:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 82:
					return 6
				case 68:
					return -1
				case 69:
					return -1
				case 99:
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
				case 100:
					return -1
				case 101:
					return 7
				case 67:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				case 68:
					return -1
				case 69:
					return 7
				case 99:
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
				case 100:
					return -1
				case 101:
					return -1
				case 67:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 99:
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
				case 100:
					return 1
				case 82:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 67:
					return -1
				case 114:
					return -1
				case 110:
					return -1
				case 68:
					return 1
				case 69:
					return -1
				case 99:
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
				case 100:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				case 101:
					return 2
				case 67:
					return -1
				case 114:
					return -1
				case 110:
					return -1
				case 68:
					return -1
				case 69:
					return 2
				case 99:
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
				case 68:
					return -1
				case 69:
					return -1
				case 99:
					return 3
				case 109:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 100:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 67:
					return 3
				case 114:
					return -1
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 67:
					return -1
				case 114:
					return 4
				case 110:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 100:
					return -1
				case 82:
					return 4
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return 5
				case 67:
					return -1
				case 114:
					return -1
				case 110:
					return -1
				case 68:
					return -1
				case 69:
					return 5
				case 99:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 100:
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
				case 110:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 109:
					return 6
				case 77:
					return 6
				case 116:
					return -1
				case 84:
					return -1
				case 100:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 67:
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
					return 7
				case 99:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 100:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				case 101:
					return 7
				case 67:
					return -1
				case 114:
					return -1
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 67:
					return -1
				case 114:
					return -1
				case 110:
					return 8
				case 68:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 100:
					return -1
				case 82:
					return -1
				case 78:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 116:
					return 9
				case 84:
					return 9
				case 100:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 67:
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
				case 82:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 67:
					return -1
				case 114:
					return -1
				case 110:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 99:
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
				case 68:
					return 1
				case 101:
					return -1
				case 114:
					return -1
				case 100:
					return 1
				case 69:
					return -1
				case 82:
					return -1
				case 105:
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
				case 100:
					return -1
				case 69:
					return 2
				case 82:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				case 68:
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
				case 100:
					return -1
				case 69:
					return -1
				case 82:
					return 3
				case 105:
					return -1
				case 73:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 114:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 68:
					return -1
				case 101:
					return -1
				case 114:
					return -1
				case 100:
					return -1
				case 69:
					return -1
				case 82:
					return -1
				case 105:
					return 4
				case 73:
					return 4
				case 118:
					return -1
				case 86:
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
				case 82:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 118:
					return 5
				case 86:
					return 5
				case 68:
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
				case 100:
					return -1
				case 69:
					return 6
				case 82:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				case 68:
					return -1
				case 101:
					return 6
				case 114:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return 7
				case 69:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				case 68:
					return 7
				case 101:
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
				case 69:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 114:
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
				case 101:
					return -1
				case 115:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 67:
					return -1
				case 68:
					return 1
				case 83:
					return -1
				case 99:
					return -1
				case 66:
					return -1
				case 100:
					return 1
				case 69:
					return -1
				case 98:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 68:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 66:
					return -1
				case 100:
					return -1
				case 69:
					return 2
				case 98:
					return -1
				case 101:
					return 2
				case 115:
					return -1
				case 114:
					return -1
				case 82:
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
				case 67:
					return -1
				case 68:
					return -1
				case 83:
					return 3
				case 99:
					return -1
				case 66:
					return -1
				case 100:
					return -1
				case 69:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 115:
					return 3
				case 114:
					return -1
				case 82:
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
				case 100:
					return -1
				case 69:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 115:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 67:
					return 4
				case 68:
					return -1
				case 83:
					return -1
				case 99:
					return 4
				case 66:
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
				case 98:
					return -1
				case 101:
					return -1
				case 115:
					return -1
				case 114:
					return 5
				case 82:
					return 5
				case 105:
					return -1
				case 73:
					return -1
				case 67:
					return -1
				case 68:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 66:
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
				case 98:
					return -1
				case 101:
					return -1
				case 115:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return 6
				case 73:
					return 6
				case 67:
					return -1
				case 68:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 66:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 115:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 67:
					return -1
				case 68:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 66:
					return 7
				case 100:
					return -1
				case 69:
					return -1
				case 98:
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
				case 83:
					return -1
				case 99:
					return -1
				case 66:
					return -1
				case 100:
					return -1
				case 69:
					return 8
				case 98:
					return -1
				case 101:
					return 8
				case 115:
					return -1
				case 114:
					return -1
				case 82:
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
				case 101:
					return -1
				case 115:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 67:
					return -1
				case 68:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 66:
					return -1
				case 100:
					return -1
				case 69:
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
				case 105:
					return -1
				case 116:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 68:
					return 1
				case 73:
					return -1
				case 115:
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
				case 68:
					return -1
				case 73:
					return 2
				case 115:
					return -1
				case 83:
					return -1
				case 84:
					return -1
				case 100:
					return -1
				case 105:
					return 2
				case 116:
					return -1
				case 110:
					return -1
				case 78:
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
				case 105:
					return -1
				case 116:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 68:
					return -1
				case 73:
					return -1
				case 115:
					return 3
				case 83:
					return 3
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 105:
					return -1
				case 116:
					return 4
				case 110:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 68:
					return -1
				case 73:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 84:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 105:
					return 5
				case 116:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 68:
					return -1
				case 73:
					return 5
				case 115:
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
				case 100:
					return -1
				case 105:
					return -1
				case 116:
					return -1
				case 110:
					return 6
				case 78:
					return 6
				case 99:
					return -1
				case 67:
					return -1
				case 68:
					return -1
				case 73:
					return -1
				case 115:
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
				case 68:
					return -1
				case 73:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 84:
					return -1
				case 100:
					return -1
				case 105:
					return -1
				case 116:
					return -1
				case 110:
					return -1
				case 78:
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
				case 100:
					return -1
				case 105:
					return -1
				case 116:
					return 8
				case 110:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 68:
					return -1
				case 73:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 84:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 100:
					return -1
				case 105:
					return -1
				case 116:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 68:
					return -1
				case 73:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 84:
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
				case 101:
					return 1
				case 69:
					return 1
				case 76:
					return -1
				case 110:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 78:
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
					return 2
				case 109:
					return -1
				case 77:
					return -1
				case 78:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return 2
				case 110:
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
				case 78:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return 3
				case 69:
					return 3
				case 76:
					return -1
				case 110:
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
				case 76:
					return -1
				case 110:
					return -1
				case 108:
					return -1
				case 109:
					return 4
				case 77:
					return 4
				case 78:
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
					return 5
				case 69:
					return 5
				case 76:
					return -1
				case 110:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 78:
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
				case 109:
					return -1
				case 77:
					return -1
				case 78:
					return 6
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				case 110:
					return 6
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
				case 78:
					return -1
				case 116:
					return 7
				case 84:
					return 7
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				case 110:
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
				case 78:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				case 110:
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
				case 101:
					return 1
				case 99:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 116:
					return -1
				case 69:
					return 1
				case 120:
					return -1
				case 88:
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
				case 101:
					return -1
				case 99:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 116:
					return -1
				case 69:
					return -1
				case 120:
					return 2
				case 88:
					return 2
				case 67:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 69:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				case 67:
					return 3
				case 84:
					return -1
				case 101:
					return -1
				case 99:
					return 3
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
					return 4
				case 99:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 116:
					return -1
				case 69:
					return 4
				case 120:
					return -1
				case 88:
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
				case 69:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				case 67:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 99:
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
				case 69:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				case 67:
					return -1
				case 84:
					return 6
				case 101:
					return -1
				case 99:
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
				case 69:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				case 67:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 99:
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
				case 99:
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
				case 69:
					return 1
				case 120:
					return -1
				case 88:
					return -1
				case 85:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 69:
					return -1
				case 120:
					return 2
				case 88:
					return 2
				case 85:
					return -1
				case 101:
					return -1
				case 99:
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
				case 99:
					return 3
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
				case 69:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				case 85:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 99:
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
				case 69:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				case 85:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 69:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				case 85:
					return 5
				case 101:
					return -1
				case 99:
					return -1
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
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 99:
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
					return 6
				case 68:
					return 6
				case 69:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				case 85:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return 7
				case 99:
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
				case 69:
					return 7
				case 120:
					return -1
				case 88:
					return -1
				case 85:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 99:
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
				case 69:
					return -1
				case 120:
					return -1
				case 88:
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
				case 120:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 116:
					return -1
				case 101:
					return 1
				case 69:
					return 1
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
					return -1
				case 69:
					return -1
				case 88:
					return 2
				case 84:
					return -1
				case 120:
					return 2
				case 99:
					return -1
				case 67:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 116:
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
				case 88:
					return -1
				case 84:
					return -1
				case 120:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 117:
					return -1
				case 85:
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
				case 88:
					return -1
				case 84:
					return -1
				case 120:
					return -1
				case 99:
					return 4
				case 67:
					return 4
				case 117:
					return -1
				case 85:
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
				case 88:
					return -1
				case 84:
					return -1
				case 120:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 117:
					return 5
				case 85:
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
				case 88:
					return -1
				case 84:
					return 6
				case 120:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 117:
					return -1
				case 85:
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
				case 84:
					return -1
				case 120:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 117:
					return -1
				case 85:
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
				case 99:
					return -1
				case 67:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 116:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 88:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [eE][xX][iI][sS][tT][sS]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 120:
					return -1
				case 88:
					return -1
				case 73:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 101:
					return 1
				case 69:
					return 1
				case 105:
					return -1
				case 115:
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
				case 105:
					return -1
				case 115:
					return -1
				case 84:
					return -1
				case 120:
					return 2
				case 88:
					return 2
				case 73:
					return -1
				case 83:
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
				case 88:
					return -1
				case 73:
					return 3
				case 83:
					return -1
				case 116:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 105:
					return 3
				case 115:
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
				case 105:
					return -1
				case 115:
					return 4
				case 84:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				case 73:
					return -1
				case 83:
					return 4
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
				case 105:
					return -1
				case 115:
					return -1
				case 84:
					return 5
				case 120:
					return -1
				case 88:
					return -1
				case 73:
					return -1
				case 83:
					return -1
				case 116:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 120:
					return -1
				case 88:
					return -1
				case 73:
					return -1
				case 83:
					return 6
				case 116:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 105:
					return -1
				case 115:
					return 6
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
				case 105:
					return -1
				case 115:
					return -1
				case 84:
					return -1
				case 120:
					return -1
				case 88:
					return -1
				case 73:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [eE][xX][pP][lL][aA][iI][nN]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 101:
					return 1
				case 88:
					return -1
				case 97:
					return -1
				case 110:
					return -1
				case 69:
					return 1
				case 112:
					return -1
				case 108:
					return -1
				case 105:
					return -1
				case 78:
					return -1
				case 120:
					return -1
				case 80:
					return -1
				case 76:
					return -1
				case 65:
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
				case 112:
					return -1
				case 108:
					return -1
				case 105:
					return -1
				case 78:
					return -1
				case 120:
					return 2
				case 80:
					return -1
				case 76:
					return -1
				case 65:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 88:
					return 2
				case 97:
					return -1
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 120:
					return -1
				case 80:
					return 3
				case 76:
					return -1
				case 65:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 88:
					return -1
				case 97:
					return -1
				case 110:
					return -1
				case 69:
					return -1
				case 112:
					return 3
				case 108:
					return -1
				case 105:
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
				case 80:
					return -1
				case 76:
					return 4
				case 65:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 88:
					return -1
				case 97:
					return -1
				case 110:
					return -1
				case 69:
					return -1
				case 112:
					return -1
				case 108:
					return 4
				case 105:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 78:
					return -1
				case 120:
					return -1
				case 80:
					return -1
				case 76:
					return -1
				case 65:
					return 5
				case 73:
					return -1
				case 101:
					return -1
				case 88:
					return -1
				case 97:
					return 5
				case 110:
					return -1
				case 69:
					return -1
				case 112:
					return -1
				case 108:
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
				case 88:
					return -1
				case 97:
					return -1
				case 110:
					return -1
				case 69:
					return -1
				case 112:
					return -1
				case 108:
					return -1
				case 105:
					return 6
				case 78:
					return -1
				case 120:
					return -1
				case 80:
					return -1
				case 76:
					return -1
				case 65:
					return -1
				case 73:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 69:
					return -1
				case 112:
					return -1
				case 108:
					return -1
				case 105:
					return -1
				case 78:
					return 7
				case 120:
					return -1
				case 80:
					return -1
				case 76:
					return -1
				case 65:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 88:
					return -1
				case 97:
					return -1
				case 110:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 120:
					return -1
				case 80:
					return -1
				case 76:
					return -1
				case 65:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 88:
					return -1
				case 97:
					return -1
				case 110:
					return -1
				case 69:
					return -1
				case 112:
					return -1
				case 108:
					return -1
				case 105:
					return -1
				case 78:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [fF][aA][lL][sS][eE]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 70:
					return 1
				case 97:
					return -1
				case 108:
					return -1
				case 83:
					return -1
				case 69:
					return -1
				case 102:
					return 1
				case 65:
					return -1
				case 76:
					return -1
				case 115:
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
				case 65:
					return 2
				case 76:
					return -1
				case 115:
					return -1
				case 101:
					return -1
				case 70:
					return -1
				case 97:
					return 2
				case 108:
					return -1
				case 83:
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
					return 3
				case 83:
					return -1
				case 69:
					return -1
				case 102:
					return -1
				case 65:
					return -1
				case 76:
					return 3
				case 115:
					return -1
				case 101:
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
				case 83:
					return 4
				case 69:
					return -1
				case 102:
					return -1
				case 65:
					return -1
				case 76:
					return -1
				case 115:
					return 4
				case 101:
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
				case 76:
					return -1
				case 115:
					return -1
				case 101:
					return 5
				case 70:
					return -1
				case 97:
					return -1
				case 108:
					return -1
				case 83:
					return -1
				case 69:
					return 5
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
				case 83:
					return -1
				case 69:
					return -1
				case 102:
					return -1
				case 65:
					return -1
				case 76:
					return -1
				case 115:
					return -1
				case 101:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [fF][eE][tT][cC][hH]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 70:
					return 1
				case 101:
					return -1
				case 116:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 104:
					return -1
				case 102:
					return 1
				case 69:
					return -1
				case 84:
					return -1
				case 72:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 69:
					return 2
				case 84:
					return -1
				case 72:
					return -1
				case 70:
					return -1
				case 101:
					return 2
				case 116:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 104:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 69:
					return -1
				case 84:
					return 3
				case 72:
					return -1
				case 70:
					return -1
				case 101:
					return -1
				case 116:
					return 3
				case 99:
					return -1
				case 67:
					return -1
				case 104:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 72:
					return -1
				case 70:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 99:
					return 4
				case 67:
					return 4
				case 104:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 70:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 104:
					return 5
				case 102:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 72:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 72:
					return -1
				case 70:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 104:
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
				case 73:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 70:
					return 1
				case 105:
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
				case 73:
					return 2
				case 114:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 70:
					return -1
				case 105:
					return 2
				case 82:
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
				case 82:
					return 3
				case 102:
					return -1
				case 73:
					return -1
				case 114:
					return 3
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
				case 102:
					return -1
				case 73:
					return -1
				case 114:
					return -1
				case 115:
					return 4
				case 83:
					return 4
				case 116:
					return -1
				case 84:
					return -1
				case 70:
					return -1
				case 105:
					return -1
				case 82:
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
				case 82:
					return -1
				case 102:
					return -1
				case 73:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 83:
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
				case 70:
					return -1
				case 105:
					return -1
				case 82:
					return -1
				case 102:
					return -1
				case 73:
					return -1
				case 114:
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
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [fF][lL][aA][tT][tT][eE][nN]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 102:
					return 1
				case 70:
					return 1
				case 108:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 76:
					return -1
				case 65:
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
				case 76:
					return 2
				case 65:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				case 108:
					return 2
				case 97:
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
				case 102:
					return -1
				case 70:
					return -1
				case 108:
					return -1
				case 97:
					return 3
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 76:
					return -1
				case 65:
					return 3
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
				case 76:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				case 108:
					return -1
				case 97:
					return -1
				case 116:
					return 4
				case 84:
					return 4
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 76:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				case 108:
					return -1
				case 97:
					return -1
				case 116:
					return 5
				case 84:
					return 5
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 76:
					return -1
				case 65:
					return -1
				case 69:
					return 6
				case 110:
					return -1
				case 78:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				case 108:
					return -1
				case 97:
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
				case 76:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				case 110:
					return 7
				case 78:
					return 7
				case 102:
					return -1
				case 70:
					return -1
				case 108:
					return -1
				case 97:
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
				case 102:
					return -1
				case 70:
					return -1
				case 108:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 76:
					return -1
				case 65:
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
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 102:
					return 1
				case 70:
					return 1
				case 111:
					return -1
				case 79:
					return -1
				case 99:
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
					return 2
				case 79:
					return 2
				case 99:
					return -1
				case 67:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return 3
				case 82:
					return 3
				case 101:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 99:
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
				case 99:
					return 4
				case 67:
					return 4
				case 69:
					return -1
				case 114:
					return -1
				case 82:
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
				case 82:
					return -1
				case 101:
					return 5
				case 102:
					return -1
				case 70:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 69:
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
				case 102:
					return -1
				case 70:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 99:
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
				case 85:
					return -1
				case 78:
					return -1
				case 67:
					return -1
				case 79:
					return -1
				case 70:
					return 1
				case 110:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 102:
					return 1
				case 117:
					return -1
				case 99:
					return -1
				case 111:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 70:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 102:
					return -1
				case 117:
					return 2
				case 99:
					return -1
				case 111:
					return -1
				case 85:
					return 2
				case 78:
					return -1
				case 67:
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
				case 110:
					return 3
				case 116:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 102:
					return -1
				case 117:
					return -1
				case 99:
					return -1
				case 111:
					return -1
				case 85:
					return -1
				case 78:
					return 3
				case 67:
					return -1
				case 79:
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
				case 99:
					return 4
				case 111:
					return -1
				case 85:
					return -1
				case 78:
					return -1
				case 67:
					return 4
				case 79:
					return -1
				case 70:
					return -1
				case 110:
					return -1
				case 116:
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
				case 102:
					return -1
				case 117:
					return -1
				case 99:
					return -1
				case 111:
					return -1
				case 85:
					return -1
				case 78:
					return -1
				case 67:
					return -1
				case 79:
					return -1
				case 70:
					return -1
				case 110:
					return -1
				case 116:
					return 5
				case 84:
					return 5
				case 105:
					return -1
				case 73:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 85:
					return -1
				case 78:
					return -1
				case 67:
					return -1
				case 79:
					return -1
				case 70:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 105:
					return 6
				case 73:
					return 6
				case 102:
					return -1
				case 117:
					return -1
				case 99:
					return -1
				case 111:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 70:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 102:
					return -1
				case 117:
					return -1
				case 99:
					return -1
				case 111:
					return 7
				case 85:
					return -1
				case 78:
					return -1
				case 67:
					return -1
				case 79:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 102:
					return -1
				case 117:
					return -1
				case 99:
					return -1
				case 111:
					return -1
				case 85:
					return -1
				case 78:
					return 8
				case 67:
					return -1
				case 79:
					return -1
				case 70:
					return -1
				case 110:
					return 8
				case 116:
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
				case 102:
					return -1
				case 117:
					return -1
				case 99:
					return -1
				case 111:
					return -1
				case 85:
					return -1
				case 78:
					return -1
				case 67:
					return -1
				case 79:
					return -1
				case 70:
					return -1
				case 110:
					return -1
				case 116:
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
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [gG][rR][aA][nN][tT]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 71:
					return 1
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 103:
					return 1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 103:
					return -1
				case 78:
					return -1
				case 71:
					return -1
				case 114:
					return 2
				case 82:
					return 2
				case 97:
					return -1
				case 65:
					return -1
				case 110:
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
				case 103:
					return -1
				case 78:
					return -1
				case 71:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return 3
				case 65:
					return 3
				case 110:
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
				case 71:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 110:
					return 4
				case 116:
					return -1
				case 84:
					return -1
				case 103:
					return -1
				case 78:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 71:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 116:
					return 5
				case 84:
					return 5
				case 103:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 103:
					return -1
				case 78:
					return -1
				case 71:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 110:
					return -1
				case 116:
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
				case 71:
					return 1
				case 82:
					return -1
				case 79:
					return -1
				case 85:
					return -1
				case 80:
					return -1
				case 103:
					return 1
				case 114:
					return -1
				case 111:
					return -1
				case 117:
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
				case 82:
					return 2
				case 79:
					return -1
				case 85:
					return -1
				case 80:
					return -1
				case 103:
					return -1
				case 114:
					return 2
				case 111:
					return -1
				case 117:
					return -1
				case 112:
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
				case 111:
					return 3
				case 117:
					return -1
				case 112:
					return -1
				case 71:
					return -1
				case 82:
					return -1
				case 79:
					return 3
				case 85:
					return -1
				case 80:
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
				case 111:
					return -1
				case 117:
					return 4
				case 112:
					return -1
				case 71:
					return -1
				case 82:
					return -1
				case 79:
					return -1
				case 85:
					return 4
				case 80:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 71:
					return -1
				case 82:
					return -1
				case 79:
					return -1
				case 85:
					return -1
				case 80:
					return 5
				case 103:
					return -1
				case 114:
					return -1
				case 111:
					return -1
				case 117:
					return -1
				case 112:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 71:
					return -1
				case 82:
					return -1
				case 79:
					return -1
				case 85:
					return -1
				case 80:
					return -1
				case 103:
					return -1
				case 114:
					return -1
				case 111:
					return -1
				case 117:
					return -1
				case 112:
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
				case 72:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				case 78:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 104:
					return 1
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 104:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 72:
					return -1
				case 97:
					return 2
				case 65:
					return 2
				case 118:
					return -1
				case 86:
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
				case 72:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 118:
					return 3
				case 86:
					return 3
				case 78:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 104:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 104:
					return -1
				case 105:
					return 4
				case 73:
					return 4
				case 110:
					return -1
				case 72:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 118:
					return -1
				case 86:
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
				case 72:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				case 78:
					return 5
				case 103:
					return -1
				case 71:
					return -1
				case 104:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 72:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 118:
					return -1
				case 86:
					return -1
				case 78:
					return -1
				case 103:
					return 6
				case 71:
					return 6
				case 104:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 104:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 72:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 118:
					return -1
				case 86:
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
				case 71:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 105:
					return 1
				case 73:
					return 1
				case 103:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 111:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 71:
					return 2
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 103:
					return 2
				case 110:
					return -1
				case 78:
					return -1
				case 111:
					return -1
				case 69:
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
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 103:
					return -1
				case 110:
					return 3
				case 78:
					return 3
				case 111:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 71:
					return -1
				case 79:
					return 4
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 103:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 111:
					return 4
				case 69:
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
				case 114:
					return 5
				case 82:
					return 5
				case 101:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 103:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 111:
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
				case 103:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 111:
					return -1
				case 69:
					return 6
				case 71:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
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
				case 103:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 111:
					return -1
				case 69:
					return -1
				case 71:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
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
				case 67:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 105:
					return 1
				case 110:
					return -1
				case 99:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 73:
					return 1
				case 78:
					return -1
				case 108:
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
				case 73:
					return -1
				case 78:
					return 2
				case 108:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 67:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 105:
					return -1
				case 110:
					return 2
				case 99:
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
				case 73:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 67:
					return 3
				case 76:
					return -1
				case 117:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 99:
					return 3
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
				case 76:
					return 4
				case 117:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 99:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 108:
					return 4
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
				case 67:
					return -1
				case 76:
					return -1
				case 117:
					return 5
				case 105:
					return -1
				case 110:
					return -1
				case 99:
					return -1
				case 85:
					return 5
				case 101:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 108:
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
				case 73:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 100:
					return 6
				case 68:
					return 6
				case 69:
					return -1
				case 67:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 99:
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
				case 73:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return 7
				case 67:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 99:
					return -1
				case 85:
					return -1
				case 101:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 73:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 67:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 99:
					return -1
				case 85:
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
				case 105:
					return 1
				case 99:
					return -1
				case 101:
					return -1
				case 109:
					return -1
				case 116:
					return -1
				case 73:
					return 1
				case 110:
					return -1
				case 77:
					return -1
				case 67:
					return -1
				case 114:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 78:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 67:
					return -1
				case 114:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 78:
					return 2
				case 82:
					return -1
				case 105:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 109:
					return -1
				case 116:
					return -1
				case 73:
					return -1
				case 110:
					return 2
				case 77:
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
				case 77:
					return -1
				case 67:
					return 3
				case 114:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 78:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 99:
					return 3
				case 101:
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
				case 78:
					return -1
				case 82:
					return 4
				case 105:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 109:
					return -1
				case 116:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 77:
					return -1
				case 67:
					return -1
				case 114:
					return 4
				case 69:
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
				case 82:
					return -1
				case 105:
					return -1
				case 99:
					return -1
				case 101:
					return 5
				case 109:
					return -1
				case 116:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 77:
					return -1
				case 67:
					return -1
				case 114:
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
				case 78:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 109:
					return 6
				case 116:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 77:
					return 6
				case 67:
					return -1
				case 114:
					return -1
				case 69:
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
				case 82:
					return -1
				case 105:
					return -1
				case 99:
					return -1
				case 101:
					return 7
				case 109:
					return -1
				case 116:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 77:
					return -1
				case 67:
					return -1
				case 114:
					return -1
				case 69:
					return 7
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 109:
					return -1
				case 116:
					return -1
				case 73:
					return -1
				case 110:
					return 8
				case 77:
					return -1
				case 67:
					return -1
				case 114:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 78:
					return 8
				case 82:
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
				case 77:
					return -1
				case 67:
					return -1
				case 114:
					return -1
				case 69:
					return -1
				case 84:
					return 9
				case 78:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 109:
					return -1
				case 116:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 109:
					return -1
				case 116:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 77:
					return -1
				case 67:
					return -1
				case 114:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 78:
					return -1
				case 82:
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
				case 110:
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
				case 78:
					return -1
				case 100:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 78:
					return 2
				case 100:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return 2
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
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 78:
					return -1
				case 100:
					return 3
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 68:
					return 3
				case 101:
					return -1
				case 69:
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
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 68:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				case 120:
					return -1
				case 88:
					return -1
				case 78:
					return -1
				case 100:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 78:
					return -1
				case 100:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 69:
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
				case 78:
					return -1
				case 100:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 110:
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
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [iI][nN][fF][eE][rR]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 105:
					return 1
				case 110:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 73:
					return 1
				case 78:
					return -1
				case 102:
					return -1
				case 70:
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
				case 78:
					return 2
				case 102:
					return -1
				case 70:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 110:
					return 2
				case 101:
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
				case 73:
					return -1
				case 78:
					return -1
				case 102:
					return 3
				case 70:
					return 3
				case 82:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 101:
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
				case 73:
					return -1
				case 78:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 101:
					return 4
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
				case 78:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				case 82:
					return 5
				case 105:
					return -1
				case 110:
					return -1
				case 101:
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
				case 73:
					return -1
				case 78:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 101:
					return -1
				case 69:
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
				case 105:
					return 1
				case 110:
					return -1
				case 83:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 116:
					return -1
				case 73:
					return 1
				case 78:
					return -1
				case 115:
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
				case 105:
					return -1
				case 110:
					return 2
				case 83:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 116:
					return -1
				case 73:
					return -1
				case 78:
					return 2
				case 115:
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
				case 105:
					return -1
				case 110:
					return -1
				case 83:
					return 3
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 116:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 115:
					return 3
				case 101:
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
				case 78:
					return -1
				case 115:
					return -1
				case 101:
					return 4
				case 84:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 83:
					return -1
				case 69:
					return 4
				case 114:
					return -1
				case 82:
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
				case 115:
					return -1
				case 101:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 83:
					return -1
				case 69:
					return -1
				case 114:
					return 5
				case 82:
					return 5
				case 116:
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
				case 83:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 116:
					return 6
				case 73:
					return -1
				case 78:
					return -1
				case 115:
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
				case 105:
					return -1
				case 110:
					return -1
				case 83:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 116:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 115:
					return -1
				case 101:
					return -1
				case 84:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [iI][nN][tT][eE][rR][sS][eE][cC][tT]
		{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 105:
					return 1
				case 110:
					return -1
				case 69:
					return -1
				case 82:
					return -1
				case 67:
					return -1
				case 78:
					return -1
				case 84:
					return -1
				case 73:
					return 1
				case 101:
					return -1
				case 116:
					return -1
				case 114:
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
				case 105:
					return -1
				case 110:
					return 2
				case 69:
					return -1
				case 82:
					return -1
				case 67:
					return -1
				case 78:
					return 2
				case 84:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 114:
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
				case 116:
					return 3
				case 114:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 69:
					return -1
				case 82:
					return -1
				case 67:
					return -1
				case 78:
					return -1
				case 84:
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
				case 78:
					return -1
				case 84:
					return -1
				case 73:
					return -1
				case 101:
					return 4
				case 116:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 69:
					return 4
				case 82:
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
				case 84:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 114:
					return 5
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 69:
					return -1
				case 82:
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
				case 84:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 115:
					return 6
				case 83:
					return 6
				case 99:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 69:
					return -1
				case 82:
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
				case 101:
					return 7
				case 116:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 69:
					return 7
				case 82:
					return -1
				case 67:
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
				case 116:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return 8
				case 105:
					return -1
				case 110:
					return -1
				case 69:
					return -1
				case 82:
					return -1
				case 67:
					return 8
				case 78:
					return -1
				case 84:
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
				case 78:
					return -1
				case 84:
					return 9
				case 73:
					return -1
				case 101:
					return -1
				case 116:
					return 9
				case 114:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 69:
					return -1
				case 82:
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
				case 84:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 99:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 69:
					return -1
				case 82:
					return -1
				case 67:
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
				case 65:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 89:
					return -1
				case 83:
					return -1
				case 80:
					return -1
				case 97:
					return -1
				case 107:
					return 1
				case 75:
					return 1
				case 101:
					return -1
				case 115:
					return -1
				case 112:
					return -1
				case 69:
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
				case 121:
					return -1
				case 65:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 89:
					return -1
				case 83:
					return -1
				case 80:
					return -1
				case 97:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return 2
				case 115:
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
				case 99:
					return -1
				case 67:
					return -1
				case 89:
					return 3
				case 83:
					return -1
				case 80:
					return -1
				case 97:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return -1
				case 115:
					return -1
				case 112:
					return -1
				case 69:
					return -1
				case 121:
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
				case 115:
					return 4
				case 112:
					return -1
				case 69:
					return -1
				case 121:
					return -1
				case 65:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 89:
					return -1
				case 83:
					return 4
				case 80:
					return -1
				case 97:
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
				case 115:
					return -1
				case 112:
					return 5
				case 69:
					return -1
				case 121:
					return -1
				case 65:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 89:
					return -1
				case 83:
					return -1
				case 80:
					return 5
				case 97:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 69:
					return -1
				case 121:
					return -1
				case 65:
					return 6
				case 99:
					return -1
				case 67:
					return -1
				case 89:
					return -1
				case 83:
					return -1
				case 80:
					return -1
				case 97:
					return 6
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return -1
				case 115:
					return -1
				case 112:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 89:
					return -1
				case 83:
					return -1
				case 80:
					return -1
				case 97:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return -1
				case 115:
					return -1
				case 112:
					return -1
				case 69:
					return -1
				case 121:
					return -1
				case 65:
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
				case 65:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 89:
					return -1
				case 83:
					return -1
				case 80:
					return -1
				case 97:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 101:
					return 8
				case 115:
					return -1
				case 112:
					return -1
				case 69:
					return 8
				case 121:
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
				case 115:
					return -1
				case 112:
					return -1
				case 69:
					return -1
				case 121:
					return -1
				case 65:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 89:
					return -1
				case 83:
					return -1
				case 80:
					return -1
				case 97:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [kK][nN][oO][wW][nN]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 107:
					return 1
				case 75:
					return 1
				case 110:
					return -1
				case 78:
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
				case 107:
					return -1
				case 75:
					return -1
				case 110:
					return 2
				case 78:
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
				case 107:
					return -1
				case 75:
					return -1
				case 110:
					return -1
				case 78:
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
				case 107:
					return -1
				case 75:
					return -1
				case 110:
					return -1
				case 78:
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
				case 107:
					return -1
				case 75:
					return -1
				case 110:
					return 5
				case 78:
					return 5
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
				case 107:
					return -1
				case 75:
					return -1
				case 110:
					return -1
				case 78:
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
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

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
				case 108:
					return 1
				case 101:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 76:
					return 1
				case 116:
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
				case 108:
					return -1
				case 101:
					return 2
				case 69:
					return 2
				case 84:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 76:
					return -1
				case 116:
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
				case 108:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 84:
					return 3
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 76:
					return -1
				case 116:
					return 3
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
				case 116:
					return 4
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 84:
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
				case 101:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 76:
					return -1
				case 116:
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
				case 116:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return 6
				case 108:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 84:
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
				case 76:
					return -1
				case 116:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 108:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 110:
					return -1
				case 103:
					return 7
				case 71:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 76:
					return -1
				case 116:
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
				case 112:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 109:
					return 1
				case 77:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 80:
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
				case 112:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 97:
					return 2
				case 65:
					return 2
				case 80:
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
				case 112:
					return 3
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 80:
					return 3
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
				case 109:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 80:
					return 4
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 112:
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
				case 109:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 80:
					return -1
				case 105:
					return 5
				case 73:
					return 5
				case 78:
					return -1
				case 112:
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
				case 112:
					return -1
				case 110:
					return 6
				case 103:
					return -1
				case 71:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 80:
					return -1
				case 105:
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
				case 109:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 80:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 112:
					return -1
				case 110:
					return -1
				case 103:
					return 7
				case 71:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 80:
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
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [mM][aA][tT][cC][hH][eE][dD]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 109:
					return 1
				case 77:
					return 1
				case 97:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 100:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 69:
					return -1
				case 65:
					return -1
				case 68:
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
				case 116:
					return -1
				case 84:
					return -1
				case 100:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 69:
					return -1
				case 65:
					return 2
				case 68:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 69:
					return -1
				case 65:
					return -1
				case 68:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 116:
					return 3
				case 84:
					return 3
				case 100:
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
				case 109:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 100:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 99:
					return 4
				case 67:
					return 4
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
				case 69:
					return -1
				case 65:
					return -1
				case 68:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 100:
					return -1
				case 104:
					return 5
				case 72:
					return 5
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return 6
				case 99:
					return -1
				case 67:
					return -1
				case 69:
					return 6
				case 65:
					return -1
				case 68:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 100:
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
				case 69:
					return -1
				case 65:
					return -1
				case 68:
					return 7
				case 109:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 100:
					return 7
				case 104:
					return -1
				case 72:
					return -1
				case 101:
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
				case 116:
					return -1
				case 84:
					return -1
				case 100:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 69:
					return -1
				case 65:
					return -1
				case 68:
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
				case 97:
					return -1
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 90:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 77:
					return 1
				case 116:
					return -1
				case 114:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 82:
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
				case 65:
					return 2
				case 69:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 82:
					return -1
				case 122:
					return -1
				case 68:
					return -1
				case 109:
					return -1
				case 97:
					return 2
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 90:
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
				case 122:
					return -1
				case 68:
					return -1
				case 109:
					return -1
				case 97:
					return -1
				case 84:
					return 3
				case 108:
					return -1
				case 76:
					return -1
				case 90:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 77:
					return -1
				case 116:
					return 3
				case 114:
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
				case 97:
					return -1
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 90:
					return -1
				case 65:
					return -1
				case 69:
					return 4
				case 105:
					return -1
				case 73:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 100:
					return -1
				case 101:
					return 4
				case 82:
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
				case 65:
					return -1
				case 69:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 114:
					return 5
				case 100:
					return -1
				case 101:
					return -1
				case 82:
					return 5
				case 122:
					return -1
				case 68:
					return -1
				case 109:
					return -1
				case 97:
					return -1
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 90:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 77:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 82:
					return -1
				case 122:
					return -1
				case 68:
					return -1
				case 109:
					return -1
				case 97:
					return -1
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 90:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				case 105:
					return 6
				case 73:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 77:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 82:
					return -1
				case 122:
					return -1
				case 68:
					return -1
				case 109:
					return -1
				case 97:
					return 7
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 90:
					return -1
				case 65:
					return 7
				case 69:
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
				case 77:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 82:
					return -1
				case 122:
					return -1
				case 68:
					return -1
				case 109:
					return -1
				case 97:
					return -1
				case 84:
					return -1
				case 108:
					return 8
				case 76:
					return 8
				case 90:
					return -1
				case 65:
					return -1
				case 69:
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
				case 65:
					return -1
				case 69:
					return -1
				case 105:
					return 9
				case 73:
					return 9
				case 77:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 82:
					return -1
				case 122:
					return -1
				case 68:
					return -1
				case 109:
					return -1
				case 97:
					return -1
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 90:
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
				case 122:
					return 10
				case 68:
					return -1
				case 109:
					return -1
				case 97:
					return -1
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 90:
					return 10
				case 65:
					return -1
				case 69:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 100:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return 11
				case 82:
					return -1
				case 122:
					return -1
				case 68:
					return -1
				case 109:
					return -1
				case 97:
					return -1
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 90:
					return -1
				case 65:
					return -1
				case 69:
					return 11
				case 105:
					return -1
				case 73:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 100:
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
				case 122:
					return -1
				case 68:
					return 12
				case 109:
					return -1
				case 97:
					return -1
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 90:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 100:
					return 12
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 97:
					return -1
				case 84:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 90:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 77:
					return -1
				case 116:
					return -1
				case 114:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 82:
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
				case 77:
					return 1
				case 105:
					return -1
				case 117:
					return -1
				case 83:
					return -1
				case 109:
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
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 77:
					return -1
				case 105:
					return 2
				case 117:
					return -1
				case 83:
					return -1
				case 109:
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
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
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
				case 77:
					return -1
				case 105:
					return -1
				case 117:
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
				case 105:
					return -1
				case 117:
					return 4
				case 83:
					return -1
				case 109:
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
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 77:
					return -1
				case 105:
					return -1
				case 117:
					return -1
				case 83:
					return 5
				case 109:
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
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 77:
					return -1
				case 105:
					return -1
				case 117:
					return -1
				case 83:
					return -1
				case 109:
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
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [mM][iI][sS][sS][iI][nN][gG]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 77:
					return 1
				case 105:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 109:
					return 1
				case 73:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 77:
					return -1
				case 105:
					return 2
				case 115:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 109:
					return -1
				case 73:
					return 2
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 77:
					return -1
				case 105:
					return -1
				case 115:
					return 3
				case 83:
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
				case 77:
					return -1
				case 105:
					return -1
				case 115:
					return 4
				case 83:
					return 4
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 109:
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
				case 77:
					return -1
				case 105:
					return 5
				case 115:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 109:
					return -1
				case 73:
					return 5
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 77:
					return -1
				case 105:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 110:
					return 6
				case 103:
					return -1
				case 71:
					return -1
				case 109:
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
				case 109:
					return -1
				case 73:
					return -1
				case 78:
					return -1
				case 77:
					return -1
				case 105:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				case 103:
					return 7
				case 71:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 77:
					return -1
				case 105:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 109:
					return -1
				case 73:
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
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 78:
					return 1
				case 65:
					return -1
				case 115:
					return -1
				case 112:
					return -1
				case 67:
					return -1
				case 109:
					return -1
				case 110:
					return 1
				case 97:
					return -1
				case 83:
					return -1
				case 80:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 78:
					return -1
				case 65:
					return 2
				case 115:
					return -1
				case 112:
					return -1
				case 67:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 97:
					return 2
				case 83:
					return -1
				case 80:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 77:
					return 3
				case 101:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 78:
					return -1
				case 65:
					return -1
				case 115:
					return -1
				case 112:
					return -1
				case 67:
					return -1
				case 109:
					return 3
				case 110:
					return -1
				case 97:
					return -1
				case 83:
					return -1
				case 80:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 77:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				case 99:
					return -1
				case 78:
					return -1
				case 65:
					return -1
				case 115:
					return -1
				case 112:
					return -1
				case 67:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 97:
					return -1
				case 83:
					return -1
				case 80:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 97:
					return -1
				case 83:
					return 5
				case 80:
					return -1
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 78:
					return -1
				case 65:
					return -1
				case 115:
					return 5
				case 112:
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
				case 65:
					return -1
				case 115:
					return -1
				case 112:
					return 6
				case 67:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 97:
					return -1
				case 83:
					return -1
				case 80:
					return 6
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 109:
					return -1
				case 110:
					return -1
				case 97:
					return 7
				case 83:
					return -1
				case 80:
					return -1
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 99:
					return -1
				case 78:
					return -1
				case 65:
					return 7
				case 115:
					return -1
				case 112:
					return -1
				case 67:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 77:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 99:
					return 8
				case 78:
					return -1
				case 65:
					return -1
				case 115:
					return -1
				case 112:
					return -1
				case 67:
					return 8
				case 109:
					return -1
				case 110:
					return -1
				case 97:
					return -1
				case 83:
					return -1
				case 80:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 110:
					return -1
				case 97:
					return -1
				case 83:
					return -1
				case 80:
					return -1
				case 77:
					return -1
				case 101:
					return 9
				case 69:
					return 9
				case 99:
					return -1
				case 78:
					return -1
				case 65:
					return -1
				case 115:
					return -1
				case 112:
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
				case 65:
					return -1
				case 115:
					return -1
				case 112:
					return -1
				case 67:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 97:
					return -1
				case 83:
					return -1
				case 80:
					return -1
				case 77:
					return -1
				case 101:
					return -1
				case 69:
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
				case 78:
					return 1
				case 109:
					return -1
				case 77:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 110:
					return 1
				case 117:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 78:
					return 2
				case 109:
					return -1
				case 77:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 110:
					return -1
				case 117:
					return 2
				case 98:
					return -1
				case 101:
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
				case 117:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				case 109:
					return 3
				case 77:
					return 3
				case 66:
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
				case 110:
					return -1
				case 117:
					return -1
				case 98:
					return 4
				case 101:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 66:
					return 4
				case 69:
					return -1
				case 114:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 78:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 66:
					return -1
				case 69:
					return 5
				case 114:
					return -1
				case 110:
					return -1
				case 117:
					return -1
				case 98:
					return -1
				case 101:
					return 5
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 78:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 114:
					return 6
				case 110:
					return -1
				case 117:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 82:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 78:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 110:
					return -1
				case 117:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 82:
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
				case 106:
					return -1
				case 69:
					return -1
				case 111:
					return 1
				case 79:
					return 1
				case 98:
					return -1
				case 74:
					return -1
				case 101:
					return -1
				case 99:
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
				case 111:
					return -1
				case 79:
					return -1
				case 98:
					return 2
				case 74:
					return -1
				case 101:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 66:
					return 2
				case 106:
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
				case 98:
					return -1
				case 74:
					return 3
				case 101:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 66:
					return -1
				case 106:
					return 3
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 66:
					return -1
				case 106:
					return -1
				case 69:
					return 4
				case 111:
					return -1
				case 79:
					return -1
				case 98:
					return -1
				case 74:
					return -1
				case 101:
					return 4
				case 99:
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
				case 66:
					return -1
				case 106:
					return -1
				case 69:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 98:
					return -1
				case 74:
					return -1
				case 101:
					return -1
				case 99:
					return 5
				case 67:
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
				case 66:
					return -1
				case 106:
					return -1
				case 69:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 98:
					return -1
				case 74:
					return -1
				case 101:
					return -1
				case 99:
					return -1
				case 67:
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
				case 111:
					return -1
				case 79:
					return -1
				case 98:
					return -1
				case 74:
					return -1
				case 101:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 66:
					return -1
				case 106:
					return -1
				case 69:
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
				case 102:
					return -1
				case 70:
					return -1
				case 115:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 83:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 83:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 102:
					return 2
				case 70:
					return 2
				case 115:
					return -1
				case 101:
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
				case 69:
					return -1
				case 84:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 102:
					return 3
				case 70:
					return 3
				case 115:
					return -1
				case 101:
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
				case 79:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				case 115:
					return 4
				case 101:
					return -1
				case 116:
					return -1
				case 83:
					return 4
				case 69:
					return -1
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 83:
					return -1
				case 69:
					return 5
				case 84:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				case 115:
					return -1
				case 101:
					return 5
				case 116:
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
				case 102:
					return -1
				case 70:
					return -1
				case 115:
					return -1
				case 101:
					return -1
				case 116:
					return 6
				case 83:
					return -1
				case 69:
					return -1
				case 84:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 83:
					return -1
				case 69:
					return -1
				case 84:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 102:
					return -1
				case 70:
					return -1
				case 115:
					return -1
				case 101:
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
				case 116:
					return -1
				case 84:
					return -1
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
			func(r rune) int {
				switch r {
				case 105:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 112:
					return 2
				case 80:
					return 2
				case 116:
					return -1
				case 84:
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
				case 111:
					return -1
				case 79:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 116:
					return 3
				case 84:
					return 3
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
			func(r rune) int {
				switch r {
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
				case 116:
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
				case 105:
					return -1
				case 110:
					return -1
				case 111:
					return 5
				case 79:
					return 5
				case 112:
					return -1
				case 80:
					return -1
				case 116:
					return -1
				case 84:
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
				case 111:
					return -1
				case 79:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 73:
					return -1
				case 78:
					return 6
				case 105:
					return -1
				case 110:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 105:
					return -1
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
				case 116:
					return -1
				case 84:
					return -1
				case 73:
					return -1
				case 78:
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
				case 111:
					return 1
				case 79:
					return 1
				case 85:
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
				case 85:
					return 2
				case 116:
					return -1
				case 84:
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
				case 111:
					return -1
				case 79:
					return -1
				case 85:
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
				case 111:
					return -1
				case 79:
					return -1
				case 85:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 117:
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
				case 111:
					return -1
				case 79:
					return -1
				case 85:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 117:
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
				case 111:
					return -1
				case 79:
					return -1
				case 85:
					return -1
				case 116:
					return -1
				case 84:
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

		// [pP][aA][rR][sS][eE]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 112:
					return 1
				case 80:
					return 1
				case 97:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 65:
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
				case 65:
					return 2
				case 82:
					return -1
				case 69:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 97:
					return 2
				case 114:
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
				case 112:
					return -1
				case 80:
					return -1
				case 97:
					return -1
				case 114:
					return 3
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 65:
					return -1
				case 82:
					return 3
				case 69:
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
				case 114:
					return -1
				case 115:
					return 4
				case 83:
					return 4
				case 101:
					return -1
				case 65:
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
				case 112:
					return -1
				case 80:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return 5
				case 65:
					return -1
				case 82:
					return -1
				case 69:
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
				case 97:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 65:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [pP][aA][rR][tT][iI][tT][iI][oO][nN]
		{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 80:
					return 1
				case 97:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 65:
					return -1
				case 111:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 112:
					return 1
				case 79:
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
				case 97:
					return 2
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 65:
					return 2
				case 111:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 112:
					return -1
				case 79:
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
				case 97:
					return -1
				case 114:
					return 3
				case 82:
					return 3
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 65:
					return -1
				case 111:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 112:
					return -1
				case 79:
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
				case 97:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 65:
					return -1
				case 111:
					return -1
				case 116:
					return 4
				case 84:
					return 4
				case 112:
					return -1
				case 79:
					return -1
				case 78:
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
				case 116:
					return -1
				case 84:
					return -1
				case 112:
					return -1
				case 79:
					return -1
				case 78:
					return -1
				case 80:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return 5
				case 73:
					return 5
				case 110:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 116:
					return 6
				case 84:
					return 6
				case 112:
					return -1
				case 79:
					return -1
				case 78:
					return -1
				case 80:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 65:
					return -1
				case 111:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 80:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return 7
				case 73:
					return 7
				case 110:
					return -1
				case 65:
					return -1
				case 111:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 112:
					return -1
				case 79:
					return -1
				case 78:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 65:
					return -1
				case 111:
					return 8
				case 116:
					return -1
				case 84:
					return -1
				case 112:
					return -1
				case 79:
					return 8
				case 78:
					return -1
				case 80:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 73:
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
				case 84:
					return -1
				case 112:
					return -1
				case 79:
					return -1
				case 78:
					return 9
				case 80:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return 9
				case 65:
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
				case 111:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 112:
					return -1
				case 79:
					return -1
				case 78:
					return -1
				case 80:
					return -1
				case 97:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [pP][aA][sS][sS][wW][oO][rR][dD]
		{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 115:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 68:
					return -1
				case 112:
					return 1
				case 83:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 100:
					return -1
				case 80:
					return 1
				case 65:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return 2
				case 115:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 68:
					return -1
				case 112:
					return -1
				case 83:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 100:
					return -1
				case 80:
					return -1
				case 65:
					return 2
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 115:
					return 3
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 68:
					return -1
				case 112:
					return -1
				case 83:
					return 3
				case 119:
					return -1
				case 87:
					return -1
				case 100:
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
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 68:
					return -1
				case 112:
					return -1
				case 83:
					return 4
				case 119:
					return -1
				case 87:
					return -1
				case 100:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				case 97:
					return -1
				case 115:
					return 4
				case 111:
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
				case 97:
					return -1
				case 115:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 68:
					return -1
				case 112:
					return -1
				case 83:
					return -1
				case 119:
					return 5
				case 87:
					return 5
				case 100:
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
				case 97:
					return -1
				case 115:
					return -1
				case 111:
					return 6
				case 79:
					return 6
				case 114:
					return -1
				case 82:
					return -1
				case 68:
					return -1
				case 112:
					return -1
				case 83:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 100:
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
				case 97:
					return -1
				case 115:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return 7
				case 82:
					return 7
				case 68:
					return -1
				case 112:
					return -1
				case 83:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 100:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 83:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 100:
					return 8
				case 80:
					return -1
				case 65:
					return -1
				case 97:
					return -1
				case 115:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 68:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 79:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 68:
					return -1
				case 112:
					return -1
				case 83:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 100:
					return -1
				case 80:
					return -1
				case 65:
					return -1
				case 97:
					return -1
				case 115:
					return -1
				case 111:
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
				case 114:
					return -1
				case 82:
					return -1
				case 73:
					return -1
				case 109:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 112:
					return 1
				case 80:
					return 1
				case 105:
					return -1
				case 77:
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
				case 114:
					return 2
				case 82:
					return 2
				case 73:
					return -1
				case 109:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 105:
					return -1
				case 77:
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
				case 105:
					return 3
				case 77:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 73:
					return 3
				case 109:
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
				case 112:
					return -1
				case 80:
					return -1
				case 105:
					return -1
				case 77:
					return 4
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 73:
					return -1
				case 109:
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
				case 114:
					return -1
				case 82:
					return -1
				case 73:
					return -1
				case 109:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 105:
					return -1
				case 77:
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
				case 114:
					return 6
				case 82:
					return 6
				case 73:
					return -1
				case 109:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 105:
					return -1
				case 77:
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
				case 114:
					return -1
				case 82:
					return -1
				case 73:
					return -1
				case 109:
					return -1
				case 121:
					return 7
				case 89:
					return 7
				case 112:
					return -1
				case 80:
					return -1
				case 105:
					return -1
				case 77:
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
				case 105:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 73:
					return -1
				case 109:
					return -1
				case 121:
					return -1
				case 89:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [pP][rR][iI][vV][aA][tT][eE]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 80:
					return 1
				case 116:
					return -1
				case 69:
					return -1
				case 105:
					return -1
				case 118:
					return -1
				case 101:
					return -1
				case 112:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 73:
					return -1
				case 86:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return 2
				case 82:
					return 2
				case 73:
					return -1
				case 86:
					return -1
				case 80:
					return -1
				case 116:
					return -1
				case 69:
					return -1
				case 105:
					return -1
				case 118:
					return -1
				case 101:
					return -1
				case 112:
					return -1
				case 97:
					return -1
				case 65:
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
				case 116:
					return -1
				case 69:
					return -1
				case 105:
					return 3
				case 118:
					return -1
				case 101:
					return -1
				case 112:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 73:
					return 3
				case 86:
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
				case 65:
					return -1
				case 84:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 73:
					return -1
				case 86:
					return 4
				case 80:
					return -1
				case 116:
					return -1
				case 69:
					return -1
				case 105:
					return -1
				case 118:
					return 4
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 97:
					return 5
				case 65:
					return 5
				case 84:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 73:
					return -1
				case 86:
					return -1
				case 80:
					return -1
				case 116:
					return -1
				case 69:
					return -1
				case 105:
					return -1
				case 118:
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
				case 97:
					return -1
				case 65:
					return -1
				case 84:
					return 6
				case 114:
					return -1
				case 82:
					return -1
				case 73:
					return -1
				case 86:
					return -1
				case 80:
					return -1
				case 116:
					return 6
				case 69:
					return -1
				case 105:
					return -1
				case 118:
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
				case 82:
					return -1
				case 73:
					return -1
				case 86:
					return -1
				case 80:
					return -1
				case 116:
					return -1
				case 69:
					return 7
				case 105:
					return -1
				case 118:
					return -1
				case 101:
					return 7
				case 112:
					return -1
				case 97:
					return -1
				case 65:
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
				case 116:
					return -1
				case 69:
					return -1
				case 105:
					return -1
				case 118:
					return -1
				case 101:
					return -1
				case 112:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 84:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 73:
					return -1
				case 86:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [pP][rR][iI][vV][iI][lL][eE][gG][eE]
		{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 86:
					return -1
				case 108:
					return -1
				case 101:
					return -1
				case 71:
					return -1
				case 112:
					return 1
				case 105:
					return -1
				case 118:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 103:
					return -1
				case 80:
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
				case 112:
					return -1
				case 105:
					return -1
				case 118:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 103:
					return -1
				case 80:
					return -1
				case 114:
					return 2
				case 82:
					return 2
				case 86:
					return -1
				case 108:
					return -1
				case 101:
					return -1
				case 71:
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
				case 86:
					return -1
				case 108:
					return -1
				case 101:
					return -1
				case 71:
					return -1
				case 112:
					return -1
				case 105:
					return 3
				case 118:
					return -1
				case 69:
					return -1
				case 73:
					return 3
				case 76:
					return -1
				case 103:
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
				case 103:
					return -1
				case 80:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 86:
					return 4
				case 108:
					return -1
				case 101:
					return -1
				case 71:
					return -1
				case 112:
					return -1
				case 105:
					return -1
				case 118:
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
				case 108:
					return -1
				case 101:
					return -1
				case 71:
					return -1
				case 112:
					return -1
				case 105:
					return 5
				case 118:
					return -1
				case 69:
					return -1
				case 73:
					return 5
				case 76:
					return -1
				case 103:
					return -1
				case 80:
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
				case 112:
					return -1
				case 105:
					return -1
				case 118:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 76:
					return 6
				case 103:
					return -1
				case 80:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 86:
					return -1
				case 108:
					return 6
				case 101:
					return -1
				case 71:
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
				case 86:
					return -1
				case 108:
					return -1
				case 101:
					return 7
				case 71:
					return -1
				case 112:
					return -1
				case 105:
					return -1
				case 118:
					return -1
				case 69:
					return 7
				case 73:
					return -1
				case 76:
					return -1
				case 103:
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
				case 103:
					return 8
				case 80:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 86:
					return -1
				case 108:
					return -1
				case 101:
					return -1
				case 71:
					return 8
				case 112:
					return -1
				case 105:
					return -1
				case 118:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 86:
					return -1
				case 108:
					return -1
				case 101:
					return 9
				case 71:
					return -1
				case 112:
					return -1
				case 105:
					return -1
				case 118:
					return -1
				case 69:
					return 9
				case 73:
					return -1
				case 76:
					return -1
				case 103:
					return -1
				case 80:
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
				case 86:
					return -1
				case 108:
					return -1
				case 101:
					return -1
				case 71:
					return -1
				case 112:
					return -1
				case 105:
					return -1
				case 118:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 103:
					return -1
				case 80:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [pP][rR][oO][cC][eE][dE][uU][rR][eE]
		{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 80:
					return 1
				case 114:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 112:
					return 1
				case 82:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 67:
					return -1
				case 100:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 80:
					return -1
				case 114:
					return 2
				case 99:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 112:
					return -1
				case 82:
					return 2
				case 111:
					return -1
				case 79:
					return -1
				case 67:
					return -1
				case 100:
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
				case 99:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 112:
					return -1
				case 82:
					return -1
				case 111:
					return 3
				case 79:
					return 3
				case 67:
					return -1
				case 100:
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
				case 111:
					return -1
				case 79:
					return -1
				case 67:
					return 4
				case 100:
					return -1
				case 80:
					return -1
				case 114:
					return -1
				case 99:
					return 4
				case 101:
					return -1
				case 69:
					return -1
				case 117:
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
				case 82:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 67:
					return -1
				case 100:
					return -1
				case 80:
					return -1
				case 114:
					return -1
				case 99:
					return -1
				case 101:
					return 5
				case 69:
					return 5
				case 117:
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
				case 82:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 67:
					return -1
				case 100:
					return 6
				case 80:
					return -1
				case 114:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 69:
					return 6
				case 117:
					return -1
				case 85:
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
				case 99:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return 7
				case 85:
					return 7
				case 112:
					return -1
				case 82:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 67:
					return -1
				case 100:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 82:
					return 8
				case 111:
					return -1
				case 79:
					return -1
				case 67:
					return -1
				case 100:
					return -1
				case 80:
					return -1
				case 114:
					return 8
				case 99:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
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
				case 82:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 67:
					return -1
				case 100:
					return -1
				case 80:
					return -1
				case 114:
					return -1
				case 99:
					return -1
				case 101:
					return 9
				case 69:
					return 9
				case 117:
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
				case 82:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 67:
					return -1
				case 100:
					return -1
				case 80:
					return -1
				case 114:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
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
				case 112:
					return 1
				case 80:
					return 1
				case 117:
					return -1
				case 85:
					return -1
				case 73:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 105:
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
				case 98:
					return -1
				case 66:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 105:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 117:
					return 2
				case 85:
					return 2
				case 73:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 98:
					return 3
				case 66:
					return 3
				case 108:
					return -1
				case 76:
					return -1
				case 105:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 117:
					return -1
				case 85:
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
				case 80:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 73:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 108:
					return 4
				case 76:
					return 4
				case 105:
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
				case 80:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 73:
					return 5
				case 98:
					return -1
				case 66:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 105:
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
				case 80:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 73:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 105:
					return -1
				case 99:
					return 6
				case 67:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 112:
					return -1
				case 80:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 73:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 105:
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
				case 114:
					return 1
				case 82:
					return 1
				case 101:
					return -1
				case 69:
					return -1
				case 65:
					return -1
				case 97:
					return -1
				case 108:
					return -1
				case 76:
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
				case 97:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return 2
				case 69:
					return 2
				case 65:
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
				case 65:
					return 3
				case 97:
					return 3
				case 108:
					return -1
				case 76:
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
				case 82:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 65:
					return -1
				case 97:
					return -1
				case 108:
					return 4
				case 76:
					return 4
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
				case 82:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 65:
					return -1
				case 97:
					return -1
				case 108:
					return -1
				case 76:
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
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 65:
					return -1
				case 97:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 109:
					return -1
				case 77:
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
				case 100:
					return -1
				case 68:
					return -1
				case 82:
					return 1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 85:
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
				case 114:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 82:
					return -1
				case 101:
					return 2
				case 69:
					return 2
				case 117:
					return -1
				case 85:
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
				case 82:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 114:
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
				case 114:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return 4
				case 85:
					return 4
				case 99:
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
				case 100:
					return -1
				case 68:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 99:
					return 5
				case 67:
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
				case 117:
					return -1
				case 85:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 114:
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
				case 82:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 114:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [rR][eE][nN][aA][mM][eE]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return 1
				case 82:
					return 1
				case 78:
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
				case 82:
					return -1
				case 78:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 101:
					return 2
				case 69:
					return 2
				case 110:
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
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return 3
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 78:
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
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 97:
					return 4
				case 65:
					return 4
				case 114:
					return -1
				case 82:
					return -1
				case 78:
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
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 78:
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
				case 101:
					return 6
				case 69:
					return 6
				case 110:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 78:
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
				case 101:
					return -1
				case 69:
					return -1
				case 110:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 78:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [rR][eE][tT][uU][rR][nN]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 85:
					return -1
				case 114:
					return 1
				case 82:
					return 1
				case 101:
					return -1
				case 117:
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
				case 82:
					return -1
				case 101:
					return 2
				case 117:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 69:
					return 2
				case 116:
					return -1
				case 84:
					return -1
				case 85:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 69:
					return -1
				case 116:
					return 3
				case 84:
					return 3
				case 85:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 117:
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
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 85:
					return 4
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 117:
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
				case 114:
					return 5
				case 82:
					return 5
				case 101:
					return -1
				case 117:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 85:
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
				case 117:
					return -1
				case 110:
					return 6
				case 78:
					return 6
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 85:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 69:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 85:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 101:
					return -1
				case 117:
					return -1
				case 110:
					return -1
				case 78:
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
				case 82:
					return 1
				case 69:
					return -1
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
				case 116:
					return -1
				case 84:
					return -1
				case 103:
					return -1
				case 71:
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
				case 114:
					return -1
				case 82:
					return -1
				case 69:
					return 2
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
				case 116:
					return -1
				case 84:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 101:
					return 2
				case 73:
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
				case 114:
					return -1
				case 82:
					return -1
				case 69:
					return -1
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
				case 116:
					return 3
				case 84:
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
				case 105:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 101:
					return -1
				case 73:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 117:
					return 4
				case 85:
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
				case 116:
					return -1
				case 84:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 101:
					return -1
				case 73:
					return -1
				case 114:
					return 5
				case 82:
					return 5
				case 69:
					return -1
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
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 101:
					return -1
				case 73:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return 6
				case 78:
					return 6
				case 105:
					return -1
				case 116:
					return -1
				case 84:
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
				case 116:
					return -1
				case 84:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 101:
					return -1
				case 73:
					return 7
				case 114:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 105:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return 8
				case 78:
					return 8
				case 105:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 103:
					return -1
				case 71:
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
				case 114:
					return -1
				case 82:
					return -1
				case 69:
					return -1
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
				case 116:
					return -1
				case 84:
					return -1
				case 103:
					return 9
				case 71:
					return 9
				case 101:
					return -1
				case 73:
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
				case 69:
					return -1
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
				case 116:
					return -1
				case 84:
					return -1
				case 103:
					return -1
				case 71:
					return -1
				case 101:
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
				case 82:
					return 1
				case 69:
					return -1
				case 86:
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
					return 2
				case 118:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 107:
					return -1
				case 82:
					return -1
				case 69:
					return 2
				case 86:
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
					return 3
				case 75:
					return -1
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
				case 82:
					return -1
				case 69:
					return -1
				case 86:
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
					return -1
				case 79:
					return -1
				case 107:
					return 5
				case 82:
					return -1
				case 69:
					return -1
				case 86:
					return -1
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
				case 75:
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
					return -1
				case 79:
					return -1
				case 107:
					return -1
				case 82:
					return -1
				case 69:
					return -1
				case 86:
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
				case 82:
					return 1
				case 73:
					return -1
				case 103:
					return -1
				case 104:
					return -1
				case 114:
					return 1
				case 105:
					return -1
				case 71:
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
				case 82:
					return -1
				case 73:
					return 2
				case 103:
					return -1
				case 104:
					return -1
				case 114:
					return -1
				case 105:
					return 2
				case 71:
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
				case 82:
					return -1
				case 73:
					return -1
				case 103:
					return 3
				case 104:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				case 71:
					return 3
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
				case 105:
					return -1
				case 71:
					return -1
				case 72:
					return 4
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 73:
					return -1
				case 103:
					return -1
				case 104:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 82:
					return -1
				case 73:
					return -1
				case 103:
					return -1
				case 104:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				case 71:
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
				case 82:
					return -1
				case 73:
					return -1
				case 103:
					return -1
				case 104:
					return -1
				case 114:
					return -1
				case 105:
					return -1
				case 71:
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
				case 98:
					return -1
				case 66:
					return -1
				case 107:
					return -1
				case 82:
					return 1
				case 111:
					return -1
				case 99:
					return -1
				case 75:
					return -1
				case 114:
					return 1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 97:
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
				case 98:
					return -1
				case 66:
					return -1
				case 107:
					return -1
				case 82:
					return -1
				case 111:
					return 2
				case 99:
					return -1
				case 75:
					return -1
				case 114:
					return -1
				case 79:
					return 2
				case 108:
					return -1
				case 76:
					return -1
				case 97:
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
				case 97:
					return -1
				case 65:
					return -1
				case 67:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 107:
					return -1
				case 82:
					return -1
				case 111:
					return -1
				case 99:
					return -1
				case 75:
					return -1
				case 114:
					return -1
				case 79:
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
				case 114:
					return -1
				case 79:
					return -1
				case 108:
					return 4
				case 76:
					return 4
				case 97:
					return -1
				case 65:
					return -1
				case 67:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 107:
					return -1
				case 82:
					return -1
				case 111:
					return -1
				case 99:
					return -1
				case 75:
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
				case 67:
					return -1
				case 98:
					return 5
				case 66:
					return 5
				case 107:
					return -1
				case 82:
					return -1
				case 111:
					return -1
				case 99:
					return -1
				case 75:
					return -1
				case 114:
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
				case 82:
					return -1
				case 111:
					return -1
				case 99:
					return -1
				case 75:
					return -1
				case 114:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 97:
					return 6
				case 65:
					return 6
				case 67:
					return -1
				case 98:
					return -1
				case 66:
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
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 67:
					return 7
				case 98:
					return -1
				case 66:
					return -1
				case 107:
					return -1
				case 82:
					return -1
				case 111:
					return -1
				case 99:
					return 7
				case 75:
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
				case 67:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 107:
					return 8
				case 82:
					return -1
				case 111:
					return -1
				case 99:
					return -1
				case 75:
					return 8
				case 114:
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
				case 114:
					return -1
				case 79:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 67:
					return -1
				case 98:
					return -1
				case 66:
					return -1
				case 107:
					return -1
				case 82:
					return -1
				case 111:
					return -1
				case 99:
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
				case 97:
					return -1
				case 65:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 70:
					return -1
				case 115:
					return 1
				case 83:
					return 1
				case 116:
					return -1
				case 84:
					return -1
				case 102:
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
				case 116:
					return -1
				case 84:
					return -1
				case 102:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return 2
				case 65:
					return 2
				case 105:
					return -1
				case 73:
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
				case 116:
					return 3
				case 84:
					return 3
				case 102:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 105:
					return -1
				case 73:
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
				case 116:
					return -1
				case 84:
					return -1
				case 102:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 105:
					return 4
				case 73:
					return 4
				case 70:
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
				case 105:
					return -1
				case 73:
					return -1
				case 70:
					return -1
				case 115:
					return 5
				case 83:
					return 5
				case 116:
					return -1
				case 84:
					return -1
				case 102:
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
				case 116:
					return -1
				case 84:
					return -1
				case 102:
					return 6
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 70:
					return 6
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
				case 102:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 105:
					return 7
				case 73:
					return 7
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
				case 116:
					return -1
				case 84:
					return -1
				case 102:
					return -1
				case 101:
					return 8
				case 69:
					return 8
				case 97:
					return -1
				case 65:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 70:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return 9
				case 83:
					return 9
				case 116:
					return -1
				case 84:
					return -1
				case 102:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 105:
					return -1
				case 73:
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
				case 116:
					return -1
				case 84:
					return -1
				case 102:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 70:
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
				case 99:
					return -1
				case 67:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 65:
					return -1
				case 83:
					return 1
				case 69:
					return -1
				case 109:
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
				case 83:
					return -1
				case 69:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 115:
					return -1
				case 99:
					return 2
				case 67:
					return 2
				case 104:
					return -1
				case 72:
					return -1
				case 101:
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
				case 99:
					return -1
				case 67:
					return -1
				case 104:
					return 3
				case 72:
					return 3
				case 101:
					return -1
				case 65:
					return -1
				case 83:
					return -1
				case 69:
					return -1
				case 109:
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
				case 83:
					return -1
				case 69:
					return 4
				case 109:
					return -1
				case 77:
					return -1
				case 97:
					return -1
				case 115:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return 4
				case 65:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 65:
					return -1
				case 83:
					return -1
				case 69:
					return -1
				case 109:
					return 5
				case 77:
					return 5
				case 97:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 65:
					return 6
				case 83:
					return -1
				case 69:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 97:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 101:
					return -1
				case 65:
					return -1
				case 83:
					return -1
				case 69:
					return -1
				case 109:
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
				case 115:
					return 1
				case 83:
					return 1
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				case 99:
					return -1
				case 116:
					return -1
				case 108:
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
				case 108:
					return -1
				case 67:
					return -1
				case 84:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return 2
				case 69:
					return 2
				case 76:
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
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return 3
				case 99:
					return -1
				case 116:
					return -1
				case 108:
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
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				case 76:
					return -1
				case 99:
					return -1
				case 116:
					return -1
				case 108:
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
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				case 99:
					return 5
				case 116:
					return -1
				case 108:
					return -1
				case 67:
					return 5
				case 84:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 67:
					return -1
				case 84:
					return 6
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 76:
					return -1
				case 99:
					return -1
				case 116:
					return 6
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
				case 76:
					return -1
				case 99:
					return -1
				case 116:
					return -1
				case 108:
					return -1
				case 67:
					return -1
				case 84:
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
				case 83:
					return 1
				case 65:
					return -1
				case 105:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 115:
					return 1
				case 116:
					return -1
				case 84:
					return -1
				case 97:
					return -1
				case 73:
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
				case 105:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 115:
					return -1
				case 116:
					return 2
				case 84:
					return 2
				case 97:
					return -1
				case 73:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 83:
					return -1
				case 65:
					return 3
				case 105:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 97:
					return 3
				case 73:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 116:
					return 4
				case 84:
					return 4
				case 97:
					return -1
				case 73:
					return -1
				case 83:
					return -1
				case 65:
					return -1
				case 105:
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
				case 115:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 97:
					return -1
				case 73:
					return 5
				case 83:
					return -1
				case 65:
					return -1
				case 105:
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
				case 115:
					return 6
				case 116:
					return -1
				case 84:
					return -1
				case 97:
					return -1
				case 73:
					return -1
				case 83:
					return 6
				case 65:
					return -1
				case 105:
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
				case 83:
					return -1
				case 65:
					return -1
				case 105:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 115:
					return -1
				case 116:
					return 7
				case 84:
					return 7
				case 97:
					return -1
				case 73:
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
				case 105:
					return 8
				case 99:
					return -1
				case 67:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 97:
					return -1
				case 73:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 97:
					return -1
				case 73:
					return -1
				case 83:
					return -1
				case 65:
					return -1
				case 105:
					return -1
				case 99:
					return 9
				case 67:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 83:
					return 10
				case 65:
					return -1
				case 105:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 115:
					return 10
				case 116:
					return -1
				case 84:
					return -1
				case 97:
					return -1
				case 73:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 97:
					return -1
				case 73:
					return -1
				case 83:
					return -1
				case 65:
					return -1
				case 105:
					return -1
				case 99:
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
				case 115:
					return 1
				case 83:
					return 1
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 103:
					return -1
				case 114:
					return -1
				case 73:
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
				case 84:
					return 2
				case 82:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 103:
					return -1
				case 114:
					return -1
				case 73:
					return -1
				case 71:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return 3
				case 73:
					return -1
				case 71:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return 3
				case 105:
					return -1
				case 110:
					return -1
				case 78:
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
				case 73:
					return 4
				case 71:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 105:
					return 4
				case 110:
					return -1
				case 78:
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
				case 73:
					return -1
				case 71:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 110:
					return 5
				case 78:
					return 5
				case 103:
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
				case 71:
					return 6
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 103:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return -1
				case 73:
					return -1
				case 71:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 105:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 103:
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
				case 83:
					return 1
				case 121:
					return -1
				case 69:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 89:
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
				case 115:
					return -1
				case 83:
					return -1
				case 121:
					return 2
				case 69:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 89:
					return 2
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
				case 115:
					return 3
				case 83:
					return 3
				case 121:
					return -1
				case 69:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 89:
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
				case 115:
					return -1
				case 83:
					return -1
				case 121:
					return -1
				case 69:
					return -1
				case 109:
					return -1
				case 77:
					return -1
				case 89:
					return -1
				case 116:
					return 4
				case 84:
					return 4
				case 101:
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
				case 121:
					return -1
				case 69:
					return 5
				case 109:
					return -1
				case 77:
					return -1
				case 89:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 115:
					return -1
				case 83:
					return -1
				case 121:
					return -1
				case 69:
					return -1
				case 109:
					return 6
				case 77:
					return 6
				case 89:
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
				case 89:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 121:
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
				case 65:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 115:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 84:
					return 1
				case 111:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return -1
				case 105:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return 2
				case 82:
					return 2
				case 97:
					return -1
				case 105:
					return -1
				case 116:
					return -1
				case 65:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 115:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 84:
					return -1
				case 111:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 84:
					return -1
				case 111:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return 3
				case 105:
					return -1
				case 116:
					return -1
				case 65:
					return 3
				case 83:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 115:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 73:
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
				case 82:
					return -1
				case 97:
					return -1
				case 105:
					return -1
				case 116:
					return -1
				case 65:
					return -1
				case 83:
					return -1
				case 110:
					return 4
				case 78:
					return 4
				case 115:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 84:
					return -1
				case 111:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 84:
					return -1
				case 111:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return -1
				case 105:
					return -1
				case 116:
					return -1
				case 65:
					return -1
				case 83:
					return 5
				case 110:
					return -1
				case 78:
					return -1
				case 115:
					return 5
				case 99:
					return -1
				case 67:
					return -1
				case 73:
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
				case 65:
					return 6
				case 83:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 115:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 84:
					return -1
				case 111:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return 6
				case 105:
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
				case 105:
					return -1
				case 116:
					return -1
				case 65:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 115:
					return -1
				case 99:
					return 7
				case 67:
					return 7
				case 73:
					return -1
				case 79:
					return -1
				case 84:
					return -1
				case 111:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 84:
					return 8
				case 111:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return -1
				case 105:
					return -1
				case 116:
					return 8
				case 65:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 115:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 73:
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
				case 65:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 115:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 73:
					return 9
				case 79:
					return -1
				case 84:
					return -1
				case 111:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return -1
				case 105:
					return 9
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
				case 105:
					return -1
				case 116:
					return -1
				case 65:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 115:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 73:
					return -1
				case 79:
					return 10
				case 84:
					return -1
				case 111:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 84:
					return -1
				case 111:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 97:
					return -1
				case 105:
					return -1
				case 116:
					return -1
				case 65:
					return -1
				case 83:
					return -1
				case 110:
					return 11
				case 78:
					return 11
				case 115:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 73:
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
				case 82:
					return -1
				case 97:
					return -1
				case 105:
					return -1
				case 116:
					return -1
				case 65:
					return -1
				case 83:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 115:
					return -1
				case 99:
					return -1
				case 67:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 84:
					return -1
				case 111:
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
				case 105:
					return -1
				case 73:
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
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return 2
				case 82:
					return 2
				case 103:
					return -1
				case 71:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
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
				case 105:
					return 3
				case 73:
					return 3
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
				case 101:
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
				case 105:
					return -1
				case 73:
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
				case 101:
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
				case 105:
					return -1
				case 73:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 103:
					return 5
				case 71:
					return 5
				case 101:
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
				case 103:
					return -1
				case 71:
					return -1
				case 101:
					return 6
				case 116:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 69:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 114:
					return 7
				case 82:
					return 7
				case 103:
					return -1
				case 71:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
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
				case 103:
					return -1
				case 71:
					return -1
				case 101:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 69:
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
				case 114:
					return -1
				case 110:
					return -1
				case 117:
					return -1
				case 99:
					return -1
				case 65:
					return -1
				case 101:
					return -1
				case 85:
					return -1
				case 78:
					return -1
				case 97:
					return -1
				case 116:
					return 1
				case 84:
					return 1
				case 82:
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
				case 117:
					return -1
				case 99:
					return -1
				case 65:
					return -1
				case 101:
					return -1
				case 85:
					return -1
				case 78:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return 2
				case 67:
					return -1
				case 69:
					return -1
				case 114:
					return 2
				case 110:
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
				case 67:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 110:
					return -1
				case 117:
					return 3
				case 99:
					return -1
				case 65:
					return -1
				case 101:
					return -1
				case 85:
					return 3
				case 78:
					return -1
				case 97:
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
				case 67:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 110:
					return 4
				case 117:
					return -1
				case 99:
					return -1
				case 65:
					return -1
				case 101:
					return -1
				case 85:
					return -1
				case 78:
					return 4
				case 97:
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
				case 67:
					return 5
				case 69:
					return -1
				case 114:
					return -1
				case 110:
					return -1
				case 117:
					return -1
				case 99:
					return 5
				case 65:
					return -1
				case 101:
					return -1
				case 85:
					return -1
				case 78:
					return -1
				case 97:
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
				case 67:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 110:
					return -1
				case 117:
					return -1
				case 99:
					return -1
				case 65:
					return 6
				case 101:
					return -1
				case 85:
					return -1
				case 78:
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
				case 110:
					return -1
				case 117:
					return -1
				case 99:
					return -1
				case 65:
					return -1
				case 101:
					return -1
				case 85:
					return -1
				case 78:
					return -1
				case 97:
					return -1
				case 116:
					return 7
				case 84:
					return 7
				case 82:
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
				case 117:
					return -1
				case 99:
					return -1
				case 65:
					return -1
				case 101:
					return 8
				case 85:
					return -1
				case 78:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 67:
					return -1
				case 69:
					return 8
				case 114:
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
				case 110:
					return -1
				case 117:
					return -1
				case 99:
					return -1
				case 65:
					return -1
				case 101:
					return -1
				case 85:
					return -1
				case 78:
					return -1
				case 97:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 82:
					return -1
				case 67:
					return -1
				case 69:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [uU][nN][dD][eE][rR]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 85:
					return 1
				case 68:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 117:
					return 1
				case 110:
					return -1
				case 78:
					return -1
				case 100:
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
				case 85:
					return -1
				case 68:
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
				case 85:
					return -1
				case 68:
					return 3
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return -1
				case 82:
					return -1
				case 117:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 100:
					return 3
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
				case 85:
					return -1
				case 68:
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
				case 85:
					return -1
				case 68:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 114:
					return 5
				case 82:
					return 5
				case 117:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 100:
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
				case 85:
					return -1
				case 68:
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
				case 117:
					return 1
				case 105:
					return -1
				case 81:
					return -1
				case 101:
					return -1
				case 85:
					return 1
				case 110:
					return -1
				case 78:
					return -1
				case 73:
					return -1
				case 113:
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
				case 81:
					return -1
				case 101:
					return -1
				case 85:
					return -1
				case 110:
					return 2
				case 78:
					return 2
				case 73:
					return -1
				case 113:
					return -1
				case 69:
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
				case 113:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 105:
					return 3
				case 81:
					return -1
				case 101:
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
				case 81:
					return 4
				case 101:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 73:
					return -1
				case 113:
					return 4
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return 5
				case 105:
					return -1
				case 81:
					return -1
				case 101:
					return -1
				case 85:
					return 5
				case 110:
					return -1
				case 78:
					return -1
				case 73:
					return -1
				case 113:
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
				case 81:
					return -1
				case 101:
					return 6
				case 85:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 73:
					return -1
				case 113:
					return -1
				case 69:
					return 6
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
				case 113:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 105:
					return -1
				case 81:
					return -1
				case 101:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [uU][nN][kK][nN][oO][wW][nN]
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 117:
					return 1
				case 110:
					return -1
				case 78:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 87:
					return -1
				case 85:
					return 1
				case 107:
					return -1
				case 75:
					return -1
				case 119:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 85:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 119:
					return -1
				case 117:
					return -1
				case 110:
					return 2
				case 78:
					return 2
				case 111:
					return -1
				case 79:
					return -1
				case 87:
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
				case 111:
					return -1
				case 79:
					return -1
				case 87:
					return -1
				case 85:
					return -1
				case 107:
					return 3
				case 75:
					return 3
				case 119:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 110:
					return 4
				case 78:
					return 4
				case 111:
					return -1
				case 79:
					return -1
				case 87:
					return -1
				case 85:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 119:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 85:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 119:
					return -1
				case 117:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 111:
					return 5
				case 79:
					return 5
				case 87:
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
				case 111:
					return -1
				case 79:
					return -1
				case 87:
					return 6
				case 85:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 119:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 117:
					return -1
				case 110:
					return 7
				case 78:
					return 7
				case 111:
					return -1
				case 79:
					return -1
				case 87:
					return -1
				case 85:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 119:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 85:
					return -1
				case 107:
					return -1
				case 75:
					return -1
				case 119:
					return -1
				case 117:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 111:
					return -1
				case 79:
					return -1
				case 87:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [uU][nN][nN][eE][sS][tT]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 85:
					return 1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return 1
				case 110:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 69:
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
				case 110:
					return 2
				case 78:
					return 2
				case 101:
					return -1
				case 69:
					return -1
				case 83:
					return -1
				case 84:
					return -1
				case 85:
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
				case 85:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 110:
					return 3
				case 78:
					return 3
				case 101:
					return -1
				case 69:
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
				case 110:
					return -1
				case 78:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				case 83:
					return -1
				case 84:
					return -1
				case 85:
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
				case 117:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 83:
					return 5
				case 84:
					return -1
				case 85:
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
				case 85:
					return -1
				case 115:
					return -1
				case 116:
					return 6
				case 117:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 83:
					return -1
				case 84:
					return 6
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
				case 101:
					return -1
				case 69:
					return -1
				case 83:
					return -1
				case 84:
					return -1
				case 85:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [uU][nN][sS][eE][tT]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 78:
					return -1
				case 115:
					return -1
				case 84:
					return -1
				case 117:
					return 1
				case 85:
					return 1
				case 110:
					return -1
				case 83:
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
				case 110:
					return 2
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 78:
					return 2
				case 115:
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
				case 115:
					return 3
				case 84:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 83:
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
				case 117:
					return -1
				case 85:
					return -1
				case 110:
					return -1
				case 83:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				case 116:
					return -1
				case 78:
					return -1
				case 115:
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
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return 5
				case 78:
					return -1
				case 115:
					return -1
				case 84:
					return 5
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
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 116:
					return -1
				case 78:
					return -1
				case 115:
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
				case 112:
					return -1
				case 80:
					return -1
				case 100:
					return -1
				case 65:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return 1
				case 85:
					return 1
				case 68:
					return -1
				case 97:
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
				case 68:
					return -1
				case 97:
					return -1
				case 112:
					return 2
				case 80:
					return 2
				case 100:
					return -1
				case 65:
					return -1
				case 116:
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
				case 117:
					return -1
				case 85:
					return -1
				case 68:
					return 3
				case 97:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 100:
					return 3
				case 65:
					return -1
				case 116:
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
				case 117:
					return -1
				case 85:
					return -1
				case 68:
					return -1
				case 97:
					return 4
				case 112:
					return -1
				case 80:
					return -1
				case 100:
					return -1
				case 65:
					return 4
				case 116:
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
				case 112:
					return -1
				case 80:
					return -1
				case 100:
					return -1
				case 65:
					return -1
				case 116:
					return 5
				case 84:
					return 5
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 68:
					return -1
				case 97:
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
				case 68:
					return -1
				case 97:
					return -1
				case 112:
					return -1
				case 80:
					return -1
				case 100:
					return -1
				case 65:
					return -1
				case 116:
					return -1
				case 84:
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
				case 112:
					return -1
				case 80:
					return -1
				case 100:
					return -1
				case 65:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 68:
					return -1
				case 97:
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
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 82:
					return -1
				case 80:
					return -1
				case 114:
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
					return 2
				case 114:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 112:
					return 2
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
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
				case 114:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 112:
					return -1
				case 115:
					return 3
				case 83:
					return 3
				case 101:
					return -1
				case 69:
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
				case 114:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 112:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return 4
				case 69:
					return 4
				case 82:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 80:
					return -1
				case 114:
					return 5
				case 116:
					return -1
				case 84:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 112:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 82:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 80:
					return -1
				case 114:
					return -1
				case 116:
					return 6
				case 84:
					return 6
				case 117:
					return -1
				case 85:
					return -1
				case 112:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
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
				case 114:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 112:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 82:
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
				case 85:
					return 1
				case 115:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 103:
					return -1
				case 117:
					return 1
				case 83:
					return -1
				case 105:
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
				case 105:
					return -1
				case 71:
					return -1
				case 85:
					return -1
				case 115:
					return 2
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 103:
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
				case 73:
					return 3
				case 110:
					return -1
				case 78:
					return -1
				case 103:
					return -1
				case 117:
					return -1
				case 83:
					return -1
				case 105:
					return 3
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
				case 105:
					return -1
				case 71:
					return -1
				case 85:
					return -1
				case 115:
					return -1
				case 73:
					return -1
				case 110:
					return 4
				case 78:
					return 4
				case 103:
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
				case 105:
					return -1
				case 71:
					return 5
				case 85:
					return -1
				case 115:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 103:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 85:
					return -1
				case 115:
					return -1
				case 73:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 103:
					return -1
				case 117:
					return -1
				case 83:
					return -1
				case 105:
					return -1
				case 71:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

		// [vV][aA][lL][iI][dD][aA][tT][eE]
		{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 108:
					return -1
				case 68:
					return -1
				case 86:
					return 1
				case 100:
					return -1
				case 101:
					return -1
				case 118:
					return 1
				case 65:
					return -1
				case 76:
					return -1
				case 105:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 118:
					return -1
				case 65:
					return 2
				case 76:
					return -1
				case 105:
					return -1
				case 69:
					return -1
				case 97:
					return 2
				case 73:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 108:
					return -1
				case 68:
					return -1
				case 86:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 97:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 108:
					return 3
				case 68:
					return -1
				case 86:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 118:
					return -1
				case 65:
					return -1
				case 76:
					return 3
				case 105:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 86:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 118:
					return -1
				case 65:
					return -1
				case 76:
					return -1
				case 105:
					return 4
				case 69:
					return -1
				case 97:
					return -1
				case 73:
					return 4
				case 116:
					return -1
				case 84:
					return -1
				case 108:
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
				case 76:
					return -1
				case 105:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 108:
					return -1
				case 68:
					return 5
				case 86:
					return -1
				case 100:
					return 5
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 108:
					return -1
				case 68:
					return -1
				case 86:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 118:
					return -1
				case 65:
					return 6
				case 76:
					return -1
				case 105:
					return -1
				case 69:
					return -1
				case 97:
					return 6
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
				case 97:
					return -1
				case 73:
					return -1
				case 116:
					return 7
				case 84:
					return 7
				case 108:
					return -1
				case 68:
					return -1
				case 86:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 118:
					return -1
				case 65:
					return -1
				case 76:
					return -1
				case 105:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 86:
					return -1
				case 100:
					return -1
				case 101:
					return 8
				case 118:
					return -1
				case 65:
					return -1
				case 76:
					return -1
				case 105:
					return -1
				case 69:
					return 8
				case 97:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 108:
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
				case 76:
					return -1
				case 105:
					return -1
				case 69:
					return -1
				case 97:
					return -1
				case 73:
					return -1
				case 116:
					return -1
				case 84:
					return -1
				case 108:
					return -1
				case 68:
					return -1
				case 86:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [vV][aA][lL][uU][eE]
		{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 86:
					return 1
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
				case 118:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 118:
					return -1
				case 97:
					return 2
				case 65:
					return 2
				case 117:
					return -1
				case 86:
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
			func(r rune) int {
				switch r {
				case 118:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 117:
					return -1
				case 86:
					return -1
				case 108:
					return 3
				case 76:
					return 3
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
				case 118:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 117:
					return 4
				case 86:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 85:
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
				case 86:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 85:
					return -1
				case 101:
					return 5
				case 69:
					return 5
				case 118:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 86:
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
				case 118:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 117:
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
				case 97:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 86:
					return 1
				case 65:
					return -1
				case 108:
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
				case 118:
					return -1
				case 97:
					return 2
				case 76:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 86:
					return -1
				case 65:
					return 2
				case 108:
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
				case 65:
					return -1
				case 108:
					return 3
				case 100:
					return -1
				case 68:
					return -1
				case 118:
					return -1
				case 97:
					return -1
				case 76:
					return 3
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
				case 118:
					return -1
				case 97:
					return -1
				case 76:
					return -1
				case 117:
					return 4
				case 85:
					return 4
				case 101:
					return -1
				case 69:
					return -1
				case 86:
					return -1
				case 65:
					return -1
				case 108:
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
				case 65:
					return -1
				case 108:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 118:
					return -1
				case 97:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 85:
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
				case 118:
					return -1
				case 97:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 69:
					return -1
				case 86:
					return -1
				case 65:
					return -1
				case 108:
					return -1
				case 100:
					return 6
				case 68:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 86:
					return -1
				case 65:
					return -1
				case 108:
					return -1
				case 100:
					return -1
				case 68:
					return -1
				case 118:
					return -1
				case 97:
					return -1
				case 76:
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
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [vV][aA][lL][uU][eE][sS]
		{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 86:
					return 1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 118:
					return 1
				case 97:
					return -1
				case 65:
					return -1
				case 69:
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
				case 118:
					return -1
				case 97:
					return 2
				case 65:
					return 2
				case 69:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 86:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
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
				case 108:
					return 3
				case 76:
					return 3
				case 117:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				case 118:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 69:
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
				case 118:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 86:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return 4
				case 85:
					return 4
				case 101:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 86:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 101:
					return 5
				case 118:
					return -1
				case 97:
					return -1
				case 65:
					return -1
				case 69:
					return 5
				case 115:
					return -1
				case 83:
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
				case 69:
					return -1
				case 115:
					return 6
				case 83:
					return 6
				case 86:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
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
				case 97:
					return -1
				case 65:
					return -1
				case 69:
					return -1
				case 115:
					return -1
				case 83:
					return -1
				case 86:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 117:
					return -1
				case 85:
					return -1
				case 101:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [vV][iI][aA]
		{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
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
				case 97:
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
				case 105:
					return 2
				case 73:
					return 2
				case 97:
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
				case 105:
					return -1
				case 73:
					return -1
				case 97:
					return 3
				case 65:
					return 3
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
				case 97:
					return -1
				case 65:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

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
				case 105:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 69:
					return -1
				case 119:
					return 1
				case 87:
					return 1
				case 104:
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
				case 72:
					return 2
				case 105:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 69:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 104:
					return 2
				case 73:
					return -1
				case 101:
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
				case 73:
					return 3
				case 101:
					return -1
				case 72:
					return -1
				case 105:
					return 3
				case 108:
					return -1
				case 76:
					return -1
				case 69:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 72:
					return -1
				case 105:
					return -1
				case 108:
					return 4
				case 76:
					return 4
				case 69:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 104:
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
				case 119:
					return -1
				case 87:
					return -1
				case 104:
					return -1
				case 73:
					return -1
				case 101:
					return 5
				case 72:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 76:
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
				case 105:
					return -1
				case 108:
					return -1
				case 76:
					return -1
				case 69:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 104:
					return -1
				case 73:
					return -1
				case 101:
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
				case 105:
					return -1
				case 73:
					return -1
				case 84:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 116:
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
				case 116:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 105:
					return 2
				case 73:
					return 2
				case 84:
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
					return 3
				case 104:
					return -1
				case 72:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 84:
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
				case 105:
					return -1
				case 73:
					return -1
				case 84:
					return -1
				case 110:
					return -1
				case 78:
					return -1
				case 116:
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
				case 116:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 105:
					return 5
				case 73:
					return 5
				case 84:
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
				case 105:
					return -1
				case 73:
					return -1
				case 84:
					return -1
				case 110:
					return 6
				case 78:
					return 6
				case 116:
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
				case 116:
					return -1
				case 104:
					return -1
				case 72:
					return -1
				case 119:
					return -1
				case 87:
					return -1
				case 105:
					return -1
				case 73:
					return -1
				case 84:
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

		// .
		{[]bool{false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				return 1
			},
			func(r rune) int {
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
				logToken(yylex.Text(), "STR - %s", lval.s)
				return STR
			}
			continue
		case 1:
			{
				lval.s, _ = UnmarshalSingleQuoted(yylex.Text())
				logToken(yylex.Text(), "STR - %s", lval.s)
				return STR
			}
			continue
		case 2:
			{
				// Case-insensitive identifier
				text := yylex.Text()
				text = text[0 : len(text)-1]
				lval.s, _ = UnmarshalBackQuoted(text)
				logToken(yylex.Text(), "IDENT_ICASE - %s", lval.s)
				return IDENT_ICASE
			}
			continue
		case 3:
			{
				// Escaped identifier
				lval.s, _ = UnmarshalBackQuoted(yylex.Text())
				logToken(yylex.Text(), "IDENT - %s", lval.s)
				return IDENT
			}
			continue
		case 4:
			{
				// We differentiate NUM from INT
				lval.f, _ = strconv.ParseFloat(yylex.Text(), 64)
				logToken(yylex.Text(), "NUM - %f", lval.f)
				return NUM
			}
			continue
		case 5:
			{
				// We differentiate NUM from INT
				lval.f, _ = strconv.ParseFloat(yylex.Text(), 64)
				logToken(yylex.Text(), "NUM - %f", lval.f)
				return NUM
			}
			continue
		case 6:
			{
				// We differentiate NUM from INT
				lval.n, _ = strconv.ParseInt(yylex.Text(), 10, 64)
				if (lval.n > math.MinInt64 && lval.n < math.MaxInt64) || strconv.FormatInt(lval.n, 10) == yylex.Text() {
					logToken(yylex.Text(), "INT - %d", lval.n)
					return INT
				} else {
					lval.f, _ = strconv.ParseFloat(yylex.Text(), 64)
					logToken(yylex.Text(), "NUM - %f", lval.f)
					return NUM
				}
			}
			continue
		case 7:
			{
				logToken(yylex.Text(), "BLOCK_COMMENT (length=%d)", len(yylex.Text())) /* eat up block comment */
			}
			continue
		case 8:
			{
				logToken(yylex.Text(), "LINE_COMMENT (length=%d)", len(yylex.Text())) /* eat up line comment */
			}
			continue
		case 9:
			{
				logToken(yylex.Text(), "WHITESPACE (count=%d)", len(yylex.Text())) /* eat up whitespace */
			}
			continue
		case 10:
			{
				logToken(yylex.Text(), "DOT")
				return DOT
			}
			continue
		case 11:
			{
				logToken(yylex.Text(), "PLUS")
				return PLUS
			}
			continue
		case 12:
			{
				logToken(yylex.Text(), "MINUS")
				return MINUS
			}
			continue
		case 13:
			{
				logToken(yylex.Text(), "MULT")
				return STAR
			}
			continue
		case 14:
			{
				logToken(yylex.Text(), "DIV")
				return DIV
			}
			continue
		case 15:
			{
				logToken(yylex.Text(), "MOD")
				return MOD
			}
			continue
		case 16:
			{
				logToken(yylex.Text(), "DEQ")
				return DEQ
			}
			continue
		case 17:
			{
				logToken(yylex.Text(), "EQ")
				return EQ
			}
			continue
		case 18:
			{
				logToken(yylex.Text(), "NE")
				return NE
			}
			continue
		case 19:
			{
				logToken(yylex.Text(), "NE")
				return NE
			}
			continue
		case 20:
			{
				logToken(yylex.Text(), "LT")
				return LT
			}
			continue
		case 21:
			{
				logToken(yylex.Text(), "LTE")
				return LE
			}
			continue
		case 22:
			{
				logToken(yylex.Text(), "GT")
				return GT
			}
			continue
		case 23:
			{
				logToken(yylex.Text(), "GTE")
				return GE
			}
			continue
		case 24:
			{
				logToken(yylex.Text(), "CONCAT")
				return CONCAT
			}
			continue
		case 25:
			{
				logToken(yylex.Text(), "LPAREN")
				return LPAREN
			}
			continue
		case 26:
			{
				logToken(yylex.Text(), "RPAREN")
				return RPAREN
			}
			continue
		case 27:
			{
				logToken(yylex.Text(), "LBRACE")
				return LBRACE
			}
			continue
		case 28:
			{
				logToken(yylex.Text(), "RBRACE")
				return RBRACE
			}
			continue
		case 29:
			{
				logToken(yylex.Text(), "COMMA")
				return COMMA
			}
			continue
		case 30:
			{
				logToken(yylex.Text(), "COLON")
				return COLON
			}
			continue
		case 31:
			{
				logToken(yylex.Text(), "LBRACKET")
				return LBRACKET
			}
			continue
		case 32:
			{
				logToken(yylex.Text(), "RBRACKET")
				return RBRACKET
			}
			continue
		case 33:
			{
				logToken(yylex.Text(), "RBRACKET_ICASE")
				return RBRACKET_ICASE
			}
			continue
		case 34:
			{
				logToken(yylex.Text(), "SEMI")
				return SEMI
			}
			continue
		case 35:
			{
				logToken(yylex.Text(), "NOT_A_TOKEN")
				return NOT_A_TOKEN
			}
			continue
		case 36:
			{
				logToken(yylex.Text(), "ALL")
				return ALL
			}
			continue
		case 37:
			{
				logToken(yylex.Text(), "ALTER")
				return ALTER
			}
			continue
		case 38:
			{
				logToken(yylex.Text(), "ANALYZE")
				return ANALYZE
			}
			continue
		case 39:
			{
				logToken(yylex.Text(), "AND")
				return AND
			}
			continue
		case 40:
			{
				logToken(yylex.Text(), "ANY")
				return ANY
			}
			continue
		case 41:
			{
				logToken(yylex.Text(), "ARRAY")
				return ARRAY
			}
			continue
		case 42:
			{
				logToken(yylex.Text(), "AS")
				lval.tokOffset = curOffset
				return AS
			}
			continue
		case 43:
			{
				logToken(yylex.Text(), "ASC")
				return ASC
			}
			continue
		case 44:
			{
				logToken(yylex.Text(), "BEGIN")
				return BEGIN
			}
			continue
		case 45:
			{
				logToken(yylex.Text(), "BETWEEN")
				return BETWEEN
			}
			continue
		case 46:
			{
				logToken(yylex.Text(), "BINARY")
				return BINARY
			}
			continue
		case 47:
			{
				logToken(yylex.Text(), "BOOLEAN")
				return BOOLEAN
			}
			continue
		case 48:
			{
				logToken(yylex.Text(), "BREAK")
				return BREAK
			}
			continue
		case 49:
			{
				logToken(yylex.Text(), "BUCKET")
				return BUCKET
			}
			continue
		case 50:
			{
				logToken(yylex.Text(), "BUILD")
				return BUILD
			}
			continue
		case 51:
			{
				logToken(yylex.Text(), "BY")
				return BY
			}
			continue
		case 52:
			{
				logToken(yylex.Text(), "CALL")
				return CALL
			}
			continue
		case 53:
			{
				logToken(yylex.Text(), "CASE")
				return CASE
			}
			continue
		case 54:
			{
				logToken(yylex.Text(), "CAST")
				return CAST
			}
			continue
		case 55:
			{
				logToken(yylex.Text(), "CLUSTER")
				return CLUSTER
			}
			continue
		case 56:
			{
				logToken(yylex.Text(), "COLLATE")
				return COLLATE
			}
			continue
		case 57:
			{
				logToken(yylex.Text(), "COLLECTION")
				return COLLECTION
			}
			continue
		case 58:
			{
				logToken(yylex.Text(), "COMMIT")
				return COMMIT
			}
			continue
		case 59:
			{
				logToken(yylex.Text(), "CONNECT")
				return CONNECT
			}
			continue
		case 60:
			{
				logToken(yylex.Text(), "CONTINUE")
				return CONTINUE
			}
			continue
		case 61:
			{
				logToken(yylex.Text(), "CORRELATE")
				return CORRELATE
			}
			continue
		case 62:
			{
				logToken(yylex.Text(), "COVER")
				return COVER
			}
			continue
		case 63:
			{
				logToken(yylex.Text(), "CREATE")
				return CREATE
			}
			continue
		case 64:
			{
				logToken(yylex.Text(), "DATABASE")
				return DATABASE
			}
			continue
		case 65:
			{
				logToken(yylex.Text(), "DATASET")
				return DATASET
			}
			continue
		case 66:
			{
				logToken(yylex.Text(), "DATASTORE")
				return DATASTORE
			}
			continue
		case 67:
			{
				logToken(yylex.Text(), "DECLARE")
				return DECLARE
			}
			continue
		case 68:
			{
				logToken(yylex.Text(), "DECREMENT")
				return DECREMENT
			}
			continue
		case 69:
			{
				logToken(yylex.Text(), "DELETE")
				return DELETE
			}
			continue
		case 70:
			{
				logToken(yylex.Text(), "DERIVED")
				return DERIVED
			}
			continue
		case 71:
			{
				logToken(yylex.Text(), "DESC")
				return DESC
			}
			continue
		case 72:
			{
				logToken(yylex.Text(), "DESCRIBE")
				return DESCRIBE
			}
			continue
		case 73:
			{
				logToken(yylex.Text(), "DISTINCT")
				return DISTINCT
			}
			continue
		case 74:
			{
				logToken(yylex.Text(), "DO")
				return DO
			}
			continue
		case 75:
			{
				logToken(yylex.Text(), "DROP")
				return DROP
			}
			continue
		case 76:
			{
				logToken(yylex.Text(), "EACH")
				return EACH
			}
			continue
		case 77:
			{
				logToken(yylex.Text(), "ELEMENT")
				return ELEMENT
			}
			continue
		case 78:
			{
				logToken(yylex.Text(), "ELSE")
				return ELSE
			}
			continue
		case 79:
			{
				logToken(yylex.Text(), "END")
				return END
			}
			continue
		case 80:
			{
				logToken(yylex.Text(), "EVERY")
				return EVERY
			}
			continue
		case 81:
			{
				logToken(yylex.Text(), "EXCEPT")
				return EXCEPT
			}
			continue
		case 82:
			{
				logToken(yylex.Text(), "EXCLUDE")
				return EXCLUDE
			}
			continue
		case 83:
			{
				logToken(yylex.Text(), "EXECUTE")
				return EXECUTE
			}
			continue
		case 84:
			{
				logToken(yylex.Text(), "EXISTS")
				return EXISTS
			}
			continue
		case 85:
			{
				logToken(yylex.Text(), "EXPLAIN")
				lval.tokOffset = curOffset
				return EXPLAIN
			}
			continue
		case 86:
			{
				logToken(yylex.Text(), "FALSE")
				return FALSE
			}
			continue
		case 87:
			{
				logToken(yylex.Text(), "FETCH")
				return FETCH
			}
			continue
		case 88:
			{
				logToken(yylex.Text(), "FIRST")
				return FIRST
			}
			continue
		case 89:
			{
				logToken(yylex.Text(), "FLATTEN")
				return FLATTEN
			}
			continue
		case 90:
			{
				logToken(yylex.Text(), "FOR")
				return FOR
			}
			continue
		case 91:
			{
				logToken(yylex.Text(), "FORCE")
				return FORCE
			}
			continue
		case 92:
			{
				logToken(yylex.Text(), "FROM")
				lval.tokOffset = curOffset
				return FROM
			}
			continue
		case 93:
			{
				logToken(yylex.Text(), "FUNCTION")
				return FUNCTION
			}
			continue
		case 94:
			{
				logToken(yylex.Text(), "GRANT")
				return GRANT
			}
			continue
		case 95:
			{
				logToken(yylex.Text(), "GROUP")
				return GROUP
			}
			continue
		case 96:
			{
				logToken(yylex.Text(), "GSI")
				return GSI
			}
			continue
		case 97:
			{
				logToken(yylex.Text(), "HAVING")
				return HAVING
			}
			continue
		case 98:
			{
				logToken(yylex.Text(), "IF")
				return IF
			}
			continue
		case 99:
			{
				logToken(yylex.Text(), "IGNORE")
				return IGNORE
			}
			continue
		case 100:
			{
				logToken(yylex.Text(), "ILIKE")
				return ILIKE
			}
			continue
		case 101:
			{
				logToken(yylex.Text(), "IN")
				return IN
			}
			continue
		case 102:
			{
				logToken(yylex.Text(), "INCLUDE")
				return INCLUDE
			}
			continue
		case 103:
			{
				logToken(yylex.Text(), "INCREMENT")
				return INCREMENT
			}
			continue
		case 104:
			{
				logToken(yylex.Text(), "INDEX")
				return INDEX
			}
			continue
		case 105:
			{
				logToken(yylex.Text(), "INFER")
				return INFER
			}
			continue
		case 106:
			{
				logToken(yylex.Text(), "INLINE")
				return INLINE
			}
			continue
		case 107:
			{
				logToken(yylex.Text(), "INNER")
				return INNER
			}
			continue
		case 108:
			{
				logToken(yylex.Text(), "INSERT")
				return INSERT
			}
			continue
		case 109:
			{
				logToken(yylex.Text(), "INTERSECT")
				return INTERSECT
			}
			continue
		case 110:
			{
				logToken(yylex.Text(), "INTO")
				return INTO
			}
			continue
		case 111:
			{
				logToken(yylex.Text(), "IS")
				return IS
			}
			continue
		case 112:
			{
				logToken(yylex.Text(), "JOIN")
				return JOIN
			}
			continue
		case 113:
			{
				logToken(yylex.Text(), "KEY")
				return KEY
			}
			continue
		case 114:
			{
				logToken(yylex.Text(), "KEYS")
				return KEYS
			}
			continue
		case 115:
			{
				logToken(yylex.Text(), "KEYSPACE")
				return KEYSPACE
			}
			continue
		case 116:
			{
				logToken(yylex.Text(), "KNOWN")
				return KNOWN
			}
			continue
		case 117:
			{
				logToken(yylex.Text(), "LAST")
				return LAST
			}
			continue
		case 118:
			{
				logToken(yylex.Text(), "LEFT")
				return LEFT
			}
			continue
		case 119:
			{
				logToken(yylex.Text(), "LET")
				return LET
			}
			continue
		case 120:
			{
				logToken(yylex.Text(), "LETTING")
				return LETTING
			}
			continue
		case 121:
			{
				logToken(yylex.Text(), "LIKE")
				return LIKE
			}
			continue
		case 122:
			{
				logToken(yylex.Text(), "LIMIT")
				return LIMIT
			}
			continue
		case 123:
			{
				logToken(yylex.Text(), "LSM")
				return LSM
			}
			continue
		case 124:
			{
				logToken(yylex.Text(), "MAP")
				return MAP
			}
			continue
		case 125:
			{
				logToken(yylex.Text(), "MAPPING")
				return MAPPING
			}
			continue
		case 126:
			{
				logToken(yylex.Text(), "MATCHED")
				return MATCHED
			}
			continue
		case 127:
			{
				logToken(yylex.Text(), "MATERIALIZED")
				return MATERIALIZED
			}
			continue
		case 128:
			{
				logToken(yylex.Text(), "MERGE")
				return MERGE
			}
			continue
		case 129:
			{
				logToken(yylex.Text(), "MINUS")
				return MINUS
			}
			continue
		case 130:
			{
				logToken(yylex.Text(), "MISSING")
				return MISSING
			}
			continue
		case 131:
			{
				logToken(yylex.Text(), "NAMESPACE")
				return NAMESPACE
			}
			continue
		case 132:
			{
				logToken(yylex.Text(), "NEST")
				return NEST
			}
			continue
		case 133:
			{
				logToken(yylex.Text(), "NOT")
				return NOT
			}
			continue
		case 134:
			{
				logToken(yylex.Text(), "NULL")
				return NULL
			}
			continue
		case 135:
			{
				logToken(yylex.Text(), "NUMBER")
				return NUMBER
			}
			continue
		case 136:
			{
				logToken(yylex.Text(), "OBJECT")
				return OBJECT
			}
			continue
		case 137:
			{
				logToken(yylex.Text(), "OFFSET")
				return OFFSET
			}
			continue
		case 138:
			{
				logToken(yylex.Text(), "ON")
				return ON
			}
			continue
		case 139:
			{
				logToken(yylex.Text(), "OPTION")
				return OPTION
			}
			continue
		case 140:
			{
				logToken(yylex.Text(), "OR")
				return OR
			}
			continue
		case 141:
			{
				logToken(yylex.Text(), "ORDER")
				return ORDER
			}
			continue
		case 142:
			{
				logToken(yylex.Text(), "OUTER")
				return OUTER
			}
			continue
		case 143:
			{
				logToken(yylex.Text(), "OVER")
				return OVER
			}
			continue
		case 144:
			{
				logToken(yylex.Text(), "PARSE")
				return PARSE
			}
			continue
		case 145:
			{
				logToken(yylex.Text(), "PARTITION")
				return PARTITION
			}
			continue
		case 146:
			{
				logToken(yylex.Text(), "PASSWORD")
				return PASSWORD
			}
			continue
		case 147:
			{
				logToken(yylex.Text(), "PATH")
				return PATH
			}
			continue
		case 148:
			{
				logToken(yylex.Text(), "POOL")
				return POOL
			}
			continue
		case 149:
			{
				logToken(yylex.Text(), "PREPARE")
				lval.tokOffset = curOffset
				return PREPARE
			}
			continue
		case 150:
			{
				logToken(yylex.Text(), "PRIMARY")
				return PRIMARY
			}
			continue
		case 151:
			{
				logToken(yylex.Text(), "PRIVATE")
				return PRIVATE
			}
			continue
		case 152:
			{
				logToken(yylex.Text(), "PRIVILEGE")
				return PRIVILEGE
			}
			continue
		case 153:
			{
				logToken(yylex.Text(), "PROCEDURE")
				return PROCEDURE
			}
			continue
		case 154:
			{
				logToken(yylex.Text(), "PUBLIC")
				return PUBLIC
			}
			continue
		case 155:
			{
				logToken(yylex.Text(), "RAW")
				return RAW
			}
			continue
		case 156:
			{
				logToken(yylex.Text(), "REALM")
				return REALM
			}
			continue
		case 157:
			{
				logToken(yylex.Text(), "REDUCE")
				return REDUCE
			}
			continue
		case 158:
			{
				logToken(yylex.Text(), "RENAME")
				return RENAME
			}
			continue
		case 159:
			{
				logToken(yylex.Text(), "RETURN")
				return RETURN
			}
			continue
		case 160:
			{
				logToken(yylex.Text(), "RETURNING")
				return RETURNING
			}
			continue
		case 161:
			{
				logToken(yylex.Text(), "REVOKE")
				return REVOKE
			}
			continue
		case 162:
			{
				logToken(yylex.Text(), "RIGHT")
				return RIGHT
			}
			continue
		case 163:
			{
				logToken(yylex.Text(), "ROLE")
				return ROLE
			}
			continue
		case 164:
			{
				logToken(yylex.Text(), "ROLLBACK")
				return ROLLBACK
			}
			continue
		case 165:
			{
				logToken(yylex.Text(), "SATISFIES")
				return SATISFIES
			}
			continue
		case 166:
			{
				logToken(yylex.Text(), "SCHEMA")
				return SCHEMA
			}
			continue
		case 167:
			{
				logToken(yylex.Text(), "SELECT")
				return SELECT
			}
			continue
		case 168:
			{
				logToken(yylex.Text(), "SELF")
				return SELF
			}
			continue
		case 169:
			{
				logToken(yylex.Text(), "SET")
				return SET
			}
			continue
		case 170:
			{
				logToken(yylex.Text(), "SHOW")
				return SHOW
			}
			continue
		case 171:
			{
				logToken(yylex.Text(), "SOME")
				return SOME
			}
			continue
		case 172:
			{
				logToken(yylex.Text(), "START")
				return START
			}
			continue
		case 173:
			{
				logToken(yylex.Text(), "STATISTICS")
				return STATISTICS
			}
			continue
		case 174:
			{
				logToken(yylex.Text(), "STRING")
				return STRING
			}
			continue
		case 175:
			{
				logToken(yylex.Text(), "SYSTEM")
				return SYSTEM
			}
			continue
		case 176:
			{
				logToken(yylex.Text(), "THEN")
				return THEN
			}
			continue
		case 177:
			{
				logToken(yylex.Text(), "TO")
				return TO
			}
			continue
		case 178:
			{
				logToken(yylex.Text(), "TRANSACTION")
				return TRANSACTION
			}
			continue
		case 179:
			{
				logToken(yylex.Text(), "TRIGGER")
				return TRIGGER
			}
			continue
		case 180:
			{
				logToken(yylex.Text(), "TRUE")
				return TRUE
			}
			continue
		case 181:
			{
				logToken(yylex.Text(), "TRUNCATE")
				return TRUNCATE
			}
			continue
		case 182:
			{
				logToken(yylex.Text(), "UNDER")
				return UNDER
			}
			continue
		case 183:
			{
				logToken(yylex.Text(), "UNION")
				return UNION
			}
			continue
		case 184:
			{
				logToken(yylex.Text(), "UNIQUE")
				return UNIQUE
			}
			continue
		case 185:
			{
				logToken(yylex.Text(), "UNKNOWN")
				return UNKNOWN
			}
			continue
		case 186:
			{
				logToken(yylex.Text(), "UNNEST")
				return UNNEST
			}
			continue
		case 187:
			{
				logToken(yylex.Text(), "UNSET")
				return UNSET
			}
			continue
		case 188:
			{
				logToken(yylex.Text(), "UPDATE")
				return UPDATE
			}
			continue
		case 189:
			{
				logToken(yylex.Text(), "UPSERT")
				return UPSERT
			}
			continue
		case 190:
			{
				logToken(yylex.Text(), "USE")
				return USE
			}
			continue
		case 191:
			{
				logToken(yylex.Text(), "USER")
				return USER
			}
			continue
		case 192:
			{
				logToken(yylex.Text(), "USING")
				return USING
			}
			continue
		case 193:
			{
				logToken(yylex.Text(), "VALIDATE")
				return VALIDATE
			}
			continue
		case 194:
			{
				logToken(yylex.Text(), "VALUE")
				return VALUE
			}
			continue
		case 195:
			{
				logToken(yylex.Text(), "VALUED")
				return VALUED
			}
			continue
		case 196:
			{
				logToken(yylex.Text(), "VALUES")
				return VALUES
			}
			continue
		case 197:
			{
				logToken(yylex.Text(), "VIA")
				return VIA
			}
			continue
		case 198:
			{
				logToken(yylex.Text(), "VIEW")
				return VIEW
			}
			continue
		case 199:
			{
				logToken(yylex.Text(), "WHEN")
				return WHEN
			}
			continue
		case 200:
			{
				logToken(yylex.Text(), "WHERE")
				return WHERE
			}
			continue
		case 201:
			{
				logToken(yylex.Text(), "WHILE")
				return WHILE
			}
			continue
		case 202:
			{
				logToken(yylex.Text(), "WITH")
				return WITH
			}
			continue
		case 203:
			{
				logToken(yylex.Text(), "WITHIN")
				return WITHIN
			}
			continue
		case 204:
			{
				logToken(yylex.Text(), "WORK")
				return WORK
			}
			continue
		case 205:
			{
				logToken(yylex.Text(), "XOR")
				return XOR
			}
			continue
		case 206:
			{
				lval.s = yylex.Text()
				logToken(yylex.Text(), "IDENT - %s", lval.s)
				return IDENT
			}
			continue
		case 207:
			{
				lval.s = yylex.Text()[1:]
				logToken(yylex.Text(), "NAMED_PARAM - %s", lval.s)
				return NAMED_PARAM
			}
			continue
		case 208:
			{
				lval.n, _ = strconv.ParseInt(yylex.Text()[1:], 10, 64)
				logToken(yylex.Text(), "POSITIONAL_PARAM - %d", lval.n)
				return POSITIONAL_PARAM
			}
			continue
		case 209:
			{
				lval.n = 0 // Handled by parser
				logToken(yylex.Text(), "NEXT_PARAM - ?")
				return NEXT_PARAM
			}
			continue
		case 210:
			{
				curOffset++
			}
			continue
		case 211:
			{
				curOffset++
			}
			continue
		}
		break
	}
	yylex.pop()

	return 0
}

var curOffset int

func logToken(text string, format string, v ...interface{}) {
	curOffset += len(text)
	clog.To("LEXER", format, v...)
}

func (this *Lexer) ResetOffset() {
	curOffset = 0
}
