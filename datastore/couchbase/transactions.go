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
	"sync"
	"sync/atomic"
	"time"

	"github.com/couchbase/query/datastore/couchbase/gcagent"
	"github.com/couchbase/query/errors"
	gctx "github.com/couchbaselabs/gocbcore-transactions"
)

type MutateOp int

var _MutateOpNames = [...]string{"UNKNOWN", "INSERT", "UPSERT", "UPDATE", "DELETE"}

const (
	MOP_NONE MutateOp = iota
	MOP_INSERT
	MOP_UPSERT
	MOP_UPDATE
	MOP_DELETE
)

const (
	MV_FLAGS_WRITE uint32 = 1 << iota
)

const (
	TL_NONE = iota
	TL_KEYSPACE
	TL_SAVEPOINT
	TL_DOCUMENT
)

func MutateOpToName(op MutateOp) string {
	i := int(op)
	if i < 0 || i >= len(_MutateOpNames) {
		i = 0
	}

	return _MutateOpNames[i]
}

type MutationValue struct {
	Op         MutateOp
	KvCas      uint64
	Cas        uint64
	Expiration uint32
	Flags      uint32
	Val        interface{}
	TxnMeta    interface{}
	memSize    int64
}

type DeltaKeyspace struct {
	ks             *keyspace
	bucketName     string
	scopeName      string
	collectionName string
	collId         uint32
	values         map[string]*MutationValue
}

type TransactionLogValue struct {
	logType       int
	key           string
	oldOp         MutateOp
	oldKvCas      uint64
	oldCas        uint64
	oldExpiration uint32
	oldFlags      uint32
	oldVal        interface{}
	oldTxnMeta    interface{}
	oldMemSize    int64
}

type TransactionLog struct {
	lastKeyspace string
	logValues    []*TransactionLogValue
}

type TransactionMutations struct {
	logSize          int
	mutex            sync.RWMutex
	transaction      *gctx.Transaction
	txnInternal      *gctx.TransactionsInternal
	tranImplicit     bool
	savepoints       map[string]uint64
	keyspaces        map[string]*DeltaKeyspace
	logs             []*TransactionLog
	curStartLogIndex uint64
	curDeltaKeyspace DeltaKeyspace
	curKeyspace      string
	curLog           int
	usedMemory       int64
	memoryQuota      uint64
}

const (
	_DK_DEF_SIZE             = 128
	_TM_DEF_LOGSIZE          = 128 //fixed log size
	_TM_DEF_SAVEPOINTS       = 4
	_TM_DEF_KEYSPACES        = 4
	_WRITE_BATCH_SIZE        = 16
	_SZ                      = int64(8)
	_TRANSACTIONMUTATIONS_SZ = int64(128)
	_DELTAKEYSPACE_SZ        = int64(64)
	_TRANSACTIONLOGVALUE_SZ  = int64(64)
	_MUTATIONVALUE_SZ        = int64(64)
)

/* New Mutations structure. One per transaction
 */

func NewTransactionMutations(implicit bool, memoryQuota uint64) (rv *TransactionMutations, err errors.Error) {
	memSize := _TRANSACTIONMUTATIONS_SZ
	rv = &TransactionMutations{logSize: _TM_DEF_LOGSIZE,
		tranImplicit: implicit,
		memoryQuota:  memoryQuota}

	if rv != nil {
		rv.curDeltaKeyspace.values = make(map[string]*MutationValue, _DK_DEF_SIZE)
		if rv.curDeltaKeyspace.values == nil {
			return nil, errors.NewMemoryAllocationError("TransactionMutations.DeltaKeyspaces()")
		}
		memSize += _DK_DEF_SIZE * _SZ
		if !implicit {
			rv.savepoints = make(map[string]uint64, _TM_DEF_SAVEPOINTS)
			rv.keyspaces = make(map[string]*DeltaKeyspace, _TM_DEF_KEYSPACES)
			if rv.savepoints == nil || rv.keyspaces == nil {
				return nil, errors.NewMemoryAllocationError("TransactionMutations.DeltaKeyspaces()")
			}
			memSize += (_TM_DEF_SAVEPOINTS + _TM_DEF_KEYSPACES) * _SZ
		}
		err = rv.TrackMemoryQuota(memSize)
	}

	return rv, err
}

