//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"fmt"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/value"
)

// Keyspace stuff

type keyspaceBase struct {
	namespace *namespace
	name      string
	fullName  string
}

func (this *keyspaceBase) Namespace() datastore.Namespace {
	return this.namespace
}

func (this *keyspaceBase) Scope() datastore.Scope {
	// System keyspaces are not part of scopes.
	return nil
}

func (this *keyspaceBase) ScopeId() string {
	// System keyspaces are not part of scopes.
	return ""
}

func (this *keyspaceBase) QualifiedName() string {
	return this.fullName
}

func (this *keyspaceBase) AuthKey() string {
	return this.name
}

func (this *keyspaceBase) Uid() string {
	return this.name
}

func (this *keyspaceBase) CreateScope(name string) errors.Error {
	return errors.NewScopesNotSupportedError(this.name)
}

func (this *keyspaceBase) DropScope(name string) errors.Error {
	return errors.NewScopesNotSupportedError(this.name)
}

var _ERRS_SYSTEM_NOT_SUPPORTED = errors.Errors{errors.NewSystemNotSupportedError(nil, "")}

func (b *keyspaceBase) Insert(inserts value.Pairs, context datastore.QueryContext) (value.Pairs, errors.Errors) {
	return nil, _ERRS_SYSTEM_NOT_SUPPORTED
}

func (b *keyspaceBase) Update(updates value.Pairs, context datastore.QueryContext) (value.Pairs, errors.Errors) {
	return nil, _ERRS_SYSTEM_NOT_SUPPORTED
}

func (b *keyspaceBase) Upsert(upserts value.Pairs, context datastore.QueryContext) (value.Pairs, errors.Errors) {
	return nil, _ERRS_SYSTEM_NOT_SUPPORTED
}

func (b *keyspaceBase) Delete(deletes value.Pairs, context datastore.QueryContext) (value.Pairs, errors.Errors) {
	return nil, _ERRS_SYSTEM_NOT_SUPPORTED
}

func (this *keyspaceBase) Flush() errors.Error {
	return errors.NewNoFlushError(this.name)
}

func (this *keyspaceBase) IsBucket() bool {
	return true
}

func (this *keyspaceBase) Stats(context datastore.QueryContext, which []datastore.KeyspaceStats) ([]int64, errors.Error) {
	var err errors.Error

	res := make([]int64, len(which))
	ks := this.namespace.keyspaces[this.name]
	for i, f := range which {
		var val int64

		switch f {
		case datastore.KEYSPACE_COUNT:
			val, err = ks.Count(context)
		case datastore.KEYSPACE_SIZE:
			val, err = ks.Size(context)
		case datastore.KEYSPACE_MEM_SIZE:
			val = -1
		}
		if err != nil {
			return nil, err
		}
		res[i] = val
	}
	return res, err
}

func setKeyspaceBase(base *keyspaceBase, namespace *namespace, name string) {
	base.namespace = namespace
	base.name = name
	base.fullName = namespace.Name() + ":" + name
}

func (this *keyspaceBase) setNamespace(namespace *namespace) {
	this.namespace = namespace
}

// Index stuff

type indexBase struct {
	indexer datastore.Indexer
}

func (this *indexBase) Indexer() datastore.Indexer {
	return this.indexer
}

func (this *indexBase) BucketId() string {
	return ""
}

func (this *indexBase) ScopeId() string {
	return ""
}

func setIndexBase(base *indexBase, indexer datastore.Indexer) {
	base.indexer = indexer
}

type compiledSpan struct {
	low      string
	high     string
	evalLow  func(val, key string) bool
	evalHigh func(val, key string) bool

	// golang does not allow to compare functional pointers to functions...
	equality bool
}

