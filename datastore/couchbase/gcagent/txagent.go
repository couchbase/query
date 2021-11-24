//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package gcagent

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	gctx "github.com/couchbase/gocbcore-transactions"
	"github.com/couchbase/gocbcore/v10"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

const (
	MOP_NONE int = iota
	MOP_INSERT
	MOP_UPSERT
	MOP_UPDATE
	MOP_DELETE
)

var _MutateOpNames = [...]string{"UNKNOWN", "INSERT", "UPSERT", "UPDATE", "DELETE"}

type GetOp struct {
	Key    string
	Val    value.AnnotatedValue
	Err    error
	Pendop gocbcore.PendingOp
}

type AgentProvider struct {
	mutex      sync.RWMutex
	client     *Client
	bucketName string
	provider   *gocbcore.Agent
}

/* gocbcore will not allow Refresh the SSL certificates.
 * We must close old agent and create new one each time cerificate change.
 * Close old agent after 2 minutes so that any transient connections will be serviced.
 * If still not finished we will return error
 */
func (ap *AgentProvider) CreateAgent() error {
	config := *ap.client.config
	config.SecurityConfig.UseTLS = (ap.client.TLSRootCAs() != nil)
	config.UserAgent = ap.bucketName
	config.BucketName = ap.bucketName

	agent, err := gocbcore.CreateAgent(&config)
	if err != nil {
		return err
	}

	if _WARMUP && config.BucketName != "" {
		// Warm up by calling wait until ready
		warmWaitCh := make(chan struct{}, 1)
		if _, werr := agent.WaitUntilReady(
			time.Now().Add(_WARMUPTIMEOUT),
			gocbcore.WaitUntilReadyOptions{},
			func(result *gocbcore.WaitUntilReadyResult, cerr error) {
				if cerr != nil {
					err = cerr
				}
				warmWaitCh <- struct{}{}
			}); werr != nil && err == nil {
			err = werr
		}
		<-warmWaitCh
	}

	ap.mutex.Lock()
	defer ap.mutex.Unlock()
	ap.provider = agent
	return nil
}

func (ap *AgentProvider) Refresh() error {
	agent := ap.Agent()
	config := ap.client.config
	useTLS := ap.client.TLSRootCAs() != nil
	if useTLS {
		defer logging.Infof("Agent (%s) certificates have been refreshed.", agent.BucketName())
	} else {
		defer logging.Infof("Agent (%s) certificates have been turned off.", agent.BucketName())
	}
	return agent.ReconfigureSecurity(gocbcore.ReconfigureSecurityOptions{
		UseTLS:            useTLS,
		TLSRootCAProvider: config.SecurityConfig.TLSRootCAProvider,
		Auth:              config.SecurityConfig.Auth,
	})
}

func (ap *AgentProvider) Agent() *gocbcore.Agent {
	ap.mutex.RLock()
	defer ap.mutex.RUnlock()
	return ap.provider
}

func (ap *AgentProvider) Close() error {
	if ap.client != nil && ap.Agent().BucketName() != "" {
		ap.client.RemoveAtrLocation(ap.Agent().BucketName())
	}
	return ap.Agent().Close()
}

func (ap *AgentProvider) Deadline(d time.Time, n int) time.Time {
	if d.IsZero() {
		return time.Now().Add(time.Duration(n) * _KVTIMEOUT)
	}
	return d
}

// Create annotated value

func (ap *AgentProvider) getTxAnnotatedValue(res *gctx.GetResult, key, fullName string) (av value.AnnotatedValue, err error) {
	av = value.NewAnnotatedValue(value.NewParsedValue(res.Value, false))
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
	if res.Meta != nil {
		meta["txnMeta"], err = json.Marshal(*res.Meta)
		if err != nil {
			return nil, err
		}
	}
	av.SetId(key)
	return av, nil
}

// bulk transactional get

