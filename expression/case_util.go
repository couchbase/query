//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

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
