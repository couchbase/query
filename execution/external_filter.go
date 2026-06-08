//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"sync"

	"github.com/couchbase/query/expression"
)

type ExternalFilterTerm struct {
	term   expression.Expression
	values []interface{}
}

type ExternalFilterTerms struct {
	sync.RWMutex
	terms map[string]*ExternalFilterTerm
}

func newExternalFilterTerms(nterms int) *ExternalFilterTerms {
	return &ExternalFilterTerms{
		terms: make(map[string]*ExternalFilterTerm, nterms),
	}
}

func (this *ExternalFilterTerms) addTermValues(term expression.Expression, vals []interface{}) {
	termStr := term.String()
	this.Lock()
	if curTerm, ok := this.terms[termStr]; ok {
		curTerm.values = append(curTerm.values, vals...)
	} else {
		this.terms[termStr] = &ExternalFilterTerm{
			term:   term,
			values: vals,
		}
	}
	this.Unlock()
}

func (this *ExternalFilterTerms) getExternalFilters() expression.Expression {
	this.RLock()
	filters := make(expression.Expressions, 0, len(this.terms))
	for _, curTerm := range this.terms {
		// dedup and sort first
		vals := expression.SortValArr(curTerm.values)
		newFilter := expression.NewIn(curTerm.term, expression.NewConstant(vals))
		filters = append(filters, newFilter)
	}
	this.RUnlock()
	if len(filters) == 0 {
		return nil
	} else if len(filters) == 1 {
		return filters[0]
	}
	return expression.NewAnd(filters...)
}

func (this *ExternalFilterTerms) clearExternalFilters() {
	this.Lock()
	for _, term := range this.terms {
		if term.term != nil {
			delete(this.terms, term.term.String())
		}
	}
	this.terms = nil
	this.Unlock()
}
