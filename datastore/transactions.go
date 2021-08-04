//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package datastore

import (
	"strings"
	"time"
)

type DurabilityLevel int8

var _DurabilityLevelNames = [...]string{"", "none", "majority", "majorityAndPersistActive", "persistToMajority"}
var _IsolationLevelNames = [...]string{"", "READ COMMITED"}

const (
	DL_UNSET DurabilityLevel = iota
	DL_NONE
	DL_MAJORITY
	DL_MAJORITY_AND_PERSIST_TO_ACTIVE
	DL_PERSIST_TO_MAJORITY
)

const (
	DEF_TXTIMEOUT          = 15 * time.Second
	DEF_DURABILITY_TIMEOUT = 2500 * time.Millisecond
	DEF_KVTIMEOUT          = 2500 * time.Millisecond
	DEF_DURABILITY_LEVEL   = DL_MAJORITY
	DEF_NUMATRS            = 1024
)

func DurabilityNameToLevel(n string) DurabilityLevel {
	n = strings.ToLower(n)
	for i, name := range _DurabilityLevelNames {
		if strings.ToLower(name) == n {
			return DurabilityLevel(i)
		}
	}
	return DurabilityLevel(-1)
}

func DurabilityLevelToName(l DurabilityLevel) string {
	i := int(l)
	if i >= 0 && i < len(_DurabilityLevelNames) {
		return _DurabilityLevelNames[i]
	}
	return "unknown"
}

type IsolationLevel int

const (
	IL_NONE IsolationLevel = iota
	IL_READ_COMMITTED
)

func IsolationLevelToName(l IsolationLevel) string {
	i := int(l)
	if i >= 0 && i < len(_IsolationLevelNames) {
		return _IsolationLevelNames[i]
	}
	return "unknown"
}

type TransactionMemory interface {
	TransactionUsedMemory() int64
}
