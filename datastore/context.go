//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package datastore

import (
	"sync"
	"time"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

var NULL_QUERY_CONTEXT QueryContext = &queryContextImpl{}
var MAJORITY_QUERY_CONTEXT QueryContext = &majorityQueryContextImpl{}

type Context interface {
	GetScanCap() int64
	MaxParallelism() int
	Fatal(errors.Error)
	Error(errors.Error)
	Warning(errors.Error)
	GetErrors() []errors.Error
	GetReqDeadline() time.Time
	TenantCtx() tenant.Context
	SetFirstCreds(string)
	FirstCreds() (string, bool)
	RecordFtsRU(ru tenant.Unit)
	RecordGsiRU(ru tenant.Unit)
	RecordKvRU(ru tenant.Unit)
	RecordKvWU(wu tenant.Unit)
	Credentials() *auth.Credentials
	ScanReportWait() time.Duration
	SkipKey(key string) bool
}

type contextImpl struct {
}

func (ci *contextImpl) GetScanCap() int64 {
	return GetScanCap()
}

func (ci *contextImpl) MaxParallelism() int {
	return 1
}

func (ci *contextImpl) Fatal(err errors.Error) {
}

func (ci *contextImpl) Error(err errors.Error) {
}

func (ci *contextImpl) Warning(err errors.Error) {
}

func (ci *contextImpl) GetErrors() []errors.Error {
	return nil
}

func (ci *contextImpl) GetReqDeadline() time.Time {
	return time.Time{}
}

func (ci *contextImpl) FeatureControl() uint64 {
	return 0
}

func (ci *contextImpl) TenantCtx() tenant.Context {
	return nil
}

func (ci *contextImpl) SetFirstCreds(string) {
}

func (ci *contextImpl) FirstCreds() (string, bool) {
	return "", true
}

func (ci *contextImpl) RecordFtsRU(ru tenant.Unit) {
}

func (ci *contextImpl) RecordGsiRU(ru tenant.Unit) {
}

func (ci *contextImpl) ScanReportWait() time.Duration {
	return time.Duration(0)
}

func (ci *contextImpl) RecordKvRU(ru tenant.Unit) {
}

func (ci *contextImpl) RecordKvWU(wu tenant.Unit) {
}

func (ci *contextImpl) Credentials() *auth.Credentials {
	return auth.NewCredentials()
}

func (ci *contextImpl) SkipKey(key string) bool {
	return false
}

// used for situations where errors need to be tracked
type systemContextImpl struct {
	contextImpl
	sync.RWMutex

	errors []errors.Error
}

func NewSystemContext() Context {
	return &systemContextImpl{}
}

func (sci *systemContextImpl) Fatal(err errors.Error) {
	sci.Error(err)
}

func (sci *systemContextImpl) Error(err errors.Error) {
	sci.Lock()
	switch err.Level() {
	case errors.EXCEPTION, errors.ERROR:
		sci.errors = append(sci.errors, err)
	}
	sci.Unlock()
}

func (sci *systemContextImpl) GetErrors() []errors.Error {
	sci.RLock()
	errs := sci.errors
	sci.RUnlock()
	return errs
}

// A subset of execution.Context that is useful at the datastore level.
type QueryContext interface {
	logging.Log
	GetReqDeadline() time.Time
	UseReplica() bool
	Credentials() *auth.Credentials
	Warning(errors.Error)
	Error(errors.Error)
	GetTxContext() interface{}
	SetTxContext(tc interface{})
	Datastore() Datastore
	TxDataVal() value.Value
	DurabilityLevel() DurabilityLevel
	KvTimeout() time.Duration
	PreserveExpiry() bool
	TenantCtx() tenant.Context
	SetFirstCreds(string)
	FirstCreds() (string, bool)
	RecordFtsRU(ru tenant.Unit)
	RecordGsiRU(ru tenant.Unit)
	RecordKvRU(ru tenant.Unit)
	RecordKvWU(wu tenant.Unit)
	IsActive() bool
	RequestId() string
	ErrorLimit() int
	ErrorCount() int
	DurationStyle() util.DurationStyle
	FormatDuration(time.Duration) string
	UserAgent() string
	Users() string
	RemoteAddr() string
}

type queryContextImpl struct {
}

func (ci *queryContextImpl) Credentials() *auth.Credentials {
	return auth.NewCredentials()
}

func (ci *queryContextImpl) Warning(err errors.Error) {
}

func (ci *queryContextImpl) Error(err errors.Error) {
}

func (ci *queryContextImpl) GetReqDeadline() time.Time {
	return time.Time{}
}

func (ci *queryContextImpl) UseReplica() bool {
	return false
}

func (ci *queryContextImpl) GetTxContext() interface{} {
	return nil
}

func (ci *queryContextImpl) Datastore() Datastore {
	return GetDatastore()
}

func (ci *queryContextImpl) SetTxContext(tc interface{}) {
}

func (ci *queryContextImpl) TxDataVal() value.Value {
	return nil
}

func (ci *queryContextImpl) DurabilityLevel() DurabilityLevel {
	return DL_NONE
}

func (ci *queryContextImpl) KvTimeout() time.Duration {
	return DEF_KVTIMEOUT
}

func (ci *queryContextImpl) PreserveExpiry() bool {
	return false
}

func (ci *queryContextImpl) TenantCtx() tenant.Context {
	return nil
}

func (ci *queryContextImpl) SetFirstCreds(string) {
}

func (ci *queryContextImpl) FirstCreds() (string, bool) {
	return "", true
}

func (ci *queryContextImpl) RecordFtsRU(ru tenant.Unit) {
}

func (ci *queryContextImpl) RecordGsiRU(ru tenant.Unit) {
}

func (ci *queryContextImpl) ScanReportWait() time.Duration {
	return time.Duration(0)
}

func (ci *queryContextImpl) RecordKvRU(ru tenant.Unit) {
}

func (ci *queryContextImpl) RecordKvWU(wu tenant.Unit) {
}

func (ci *queryContextImpl) IsActive() bool {
	return true
}

func (ci *queryContextImpl) RequestId() string {
	return ""
}

func (ci *queryContextImpl) UserAgent() string {
	return ""
}

func (ci *queryContextImpl) Users() string {
	return ""
}

func (ci *queryContextImpl) RemoteAddr() string {
	return ""
}

func (ci *queryContextImpl) Loga(l logging.Level, f func() string)                   {}
func (ci *queryContextImpl) Debuga(f func() string)                                  {}
func (ci *queryContextImpl) Tracea(f func() string)                                  {}
func (ci *queryContextImpl) Infoa(f func() string)                                   {}
func (ci *queryContextImpl) Warna(f func() string)                                   {}
func (ci *queryContextImpl) Errora(f func() string)                                  {}
func (ci *queryContextImpl) Severea(f func() string)                                 {}
func (ci *queryContextImpl) Fatala(f func() string)                                  {}
func (ci *queryContextImpl) Logf(level logging.Level, f string, args ...interface{}) {}
func (ci *queryContextImpl) Debugf(f string, args ...interface{})                    {}
func (ci *queryContextImpl) Tracef(f string, args ...interface{})                    {}
func (ci *queryContextImpl) Infof(f string, args ...interface{})                     {}
func (ci *queryContextImpl) Warnf(f string, args ...interface{})                     {}
func (ci *queryContextImpl) Errorf(f string, args ...interface{})                    {}
func (ci *queryContextImpl) Severef(f string, args ...interface{})                   {}
func (ci *queryContextImpl) Fatalf(f string, args ...interface{})                    {}

func (ci *queryContextImpl) ErrorLimit() int {
	return errors.DEFAULT_REQUEST_ERROR_LIMIT
}

func (ci *queryContextImpl) ErrorCount() int {
	return 0
}

func (ci *queryContextImpl) DurationStyle() util.DurationStyle {
	return util.DEFAULT
}

func (ci *queryContextImpl) FormatDuration(time.Duration) string {
	return ""
}

type majorityQueryContextImpl struct {
	queryContextImpl
}

func (ci *majorityQueryContextImpl) DurabilityLevel() DurabilityLevel {
	return DL_MAJORITY
}

func GetDurableQueryContextFor(b Keyspace) QueryContext {
	if eb, ok := b.Scope().Bucket().(ExtendedBucket); ok {
		if !eb.DurabilityPossible() {
			return NULL_QUERY_CONTEXT
		}
	}
	return MAJORITY_QUERY_CONTEXT
}
