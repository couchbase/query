//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

func (this *builder) VisitPrepare(stmt *algebra.Prepare) (interface{}, error) {
	pl, err := BuildPrepared(stmt.Statement(), this.datastore, this.systemstore, this.namespace, false,
		this.namedArgs, this.positionalArgs, this.indexApiVersion, this.featureControls)
	if err != nil {
		return nil, err
	}

	if stmt.Name() == "" {
		uuid, err := util.UUID()
		if err != nil {
			return nil, errors.NewPreparedNameError(err.Error())
		}
		pl.SetName(uuid)
	} else {
		pl.SetName(stmt.Name())
	}

	pl.SetText(stmt.Text())
	pl.SetType(stmt.Type())

	json_bytes, err := pl.MarshalJSON()
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write(json_bytes)
	w.Close()
	str := base64.StdEncoding.EncodeToString(b.Bytes())
	pl.SetEncodedPlan(str)
	val := value.NewValue(json_bytes)
	err = val.SetField("encoded_plan", value.NewValue(str))
	if err != nil {
		return nil, err
	}

	return plan.NewPrepare(val, pl), nil
}
