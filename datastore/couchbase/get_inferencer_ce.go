// Copyright (c) 2016 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you
// may not use this file except in compliance with the License. You
// may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.
//
// Currently, the community edition does not have access to schema
// inferencing, so this stub returns an error.

// +build !enterprise

package couchbase

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

type NopInferencer struct {
	cluster_url string
}

func (di *NopInferencer) Name() datastore.InferenceType {
	return ("InferencingUnsuppored")
}

func (di *NopInferencer) InferKeyspace(ks datastore.Keyspace, with value.Value, conn *datastore.ValueConnection) {
	conn.Error(errors.NewOtherNotImplementedError(nil, "INFER"))
	close(conn.ValueChannel())
}

func GetDefaultInferencer(store datastore.Datastore) (datastore.Inferencer, errors.Error) {
	inferencer := new(NopInferencer)
	return inferencer, nil
}
