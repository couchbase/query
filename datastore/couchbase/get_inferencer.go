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
// The enterprise edition has access to couchbase/query-ee, which
// includes schema inferencing. This file is only built in with
// the enterprise edition.

package couchbase

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	infer "github.com/couchbase/query/inferencer"
)

func GetDefaultInferencer(store datastore.Datastore) (datastore.Inferencer, errors.Error) {
	return infer.NewDefaultSchemaInferencer(store)
}