func (this *TransactionMutations) LogSize() int {
	return this.logSize
}

/* gocbcore-transaction
 */

func (this *TransactionMutations) SetTransaction(transaction *gctx.Transaction, txnInternal *gctx.TransactionsInternal) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.transaction = transaction
	this.txnInternal = txnInternal
}

/* gocbcore-transaction
 */

func (this *TransactionMutations) Transaction() *gctx.Transaction {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.transaction
}

func (this *TransactionMutations) TransactionsInternal() *gctx.TransactionsInternal {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.txnInternal
}

func (this *TransactionMutations) TranImplicit() bool {
	// lock is not required. Only set at start
	return this.tranImplicit
}

/* List of delta keyspace names with in the transaction
 */

func (this *TransactionMutations) DeltaKeyspaces(dks map[string]bool) (err errors.Error) {
	// lock is not required. Only set at start
	if this.tranImplicit {
		return
	}

	this.mutex.RLock()
	defer this.mutex.RUnlock()

	for k, dk := range this.keyspaces {
		if dk != nil && len(dk.values) > 0 {
			dks[k] = true
		}
	}

	return
}

/* Absolute log position of transaction log
 */

func (this *TransactionMutations) TotalMutations() (tm uint64) {
	nLogs := len(this.logs)
	if nLogs > 0 {
		tm = uint64((this.logSize * (nLogs - 1)) + this.logs[nLogs-1].Len())
	}
	return tm
}

/* Set savepoint. Overwrite if already exist
 */

func (this *TransactionMutations) SetSavepoint(sname string) (err errors.Error) {
	// lock is not required. Only set at start. No Savepoints for implicit transaction.
	if this.tranImplicit {
		return nil
	}

	this.mutex.Lock()
	defer this.mutex.Unlock()

	// set log position and Add savepoint marker to transaction log
	this.savepoints[sname] = this.TotalMutations()
	return this.AddMarker(sname, TL_SAVEPOINT)
}

/* Given savepoint, get transaction log number, and position in the log.
 * If name == "" use as keyspace marker position
 */

func (this *TransactionMutations) GetSavepointRange(sname string) (slog, sindex uint64, err errors.Error) {
	// lock is not required. Only set at start. No Savepoints for implicit transaction.
	if this.tranImplicit {
		return
	}

	this.mutex.RLock()
	defer this.mutex.RUnlock()

	var ok bool
	if sname != "" {
		sindex, ok = this.savepoints[sname]
		if !ok {
			// If Actual savepoint not present error
			return slog, sindex, errors.NewNoSavepointError(sname)
		}
	} else {
		sindex = this.curStartLogIndex
	}

	slog = sindex / uint64(this.logSize)
	sindex %= uint64(this.logSize)
	return
}

/* Mutations Fetch
 *   Returns mutation values from delta keyspace
 *           keys that are not part of delta keyspace.
 */
func (this *TransactionMutations) Fetch(keyspace string, keys []string, mvs map[string]*MutationValue) (
	rkeys []string, err errors.Error) {
	// lock is not required. Only set at start. can't read local mutations for implicit transaction.
	if this.tranImplicit {
		return keys, nil
	}

	this.mutex.RLock()
	defer this.mutex.RUnlock()

	dk, _ := this.keyspaces[keyspace]
	if dk == nil {
		// no delta keyspace, get all keys from KV
		return keys, nil
	}

	rkeys = make([]string, 0, len(keys))
	for _, k := range keys {
		if mv, ok := dk.values[k]; ok && (mv.Flags&MV_FLAGS_WRITE) != 0 {
			// consider only n1ql mutated  keys
			// delta keyspace has entry. ignore deleted key
			if mv.Op != MOP_DELETE && mv.Op != MOP_NONE {
				mvs[k] = mv
			}
		} else {
			// keys that are not part of delta keyspace. Will be fetched from KV
			rkeys = append(rkeys, k)
		}
	}

	return rkeys, nil
}

// Document Deleted returns true

func (this *TransactionMutations) IsDeletedMutation(keyspace string, key string) bool {

	// lock is not required. Only set at start. can't read local mutations for implicit transaction.
	if this.tranImplicit {
		return false
	}

	this.mutex.RLock()
	defer this.mutex.RUnlock()

	if dk, _ := this.keyspaces[keyspace]; dk != nil {
		if mv, ok := dk.values[key]; ok && mv.Op == MOP_DELETE {
			return true
		}
	}

	return false
}

