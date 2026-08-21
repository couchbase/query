//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"encoding/json"
	"strings"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
Represents all three forms of the DROP KNOWLEDGE ddl statement:

	DROP KNOWLEDGE [IF EXISTS] <name>[, <name>, ...] FOR <keyspace>
	DROP KNOWLEDGE [IF EXISTS] FOR <keyspace>

The first two remove one or more specific named entries; the third (Names() == nil) removes every
entry stored for the keyspace in one write. IF EXISTS (FailIfNotExists() == false) suppresses the
not-found error when a named entry - or, for the bare form, the keyspace's knowledge document
itself - doesn't exist; without it, that condition is an error.
*/
type DropKnowledge struct {
	statementBase

	keyspace        *Path    `json:"keyspace"`
	names           []string `json:"names"`
	failIfNotExists bool     `json:"failIfNotExists"`
}

func NewDropKnowledge(keyspace *Path, names []string, failIfNotExists bool) *DropKnowledge {
	// a bare bucket name (no scope/collection given) implicitly means bucket._default._default,
	// mirroring newCreateKnowledge's normalization - without this, DROP KNOWLEDGE ... FOR <bucket>
	// would never match what CREATE KNOWLEDGE ... FOR <bucket> actually stored.
	if keyspace != nil && keyspace.Scope() == "" {
		keyspace = NewPathLong(keyspace.Namespace(), keyspace.Bucket(), "_default", "_default")
	}

	rv := &DropKnowledge{
		keyspace:        keyspace,
		names:           names,
		failIfNotExists: failIfNotExists,
	}

	rv.stmt = rv
	return rv
}

func (this *DropKnowledge) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropKnowledge(this)
}

func (this *DropKnowledge) Signature() value.Value {
	return nil
}

func (this *DropKnowledge) Formalize() error {
	return nil
}

func (this *DropKnowledge) MapExpressions(mapper expression.Mapper) error {
	return nil
}

func (this *DropKnowledge) Expressions() expression.Expressions {
	return nil
}

func (this *DropKnowledge) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_ADMIN, auth.PRIV_PROPS_NONE)
	return privs, nil
}

func (this *DropKnowledge) Keyspace() *Path {
	return this.keyspace
}

// Names returns the entry names to drop, or nil for the bare form that drops every entry stored
// for the keyspace.
func (this *DropKnowledge) Names() []string {
	return this.names
}

func (this *DropKnowledge) FailIfNotExists() bool {
	return this.failIfNotExists
}

func (this *DropKnowledge) MarshalName(m map[string]interface{}) {
	m["namespace"] = this.keyspace.Namespace()
	m["bucket"] = this.keyspace.Bucket()
	m["scope"] = this.keyspace.Scope()
	m["keyspace"] = this.keyspace.Keyspace()
	m["names"] = this.names
}

func (this *DropKnowledge) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "dropKnowledge"}
	this.MarshalName(r)
	r["failIfNotExists"] = this.failIfNotExists
	return json.Marshal(r)
}

func (this *DropKnowledge) Type() string {
	return "DROP_KNOWLEDGE"
}

func (this *DropKnowledge) String() string {
	var s strings.Builder
	s.WriteString("DROP KNOWLEDGE ")
	if !this.failIfNotExists {
		s.WriteString("IF EXISTS ")
	}
	for i, n := range this.names {
		if i > 0 {
			s.WriteString(", ")
		}
		s.WriteString("`")
		s.WriteString(n)
		s.WriteString("`")
	}
	if len(this.names) > 0 {
		s.WriteString(" ")
	}
	s.WriteString("FOR ")
	s.WriteString(this.keyspace.ProtectedString())
	return s.String()
}
