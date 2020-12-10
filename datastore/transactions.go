//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
