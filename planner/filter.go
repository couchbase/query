//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

type Filter struct {
	fltrExpr  expression.Expression // filter expression
	origExpr  expression.Expression // original filter expression
	keyspaces map[string]bool
}

type Filters []*Filter

func newFilter(fltrExpr, origExpr expression.Expression, keyspaces map[string]bool) *Filter {
	rv := &Filter{
		fltrExpr:  fltrExpr,
		origExpr:  origExpr,
		keyspaces: keyspaces,
	}

	return rv
}

// Combine an array of filters into a single expression by ANDing each filter expression,
// perform transformation on each filter, and if an OR filter is involved, perform DNF
// transformation on the combined filter
func combineFilters(filters Filters) (expression.Expression, expression.Expression, error) {
	var err error
	var hasOr bool = false
	var dnfPred, origPred expression.Expression

	for _, fl := range filters {
		if dnfPred == nil {
			dnfPred = fl.fltrExpr
		} else {
			dnfPred = expression.NewAnd(dnfPred, fl.fltrExpr)
		}

		if fl.origExpr != nil {
			if origPred == nil {
				origPred = fl.origExpr
			} else {
				origPred = expression.NewAnd(origPred, fl.origExpr)
			}
		}

		if _, ok := fl.fltrExpr.(*expression.Or); ok {
			hasOr = true
		}
	}

	if hasOr {
		dnf := NewDNF(dnfPred.Copy(), true, true)
		dnfPred, err = dnf.Map(dnfPred)
		if err != nil {
			return nil, nil, err
		}
	}

	return dnfPred, origPred, nil
}

// Once a keyspace has been visited, join filters referring to this keyspace can remove
// this keyspace reference since it's now "available", and if there are no other
// keyspace references the join filter can be moved to filters
func (this *builder) moveJoinFilters(keyspace string, baseKeyspace *baseKeyspace) error {
	compact := false
	for i, jfl := range baseKeyspace.joinfilters {
		delete(jfl.keyspaces, keyspace)
		if len(jfl.keyspaces) == 1 {
			for ksName, _ := range jfl.keyspaces {
				if baseKeyspace.name != ksName {
					return errors.NewPlanInternalError(fmt.Sprintf("moveJoinFilters: keyspace mismatch: %s vs %s", baseKeyspace.name, ksName))
				}
				break
			}

			// move to filters
			baseKeyspace.filters = append(baseKeyspace.filters, jfl)
			baseKeyspace.joinfilters[i] = nil
			compact = true
		}
	}

	if compact == true {
		curlen := len(baseKeyspace.joinfilters)
		newlen := curlen
		for i := 0; i < curlen; i++ {
			if i >= newlen {
				break
			}
			if baseKeyspace.joinfilters[i] == nil {
				if i < newlen-1 {
					baseKeyspace.joinfilters[i] = baseKeyspace.joinfilters[newlen-1]
				}
				baseKeyspace.joinfilters[newlen-1] = nil
				newlen--
			}
		}
		baseKeyspace.joinfilters = baseKeyspace.joinfilters[:newlen]
	}

	return nil
}

func (this *builder) processKeyspaceDone(keyspace string) error {
	var err error
	for _, baseKeyspace := range this.baseKeyspaces {
		if keyspace == baseKeyspace.name {
			continue
		}

		err = this.moveJoinFilters(keyspace, baseKeyspace)
		if err != nil {
			return err
		}
	}

	return nil
}
