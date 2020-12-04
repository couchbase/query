//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

// +build enterprise

package couchbase

import (
	"encoding/json"
	gerrors "errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/couchbase/gocbcore/v9"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/couchbase/gcagent"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/transactions"
	"github.com/couchbase/query/value"
	gctx "github.com/couchbaselabs/gocbcore-transactions"
)

func (s *store) StartTransaction(stmtAtomicity bool, context datastore.QueryContext) (dks map[string]bool, err errors.Error) {
	txContext, _ := context.GetTxContext().(*transactions.TranContext)
	if txContext == nil {
		return
	}

	if txContext.TxExpired() {
		return nil, errors.NewTransactionExpired()
	}

	// Initalize  gocbcore-transactions first time
	if s.gcClient == nil {
		if err = initGocb(s); err != nil {
			return
		}
	}

	txMutations, _ := txContext.TxMutations().(*TransactionMutations)
	if stmtAtomicity {
		// statement level atomicity
		dks = make(map[string]bool, 8)
		if dks == nil {
			return nil, errors.NewMemoryAllocationError("StartTransaction()")
		}
		if txMutations != nil {
			// Get Delta keyspace names with in the transaction
			err = txMutations.DeltaKeyspaces(dks)
		}
		return
	} else {
		// Actual start transaction
		// Initalize new transaction mutations
		txMutations, err = NewTransactionMutations(txContext.TxImplicit(), txContext.MemoryQuota())
		if err != nil {
			return
		}

		defer func() {
			// protect from the panics
			if r := recover(); r != nil {
				err = errors.NewStartTransactionError(fmt.Errorf("Panic: %v", r), nil)
			}
		}()

		gcAgentTxs := s.gcClient.Transactions()
		if gcAgentTxs == nil {
			return nil, errors.NewStartTransactionError(gcagent.ErrNoInitTransactions, nil)
		}

		txnData := txContext.TxData()
		var transaction *gctx.Transaction
		var expiryTime time.Time

		resume, terr := isResumeTransaction(txnData)
		if terr != nil {
			return nil, errors.NewStartTransactionError(terr, nil)
		}

		if resume {
			transaction, terr = gcAgentTxs.ResumeTransactionAttempt(txnData)
			expiryTime = time.Now().Add(txContext.TxTimeout())
		} else {
			txConfig := &gctx.PerTransactionConfig{ExpirationTime: txContext.TxTimeout(),
				DurabilityLevel: gctx.DurabilityLevel(txContext.TxDurabilityLevel()),
			}

			txConfig.CustomATRLocation.ScopeName, txConfig.CustomATRLocation.CollectionName,
				txConfig.CustomATRLocation.Agent, terr = AtrCollectionAgentPovider(txContext.AtrCollection())
			if terr != nil {
				return nil, errors.NewStartTransactionError(terr, nil)
			}

			transaction, terr = gcAgentTxs.BeginTransaction(txConfig)
			if terr == nil {
				terr = transaction.NewAttempt()
				expiryTime = time.Now().Add(txContext.TxTimeout())
			}
		}

		// no detach for resume
		if terr != nil {
			e, c := errorType(terr)
			return nil, errors.NewStartTransactionError(e, c)
		}

		if resume {
			var dataSize int64
			for _, mutation := range transaction.GetMutations() {
				var op MutateOp
				switch mutation.OpType {
				case gctx.StagedMutationInsert:
					op = MOP_INSERT
				case gctx.StagedMutationReplace:
					op = MOP_UPDATE
				case gctx.StagedMutationRemove:
					op = MOP_DELETE
				default:
					continue
				}
				qualifiedName := "default:" + mutation.BucketName + "." +
					mutation.ScopeName + "." + mutation.CollectionName

				dataSize = int64(len(mutation.Staged))
				_, err = txMutations.Add(op, qualifiedName, mutation.BucketName, mutation.ScopeName,
					mutation.CollectionName, uint32(0),
					string(mutation.Key), mutation.Staged, uint64(mutation.Cas), uint32(0), uint32(0),
					nil, nil, nil, dataSize)
				if err != nil {
					return
				}
			}
		}
		txMutations.SetTransaction(transaction, gcAgentTxs.Internal())
		txContext.SetTxMutations(txMutations)
		txContext.SetTxId(transaction.Attempt().ID, expiryTime)
	}

	return
}