/*
 Add the entries to transaction mutations.
     current Delta keysapce
     Transaction log
     KV has INSERT, UPDATE, DELETE ops separate and we are staging localy we must go through transformation
     and protect original operation. ALso need to preserve orginal CAS.

prev  +  cur     --->  future                  SDK-Mutations
------------------------------------------------------------
INSERT   INSERT   ---  error                   error
         UPSERT   ---  INSERT                  UPDATE
         UPDATE   ---  INSERT                  UPDATE
         DELETE   ---  Remove with 0 cas       DELETE
UPSERT   INSERT   ---  error                   error
         UPSERT   ---- UPSERT                  UPDATE
         UPDATE   ---- UPSERT                  UPDATE
         DELETE   ---- DELETE                  DELETE
UPDATE   INSERT   ---  error                   error
         UPSERT   ---- UPDATE                  UPDATE
         UPDATE   ---- UPDATE                  UPDATE
         DELETE   ---- DELETE                  DELETE
DELETE   INSERT   ---  UPDATE with cas  *      INSERT
         UPSERT   ---- UPDATE with cas  *      INSERT
         UPDATE   ---- N/A                     N/A
         DELETE   ---- N/A                     N/A
*/

func (this *TransactionMutations) Add(op MutateOp, keyspace, bucketName, scopeName, collectionName string,
	collId uint32, key string, val interface{}, cas uint64, flags, exptime uint32, txnMeta interface{},
	paths []string, ks *keyspace, valSize int64) (retCas uint64, err errors.Error) {

	this.mutex.Lock()
	defer this.mutex.Unlock()

	var mv, mmv *MutationValue
	var dk, mdk *DeltaKeyspace
	var addMarker bool
	var memSize int64

	mdk, _ = this.keyspaces[keyspace]

	if (flags & MV_FLAGS_WRITE) != 0 {
		// Get mutation value from current delta keyspace (current statement)
		dk = &this.curDeltaKeyspace
		mv = dk.Get(key)
	} else {
		if mdk == nil {
			mdk = &DeltaKeyspace{
				values:         make(map[string]*MutationValue, _DK_DEF_SIZE),
				ks:             ks,
				collId:         collId,
				bucketName:     bucketName,
				scopeName:      scopeName,
				collectionName: collectionName}
			if mdk.values == nil {
				return retCas, errors.NewMemoryAllocationError("TransactionMutations.AddToDeltaKeyspace()")
			}
			this.keyspaces[keyspace] = mdk
			memSize += _DELTAKEYSPACE_SZ + _SZ*_DK_DEF_SIZE
		}
		dk = mdk
	}

	// Get mutation value from keyspace from previous statements
	if mdk != nil {
		mmv = mdk.Get(key)
		if mmv != nil {
			if mmv.KvCas != 0 {
				cas = mmv.KvCas // new CAS value becoms original CAS value
			}
			if mmv.TxnMeta != nil {
				txnMeta = mmv.TxnMeta // new txnMeta value becomes original TxnMeta value
			}
		}
	}

	switch op {
	case MOP_INSERT:
		// Inserted key present current statement or previous statement error.
		if mv != nil || (mmv != nil && (mmv.Op == MOP_INSERT || mmv.Op == MOP_UPSERT || mmv.Op == MOP_UPDATE)) {
			return retCas, errors.NewDuplicateKeyError(key)
		}

		// Previous statement has MOP_DELETE and non zero CAS transform to MOP_UPDATE
		if mmv != nil && mmv.Op == MOP_DELETE && cas != 0 && (mmv.Flags&MV_FLAGS_WRITE) != 0 {
			op = MOP_UPDATE
		}

	case MOP_UPSERT:
		if mmv != nil {
			if (mmv.Flags & MV_FLAGS_WRITE) != 0 {
				if mmv.Op == MOP_INSERT || mmv.Op == MOP_UPDATE {
					// Previous statement has MOP_INSERT, MOP_UPDATE retain previous Operation
					op = mmv.Op
				} else if mmv.Op == MOP_DELETE && cas != 0 {
					// Previous statement has MOP_DELETE and non zero CAS transform to MOP_UPDATE
					op = MOP_UPDATE
				}
			} else if mmv.Op == MOP_DELETE {
				op = MOP_INSERT
			} else {
				op = MOP_UPDATE
			}
		}
	case MOP_UPDATE:
		if mmv != nil && (mmv.Op == MOP_INSERT || mmv.Op == MOP_UPSERT) && (mmv.Flags&MV_FLAGS_WRITE) != 0 {
			// Previous statement has MOP_INSERT, MOP_UPSERT retain previous Operation
			op = mmv.Op
		}
	case MOP_DELETE:

	default:
		return retCas, nil
	}

	// If curKeyspace and keyspace is different store the info in current delta keyspace (Statement switch)
	if this.curKeyspace != keyspace && (flags&MV_FLAGS_WRITE) != 0 {
		this.curKeyspace = keyspace
		addMarker = true
		dk.ks = ks
		dk.bucketName = bucketName
		dk.scopeName = scopeName
		dk.collectionName = collectionName
		dk.collId = collId
	}

	if len(this.savepoints) > 0 {
		/* Savepoints present then only use transaction log.
		   Otherwise statement level atomicity handled by current delta keyspace
		*/
		if addMarker {
			// Add keyspace marker to transaction log
			this.curStartLogIndex = this.TotalMutations()
			if err = this.AddMarker(keyspace, TL_KEYSPACE); err != nil {
				return retCas, err
			}
		}

		// Add document to transaction log
		var tl *TransactionLog
		if tl, err = this.SetCurLog(); err == nil && tl != nil {
			err = tl.Add(key, mmv, TL_DOCUMENT, &memSize)

		}
	}

	if err == nil {
		// Add mutation value to current delta keyspace
		retCas = uint64(time.Now().UTC().UnixNano())
		mv = &MutationValue{Op: op, Val: val, Cas: retCas, KvCas: cas, Expiration: exptime, Flags: flags, TxnMeta: txnMeta}
		if mv == nil {
			return retCas, errors.NewMemoryAllocationError("TransactionMutations.Add()")
		}

		if len(this.savepoints) == 0 && mmv != nil {
			memSize -= mmv.memSize
		} else {
			memSize += _MUTATIONVALUE_SZ + int64(len(key))
		}

		var b []byte
		if mv.TxnMeta != nil {
			b, _ = mv.TxnMeta.([]byte)
		}

		// Even though key is not part of mv include so that map part is covered
		mv.memSize = valSize + int64(len(b))
		memSize += mv.memSize

		if err = this.TrackMemoryQuota(memSize); err != nil {
			return retCas, err
		}

		dk.Add(key, mv)
	}

	return retCas, err
}

