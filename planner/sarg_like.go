//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"math"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
)

var _EMPTY_STRING = expression.Expressions{expression.EMPTY_STRING_EXPR}
var _EMPTY_ARRAY = expression.Expressions{expression.EMPTY_ARRAY_EXPR}

func (this *sarg) visitLike(pred expression.LikeFunction) (interface{}, error) {
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

	if SubsetOf(pred, this.key) {
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
		return likeSpans(pred), nil
	}

	span := &plan.Span{}
	span.Exact = false
	span.Range.Low = expression.Expressions{expression.NewConstant(prefix)}

	last := len(prefix) - 1
	if last >= 0 && prefix[last] < math.MaxUint8 {
		bytes := []byte(prefix)
		bytes[last]++
		span.Range.High = expression.Expressions{expression.NewConstant(string(bytes))}
		if re.NumSubexp() == 1 && re.String()[len(prefix):] == "(.*)" {
			span.Exact = true
		}
	} else {
		span.Range.High = _EMPTY_ARRAY
	}

	span.Range.Inclusion = datastore.LOW
	return NewTermSpans(span), nil
}

func likeSpans(pred expression.LikeFunction) SargSpans {
	span := &plan.Span{}
	span.Exact = false

	switch pred := pred.(type) {
	case *expression.Like:
		span.Range.Low = expression.Expressions{expression.NewLikePrefix(pred.Second())}
		span.Range.High = expression.Expressions{expression.NewLikeStop(pred.Second())}
	case *expression.RegexpLike:
		span.Range.Low = expression.Expressions{expression.NewRegexpPrefix(pred.Second())}
		span.Range.High = expression.Expressions{expression.NewRegexpStop(pred.Second())}
	default:
		span.Range.Low = _EMPTY_STRING
		span.Range.High = _EMPTY_ARRAY
	}

	span.Range.Inclusion = datastore.LOW
	return NewTermSpans(span)
}
