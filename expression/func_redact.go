//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

type Redact struct {
	FunctionBase
	filters []*redactFilter
}

func NewRedact(operands ...Expression) Function {
	rv := &Redact{}
	rv.Init("redact", operands...)

	rv.expr = rv
	return rv
}

func (this *Redact) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Redact) Type() value.Type { return value.OBJECT }

type redactFilter struct {
	re         *regexp.Regexp
	name       value.Tristate
	ignorecase value.Tristate
	strict     value.Tristate
	omit       value.Tristate
	fixedlen   value.Tristate
	mask       string
	exclude    value.Tristate
}

func (this *Redact) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() == value.NULL || arg.Type() == value.BINARY {
		return value.NULL_VALUE, nil
	}

	missing := false
	null := false
	this.filters = nil

	for n := 1; n < len(this.operands); n++ {
		options, err := this.operands[n].Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if options.Type() == value.MISSING {
			missing = true
		} else if options.Type() != value.OBJECT {
			null = true
		}
		if missing || null {
			continue
		}

		rf := &redactFilter{mask: "x"}
		if nm, ok := options.Field("name"); ok && nm.Type() == value.BOOLEAN {
			if nm.Truth() {
				rf.name = value.TRUE
			} else {
				rf.name = value.FALSE
			}
		}
		if i, ok := options.Field("ignorecase"); ok && i.Type() == value.BOOLEAN {
			if i.Truth() {
				rf.ignorecase = value.TRUE
			} else {
				rf.ignorecase = value.FALSE
			}
		}
		if s, ok := options.Field("strict"); ok && s.Type() == value.BOOLEAN {
			if s.Truth() {
				rf.strict = value.TRUE
			} else {
				rf.strict = value.FALSE
			}
		}
		if s, ok := options.Field("omit"); ok && s.Type() == value.BOOLEAN {
			if s.Truth() {
				rf.omit = value.TRUE
			} else {
				rf.omit = value.FALSE
			}
		}
		if s, ok := options.Field("mask"); ok && s.Type() == value.STRING {
			rf.mask = s.ToString()
		}
		if s, ok := options.Field("fixedlength"); ok && s.Type() == value.BOOLEAN {
			if s.Truth() {
				rf.fixedlen = value.TRUE
			} else {
				rf.fixedlen = value.FALSE
			}
		}

		if exclude, ok := options.Field("exclude"); ok && exclude.Type() == value.BOOLEAN {
			if exclude.Truth() {
				rf.exclude = value.TRUE
			} else {
				rf.exclude = value.FALSE
			}
		}

		if p, ok := options.Field("pattern"); ok {
			pattern := p.ToString()
			if len(pattern) > 0 {
				if rex, ok := options.Field("regex"); ok && rex.Type() == value.BOOLEAN {
					if !rex.Truth() {
						pattern = regexp.QuoteMeta(pattern)
					}
				}
				if e, ok := options.Field("exact"); ok && e.Type() == value.BOOLEAN {
					if e.Truth() {
						r := strings.NewReader(pattern)
						var w strings.Builder
						w.WriteRune('^')
						escaped := false
						for {
							ru, _, err := r.ReadRune()
							if err != nil {
								break
							}
							if escaped {
								escaped = false
							} else if ru == '\\' {
								escaped = true
							} else if ru == '|' {
								ru, _, err = r.ReadRune()
								if err != nil {
									break
								}
								// doesn't matter if we double up on the end anchor; this way we don't have to care about escaping
								w.WriteRune('$')
								w.WriteRune('|')
								if ru != '^' {
									w.WriteRune('^')
								}
							}
							w.WriteRune(ru)
						}
						w.WriteRune('$')
						pattern = w.String()
					}
				}
				if rf.ignorecase == value.TRUE {
					pattern = strings.ToLower(pattern)
				}
				rf.re, err = regexp.Compile(pattern)
				if err != nil {
					return nil, err
				}
				this.filters = append(this.filters, rf)
			}
		} else {
			this.filters = append(this.filters, rf)
		}
	}
	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}
	if len(this.filters) == 0 {
		this.filters = append(this.filters, &redactFilter{mask: "x"})
	}

	v := arg.Actual()
	var redactV, redactN, strict, omit, fixedLen bool
	var mask string
	switch v.(type) {
	case []interface{}:
	case map[string]interface{}:
		redactV, redactN, strict, omit, fixedLen, mask = false, false, false, false, false, "x"
	default:
		redactV, redactN, strict, omit, fixedLen, mask = this.shouldRedact("", false, false, false, false, "x")
	}

	if !omit {
		return value.NewValue(this.redact(v, "", redactV, redactN, strict, fixedLen, mask)), nil
	}
	return value.NewValue(v), nil

}