/* Set current Log position
   Allocate log. If slice limit is reached allocate new log.
*/

func (this *TransactionMutations) SetCurLog() (tl *TransactionLog, err errors.Error) {

	if this.logs == nil || this.logs[this.curLog].Len() == this.logSize {
		tl := &TransactionLog{logValues: make([]*TransactionLogValue, 0, this.logSize)}
		if tl == nil || tl.logValues == nil {
			return nil, errors.NewMemoryAllocationError("TransactionMutations.SetCurLog()")
		}
		this.logs = append(this.logs, tl)
		if this.logs[this.curLog].Len() == this.logSize {
			this.curLog++
		}
	}

	return this.logs[this.curLog], nil
}

/* Add TL_KEYSPACE or TL_SAVEPOINT marker to transaction log
 */
func (this *TransactionMutations) AddMarker(keyspace string, logType int) (err errors.Error) {
	var tl *TransactionLog
	if tl, err = this.SetCurLog(); err == nil && tl != nil {
		var memSize int64
		err = tl.Add(keyspace, nil, logType, &memSize)
		if err == nil {
			err = this.TrackMemoryQuota(memSize)
		}
	}
	return err
}

/* Write transaction mutations to  gocbcore transaction
 */
func (this *TransactionMutations) Write(deadline time.Time) (err error) {
	// Delete Transaction log. savepoints.
	var memSize int64

	this.DeleteAll(false, &memSize)

	this.mutex.Lock()
	defer this.mutex.Unlock()
	if this.tranImplicit {
		// write current delta keyspace
		dk := &this.curDeltaKeyspace
		if err = dk.Write(this.transaction, this.txnInternal, this.curKeyspace, deadline, &memSize); err != nil {
			return err
		}
	}

	// write other keyspaces
	for k, dk := range this.keyspaces {
		if err = dk.Write(this.transaction, this.txnInternal, k, deadline, &memSize); err != nil {
			return err
		}
	}

	return this.TrackMemoryQuota(-memSize)
}

