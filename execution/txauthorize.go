//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execution

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
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
	ID struct {
		Transaction string `json:"txn"`
		Attempt     string `json:"atmpt"`
	} `json:"id"`
	ATR struct {
		Bucket     string `json:"bkt"`
		Scope      string `json:"scp"`
		Collection string `json:"coll"`
		ID         string `json:"id"`
	} `json:"atr"`
	Config struct {
		KvTimeoutMs        int    `json:"kvTimeoutMs"`
		KvDurableTimeoutMs int    `json:"kvDurableTimeoutMs"`
		DurabilityLevel    string `json:"durabilityLevel"`
		NumAtrs            int    `json:"numAtrs"`
	} `json:"config"`
	State struct {
		TimeLeftMs int `json:"timeLeftMs"`
	} `json:"state"`
	Mutations []jsonSerializedMutation `json:"mutations"`
}

func (this *Authorize) getTxPrivileges(privs *auth.Privileges, checkAtrPrivs bool, context *Context) (
	atrPrivs, keyspacePrivs *auth.Privileges, err error) {

	var txData jsonSerializedAttempt
	var kPath *algebra.Path

	atrPrivs = auth.NewPrivileges()
	keyspacePrivs = auth.NewPrivileges()
	atrPath := context.atrPath

	if len(context.txData) > 0 && context.txContext == nil {
		// SDK resumed transactions (BEGIN WORK).
		if err = json.Unmarshal(context.txData, &txData); err != nil {
			return
		}

		// Get the transaction required data from txData and set in then context.
		context.durabilityLevel = datastore.DurabilityNameToLevel(txData.Config.DurabilityLevel)
		if context.durabilityLevel < 0 {
			return nil, nil, fmt.Errorf("Invalid DurabilityLevel %v", txData.Config.DurabilityLevel)
		}

		if txData.State.TimeLeftMs > 0 {
			context.txTimeout = time.Duration(txData.State.TimeLeftMs) * time.Millisecond
		}

		if txData.Config.KvDurableTimeoutMs > 0 {
			context.durabilityTimeout = time.Duration(txData.Config.KvDurableTimeoutMs) * time.Millisecond
		} else if txData.Config.KvTimeoutMs > 0 {
			// set durabilityTimeout as KvTimeout
			context.durabilityTimeout = time.Duration(txData.Config.KvTimeoutMs) * time.Millisecond
		}

		// ATR collection is provided. At present N1QL doesn't support it or will not allow to change it.
		if txData.ATR.Bucket != "" {
			if atrPath == nil {
				return nil, nil, fmt.Errorf("Transactions are not configured for ATR Bucket")
			}

			txAtrPath := algebra.NewPathLong(context.namespace, txData.ATR.Bucket, txData.ATR.Scope, txData.ATR.Collection)
			if atrPath.SimpleString() != txAtrPath.SimpleString() {
				return nil, nil,
					fmt.Errorf("Transactions configured ATR collection(%s) didn't matched with passed in collection(%s)",
						atrPath.SimpleString(), txAtrPath.SimpleString())
			}
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
				case auth.PRIV_QUERY_SELECT, auth.PRIV_QUERY_UPDATE, auth.PRIV_QUERY_INSERT, auth.PRIV_QUERY_DELETE:
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
			// Atr collection check is disabled (MB-42217). By turning true those are enabled.
			checkAtrPrivs := false
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
