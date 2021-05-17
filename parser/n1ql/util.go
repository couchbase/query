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
	"errors"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
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
				return unicode.ReplacementChar, errors.New("invalid escape sequence")
			}
		}
		if b < 0 || b > 15 {
			return unicode.ReplacementChar, errors.New("invalid escape sequence")
		}
		rn = rn << 4
		rn = rn | rune(b)
	}
	return rn, nil
}

// Handle permitted JSON escape sequences along with appropriate quotation mark escaping (allows '' as an escaped ')
// Ref: https://www.json.org/json-en.html
func ProcessEscapeSequences(s string) (t string, e error) {
	b := make([]byte, len(s)-2)
	w := 0
	for r := 1; r < len(s)-1; {
		c := s[r]
		r++
		if c == '\\' {
			if r >= len(s)-1 {
				return t, errors.New("invalid escape sequence")
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
			case 'u':
				if r+4 > len(s)-1 {
					return t, errors.New("invalid escape sequence")
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
				return t, errors.New("invalid escape sequence")
			}
		} else {
			// we don't need validate UTF8 sequences so we can simply copy them byte-by-byte
			b[w] = c
			w++
			// if single quoted, allow '' as an escaped single quote
			if s[0] == '\'' && c == '\'' {
				if r >= len(s)-1 || s[r] != '\'' {
					return t, errors.New("unescaped embedded quote")
				}
				r++
			} else if s[0] == c {
				return t, errors.New("unescaped embedded quote")
			}
		}
	}
	b = b[:w]
	// this is what strings.Builder.String() does... https://golang.org/src/strings/builder.go?s=1395:1428#L37
	t = *(*string)(unsafe.Pointer(&b))
	return t, nil
}
