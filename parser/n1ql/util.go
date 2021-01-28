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
	"unicode/utf8"
	"unsafe"
)

// Handle permitted JSON escape sequences along with appropriate quotation mark escaping
// Ref: https://www.json.org/json-en.html
func ProcessEscapeSequences(s string) (t string, e error) {
	b := make([]byte, len(s)-2)
	w := 0
	bq := false
	for r := 1; r < len(s)-1; r++ {
		c := s[r]
		if bq {
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
				// next 4 chars are hex digits which we build a UTF-8 encoded rune from
				r++
				if r+4 > len(s)-1 {
					return t, errors.New("invalid unicode escape sequence")
				}
				var rn rune = rune(0x0)
				for end := r + 4; r < end; r++ {
					b := byte(s[r]) - '0'
					if b > 9 {
						b = b - 7
						if b > 15 {
							b = b - 32
						}
						if b < 10 {
							return t, errors.New("invalid unicode escape sequence")
						}
					}
					if b < 0 || b > 15 {
						return t, errors.New("invalid unicode escape sequence")
					}
					rn = rn << 4
					rn = rn | rune(b)
				}
				buf := make([]byte, utf8.UTFMax)
				n := utf8.EncodeRune(buf, rn)
				for i := 0; i < n; i++ {
					b[w] = buf[i]
					w++
				}
			default:
				return t, errors.New("invalid escape sequence")
			}
			bq = false
		} else if c == '\\' {
			bq = true
		} else {
			b[w] = c
			w++
			// if single quoted, allow '' as an escaped single quote
			if s[0] == '\'' && c == '\'' {
				r++
				if r >= len(s)-1 || s[r] != '\'' {
					return t, errors.New("unescaped embedded quote")
				}
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
