//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

// +build !enterprise

package couchbase

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/transactions"
	"github.com/couchbase/query/value"
)

func (s *store) StartTransaction(stmtAtomicity bool, context datastore.QueryContext) (dks map[string]bool, err errors.Error) {
	return nil, errors.NewTranCENotsupported()
}

func (s *store) CommitTransaction(stmtAtomicity bool, context datastore.QueryContext) errors.Error {
	return errors.NewTranCENotsupported()
}

func (s *store) RollbackTransaction(stmtAtomicity bool, context datastore.QueryContext, sname string) errors.Error {
	return errors.NewTranCENotsupported()
}

// Delta keyspace scan
func (s *store) TransactionDeltaKeyScan(keyspace string, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()
	conn.Fatal(errors.NewTranCENotsupported())
}

func (s *store) SetSavepoint(stmtAtomicity bool, context datastore.QueryContext, sname string) errors.Error {
	return errors.NewTranCENotsupported()
}

func (ks *keyspace) txReady(txContext *transactions.TranContext) errors.Error {
	return errors.NewTranCENotsupported()
}

func (ks *keyspace) txFetch(fullName, qualifiedName, scopeName, collectionName string, collId uint32, keys []string,
	fetchMap map[string]value.AnnotatedValue, context datastore.QueryContext, subPaths []string, sdkKvInsert bool,
	txContext *transactions.TranContext) errors.Errors {
	return errors.Errors{errors.NewTranCENotsupported()}
}

func (ks *keyspace) txPerformOp(op MutateOp, qualifiedName, scopeName, collectionName string, collId uint32, pairs value.Pairs,
	context datastore.QueryContext, txContext *transactions.TranContext) (
	mPairs value.Pairs, err errors.Error) {
	return nil, errors.NewTranCENotsupported()
}

func initGocb(s *store) (err errors.Error) {
	return errors.NewTranCENotsupported()
}

type MutateOp int

var _MutateOpNames = [...]string{"UNKNOWN", "INSERT", "UPSERT", "UPDATE", "DELETE"}

const (
	MOP_NONE MutateOp = iota
	MOP_INSERT
	MOP_UPSERT
	MOP_UPDATE
	MOP_DELETE
)

func MutateOpToName(op MutateOp) string {
	i := int(op)
	if i < 0 || i >= len(_MutateOpNames) {
		i = 0
	}

	return _MutateOpNames[i]
}

type TransactionMutations struct {
}
