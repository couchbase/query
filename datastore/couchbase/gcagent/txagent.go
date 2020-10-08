//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

// +build enterprise

package gcagent

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/couchbase/gocbcore/v9"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
	gctx "github.com/couchbaselabs/gocbcore-transactions"
)

const (
	MOP_NONE int = iota
	MOP_INSERT
	MOP_UPSERT
	MOP_UPDATE
	MOP_DELETE
)

type GetOp struct {
	Key    string
	Val    value.AnnotatedValue
	Err    error
	Pendop gocbcore.PendingOp
}

type AgentProvider struct {
	provider *gocbcore.Agent
}

func (ap *AgentProvider) Close() error {
	return ap.provider.Close()
}

func (ap *AgentProvider) Deadline(d time.Time, n int) time.Time {
	if d.IsZero() {
		return time.Now().Add(time.Duration(n) * _KVTIMEOUT)
	}
	return d
}

// Create annotated value

func (ap *AgentProvider) getTxAnnotatedValue(res *gctx.GetResult, key, fullName string) (value.AnnotatedValue, error) {
	txnMetaBytes, err := json.Marshal(res.Meta)
	if err != nil {
		return nil, err
	}

	av := value.NewAnnotatedValue(value.NewParsedValue(res.Value, false))
	meta_type := "json"
	if av.Type() == value.BINARY {
		meta_type = "base64"
	}

	meta := av.NewMeta()
	meta["keyspace"] = fullName
	meta["cas"] = uint64(res.Cas)
	meta["type"] = meta_type
	meta["flags"] = uint32(0)
	meta["expiration"] = uint32(0)
	meta["txnMeta"] = txnMetaBytes
	av.SetId(key)

	return av, nil
}

// bulk transactional get

func (ap *AgentProvider) TxGet(transaction *gctx.Transaction, fullName, bucketName, scopeName, collectionName string,
	collectionID uint32, keys, paths []string, reqDeadline time.Time, replica, notFoundErr bool,
	fetchMap map[string]value.AnnotatedValue) (errs []error) {

	if len(paths) > 0 && paths[0] != "$document.exptime" {
		return append(errs, ErrNoSubDocInTransaction)
	}

	defer func() {
		// protect from panics
		if r := recover(); r != nil {
			errs = append(errs, fmt.Errorf("TxGet() Panic: %v", r))
		}
	}()

	// send the request and get results in call back
	wg := &sync.WaitGroup{}
	sendOneGet := func(item *GetOp) error {
		wg.Add(1)
		cerr := transaction.Get(gctx.GetOptions{
			Agent:          ap.provider,
			ScopeName:      scopeName,
			CollectionName: collectionName,
			Key:            []byte(item.Key),
		}, func(res *gctx.GetResult, resErr error) {
			defer wg.Done()
			item.Err = resErr
			if item.Err == nil && res != nil {
				item.Val, item.Err = ap.getTxAnnotatedValue(res, item.Key, fullName)
			}
		})

		if cerr != nil {
			wg.Add(-1)
		}
		return cerr
	}

	var prevErr error
	items := make([]*GetOp, 0, len(keys))
	for _, k := range keys {
		gop := &GetOp{Key: k}
		if err := sendOneGet(gop); err != nil {
			// process other errors before processing PreviousOperationFailed
			if err1, ok1 := err.(*gctx.TransactionOperationFailedError); ok1 &&
				errors.Is(err1.Unwrap(), gctx.ErrPreviousOperationFailed) {
				prevErr = err
				break
			} else {
				// request send failed. no need to wait to complete.
				return append(errs, err)
			}
		}
		items = append(items, gop)
	}

	// wait all requests are completed
	wg.Wait()

	for _, item := range items {
		if item.Err == nil && item.Val != nil {
			fetchMap[item.Key] = item.Val
		} else if notFoundErr || !errors.Is(item.Err, gocbcore.ErrDocumentNotFound) {
			// handle key not found error
			errs = append(errs, item.Err)
		}
	}

	if len(errs) == 0 && prevErr != nil {
		errs = append(errs, prevErr)
	}

	return errs
}

type WriteOps []*WriteOp

