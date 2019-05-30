//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

// retrieve identifier expression for a keyspace
func GetKeyspaceIdentifier(alias string, expr Expression) *Identifier {
	retriever := NewKSIdentRetriever(alias)
	_, err := expr.Accept(retriever)
	if err != nil {
		return nil
	}
	return retriever.ident
}

type ksIdentRetriever struct {
	TraverserBase

	keyspace string
	ident    *Identifier
}

func NewKSIdentRetriever(keyspace string) *ksIdentRetriever {
	rv := &ksIdentRetriever{
		keyspace: keyspace,
	}

	rv.traverser = rv
	return rv
}

func (this *ksIdentRetriever) VisitIdentifier(ident *Identifier) (interface{}, error) {
	if this.ident == nil && this.keyspace == ident.Identifier() {
		this.ident = ident
	}
	return nil, nil
}
