//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"math"

	"github.com/couchbaselabs/query/value"
)

type Slice struct {
	ExpressionBase
	source Expression
	start  Expression
	end    Expression
}

func NewSlice(source, start, end Expression) Expression {
	return &Slice{
		source: source,
		start:  start,
		end:    end,
	}
}

func (this *Slice) Evaluate(item value.Value, context Context) (rv value.Value, re error) {
	source, e := this.source.Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if source.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	start, e := this.start.Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if start.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	ev := -1
	if this.end != nil {
		end, e := this.end.Evaluate(item, context)
		if e != nil {
			return nil, e
		}

		if end.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		}

		ea, ok := end.Actual().(float64)
		if !ok || ea != math.Trunc(ea) {
			return value.NULL_VALUE, nil
		}

		ev = int(ea)
	}

	sa, ok := start.Actual().(float64)
	if !ok || sa != math.Trunc(sa) {
		return value.NULL_VALUE, nil
	}

	if source.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	if this.end != nil {
		rv, _ = source.Slice(int(sa), ev)
	} else {
		rv, _ = source.SliceTail(int(sa))
	}

	return
}

func (this *Slice) Children() Expressions {
	rv := make(Expressions, 0, 3)
	rv = append(rv, this.source)
	rv = append(rv, this.start)

	if this.end != nil {
		rv = append(rv, this.end)
	}

	return rv
}

func (this *Slice) VisitChildren(visitor Visitor) (Expression, error) {
	var e error
	this.source, e = visitor.Visit(this.source)
	if e != nil {
		return nil, e
	}

	this.start, e = visitor.Visit(this.start)
	if e != nil {
		return nil, e
	}

	if this.end != nil {
		this.end, e = visitor.Visit(this.end)
		if e != nil {
			return nil, e
		}
	}

	return this, nil
}