func (ap *AgentProvider) TxGet(transaction *gctx.Transaction, fullName, bucketName, scopeName, collectionName, user string,
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
			Agent:          ap.Agent(),
			ScopeName:      scopeName,
			CollectionName: collectionName,
			Key:            []byte(item.Key),
			OboUser:        user,
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
		} else if notFoundErr ||
			!(errors.Is(item.Err, gocbcore.ErrDocumentNotFound) || errors.Is(item.Err, gctx.ErrDocumentNotFound)) {
			// handle key not found error
			// process other errors before processing PreviousOperationFailed
			if err1, ok1 := item.Err.(*gctx.TransactionOperationFailedError); ok1 &&
				errors.Is(err1.Unwrap(), gctx.ErrPreviousOperationFailed) {
				prevErr = item.Err
			} else {
				errs = append(errs, item.Err)
			}
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
	User    string
	Expiry  uint32
	Pendop  gocbcore.PendingOp
	Err     error
}

// bulk transactional write

func (ap *AgentProvider) TxWrite(transaction *gctx.Transaction, txnInternal *gctx.ManagerInternal,
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
	defer logging.Tracea(func() string { return fmt.Sprintf("=====%v=====end   TxWrite(%v)========", txId, len(wops)) })
	logging.Tracea(func() string { return fmt.Sprintf("=====%v=====begin   TxWrite(%v)========", txId, len(wops)) })

	// insert request and get results in call back
	sendInsertOne := func(wop *WriteOp) error {
		wg.Add(1)
		cerr := transaction.Insert(gctx.InsertOptions{
			Agent:          ap.Agent(),
			ScopeName:      scopeName,
			CollectionName: collectionName,
			Key:            []byte(wop.Key),
			Value:          wop.Data,
			OboUser:        wop.User,
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
		logging.Debuga(func() string {
			return fmt.Sprintf("TxWrite txid=%s, op=%s, collection=%s.%s.%s, key=%s", txId,
				_MutateOpNames[op.Op], ap.Agent().BucketName(), scopeName, collectionName, op.Key)
		})
		switch op.Op {
		case MOP_INSERT:
			errOut = sendInsertOne(op)
		case MOP_UPDATE:
			var txnMeta *gctx.MutableItemMeta
			if len(op.TxnMeta) > 0 {
				txnMeta = &gctx.MutableItemMeta{}
				errOut = json.Unmarshal(op.TxnMeta, &txnMeta)
			}
			if errOut == nil {
				tmpRes := txnInternal.CreateGetResult(gctx.CreateGetResultOptions{
					Agent:          ap.Agent(),
					ScopeName:      scopeName,
					CollectionName: collectionName,
					Key:            []byte(op.Key),
					Cas:            gocbcore.Cas(op.Cas),
					Meta:           txnMeta,
					OboUser:        op.User,
				})
				errOut = sendUpdateOne(op, tmpRes)
			}
		case MOP_DELETE:
			var txnMeta *gctx.MutableItemMeta
			if len(op.TxnMeta) > 0 {
				txnMeta = &gctx.MutableItemMeta{}
				errOut = json.Unmarshal(op.TxnMeta, &txnMeta)
			}
			if errOut == nil {
				tmpRes := txnInternal.CreateGetResult(gctx.CreateGetResultOptions{
					Agent:          ap.Agent(),
					ScopeName:      scopeName,
					CollectionName: collectionName,
					Key:            []byte(op.Key),
					Cas:            gocbcore.Cas(op.Cas),
					Meta:           txnMeta,
					OboUser:        op.User,
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
			// process other errors before processing PreviousOperationFailed
			if err1, ok1 := op.Err.(*gctx.TransactionOperationFailedError); ok1 &&
				errors.Is(err1.Unwrap(), gctx.ErrPreviousOperationFailed) {
				prevErr = op.Err
			} else {
				return op.Err
			}
		}
	}

	return prevErr
}
