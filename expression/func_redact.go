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
	rv := &Redact{
		*NewFunctionBase("redact", operands...),
		nil,
	}

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
}

func (this *Redact) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
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

		rf := &redactFilter{}
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
								ru, _, err := r.ReadRune()
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
		this.filters = append(this.filters, &redactFilter{})
	}

	n := make(map[string]interface{})
	this.redact(arg, n, "", false, false, false)

	return value.NewValue(n), nil
}

func (this *Redact) redact(v value.Value, n map[string]interface{}, base string, defRedactV bool, defRedactN bool, defStrict bool) {
	flds := v.Fields()
	if flds == nil {
		return
	}
	i := 0
	names := make([]string, len(flds))
	for k, _ := range flds {
		names[i] = k
		i++
	}
	sort.Strings(names)
	for i = range names {

		redactV, redactN, strict, omit := this.shouldRedact(names[i], defRedactV, defRedactN, defStrict)

		if omit {
			continue
		}

		var nk string
		if redactN {
			if len(base) > 0 {
				nk = fmt.Sprintf("%s_f%04d", base, i)
			} else {
				nk = fmt.Sprintf("f%04d", i)
			}
		} else {
			nk = names[i]
		}

		v := value.NewValue(flds[names[i]])

		switch v.Type() {
		case value.OBJECT:
			nn := make(map[string]interface{})
			this.redact(v, nn, nk, redactV, redactN, strict)
			n[nk] = value.NewValue(nn)
		case value.ARRAY:
			act := v.Actual().([]interface{})
			nn := make([]interface{}, len(act))
			for i := range act {
				av := value.NewValue(act[i])
				if av.Type() == value.OBJECT || av.Type() == value.ARRAY {
					nm := make(map[string]interface{})
					this.redact(value.NewValue(act[i]), nm, nk, redactV, redactN, strict)
					nn[i] = value.NewValue(nm)
				} else {
					if redactV {
						nn[i] = this.redactValue(av.Actual(), strict)
					} else {
						nn[i] = av
					}
				}
			}
			n[nk] = value.NewValue(nn)
		case value.NUMBER:
			if redactV {
				if i, ok := value.IsIntValue(v); ok {
					n[nk] = this.redactValue(i, strict)
				} else {
					n[nk] = this.redactValue(v.Actual(), strict)
				}
			} else {
				n[nk] = v
			}
		default:
			if redactV {
				n[nk] = this.redactValue(v.Actual(), strict)
			} else {
				n[nk] = v
			}
		}
	}
}

func (this *Redact) shouldRedact(name string, defV bool, defN bool, defStrict bool) (bool, bool, bool, bool) {
	if len(this.filters) == 0 {
		return defV, defN, defStrict, false
	}
	for i := range this.filters {
		n := name
		if this.filters[i].ignorecase == value.TRUE {
			n = strings.ToLower(n)
		}
		if this.filters[i].re == nil || this.filters[i].re.MatchString(n) {
			n := false
			switch this.filters[i].name {
			case value.TRUE:
				n = true
			case value.FALSE:
				n = false
			default:
				n = defN
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
			return true, n, s, (this.filters[i].omit == value.TRUE)
		}
	}
	return defV, defN, defStrict, false
}

func (this *Redact) redactValue(v interface{}, strict bool) interface{} {
	switch v := v.(type) {
	case string:
		w := strings.Builder{}
		r := strings.NewReader(v)
		subs := rune('x')
		if !strict && len(v) <= 30 {
			// if it is a date then redact with a numeral
			if _, err := strToTimeTryAllDefaultFormats(v); err == nil {
				subs = rune('1')
			}
		}
		for {
			ru, _, err := r.ReadRune()
			if err != nil {
				break
			}
			if strict || !unicode.In(ru, unicode.Punct, unicode.Space, unicode.Symbol) {
				w.WriteRune(subs)
			} else {
				w.WriteRune(ru)
			}
		}
		return w.String()
	case int:
		s := []byte(fmt.Sprintf("%v", v))
		for i := range s {
			if s[i] >= '0' && s[i] <= '9' {
				s[i] = '1'
			}
		}
		rv, _ := strconv.Atoi(string(s))
		return rv
	case int64:
		s := []byte(fmt.Sprintf("%v", v))
		for i := range s {
			if s[i] >= '0' && s[i] <= '9' {
				s[i] = '1'
			}
		}
		rv, _ := strconv.ParseInt(string(s), 10, 64)
		return rv
	case float64:
		s := []byte(fmt.Sprintf("%g", v))
		for i := range s {
			if s[i] >= '0' && s[i] <= '9' {
				s[i] = '1'
			}
		}
		rv, _ := strconv.ParseFloat(string(s), 10)
		return rv
	case bool:
		if strict {
			return false
		}
		return v
	case nil:
		return nil
	default:
		logging.Infof("DVJH: unhandled type %T", v)
		return v
	}
}

func (this *Redact) MinArgs() int { return 1 }

func (this *Redact) MaxArgs() int { return math.MaxInt16 }

func (this *Redact) Constructor() FunctionConstructor {
	return NewRedact
}
