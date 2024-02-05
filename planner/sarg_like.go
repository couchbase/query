//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"math"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
)

func (this *sarg) visitLike(pred expression.LikeFunction) (interface{}, error) {
	if this.isVector {
		return nil, nil
	}

	if len(this.context.NamedArgs()) > 0 || len(this.context.PositionalArgs()) > 0 {
		replaced, err := base.ReplaceParameters(pred, this.context.NamedArgs(), this.context.PositionalArgs())
		if err != nil {
			return nil, err
		}
		if repFunc, ok := replaced.(expression.LikeFunction); ok {
			pred = repFunc
		}
	}

	prefix := ""
	re := pred.Regexp()

	if re != nil {
		var complete bool
		prefix, complete = re.LiteralPrefix()
		if complete {
			eq := expression.NewEq(pred.First(), expression.NewConstant(prefix))
			return eq.Accept(this)
		}
	}

	if base.SubsetOf(pred, this.key) {
		if expression.Equivalent(pred, this.key) {
			return _EXACT_SELF_SPANS, nil
		}
		return _SELF_SPANS, nil
	}

	if !pred.First().EquivalentTo(this.key) {
		if pred.DependsOn(this.key) {
			return _VALUED_SPANS, nil
		} else {
			return nil, nil
		}
	}

	if re == nil {
		selec := OPT_SELEC_NOT_AVAIL
		if this.doSelec {
			selec = optDefLikeSelec(this.baseKeyspace.Keyspace(), this.key.String(), this.advisorValidate)
		}
		return likeSpans(pred, selec), nil
	}

	exact := false
	range2 := &plan.Range2{}
	range2.Low = expression.NewConstant(prefix)

	last := len(prefix) - 1
	if last >= 0 && prefix[last] < math.MaxUint8 {
		bytes := []byte(prefix)
		bytes[last]++
		range2.High = expression.NewConstant(string(bytes))
		if re.NumSubexp() == 1 && re.String()[len(prefix):] == "(.*)" {
			exact = true
		}
	} else {
		if last < 0 {
			range2.SetFlag(plan.RANGE_DEFAULT_LIKE)
		}
		range2.High = expression.EMPTY_ARRAY_EXPR
	}

	selec := this.getSelec(pred)
	range2.Inclusion = datastore.LOW
	range2.Selec1 = selec
	range2.Selec2 = OPT_SELEC_NOT_AVAIL
	span := plan.NewSpan2(nil, plan.Ranges2{range2}, exact)
	return NewTermSpans(span), nil
}

func likeSpans(pred expression.LikeFunction, selec float64) SargSpans {
	range2 := plan.NewRange2(expression.EMPTY_STRING_EXPR, expression.EMPTY_ARRAY_EXPR, datastore.LOW, selec,
		OPT_SELEC_NOT_AVAIL, 0)
	range2.SetFlag(plan.RANGE_DEFAULT_LIKE)

	switch pred := pred.(type) {
	case *expression.Like:
		if pred.Second().Static() != nil && pred.Escape().Static() != nil {
			range2.Low = expression.NewLikePrefix(pred.Second(), pred.Escape().Static())
			range2.High = expression.NewLikeStop(pred.Second(), pred.Escape().Static())
		}
	case *expression.RegexpLike:
		if pred.Second().Static() != nil {
			range2.Low = expression.NewRegexpPrefix(pred.Second())
			range2.High = expression.NewRegexpStop(pred.Second())
		}
	}

	span := plan.NewSpan2(nil, plan.Ranges2{range2}, false)
	return NewTermSpans(span)
}
