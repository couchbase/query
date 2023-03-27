//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"time"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

/*
Type IndexContext is a structure containing a variable
now that is of type Time which represents an instant in
time.
*/
type IndexContext struct {
	now time.Time
}

/*
This method returns a pointer to the IndecContext
structure, after assigning its value now with the
current local time using the time package's Now
function.
*/
func NewIndexContext() Context {
	return &IndexContext{
		now: time.Now(),
	}
}

/*
This method allows us to access the value now in the
receiver of type IndexContext. It returns the now
value from the receiver.
*/
func (this *IndexContext) Now() time.Time {
	return this.now
}

// next methods are unused and only for expression Context compatibility
func (this *IndexContext) GetTimeout() time.Duration {
	return time.Duration(0)
}

func (this *IndexContext) Credentials() *auth.Credentials {
	return nil
}

func (this *IndexContext) DatastoreVersion() string {
	return "BOGUS_VERSION"
}

func (this *IndexContext) EvaluateStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values, subquery, readonly bool, doCaching bool) (value.Value, uint64, error) {
	return nil, 0, nil
}

func (this *IndexContext) OpenStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values, subquery, readonly bool, doCaching bool) (
	interface {
		Type() string
		Mutations() uint64
		Results() (interface{}, uint64, error)
		Complete() (uint64, error)
		NextDocument() (value.Value, error)
		Cancel()
	}, error) {
	return nil, nil
}

func (this *IndexContext) Parse(s string) (interface{}, error) {
	return nil, nil
}

func (this *IndexContext) Infer(value.Value, value.Value) (value.Value, error) {
	return nil, nil
}

func (this *IndexContext) Readonly() bool {
	return true
}

func (this *IndexContext) NewQueryContext(queryContext string, readonly bool) interface{} {
	return nil
}

func (this *IndexContext) AdminContext() (interface{}, error) {
	return nil, nil
}

func (this *IndexContext) QueryContext() string {
	return ""
}

func (this *IndexContext) QueryContextParts() []string {
	return nil
}

func (this *IndexContext) GetTxContext() interface{} {
	return nil
}

func (this *IndexContext) SetTxContext(c interface{}) {
	// no-op
}

func (this *IndexContext) IsAdmin() bool {
	return false
}

func (this *IndexContext) SetAdvisor() {
	// no-op
}

func (this *IndexContext) IncRecursionCount(inc int) int {
	return 0
}

func (this *IndexContext) RecursionCount() int {
	return 0
}

func (this *IndexContext) StoreValue(key string, val interface{}) {
	// no-op
}

func (this *IndexContext) RetrieveValue(key string) interface{} {
	return nil
}

func (this *IndexContext) ReleaseValue(key string) {
	// no-op
}

func (this *IndexContext) SetTracked(t bool) {
	// no-op
}

func (this *IndexContext) IsTracked() bool {
	return false
}

func (this *IndexContext) RecordJsCU(d time.Duration, m uint64) {
}

func (this *IndexContext) PreserveProjectionOrder() bool {
	return false
}

func (this *IndexContext) Park(stop func(stop bool)) {
	// no-op
}

func (this *IndexContext) Resume() {
	// no-op
}

func (this *IndexContext) Loga(level logging.Level, f func() string)               {}
func (this *IndexContext) Debuga(f func() string)                                  {}
func (this *IndexContext) Tracea(f func() string)                                  {}
func (this *IndexContext) Infoa(f func() string)                                   {}
func (this *IndexContext) Warna(f func() string)                                   {}
func (this *IndexContext) Errora(f func() string)                                  {}
func (this *IndexContext) Severea(f func() string)                                 {}
func (this *IndexContext) Fatala(f func() string)                                  {}
func (this *IndexContext) Logf(level logging.Level, f string, args ...interface{}) {}
func (this *IndexContext) Debugf(f string, args ...interface{})                    {}
func (this *IndexContext) Tracef(f string, args ...interface{})                    {}
func (this *IndexContext) Infof(f string, args ...interface{})                     {}
func (this *IndexContext) Warnf(f string, args ...interface{})                     {}
func (this *IndexContext) Errorf(f string, args ...interface{})                    {}
func (this *IndexContext) Severef(f string, args ...interface{})                   {}
func (this *IndexContext) Fatalf(f string, args ...interface{})                    {}

func (this *IndexContext) InitUdfPlans() {
	// no op
}

func (this *IndexContext) InitUdfStmtExecTrees() {
	// no op
}
