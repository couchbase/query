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
	"sort"
	"strconv"
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
	namespace       string
	queryContext    string
	useFts          bool
	useCBO          bool

	indexScanKeyspaces              map[string]bool
	indexers                        []idxVersion // for reprepare checking
	keyspaces                       []ksVersion
	subqueryPlans                   map[*algebra.Select]interface{}
	subqueryPlansIndexScanKeyspaces map[*algebra.Select]interface{}
	txPrepareds                     map[string]*Prepared
	sync.RWMutex
}

type idxVersion struct {
	indexer datastore.Indexer
	version uint64
}

type ksVersion struct {
	ksMeta  datastore.KeyspaceMetadata
	version uint64
}

func NewPrepared(operator Operator, signature value.Value, indexScanKeyspaces map[string]bool) *Prepared {
	return &Prepared{
		Operator:           operator,
		signature:          signature,
		indexScanKeyspaces: indexScanKeyspaces,
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
	r["queryContext"] = this.queryContext
	if this.useFts {
		r["useFts"] = this.useFts
	}
	if this.useCBO {
		r["useCBO"] = this.useCBO
	}
	if len(this.indexScanKeyspaces) > 0 {
		r["indexScanKeyspaces"] = this.IndexScanKeyspaces()
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *Prepared) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Operator           json.RawMessage        `json:"operator"`
		Signature          json.RawMessage        `json:"signature"`
		Name               string                 `json:"name"`
		EncodedPlan        string                 `json:"encoded_plan"`
		Text               string                 `json:"text"`
		ReqType            string                 `json:"reqType"`
		ApiVersion         int                    `json:"indexApiVersion"`
		FeatureControls    uint64                 `json:"featureControls"`
		Namespace          string                 `json:"namespace"`
		QueryContext       string                 `json:"queryContext"`
		UseFts             bool                   `json:"useFts"`
		UseCBO             bool                   `json:"useCBO"`
		IndexScanKeyspaces map[string]interface{} `json:"indexScanKeyspaces"`
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
	this.queryContext = _unmarshalled.QueryContext
	this.useFts = _unmarshalled.UseFts
	this.useCBO = _unmarshalled.UseCBO
	if len(_unmarshalled.IndexScanKeyspaces) > 0 {
		this.indexScanKeyspaces = make(map[string]bool, len(_unmarshalled.IndexScanKeyspaces))
		for ks, v := range _unmarshalled.IndexScanKeyspaces {
			this.indexScanKeyspaces[ks] = v.(bool)
		}
	}
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

func (this *Prepared) QueryContext() string {
	return this.queryContext
}

func (this *Prepared) SetQueryContext(queryContext string) {
	this.queryContext = queryContext
}

func (this *Prepared) UseFts() bool {
	return this.useFts
}

func (this *Prepared) SetUseFts(useFts bool) {
	this.useFts = useFts
}

func (this *Prepared) UseCBO() bool {
	return this.useCBO
}

func (this *Prepared) SetUseCBO(useCBO bool) {
	this.useCBO = useCBO
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

func (this *Prepared) SetIndexScanKeyspaces(ik map[string]bool) {
	this.indexScanKeyspaces = ik
}

func (this *Prepared) IndexScanKeyspaces() (rv map[string]interface{}) {
	if len(this.indexScanKeyspaces) > 0 {
		rv = make(map[string]interface{}, len(this.indexScanKeyspaces))
		for ks, v := range this.indexScanKeyspaces {
			rv[ks] = v
		}
	}
	return rv
}

// Locking is handled by the top level caller!
func (this *Prepared) addIndexer(indexer datastore.Indexer) {
	indexer.Refresh()
	version := indexer.MetadataVersion()
	for i, idx := range this.indexers {
		if idx.indexer.Name() == indexer.Name() &&
			datastore.IndexerQualifiedKeyspacePath(idx.indexer) == datastore.IndexerQualifiedKeyspacePath(indexer) {
			this.indexers[i].indexer = indexer
			this.indexers[i].version = version
			return
		}
	}
	this.indexers = append(this.indexers, idxVersion{indexer, version})
}

// Locking is handled by the top level caller!
func (this *Prepared) addKeyspaceMetadata(ksMeta datastore.KeyspaceMetadata) {
	version := ksMeta.MetadataVersion()
	for i, ks := range this.keyspaces {
		if ks.ksMeta.MetadataId() == ksMeta.MetadataId() {
			this.keyspaces[i].ksMeta = ksMeta
			this.keyspaces[i].version = version
			return
		}
	}
	this.keyspaces = append(this.keyspaces, ksVersion{ksMeta, version})
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
	// if the bucket has been deleted, the version is expected to be different
	for _, ks := range this.keyspaces {
		if ks.ksMeta.MetadataVersion() != ks.version {
			return false
		}
	}
	return true
}

func (this *Prepared) Verify() bool {
	return this.Operator.verify(this)
}

// must be called with the prepared read locked
func (this *Prepared) GetSubqueryPlan(key *algebra.Select) (interface{}, interface{}, bool) {
	if this.subqueryPlans == nil {
		return nil, nil, false
	}
	rv, ok := this.subqueryPlans[key]
	irv, _ := this.subqueryPlansIndexScanKeyspaces[key]
	return rv, irv, ok
}

// must be called with the prepared write locked
func (this *Prepared) SetSubqueryPlan(key *algebra.Select, value, iks interface{}) {
	if this.subqueryPlans == nil {
		this.subqueryPlans = make(map[*algebra.Select]interface{})
		this.subqueryPlansIndexScanKeyspaces = make(map[*algebra.Select]interface{})
	}
	this.subqueryPlans[key] = value
	this.subqueryPlansIndexScanKeyspaces[key] = iks
}

const (
	_TX_KEYSPACES = 2
)

func (this *Prepared) SetTxPrepared(txPrepared *Prepared, hashCode string) {
	if hashCode == "" || len(this.indexScanKeyspaces) > _TX_KEYSPACES {
		return
	}
	this.Lock()
	defer this.Unlock()
	if this.txPrepareds == nil {
		this.txPrepareds = make(map[string]*Prepared, (1 << _TX_KEYSPACES))
	}
	this.txPrepareds[hashCode] = txPrepared
}

func (this *Prepared) GetTxPrepared(deltaKeyspaces map[string]bool) (*Prepared, string) {
	good := this.Type() != "DELETE"
	if len(deltaKeyspaces) > 0 {
		if len(this.indexScanKeyspaces) == 0 {
			if good {
				return this, ""
			} else {
				deltaKeyspaces = nil
			}
		} else if len(this.indexScanKeyspaces) > _TX_KEYSPACES {
			return nil, ""
		}
		if good {
			for ks, _ := range this.indexScanKeyspaces {
				if _, ok := deltaKeyspaces[ks]; ok {
					good = false
					break
				}
			}
			if good {
				return this, ""
			}
		}

	}

	hashCode := this.txHashCode(deltaKeyspaces)
	this.RLock()
	defer this.RUnlock()
	prepared, _ := this.txPrepareds[hashCode]
	return prepared, hashCode
}

func (this *Prepared) TxPrepared() (rv map[string]interface{}, rvp map[string]interface{}) {
	if len(this.txPrepareds) == 0 {
		return
	}
	this.RLock()
	defer this.RUnlock()
	rv = make(map[string]interface{}, len(this.txPrepareds))
	rvp = make(map[string]interface{}, len(this.txPrepareds))
	i := 1
	for _, p := range this.txPrepareds {
		plan := p.EncodedPlan()
		isks := p.IndexScanKeyspaces()
		if plan != "" && len(isks) > 0 {
			key := "p" + strconv.Itoa(i)
			i++
			rv[key] = map[string]interface{}{
				"plan":               plan,
				"indexScanKeyspaces": isks,
			}
			b, _ := json.Marshal(p.Operator)
			rvp[key] = map[string]interface{}{
				"plan":               b,
				"indexScanKeyspaces": isks,
			}
		}
	}
	return
}

func (this *Prepared) txHashCode(deltaKeyspaces map[string]bool) (hashCode string) {
	if len(deltaKeyspaces) == 0 {
		return "(delete)"
	}

	var i int
	var nameBuf [_TX_KEYSPACES]string
	names := nameBuf[0:len(this.indexScanKeyspaces)]

	for ks, _ := range this.indexScanKeyspaces {
		names[i] = ks
		i++
	}

	sort.Strings(names)

	for i, ks := range names {
		if i != 0 {
			hashCode += ","
		}
		if ok, _ := deltaKeyspaces[ks]; ok {
			hashCode += "(" + ks + "):true"
		} else {
			hashCode += "(" + ks + "):false"
		}
	}

	return hashCode
}
