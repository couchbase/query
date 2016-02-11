//  Copyright (c) 2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package datastore

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

type InferenceType string

const (
	INF_DEFAULT InferenceType = "default"
)

type RandomDocumentProvider interface {
	GetRandomDoc() (string, value.Value, errors.Error)
}

type Inferencer interface {
	Name() InferenceType
	InferKeyspace(ks Keyspace, with value.Value, conn *ValueConnection) // The Inferencer should populate the connection.
}
