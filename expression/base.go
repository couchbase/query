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
	"reflect"

	"github.com/couchbaselabs/query/value"
)

type ExpressionBase struct {
}

func (this *ExpressionBase) Evaluate(item value.Value, context Context) (value.Value, error) {
	panic("Must override.")
}

func (this *ExpressionBase) EquivalentTo(other Expression) bool {
	if reflect.TypeOf(this) != reflect.TypeOf(other) {
		return false
	}

	ours := Expression(this).Children()
	theirs := other.Children()
	if len(ours) != len(theirs) {
		return false
	}

	for i, o := range ours {
		if !o.EquivalentTo(theirs[i]) {
			return false
		}
	}

	return true
}

func (this *ExpressionBase) Alias() string {
	return ""
}

func (this *ExpressionBase) Fold() (Expression, error) {
	return Expression(this).VisitChildren(&Folder{})
}

func (this *ExpressionBase) Formalize(forbidden, allowed value.Value,
	bucket string) (Expression, error) {
	f := &Formalizer{
		Forbidden: forbidden,
		Allowed:   allowed,
		Bucket:    bucket,
	}

	return Expression(this).VisitChildren(f)
}

func (this *ExpressionBase) SubsetOf(other Expression) bool {
	return this.EquivalentTo(other)
}

func (this *ExpressionBase) Children() Expressions {
	return nil
}

func (this *ExpressionBase) VisitChildren(visitor Visitor) (Expression, error) {
	return this, nil
}

func (this *ExpressionBase) MinArgs() int { return 0 }

func (this *ExpressionBase) MaxArgs() int { return 0 }