func (s *store) CommitTransaction(stmtAtomicity bool, context datastore.QueryContext) (errOut errors.Error) {
	txContext, _ := context.GetTxContext().(*transactions.TranContext)
	if txContext == nil {
		return nil
	}

	if txContext.TxExpired() {
		return errors.NewTransactionExpired()
	}

	txMutations, _ := txContext.TxMutations().(*TransactionMutations)
	if txMutations == nil {
		return nil
	}

	if stmtAtomicity {
		// Statement level atomicity.
		return txMutations.MergeDeltaKeyspace()
	}

	var err, cerr error

	transaction := txMutations.Transaction()
	txId := transaction.Attempt().ID
	logging.Tracef("=====%v=====Commit begin write========", txId)
	// write all mutations to KV
	if err = txMutations.Write(context.GetReqDeadline()); err != nil {
		e, c := errorType(err)
		return errors.NewCommitTransactionError(e, c)
	}
	logging.Tracef("=====%v=====Commit end   write========", txId)

	if transaction != nil {
		var wg sync.WaitGroup

		defer func() {
			// protect from the panics
			if r := recover(); r != nil {
				errOut = errors.NewCommitTransactionError(fmt.Errorf("Panic: %v", r), nil)
			}
		}()

		logging.Tracef("=====%v=====Actual Commit begin========", txId)
		wg.Add(1)
		err = transaction.Commit(func(resErr error) {
			defer wg.Done()
			cerr = resErr
		})

		if err == nil {
			wg.Wait()
			if cerr != nil {
				err = cerr
			}
		}

		logging.Tracef("=====%v=====Actual Commit end==========", txId)

		txMutations.SetTransaction(nil, nil)
	} else {
		err = gcagent.ErrNoTransaction
	}

	// Release transaction mutations
	var memSize int64
	txMutations.DeleteAll(true, &memSize)
	txContext.SetTxMutations(nil)

	if err != nil {
		e, c := errorType(err)
		return errors.NewCommitTransactionError(e, c)
	}

	return nil
}

func (s *store) RollbackTransaction(stmtAtomicity bool, context datastore.QueryContext, sname string) (errOut errors.Error) {
	txContext, _ := context.GetTxContext().(*transactions.TranContext)
	if txContext == nil {
		return nil
	}

	if txContext.TxExpired() {
		return errors.NewTransactionExpired()
	}

	txMutations, _ := txContext.TxMutations().(*TransactionMutations)
	if txMutations == nil {
		return nil
	}

	if !txMutations.TranImplicit() && (stmtAtomicity || sname != "") {
		// Statement level atomicity or savepoint rollback
		slog, sindex, err := txMutations.GetSavepointRange(sname)
		if err == nil {
			err = txMutations.UndoLog(slog, sindex)
		}
		return err
	}

	var err, cerr error

	transaction := txMutations.Transaction()
	if transaction != nil {
		var wg sync.WaitGroup

		defer func() {
			// protect from the panics
			if r := recover(); r != nil {
				errOut = errors.NewRollbackTransactionError(fmt.Errorf("Panic: %v", r), nil)
			}
		}()

		wg.Add(1)
		err = transaction.Rollback(func(resErr error) {
			defer wg.Done()
			cerr = resErr
		})

		if err == nil {
			wg.Wait()
			if cerr != nil {
				err = cerr
			}
		}

		txMutations.SetTransaction(nil, nil)
	} else {
		err = gcagent.ErrNoTransaction
	}

	var memSize int64
	txMutations.DeleteAll(true, &memSize)
	txContext.SetTxMutations(nil)

	if err != nil {
		e, c := errorType(err)
		return errors.NewRollbackTransactionError(e, c)
	}

	return nil
}

// Delta keyspace scan
func (s *store) TransactionDeltaKeyScan(keyspace string, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()
	var keys map[string]bool
	var err errors.Error

	if context := conn.QueryContext(); context != nil {
		if txContext, _ := context.GetTxContext().(*transactions.TranContext); txContext != nil {
			if txMutations, _ := txContext.TxMutations().(*TransactionMutations); txMutations != nil {
				keys, err = txMutations.GetDeltaKeyspaceKeys(keyspace)
				if err != nil {
					conn.Fatal(err)
					return
				}
			}
		}
	}

	for k, ok := range keys {
		ie := &datastore.IndexEntry{PrimaryKey: k}
		if ok {
			ie.MetaData = value.NULL_VALUE
		}
		if !conn.Sender().SendEntry(ie) {
			return
		}
	}
}

