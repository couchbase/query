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
Represents a slice of when terms
*/
type WhenTerms []*WhenTerm

/*
Type WhenTerm is a struct that has two fields representing the When and
then expressions for a case statement.
*/
type WhenTerm struct {
	When Expression
	Then Expression
}

func (this *WhenTerm) Copy() *WhenTerm {
	return &WhenTerm{
		When: this.When.Copy(),
		Then: this.Then.Copy(),
	}
}

func (this WhenTerms) Copy() WhenTerms {
	copies := make(WhenTerms, len(this))
	for i, term := range this {
		copies[i] = term.Copy()
	}

	return copies
}