func (this *TransactionMutations) DeleteAll(delta bool, memSize *int64) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	// delete save points
	for k, _ := range this.savepoints {
		*memSize += int64(len(k))
		delete(this.savepoints, k)
	}

	// delete trasaction logs
	for _, tl := range this.logs {
		if tl != nil {
			tl.DeleteAll(0, memSize)
		}
	}
	this.logs = this.logs[0:0]

	if delta {
		// delete current delta keysapce entries
		this.curDeltaKeyspace.DeleteAll(memSize)

		// delete all keyspace entries
		for k, dk := range this.keyspaces {
			if dk != nil {
				dk.DeleteAll(memSize)
			}
			this.keyspaces[k] = nil
			delete(this.keyspaces, k)
		}
	}
}

// Add entry to transaction log

func (this *TransactionLog) Add(key string, mmv *MutationValue, logType int, memSize *int64) (err errors.Error) {
	tlv := &TransactionLogValue{key: key, logType: logType}
	if tlv == nil {
		return errors.NewMemoryAllocationError("TransactionLog.Add()")
	}

	*memSize += _TRANSACTIONLOGVALUE_SZ + int64(len(key))

	if err = tlv.Set(mmv); err == nil {
		this.logValues = append(this.logValues, tlv)
	}
	return err
}

// Replay (undo) transaction log

func (this *TransactionMutations) UndoLog(sLog, sLogValIndex uint64) (err errors.Error) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	var memSize int64

	// delete entries from current delta keyspace
	dk := &this.curDeltaKeyspace
	dk.DeleteAll(&memSize)

	// save points undo transaction log and replay delta keyspace in reverse order
	if len(this.savepoints) > 0 && len(this.logs) > 0 {
		var tlv *TransactionLogValue
		var tl *TransactionLog
		var dk *DeltaKeyspace
		var cl, ci int

		// current keyspace logs can be truncated and no replay required. (Those are only in current delata keyspace)
		startLog := int(sLog)
		startLogValIndex := int(sLogValIndex)
		cKeyspace := this.curKeyspace
		if cKeyspace == "" {
			cl = len(this.logs) - 1
			if tl = this.logs[cl]; tl != nil {
				ci = len(tl.logValues) - 1
			}
		} else {
			cl = int(this.curStartLogIndex / uint64(this.logSize))
			ci = int(this.curStartLogIndex % uint64(this.logSize))

			for dcl := len(this.logs) - 1; dcl >= cl; dcl-- {
				if tl = this.logs[dcl]; tl != nil {
					pos := 0
					if cl == dcl {
						pos = ci
					} else {
						this.logs = this.logs[:dcl]
					}
					tl.DeleteAll(pos, &memSize)
				}
			}
		}

		// replay previous statement logs
		for ; cl >= startLog; cl-- {
			if tl = this.logs[cl]; tl != nil {
				sci := 0
				if cl == startLog {
					sci = startLogValIndex
				}

				for ci := len(tl.logValues) - 1; ci >= sci; ci-- {
					if tlv = tl.logValues[ci]; tlv != nil {
						memSize += _TRANSACTIONLOGVALUE_SZ + int64(len(tlv.key))
						switch tlv.logType {
						case TL_KEYSPACE:
							cKeyspace = tlv.key
							dk, _ = this.keyspaces[cKeyspace]
						case TL_SAVEPOINT:
							delete(this.savepoints, tlv.key)
						case TL_DOCUMENT:
							if err1 := tlv.Undo(dk, &memSize); err1 == nil && err == nil {
								err = err1
							}
						}
					}
				}
				tl.logValues = tl.logValues[:sci]
			}
		}
		this.curLog = startLog
		this.logs = this.logs[:this.curLog+1]
		this.curKeyspace = ""
		this.curStartLogIndex = this.TotalMutations()
		for s, v := range this.savepoints {
			if v > this.curStartLogIndex {
				memSize += int64(len(s))
				delete(this.savepoints, s)
			}
		}
	}
	if err == nil {
		err = this.TrackMemoryQuota(-memSize)
	}

	return err
}