func (s *store) SetSavepoint(stmtAtomicity bool, context datastore.QueryContext, sname string) errors.Error {
	if sname == "" {
		return nil
	}

	txContext, _ := context.GetTxContext().(*transactions.TranContext)
	if txContext == nil {
		return nil
	}

	if txContext.TxExpired() {
		return errors.NewTransactionExpired()
	}

	txMutations, _ := txContext.TxMutations().(*TransactionMutations)
	if txMutations == nil {
		return nil
	}

	return txMutations.SetSavepoint(sname)
}

func (ks *keyspace) txReady(txContext *transactions.TranContext) errors.Error {
	if txContext != nil && txContext.TxExpired() {
		return errors.NewTransactionExpired()
	}

	// gocbcore agent is present
	if ks.agentProvider != nil {
		return nil
	}

	ks.Lock()
	defer ks.Unlock()

	if ks.agentProvider != nil {
		return nil
	}

	// create gocbcore agent
	var err error
	ks.agentProvider, err = ks.namespace.store.gcClient.CreateAgentProvider(ks.name)
	if err != nil {
		return errors.NewError(err, "gcagent agent creation failed")
	}
	return nil
}

func (ks *keyspace) txFetch(fullName, qualifiedName, scopeName, collectionName string, collId uint32, keys []string,
	fetchMap map[string]value.AnnotatedValue, context datastore.QueryContext, subPaths []string, sdkKvInsert bool,
	txContext *transactions.TranContext) errors.Errors {

	err := ks.txReady(txContext)
	if err != nil {
		return errors.Errors{err}
	}

	var transaction *gctx.Transaction
	fkeys := keys
	sdkKv, sdkCas, sdkTxnMeta := GetTxDataValues(context.TxDataVal())
	if txMutations, _ := txContext.TxMutations().(*TransactionMutations); txMutations != nil {
		var err errors.Error
		mvs := make(map[string]*MutationValue, len(keys))
		transaction = txMutations.Transaction()

		// Fetch the keys from delta  keyspace
		fkeys, err = txMutations.Fetch(qualifiedName, keys, mvs)
		if err != nil {
			return errors.Errors{err}
		}

		if sdkKv && sdkCas != 0 && len(keys) == 1 {
			// Transformed SDK REPLACE, DELETE with CAS don't read the document
			k := keys[0]
			if len(fkeys) == 0 && txMutations.IsDeletedMutation(qualifiedName, k) {
				return errors.Errors{errors.NewKeyNotFoundError(k, nil)}
			} else if len(fkeys) == 1 {
				mvs[k] = &MutationValue{Val: value.NewValue(nil), Cas: sdkCas, TxnMeta: sdkTxnMeta}
				fkeys = fkeys[0:0]
			}
		}

		for k, mv := range mvs {
			av := value.NewAnnotatedValue(mv.Val)
			meta := av.NewMeta()
			meta["keyspace"] = fullName
			meta["cas"] = mv.Cas
			meta["type"] = "json"
			meta["flags"] = uint32(0)
			meta["expiration"] = mv.Expiration
			if mv.TxnMeta != nil {
				meta["txnMeta"] = mv.TxnMeta
			}
			av.SetId(k)
			fetchMap[k] = av
		}
	}

	if len(fkeys) > 0 {
		// Transformed SDK operation, don't ignore key not found error (except insert check)
		notFoundErr := sdkKv && !sdkKvInsert
		// fetch the keys that are not present in delta keyspace
		errs := ks.agentProvider.TxGet(transaction, fullName, ks.name, scopeName, collectionName,
			collId, fkeys, subPaths, context.GetReqDeadline(), false, notFoundErr, fetchMap)
		if len(errs) > 0 {
			if notFoundErr && gerrors.Is(errs[0], gocbcore.ErrDocumentNotFound) {
				_, c := errorType(errs[0])
				return errors.Errors{errors.NewKeyNotFoundError(fkeys[0], c)}
			}

			var rerrs errors.Errors
			for _, e := range errs {
				e1, c := errorType(e)
				rerr := errors.NewError(e1, "txFetch")
				if c != nil {
					rerr.SetCause(c)
				}
				rerrs = append(rerrs, rerr)
			}
			return rerrs
		}
	}

	return nil
}

