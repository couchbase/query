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

func (b *keyspaceBase) Insert(inserts value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {

	return 0, nil, _ERRS_SYSTEM_NOT_SUPPORTED
}

func (b *keyspaceBase) Update(updates value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {

	return 0, nil, _ERRS_SYSTEM_NOT_SUPPORTED
}

func (b *keyspaceBase) Upsert(upserts value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {

	return 0, nil, _ERRS_SYSTEM_NOT_SUPPORTED
}

func (b *keyspaceBase) Delete(deletes value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {

	return 0, nil, _ERRS_SYSTEM_NOT_SUPPORTED
}

func (this *keyspaceBase) Flush() errors.Error {
	return errors.NewNoFlushError(this.name)
}

func (this *keyspaceBase) IsBucket() bool {
	return true
}

func (this *keyspaceBase) IsSystemCollection() bool {
	return false
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

func (this *keyspaceBase) SetSubDoc(string, value.Pairs, datastore.QueryContext) (value.Pairs, errors.Error) {
	return nil, _ERRS_SYSTEM_NOT_SUPPORTED[0]
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

	// check for missing
	isMissingTest bool
}

type compiledSpans []compiledSpan

func (this compiledSpans) isEquals() int {
	for idx, s := range this {
		if s.isEquals() {
			return idx
		}
	}
	return -1
}

func (this compiledSpans) evaluate(key string) bool {
	for _, s := range this {
		if s.evaluate(key) {
			return true
		}
	}
	return false
}

func (this compiledSpans) acceptMissing() bool {
	for _, s := range this {
		if s.acceptMissing() {
			return true
		}
	}
	return false
}

func (this compiledSpans) key(idx int) string {
	if idx < 0 || idx >= len(this) {
		return ""
	}
	return this[idx].key()
}

func compileSpan(span *datastore.Span) (compiledSpans, errors.Error) {
	var err errors.Error
	var isLowValued, isHighValued bool

	cSpans := make(compiledSpans, 0, 2)
	for _, seek := range span.Seek {
		spanEvaluator := compiledSpan{}
		val := seek.Actual()
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
		cSpans = append(cSpans, spanEvaluator)
	}
	spanEvaluator := compiledSpan{}
	spanEvaluator.low, spanEvaluator.evalLow, isLowValued, err = compileRange(span.Range.Low, span.Range.Inclusion, datastore.LOW)
	if err != nil {
		return nil, err
	}
	spanEvaluator.high, spanEvaluator.evalHigh, isHighValued, err = compileRange(span.Range.High, span.Range.Inclusion,
		datastore.HIGH)
	if err != nil {
		return nil, err
	}

	if spanEvaluator.high == spanEvaluator.low && isHighValued {
		spanEvaluator.evalHigh = noop

		if isLowValued {
			spanEvaluator.evalLow = equals
			spanEvaluator.equality = true
		}
	}
	cSpans = append(cSpans, spanEvaluator)
	return cSpans, nil
}

func compileSpan2(spans datastore.Spans2) (compiledSpans, errors.Error) {
	var err errors.Error
	var isLowValued, isHighValued bool

	cSpans := make(compiledSpans, 0, len(spans))
	for _, span := range spans {
		for _, seek := range span.Seek {
			spanEvaluator := compiledSpan{}
			val := seek.Actual()
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
			cSpans = append(cSpans, spanEvaluator)
		}
		for _, rng := range span.Ranges {
			spanEvaluator := compiledSpan{}
			spanEvaluator.low, spanEvaluator.evalLow, isLowValued, err = compileRange2(rng.Low, rng.Inclusion, datastore.LOW)
			if err != nil {
				return nil, err
			}
			spanEvaluator.high, spanEvaluator.evalHigh, isHighValued, err = compileRange2(rng.High, rng.Inclusion, datastore.HIGH)
			if err != nil {
				return nil, err
			}

			if !isLowValued {
				spanEvaluator.isMissingTest = true
			}

			if spanEvaluator.high == spanEvaluator.low && isLowValued && isHighValued {
				spanEvaluator.evalLow = equals
				spanEvaluator.evalHigh = noop
				spanEvaluator.equality = true
			} else if isHighValued && spanEvaluator.evalHigh == nil {
				if !isLowValued {
					spanEvaluator.evalHigh = fail
				} else {
					spanEvaluator.evalHigh = noop
				}
			}
			cSpans = append(cSpans, spanEvaluator)
		}
	}
	return cSpans, nil
}

func (this *compiledSpan) evaluate(key string) bool {
	return this.evalHigh(this.high, key) && this.evalLow(this.low, key)
}

func (this *compiledSpan) acceptMissing() bool {
	return this.isMissingTest
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
	return compileRange2(in[0], incl, side)
}

func compileRange2(in value.Value, incl, side datastore.Inclusion) (string, func(string, string) bool, bool, errors.Error) {
	if in == nil {
		return "", noop, false, nil
	}
	val := in.Actual()
	t := in.Type()
	switch t {
	case value.STRING:
	case value.NULL:

		// > null is a noop, < null should never occur and it's an error
		if side == datastore.LOW {
			return "", noop, false, nil
		} else if side == datastore.HIGH {

			if incl == 0 {
				return "", nil, true, nil
			} else if incl == 2 || incl == 3 {
				return "", fail, false, nil
			}
		}
		fallthrough
	default:
		return "", nil, false, errors.NewSystemDatastoreError(nil, fmt.Sprintf("Invalid seek value %v of type %v.", val,
			t.String()))
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

func fail(val, key string) bool {
	return false
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

	// Since this is an internal action to check if the user is an admin
	if datastore.GetDatastore().AuthorizeInternal(privs, creds) == nil {

		// it's an admin, no restrictions
		userName = ""
	} else {
		userName = datastore.EncodeName(datastore.FirstCred(creds))
	}
	context.SetFirstCreds(userName)
	return userName
}

// In serverless encodes Node Name to NodeUUID
func encodeNodeName(node string) string {
	if node != "" {
		return tenant.EncodeNodeName(node)
	}
	return ""
}

// In serverless decodes NodeUUID to Node Name
func decodeNodeName(node string) string {
	if node != "" {
		return tenant.DecodeNodeName(node)
	}
	return ""
}