// Statement is success merge current delta keyspace to delta keyspaces

func (this *TransactionMutations) MergeDeltaKeyspace() (err errors.Error) {
	// implicit, no current keyspace nothing to do
	if this.tranImplicit || this.curKeyspace == "" {
		return nil
	}

	this.mutex.Lock()
	defer this.mutex.Unlock()

	dk := &this.curDeltaKeyspace
	keyspace := this.curKeyspace

	var memSize int64

	mdk, ok := this.keyspaces[keyspace]
	if !ok {
		// if not already present create new delta keyspace
		mdk = &DeltaKeyspace{
			values:         make(map[string]*MutationValue, _DK_DEF_SIZE),
			ks:             dk.ks,
			collId:         dk.collId,
			bucketName:     dk.bucketName,
			scopeName:      dk.scopeName,
			collectionName: dk.collectionName}
		if mdk.values == nil {
			return errors.NewMemoryAllocationError("TransactionMutations.AddToDeltaKeyspace()")
		}
		this.keyspaces[keyspace] = mdk
		memSize += _DELTAKEYSPACE_SZ + _SZ*_DK_DEF_SIZE
	} else if mdk.ks == nil {
		mdk.ks = dk.ks
		mdk.collId = dk.collId
	}

	for key, mv := range dk.GetAll() {
		mmv, ok := mdk.values[key]
		// current is DELETE and original one is INSERT remove from delta keyspace
		if ok && mv.Op == MOP_DELETE && mmv.Op == MOP_INSERT && (mmv.Flags&MV_FLAGS_WRITE) != 0 {
			memSize -= _MUTATIONVALUE_SZ + int64(len(key))
			mdk.values[key] = nil
			delete(mdk.values, key)
		} else {
			mdk.values[key] = mv
		}

		if ok && len(this.savepoints) == 0 {
			memSize -= mmv.memSize
		}
	}

	// delete current delta keyspace entries, ignore size
	var dkMemSize int64
	dk.DeleteAll(&dkMemSize)

	if len(this.savepoints) > 0 {
		// savepoints present add end TL_KEYSPACE marker
		err = this.AddMarker(keyspace, TL_KEYSPACE)
	}

	// reset curKeyspace
	this.curKeyspace = ""

	return this.TrackMemoryQuota(memSize)
}

// Get keys in given delta keyspace
func (this *TransactionMutations) GetDeltaKeyspaceKeys(keysapce string) (keys map[string]bool, err errors.Error) {
	if this.tranImplicit {
		return
	}

	this.mutex.Lock()
	defer this.mutex.Unlock()

	dk := this.keyspaces[keysapce]
	keys = make(map[string]bool, len(dk.values))
	for k, mv := range dk.values {
		if mv.Op != MOP_NONE {
			// mark it true if deleted
			keys[k] = (mv.Op == MOP_DELETE)
		}
	}
	return
}

// write mutations to gocbcore-transactions in batches

func (this *DeltaKeyspace) Write(transaction *gctx.Transaction, txnInternal *gctx.TransactionsInternal,
	keyspace string, deadline time.Time, memSize *int64) (err error) {
	bSize := len(this.values)
	if bSize == 0 {
		return
	}

	if bSize > _WRITE_BATCH_SIZE {
		bSize = _WRITE_BATCH_SIZE
	}

	wops := make(gcagent.WriteOps, 0, bSize)
	for key, mv := range this.values {
		// delete from delta keyspace upfront so that memory will be released
		this.Delete(key, memSize)
		if mv != nil && (mv.Flags&MV_FLAGS_WRITE) != 0 {
			var data []byte
			if mv.Op != MOP_DELETE {
				// for non delete marshall the data
				if data, err = json.Marshal(mv.Val); err != nil {
					return err
				}
			}

			var txnMeta []byte
			if mv.TxnMeta != nil {
				txnMeta, _ = mv.TxnMeta.([]byte)
			}

			// batch of write ops
			wops = append(wops, &gcagent.WriteOp{Op: int(mv.Op),
				Key:     key,
				Data:    data,
				TxnMeta: txnMeta,
				Cas:     mv.KvCas,
				Expiry:  mv.Expiration})

			if len(wops) == bSize {
				// write once batch size reached
				err = this.ks.agentProvider.TxWrite(transaction, txnInternal,
					this.bucketName, this.scopeName, this.collectionName, this.collId, deadline, wops)
				if err != nil {
					return err
				}
				wops = wops[0:0]
			}
		}
	}

	if len(wops) > 0 {
		// write partial batch
		err = this.ks.agentProvider.TxWrite(transaction, txnInternal,
			this.bucketName, this.scopeName, this.collectionName, this.collId, deadline, wops)
		if err != nil {
			return err
		}
	}

	return nil
}

