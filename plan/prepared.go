//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plan

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"sync"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/value"
)

type Prepared struct {
	Operator
	signature       value.Value
	name            string
	encoded_plan    string
	text            string
	reqType         string
	indexApiVersion int
	featureControls uint64
	namespace       string // TODO change into scope
	useFts          bool

	indexers      []idxVersion // for reprepare checking
	namespaces    []nsVersion
	subqueryPlans map[*algebra.Select]interface{}
	sync.RWMutex
}

type idxVersion struct {
	indexer datastore.Indexer
	version uint64
}

type nsVersion struct {
	namespace datastore.Namespace
	version   uint64
}

func NewPrepared(operator Operator, signature value.Value) *Prepared {
	return &Prepared{
		Operator:  operator,
		signature: signature,
	}
}

func (this *Prepared) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Prepared) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := make(map[string]interface{}, 5)
	r["operator"] = this.Operator
	r["signature"] = this.signature
	r["name"] = this.name
	r["encoded_plan"] = this.encoded_plan
	r["text"] = this.text
	r["indexApiVersion"] = this.indexApiVersion
	r["featureControls"] = this.featureControls
	r["namespace"] = this.namespace
	if this.useFts {
		r["useFts"] = this.useFts
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *Prepared) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Operator        json.RawMessage `json:"operator"`
		Signature       json.RawMessage `json:"signature"`
		Name            string          `json:"name"`
		EncodedPlan     string          `json:"encoded_plan"`
		Text            string          `json:"text"`
		ReqType         string          `json:"reqType"`
		ApiVersion      int             `json:"indexApiVersion"`
		FeatureControls uint64          `json:"featureControls"`
		Namespace       string          `json:"namespace"`
		UseFts          bool            `json:"useFts"`
	}

	var op_type struct {
		Operator string `json:"#operator"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	err = json.Unmarshal(_unmarshalled.Operator, &op_type)
	if err != nil {
		return err
	}

	if _unmarshalled.ApiVersion < datastore.INDEX_API_MIN {
		_unmarshalled.ApiVersion = datastore.INDEX_API_MIN
	} else if _unmarshalled.ApiVersion > datastore.INDEX_API_MAX {
		_unmarshalled.ApiVersion = datastore.INDEX_API_MAX
	}
	this.signature = value.NewValue(_unmarshalled.Signature)
	this.name = _unmarshalled.Name
	this.encoded_plan = _unmarshalled.EncodedPlan
	this.text = _unmarshalled.Text
	this.reqType = _unmarshalled.ReqType
	this.indexApiVersion = _unmarshalled.ApiVersion
	this.featureControls = _unmarshalled.FeatureControls
	this.namespace = _unmarshalled.Namespace
	this.useFts = _unmarshalled.UseFts
	this.Operator, err = MakeOperator(op_type.Operator, _unmarshalled.Operator)

	return err
}

func (this *Prepared) Signature() value.Value {
	return this.signature
}

func (this *Prepared) Name() string {
	return this.name
}

func (this *Prepared) SetName(name string) {
	this.name = name
}

func (this *Prepared) Text() string {
	return this.text
}

func (this *Prepared) SetText(text string) {
	this.text = text
}

func (this *Prepared) Type() string {
	return this.reqType
}

func (this *Prepared) SetType(reqType string) {
	this.reqType = reqType
}

func (this *Prepared) IndexApiVersion() int {
	return this.indexApiVersion
}

func (this *Prepared) SetIndexApiVersion(indexApiVersion int) {
	this.indexApiVersion = indexApiVersion
}

func (this *Prepared) FeatureControls() uint64 {
	return this.featureControls
}

func (this *Prepared) SetFeatureControls(featureControls uint64) {
	this.featureControls = featureControls
}

func (this *Prepared) Namespace() string {
	return this.namespace
}

func (this *Prepared) SetNamespace(namespace string) {
	this.namespace = namespace
}

func (this *Prepared) UseFts() bool {
	return this.useFts
}

func (this *Prepared) SetUseFts(useFts bool) {
	this.useFts = useFts
}

func (this *Prepared) EncodedPlan() string {
	return this.encoded_plan
}

func (this *Prepared) SetEncodedPlan(encoded_plan string) {
	this.encoded_plan = encoded_plan
}

func (this *Prepared) BuildEncodedPlan(json_bytes []byte) string {
	var b bytes.Buffer

	w := gzip.NewWriter(&b)
	w.Write(json_bytes)
	w.Close()
	str := base64.StdEncoding.EncodeToString(b.Bytes())
	this.encoded_plan = str
	return str
}

func (this *Prepared) MismatchingEncodedPlan(encoded_plan string) bool {
	return this.encoded_plan != encoded_plan
}

// Locking is handled by the top level caller!
func (this *Prepared) addIndexer(indexer datastore.Indexer) {
	indexer.Refresh()
	version := indexer.MetadataVersion()
	for i, idx := range this.indexers {
		if idx.indexer.Name() == indexer.Name() &&
			idx.indexer.KeyspaceId() == indexer.KeyspaceId() {
			this.indexers[i].indexer = indexer
			this.indexers[i].version = version
			return
		}
	}
	this.indexers = append(this.indexers, idxVersion{indexer, version})
}

// Locking is handled by the top level caller!
func (this *Prepared) addNamespace(namespace datastore.Namespace) {
	version := namespace.MetadataVersion()
	for i, ns := range this.namespaces {
		if ns.namespace.Name() == namespace.Name() {
			this.namespaces[i].namespace = namespace
			this.namespaces[i].version = version
			return
		}
	}
	this.namespaces = append(this.namespaces, nsVersion{namespace, version})
}

func (this *Prepared) MetadataCheck() bool {

	// check that metadata is the same for the indexers involved
	for _, idx := range this.indexers {
		idx.indexer.Refresh()
		if idx.indexer.MetadataVersion() != idx.version {
			return false
		}
	}

	// now check that metadata is good for the namespaces involved
	for _, ns := range this.namespaces {
		if ns.namespace.MetadataVersion() != ns.version {
			return false
		}
	}
	return true
}

func (this *Prepared) Verify() bool {
	return this.Operator.verify(this)
}

// must be called with the prepared read locked
func (this *Prepared) GetSubqueryPlan(key *algebra.Select) (interface{}, bool) {
	if this.subqueryPlans == nil {
		return nil, false
	}
	rv, ok := this.subqueryPlans[key]
	return rv, ok
}

// must be called with the prepared write locked
func (this *Prepared) SetSubqueryPlan(key *algebra.Select, value interface{}) {
	if this.subqueryPlans == nil {
		this.subqueryPlans = make(map[*algebra.Select]interface{})
	}
	this.subqueryPlans[key] = value
}