func compileSpan(span *datastore.Span) (*compiledSpan, errors.Error) {
	var err errors.Error
	var isLowValued, isHighValued bool

	// currently system indexes are either primary or on a single field
	if len(span.Seek) > 1 || len(span.Range.Low) > 1 || len(span.Range.High) > 1 {
		return nil, errors.NewSystemDatastoreError(nil, "Invalid number of fields in span")
	}

	spanEvaluator := &compiledSpan{}
	if span.Seek != nil {
		val := span.Seek[0].Actual()
		switch t := val.(type) {
		case string:
		default:
			return nil, errors.NewSystemDatastoreError(nil, fmt.Sprintf("Invalid seek value %v of type %T.", t, val))
		}
		spanEvaluator.low = val.(string)
		spanEvaluator.high = spanEvaluator.low
		spanEvaluator.evalLow = equals
		spanEvaluator.evalHigh = noop
		spanEvaluator.equality = true
	} else {
		spanEvaluator.low, spanEvaluator.evalLow, isLowValued, err = compileRange(span.Range.Low, span.Range.Inclusion, datastore.LOW)
		if err != nil {
			return nil, err
		}
		spanEvaluator.high, spanEvaluator.evalHigh, isHighValued, err = compileRange(span.Range.High, span.Range.Inclusion, datastore.HIGH)
		if err != nil {
			return nil, err
		}
		if spanEvaluator.high == spanEvaluator.low && isLowValued && isHighValued {
			spanEvaluator.evalLow = equals
			spanEvaluator.evalHigh = noop
			spanEvaluator.equality = true
		}
	}
	return spanEvaluator, nil
}

func (this *compiledSpan) evaluate(key string) bool {
	return this.evalHigh(this.high, key) && this.evalLow(this.low, key)
}

func (this *compiledSpan) isEquals() bool {
	return this.equality
}

func (this *compiledSpan) key() string {
	return this.low
}

func compileRange(in value.Values, incl, side datastore.Inclusion) (string, func(string, string) bool, bool, errors.Error) {
	if len(in) == 0 {
		return "", noop, false, nil
	}
	val := in[0].Actual()
	t := in[0].Type()
	switch t {
	case value.STRING:
	case value.NULL:

		// > null is a noop, < null should never occur and it's an error
		if side == datastore.LOW {
			return "", noop, false, nil
		}
		fallthrough
	default:
		return "", nil, false, errors.NewSystemDatastoreError(nil, fmt.Sprintf("Invalid seek value %v of type %T.", val, t.String()))
	}
	retVal := val.(string)
	op := (incl & side) > 0
	if side == datastore.HIGH {
		if op {
			return retVal, lessOrEqual, true, nil
		} else {
			return retVal, less, true, nil
		}
	} else {
		if op {
			return retVal, greaterOrEqual, true, nil
		} else {
			return retVal, greater, true, nil
		}
	}
}

func equals(val, key string) bool {
	return key == val
}

func less(top, key string) bool {
	return key < top
}

func lessOrEqual(top, key string) bool {
	return key <= top
}

func greater(bottom, key string) bool {
	return key > bottom
}

func greaterOrEqual(bottom, key string) bool {
	return key >= bottom
}

func noop(val, key string) bool {
	return true
}

// Credentials

type SystemContext interface {
	Credentials() *auth.Credentials
	SetFirstCreds(string)
	FirstCreds() (string, bool)
}

// Return the user to impersonate from the context credentials
func credsFromContext(context SystemContext) string {
	if !tenant.IsServerless() {
		return ""
	}

	// do we have a cached full user name?
	userName, isSet := context.FirstCreds()
	if isSet {
		return userName
	}
	creds := context.Credentials()
	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_SYSTEM_READ, auth.PRIV_PROPS_NONE)
	if datastore.GetDatastore().Authorize(privs, creds) == nil {

		// it's an admin, no restrictions
		userName = ""
	} else {
		userName = datastore.EncodeName(datastore.FirstCred(creds))
	}
	context.SetFirstCreds(userName)
	return userName
}
