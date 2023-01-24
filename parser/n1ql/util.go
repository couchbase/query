//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package n1ql

import (
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"

	"github.com/couchbase/query/errors"
)

// Extract first 4 characters in provided string as (ASCII) hex digits of a UTF16 encoding
func getUTF16Rune(s string) (rune, error) {
	var rn rune = rune(0x0)
	for _, b := range []byte(s[0:4]) {
		b -= '0'
		if b > 9 {
			b = b - 7
			if b > 15 {
				b = b - 32
			}
			if b < 10 {
				return unicode.ReplacementChar, errors.NewParseInvalidEscapeSequenceError()
			}
		}
		if b < 0 || b > 15 {
			return unicode.ReplacementChar, errors.NewParseInvalidEscapeSequenceError()
		}
		rn = rn << 4
		rn = rn | rune(b)
	}
	return rn, nil
}

// Handle permitted JSON escape sequences along with appropriate quotation mark escaping (allows ” as an escaped ')
// Ref: https://www.json.org/json-en.html
func ProcessEscapeSequences(s string) (t string, e error) {
	if len(s) < 2 {
		return t, errors.NewParseInvalidStringError()
	}
	if s[0] != s[len(s)-1] {
		return t, errors.NewParseMissingClosingQuoteError()
	}
	b := make([]byte, len(s)-2)
	w := 0
	for r := 1; r < len(s)-1; {
		c := s[r]
		r++
		if c == '\\' {
			if r == len(s)-1 {
				return t, errors.NewParseMissingClosingQuoteError()
			}
			c = s[r]
			r++
			switch c {
			case 'b':
				b[w] = '\b'
				w++
			case 'f':
				b[w] = '\f'
				w++
			case 'n':
				b[w] = '\n'
				w++
			case 'r':
				b[w] = '\r'
				w++
			case 't':
				b[w] = '\t'
				w++
			case '/':
				b[w] = c
				w++
			case '\\':
				b[w] = c
				w++
			case '"':
				b[w] = c
				w++
			case '\'':
				b[w] = c
				w++
			case '`':
				b[w] = c
				w++
			case 'u':
				if r+4 > len(s)-1 {
					return t, errors.NewParseInvalidEscapeSequenceError()
				}
				rn, err := getUTF16Rune(s[r:])
				if err != nil {
					return t, err
				}
				r += 4
				if utf16.IsSurrogate(rn) {
					if r+6 > len(s)-1 || s[r] != '\\' || s[r+1] != 'u' {
						rn = unicode.ReplacementChar
					} else {
						r += 2
						rn2, err := getUTF16Rune(s[r:])
						if err != nil {
							return t, err
						}
						r += 4
						// returns unicode.ReplacementChar if not a valid surrogate which we'll use instead
						rn = utf16.DecodeRune(rn, rn2)
					}
				}
				w += utf8.EncodeRune(b[w:], rn)
			default:
				return t, errors.NewParseInvalidEscapeSequenceError()
			}
		} else {
			// we don't need validate UTF8 sequences so we can simply copy them byte-by-byte
			b[w] = c
			w++
			// if single quoted, allow '' as an escaped single quote
			if s[0] == '\'' && c == '\'' {
				if r >= len(s)-1 || s[r] != '\'' {
					return t, errors.NewParseUnescapedEmbeddedQuoteError()
				}
				r++
			} else if s[0] == '`' && c == '`' { // same for back quote
				if r >= len(s)-1 || s[r] != '`' {
					return t, errors.NewParseUnescapedEmbeddedQuoteError()
				}
				r++
			} else if s[0] == c {
				return t, errors.NewParseUnescapedEmbeddedQuoteError()
			}
		}
	}
	b = b[:w]
	// this is what strings.Builder.String() does... https://golang.org/src/strings/builder.go?s=1395:1428#L37
	t = *(*string)(unsafe.Pointer(&b))
	return t, nil
}
