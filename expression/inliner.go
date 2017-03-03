//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

/*
Inliner is a mapper that inlines bindings, e.g. of a LET clause.
*/
type Inliner struct {
	MapperBase
	mappings map[string]Expression
}

func NewInliner(mappings map[string]Expression) *Inliner {
	rv := &Inliner{
		mappings: mappings,
	}

	rv.mapper = rv
	return rv
}

func (this *Inliner) VisitIdentifier(id *Identifier) (interface{}, error) {
	repl, ok := this.mappings[id.Identifier()]
	if ok {
		return repl, nil
	} else {
		return id, nil
	}
}
