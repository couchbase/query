//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package main

import (
	"encoding/json"
	"fmt"

	"github.com/couchbase/query/logging"
)

type Index struct {
	defn string
}

func NewIndex(i interface{}) (*Index, error) {
	logging.Tracef("%v", i)
	s, ok := i.(string)
	if !ok {
		return nil, fmt.Errorf("Index definition is not a string.")
	}
	return &Index{defn: s}, nil
}

func NewIndexFromKey(key string) *Index {
	return &Index{defn: fmt.Sprintf("(%s)", key)}
}

// ---------------------------------------------------------------------------------------------------------------------------------

func (this *Index) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.defn)
}

func (this *Index) sameAs(key string) bool {
	return this.defn == key || this.defn == fmt.Sprintf("(%s)", key)
}
