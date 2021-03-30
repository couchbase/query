//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package execution

import (
	"encoding/json"
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
)

const (
	_DEFAULT = "_default"
)

type jsonSerializedMutation struct {
	Bucket     string `json:"bkt"`
	Scope      string `json:"scp"`
	Collection string `json:"coll"`
	ID         string `json:"id"`
	Cas        string `json:"cas"`
	Type       string `json:"type"`
}

type jsonSerializedAttempt struct {
	ATR struct {
		Bucket     string `json:"bkt"`
		Scope      string `json:"scp"`
		Collection string `json:"coll"`
		ID         string `json:"id"`
	} `json:"atr"`
	Mutations []jsonSerializedMutation `json:"mutations"`
}

func (this *Authorize) getTxPrivileges(privs *auth.Privileges, checkAtrPrivs bool, context *Context) (
	atrPrivs, keyspacePrivs *auth.Privileges, err error) {

	var txData jsonSerializedAttempt
	var kPath, atrPath *algebra.Path

	atrPrivs = auth.NewPrivileges()
	keyspacePrivs = auth.NewPrivileges()

	if context.atrCollection != "" {
		atrPath, err = algebra.NewVariablePathWithContext(context.atrCollection, context.Namespace(), context.queryContext)
		if err != nil {
			return
		}
	}

	if len(context.txData) > 0 && context.txContext == nil {
		// SDK resumed transactions (BEGIN WORK).
		if err = json.Unmarshal(context.txData, &txData); err != nil {
			return
		}

		if checkAtrPrivs && txData.ATR.Bucket != "" && txData.ATR.Scope != "" && txData.ATR.Collection != "" {
			atrPath = algebra.NewPathLong(context.namespace, txData.ATR.Bucket, txData.ATR.Scope, txData.ATR.Collection)

		}

		var priv auth.Privilege

		// Add collection privilages of SDK Mutations.
		for _, m := range txData.Mutations {
			switch m.Type {
			case "INSERT":
				priv = auth.PRIV_QUERY_INSERT
			case "REPLACE":
				priv = auth.PRIV_QUERY_UPDATE
			case "REMOVE":
				priv = auth.PRIV_QUERY_DELETE
			default:
				return nil, nil, fmt.Errorf("Invalid mutation type %v", m.Type)
			}

			kPath = algebra.NewPathLong(context.namespace, m.Bucket, m.Scope, m.Collection)
			keyspacePrivs.Add(kPath.SimpleString(), priv, auth.PRIV_PROPS_NONE)

			// Atr collection Privileges (DATA UPSERT)
			if checkAtrPrivs && atrPath == nil {
				kPath = algebra.NewPathLong(context.namespace, m.Bucket, _DEFAULT, _DEFAULT)
				atrPrivs.Add(kPath.SimpleString(), auth.PRIV_UPSERT, auth.PRIV_PROPS_NONE)
			}
		}
	}

	if checkAtrPrivs {
		if atrPath != nil {
			// Atr collection Privileges (DATA UPSERT)
			atrPrivs.Add(atrPath.SimpleString(), auth.PRIV_UPSERT, auth.PRIV_PROPS_NONE)
		} else {
			// No Atr collection specified. ATR records goes to bucket default collection.
			// Add bucket default collection Privileges (DATA UPSERT)
			privs.ForEach(func(pp auth.PrivilegePair) {
				var ferr error
				switch pp.Priv {
				case /* auth.PRIV_QUERY_SELECT, */ auth.PRIV_QUERY_UPDATE, auth.PRIV_QUERY_INSERT, auth.PRIV_QUERY_DELETE:
					if kPath, ferr = algebra.NewVariablePathWithContext(pp.Target, context.namespace, ""); ferr == nil {
						kPath = algebra.NewPathLong(kPath.Namespace(), kPath.Bucket(), _DEFAULT, _DEFAULT)
						if pp.Priv != auth.PRIV_QUERY_SELECT {
							atrPrivs.Add(kPath.SimpleString(), auth.PRIV_UPSERT, auth.PRIV_PROPS_NONE)
						} else {
							atrPrivs.Add(kPath.SimpleString(), auth.PRIV_READ, auth.PRIV_PROPS_NONE)
						}
					} else if err == nil {
						err = ferr
					}
				}
			})
			if err != nil {
				return nil, nil, err
			}
		}
	}

	return
}

func (this *Authorize) addTxPrivileges(privs *auth.Privileges, context *Context) (*auth.Privileges, error) {
	if privs != nil {

		// Statement is within the transaction OR
		// If it is BEGIN WORK (All other statements will have context.txContext).
		addTxPrivileges := (context.txContext != nil) ||
			(len(privs.List) > 0 && privs.List[0].Priv == auth.PRIV_QUERY_TRANSACTION_STMT)

		if addTxPrivileges {
			nprivs := auth.NewPrivileges()
			checkAtrPrivs := true
			atrPrivs, keyspacePrivs, err := this.getTxPrivileges(privs, checkAtrPrivs, context)
			if err != nil {
				return nil, err
			}

			nprivs.AddAll(privs)
			nprivs.AddAll(keyspacePrivs)
			if checkAtrPrivs {
				nprivs.AddAll(atrPrivs)
			}
			return nprivs, nil
		}
	}

	return privs, nil
}