func (this *Redact) redact(v interface{}, base string, redactV, redactN, strict, fixedLen bool, mask string) interface{} {
	switch v := v.(type) {
	case []interface{}:
		n := make([]interface{}, len(v))
		for i, v1 := range v {
			vact := value.NewValue(v1)
			nredactV, nredactN, nstrict, omit, nfixedLen, nmask := this.shouldRedact("", redactV, redactN, strict, fixedLen, mask)
			if omit {
				continue
			}
			n[i] = value.NewValue(this.redact(vact.Actual(), base, nredactV, nredactN, nstrict, nfixedLen, nmask))
		}
		return n
	case map[string]interface{}:
		o := make(map[string]interface{}, len(v))
		i := 0
		names := make([]string, len(v))
		for k, _ := range v {
			names[i] = k
			i++
		}
		sort.Strings(names)
		for i = range names {
			v1 := v[names[i]]
			vact := value.NewValue(v1)
			nredactV, nredactN, nstrict, omit, nfixedLen, nmask := this.shouldRedact(names[i], redactV, redactN, strict, fixedLen, mask)
			if omit {
				continue
			}
			nk := names[i]
			if nredactN {
				if len(base) > 0 {
					nk = fmt.Sprintf("%s_f%04d", base, i)
				} else {
					nk = fmt.Sprintf("f%04d", i)
				}
			}
			o[nk] = value.NewValue(this.redact(vact.Actual(), nk, nredactV, nredactN, nstrict, nfixedLen, nmask))
		}
		return o
	case string:
		if !redactV {
			return v
		}
		w := strings.Builder{}
		r := strings.NewReader(v)
		subs := mask
		if !strict && len(v) <= 30 {
			// if it is a date then redact with a numeral
			if _, err := strToTimeTryAllDefaultFormats(v); err == nil {
				subs = "1"
				fixedLen = false
			}
		}
		if fixedLen {
			return mask
		}
		n := 0
		for {
			ru, _, err := r.ReadRune()
			if err != nil {
				break
			}
			if strict || !unicode.In(ru, unicode.Punct, unicode.Space, unicode.Symbol) {
				if n < len(subs) {
					w.WriteRune(rune(subs[n]))
				}
			} else {
				w.WriteRune(ru)
			}
			n++
			if n == len(subs) {
				n = 0
			}
		}
		return w.String()
	case int:
		if !redactV {
			return v
		}
		if fixedLen {
			return 0
		}
		s := []byte(fmt.Sprintf("%v", v))
		for i := range s {
			if s[i] >= '0' && s[i] <= '9' {
				s[i] = '1'
			}
		}
		rv, _ := strconv.Atoi(string(s))
		return rv
	case int64:
		if !redactV {
			return v
		}
		if fixedLen {
			return 0
		}
		s := []byte(fmt.Sprintf("%v", v))
		for i := range s {
			if s[i] >= '0' && s[i] <= '9' {
				s[i] = '1'
			}
		}
		rv, _ := strconv.ParseInt(string(s), 10, 64)
		return rv
	case float64:
		if !redactV {
			return v
		}
		if fixedLen {
			return 0.0
		}
		s := []byte(fmt.Sprintf("%g", v))
		for i := range s {
			if s[i] >= '0' && s[i] <= '9' {
				s[i] = '1'
			}
		}
		rv, _ := strconv.ParseFloat(string(s), 10)
		return rv
	case bool:
		if !redactV {
			return v
		}
		if strict || fixedLen {
			return false
		}
		return v
	case nil:
		return nil
	default:
		logging.Debugf("Redact: unhandled type %T", v)
		return v
	}
}

func (this *Redact) shouldRedact(name string, defV bool, defN bool, defStrict bool, defFixedLen bool, defMask string) (
	bool, bool, bool, bool, bool, string) {

	if len(this.filters) == 0 {
		return defV, defN, defStrict, false, defFixedLen, defMask
	}
	for i := range this.filters {
		n := name

		exclude := value.ToBool(this.filters[i].exclude)
		if this.filters[i].ignorecase == value.TRUE {
			n = strings.ToLower(n)
		}

		redactN := false
		switch this.filters[i].name {
		case value.TRUE:
			redactN = true
		case value.FALSE:
			redactN = false
		default:
			redactN = defN
		}

		s := false
		switch this.filters[i].strict {
		case value.TRUE:
			s = true
		case value.FALSE:
			s = false
		default:
			s = defStrict
		}

		if this.filters[i].re == nil || this.filters[i].re.MatchString(n) {
			return !exclude, redactN && !exclude, s, (this.filters[i].omit == value.TRUE),
				(this.filters[i].fixedlen == value.TRUE), this.filters[i].mask
		} else if exclude == true {
			return true, redactN, s, false, (this.filters[i].fixedlen == value.TRUE), this.filters[i].mask
		}
	}
	return defV, defN, defStrict, false, defFixedLen, defMask
}

func (this *Redact) MinArgs() int { return 1 }

func (this *Redact) MaxArgs() int { return math.MaxInt16 }

func (this *Redact) Constructor() FunctionConstructor {
	return NewRedact
}
