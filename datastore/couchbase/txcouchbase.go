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
	"fmt"
	"sync"
	"time"

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
		txMutations, err = NewTransactionMutations(txContext.TxImplicit())
		if err != nil {
			return
		}

		defer func() {
			// protect from the panics
			if r := recover(); r != nil {
				err = errors.NewStartTransactionError(fmt.Errorf("Panic: %v", r))
			}
		}()

		gcAgentTxs := s.gcClient.Transactions()
		if gcAgentTxs == nil {
			return nil, errors.NewStartTransactionError(gcagent.ErrNoInitTransactions)
		}

		txnData := txContext.TxData()
		var transaction *gctx.Transaction
		var terr error
		var expiryTime time.Time

		if len(txnData) > 0 {
			transaction, terr = gcAgentTxs.ResumeTransactionAttempt(txnData)
			expiryTime = time.Now().Add(txContext.TxTimeout())
		} else {
			txConfig := &gctx.PerTransactionConfig{ExpirationTime: txContext.TxTimeout(),
				DurabilityLevel:  gctx.DurabilityLevel(txContext.TxDurabilityLevel()),
				KvDurableTimeout: txContext.TxDurabilityTimeout()}

			transaction, terr = gcAgentTxs.BeginTransaction(txConfig)
			if terr == nil {
				terr = transaction.NewAttempt()
				expiryTime = time.Now().Add(txContext.TxTimeout())
			}
		}
		if terr != nil {
			return nil, errors.NewStartTransactionError(terr)
		}

		txMutations.SetTransaction(transaction, gcAgentTxs.Internal())
		txContext.SetTxMutations(txMutations)
		txContext.SetTxId(transaction.Attempt().ID, expiryTime)
		if len(txnData) > 0 {
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

				_, err = txMutations.Add(op, qualifiedName, mutation.BucketName, mutation.ScopeName,
					mutation.CollectionName, uint32(0),
					string(mutation.Key), mutation.Staged, uint64(mutation.Cas), uint32(0), uint32(0),
					nil, nil, nil)
				if err != nil {
					return
				}
			}
		}
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
	var diag interface{}

	transaction := txMutations.Transaction()
	txId := transaction.Attempt().ID
	logging.Tracef("=====%v=====Commit begin write========", txId)
	// write all mutations to KV
	if err = txMutations.Write(context.GetReqDeadline()); err != nil {
		return errors.NewCommitTransactionError(err, diag)
	}
	logging.Tracef("=====%v=====Commit end   write========", txId)

	if transaction != nil {
		var wg sync.WaitGroup

		defer func() {
			// protect from the panics
			if r := recover(); r != nil {
				errOut = errors.NewCommitTransactionError(fmt.Errorf("Panic: %v", r), diag)
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
	txMutations.DeleteAll(true)

	if err != nil {
		return errors.NewCommitTransactionError(err, diag)
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
	var diag interface{}

	transaction := txMutations.Transaction()
	if transaction != nil {
		var wg sync.WaitGroup

		defer func() {
			// protect from the panics
			if r := recover(); r != nil {
				errOut = errors.NewRollbackTransactionError(fmt.Errorf("Panic: %v", r), diag)
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

	txMutations.DeleteAll(true)
	txContext.SetTxMutations(nil)

	if err != nil {
		return errors.NewRollbackTransactionError(err, diag)
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
	if txContext.TxExpired() {
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
	if txMutations, _ := txContext.TxMutations().(*TransactionMutations); txMutations != nil {
		var err errors.Error
		mvs := make(map[string]*MutationValue, len(keys))
		transaction = txMutations.Transaction()

		// Fetch the keys from delta  keyspace
		fkeys, err = txMutations.Fetch(qualifiedName, keys, mvs)
		if err != nil {
			return errors.Errors{err}
		}

		for k, mv := range mvs {
			av := value.NewAnnotatedValue(mv.Val)
			meta := av.NewMeta()
			meta["keyspace"] = fullName
			meta["cas"] = mv.Cas
			meta["type"] = "json"
			meta["flags"] = uint32(0)
			meta["expiration"] = mv.Expiration
			meta["txnMeta"] = mv.TxnMeta
			av.SetId(k)
			fetchMap[k] = av
		}
	}

	if len(fkeys) > 0 {
		sdkKv, sdkCas, sdkTxnMeta := GetTxDataValues(context.TxDataVal())
		if sdkKv && sdkCas != 0 && len(fkeys) == 1 && sdkTxnMeta != nil {
			// Transformed SDK REPLACE, DELETE with CAS don't read the document
			k := fkeys[0]
			av := value.NewAnnotatedValue(value.NewValue(nil))
			meta := av.NewMeta()
			meta["keyspace"] = fullName
			meta["cas"] = sdkCas
			meta["type"] = "json"
			meta["flags"] = uint32(0)
			meta["expiration"] = uint32(0)
			meta["txnMeta"] = sdkTxnMeta
			av.SetId(k)
			fetchMap[k] = av
		} else {
			// Transformed SDK operation, don't ignore key not found error (except insert check)
			notFoundErr := sdkKv && !sdkKvInsert
			// fetch the keys that are not present in delta keyspace
			errs := ks.agentProvider.TxGet(transaction, fullName, ks.name, scopeName, collectionName,
				collId, fkeys, subPaths, context.GetReqDeadline(), false, notFoundErr, fetchMap)
			if len(errs) > 0 {
				return errors.NewErrors(errs, "txFetch")
			}
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
	sdkKv, sdkCas, sdkTxnMeta := GetTxDataValues(context.TxDataVal())
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

		key := kv.Name
		val := kv.Value
		nop := op

		if op != MOP_DELETE {
			data = val.ActualForIndex()
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
			} else {
				nop = MOP_INSERT
			}
		}

		must := (nop == MOP_UPDATE || nop == MOP_DELETE)
		cas, _, txnMeta, err1 := getMeta(kv.Name, val, must)
		if err1 == nil && must {
			if txnMeta == nil || (sdkKv && sdkTxnMeta == nil) {
				err1 = fmt.Errorf("Not valid txnMeta value for key %v", kv.Name)
			} else if sdkKv && sdkCas != cas {
				err1 = fmt.Errorf("Missmatch cas values(%v,%v) for key %v", sdkCas, cas, kv.Name)
			}
		}

		if err1 != nil {
			return nil, errors.NewTransactionError(err1, _MutateOpNames[op])
		}

		if nop == MOP_INSERT {
			txnMeta = []byte("{}")
		}

		// Add to mutations
		retCas, err = txMutations.Add(nop, qualifiedName, ks.name, scopeName, collectionName, collId,
			key, data, cas, MV_FLAGS_WRITE, exptime, txnMeta, nil, ks)

		if err != nil {
			return nil, err
		}

		if retCas > 0 && !SetMetaCas(val, retCas) {
			return nil, errors.NewTransactionError(fmt.Errorf("Setting return cas error"), _MutateOpNames[op])
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

		if v, ok := txDataVal.Field("txnMeta"); ok && v.Type() == value.OBJECT {
			txnMeta, _ = v.MarshalJSON()
		}
	}
	return
}

func initGocb(s *store) (err errors.Error) {
	var certFile string
	if s.connSecConfig != nil && s.connSecConfig.ClusterEncryptionConfig.EncryptData {
		certFile = s.connSecConfig.CertFile
	}

	client, cerr := gcagent.NewClient(s.URL(), certFile, datastore.DEF_TXTIMEOUT)

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
	return nil
}