func (ks *keyspace) txPerformOp(op MutateOp, qualifiedName, scopeName, collectionName string, collId uint32, pairs value.Pairs,
	context datastore.QueryContext, txContext *transactions.TranContext) (
	mPairs value.Pairs, err errors.Error) {

	err = ks.txReady(txContext)
	if err != nil {
		return
	}

	txMutations := txContext.TxMutations().(*TransactionMutations)
	var fetchMap map[string]value.AnnotatedValue
	sdkKv, sdkCas, _ := GetTxDataValues(context.TxDataVal())
	sdkKvInsert := sdkKv && op == MOP_INSERT

	if op == MOP_UPSERT || sdkKvInsert {
		// SDK INSERT check key in KV by reading
		// UPSERT check keys and transform to INSERT or UPDATE

		fetchMap = make(map[string]value.AnnotatedValue, len(pairs))
		fkeys := make([]string, 0, len(pairs))
		for _, kv := range pairs {
			fkeys = append(fkeys, kv.Name)
		}
		errs := ks.txFetch("", qualifiedName, scopeName, collectionName, collId,
			fkeys, fetchMap, context, nil, sdkKvInsert, txContext)
		if len(errs) > 0 {
			return nil, errs[0]
		}
	}

	mPairs = make(value.Pairs, 0, len(pairs))
	var retCas uint64
	for _, kv := range pairs {
		var data interface{}
		var exptime uint32
		var dataSize int64

		key := kv.Name
		val := kv.Value
		nop := op

		if val != nil && val.Type() == value.BINARY {
			return nil, errors.NewBinaryDocumentMutationError(_MutateOpNames[op], key)
		}

		if op != MOP_DELETE {
			data = val.ActualForIndex()
			dataSize = int64(val.Size())
			exptime = getExpiration(kv.Options)
		}

		if op == MOP_INSERT || op == MOP_UPSERT {
			// INSERT, UPSERT transform to INSERT or UPDATE
			if av, ok := fetchMap[key]; ok {
				if op == MOP_UPSERT {
					nop = MOP_UPDATE
				} else {
					return nil, errors.NewDuplicateKeyError(key)
				}
				val = av
				kv.Value = val
			} else {
				nop = MOP_INSERT
			}
		}

		must := (nop == MOP_UPDATE || nop == MOP_DELETE)
		cas, _, txnMeta, err1 := getMeta(kv.Name, val, must)
		if err1 == nil && must {
			if sdkKv && sdkCas != cas {
				return nil, errors.NewCasMissmatch(_MutateOpNames[op], kv.Name, sdkCas, cas)
			}
		}

		if err1 != nil {
			return nil, errors.NewTransactionError(err1, _MutateOpNames[op])
		}

		if nop == MOP_INSERT {
			txnMeta = nil
		}

		// Add to mutations
		retCas, err = txMutations.Add(nop, qualifiedName, ks.name, scopeName, collectionName, collId,
			key, data, cas, MV_FLAGS_WRITE, exptime, txnMeta, nil, ks, dataSize)

		if err != nil {
			return nil, err
		}

		if retCas > 0 && !SetMetaCas(val, retCas) {
			return nil, errors.NewTransactionError(fmt.Errorf("Setting return cas error"), _MutateOpNames[op])
		}

		// upsert and not already in the fetchMap then add so that same upsert key will make it update in same statement
		if op == MOP_UPSERT {
			if _, ok := fetchMap[key]; !ok {
				fetchMap[key] = val.(value.AnnotatedValue)
			}
		}

		mPairs = append(mPairs, kv)
	}

	if txMutations.TranImplicit() {
		// implict transaction write the current batch
		if terr := txMutations.Write(context.GetReqDeadline()); terr != nil {
			return nil, errors.NewError(terr, "write error")
		}
	}

	return
}

func GetTxDataValues(txDataVal value.Value) (kv bool, cas uint64, txnMeta interface{}) {
	if txDataVal != nil {
		if v, ok := txDataVal.Field("kv"); ok {
			kv, _ = v.Actual().(bool)
		}

		if v, ok := txDataVal.Field("cas"); ok && v.Type() == value.NUMBER {
			cas = uint64(value.AsNumberValue(v).Int64())
		}

		if v, ok := txDataVal.Field("scas"); ok && v.Type() == value.STRING {
			s, _ := v.Actual().(string)
			if u64, err := strconv.ParseUint(s, 10, 64); err == nil {
				cas = u64
			}
		}

		if v, ok := txDataVal.Field("txnMeta"); ok && v.Type() != value.MISSING {
			txnMeta, _ = v.MarshalJSON()
		}
	}
	return
}

