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
Renamer is used to rename binding variables, but is a generic
expression renamer.
*/
type Renamer struct {
	MapperBase

	names map[string]*Identifier
}

func NewRenamer(from, to Bindings) *Renamer {
	var names map[string]*Identifier

	if from.SubsetOf(to) {
		for i, f := range from {
			t := to[i]

			if f.variable != t.variable {
				if names == nil {
					names = make(map[string]*Identifier, len(from))
				}

				names[f.variable] = NewIdentifier(t.variable)
			}

			if f.nameVariable != t.nameVariable {
				if names == nil {
					names = make(map[string]*Identifier, len(from))
				}

				names[f.nameVariable] = NewIdentifier(t.nameVariable)
			}
		}
	}

	rv := &Renamer{
		names: names,
	}

	rv.mapFunc = func(expr Expression) (Expression, error) {
		if len(names) == 0 {
			return expr, nil
		} else {
			return expr, expr.MapChildren(rv)
		}
	}

	rv.mapper = rv
	return rv
}

func (this *Renamer) VisitIdentifier(expr *Identifier) (interface{}, error) {
	name, ok := this.names[expr.identifier]
	if ok {
		return name, nil
	} else {
		return expr, nil
	}
}
