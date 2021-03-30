//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package plan

import (
	"encoding/json"
	"fmt"

	"github.com/couchbase/query/functions"
	globalName "github.com/couchbase/query/functions/metakv"
)

func makeName(bytes []byte) (functions.FunctionName, error) {
	var name_type struct {
		Type string `json:"type"`
	}

	err := json.Unmarshal(bytes, &name_type)
	if err != nil {
		return nil, err
	}

	switch name_type.Type {
	case "global":
		var _unmarshalled struct {
			_         string `json:"type"`
			Namespace string `json:"namespace"`
			Name      string `json:"name"`
		}

		err = json.Unmarshal(bytes, &_unmarshalled)
		if err != nil {
			return nil, err
		}
		return globalName.NewGlobalFunction(_unmarshalled.Namespace, _unmarshalled.Name)

	default:
		return nil, fmt.Errorf("unknown name type %v", name_type.Type)
	}
}