type WriteOp struct {
	Op      int
	Key     string
	Data    []byte
	TxnMeta []byte
	Cas     uint64
	Expiry  uint32
	Pendop  gocbcore.PendingOp
	Err     error
}

// bulk transactional write

func (ap *AgentProvider) TxWrite(transaction *gctx.Transaction, txnInternal *gctx.TransactionsInternal,
	bucketName, scopeName, collectionName string,
	collectionID uint32, reqDeadline time.Time, wops WriteOps) (errOut error) {

	defer func() {
		// protect from panics
		if r := recover(); r != nil {
			errOut = fmt.Errorf("TxWrite() Panic: %v", r)
		}
	}()

	wg := &sync.WaitGroup{}
	txId := transaction.Attempt().ID
	defer logging.Tracef("=====%v=====end   TxWrite(%v)========", txId, len(wops))
	logging.Tracef("=====%v=====begin TxWrite(%v)========", txId, len(wops))

	// insert request and get results in call back
	sendInsertOne := func(wop *WriteOp) error {
		wg.Add(1)
		cerr := transaction.Insert(gctx.InsertOptions{
			Agent:          ap.provider,
			ScopeName:      scopeName,
			CollectionName: collectionName,
			Key:            []byte(wop.Key),
			Value:          wop.Data,
		}, func(res *gctx.GetResult, resErr error) {
			defer wg.Done()
			wop.Err = resErr
		})

		if cerr != nil {
			wg.Add(-1)
		}
		return cerr
	}

	// update request and get results in call back
	sendUpdateOne := func(wop *WriteOp, reqRes *gctx.GetResult) error {
		wg.Add(1)
		cerr := transaction.Replace(gctx.ReplaceOptions{
			Document: reqRes,
			Value:    wop.Data,
		}, func(res *gctx.GetResult, resErr error) {
			defer wg.Done()
			wop.Err = resErr
		})

		if cerr != nil {
			wg.Add(-1)
		}
		return cerr
	}

	// delete request and get results in call back
	sendDeleteOne := func(wop *WriteOp, reqRes *gctx.GetResult) error {
		wg.Add(1)
		cerr := transaction.Remove(gctx.RemoveOptions{
			Document: reqRes,
		}, func(res *gctx.GetResult, resErr error) {
			defer wg.Done()
			wop.Err = resErr
		})

		if cerr != nil {
			wg.Add(-1)
		}
		return cerr
	}

	var prevErr error
	for _, op := range wops {
		switch op.Op {
		case MOP_INSERT:
			errOut = sendInsertOne(op)
		case MOP_UPDATE:
			var txnMeta gctx.MutableItemMeta
			errOut = json.Unmarshal(op.TxnMeta, &txnMeta)
			if errOut == nil {
				tmpRes := txnInternal.CreateGetResult(gctx.CreateGetResultOptions{
					Agent:          ap.provider,
					ScopeName:      scopeName,
					CollectionName: collectionName,
					Key:            []byte(op.Key),
					Cas:            gocbcore.Cas(op.Cas),
					Meta:           txnMeta,
				})
				errOut = sendUpdateOne(op, tmpRes)
			}
		case MOP_DELETE:
			var txnMeta gctx.MutableItemMeta
			errOut = json.Unmarshal(op.TxnMeta, &txnMeta)
			if errOut == nil {
				tmpRes := txnInternal.CreateGetResult(gctx.CreateGetResultOptions{
					Agent:          ap.provider,
					ScopeName:      scopeName,
					CollectionName: collectionName,
					Key:            []byte(op.Key),
					Cas:            gocbcore.Cas(op.Cas),
					Meta:           txnMeta,
				})
				errOut = sendDeleteOne(op, tmpRes)
			}
		default:
			errOut = ErrUnknownOperation
		}
		if errOut != nil {
			// process other errors before processing PreviousOperationFailed
			if err1, ok1 := errOut.(*gctx.TransactionOperationFailedError); ok1 &&
				errors.Is(err1.Unwrap(), gctx.ErrPreviousOperationFailed) {
				prevErr = errOut
				break
			} else {
				return errOut
			}

		}
	}

	wg.Wait()
	for _, op := range wops {
		if op.Err != nil {
			return op.Err
		}
	}

	return prevErr
}