func (this *DeltaKeyspace) Add(key string, mv *MutationValue) {
	this.values[key] = mv
}

func (this *DeltaKeyspace) Get(key string) (mv *MutationValue) {
	mv, _ = this.values[key]
	return
}

func (this *DeltaKeyspace) GetAll() map[string]*MutationValue {
	return this.values
}

func (this *DeltaKeyspace) DeleteAll(memSize *int64) {
	for k, v := range this.values {
		if v != nil {
			*memSize += v.memSize + _MUTATIONVALUE_SZ + int64(len(k))
		}
		this.values[k] = nil
		delete(this.values, k)
	}
}

func (this *DeltaKeyspace) Delete(key string, memSize *int64) {
	if v, ok := this.values[key]; ok {
		*memSize += v.memSize + _MUTATIONVALUE_SZ + int64(len(key))
		this.values[key] = nil
		delete(this.values, key)
	}
}

func (this *TransactionLog) Len() int {
	return len(this.logValues)
}

func (this *TransactionLog) DeleteAll(pos int, memSize *int64) {
	for i, tlv := range this.logValues {
		if i >= pos && tlv != nil {
			*memSize += tlv.oldMemSize
		}
	}
	this.logValues = this.logValues[:pos]

}

func (this *TransactionLogValue) Set(mv *MutationValue) (err errors.Error) {
	if mv != nil {
		this.oldOp = mv.Op
		this.oldCas = mv.Cas
		this.oldKvCas = mv.KvCas
		this.oldExpiration = mv.Expiration
		this.oldFlags = mv.Flags
		this.oldVal = mv.Val
		this.oldTxnMeta = mv.TxnMeta
		this.oldMemSize = mv.memSize
	} else {
		this.oldOp = MOP_NONE
	}

	return err
}

func (this *TransactionLogValue) Undo(dk *DeltaKeyspace, memSize *int64) (err errors.Error) {
	if dk == nil {
		return errors.NewTransactionError(nil, "TransactionLogValue.Undo() deleta keyspace is nil")
	}
	mv, ok := dk.values[this.key]
	switch this.oldOp {
	case MOP_NONE:
		dk.values[this.key] = nil
		delete(dk.values, this.key)
		if ok {
			*memSize += _MUTATIONVALUE_SZ + mv.memSize + int64(len(this.key))
		}
	case MOP_INSERT, MOP_UPSERT, MOP_UPDATE, MOP_DELETE:
		if mv == nil {
			mv = &MutationValue{}
			*memSize -= _MUTATIONVALUE_SZ + int64(len(this.key))
		} else {
			*memSize += mv.memSize
		}
		mv.Op = this.oldOp
		mv.Cas = this.oldCas
		mv.KvCas = this.oldKvCas
		mv.Expiration = this.oldExpiration
		mv.Flags = this.oldFlags
		mv.Val = this.oldVal
		mv.TxnMeta = this.oldTxnMeta
		mv.memSize = this.oldMemSize
		dk.values[this.key] = mv
	}
	return nil
}

func (this *TransactionMutations) TrackMemoryQuota(size int64) errors.Error {
	sz := atomic.AddInt64(&this.usedMemory, size)
	if this.memoryQuota > 0 && sz > int64(this.memoryQuota) {
		return errors.NewTransactionMemoryQuotaExceededError(int64(this.memoryQuota), sz)
	}

	return nil
}

func (this *TransactionMutations) TransactionUsedMemory() int64 {
	return this.usedMemory
}