func isResumeTransaction(b []byte) (bool, error) {
	if len(b) == 0 {
		return false, nil
	}

	type jsonSerializedAttempt struct {
		ID struct {
			Transaction string `json:"txn"`
			Attempt     string `json:"atmpt"`
		} `json:"id"`
	}

	var txData jsonSerializedAttempt

	if err := json.Unmarshal(b, &txData); err != nil {
		return false, err
	}

	return txData.ID.Transaction != "", nil
}

func AtrCollectionAgentPovider(atrCollection string) (string, string, *gocbcore.Agent, error) {
	if atrCollection == "" {
		return "", "", nil, nil
	}
	path, err := algebra.NewVariablePathWithContext(atrCollection, "default", "")
	if err != nil {
		return "", "", nil, err
	}

	agent, cerr := CollectionAgentProvider(path.Bucket(), path.Scope(), path.Keyspace())
	return path.Scope(), path.Keyspace(), agent, cerr
}

func CollectionAgentProvider(bucketName, scpName, collName string) (agent *gocbcore.Agent, rerr error) {
	if bucketName == "" || scpName == "" || collName == "" {
		return nil, fmt.Errorf("Not valid collection : `%v`.`%v`.`%v`", bucketName, scpName, collName)
	}

	ks, cerr := datastore.GetKeyspace("default", bucketName, scpName, collName)
	if cerr != nil {
		return nil, cerr
	}

	coll, ok := ks.(*collection)
	if !ok {
		return nil, fmt.Errorf("%v is not a collection", ks.QualifiedName())
	}

	if cerr = coll.bucket.txReady(nil); cerr != nil {
		return nil, cerr.GetICause()
	}
	return coll.bucket.agentProvider.Agent(), nil
}

func errorType(err error) (error, interface{}) {
	if terr, ok := err.(*gctx.TransactionOperationFailedError); ok {
		// TODO:: Until gocbcore-transactions provides API populate the info.
		c := make(map[string]interface{}, 5)
		if terr.Retry() {
			c["retry"] = terr.Retry()
		}

		if terr.Rollback() {
			c["rollback"] = terr.Rollback()
		}

		c["raise"] = terr.ToRaise()
		c["class"] = terr.ErrorClass()
		c["msg"] = terr.Unwrap().Error()

		return nil, c
	}
	return err, nil
}

func initGocb(s *store) (err errors.Error) {
	var certFile string
	if s.connSecConfig != nil && s.connSecConfig.ClusterEncryptionConfig.EncryptData {
		certFile = s.connSecConfig.CertFile
	}

	tranSettings := datastore.GetTransactionSettings()
	txConfig := &gctx.Config{
		ExpirationTime:        tranSettings.TxTimeout(),
		CleanupWindow:         tranSettings.CleanupWindow(),
		CleanupClientAttempts: tranSettings.CleanupClientAttempts(),
		CleanupLostAttempts:   tranSettings.CleanupLostAttempts(),
		BucketAgentProvider: func(bucketName string) (agent *gocbcore.Agent, rerr error) {
			return CollectionAgentProvider(bucketName, "_default", "_default")
		},
	}

	client, cerr := gcagent.NewClient(s.URL(), certFile)
	s.nslock.Lock()
	defer s.nslock.Unlock()

	if s.gcClient != nil {
		if client != nil {
			client.Close()
		}
		return
	}

	if client == nil {
		err = errors.NewError(cerr, "gcagent client initalization failed")
		logging.Errorf(err.Error())
		return err
	}

	s.gcClient = client

	// don't raise error not able to setup ATR Collection. Disable time being

	//	txConfig.CustomATRLocation.ScopeName, txConfig.CustomATRLocation.CollectionName,
	//		txConfig.CustomATRLocation.Agent, _ = AtrCollectionAgentPovider(tranSettings.AtrCollection())

	logging.Infof("Transaction Initalization: ExpirationTime: %v, CleanupWindow: %v, CleanupClientAttempts: %v, CleanupLostAttempts: %v",
		txConfig.ExpirationTime, txConfig.CleanupWindow, txConfig.CleanupClientAttempts, txConfig.CleanupLostAttempts)

	cerr = client.InitTransactions(txConfig)
	if cerr != nil {
		client.Close()
		s.gcClient = nil
		return errors.NewError(cerr, "Transaction initalization failed")
	}

	return nil
}
