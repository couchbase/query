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
	"github.com/couchbaselabs/query/value"
)

type Visitor interface {
	Visit(expr Expression) (Expression, error)
}

type Folder struct {
}

func (this *Folder) Visit(expr Expression) (Expression, error) {
	return expr.Fold()
}

type Formalizer struct {
	Forbidden value.Value
	Allowed   value.Value
	Bucket    string
}

func (this *Formalizer) Visit(expr Expression) (Expression, error) {
	return expr.Formalize(this.Forbidden, this.Allowed, this.Bucket)
}
